package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/customize"
)

func (a *App) cuDemo() bool { return a.demoWs() || a.demoExtra }

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.IndexFunc(s, func(r rune) bool {
		return r > ' ' && r != '/' && r != '-' && r != '_' && r != '.' && r != '@'
	}) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func (a *App) cuDisk() *customize.DiskStore {
	if a.cuDemo() {
		return nil
	}
	if a.cu.disk != nil && a.cu.disk.ProjRoot() == a.proj && a.cust != nil {
		return a.cu.disk
	}
	a.cuEnsureDisk()
	return a.cu.disk
}

func (a *App) cuEnsureDisk() {
	if a.cuDemo() {
		return
	}
	if a.cu.disk != nil && a.cu.disk.ProjRoot() == a.proj {
		return
	}
	ds, err := customize.OpenDisk(a.proj)
	if err != nil {
		log.Warn("加载自定义配置失败", "err", err)
		a.cust = customize.NewStore(nil, nil)
		a.cu.disk = nil
		return
	}
	a.cu.disk, a.cust = ds, ds.Store
	a.bg("刷新MCP状态", func() func() {
		ds.RefreshMCPStatus()
		return func() {}
	})
}

func (a *App) cuSaveDisk(st *customize.Store, it *customize.Item) {
	s := &a.cu
	if err := st.Save(a.cuCat(), it, s.draft); err != nil {
		s.err = err.Error()
		return
	}
	if ds := a.cuDisk(); ds != nil {
		if err := ds.Persist(a.cuCat(), it, s.draft); err != nil {
			s.err = "保存到磁盘失败: " + err.Error()
			log.Warn("自定义写盘失败", "cat", a.cuCat(), "name", it.Name, "err", err)
			return
		}
	}
	s.err, s.dOf = "", nil
	a.toast = "已保存「" + it.Name + "」"
	log.Info("保存自定义项", "cat", a.cuCat(), "name", it.Name)
}

func (a *App) cuOpenOK(it *customize.Item) bool {
	if it.Path == "" {
		return false
	}
	if a.cuCat() == customize.Skills || a.cuCat() == customize.Rules {
		return true
	}
	return a.cuCat() == customize.Commands || a.cuCat() == customize.Agents
}

func (a *App) cuOpenPath(p string) {
	p = strings.TrimSpace(p)
	if p == "" {
		return
	}
	abs := p
	if strings.HasPrefix(p, "~") {
		if h, err := os.UserHomeDir(); err == nil {
			abs = filepath.Join(h, strings.TrimPrefix(p, "~/"))
		}
	} else if a.proj != "" && !filepath.IsAbs(p) {
		abs = filepath.Join(a.proj, p)
	}
	if cat := a.cuCat(); cat == customize.Skills {
		abs = filepath.Join(abs, "SKILL.md")
	}
	if a.mode != ModeIDE {
		a.mode = ModeIDE
	}
	a.openAnyFile(abs, 0, 0)
}

func (a *App) cuAuthReal(it *customize.Item) {
	path, ok := claude.FindClaude("")
	if !ok {
		a.toast = "没有找到 claude 命令"
		return
	}
	line := path + " mcp login " + shellQuote(it.Name)
	if a.proj != "" {
		line = "cd " + shellQuote(a.proj) + " && " + line
	}
	a.mode = ModeIDE
	a.screen = ScreenWork
	a.ide.bottomTab = "term"
	a.ide.tpc().initCmd = line
	a.toast = "已在终端面板填入授权命令,在终端里执行并完成登录"
	log.Info("MCP授权", "name", it.Name)
}

func (a *App) cuReconnectReal(it *customize.Item) {
	if a.cuDemo() {
		a.cuReconnect(it)
		return
	}
	a.bg("MCP重连", func() func() {
		if ds := a.cuDisk(); ds != nil {
			ds.RefreshMCPStatus()
		}
		ok := it.Status == customize.StOK
		name, desc := it.Name, it.Desc
		return func() {
			if ok {
				a.toast = "已连接「" + name + "」"
			} else {
				a.toast = "仍无法连接「" + name + "」:" + desc
			}
		}
	})
}

