package app

import (
	"os"
	"os/exec"
	"strings"
)

// virtNewWindowWhy 虚拟标签「在新窗口中打开」不可用时的人类可读原因("" = 可以)。
func (a *App) virtNewWindowWhy(kind string) string {
	if a.proj == "" {
		return "演示模式没有真实项目目录"
	}
	switch kind {
	case vPlan, vSub, vTask:
		if !a.demoWs() && !a.demoExtra {
			return "新窗口不会加载当前会话,计划 / 子 Agent / 任务只能在演示模式或本会话里查看"
		}
	}
	return ""
}

// SpawnVirtWindow 在新进程打开虚拟工作区标签(审查 / 计划 / 任务等)。
func (a *App) SpawnVirtWindow(tb *ideTab) {
	if tb == nil || tb.virt == "" {
		return
	}
	if why := a.virtNewWindowWhy(tb.virt); why != "" {
		a.toast = why
		return
	}
	exe, err := os.Executable()
	if err != nil {
		nwLog.Warn("无法取得可执行文件路径", "err", err)
		return
	}
	spec := tb.virt
	if tb.ref != "" {
		spec += ":" + tb.ref
	}
	args := []string{"--new-window", "--project", a.proj, "--virt", spec}
	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), "MINI_VIRT="+spec)
	if a.curSess != "" {
		cmd.Env = append(cmd.Env, "MINI_SESSION="+a.curSess)
	}
	if err := cmd.Start(); err != nil {
		nwLog.Warn("启动虚拟新窗口失败", "err", err)
		a.toast = "无法打开新窗口"
		return
	}
	nwLog.Info("新窗口打开虚拟标签", "virt", spec)
	a.toast = "已在新窗口打开"
}

// ApplyStartupVirt 从环境变量或启动参数打开虚拟标签。
func (a *App) ApplyStartupVirt(spec string) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return
	}
	kind, ref, _ := strings.Cut(spec, ":")
	if kind == "" {
		return
	}
	a.SetMode(ModeIDE)
	a.screen = ScreenWork
	a.rightOn = true
	a.OpenWorkspaceTab(kind, ref)
	nwLog.Info("启动时打开虚拟标签", "kind", kind, "ref", ref)
}
