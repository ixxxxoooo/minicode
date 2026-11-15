package app

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) privacyCrashesBody(c *ui.Context) {
	t := a.tk
	recs := a.crashesCached()
	if len(recs) == 0 {
		rowBox(a, c, true, 36).Children(func() {
			kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
			if a.set.crAt.IsZero() {
				ui.Text(c, "正在读取…").TextColor(t.Tx3).Grow(1)
			} else {
				ui.Text(c, "最近 30 天没有崩溃记录").Grow(1)
			}
		})
		return
	}
	for i, r := range recs {
		r := r
		rowBox(a, c, i == 0, 40).Children(func() {
			kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				title, sub := crashKindLabel(r.Kind), r.When.Format("2006-01-02 15:04")
				if r.NoMsg {
					sub += " · " + r.File // 没有摘要的旧记录:文件名放副标题
				} else {
					title += " · " + r.Message
				}
				ui.Text(c, title).SingleLine()
				ui.Text(c, sub).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
			})
		})
	}
}

type crashBrief struct {
	Kind, Message string
	File          string // 记录文件名
	NoMsg         bool   // 记录里没有摘要
	When          time.Time
}

// crashKindLabel 把记录类别翻成界面用语;不认识的原样显示。
func crashKindLabel(kind string) string {
	switch kind {
	case "panic":
		return "程序崩溃"
	case "hang":
		return "界面卡死"
	case "unclean-exit":
		return "未正常退出"
	case "abnormal-exit":
		return "异常退出"
	case "fatal":
		return "致命错误"
	case "ips":
		return "系统崩溃报告"
	case "native":
		return "原生崩溃"
	case "":
		return "记录"
	}
	return kind
}

// crashesCached 返回缓存的崩溃记录(5 秒过期,后台读盘,渲染线程不碰磁盘)。
func (a *App) crashesCached() []crashBrief {
	if !a.set.crBusy && time.Since(a.set.crAt) > 5*time.Second {
		a.set.crBusy = true
		a.bg("crash-list", func() func() {
			recs := recentCrashes(30)
			return func() {
				a.set.cr = recs
				a.set.crAt = time.Now()
				a.set.crBusy = false
			}
		})
	}
	return a.set.cr
}

func recentCrashes(days int) []crashBrief {
	dir := diag.CrashDir()
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	cut := time.Now().AddDate(0, 0, -days)
	var out []crashBrief
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var rec diag.CrashRecord
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		if json.Unmarshal(b, &rec) != nil {
			continue
		}
		when := time.Time{}
		if rec.Time != "" {
			when, _ = time.Parse(time.RFC3339, rec.Time)
		}
		if when.IsZero() {
			if info, err := e.Info(); err == nil {
				when = info.ModTime()
			}
		}
		if when.Before(cut) {
			continue
		}
		out = append(out, crashBrief{Kind: rec.Kind, Message: rec.Message, File: e.Name(), NoMsg: rec.Message == "", When: when})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].When.After(out[j].When) })
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}

func (a *App) exportDiagBundle() {
	a.bg("diag-export", func() func() {
		end := diag.SpanSlow("app", "导出诊断包", 10*time.Second)
		path, err := writeDiagZip(a)
		end()
		return func() {
			if err != nil {
				a.toast = "导出失败:" + err.Error()
				setLog.Warn("导出诊断包失败", "err", err)
				return
			}
			a.toast = "已导出诊断包"
			_ = exec.Command("open", "-R", path).Start()
			setLog.Info("导出诊断包", "path", path)
		}
	})
}

func writeDiagZip(a *App) (string, error) {
	dir := filepath.Join(diag.Home(), "diag")
	_ = os.MkdirAll(dir, 0o755)
	name := fmt.Sprintf("diag-%s.zip", time.Now().Format("20060102-150405"))
	outPath := filepath.Join(dir, name)
	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	summary := map[string]any{
		"version": Version,
		"home":    "~/.minicode",
		"time":    time.Now().Format(time.RFC3339),
	}
	if a.prefs != nil {
		summary["prefKeys"] = a.prefs.Keys()
	}
	summary["note"] = "不含环境变量值与密钥"
	b, _ := json.MarshalIndent(summary, "", "  ")
	_ = zipWrite(zw, "summary.json", b)
	logPath := diag.LogPath()
	if b, err := os.ReadFile(logPath); err == nil && len(b) < 2<<20 {
		_ = zipWrite(zw, "logs/minicode.log.tail", tailBytes(b, 200<<10))
	}
	return outPath, nil
}

func zipWrite(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, strings.NewReader(string(data)))
	return err
}

func tailBytes(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return b[len(b)-n:]
}

func (a *App) runClaudeDoctor() {
	if a.set.doctorRunning {
		return
	}
	a.set.doctorRunning = true
	a.set.doctorLines = nil
	a.bg("claude-doctor", func() func() {
		end := diag.SpanSlow("app", "claude doctor", 30*time.Second)
		lines, err := runDoctor(a)
		end()
		return func() {
			a.set.doctorRunning = false
			a.set.doctorOpen = true
			if err != nil {
				a.set.doctorLines = []string{"运行失败:" + err.Error()}
				return
			}
			a.set.doctorLines = lines
		}
	})
}

func runDoctor(a *App) ([]string, error) {
	if a.demoWs() {
		time.Sleep(400 * time.Millisecond)
		return demo.DemoClaudeDoctor(), nil
	}
	bin := a.claudeBinForDetect()
	if bin == "" {
		return nil, fmt.Errorf("找不到 claude 可执行文件")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "doctor").CombinedOutput()
	if err != nil && len(out) == 0 {
		return nil, err
	}
	var lines []string
	for _, ln := range strings.Split(string(out), "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			lines = append(lines, ln)
		}
	}
	if len(lines) == 0 {
		lines = []string{strings.TrimSpace(string(out))}
	}
	return lines, nil
}

func (a *App) doctorFoldBody(c *ui.Context) {
	if !a.set.doctorOpen && len(a.set.doctorLines) == 0 {
		return
	}
	t := a.tk
	ui.Column(c).Gap(4).Padding(0, 12, 12, 12).Children(func() {
		for _, ln := range a.set.doctorLines {
			ui.Text(c, ln).FontSize(theme.FsCap).TextColor(t.Tx2).LineHeight(1.5)
		}
	})
}
