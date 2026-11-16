// shot 在没有窗口的情况下把界面渲染成 PNG,用来和设计稿截图逐像素对照。
//
//	go run ./cmd/shot -scheme a -theme paper -mode agent -screen work -o /tmp/go-work-a-paper.png
//
// 与 scripts/proto-shot.sh 出的设计稿截图同尺寸(1280×800,2 倍),可直接并排比较。
package main

import (
	"flag"
	"fmt"
	_ "github.com/egoist/mygo/locales/zh-Hans" // 控件自带文案的简体中文译文
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/app"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
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
	noMotion := flag.Bool("nomotion", false, "关闭动效直接取终态(截图里没有窗口,frameSoon 是空操作,按真实时钟推进的入场动画如用量柱状图会停在第一帧)")
	palette := flag.String("palette", "", "打开命令面板并预填查询(用 _ 表示空查询)")
	tab := flag.String("tab", "", "设置页 / 自定义页分类(配合 -screen settings / customize)")
	squery := flag.String("squery", "", "设置页搜索词")
	state := flag.String("state", "", "会话:welcome/perm/ask/plan/queue/steer/queue-paused/ask2/askdone/latest/latest-fresh;scn:<场景 id> 回放任一演示场景")
	askCur := flag.Int("ask-cur", 1, "ask2 状态:当前题号(1 起)")
	quick := flag.String("quick", "", "快速选择浮层:history / branch,可带预填查询,如 branch:feat")
	scm := flag.String("scm", "", "IDE 左栏源代码管理:[提交说明][|展开的提交哈希],如 |b7d02e4")
	srch := flag.String("search", "", "IDE 左栏搜索:查询[|替换文本],如 match_path|find_route")
	ed := flag.String("ed", "", "编辑器动作:cursor:17:20,select:11:1-13:20,find:查询,scroll:行,peek,rename,acts,inline,refs,def,hover:行:列")
	wsFlag := flag.Bool("ws", false, "进入演示工作区「全栈工作区」(多根文件树 / 多仓库源代码管理)")
	dlg := flag.String("dlg", "", "打开项目对话框:add / ws / wsedit / confirm;add:路径 = 预填并提交")
	repoIx := flag.Int("repo", -1, "多根时把第 N 个仓库设为源代码管理的活动仓库")
	stress := flag.Bool("stress", false, "打开演示的极端数据(300 文件 bulk / 冲突 / 5 条储藏 / 220+ 提交),默认关闭以便与设计稿对照")
	detail := flag.Int("detail", -1, "工具调用详细程度 0 精简 / 1 标准 / 2 详细(默认沿用偏好)")
	flag.Parse()
	demo.SetStress(*stress)

	home := "/tmp/minicode-shot-home"
	os.Setenv("MINI_HOME", home)
	diag.Init(diag.Options{Version: "shot", Mode: "shot", Level: "warn"})

	m := app.ModeAgent
	if *mode == "ide" {
		m = app.ModeIDE
	}
	a := app.New(app.Options{Palette: *pal, Scheme: *scheme, Mode: m, Screen: app.Screen(*screen), FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.SetRepo(demo.Repo())
	demo.FixedClock = true
	a.SetClock(func() time.Time { return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) }) // 与设计稿的「今天」一致
	a.SeedDemoExtras()
	if *detail >= 0 {
		a.SetDetailLevel(*detail)
	}
	if *noMotion {
		kit.SetMotion(false) // 要放在应用创建之后,创建时会按偏好把动效重新打开
	}
	if *wsFlag {
		a.ActivateWorkspace("ws-fullstack")
		if *repoIx >= 0 {
			a.SetActiveRepoIndex(*repoIx)
		}
	}
	skipTop := false
	switch *state {
	case "welcome":
		a.NewSession() // 欢迎页只在没有当前会话 id 时显示,SetProjects 已选中 fixbug,要显式开新会话
	case "perm":
		demo.Replay(a.State(), demo.PendingPermission())
	case "ask":
		demo.Replay(a.State(), demo.PendingQuestion())
	case "plan":
		demo.Replay(a.State(), demo.PendingPlan())
	case "steer":
		demo.Replay(a.State(), demo.RunningFixBug())
		a.SeedSteers("先只看 normalize 行为", "README 里补一句说明")
		a.QueuePrompts("顺便把 README 里的路由示例也更新一下", "再跑一遍完整测试")
	case "queue-paused":
		demo.Replay(a.State(), demo.RunningFixBug())
		a.SeedQueuePaused("顺便把 README 里的路由示例也更新一下", "再跑一遍完整测试")
	case "ask2":
		demo.Replay(a.State(), demo.PendingQuestion3())
		cur := *askCur - 1
		if cur < 0 {
			cur = 0
		}
		a.SetAskCur("q1", cur, *askCur > 1)
	case "askdone":
		demo.Replay(a.State(), demo.AskDoneReplay())
	case "latest", "latest-fresh":
		demo.Replay(a.State(), demo.FixBug(false))
		demo.ReplayTasks(a.State())
		a.PrepFlowScrollAway(*state == "latest-fresh")
		skipTop = true
	default:
		if id, ok := strings.CutPrefix(*state, "scn:"); ok { // scn:<场景 id> 静态回放任一演示场景(如 scn:mermaid-samples)
			found := false
			for _, sc := range demo.Scenarios() {
				if sc.ID == id {
					demo.Replay(a.State(), sc.Build())
					found = true
					break
				}
			}
			if !found {
				fmt.Fprintln(os.Stderr, "没有这个演示场景:", id)
				os.Exit(2)
			}
			break
		}
		demo.Replay(a.State(), demo.FixBug(false)) // 静态回放:不等待、同步归约
		demo.ReplayTasks(a.State())
		if *state == "queue" {
			a.QueuePrompts("顺便把 README 里的路由示例也更新一下", "再跑一遍完整测试")
		}
		if *state == "queue-imgs" {
			a.QueuePrompts(demo.QueuePromptSamples()...)
		}
	}

	tt := ui.NewTester(a.View, *w, *h)
	tt.SetLocale("zh-Hans")
	tt.SetScale(float32(*scale))
	tt.SetDark(strings.Contains("ink graphite aurora", *pal))
	a.SetPopup(*popup)
	if *screen == "settings" {
		a.SetSettings(*tab, *squery)
	}
	if *screen == "customize" && *tab != "" {
		a.SetCustomize(*tab)
	}
	if *top && !skipTop {
		a.ScrollFlowTop()
		tt.Frame()
	}
	if skipTop {
		tt.Frame()
		a.ApplyFlowScrollAway()
		if *state == "latest-fresh" {
			a.AppendFlowTailForFresh()
		}
		tt.Frame()
	}
	if *srch != "" {
		q, r, hasR := strings.Cut(*srch, "|")
		a.OpenSearch()
		a.SetSearch(q, r, hasR)
		tt.Frame()
	}
	if *scm != "" {
		msg, rev, _ := strings.Cut(*scm, "|")
		a.OpenSCM()
		a.SetSCM(msg, rev)
		tt.Frame()
		tt.Frame()
	}
	if *dlg != "" {
		kind, p, _ := strings.Cut(*dlg, ":")
		a.OpenProjDialog(kind)
		if p != "" {
			a.SetAddProjectPath(p)
		}
		tt.Frame()
		time.Sleep(300 * time.Millisecond) // 弹层入场动画按真实时钟计时
		tt.Frame()
	}
	if *quick != "" {
		kind, q, _ := strings.Cut(*quick, ":")
		a.OpenQuick(kind, q)
		tt.Frame()
	}
	if *palette != "" {
		q := *palette
		if q == "_" {
			q = ""
		}
		a.OpenPalette(q)
		tt.Frame()
	}
	tt.Advance(800 * time.Millisecond)
	if *ed != "" {
		a.EditorDemo(*ed)
		tt.Frame()
		tt.Advance(500 * time.Millisecond)
	}
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