func (a *App) cuMCPOn(it *customize.Item, on bool) {
	it.On = on
	if ds := a.cuDisk(); ds != nil {
		if err := ds.SetMCPEnabled(it.Name, on); err != nil {
			a.toast = "切换失败:" + err.Error()
			it.On = !on
			return
		}
	}
	log.Info("MCP启停", "name", it.Name, "on", on)
}

func (a *App) cuMCPTool(it *customize.Item, i int) {
	if i < 0 || i >= len(it.Tools) {
		return
	}
	it.ToggleTool(i)
	if ds := a.cuDisk(); ds != nil {
		_ = ds.SetMCPTool(it.Name, it.Tools[i].Name, it.Tools[i].On)
	}
}

func (a *App) cuTryCommand(it *customize.Item) {
	prompt := strings.TrimSpace(it.Body)
	if prompt == "" {
		a.toast = "提示词为空"
		return
	}
	dir := a.proj
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	a.cu.runOut = "正在试运行…"
	a.bg("命令试运行", func() func() {
		out, err := claude.RunPrompt(dir, prompt, "plan", 2*time.Minute)
		return func() {
			if err != nil {
				a.cu.runOut = "退出: 失败\n" + err.Error()
			} else {
				a.cu.runOut = out
			}
		}
	})
}

func (a *App) cuTryAgent(it *customize.Item) {
	prompt := strings.TrimSpace(it.Body)
	if prompt == "" {
		a.toast = "系统提示词为空"
		return
	}
	dir := a.proj
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	a.cu.runOut = "正在试运行…"
	a.bg("子Agent试运行", func() func() {
		wrapped := "你是子 Agent「" + it.Name + "」。\n\n" + prompt + "\n\n用一句话确认你已就绪。"
		model := ""
		if it.Model > 0 && it.Model < len(customize.Models) {
			switch it.Model {
			case 1:
				model = "sonnet"
			case 2:
				model = "opus"
			case 3:
				model = "haiku"
			}
		}
		out, err := claude.RunPromptModel(dir, wrapped, "plan", model, 2*time.Minute)
		return func() {
			if err != nil {
				a.cu.runOut = "退出: 失败\n" + err.Error()
			} else {
				a.cu.runOut = out
			}
		}
	})
}

func (a *App) cuTryHook(d *customize.Item) {
	cmd := strings.TrimSpace(d.Cmd)
	if cmd == "" {
		a.cu.hookOut = "命令为空"
		return
	}
	a.confirmDlg("试运行这个钩子?", "$ "+cmd+"\n超时 "+itoa(d.Timeout)+" 秒", "运行", false, func() {
		a.cuRunHook(cmd, d.Timeout)
	})
}

func (a *App) cuRunHook(cmd string, timeout int) {
	if timeout < 1 {
		timeout = 10
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	a.cu.hookOut = "运行中…"
	a.bg("钩子试运行", func() func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()
		c := exec.CommandContext(ctx, shell, "-lc", cmd)
		if a.proj != "" {
			c.Dir = a.proj
		}
		out, err := c.CombinedOutput()
		code := 0
		if c.ProcessState != nil {
			code = c.ProcessState.ExitCode()
		}
		text := fmt.Sprintf("退出码 %d\n%s", code, strings.TrimSpace(string(out)))
		if ctx.Err() == context.DeadlineExceeded {
			text = fmt.Sprintf("超时(%d 秒)", timeout)
		} else if err != nil && text == fmt.Sprintf("退出码 %d\n", code) {
			text += err.Error()
		}
		return func() { a.cu.hookOut = text }
	})
}

