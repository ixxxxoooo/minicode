package app

import (
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/canvas"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) cvLib() *canvas.Library {
	if a.cvLibData != nil {
		return a.cvLibData
	}
	if a.demoExtra {
		// 演示库每次运行用独立的临时目录:共用固定目录会跨次累积(测试 / 截图里列表越来越长)
		dir, err := os.MkdirTemp("", "minicode-demo-canvas-")
		if err != nil {
			dir = filepath.Join(os.TempDir(), "minicode-demo-canvas")
		}
		l, _ := canvas.OpenLibrary(dir)
		active, stores := demo.CanvasDemoLibrary(a.demoPgCanvas, a.now())
		for id, s := range stores {
			l.SeedDemo(id, s)
		}
		l.SetActive(active)
		a.cvLibData = l
		if s, ok := stores[active]; ok {
			a.canv = s
		}
		return l
	}
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".minicode", "canvases")
	l, err := canvas.OpenLibrary(dir)
	if err != nil {
		log.Warn("打开画布库失败", "err", err)
		l, _ = canvas.OpenLibrary(dir)
	}
	if len(l.IDs()) == 0 {
		doc := canvas.Doc{Title: "未命名画布", Status: "草稿", Blocks: []canvas.Block{
			{ID: "p1", Kind: canvas.Para, Lead: true, Anchor: "概述", Text: "在这里写下想法,或点「让 Claude 修改」让它帮你起草。"},
		}}
		s, id, err := l.New("未命名画布", doc, "你", "新建", a.now())
		if err != nil {
			log.Warn("新建默认画布失败", "err", err)
		}
		a.canv, _ = s, id
	} else {
		s, err := l.ActiveStore()
		if err == nil {
			a.canv = s
		}
	}
	a.cvLibData = l
	return l
}

func (a *App) canvasList(c *ui.Context) {
	t := a.tk
	l := a.cvLib()
	ui.Column(c).Width(216).Shrink(0).Padding(12, 12, 0, 16).Gap(8).BorderWidth(0, 1, 0, 0).BorderColor(t.Bd1).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "画布").FontWeight(500).Grow(1)
			if kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "新建"}).Clicked() {
				doc := canvas.Doc{Title: "未命名画布", Status: "草稿"}
				if s, id, err := l.New("未命名画布", doc, "你", "新建", a.now()); err == nil {
					a.canv = s
					l.SetActive(id)
					a.toast = "已新建画布"
				}
			}
		})
		a.canvasUndoBar(c)
		for _, id := range l.IDs() {
			s, err := l.Store(id)
			if err != nil {
				continue
			}
			a.canvasListRow(c, l, id, s, l.Active() == id)
		}
	})
}

// newBlankCanvas 新建一张空白画布并设为当前;失败只记日志并提示,不抛出。
func (a *App) newBlankCanvas() {
	l := a.cvLib()
	doc := canvas.Doc{Title: "未命名画布", Status: "草稿", Blocks: []canvas.Block{
		{ID: "p1", Kind: canvas.Para, Lead: true, Anchor: "概述", Text: "在这里写下想法,或点「让 Claude 修改」让它帮你起草。"},
	}}
	s, id, err := l.New("未命名画布", doc, "你", "新建", a.now())
	if err != nil {
		log.Warn("新建画布失败", "err", err)
		a.toast = "新建画布失败:" + err.Error()
	}
	if s != nil {
		a.canv = s
		l.SetActive(id)
		a.cv.verIx = 0
		a.cv.preview, a.cv.code = false, false
		if err == nil {
			a.toast = "已新建画布"
		}
	}
}

// canvasEmpty 画布库为空时的整页空态(左侧仍保留列表与撤销条,方便撤销刚做的删除)。
func (a *App) canvasEmpty(c *ui.Context) {
	t := a.tk
	a.pane(c).Grow(1).Children(func() {
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.canvasList(c)
			ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(10).Children(func() {
				kit.Ic(c, "canvas", kit.S16).TextColor(t.Tx4)
				ui.Text(c, "还没有画布").FontWeight(500)
				ui.Text(c, "新建一张,或让 Claude 在对话里起草").FontSize(theme.FsCap).TextColor(t.Tx3)
				kit.Btn(c, "新建画布", kit.BtnOpt{Icon: "plus"}).OnClick(a.newBlankCanvas)
			})
		})
	})
}
