package claude

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 启动开销剖析(只量数据,不做断言;会产生少量 token):
//
//	cd mygo && MINI_REAL_CLAUDE=1 MINI_PROFILE=1 ../scripts/with-deepseek.sh go test ./internal/claude -run RealStartupProfile -v -timeout 30m
//
// 量三件事:
//  1. 握手耗时:spawn → initialize 应答(预热能搬走的部分),分「空目录 / 真实项目 / 恢复小会话 / 恢复大会话」;
//  2. 空闲开销:握手后进程树的内存与 CPU(预热的代价);
//  3. 冷 vs 预热的首响应(预热收益)。
func TestRealStartupProfile(t *testing.T) {
	realSkip(t)
	if os.Getenv("MINI_PROFILE") != "1" {
		t.Skip("设置 MINI_PROFILE=1 才跑启动剖析")
	}
	repo := os.Getenv("MINI_PROFILE_REPO") // 真实项目目录(带 CLAUDE.md / 插件配置)
	if repo == "" {
		repo = filepath.Join(os.Getenv("HOME"), "MiniCode")
	}
	// 恢复用的会话:把真实项目里最大的几份会话拷到临时目录的项目下(只读不改原件)
	small, large := pickSessions(t, repo)

	type scen struct {
		name   string
		cwd    string
		resume string // 会话 id
		note   string
		prompt bool // 是否也量冷 / 预热首响应(会发请求)
		clean  func()
	}
	var scens []scen
	scens = append(scens, scen{name: "空目录·新会话", cwd: tmpProj(t, nil), prompt: true})
	scens = append(scens, scen{name: "真实项目·新会话", cwd: repo, note: "不发提示,不落盘"})
	if small.path != "" {
		cwd, id, clean := copyResume(t, small.path)
		scens = append(scens, scen{name: fmt.Sprintf("恢复小会话(%s)", human(small.size)), cwd: cwd, resume: id, prompt: true, clean: clean})
	}
	if large.path != "" {
		cwd, id, clean := copyResume(t, large.path)
		scens = append(scens, scen{name: fmt.Sprintf("恢复大会话(%s)", human(large.size)), cwd: cwd, resume: id, clean: clean, note: "只量握手"})
	}

	const nHand, nPrompt = 5, 3
	mode := core.ModePlan
	var report []string
	for _, sc := range scens {
		if sc.clean != nil {
			defer sc.clean()
		}
		cfg := func() Config {
			c := Config{Mode: &mode, Cwd: sc.cwd}
			if sc.resume != "" {
				c.SessionID = sc.resume
			}
			return c
		}
		var hs []float64
		var rssAfter, rssIdle, nproc []float64
		var cpuIdle []float64
		for i := 0; i < nHand; i++ {
			r := startRealRig(t, cfg())
			r.b.Send(core.CmdWarmup{})
			t0 := time.Now()
			r.waitTimeout(90*time.Second, "握手完成", func() bool { return r.st.Conn == core.ConnReady })
			wall := time.Since(t0) // 从发出预热命令到握手完成:进程启动 + 握手
			hs = append(hs, float64(wall.Milliseconds()))
			time.Sleep(200 * time.Millisecond)
			p := r.b.p
			if i == 0 { // 空闲开销只在第一轮量:握手后立刻、再空闲 20 秒后
				m0 := treeStat(p.pid)
				c0 := treeCPU(p.pid)
				time.Sleep(20 * time.Second)
				m1 := treeStat(p.pid)
				c1 := treeCPU(p.pid)
				rssAfter = append(rssAfter, m0.rssMB)
				rssIdle = append(rssIdle, m1.rssMB)
				nproc = append(nproc, float64(m1.n))
				cpuIdle = append(cpuIdle, (c1-c0)/20*100) // 20 秒内 CPU 秒数 → 平均占用百分比
			}
			r.b.Close()
			time.Sleep(500 * time.Millisecond)
		}
		line := fmt.Sprintf("%-22s 握手 ms:%s | 进程树 握手后 %.0f MB → 空闲20s后 %.0f MB(%d 个进程)| 空闲 CPU %.2f%%",
			sc.name, stats(hs), rssAfter[0], rssIdle[0], int(nproc[0]), cpuIdle[0])
		report = append(report, line)
		t.Log(line)

		if !sc.prompt {
			continue
		}
		var cold, warm [][3]float64 // 首响应 / 总 / 握手
		for i := 0; i < nPrompt; i++ {
			for _, w := range []bool{false, true} {
				r := startRealRig(t, cfg())
				if w {
					r.b.Send(core.CmdWarmup{})
					r.waitTimeout(90*time.Second, "握手完成", func() bool { return r.st.Conn == core.ConnReady })
					time.Sleep(300 * time.Millisecond)
				}
				r.b.Send(core.CmdPrompt{Text: "只回复两个字母:ok,不要调用任何工具。"})
				r.waitTimeout(180*time.Second, "回合结束", func() bool { return r.turnsEndedLocked() == 1 })
				var tm core.TurnTiming
				r.mu.Lock()
				for _, e := range r.evs {
					if x, ok := e.(core.EvTurnTiming); ok {
						tm = x.Timing
					}
				}
				r.mu.Unlock()
				row := [3]float64{float64(tm.FirstMs), float64(tm.TotalMs), float64(tm.StartupMs)}
				if w {
					warm = append(warm, row)
				} else {
					cold = append(cold, row)
				}
				r.b.Close()
				time.Sleep(500 * time.Millisecond)
			}
		}
		col := func(rows [][3]float64, k int) []float64 {
			var o []float64
			for _, r := range rows {
				o = append(o, r[k])
			}
			return o
		}
		l1 := fmt.Sprintf("%-22s 冷启动  首响应 ms:%s | 总 ms:%s | 其中启动握手 ms:%s", "", stats(col(cold, 0)), stats(col(cold, 1)), stats(col(cold, 2)))
		l2 := fmt.Sprintf("%-22s 预热后  首响应 ms:%s | 总 ms:%s", "", stats(col(warm, 0)), stats(col(warm, 1)))
		report = append(report, l1, l2)
		t.Log(l1)
		t.Log(l2)
	}
	_ = os.WriteFile("/tmp/mc-startup-profile.txt", []byte(strings.Join(report, "\n")+"\n"), 0o644)
}

