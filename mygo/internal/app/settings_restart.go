package app

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// restartForSettings 自启动同一可执行文件并退出(需重启条与运行模式切换用,会写入运行模式记忆)。
func (a *App) restartForSettings(reason string) { a.restartApp(reason, true) }

// restartApp 自启动同一可执行文件并退出;persistMode 为真时先把设置里的「真实 / 演示」写入 dev-mode。
func (a *App) restartApp(reason string, persistMode bool) {
	exe, err := os.Executable()
	if err != nil {
		setLog.Error("重启失败:无法解析可执行文件", "err", err)
		return
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		setLog.Warn("解析可执行文件符号链接失败", "err", err)
	}
	args := append([]string(nil), os.Args[1:]...)
	cmd := exec.Command(exe, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir, _ = os.Getwd()
	// 开发版 .app:工作目录常是包内,切回用户目录避免相对路径错乱
	if wd, err := os.UserHomeDir(); err == nil {
		cmd.Dir = wd
	}
	if persistMode {
		a.persistRunModeChoice()
	}
	setLog.Info("触发重启", "reason", reason, "exe", exe, "args", len(args))
	if err := cmd.Start(); err != nil {
		setLog.Error("启动新进程失败", "err", err)
		return
	}
	diag.CleanExit("设置重启")
	if a.win != nil {
		a.win.Close()
	}
	os.Exit(0)
}

var setLog = diag.L("settings")

// persistRunModeChoice 在重启前把「真实/演示」写入 dev-mode(仅开发版生效)。
func (a *App) persistRunModeChoice() {
	mode := "real"
	if a.prefs.Int("adv.runmode", 0) == 1 {
		mode = "demo"
	}
	path := filepath.Join(diag.Home(), "dev-mode")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(mode), 0o644)
}
