package app

import (
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/kit"
)

func TestTermTabTitlePathMiddle(t *testing.T) {
	raw := "~/Users/me/Projects/router-demo/src"
	got := termTabTitle(raw, 120)
	if !strings.Contains(got, "…") {
		t.Fatalf("长路径应中间省略: %q", got)
	}
	if !strings.HasPrefix(got, "~") || !strings.HasSuffix(got, "src") {
		t.Fatalf("应保留首尾: %q", got)
	}
}

func TestTermNormalizeCommandBasename(t *testing.T) {
	raw := "/opt/homebrew/bin/cargo test --all"
	got := termNormalizeTitle(raw)
	if !strings.HasPrefix(got, "cargo ") {
		t.Fatalf("应取 basename: %q", got)
	}
}

func TestTermTabWidthsCompress(t *testing.T) {
	texts := []string{"终端一", "终端二很长很长", "三"}
	widths := termTabWidths(texts, 200)
	var sum float32
	for _, w := range widths {
		if w < termTabMinW {
			t.Fatalf("宽度不应低于 min: %v", w)
		}
		sum += w
	}
	if sum > 200+1 {
		t.Fatalf("总宽应不超过可用区: %v > 200", sum)
	}
}

func TestTermTabRawTitlePriority(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a"})
	a.proj = "/tmp/demo"
	a.ide.tps = []*termPanel{{label: "自定义", demoTitle: "ignored", cwd: "/x"}}
	if got := a.termTabRawTitle(0); got != "自定义" {
		t.Fatalf("重命名优先: %q", got)
	}
	a.ide.tps[0].label = ""
	a.ide.tps[0].demoTitle = "OSC 标题"
	if got := a.termTabRawTitle(0); got != "OSC 标题" {
		t.Fatalf("demo/OSC 替身: %q", got)
	}
	a.ide.tps[0].demoTitle = ""
	if got := a.termTabRawTitle(0); got != "demo" {
		t.Fatalf("cwd basename: %q", got)
	}
}

func TestEllipsizeTailWidth(t *testing.T) {
	s := strings.Repeat("a", 40)
	got := ellipsizeTail(s, 80, termTitleFontSize)
	if kit.TextW(got, termTitleFontSize) > 80+0.5 {
		t.Fatalf("截断后仍超宽: %q", got)
	}
}

func TestTermTabsVerticalAuto(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a"})
	a.proj = "/tmp"
	a.ide.tps = []*termPanel{{}, {}}
	a.prefs.Set(pTermTabs, termTabAuto)
	if a.termTabsVertical(700) {
		// ok
	} else {
		t.Fatal("宽面板应竖向")
	}
	if a.termTabsVertical(500) {
		t.Fatal("窄面板应横向")
	}
	a.prefs.Set(pTermTabs, termTabHorz)
	if a.termTabsVertical(900) {
		t.Fatal("横向偏好应强制横向")
	}
}

func TestCloseOtherTermTabs(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a"})
	a.proj = "/tmp"
	a.ide.tps = []*termPanel{{}, {}, {}}
	a.closeOtherTermTabs(1)
	if len(a.ide.tps) != 1 || a.ide.tpSel != 0 {
		t.Fatalf("应只剩一个且选中索引回落: n=%d sel=%d", len(a.ide.tps), a.ide.tpSel)
	}
}