func (a *App) cuInstallPlugin(st *customize.Store, m *customize.Market) {
	if m == nil {
		return
	}
	if a.cuDemo() {
		a.cu.plug = &pluginInstall{name: m.Name, phase: 0}
		a.bg("演示安装插件", func() func() {
			time.Sleep(800 * time.Millisecond)
			a.cu.plug.phase = 1
			time.Sleep(700 * time.Millisecond)
			inst, err := st.Install(m)
			return func() {
				if err == nil {
					a.cu.plug.phase = 2
					a.toast = "已安装「" + inst.Name + "」"
				} else {
					a.cu.plug.phase = 3
					a.cu.plug.err = err.Error()
				}
				a.cu.plug = nil
			}
		})
		return
	}
	a.cu.plug = &pluginInstall{name: m.Name, market: "claude-plugins-official", phase: 0}
	a.bg("安装插件", func() func() {
		res := customize.RunPluginInstall(m.Name, a.cu.plug.market)
		return func() {
			if res.Code != 0 {
				a.cu.plug.phase = 3
				a.cu.plug.err = res.Err
				if res.Err == "" {
					a.cu.plug.err = fmt.Sprintf("退出码 %d", res.Code)
				}
				return
			}
			a.cu.plug.phase = 2
			if ds := a.cuDisk(); ds != nil {
				ds.Reload()
				m.Installed = true
			}
			a.toast = "已安装「" + m.Name + "」"
			a.cu.plug = nil
		}
	})
}

func (a *App) cuRefreshMCP() {
	if ds := a.cuDisk(); ds != nil {
		a.bg("刷新MCP", func() func() {
			ds.RefreshMCPStatus()
			return func() { a.toast = "已刷新 MCP 状态" }
		})
	} else {
		a.toast = "演示模式:mcp 状态为模拟数据"
	}
}

func (a *App) cuRemoveDisk(st *customize.Store, it *customize.Item) {
	if ds := a.cuDisk(); ds != nil {
		if err := ds.RemoveDisk(a.cuCat(), it); err != nil {
			a.toast = "删除失败:" + err.Error()
			return
		}
	}
	if a.cuCat() == customize.Plugins {
		st.Uninstall(it)
	} else {
		st.Remove(a.cuCat(), it)
	}
}

// usageLimitsFromSession 把会话侧的限额抄到用量页缓存。
func (a *App) usageLimitsFromSession() {
	if a.st == nil {
		return
	}
	u := a.st.Usage
	prev := a.use.limitsState
	a.use.limits = limitsFromUsage(u)
	a.use.limitsAt = u.LimitsAt
	a.use.limitsState = u.LimitsState
	if u.LimitsState == core.LimitsUnavailable && u.LimitsAt == 0 && prev == core.LimitsChecking {
		a.use.limitsAt = time.Now().Unix()
	}
}

func limitsFromUsage(u core.Usage) (out []core.Limit) {
	if u.FiveHourPct != nil {
		out = append(out, core.Limit{Name: "5 小时窗口", Percent: int(*u.FiveHourPct + .5), Reset: resetText(u.FiveHourResets)})
	}
	if u.WeeklyPct != nil {
		out = append(out, core.Limit{Name: "每周", Percent: int(*u.WeeklyPct + .5), Reset: resetText(u.WeeklyResets)})
	}
	return out
}

func resetText(ts int64) string {
	if ts == 0 {
		return ""
	}
	d := time.Until(time.Unix(ts, 0))
	if d < 0 {
		return "已重置"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d 分钟后重置", int(d.Minutes()+.5))
	}
	return fmt.Sprintf("%d 小时 %d 分后重置", int(d.Hours()), int(d.Minutes())%60)
}

func (a *App) usageRefreshLimits(force bool) {
	if a.cuDemo() {
		a.use.limitsUpdated = "刚刚"
		a.toast = "演示模式:限额为演示数据"
		return
	}
	if !force && a.use.limitsAt > 0 && time.Now().Unix()-a.use.limitsAt < core.LimitsTTLSecs {
		return
	}
	a.use.limitsState = core.LimitsChecking
	a.send(core.CmdRefreshUsage{Force: force})
}

func relLimitsAge(at int64) string {
	if at == 0 {
		return "尚未更新"
	}
	d := time.Since(time.Unix(at, 0))
	if d < time.Minute {
		return "刚刚"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d 分钟前", int(d.Minutes()+.5))
	}
	return fmt.Sprintf("%d 小时前", int(d.Hours()+.5))
}