// ---------- 辅助 ----------

type sessFile struct {
	path string
	size int64
}

// pickSessions 在真实项目的会话里挑一份约 0.5 MB 的「小」会话和最大的一份。
func pickSessions(t *testing.T, repo string) (small, large sessFile) {
	dir := filepath.Join(ProjectsDir(), EncodeProjectDir(repo))
	ents, _ := os.ReadDir(dir)
	var fs []sessFile
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			if fi, err := e.Info(); err == nil {
				fs = append(fs, sessFile{filepath.Join(dir, e.Name()), fi.Size()})
			}
		}
	}
	sort.Slice(fs, func(i, j int) bool { return fs[i].size > fs[j].size })
	for _, f := range fs {
		if large.path == "" {
			large = f
		}
		if f.size < 700<<10 && f.size > 200<<10 && small.path == "" {
			small = f
		}
	}
	return
}

func human(n int64) string {
	if n >= 1<<20 {
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%d KB", n>>10)
}

func tmpProj(t *testing.T, files map[string]string) string {
	t.Helper()
	d, err := os.MkdirTemp("/tmp", "mc-prof-")
	if err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(d)
	for n, c := range files {
		_ = os.WriteFile(filepath.Join(real, n), []byte(c), 0o644)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(real)
		_ = os.RemoveAll(filepath.Join(ProjectsDir(), EncodeProjectDir(real)))
	})
	return real
}

// copyResume 把会话文件拷进临时目录对应的项目下,返回可恢复的 cwd 与会话 id。
func copyResume(t *testing.T, src string) (cwd, id string, clean func()) {
	t.Helper()
	cwd = tmpProj(t, nil)
	id = strings.TrimSuffix(filepath.Base(src), ".jsonl")
	dst := filepath.Join(ProjectsDir(), EncodeProjectDir(cwd))
	_ = os.MkdirAll(dst, 0o755)
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, id+".jsonl"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	return cwd, id, func() { _ = os.RemoveAll(dst) }
}

type treeInfo struct {
	rssMB float64
	n     int
}

// psTree 取 pid 及其全部后代的 (pid, rss KB, cputime 秒)。
func psTree(root int) (rows [][3]float64) {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,rss=,time=").Output()
	if err != nil {
		return nil
	}
	type pr struct {
		pid, ppid int
		rss       float64
		cpu       float64
	}
	var all []pr
	for _, ln := range strings.Split(string(out), "\n") {
		f := strings.Fields(ln)
		if len(f) < 4 {
			continue
		}
		pid, _ := strconv.Atoi(f[0])
		ppid, _ := strconv.Atoi(f[1])
		rss, _ := strconv.ParseFloat(f[2], 64)
		all = append(all, pr{pid, ppid, rss, parseCPUTime(f[3])})
	}
	in := map[int]bool{root: true}
	for changed := true; changed; {
		changed = false
		for _, p := range all {
			if in[p.ppid] && !in[p.pid] {
				in[p.pid] = true
				changed = true
			}
		}
	}
	for _, p := range all {
		if in[p.pid] {
			rows = append(rows, [3]float64{float64(p.pid), p.rss, p.cpu})
		}
	}
	return
}

// parseCPUTime 解析 ps 的 time 列(mm:ss.xx 或 h:mm:ss)。
func parseCPUTime(s string) float64 {
	parts := strings.Split(s, ":")
	var sec float64
	for _, p := range parts {
		v, _ := strconv.ParseFloat(p, 64)
		sec = sec*60 + v
	}
	return sec
}

func treeStat(pid int) treeInfo {
	var ti treeInfo
	for _, r := range psTree(pid) {
		ti.rssMB += r[1] / 1024
		ti.n++
	}
	return ti
}

func treeCPU(pid int) float64 {
	var s float64
	for _, r := range psTree(pid) {
		s += r[2]
	}
	return s
}

func stats(v []float64) string {
	if len(v) == 0 {
		return "-"
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	var sum float64
	for _, x := range s {
		sum += x
	}
	return fmt.Sprintf("n=%d 最小 %.0f / 中位 %.0f / 平均 %.0f / 最大 %.0f", len(s), s[0], s[len(s)/2], sum/float64(len(s)), s[len(s)-1])
}
