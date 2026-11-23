package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 用量统计:扫描 ~/.claude/projects/*/*.jsonl 里每条助手消息自带的 usage(同一条消息流式写多行,
// 取 message.id 最后一次出现的),按模型价目表估算费用。第三方服务商 / 未知模型没有价目,只统计 token。
// 只扫每个项目目录下的会话文件(不含子 Agent 的独立文件);文件按「大小 + 修改时间」缓存,没变就不重读。

// UsageRec 是一条助手消息的用量。
type UsageRec struct {
	Time                time.Time
	Session, Cwd, Model string
	In, Out, CacheR     int64
	CacheW              int64
}

// price 是每百万 token 的美元价目。
type price struct{ in, out, cacheR, cacheW float64 }

// priceOf 按模型名里的系列取价目(估算;认不出的返回 false)。
func priceOf(model string) (price, bool) {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "opus"):
		return price{5, 25, .5, 6.25}, true
	case strings.Contains(m, "sonnet"):
		return price{3, 15, .3, 3.75}, true
	case strings.Contains(m, "haiku"):
		return price{1, 5, .1, 1.25}, true
	}
	return price{}, false
}

// Cost 估算这条记录的费用(未知模型 = 0)。
func (r UsageRec) Cost() float64 {
	p, ok := priceOf(r.Model)
	if !ok {
		return 0
	}
	return (float64(r.In)*p.in + float64(r.Out)*p.out + float64(r.CacheR)*p.cacheR + float64(r.CacheW)*p.cacheW) / 1e6
}

// usageFile 是一个会话文件的增量读取状态。会话文件只追加,所以文件变长时从 off 接着读,不必整份重读
// (进行中的长会话文件可达几十 MB,原先每分钟整读一次)。
//
// 状态只记「以换行结尾的完整行」:off 是最后一个换行之后的位置;正在写的半行(没有换行)
// 每次扫描临时解析、参与结果,但不进状态,补齐换行后再正式读入。这样结果与整份重读严格一致。
//
// 接着读的前提是「只追加」:换了文件(改名替换)、变短、或 off 之前最后 guardLen 字节变了,都退回整份重读。
// 已知边界:原地改写了前面的内容、却恰好保留 off 之前那 guardLen 字节,这种改写识别不出来。
// Claude Code 只追加会话文件;MiniCode 自己不改写会话 jsonl(编辑提问走 resume_session_at),所以实际不会发生。
type usageFile struct {
	info  os.FileInfo // 上次读时的文件信息(判断是不是同一个文件)
	size  int64       // 上次读时的大小
	mtime int64       // 上次读时的修改时间
	off   int64       // 已读完整行的末尾位置
	guard []byte      // off 之前的最后几个字节:接着读之前核对,对不上说明文件被改写,退回整份重读
	recs  []UsageRec  // 完整行得出的记录(同一 message.id 只留一条)
	idx   map[string]int
	out   []UsageRec // 本文件对外的结果 = recs 再叠上半行(没有半行时就是 recs)
}

// guardLen 是接着读之前核对的尾部字节数。
const guardLen = 64

// UsageScanner 带缓存地扫描会话文件。并发安全。
type UsageScanner struct {
	mu    sync.Mutex
	files map[string]*usageFile
	all   []UsageRec // 上次的合并结果(时间升序);没有文件变化时直接返回
}

// UsageScanStats 是一次扫描做了多少事(用于日志与测试)。Changed 为 false 时返回的就是上一次的同一个切片。
type UsageScanStats struct {
	Files, Full, Incremental, Unchanged int
	BytesRead                           int64
	Changed                             bool
}

// Scan 扫描 projects 目录,返回全部用量记录(时间升序)。调用方不得修改返回的切片。
func (s *UsageScanner) Scan(projects string) []UsageRec {
	recs, _ := s.ScanStats(projects)
	return recs
}

// ScanStats 同 Scan,另外返回这次扫描的工作量。
func (s *UsageScanner) ScanStats(projects string) ([]UsageRec, UsageScanStats) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.files == nil {
		s.files = map[string]*usageFile{}
	}
	var st UsageScanStats
	paths, _ := filepath.Glob(filepath.Join(projects, "*", "*.jsonl"))
	seen := make(map[string]bool, len(paths))
	changed := false
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		seen[p] = true
		st.Files++
		f := s.files[p]
		switch {
		case f != nil && f.size == info.Size() && f.mtime == info.ModTime().UnixNano():
			st.Unchanged++
			continue
		case f != nil && os.SameFile(f.info, info) && info.Size() >= f.off && f.guardOK(p):
			st.Incremental++
		default:
			f = &usageFile{idx: map[string]int{}}
			s.files[p] = f
			st.Full++
		}
		st.BytesRead += f.readFrom(p, info)
		changed = true
	}
	for p := range s.files { // 已删除的会话
		if !seen[p] {
			delete(s.files, p)
			changed = true
		}
	}
	if !changed && s.all != nil {
		return s.all, st
	}
	st.Changed = true
	var all []UsageRec
	for _, p := range paths { // 按路径顺序合并(Glob 已排序),同一时刻的记录顺序稳定
		if f := s.files[p]; f != nil && seen[p] {
			all = append(all, f.out...)
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Time.Before(all[j].Time) })
	s.all = all
	return all, st
}

// guardOK 核对文件 off 之前的最后几个字节与上次读到的一致(只追加的文件必然一致)。
func (f *usageFile) guardOK(path string) bool {
	if f.off == 0 {
		return true
	}
	fh, err := os.Open(path)
	if err != nil {
		return false
	}
	defer fh.Close()
	b := make([]byte, len(f.guard))
	if _, err := fh.ReadAt(b, f.off-int64(len(b))); err != nil {
		return false
	}
	return bytes.Equal(b, f.guard)
}

// readFrom 从 f.off 接着读到文件末尾,更新状态与对外结果,返回读了多少字节。
func (f *usageFile) readFrom(path string, info os.FileInfo) int64 {
	f.info, f.size, f.mtime = info, info.Size(), info.ModTime().UnixNano()
	fh, err := os.Open(path)
	if err != nil {
		f.out = f.recs
		return 0
	}
	defer fh.Close()
	if _, err := fh.Seek(f.off, io.SeekStart); err != nil {
		f.out = f.recs
		return 0
	}
	sid := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	r := bufio.NewReaderSize(fh, 1<<20)
	var read int64
	var tail []byte // 结尾没有换行的半行
	start := f.off
	for {
		line, err := r.ReadBytes('\n')
		read += int64(len(line))
		if len(line) > 0 && line[len(line)-1] == '\n' {
			if rec, id, ok := parseUsageLine(line, sid); ok {
				addUsage(&f.recs, f.idx, rec, id)
			}
			f.off += int64(len(line))
		} else if len(line) > 0 {
			tail = line
		}
		if err != nil {
			break
		}
	}
	if f.off != start { // 读完再取一次尾部(不在每行都分配)
		n := min(int64(guardLen), f.off)
		g := make([]byte, n)
		if _, err := fh.ReadAt(g, f.off-n); err == nil {
			f.guard = g
		} else { // 取不到尾部:这次的记录照用,但让下次一定整份重读(info 置空 → SameFile 不成立,mtime 对不上)
			f.info, f.mtime, f.guard = nil, -1, nil
		}
	}
	f.out = f.recs
	if rec, id, ok := parseUsageLine(tail, sid); ok { // 半行:临时叠加,不进状态
		out := append([]UsageRec(nil), f.recs...)
		if i, hit := f.idx[id]; hit && id != "" {
			out[i] = rec
		} else {
			out = append(out, rec)
		}
		f.out = out
	}
	return read
}

// addUsage 把一条记录并入:同一 message.id 的多行(流式)后出现的覆盖先出现的。
func addUsage(recs *[]UsageRec, idx map[string]int, rec UsageRec, id string) {
	if i, ok := idx[id]; ok && id != "" {
		(*recs)[i] = rec
		return
	}
	if id != "" {
		idx[id] = len(*recs)
	}
	*recs = append(*recs, rec)
}

// parseUsageLine 解析一行会话记录里的用量;不是带用量的助手消息返回 false。
func parseUsageLine(line []byte, sid string) (UsageRec, string, bool) {
	if len(line) == 0 || !bytes.Contains(line, []byte(`"usage"`)) || !bytes.Contains(line, []byte(`"assistant"`)) {
		return UsageRec{}, "", false
	}
	var e struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		Cwd       string `json:"cwd"`
		Message   struct {
			ID    string `json:"id"`
			Model string `json:"model"`
			Usage struct {
				In     int64 `json:"input_tokens"`
				Out    int64 `json:"output_tokens"`
				CacheR int64 `json:"cache_read_input_tokens"`
				CacheW int64 `json:"cache_creation_input_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &e) != nil || e.Type != "assistant" || e.Message.Model == "" || e.Message.Model == "<synthetic>" {
		return UsageRec{}, "", false
	}
	ts, _ := time.Parse(time.RFC3339Nano, e.Timestamp)
	u := e.Message.Usage
	return UsageRec{Time: ts, Session: sid, Cwd: e.Cwd, Model: e.Message.Model, In: u.In, Out: u.Out, CacheR: u.CacheR, CacheW: u.CacheW}, e.Message.ID, true
}

// readUsage 整份读一个会话文件里的用量记录(测试里当作「整份重读」的参照)。
func readUsage(path string) []UsageRec {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	f := &usageFile{idx: map[string]int{}}
	f.readFrom(path, info)
	return f.out
}

// 范围:0 = 最近 7 天 / 1 = 最近 30 天 / 2 = 本月。
var usageRangeNames = []string{"7 天", "30 天", "本月"}

func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// BuildReport 把记录汇总成某个范围的报告。title 把会话 id 换成标题与项目目录(找不到返回空)。
func BuildReport(recs []UsageRec, rng int, now time.Time, title func(id string) (string, string)) core.UsageReport {
	if rng < 0 || rng >= len(usageRangeNames) {
		rng = 1
	}
	today := dayStart(now)
	var from time.Time
	switch rng {
	case 0:
		from = today.AddDate(0, 0, -6)
	case 1:
		from = today.AddDate(0, 0, -29)
	default:
		from = time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	}
	nDays := int(today.Sub(from).Hours()/24+.5) + 1
	rep := core.UsageReport{RangeName: usageRangeNames[rng]}
	days := make([]float64, nDays)
	span := today.AddDate(0, 0, 1).Sub(from)
	prevFrom := from.Add(-span)
	var prevTotal float64
	type agg struct {
		cost float64
		sess map[string]bool
	}
	byModel, byProj := map[string]*agg{}, map[string]*agg{}
	bySess := map[string]float64{}
	sessModel := map[string]map[string]float64{}
	sessions := map[string]bool{}
	var in, out, cacheR, cacheW int64
	for _, r := range recs {
		c := r.Cost()
		lt := r.Time.In(now.Location())
		if !lt.Before(prevFrom) && lt.Before(from) {
			prevTotal += c
		}
		if lt.Before(from) || !lt.Before(today.AddDate(0, 0, 1)) {
			continue
		}
		days[int(dayStart(lt).Sub(from).Hours()/24+.5)] += c
		rep.Total += c
		in += r.In + r.CacheR + r.CacheW
		out += r.Out
		cacheR += r.CacheR
		cacheW += r.CacheW
		sessions[r.Session] = true
		name := modelLabel(r.Model)
		for k, m := range map[string]map[string]*agg{name: byModel, filepath.Base(r.Cwd): byProj} {
			a := m[k]
			if a == nil {
				a = &agg{sess: map[string]bool{}}
				m[k] = a
			}
			a.cost += c
			a.sess[r.Session] = true
		}
		bySess[r.Session] += c
		if sessModel[r.Session] == nil {
			sessModel[r.Session] = map[string]float64{}
		}
		sessModel[r.Session][name] += float64(r.Out) + c*1e3
	}
	for i := 0; i < nDays; i++ {
		d := from.AddDate(0, 0, i)
		rep.Days = append(rep.Days, core.Day{Label: strconv.Itoa(int(d.Month())) + " 月 " + strconv.Itoa(d.Day()) + " 日", Cost: days[i]})
	}
	if prevTotal > 0 {
		rep.DeltaPercent = (rep.Total - prevTotal) / prevTotal * 100
	}
	rep.InputTokens, rep.OutputTokens = tokFmt(in), tokFmt(out)
	rep.Sessions = len(sessions)
	if in > 0 {
		rep.CacheHit = int(float64(cacheR)/float64(in)*100 + .5)
	}
	kinds := map[string]int{"Sonnet": 0, "Opus": 1, "Haiku": 2}
	for name, a := range byModel {
		sh := core.Share{Name: name, Cost: a.cost, Kind: 0}
		for k, v := range kinds {
			if strings.HasPrefix(name, k) {
				sh.Kind = v
			}
		}
		if rep.Total > 0 {
			sh.Fraction = a.cost / rep.Total
		}
		rep.Models = append(rep.Models, sh)
	}
	sort.Slice(rep.Models, func(i, j int) bool { return rep.Models[i].Cost > rep.Models[j].Cost })
	for name, a := range byProj {
		rep.Projects = append(rep.Projects, core.Share{Name: name, Cost: a.cost, Sessions: len(a.sess)})
	}
	sort.Slice(rep.Projects, func(i, j int) bool { return rep.Projects[i].Cost > rep.Projects[j].Cost })
	if len(rep.Projects) > 6 {
		rep.Projects = rep.Projects[:6]
	}
	type kv struct {
		id string
		c  float64
	}
	var tops []kv
	for id, c := range bySess {
		tops = append(tops, kv{id, c})
	}
	sort.Slice(tops, func(i, j int) bool { return tops[i].c > tops[j].c })
	for i, t := range tops {
		if i >= 5 || t.c <= 0 {
			break
		}
		ttl, proj := "", ""
		if title != nil {
			ttl, proj = title(t.id)
		}
		if ttl == "" {
			ttl = "会话 " + t.id[:min(8, len(t.id))]
		}
		best, bv := "", -1.0
		for m, v := range sessModel[t.id] {
			if v > bv {
				best, bv = m, v
			}
		}
		rep.TopSessions = append(rep.TopSessions, core.CostSession{SessionID: t.id, Title: ttl, Project: filepath.Base(proj), Model: best, Cost: t.c})
	}
	rep.Tips = usageTips(rep, cacheW, cacheR)
	return rep
}

// usageTips 从数据里得出的本地建议(不联网)。
func usageTips(rep core.UsageReport, cacheW, cacheR int64) []core.Tip {
	var tips []core.Tip
	switch {
	case rep.Sessions == 0:
		tips = append(tips, core.Tip{Level: 2, Title: "这个范围里还没有记录", Desc: "会话记录取自 ~/.claude/projects;第三方服务商的会话没有价目,只统计 token。"})
	case rep.CacheHit >= 80:
		tips = append(tips, core.Tip{Level: 0, Title: "缓存命中率很高", Desc: "命中率 " + strconv.Itoa(rep.CacheHit) + "%,重复上下文基本都走缓存,继续保持长会话即可。"})
	case rep.CacheHit < 50:
		tips = append(tips, core.Tip{Level: 1, Title: "缓存命中率偏低", Desc: "命中率 " + strconv.Itoa(rep.CacheHit) + "%。频繁新开会话或改动 CLAUDE.md 会让缓存失效。"})
	}
	if len(rep.Models) > 0 && rep.Models[0].Kind == 1 && rep.Models[0].Fraction > .5 {
		tips = append(tips, core.Tip{Level: 1, Title: "费用主要来自 Opus", Desc: "简单的查找与小改动可以切到 Sonnet,费用约为 Opus 的五分之三以下。"})
	}
	tips = append(tips, core.Tip{Level: 2, Title: "费用为估算", Desc: "按公开价目与消息自带的 token 数估算,与账单可能有出入。"})
	return tips
}

// modelLabel 把模型 id 写成「系列 版本」:claude-sonnet-5-5-20260101 → Sonnet 5.5。
func modelLabel(id string) string {
	m := strings.TrimPrefix(strings.ToLower(id), "claude-")
	parts := strings.Split(m, "-")
	if len(parts) == 0 {
		return id
	}
	fam := strings.ToUpper(parts[0][:1]) + parts[0][1:]
	var ver []string
	for _, p := range parts[1:] {
		if len(p) > 2 { // 日期后缀
			break
		}
		if _, err := strconv.Atoi(p); err != nil {
			break
		}
		ver = append(ver, p)
	}
	if len(ver) == 0 {
		return fam
	}
	return fam + " " + strings.Join(ver, ".")
}

// tokFmt 把 token 数写成 5.0M / 624k。
func tokFmt(n int64) string {
	switch {
	case n >= 1_000_000:
		return strconv.FormatFloat(float64(n)/1e6, 'f', 1, 64) + "M"
	case n >= 1000:
		return strconv.FormatInt(n/1000, 10) + "k"
	}
	return strconv.FormatInt(n, 10)
}
