package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// ConfigDir 是 claude 的配置根目录:$CLAUDE_CONFIG_DIR 或 ~/.claude。
func ConfigDir() string {
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return d
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".claude")
}

// ProjectsDir 是会话记录目录。
func ProjectsDir() string { return filepath.Join(ConfigDir(), "projects") }

// EncodeProjectDir 工作目录 → projects 下的目录名(非字母数字一律替换为 -,有损)。
func EncodeProjectDir(cwd string) string {
	var b strings.Builder
	for _, r := range cwd {
		if r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// SessionOnDisk 会话记录是否已写入磁盘(claude 可能按解开符号链接后的路径记录,两种写法都查)。
func SessionOnDisk(projects, cwd, id string) bool {
	cands := []string{cwd}
	if real, err := filepath.EvalSymlinks(cwd); err == nil {
		cands = append(cands, real)
	}
	for _, c := range cands {
		if st, err := os.Stat(filepath.Join(projects, EncodeProjectDir(c), id+".jsonl")); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

// scan 是一个会话文件的扫描进度与已提取的字段。
// 会话文件只追加不改写,所以记下「已扫描到的字节偏移」:文件变大只扫新增部分;
// 变小或文件头变了才从头重扫;大小与修改时间都没变直接用缓存。读取按行进行,内存峰值只是最长的一行。
type scan struct {
	Offset     int64  `json:"offset"`
	Size       int64  `json:"size"`
	MTime      int64  `json:"mtime"`
	Head       uint64 `json:"head"`
	Cwd        string `json:"cwd"`
	AITitle    string `json:"ai_title"`
	LastPrompt string `json:"last_prompt"`
	FirstUser  string `json:"first_user"`
}

const headLen = 256

func (s *scan) title() string {
	for _, t := range []string{s.AITitle, s.FirstUser, s.LastPrompt} {
		if t != "" {
			line := strings.SplitN(t, "\n", 2)[0]
			if r := []rune(line); len(r) > 60 {
				line = string(r[:60])
			}
			return line
		}
	}
	return ""
}

var scanCache = struct {
	sync.Mutex
	m     map[string]scan
	file  string
	dirty bool
}{m: map[string]scan{}}

// SetCacheFile 指定缓存落盘位置并载入已有缓存(启动时调用一次;不调用则只在内存里缓存)。
func SetCacheFile(path string) {
	scanCache.Lock()
	defer scanCache.Unlock()
	scanCache.file = path
	var loaded map[string]scan
	if b, err := os.ReadFile(path); err == nil && json.Unmarshal(b, &loaded) == nil {
		for k, v := range loaded {
			if _, ok := scanCache.m[k]; !ok {
				scanCache.m[k] = v
			}
		}
	}
}

func flushCache() {
	scanCache.Lock()
	if scanCache.file == "" || !scanCache.dirty {
		scanCache.Unlock()
		return
	}
	for p := range scanCache.m { // 只保留仍然存在的文件
		if _, err := os.Stat(p); err != nil {
			delete(scanCache.m, p)
		}
	}
	scanCache.dirty = false
	b, err := json.Marshal(scanCache.m)
	file := scanCache.file
	scanCache.Unlock()
	if err != nil {
		return
	}
	tmp := file + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil && os.Rename(tmp, file) != nil {
		os.Remove(tmp)
	}
}

func hashBytes(b []byte) uint64 {
	h := fnv.New64a()
	h.Write(b)
	return h.Sum64()
}

func scanSession(path string, size, mtime int64) (scan, bool) {
	scanCache.Lock()
	cached, has := scanCache.m[path]
	scanCache.Unlock()
	if has && cached.Size == size && cached.MTime == mtime {
		return cached, true
	}
	fresh, ok := scanFile(path, size, mtime, cached, has)
	if !ok {
		return scan{}, false
	}
	scanCache.Lock()
	scanCache.m[path] = fresh
	scanCache.dirty = true
	scanCache.Unlock()
	return fresh, true
}

func scanFile(path string, size, mtime int64, prev scan, hasPrev bool) (scan, bool) {
	f, err := os.Open(path)
	if err != nil {
		return scan{}, false
	}
	defer f.Close()
	head := make([]byte, min(int64(headLen), size))
	if _, err := io.ReadFull(f, head); err != nil && len(head) > 0 {
		return scan{}, false
	}
	h := hashBytes(head)
	s := scan{}
	// 上次扫描时文件不足 headLen,头部哈希会随增长变化,保守起见从头扫
	if hasPrev && prev.Head == h && prev.Size >= headLen && size >= prev.Offset {
		s = prev
	}
	s.Size, s.MTime, s.Head = size, mtime, h
	if _, err := f.Seek(s.Offset, io.SeekStart); err != nil {
		return scan{}, false
	}
	r := bufio.NewReaderSize(f, 64*1024)
	for {
		line, err := r.ReadBytes('\n')
		// 末尾没有换行的半行(claude 正在写)留到下次再读
		if err != nil || len(line) == 0 || line[len(line)-1] != '\n' {
			break
		}
		s.Offset += int64(len(line))
		absorb(&s, strings.TrimRight(string(line), "\r\n"))
	}
	return s, true
}

// absorb 吸收一行:只解析可能有用的行,避免对巨大的会话文件做全量 JSON 解析。
func absorb(s *scan, line string) {
	interesting := strings.HasPrefix(line, `{"type":"ai-title"`) || strings.HasPrefix(line, `{"type":"last-prompt"`) ||
		(s.Cwd == "" && strings.Contains(line, `"cwd":"`)) ||
		(s.FirstUser == "" && strings.HasPrefix(line, `{"type":"user"`))
	if !interesting {
		return
	}
	v := parseLine(line)
	if v == nil {
		return
	}
	if s.Cwd == "" {
		s.Cwd = str(v, "cwd")
	}
	switch str(v, "type") {
	case "ai-title":
		s.AITitle = str(v, "aiTitle")
	case "last-prompt":
		s.LastPrompt = str(v, "lastPrompt")
	case "user":
		if s.FirstUser == "" {
			if t := userText(v); !strings.HasPrefix(t, "<") && !strings.HasPrefix(t, "Caveat:") {
				s.FirstUser = t
			}
		}
	}
}

func userText(v obj) string {
	msg := sub(v, "message")
	switch c := msg["content"].(type) {
	case string:
		return c
	case []any:
		for _, b := range c {
			if bm := asObj(b); str(bm, "type") == "text" {
				return str(bm, "text")
			}
		}
	}
	return ""
}

// SessionMeta 是一个历史会话的概要。
type SessionMeta struct {
	ID, Path, Cwd, Title string
	Modified             time.Time
}

// ProjectMeta 是一个项目目录的概要(工作目录取自会话内容,目录名的编码是有损的)。
type ProjectMeta struct {
	Cwd      string
	Sessions []SessionMeta // 最近修改的在前
}

// ListProjects 列出 projects 下所有项目及其会话,项目与会话都按最近活动倒序;没有有效会话的目录忽略。
func ListProjects(projects string) []ProjectMeta {
	dirs, err := os.ReadDir(projects)
	if err != nil {
		return nil
	}
	var out []ProjectMeta
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		sessions := sessionsIn(filepath.Join(projects, d.Name()))
		cwd := ""
		for _, s := range sessions {
			if s.Cwd != "" {
				cwd = s.Cwd
				break
			}
		}
		if len(sessions) == 0 || cwd == "" {
			continue
		}
		out = append(out, ProjectMeta{Cwd: cwd, Sessions: sessions})
	}
	out = mergeByCwd(out)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sessions[0].Modified.After(out[j].Sessions[0].Modified) })
	flushCache()
	return out
}

// mergeByCwd 把工作目录相同的项目合成一个。
// 用途:claude 按「启动时所在目录」分目录存会话,从别的目录 --resume 过来的会话会落在另一个
// 目录里、cwd 却仍是原项目,不合并侧栏 / 项目下拉就会出现好几个同名同路径的项目;
// 同一个会话 id 出现在多处时只留最近修改的那份。
// 参数:projs 为按目录扫描出的项目列表。返回值:按 cwd 去重后的项目列表,会话仍按最近修改倒序。
func mergeByCwd(projs []ProjectMeta) []ProjectMeta {
	idx := map[string]int{}
	var out []ProjectMeta
	for _, p := range projs {
		key := filepath.Clean(p.Cwd)
		i, ok := idx[key]
		if !ok {
			idx[key] = len(out)
			out = append(out, p)
			continue
		}
		out[i].Sessions = append(out[i].Sessions, p.Sessions...)
	}
	for i := range out {
		ss := out[i].Sessions
		sort.SliceStable(ss, func(a, b int) bool { return ss[a].Modified.After(ss[b].Modified) })
		seen := map[string]bool{}
		kept := ss[:0]
		for _, s := range ss {
			if seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			kept = append(kept, s)
		}
		out[i].Sessions = kept
	}
	return out
}

func sessionsIn(dir string) []SessionMeta {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []SessionMeta
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		p := filepath.Join(dir, e.Name())
		sc, ok := scanSession(p, info.Size(), info.ModTime().UnixNano())
		if !ok {
			continue
		}
		title := sc.title()
		if title == "" { // 没有任何用户输入的会话不展示
			continue
		}
		if isInternalPrompt(sc.FirstUser) { // 应用自己的一次性调用(旧版本没加 --no-session-persistence)留下的会话
			continue
		}
		out = append(out, SessionMeta{ID: strings.TrimSuffix(e.Name(), ".jsonl"), Path: p, Cwd: sc.Cwd, Title: title, Modified: info.ModTime()})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out
}

// internalPrompts 是 MiniCode 自己做一次性 `claude -p` 调用时的提示词开头(AI 提交说明、行内补全 / 编辑)。
// 旧版本这类调用会在 ~/.claude/projects 留下会话,混进用户的历史列表;现在的调用已加 --no-session-persistence,
// 这里只负责不展示老版本留下的那些。
var internalPrompts = []string{
	"你是 Git 提交说明助手",
	"你是代码编辑助手。只输出要替换进编辑器的代码片段",
	"你是代码补全助手。只输出要接在光标后的补全文本",
}

func isInternalPrompt(first string) bool {
	first = strings.TrimSpace(first)
	for _, p := range internalPrompts {
		if strings.HasPrefix(first, p) {
			return true
		}
	}
	return false
}

// DeleteSession 删除某个会话的历史文件;id 只允许字母、数字、-、_,防止借 .. 删到目录之外。文件不存在视为成功。
func DeleteSession(projects, cwd, id string) error {
	if id == "" || strings.IndexFunc(id, func(r rune) bool {
		return !(r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_'))
	}) >= 0 {
		return fmt.Errorf("会话 id 不合法")
	}
	err := os.Remove(filepath.Join(projects, EncodeProjectDir(cwd), id+".jsonl"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ---------- 侧栏数据 ----------

// Located 是按会话 id 反查到的位置。
type Located struct{ Path, Cwd, Title string }

// Index 是一次扫描的结果:侧栏项目列表 + 会话 id → 位置。
type Index struct {
	Projects []core.ProjectInfo
	byID     map[string]Located
}

// Locate 按会话 id 找文件与工作目录。
func (ix *Index) Locate(id string) (Located, bool) {
	if ix == nil {
		return Located{}, false
	}
	l, ok := ix.byID[id]
	return l, ok
}

// MaxSidebarSessions 是每个项目在侧栏里最多直接带多少条会话(Count 仍是总数)。
const MaxSidebarSessions = 30

// BuildIndex 扫描磁盘并生成侧栏用的项目列表。now 注入便于测试。
func BuildIndex(projects string, now time.Time) *Index {
	ix := &Index{byID: map[string]Located{}}
	for _, p := range ListProjects(projects) {
		info := core.ProjectInfo{ID: p.Cwd, Name: filepath.Base(p.Cwd), Path: p.Cwd, Count: len(p.Sessions)}
		for i, s := range p.Sessions {
			ix.byID[s.ID] = Located{Path: s.Path, Cwd: s.Cwd, Title: s.Title}
			if i >= MaxSidebarSessions {
				continue
			}
			group, clock := TimeLabels(s.Modified, now)
			info.Sessions = append(info.Sessions, core.SessionInfo{
				ID: s.ID, Title: s.Title, Updated: s.Modified.UnixMilli(),
				Label: RelTime(s.Modified, now), Group: group, Clock: clock,
			})
		}
		ix.Projects = append(ix.Projects, info)
	}
	return ix
}

// RelTime 相对时间文案:刚刚 / N 分钟 / N 小时 / 昨天 / 周X / M月D日。
func RelTime(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d 分钟", int(d/time.Minute))
	case sameDay(t, now):
		return fmt.Sprintf("%d 小时", int(d/time.Hour))
	case sameDay(t, now.AddDate(0, 0, -1)):
		return "昨天"
	case d < 7*24*time.Hour:
		return weekday(t)
	}
	return fmt.Sprintf("%d月%d日", int(t.Month()), t.Day())
}

// TimeLabels 返回侧栏分段(今天 / 昨天 / 更早)与行尾时间。
func TimeLabels(t, now time.Time) (group, clock string) {
	switch {
	case sameDay(t, now):
		return "今天", RelTime(t, now)
	case sameDay(t, now.AddDate(0, 0, -1)):
		return "昨天", t.Format("15:04")
	case now.Sub(t) < 7*24*time.Hour:
		return "更早", weekday(t)
	}
	return "更早", fmt.Sprintf("%d月%d日", int(t.Month()), t.Day())
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.In(b.Location()).Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func weekday(t time.Time) string {
	return [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}[t.Weekday()]
}
