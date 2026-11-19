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
			ui.Text(c, "今天想在 "+a.projectName()+" 里做点什么?").FontSize(theme.FsHead).FontWeight(600).Margin(2, 0, 0, 0)
			ui.Text(c, a.welcomeStatus()).TextColor(t.Tx3)
		})
		a.flowCol(c, func() {
			a.welcomeFocus = true
			a.composer(c)
			a.welcomeFocus = false
		})
		a.welcomeStarters(c)
		a.welcomeRecentBlock(c, t) // 设计稿顺序:输入框 → 起手卡 → 最近会话
	})
}

// 起手卡的三种排法:四张一排(放得下)/ 两列两行 / 单列(窄档)。不再出现 3+1 折行落单的样子。
const (
	starterCardW = 150 // 一排四张时每张的宽度
	starterGap   = 8
	starterMaxW  = 640
)

// welcomeStarters 起手卡(设计稿 .starters):按这一栏实际能用的宽度选排法——
// 四张一排放得下就一排;放不下就 2×2 两列(卡片等分宽度);再窄就单列只留图标与标题。
func (a *App) welcomeStarters(c *ui.Context) {
	list := a.starterList()
	box := ui.Column(c).Key("starters").WidthPercent(100).MaxWidth(starterMaxW).AlignItems(ui.Center).Gap(starterGap)
	w := box.Bounds().W // 上一帧量到的宽度;第一帧还没有,按对话栏宽度估
	if w <= 0 {
		w = min(a.chatW-40, starterMaxW)
	}
	cols := 1
	switch {
	case a.chatTier() == tierNarrow || w < 2*starterCardW+starterGap:
		cols = 1
	case w >= 4*starterCardW+3*starterGap:
		cols = 4
	default:
		cols = 2
	}
	box.Children(func() {
		switch cols {
		case 4:
			ui.Row(c).Gap(starterGap).Justify(ui.Center).Children(func() {
				for _, s := range list {
					a.starterCard(c, s, starterFixed)
				}
			})
		case 2:
			for i := 0; i < len(list); i += 2 {
				ui.Row(c).Gap(starterGap).WidthPercent(100).Children(func() {
					for _, s := range list[i:min(i+2, len(list))] {
						a.starterCard(c, s, starterFlex)
					}
				})
			}
		default:
			ui.Column(c).Gap(6).WidthPercent(100).PaddingX(20).Children(func() {
				for _, s := range list {
					a.starterCard(c, s, starterCompact)
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
	ui.Column(c).Gap(4).WidthPercent(100).MaxWidth(600).PaddingX(20).Margin(10, 0, 0, 0).Children(func() {
		ui.Text(c, "最近会话").FontSize(theme.FsCap).FontWeight(500).TextColor(t.Tx3)
		for _, r := range rows {
			r := r
			key := r.ID
			if key == "" {
				key = r.Title
			}
			row := kit.RowEl(c, false).Height(theme.Px(28)).PaddingX(8).Gap(8).Key("wrec-" + key)
			row.OnClick(func() { a.openWelcomeRecent(r.ID, r.Title) })
			row.Children(func() {
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

type starterMode int

const (
	starterFixed   starterMode = iota // 固定 150 宽,一排四张
	starterFlex                       // 两列时等分宽度
	starterCompact                    // 单列:一行,只有图标与标题
)

func (a *App) starterCard(c *ui.Context, s starter, mode starterMode) {
	t := a.tk
	narrow := mode == starterCompact
	b := ui.ButtonBase(c).Key(s.title).Radius(t.RMd()).Background(t.Raised).Cursor(ui.CursorPointer).
		Transition(ui.ElementTransition{Position: true, Shadows: true, Duration: kit.DurBase, Ease: kit.EaseOut})
	if narrow {
		b.WidthPercent(100).Height(theme.Px(32)).PaddingX(10).Gap(8).AlignItems(ui.Center).Justify(ui.Start)
	} else if mode == starterFlex {
		b.Grow(1).Basis(0).MinWidth(0).Padding(9, 10).Gap(3).AlignItems(ui.Start)
	} else {
		b.Width(starterCardW).Padding(9, 10).Gap(3).AlignItems(ui.Start)
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
			ui.Text(c, s.title).FontWeight(500)
			return
		}
		ui.Column(c).Gap(3).Grow(1).MinWidth(0).Children(func() {
			// 设计稿里图标是行内元素,所在行高约 20px
			ui.Row(c).Height(theme.Px(20)).AlignItems(ui.Center).Children(func() { kit.Ic(c, s.icon, 14).TextColor(t.AcTx) })
			ui.Text(c, s.title).FontWeight(500).SingleLine()
			ui.Text(c, s.desc).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.35)
		})
	})
}
