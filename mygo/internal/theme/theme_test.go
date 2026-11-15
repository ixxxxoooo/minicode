package theme

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func near(a, b uint8) bool {
	d := int(a) - int(b)
	return d >= -1 && d <= 1
}

func TestSixThemesThreeSchemes(t *testing.T) {
	if len(Palettes) != 6 || len(Schemes) != 3 {
		t.Fatal("应有 6 套主题、3 套方案")
	}
	var dark int
	for _, p := range Palettes {
		if p.Dark {
			dark++
		}
		tk := Resolve(p, Schemes[0], "")
		if tk.Chrome.A != 255 || tk.Tx1 != ui.Hex(p.Ink) {
			t.Fatalf("%s 令牌不对", p.ID)
		}
	}
	if dark != 3 {
		t.Fatalf("深色主题应有 3 套,实际 %d", dark)
	}
}

// 手算 paper 主题的 --tx-2 = color-mix(ink 68%, surface):
// ink #1F1D1A(31,29,26),surface #FAF9F6(250,249,246) → 31*.68+250*.32 = 101.1
func TestColorMixMatchesCSS(t *testing.T) {
	tk := Resolve(Find("paper"), FindScheme("a"), "")
	if !near(tk.Tx2.R, 101) || !near(tk.Tx2.G, 99) || !near(tk.Tx2.B, 96) {
		t.Fatalf("tx-2 = %+v", tk.Tx2)
	}
	// --bd-2 = ink 12% 透明:颜色是 ink,不透明度 .12
	if tk.Bd2.R != 31 || !near(tk.Bd2.A, 31) { // .12*255 = 30.6
		t.Fatalf("bd-2 = %+v", tk.Bd2)
	}
}

func TestSchemeRadii(t *testing.T) {
	a := Resolve(Find("paper"), FindScheme("a"), "")
	b := Resolve(Find("paper"), FindScheme("b"), "")
	if a.RSm() != 4 || b.RSm() != 7 { // 5×.8、5×1.4
		t.Fatalf("圆角倍率不对: %v %v", a.RSm(), b.RSm())
	}
}

func TestAccentOverride(t *testing.T) {
	tk := Resolve(Find("ink"), FindScheme("c"), "#112233")
	if tk.Ac != ui.Hex("#112233") || tk.ToolAgent != tk.Ac {
		t.Fatal("强调色覆盖失败")
	}
}

func TestUIThemeFollowsScheme(t *testing.T) {
	if !Resolve(Find("aurora"), FindScheme("a"), "").UITheme().Dark {
		t.Fatal("极光应是深色 ui 主题")
	}
}
