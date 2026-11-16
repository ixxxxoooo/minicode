package app

import (
	"bytes"
	"image/color"
	"image/png"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/prefs"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

// 样式下标越界时退回默认样式,仍要给出可用的 Dock 图标。
func TestDockIconDataFallback(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true, Prefs: prefs.Memory()})
	a.prefs.Set(pAppStyle, 99)
	if len(a.dockIconData(false)) == 0 {
		t.Fatal("越界样式应退回默认图标")
	}
}

// 六种样式 × 浅 / 深 / 透明 / 着色 的内嵌素材必须齐全,否则换样式后 Dock 会回退到默认图。
func TestEmbeddedAppIconsComplete(t *testing.T) {
	for si := range appStyles {
		for _, suf := range []string{"", "-dark", "-clear", "-tint"} {
			if _, ok := embeddedIcon("app-style" + itoa(si) + suf + ".png"); !ok {
				t.Errorf("缺少内嵌素材 app-style%d%s.png", si, suf)
			}
		}
	}
}

// 五种托盘样式 × 全部状态(含运行动画三帧)的内嵌素材必须齐全。
func TestEmbeddedTrayIconsComplete(t *testing.T) {
	states := []string{"idle", "running", "running-0", "running-1", "running-2", "waiting", "unread", "error", "update", "paused", "demo"}
	for si := range trayStyles {
		pre := "tray"
		if si > 0 {
			pre = "tray-s" + itoa(si)
		}
		for _, st := range states {
			if _, ok := embeddedIcon(pre + "-" + st + ".png"); !ok {
				t.Errorf("缺少内嵌素材 %s-%s.png", pre, st)
			}
		}
	}
}

// 彩色徽标:徽标处染成状态色,其余主体按菜单栏深浅染成白 / 黑;没有徽标的状态不染色。
func TestTintTrayPNG(t *testing.T) {
	data, _ := embeddedIcon("tray-error.png")
	out, ok := tintTrayPNG(data, "error", false)
	if !ok {
		t.Fatal("error 状态应有彩色徽标")
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	// 徽标外沿(靠圆心偏上,避开中间的白色 ×):红色
	nrgba := func(x, y int) color.NRGBA { return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA) }
	if c := nrgba(28, 4); c.R < 0xe0 || c.G > 0x80 || c.A < 0xe0 {
		t.Fatalf("徽标应为红色,实际 %+v", c)
	}
	// 主体(左上织纹)在浅色菜单栏上是黑色
	var sawBlack bool
	for y := 4; y < 30 && !sawBlack; y++ {
		for x := 4; x < 20; x++ {
			if c := nrgba(x, y); c.A > 0xf0 {
				if c.R != 0 || c.G != 0 || c.B != 0 {
					t.Fatalf("浅色菜单栏上主体应为黑色,实际 %+v", c)
				}
				sawBlack = true
				break
			}
		}
	}
	if !sawBlack {
		t.Fatal("没有找到主体像素")
	}
	dark, _ := tintTrayPNG(data, "error", true)
	img, _ = png.Decode(bytes.NewReader(dark))
	var sawWhite bool
	for y := 4; y < 30 && !sawWhite; y++ {
		for x := 4; x < 20; x++ {
			if c := nrgba(x, y); c.A > 0xf0 && c.R == 0xff && c.G == 0xff {
				sawWhite = true
				break
			}
		}
	}
	if !sawWhite {
		t.Fatal("深色菜单栏上主体应为白色")
	}
	if _, ok := tintTrayPNG(data, "idle", false); ok {
		t.Fatal("空闲状态没有徽标,不应染色")
	}
}

// 托盘素材名随样式与状态变化,运行中三帧轮流。
func TestTrayAssetName(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true, Prefs: prefs.Memory()})
	if n := a.trayAssetName(traySum{mode: "waiting"}, 0); n != "tray-waiting.png" {
		t.Fatalf("默认样式: %s", n)
	}
	a.prefs.Set(pTrayStyle, 2)
	if n := a.trayAssetName(traySum{mode: "error"}, 0); n != "tray-s2-error.png" {
		t.Fatalf("样式 2: %s", n)
	}
	if n := a.trayAssetName(traySum{mode: "running"}, 4); n != "tray-s2-running-1.png" {
		t.Fatalf("运行帧: %s", n)
	}
	a.prefs.Set(pTrayDyn, false)
	if n := a.trayAssetName(traySum{mode: "error"}, 0); n != "tray-s2-idle.png" {
		t.Fatalf("关闭随状态变化应固定空闲图: %s", n)
	}
}

