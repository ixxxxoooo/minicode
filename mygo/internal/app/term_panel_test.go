package app

import (
	"strings"
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

// 真实终端里:拖选输出 → ⌘C 复制;⌘F 打开查找并命中;⌘+ 放大字号。
func TestRealTerminalSelectFind(t *testing.T) {
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
	pump(func() bool { return containsText(tt.Texts(), "$") }, "提示符")
	tt.Type("printf 'zq%s\\n' xwvu")
	tt.Key(0, ui.KeyEnter)
	pump(func() bool { return tt.HasText("zqxwvu") }, "输出 zqxwvu")

	// 在真实屏幕里找到输出行,双击选词(指针坐标换算由 TestTermSelectionCopyAndWord 与手工验证覆盖)
	tp := a.ide.tpc()
	row := -1
	for i, ln := range tp.scr.Lines {
		if strings.TrimRight(strings.Join(lineCells(ln), ""), " ") == "zqxwvu" {
			row = i
		}
	}
	if row < 0 {
		t.Fatalf("屏幕里没有输出行 zqxwvu")
	}
	tp.wordAt(cellPos{row, 2})
	if got := tp.selectedText(); got != "zqxwvu" {
		t.Fatalf("双击选词得到 %q", got)
	}

	// ⌘F:焦点在终端时打开查找,输入后命中 ≥1
	a.termHasFocus = true
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Frame()
	if !a.ide.tpc().find.open {
		t.Fatal("⌘F 应打开终端查找")
	}
	// 选中了文字时,它就是初始查询
	if a.ide.tpc().find.query != "zqxwvu" {
		t.Fatalf("初始查询应是选中的文字,实际 %q", a.ide.tpc().find.query)
	}
	pump(func() bool { return len(a.ide.tpc().find.hits) >= 1 }, "查找命中")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.ide.tpc().find.open {
		t.Fatal("Esc 应关闭查找")
	}

	before := a.prefs.Int(pTermFont, 12)
	a.termHasFocus = true
	tt.Key(ui.Cmd, ui.KeyEqual)
	tt.Frame()
	if a.prefs.Int(pTermFont, 12) != before+1 {
		t.Fatalf("⌘+ 应把字号 %d 加到 %d,实际 %d", before, before+1, a.prefs.Int(pTermFont, 12))
	}
}
