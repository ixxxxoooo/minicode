package app

import (
	"github.com/ixxxxoooo/minicode/internal/kit"
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 整页功能(用量 / 自动化 / 画布 / 浏览器 / 自定义)的公共骨架:
// 它们替换工作台的「编辑区 + 对话栏 + 检查器」,导航栏与标题栏不变,所以三套方案下都成立。

const drawerW = 300 // 右侧抽屉宽度(设计稿 .drawer);自定义页用更宽的 cuDrawerW / cuDrawerWideW

// page 画当前整页功能;返回 false 表示不是整页。
func (a *App) page(c *ui.Context) bool {
	switch a.screen {
	case ScreenUsage:
		a.usagePage(c)
	case ScreenAutomation:
		a.autoPage(c)
	case ScreenCanvas:
		a.canvasPage(c)
	case ScreenBrowser:
		a.browserPage(c)
	case ScreenCustomize:
		a.customizePage(c)
	default:
		return false
	}
	return true
}

// pageHead 是页头:标题 + 说明在左,右侧操作。
func (a *App) pageHead(c *ui.Context, title, desc string, actions func()) {
	t := a.tk
	ui.Row(c).Padding(16, 20, 10, 20).Gap(12).AlignItems(ui.Start).Shrink(0).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(theme.FsHead).FontWeight(600)
			ui.Text(c, desc).TextColor(t.Tx3).LineHeight(1.5)
		})
		if actions != nil {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(actions)
		}
	})
}

// pageSplit 是页面主体:左侧主区(可滚动)+ 右侧抽屉。
func (a *App) pageSplit(c *ui.Context, key string, main func(), drawer func()) {
	a.pageSplitW(c, key, drawerW, nil, main, drawer)
}

// pageSplitW 同 pageSplit,抽屉宽度可调(画布的抽屉比其它页窄)。
func (a *App) pageSplitW(c *ui.Context, key string, w float32, st *ui.ScrollState, main func(), drawer func()) {
	t := a.tk
	ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
		sc := kit.VScroll(c).Key(key + "-main").Grow(1).MinWidth(0)
		if st != nil {
			sc.TrackScroll(st)
		}
		sc.Children(func() {
			ui.Column(c).Padding(4, 20, 24, 20).Children(main)
		})
		if drawer != nil {
			ui.Column(c).Key(key+"-drawer").Width(w).Shrink(0).Padding(14, 16).Gap(12).BorderWidth(0, 0, 0, 1).BorderColor(t.Bd1).Children(drawer)
		}
	})
}

// box 是白底描边卡片(设计稿 .card)。
func (a *App) box(c *ui.Context) ui.Element {
	t := a.tk
	e := ui.Column(c).Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd1)
	for _, s := range t.Shadow1() {
		e.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	return e
}

// writeTemp 把内容写到临时目录里的文件,返回路径。
func writeTemp(name, content string) (string, error) {
	dir, err := os.MkdirTemp("", "minicode-")
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, name)
	return p, os.WriteFile(p, []byte(content), 0o644)
}

// AskClaude 把提示词填进输入框并回到 Agent 的对话页(画布 / 浏览器的「让 Claude …」入口)。
func (a *App) AskClaude(prompt string) {
	a.draft = prompt
	a.screen = ScreenWork
	if a.mode != ModeAgent {
		a.SetMode(ModeAgent)
	}
}

// miniComposer 是抽屉底部的迷你输入框:一行输入 + 工具行(附件、可选标签、发送)。
func (a *App) miniComposer(c *ui.Context, key, ph string, buf *string, chip string, send func(text string)) {
	t := a.tk
	box := ui.Column(c).Key(key+"-box").Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd2)
	if box.FocusWithin() {
		box.BorderColor(t.Ac)
	}
	box.Children(func() {
		in := ui.TextArea(c.Key(key), buf).Lines(1, 4).Placeholder(ph).PlaceholderColor(t.Tx4).Background(ui.Transparent).BorderWidth(0).
			Padding(8, 10, 2, 10).FontSize(theme.FsBody).FocusRing(false).MinHeight(24)
		in.OnTextKey(func(k ui.TextKey) bool {
			if k.Key == ui.KeyEnter && k.Mods == 0 {
				txt := k.Text()
				k.SetText("")
				send(txt)
				return true
			}
			return false
		})
		ui.Row(c).Height(theme.Px(30)).PaddingX(4).Gap(4).AlignItems(ui.Center).Children(func() {
			kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "添加附件"})
			if chip != "" {
				kit.Chip(c, chip, kit.ChipOpt{Icon: "at", Line: true})
			}
			ui.Spacer(c).Grow(1)
			b := kit.Ib(c, "send", kit.IbOpt{Small: true, Tip: "发送 ⏎", NoHover: true}).Background(t.Ac).Radius(t.RSm()).TextColor(t.OnAc)
			kit.GlassAccent(b) // 流光:着色玻璃 + 强调色光晕
			b.OnClick(func() { send(*buf); *buf = "" })
		})
	})
}
