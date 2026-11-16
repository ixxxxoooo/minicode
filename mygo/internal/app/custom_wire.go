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
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
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
	// 插件市场的真实内容由 OpenDisk 直接读 ~/.claude/plugins/marketplaces(不起子进程)。
	// 此前这里还会后台跑 claude plugin marketplace list,它列的是「市场」而不是插件,会把真实列表换成假数据。
}

// cuPluginToggle 启用 / 停用已安装的插件:先按点击结果显示,后台跑 claude plugin enable|disable 落盘,失败则还原并提示。
func (a *App) cuPluginToggle(it *customize.Item) {
	on := !it.On
	if a.cuDemo() {
		it.On = on
		a.toast = map[bool]string{true: "已启用「", false: "已停用「"}[on] + it.Name + "」(演示)"
		return
	}
	ds := a.cuDisk()
	id, proj := customize.PluginID(it), a.proj
	it.On = on
	a.toast = "正在" + map[bool]string{true: "启用", false: "停用"}[on] + "「" + it.Name + "」…"
	a.bg("切换插件", func() func() {
		res := customize.RunPluginToggle(id, on, proj)
		return func() {
			if res.Code != 0 {
				it.On = !on
				a.toast = "切换失败:" + customize.ShortMCPErr(res.Err)
				log.Warn("插件启停失败", "id", id, "on", on, "err", res.Err)
				return
			}
			if ds != nil && a.cu.disk == ds {
				it.On = on
			}
			a.toast = map[bool]string{true: "已启用「", false: "已停用「"}[on] + it.Name + "」,新会话生效"
			log.Info("插件启停", "id", id, "on", on)
		}
	})
}

// cuUninstallPlugin 后台跑 claude plugin uninstall <名称@市场>,成功后重读磁盘;失败给出原因,列表不动。
func (a *App) cuUninstallPlugin(it *customize.Item) {
	ds := a.cuDisk()
	id, name := customize.PluginID(it), it.Name
	a.toast = "正在卸载「" + name + "」…"
	a.bg("卸载插件", func() func() {
		res := customize.RunPluginUninstall(id)
		return func() {
			if res.Code != 0 {
				a.toast = "卸载失败:" + customize.ShortMCPErr(res.Err)
				log.Warn("插件卸载失败", "id", id, "err", res.Err)
				return
			}
			if ds != nil && a.cu.disk == ds {
				ds.Reload()
			}
			a.cu.sel, a.cu.dOf, a.cu.delArm = nil, nil, nil
			a.toast = "已卸载「" + name + "」"
			log.Info("插件卸载", "id", id)
		}
	})
}

