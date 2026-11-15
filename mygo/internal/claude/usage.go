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

type usageFile struct {
	size, mtime int64
	recs        []UsageRec
}

// UsageScanner 带缓存地扫描会话文件。并发安全。
type UsageScanner struct {
	mu    sync.Mutex
	files map[string]*usageFile
}

// Scan 扫描 projects 目录,返回全部用量记录(时间升序)。
func (s *UsageScanner) Scan(projects string) []UsageRec {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.files == nil {
		s.files = map[string]*usageFile{}
	}
	paths, _ := filepath.Glob(filepath.Join(projects, "*", "*.jsonl"))
	seen := make(map[string]bool, len(paths))
	var all []UsageRec
	for _, p := range paths {
		seen[p] = true
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		f := s.files[p]
		if f == nil || f.size != st.Size() || f.mtime != st.ModTime().UnixNano() {
			f = &usageFile{size: st.Size(), mtime: st.ModTime().UnixNano(), recs: readUsage(p)}
			s.files[p] = f
		}
		all = append(all, f.recs...)
	}
	for p := range s.files { // 已删除的会话
		if !seen[p] {
			delete(s.files, p)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Time.Before(all[j].Time) })
	return all
}

// readUsage 读一个会话文件里的用量记录。
func readUsage(path string) []UsageRec {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	sid := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	r := bufio.NewReaderSize(f, 1<<20)
	idx := map[string]int{}
	var recs []UsageRec
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 && bytes.Contains(line, []byte(`"usage"`)) && bytes.Contains(line, []byte(`"assistant"`)) {
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
			if json.Unmarshal(line, &e) == nil && e.Type == "assistant" && e.Message.Model != "" && e.Message.Model != "<synthetic>" {
				ts, _ := time.Parse(time.RFC3339Nano, e.Timestamp)
				u := e.Message.Usage
				rec := UsageRec{Time: ts, Session: sid, Cwd: e.Cwd, Model: e.Message.Model, In: u.In, Out: u.Out, CacheR: u.CacheR, CacheW: u.CacheW}
				if i, ok := idx[e.Message.ID]; ok && e.Message.ID != "" {
					recs[i] = rec // 流式的多行记录:后出现的覆盖先出现的
				} else {
					if e.Message.ID != "" {
						idx[e.Message.ID] = len(recs)
					}
					recs = append(recs, rec)
				}
			}
		}
		if err == io.EOF || err != nil {
			break
		}
	}
	return recs
}

// 范围:0 = 最近 7 天 / 1 = 最近 30 天 / 2 = 本月。
var usageRangeNames = []string{"7 天", "30 天", "本月"}

// RangeNames 是范围切换的选项。
func RangeNames() []string { return usageRangeNames }

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
		rep.TopSessions = append(rep.TopSessions, core.CostSession{Title: ttl, Project: filepath.Base(proj), Model: best, Cost: t.c})
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
