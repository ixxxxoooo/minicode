package app

import (
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 长输出限高折叠:超过 12 行限高 196px 内部滚动,展开后上限 480px,运行中贴底跟随。

const (
	foldLineCap   = 12
	foldMaxH      = 196
	foldExpandMax = 480
)

func foldKey(prefix, id string) string { return prefix + ":" + id }

func (a *App) foldExpanded(key string) bool {
	if a.outFold == nil {
		return false
	}
	return a.outFold[key]
}

func (a *App) setFold(key string, on bool) {
	if a.outFold == nil {
		a.outFold = map[string]bool{}
	}
	if on {
		a.outFold[key] = true
	} else {
		delete(a.outFold, key)
	}
}

// foldBody 渲染可折叠的等宽正文;live 为真时展开区滚到底。
func (a *App) foldBody(c *ui.Context, key, full string, live bool, mono bool, copy func()) {
	t := a.tk
	lines := strings.Split(strings.TrimRight(full, "\n"), "\n")
	n := len(lines)
	if n == 0 {
		return
	}
	exp := a.foldExpanded(key)
	needFold := n > foldLineCap && !exp
	maxH := foldExpandMax
	show := lines
	if needFold {
		maxH = foldMaxH
		show = lines[:foldLineCap]
	}
	var st ui.ScrollState
	if live {
		st.Y = math.MaxFloat32
	}
	// 只创建一个元素:先建 Column 再换成 Scroll 会让同一个键注册两次(同父下重复键 panic)。
	inner := func() {
		ui.Column(c).Padding(6, 10).Gap(0).Children(func() {
			for i, ln := range show {
				row := ui.Row(c).Key(i).Height(monoLH).AlignItems(ui.Center)
				if mono {
					row.Children(func() {
						ui.RichText(c, a.termLine(ln)...).NoWrap().Clip()
						if live && i == len(show)-1 && !needFold {
							kit.Caret(c)
						}
					})
				} else {
					ui.Text(c, ln).FontSize(theme.FsCap).LineHeight(1.6).TextColor(t.Tx1)
				}
			}
		})
	}
	if needFold || exp {
		if live && st.Y >= st.MaxY {
			st.Y = math.MaxFloat32
		}
		ui.Scroll(c).TrackScroll(&st).MaxHeight(float32(maxH)).Key("foldout-" + key).Children(inner)
	} else {
		ui.Column(c).Key("foldout-" + key).Children(inner)
	}
	if n > foldLineCap {
		ui.Row(c).Height(26).PaddingX(10).Gap(8).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Children(func() {
			label := "展开 · 共 " + itoa(n) + " 行"
			if exp {
				label = "收起"
			}
			kit.Btn(c, label, kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
				a.setFold(key, !exp)
			})
			if copy != nil {
				kit.Btn(c, "复制", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "copy"}).OnClick(copy)
			}
		})
	}
}

// foldCodeBlock 是会话流 Markdown 围栏代码块的折叠渲染。
func (a *App) foldCodeBlock(c *ui.Context, b *markdown.CodeBlockView) {
	key := foldKey("cb", itoa(b.Index))
	body := strings.TrimRight(b.Code(), "\n")
	a.foldBody(c, key, body, false, true, func() { a.copyOut(body, "代码") })
}

// hookMDFold 给 markdown.Options 装上代码块折叠。
func (a *App) hookMDFold(o *markdown.Options) {
	prev := o.CodeBlock
	o.CodeBlock = func(c *ui.Context, b *markdown.CodeBlockView) {
		if isMermaidInfo(b.Info) {
			a.foldMermaidBlock(c, b)
			return
		}
		if prev != nil {
			prev(c, b)
			return
		}
		t := a.tk
		box := ui.Column(c).Background(t.Sunken).Radius(t.RMd()).Border(1, t.Bd1).Clip()
		box.Children(func() {
			ui.Row(c).Height(24).PaddingX(8).Gap(6).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
				if b.Info != "" {
					ui.Text(c, b.Info).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Grow(1)
				} else {
					ui.Spacer(c).Grow(1)
				}
				b.CopyButton(c)
			})
			a.foldBody(c, foldKey("cb", itoa(b.Index)), bodyTrim(b.Code()), false, true, func() {
				a.copyOut(b.Code(), "代码")
			})
		})
	}
}

func bodyTrim(s string) string { return strings.TrimRight(s, "\n") }
