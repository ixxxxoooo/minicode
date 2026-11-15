package app

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 新会话欢迎页(设计稿 .welcome):会话还没有任何内容时的第一屏——
// 居中的问候、状态行、输入框、四张起手卡(悬停上浮,点击把提示填进输入框)。

type starter struct{ icon, title, desc, prompt string }

var starters = []starter{
	{"bug", "修复一个 Bug", "描述现象,我来定位", "路由匹配对大小写敏感,帮我定位并修复"},
	{"book", "解释这个项目", "梳理结构与关键流程", "梳理一下这个项目的结构与关键流程"},
	{"check", "补充测试", "为最近改动补单测", "为最近的改动补充单元测试"},
	{"eye", "审查改动", "", "审查当前未提交的改动,指出风险"}, // desc 由 starterList 按仓库真实状态填
}

// starterList 起手卡列表:「审查改动」的副标题取仓库里真实的未提交改动(文件数与增删行数),
// 还没读到状态时只写一句说明,不再写死数字。
func (a *App) starterList() []starter {
	out := make([]starter, len(starters))
	copy(out, starters)
	for i := range out {
		if out[i].icon == "eye" {
			out[i].desc = a.reviewStarterDesc()
		}
	}
	return out
}

func (a *App) reviewStarterDesc() string {
	if !a.sc.loaded {
		return "审查未提交的改动"
	}
	files := map[string]bool{}
	add, del := 0, 0
	for _, ch := range a.sc.changes {
		files[ch.Path] = true
		add += ch.Add
		del += ch.Del
	}
	if len(files) == 0 {
		return "没有未提交的改动"
	}
	return itoa(len(files)) + " 个文件 · +" + itoa(add) + " −" + itoa(del)
}

// projectName 当前会话所在项目名。
func (a *App) projectName() string {
	if a.proj != "" {
		return filepath.Base(a.proj)
	}
	for _, p := range a.projects {
		for _, s := range p.Sessions {
			if s.ID == a.curSess {
				return p.Name
			}
		}
	}
	if len(a.projects) > 0 {
		return a.projects[0].Name
	}
	return "项目"
}

func (a *App) welcome(c *ui.Context) {
	t := a.tk
	a.chatHead(c, "新对话")
	ui.Column(c).Grow(1).MinHeight(0).AlignItems(ui.Center).Justify(ui.Center).Padding(0, 0, 48, 0).Gap(16).Children(func() {
		ui.Column(c).Gap(3).AlignItems(ui.Center).Children(func() {
			a.brandLogo(c, 64, true)
			ui.Text(c, "今天想在 "+a.projectName()+" 里做点什么?").FontSize(theme.FsHead).FontWeight(700)
			ui.Text(c, a.welcomeStatus()).TextColor(t.Tx3)
		})
		ui.Column(c).WidthPercent(100).MaxWidth(600).PaddingX(20).Children(func() {
			a.welcomeFocus = true
			a.composer(c)
			a.welcomeFocus = false
		})
		a.welcomeRecentBlock(c, t)
		if a.chatTier() == tierNarrow { // 窄档:起手卡单列,只留图标与标题
			ui.Column(c).Gap(6).WidthPercent(100).PaddingX(20).Children(func() {
				for _, s := range a.starterList() {
					a.starterCard(c, s)
				}
			})
		} else {
			// 放不下四张时折行居中,不要横向溢出被裁掉
			ui.Row(c).Gap(6).Wrap().Justify(ui.Center).WidthPercent(100).MaxWidth(640).PaddingX(20).Children(func() {
				for _, s := range a.starterList() {
					a.starterCard(c, s)
				}
			})
		}
	})
}

func (a *App) welcomeRecentBlock(c *ui.Context, t *theme.Tokens) {
	var rows []welcomeRecent
	if a.demoWs() {
		for _, pair := range demo.WelcomeRecent() {
			rows = append(rows, welcomeRecent{Title: pair[0], When: pair[1]})
		}
	} else {
		rows = a.welcomeRecentRows()
	}
	if len(rows) == 0 {
		return
	}
	ui.Column(c).Gap(4).WidthPercent(100).MaxWidth(600).PaddingX(20).Margin(4, 0, 0, 0).Children(func() {
		ui.Text(c, "最近会话").FontSize(theme.FsCap).FontWeight(600).TextColor(t.Tx3)
		for _, r := range rows {
			r := r
			key := r.ID
			if key == "" {
				key = r.Title
			}
			row := kit.RowEl(c, false).Height(28).PaddingX(8).Gap(8).Key("wrec-" + key)
			row.OnClick(func() { a.openWelcomeRecent(r.ID, r.Title) })
			row.Children(func() {
				kit.Ic(c, "chat", kit.S12).TextColor(t.Tx4)
				ui.Text(c, r.Title).Grow(1).SingleLine()
				ui.Text(c, r.When).FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
		}
	})
}

func (a *App) openWelcomeRecent(id, title string) {
	if strings.TrimSpace(id) != "" {
		a.OpenSession(id)
		return
	}
	for _, p := range a.projects {
		for _, s := range p.Sessions {
			if s.Title == title || strings.Contains(s.Title, title) {
				a.OpenSession(s.ID)
				return
			}
		}
	}
}

func (a *App) starterCard(c *ui.Context, s starter) {
	t := a.tk
	narrow := a.chatTier() == tierNarrow
	b := ui.ButtonBase(c).Key(s.title).Radius(t.RMd()).Background(t.Raised).Cursor(ui.CursorPointer).
		Transition(ui.ElementTransition{Position: true, Shadows: true, Duration: kit.DurBase, Ease: kit.EaseOut})
	if narrow {
		b.WidthPercent(100).Height(32).PaddingX(10).Gap(8).AlignItems(ui.Center).Justify(ui.Start)
	} else {
		b.Width(150).Padding(9, 10).Gap(3)
	}
	sh := t.Shadow1()
	if b.Hovered() {
		sh = t.Shadow2()
		if !narrow {
			b.Translate(0, -2)
		}
	}
	for _, x := range sh {
		b.Shadow(x.X, x.Y, x.Blur, x.Spread, x.Color)
	}
	b.OnClick(func() { a.draft = s.prompt })
	b.Children(func() {
		if narrow {
			kit.Ic(c, s.icon, kit.S12).TextColor(t.AcTx)
			ui.Text(c, s.title).FontWeight(700)
			return
		}
		ui.Column(c).Gap(3).Grow(1).Children(func() {
			kit.Ic(c, s.icon, 14).TextColor(t.AcTx)
			ui.Text(c, s.title).FontWeight(700)
			ui.Text(c, s.desc).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.35)
		})
	})
}
