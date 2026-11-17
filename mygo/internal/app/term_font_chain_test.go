package app

import (
	"strings"
	"testing"
)

func TestParseFontFamiliesQuoted(t *testing.T) {
	got := parseFontFamilies(`"JetBrains Mono", Menlo, monospace`)
	if len(got) != 3 || got[0] != "JetBrains Mono" || got[2] != "monospace" {
		t.Fatalf("解析字体列表: %v", got)
	}
}

func TestBuildTermFontChainAppendsFallback(t *testing.T) {
	chain := buildTermFontChain("Fira Code", true)
	if !strings.Contains(chain, "Fira Code") || !strings.Contains(chain, "monospace") || !strings.Contains(chain, "PingFang SC") {
		t.Fatalf("应追加回退链: %q", chain)
	}
}

func TestBuildTermFontChainSkipsMissingNerd(t *testing.T) {
	setFakeFonts(t, "Menlo.ttc")
	chain := buildTermFontChain("JetBrainsMono Nerd Font", true)
	if !strings.Contains(chain, "monospace") || !strings.Contains(chain, "Menlo") {
		t.Fatalf("没装的 Nerd 应变回退,得 %q", chain)
	}
	if strings.Contains(chain, "JetBrainsMono Nerd Font") {
		t.Fatalf("未装的不应留在链里: %q", chain)
	}
}
