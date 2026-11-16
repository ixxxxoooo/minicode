package app

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/prefs"
)

func settingsApp(t *testing.T, tab string) (*App, *ui.Tester, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true, Prefs: prefs.Open(path)})
	a.SetProjects(demo.Projects(), "fixbug")
	demo.Replay(a.State(), demo.FixBug(false))
	a.SetSettings(tab, "")
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	return a, tt, path
}

// 点主题卡立即换主题并落盘;「恢复默认」把它还原。
func TestSettingsThemeCardAppliesAndPersists(t *testing.T) {
	a, tt, path := settingsApp(t, "appearance")
	if err := tt.Click("雾霭"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.pal.ID != "mist" {
		t.Fatalf("应换成雾霭,得到 %s", a.pal.ID)
	}
	if got := prefs.Open(path).Str(kTheme, ""); got != "mist" {
		t.Fatalf("主题应落盘,读到 %q", got)
	}
}

// 密度与圆角实时改变令牌。
func TestSettingsDensityRadius(t *testing.T) {
	a, tt, _ := settingsApp(t, "appearance")
	base := a.tk.Scheme.RowH
	// 「密度与形状」在应用图标 / 菜单栏图标两组之后(设计稿顺序),要先把设置正文滚下去
	scrollSettingsTo := func(y float32) {
		a.set.mainSt.Y = y
		tt.Frame()
		tt.Frame()
	}
	scrollSettingsTo(math.MaxFloat32)
	scrollSettingsTo(a.set.mainSt.Y - 400) // 滚到底后上移一屏,让「密度与形状」落进可视区
	if err := tt.Click("紧凑"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.density != 0 || a.tk.Scheme.RowH != base-2 {
		t.Fatalf("紧凑应让行高 -2:density=%d RowH=%v base=%v", a.density, a.tk.Scheme.RowH, base)
	}
	rk := a.tk.Scheme.RK
	if err := tt.Click("利落"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.tk.Scheme.RK >= rk {
		t.Fatalf("利落应缩小圆角倍率:%v → %v", rk, a.tk.Scheme.RK)
	}
}

// 搜索命中跨分类;点结果跳到对应分类并清空搜索。
func TestSettingsSearchJump(t *testing.T) {
	a, tt, _ := settingsApp(t, "appearance")
	hs := a.settingsHits("代理")
	if len(hs) == 0 {
		t.Fatal("「代理」应有命中")
	}
	found := false
	for _, h := range hs {
		found = found || h.tab == "runtime"
	}
	if !found {
		t.Fatal("「代理」应命中运行环境")
	}
	a.set.query = "代理"
	tt.Frame()
	if err := tt.Click("代理模式"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.setTab != "runtime" || a.set.query != "" {
		t.Fatalf("应跳到运行环境并清空搜索:tab=%s q=%q", a.setTab, a.set.query)
	}
}

// 需重启的项被改动后出现「需重启」条,值等于默认时消失。
func TestSettingsRestartBar(t *testing.T) {
	a, tt, _ := settingsApp(t, "permissions")
	r := setRow{key: "perm.sandbox", def: false, restart: true, kind: kTog}
	a.setPref(r, true)
	tt.Frame()
	if len(a.set.restart) != 1 {
		t.Fatal("应记下一项需重启")
	}
	a.setPref(r, false)
	if len(a.set.restart) != 0 {
		t.Fatal("改回默认后不再需要重启")
	}
}

// 每个分类都能渲染(不 panic),且至少有一个分组。
func TestSettingsAllTabsRender(t *testing.T) {
	for _, tab := range settingsTabs {
		a, tt, _ := settingsApp(t, tab.id)
		tt.Frame()
		if a.setTab != tab.id || len(tab.groups) == 0 {
			t.Fatalf("分类 %s 异常", tab.id)
		}
	}
	if len(settingsTabs) != 14 {
		t.Fatalf("应有 14 个分类,实际 %d", len(settingsTabs))
	}
}

// 分组既有行又有自定义块时两者都要画出来(原先有块就丢掉了行)。
func TestSettingsGroupRowsAndBodyBothRender(t *testing.T) {
	_, tt, _ := settingsApp(t, "privacy")
	for _, want := range []string{"日志级别", "诊断包", "崩溃与卡顿"} {
		if !tt.HasText(want) {
			t.Fatalf("隐私页缺少 %q", want)
		}
	}
}

// 改过又改回默认值的行不再算「已修改」(齿轮弹层等直接写键的入口会留下等于默认值的键)。
func TestRowChangedComparesDefault(t *testing.T) {
	a, _ := newTestApp(t)
	r := setRow{key: pTermFont, def: 12}
	if a.rowChanged(r) {
		t.Fatal("没写键不该算已修改")
	}
	a.prefs.Set(pTermFont, 14)
	if !a.rowChanged(r) {
		t.Fatal("值与默认不同应算已修改")
	}
	a.prefs.Set(pTermFont, 12)
	if a.rowChanged(r) {
		t.Fatal("值等于默认不该算已修改")
	}
}
