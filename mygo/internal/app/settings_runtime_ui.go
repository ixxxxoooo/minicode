package app

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// claudeCmdRow 是运行环境里的「可执行文件 + 检测」与版本 tag。
func (a *App) claudeCmdRow(c *ui.Context) {
	row := setRow{key: pRuntimeBin, title: "可执行文件", desc: "MiniCode 直连本机的 claude(stream-json 协议),不经过 Node;对新会话生效", kind: kField, def: "", w: 210, mono: true}
	ok, _ := a.RuntimeBinValid()
	if !ok {
		a.settingsFieldErr(c, row, "找不到可执行文件")
	} else {
		a.settingsRow(c, row, false)
	}
	ui.Row(c).Gap(8).Padding(0, 12, 8, 12).Children(func() {
		kit.Btn(c, "检测", kit.BtnOpt{Small: true, Icon: "refresh"}).OnClick(func() { a.runClaudeDetect() })
	})
	if a.set.claudeVer == "" && !a.set.claudeDetecting {
		a.settingsRow(c, rCustom("版本", "点「检测」获取", func(a2 *App, c2 *ui.Context) {
			ui.Text(c2, "尚未检测").FontSize(theme.FsCap).TextColor(a2.tk.Tx3)
		}), false)
		return
	}
	a.settingsRow(c, rCustom("版本", "", func(a2 *App, c2 *ui.Context) {
		if a2.set.claudeDetecting {
			kit.Tag(c2, "检测中…", kit.KWarn)
			return
		}
		if a2.set.claudeVerErr != "" {
			kit.Tag(c2, a2.set.claudeVerErr, kit.KErr)
			return
		}
		kit.Tag(c2, a2.set.claudeVer, kit.KOk)
	}), false)
}

func (a *App) runClaudeDetect() {
	a.set.claudeDetecting = true
	a.set.claudeVerErr = ""
	a.bg("claude-version", func() func() {
		end := diag.SpanSlow("app", "claude --version", 5*time.Second)
		ver, err := detectClaudeVersion(a, a.claudeBinForDetect())
		end()
		return func() {
			a.set.claudeDetecting = false
			if err != nil {
				a.set.claudeVerErr = "检测失败"
				setLog.Warn("claude 版本检测失败", "err", err)
				return
			}
			a.set.claudeVer = strings.TrimSpace(ver)
			setLog.Info("claude 版本", "ver", a.set.claudeVer)
		}
	})
}

func (a *App) claudeBinForDetect() string {
	if a.demoWs() {
		return ""
	}
	if ok, path := a.RuntimeBinValid(); ok && path != "" {
		return path
	}
	if p, ok := claude.FindClaude(a.prefs.Str(pRuntimeBin, "")); ok {
		return p
	}
	return ""
}

func detectClaudeVersion(a *App, bin string) (string, error) {
	if a.demoWs() {
		return demo.DemoClaudeVersion(), nil
	}
	if bin == "" {
		return "", exec.ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").CombinedOutput()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	if line == "" {
		return strings.TrimSpace(string(out)), nil
	}
	return line, nil
}

// settingsFieldErr 带错误态的 Field 行。
func (a *App) settingsFieldErr(c *ui.Context, r setRow, err string) {
	t := a.tk
	if a.set.bufs == nil {
		a.set.bufs = map[string]*string{}
	}
	buf, ok := a.set.bufs[r.key]
	if !ok {
		s := a.prefStr(r)
		buf = &s
		a.set.bufs[r.key] = buf
	}
	rowBox(a, c, false, 54).Children(func() {
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Text(c, r.title).FontWeight(500)
			if r.desc != "" {
				ui.Text(c, r.desc).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.45)
			}
		})
		kit.Field(c, r.key, kit.FieldOpt{Err: err}, func(box ui.Element) {
			box.BorderWidth(0).Background(ui.Transparent).PaddingX(0).Height(24).Width(r.w).Shrink(0)
			in := ui.TextInputBase(c.Key("fld-"+r.key), buf).Grow(1).FontSize(theme.FsCap + .5).Font("monospace")
			in.OnChange(func() { a.setPref(r, *buf) })
		})
		kit.Ib(c, "undo", kit.IbOpt{Small: true, Tip: "恢复默认"}).OnClick(func() { a.resetPref(r) })
	})
}