func endsWithAny(p string, suffixes ...string) bool {
	for _, s := range suffixes {
		if len(p) >= len(s) && p[len(p)-len(s):] == s {
			return true
		}
	}
	return false
}

func TestTrayA11yLabel(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true, Prefs: prefs.Memory()})
	l := a.trayA11yLabel(traySum{pending: 1, running: 2})
	if l != "MiniCode,1 个待确认,2 个运行中" {
		t.Fatalf("标签不对: %q", l)
	}
}

// 托盘状态优先级:待确认 > 出错 > 已完成 > 运行中 > 暂停 > 空闲;回到前台后出错 / 已完成清除(测试里 win == nil 视为前台)。
func TestTrayModePriority(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true, Prefs: prefs.Memory()})
	a.reg = projects.Open(filepath.Join(t.TempDir(), "ws.json")) // 持久存储 = 非演示
	if m := a.brandSummarize().mode; m != "idle" {
		t.Fatalf("初始应为 idle,得到 %s", m)
	}
	a.prefs.Set(pNotifyPause, true)
	if m := a.brandSummarize().mode; m != "paused" {
		t.Fatalf("暂停通知应为 paused,得到 %s", m)
	}
	// 前台时 error / unread 视为已看过,会被清掉
	a.brand.trayErr, a.brand.trayUnread = true, true
	if m := a.brandSummarize().mode; m != "paused" || a.brand.trayErr || a.brand.trayUnread {
		t.Fatalf("前台应清除出错 / 已完成标记,mode=%s err=%v unread=%v", m, a.brand.trayErr, a.brand.trayUnread)
	}
}

// 菜单里的会话标记:运行 / 待确认 / 未读各不相同,看过的会话用透明占位(对齐文字)。
func TestTrayMarkKind(t *testing.T) {
	seen := map[string]bool{}
	for _, k := range []string{"run", "wait", "unread", "idle"} {
		seen[trayMarkKind(k)] = true
	}
	if len(seen) != 4 || trayMarkKind("idle") != "blank" || trayMarkKind("ok") != "blank" {
		t.Fatalf("四种状态应各有标记种类、其余为 blank,得到 %v", seen)
	}
}

// 托盘菜单:按项目分组(灰色组名)、含「重启 MiniCode」、会话行带标记、标题截短不撑宽。
func TestTrayMenuGroupedWithRestart(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true, Prefs: prefs.Memory()})
	a.curSess = "s1"
	a.projects = []core.ProjectInfo{{ID: "p", Name: "MiniCode", Sessions: []core.SessionInfo{{ID: "s1", Title: "这是一个特别特别特别特别特别特别长的会话标题"}}}}
	var labels []string
	var restart, groupHdr bool
	for _, it := range a.trayMenuFull().Items() {
		labels = append(labels, it.Label)
		if it.Label == "重启 MiniCode" {
			restart = true
		}
		if it.Disabled && it.Label == a.projectName() {
			groupHdr = true
		}
		if n := len([]rune(it.Label)); n > 30 {
			t.Fatalf("菜单项过长会撑宽菜单:%q(%d)", it.Label, n)
		}
	}
	if !restart || !groupHdr {
		t.Fatalf("应有重启项与项目组名,restart=%v group=%v labels=%v", restart, groupHdr, labels)
	}
}

// 品牌小标跟随所选样式;织纹在小尺寸自动换「经纬」;空状态小标取托盘空闲图。
func TestBrandLogoSource(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true, Prefs: prefs.Memory()})
	if a.brandLogoSource(64) == nil || a.brandLogoSource(20) == nil {
		t.Fatal("默认样式应能取到品牌小标")
	}
	a.prefs.Set(pAppStyle, 3)
	if a.brandLogoSource(64) == nil {
		t.Fatal("选了其它样式也应取到品牌小标")
	}
	if brandInkImage("tray-idle.png", ui.Color{R: 0, G: 0, B: 0, A: 64}) == nil {
		t.Fatal("空状态小标应能由托盘图染色得到")
	}
}
