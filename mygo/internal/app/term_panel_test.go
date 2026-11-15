package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// 真实终端面板:IDE 里打开底部终端后起 shell,键盘输入能到达 shell,输出画回界面。
func TestRealTerminalPanel(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	a, tt := newTestApp(t)
	a.SetProject(t.TempDir())
	a.SetMode(ModeIDE)
	a.ide.bottomOpen, a.ide.bottomTab = true, "term"
	t.Cleanup(a.CloseTerminals)

	pump := func(until func() bool, what string) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			tt.Frame()
			if until() {
				return
			}
			time.Sleep(30 * time.Millisecond)
		}
		t.Fatalf("等不到:%s;当前文字 %v", what, tt.Texts())
	}
	pump(func() bool { return a.ide.tpc().t != nil }, "终端启动")
	pump(func() bool { return containsText(tt.Texts(), "$") }, "出现提示符")

	tt.Type("printf 'ab%s' cd")
	tt.Key(0, ui.KeyEnter)
	pump(func() bool { return tt.HasText("abcd") }, "命令输出 abcd")

	// 没有文字事件的打字键(只有 KeyDown / KeyUp)按美式键盘补发
	tt.Key(0, ui.KeyE)
	tt.Key(0, ui.KeyC)
	tt.Key(0, ui.KeyH)
	tt.Key(0, ui.KeyO)
	tt.Key(0, ui.KeySpace)
	tt.Key(ui.Shift, ui.KeyX)
	tt.Key(ui.Shift, ui.KeyY)
	tt.Key(0, ui.KeyEnter)
	pump(func() bool { return tt.HasText("XY") }, "补发的按键输出 XY")

	// Ctrl-D 让 shell 退出,界面提示可重新开始
	tt.Key(ui.Ctrl, ui.KeyD)
	pump(func() bool {
		return tt.HasText("按回车重新开始") || containsText(tt.Texts(), "按回车重新开始")
	}, "退出提示")
	tt.Key(0, ui.KeyEnter)
	pump(func() bool { return a.ide.tpc().t != nil && !exitedNow(a) }, "回车后重新启动")
}

func exitedNow(a *App) bool { ex, _ := a.ide.tpc().t.Exited(); return ex }

func containsText(ts []string, sub string) bool {
	for _, s := range ts {
		if len(s) >= len(sub) {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// 字母键在 Key 枚举里是连续的(termKey 用差值换算字母)。
func TestKeyLettersContiguous(t *testing.T) {
	if ui.KeyZ-ui.KeyA != 25 || ui.Key9-ui.Key0 != 9 {
		t.Fatal("Key 枚举里字母 / 数字不连续,termKey 的换算要改")
	}
	if c, _, ok := termKey(ui.KeyC, ui.Ctrl); !ok || c != 'c' {
		t.Fatalf("Ctrl-C 编码 %q %v", c, ok)
	}
	if _, _, ok := termKey(ui.KeyC, 0); ok {
		t.Fatal("普通字母应等 InputText")
	}
}
