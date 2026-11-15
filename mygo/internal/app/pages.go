package app

import (
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 整页功能(用量 / 自动化 / 画布 / 浏览器 / 自定义)的公共骨架:
// 它们替换工作台的「编辑区 + 对话栏 + 检查器」,导航栏与标题栏不变,所以三套方案下都成立。

const drawerW = 300 // 右侧抽屉宽度(设计稿 .drawer)

// page 画当前整页功能;返回 false 表示不是整页。
func (a *App) page(c *ui.Context) bool {
	switch a.screen {
	case ScreenUsage:
		a.usagePage(c)
	case ScreenAutomation:
		a.autoPage(c)
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
			ui.Text(c, title).FontSize(theme.FsHead).FontWeight(700)
			ui.Text(c, desc).TextColor(t.Tx3).LineHeight(1.5)
		})
		if actions != nil {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(actions)
		}
	})
}

// pageSplit 是页面主体:左侧主区(可滚动)+ 右侧抽屉。
func (a *App) pageSplit(c *ui.Context, key string, main func(), drawer func()) {
	t := a.tk
	ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
		ui.Scroll(c).Key(key + "-main").Grow(1).MinWidth(0).Children(func() {
			ui.Column(c).Padding(4, 20, 24, 20).Children(main)
		})
		if drawer != nil {
			ui.Column(c).Key(key+"-drawer").Width(drawerW).Shrink(0).Padding(14, 16).Gap(12).BorderWidth(0, 0, 0, 1).BorderColor(t.Bd1).Children(drawer)
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