// cuAddMarketplace 选一个本地目录当作插件市场添加(claude plugin marketplace add <目录>)。
func (a *App) cuAddMarketplace() {
	if a.cuDemo() {
		a.toast = "演示模式不会真的添加插件市场"
		return
	}
	a.pickFolders(false, func(ps []string) {
		src := ps[0]
		ds := a.cuDisk()
		a.toast = "正在添加插件市场…"
		a.bg("添加插件市场", func() func() {
			res := customize.RunMarketplaceAdd(src)
			return func() {
				if res.Code != 0 {
					a.toast = "添加失败:" + customize.ShortMCPErr(res.Err)
					return
				}
				if ds != nil && a.cu.disk == ds {
					ds.Reload()
				}
				a.toast = "已添加插件市场,可在「市场」里浏览"
				a.cu.market = true
			}
		})
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
	abs, ok := a.resolveUserPath(p)
	if !ok {
		a.toast = "路径无效或无法解析"
		return
	}
	if cat := a.cuCat(); cat == customize.Skills {
		if st, err := os.Stat(abs); err != nil || st.IsDir() {
			abs = filepath.Join(abs, "SKILL.md")
		}
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
	ds := a.cuDisk()
	a.bg("MCP重连", func() func() {
		var fresh []*customize.Item
		var st *customize.MCPStatus
		if ds != nil {
			fresh = ds.ReadMCPConfig()
			st, _ = ds.FetchMCPStatus() // 子进程,只能在后台跑
		}
		return func() {
			if ds != nil && a.cu.disk == ds {
				ds.ApplyMCPConfig(fresh)
				ds.ApplyMCPStatus(st)
			}
			ok := it.Status == customize.StOK
			name, desc := it.Name, it.Desc
			a.mcpManualRefresh(a.mcpRuntimeReady())
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
	if a.mcpOn == nil {
		a.mcpOn = map[string]mcpOverride{}
	}
	a.mcpOn[it.Name] = mcpOverride{on: on, at: time.Now()}
	if ds := a.cuDisk(); ds != nil {
		if err := ds.SetMCPEnabled(it.Name, on); err != nil {
			a.toast = "切换失败:" + err.Error()
			it.On = !on
			delete(a.mcpOn, it.Name) // 没写成功就不能再按「点过的结果」显示
			return
		}
	}
	log.Info("MCP启停", "name", it.Name, "on", on)
	a.hotToggleMcp(it.Name, on)
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

// cuDraftBody 返回当前项的草稿正文(详情里编辑的是 draft,试运行应跟表单一致)。
func (a *App) cuDraftItem(it *customize.Item) customize.Item {
	s := &a.cu
	if s.dOf == it {
		return s.draft
	}
	return it.Clone()
}

func (a *App) cuTryCommand(it *customize.Item) {
	prompt := strings.TrimSpace(a.cuDraftItem(it).Body)
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
	d := a.cuDraftItem(it)
	prompt := strings.TrimSpace(d.Body)
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
		wrapped := "你是子 Agent「" + d.Name + "」。\n\n" + prompt + "\n\n用一句话确认你已就绪。"
		model := ""
		if d.Model > 0 && d.Model < len(customize.Models) {
			switch d.Model {
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
	mp := m.MP
	if mp == "" {
		mp = "claude-plugins-official"
	}
	a.cu.plug = &pluginInstall{name: m.Name, market: mp, phase: 0}
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
	if a.cuDemo() {
		a.mcpManualRefresh(false)
		a.toast = "演示模式:已刷新 MCP 状态"
		return
	}
	a.mcpManualRefresh(false)
	if !a.mcpRuntimeReady() {
		a.toast = "已重读配置;发送第一条消息后可查看连接状态"
		return
	}
	a.toast = "正在刷新 MCP 状态…"
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
	// 自动路径(用量页每帧都会走到这里)必须节流:查询失败时 limitsAt 仍是 0,
	// 没有节流就每帧重发 CmdRefreshUsage,状态在「更新中 / 查询失败」之间来回翻,整页闪烁。
	if !force && time.Since(a.use.limitsTry) < time.Duration(core.LimitsTTLSecs)*time.Second {
		return
	}
	a.use.limitsTry = time.Now()
	a.use.limitsState = core.LimitsChecking
	a.send(core.CmdRefreshUsage{Limits: true, Force: force})
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

// cuMCPOnDemand 切换「按需加载工具」。它对应 Claude 的 ENABLE_TOOL_SEARCH,是全局设置(不分服务器):
// 开 = auto(工具很多时只在需要时加载描述),关 = false(启动就把全部工具描述放进上下文)。
// 写进它当前所在的配置层(本地 > 项目 > 用户,都没有就写用户层),所以改了一定生效;新会话(新进程)起效。
// 演示模式只改内存里的显示。
func (a *App) cuMCPOnDemand(on bool) {
	set := func() {
		for _, it := range a.cuStore().List(customize.MCP, customize.All, "") {
			it.OnDemand = on
		}
	}
	ds := a.cuDisk()
	if ds == nil || a.cuDemo() {
		set()
		return
	}
	_, layer := ds.ToolSearchValue()
	lay := claudecfg.LayerUser
	switch layer {
	case 1:
		lay = claudecfg.LayerProject
	case 2:
		lay = claudecfg.LayerLocal
	}
	val := "false"
	if on {
		val = "auto"
	}
	if err := claudecfg.SetEnv("ENABLE_TOOL_SEARCH", val, lay, a.proj); err != nil {
		a.toast = "切换失败:" + err.Error()
		log.Warn("写入 ENABLE_TOOL_SEARCH 失败", "err", err)
		return
	}
	set()
	log.Info("按需加载工具", "on", on, "层", layer)
	a.toast = "已" + map[bool]string{true: "开启", false: "关闭"}[on] + "按需加载工具,新会话生效"
}
