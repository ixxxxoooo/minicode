// shot 在没有窗口的情况下把界面渲染成 PNG,用来和设计稿截图逐像素对照。
//
//	go run ./cmd/shot -scheme a -theme paper -mode agent -screen work -o /tmp/go-work-a-paper.png
//
// 与 scripts/proto-shot.sh 出的设计稿截图同尺寸(1280×800,2 倍),可直接并排比较。
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/app"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

func main() {
	scheme := flag.String("scheme", "a", "布局方案 a / b / c")
	pal := flag.String("theme", "paper", "主题 id")
	mode := flag.String("mode", "agent", "agent / ide")
	screen := flag.String("screen", "work", "屏幕 id")
	out := flag.String("o", "/tmp/go-shot.png", "输出文件")
	w := flag.Int("w", 1280, "宽(DIP)")
	h := flag.Int("h", 800, "高(DIP)")
	scale := flag.Float64("scale", 2, "像素倍率")
	top := flag.Bool("top", true, "会话流停在顶部(设计稿的截图就是这样)")
	popup := flag.String("popup", "", "打开某个弹层:cfg / sbcfg / ctx / upd / acct")
	clicks := flag.String("click", "", "截图前依次点击,形如 IDE;100,200(文字或 DIP 坐标);每次点击后推进 -step 毫秒")
	step := flag.Int("step", 600, "每次点击后推进的毫秒数")
	flag.Parse()

	home := "/tmp/minicode-shot-home"
	os.Setenv("MINI_HOME", home)
	diag.Init(diag.Options{Version: "shot", Mode: "shot", Level: "warn"})

	m := app.ModeAgent
	if *mode == "ide" {
		m = app.ModeIDE
	}
	a := app.New(app.Options{Palette: *pal, Scheme: *scheme, Mode: m, Screen: app.Screen(*screen), FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.SeedDemoExtras()
	demo.Replay(a.State(), demo.FixBug(false)) // 静态回放:不等待、同步归约
	demo.ReplayTasks(a.State())

	tt := ui.NewTester(a.View, *w, *h)
	tt.SetScale(float32(*scale))
	tt.SetDark(strings.Contains("ink graphite aurora", *pal))
	a.SetPopup(*popup)
	if *top {
		a.ScrollFlowTop()
		tt.Frame()
	}
	tt.Advance(800 * time.Millisecond)
	for _, cl := range strings.Split(*clicks, ";") {
		if cl == "" {
			continue
		}
		var x, y float32
		if _, err := fmt.Sscanf(cl, "%f,%f", &x, &y); err == nil {
			tt.ClickAt(x, y)
		} else if err := tt.Click(cl); err != nil { // 非坐标:按文字 / 标签点击
			fmt.Fprintln(os.Stderr, "点击失败:", err)
			os.Exit(2)
		}
		tt.Advance(time.Duration(*step) * time.Millisecond)
	}
	img := tt.Image()
	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("写入", *out, img.Bounds().Dx(), "x", img.Bounds().Dy())
}
