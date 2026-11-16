package term

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
)

// 静默注入:终端里不能出现 source 回显,命令块照常记录,
// 用户自己的 .zshrc 照常执行,ZDOTDIR 在启动完成后还原(原来没设就是未设置)。
func TestZshSilentInject(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("没有 zsh")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ZDOTDIR", "")
	os.Unsetenv("ZDOTDIR")
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export MC_USER_RC_RAN=yes\nPS1='MCP> '\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tm, err := Start(Options{Shell: "/bin/zsh", Cwd: home, Cols: 80, Rows: 12, LoginShell: true, Integration: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tm.Close)
	if !tm.Integrated() {
		t.Fatal("zsh 应走静默注入")
	}
	waitFor(t, tm, "MCP>")
	run := func(cmd, wantOut string) {
		tm.SendText(cmd)
		tm.SendKey(uv.KeyEnter, "", 0)
		waitFor(t, tm, wantOut)
		time.Sleep(400 * time.Millisecond)
	}
	run("echo rc=$MC_USER_RC_RAN zd=${ZDOTDIR:-unset} hf=${HISTFILE##*/}:${HISTFILE%/*}", "rc=yes")
	scr := text(tm.Snapshot(100))
	if strings.Contains(scr, "source ") || strings.Contains(scr, "integration.zsh") {
		t.Errorf("画面里不该出现注入命令回显:\n%s", scr)
	}
	if !strings.Contains(scr, "zd=unset") {
		t.Errorf("ZDOTDIR 应还原为未设置:\n%s", scr)
	}
	if !strings.Contains(scr, "hf=.zsh_history") || strings.Contains(scr, "zdotdir") {
		t.Errorf("HISTFILE 应回到用户目录:\n%s", scr)
	}
	run("false", "")
	run("echo hi-silent", "hi-silent")
	bl := tm.CommandBlocks()
	if len(bl) < 2 || bl[len(bl)-2].ExitOK || !bl[len(bl)-1].ExitOK {
		t.Errorf("命令块记录不对:%+v\n画面:\n%s", bl, text(tm.Snapshot(100)))
	}
}

// 不是 zsh 就不能静默注入(调用方退回 IntegrationCmd)。
func TestInjectSkipsOtherShells(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, ok := zshInjectEnv("/bin/bash", nil); ok {
		t.Error("bash 不该走 zsh 注入")
	}
}
