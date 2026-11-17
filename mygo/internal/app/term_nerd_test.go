package app

import (
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/term"
)

// 用假的字体清单驱动 Nerd Font 判断(不碰真实字体目录)。
func setFakeFonts(t *testing.T, raw ...string) {
	t.Helper()
	oldF, oldR, oldD := fontScan.files, fontScan.raw, fontScan.done.Load()
	fontScan.files, fontScan.raw = nil, nil
	for _, r := range raw {
		fontScan.raw = append(fontScan.raw, strings.ToLower(r))
		fontScan.files = append(fontScan.files, normFont(r))
	}
	fontScan.done.Store(true)
	t.Cleanup(func() { fontScan.files, fontScan.raw = oldF, oldR; fontScan.done.Store(oldD) })
}

func puaPanel() *termPanel {
	return &termPanel{scr: &term.Screen{Lines: []term.Line{{{Text: "\uf15b main.go"}}}}}
}

// 画面有 Nerd 图标:没装 Nerd Font → 提示安装;装了但没选 → 提示去选;选了已装的 → 不提示。
func TestNerdWarning(t *testing.T) {
	a, _ := newTestApp(t)
	tp := puaPanel()

	setFakeFonts(t, "Menlo.ttc")
	if w := a.nerdWarning(tp); !strings.Contains(w, "brew install") {
		t.Fatalf("没装 Nerd Font 应提示安装,得到 %q", w)
	}

	setFakeFonts(t, "JetBrainsMonoNerdFont-Regular.ttf")
	if w := a.nerdWarning(tp); !strings.Contains(w, "已检测到") {
		t.Fatalf("装了但没选,应提示去选,得到 %q", w)
	}

	a.prefs.Set("term.font", "JetBrainsMono Nerd Font")
	if w := a.nerdWarning(tp); w != "" {
		t.Fatalf("选了已装的 Nerd Font 不该提示,得到 %q", w)
	}
	if got := a.termFontFamily(); !strings.Contains(got, "JetBrainsMono Nerd Font") {
		t.Fatalf("已装的 Nerd Font 应在字体链里,得到 %q", got)
	}

	// 选了但没装:回退系统等宽(开关默认开),并提示
	setFakeFonts(t, "Menlo.ttc")
	if got := a.termFontFamily(); !strings.Contains(got, "monospace") || strings.Contains(got, "JetBrainsMono Nerd Font") {
		t.Fatalf("没装的 Nerd Font 应换成系统等宽链,得到 %q", got)
	}
	if w := a.nerdWarning(tp); w == "" {
		t.Fatal("选了没装的 Nerd Font,图标缺字,应提示")
	}

	// 没有私有区字符就不打扰
	plain := &termPanel{scr: &term.Screen{Lines: []term.Line{{{Text: "hello"}}}}}
	if w := a.nerdWarning(plain); w != "" {
		t.Fatalf("没有图标字符不该提示,得到 %q", w)
	}
}
