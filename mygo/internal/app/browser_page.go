package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/browser"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 浏览器整页(设计稿 pages.js):地址栏 + 阅读模式正文 + 右侧「与此页面对话」抽屉(摘要 / 大纲 / 链接)。
// 页面模型与历史在 internal/browser;抓取目前只有演示数据源,真实网页抓取尚未接入。

type browserState struct {
	url     string // 地址栏文本
	lastURL string // 上次同步进地址栏的地址(变了才覆盖,免得打断输入)
	tab     string // sum / out / lnk
	hideDr  bool   // 用户关掉了抽屉
	ask     string
	st      ui.ScrollState
	pos     map[int]float32 // 节下标 → 相对正文顶端的 Y
}

// noFetch 是没有演示数据时的数据源:什么都抓不了。
type noFetch struct{}

func (noFetch) Fetch(string) (browser.Page, error) { return browser.Page{}, browser.ErrNoFetch }

func (a *App) brw() *browser.Browser {
	a.initBrowser()
	return a.br
}

// browseTo 在地址栏里提交一个地址。
func (a *App) browseTo(u string) {
	if !a.demoExtra && a.prefs != nil && !a.prefs.Bool(prefBrowserFetch, true) {
		a.toast = "抓取已关闭:设置 → 浏览器 → 允许抓取网页"
		return
	}
	b := a.brw()
	host := browser.Normalize(u)
	if err := b.Go(u); err != nil {
		log.Warn("浏览器打开失败", "host", host, "err", err)
		a.toast = err.Error()
	}
	a.brs.st.Y = 0
}

func (a *App) browserPage(c *ui.Context) {
	t := a.tk
	b := a.brw()
	v := &a.brs
	if v.pos == nil {
		v.pos = map[int]float32{}
		v.tab = "sum"
	}
	if b.URL() != v.lastURL {
		v.url, v.lastURL = b.URL(), b.URL()
	}
	pg := b.Page()
	a.pane(c).Grow(1).Children(func() {
		// 地址栏
		ui.Row(c).Height(40).Padding(0, 12).Gap(4).AlignItems(ui.Center).Shrink(0).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
			if kit.Ib(c, "chevL", kit.IbOpt{Tip: "后退", Color: dim(b.CanBack(), t)}).Clicked() {
				b.Back()
			}
			if kit.Ib(c, "chevR", kit.IbOpt{Tip: "前进", Color: dim(b.CanForward(), t)}).Clicked() {
				b.Forward()
			}
			if kit.Ib(c, "refresh", kit.IbOpt{Tip: "重新载入"}).Clicked() {
				b.Reload()
			}
			box := ui.Row(c.Key("br-urlbox")).Grow(1).MinWidth(0).Height(theme.Px(28)).PaddingX(8).Gap(6).AlignItems(ui.Center).Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd2)
			if box.FocusWithin() {
				box.BorderColor(t.Ac)
			}
			box.Children(func() {
				// 只有真的是 https 抓来的页面才显示绿色锁;明文 http(含本机开发服务器)用黄色警示,还没打开页面用中性地球
				switch {
				case pg.URL == "":
					kit.Ic(c, "globe", kit.S12).TextColor(t.Tx4)
				case pg.Insecure:
					kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
				default:
					kit.Ic(c, "lock", kit.S12).TextColor(t.Ok)
				}
				in := ui.TextInput(c.Key("br-url"), &v.url).Grow(1).MinWidth(0).Background(ui.Transparent).BorderWidth(0).Padding(0).FontSize(theme.FsBody).Placeholder("输入网址").PlaceholderColor(t.Tx4).FocusRing(false)
				in.OnTextKey(func(k ui.TextKey) bool {
					if k.Key == ui.KeyEnter && k.Mods == 0 {
						a.browseTo(k.Text())
						return true
					}
					return false
				})
				fav := b.Fav[b.URL()]
				ib := kit.Ib(c, "bookmark", kit.IbOpt{Small: true, Tip: "收藏"})
				if fav {
					ib.TextColor(t.Ac)
				}
				if ib.Clicked() && b.URL() != "" {
					if b.ToggleFav() {
						a.toast = "已收藏"
					} else {
						a.toast = "已取消收藏"
					}
				}
			})
			if kit.Ib(c, "ext", kit.IbOpt{Tip: "在系统浏览器中打开"}).Clicked() && b.URL() != "" {
				a.openExtBrowser(b.URL())
			}
			if kit.Chip(c, "与此页面对话", kit.ChipOpt{Icon: "chat", Line: true}).Clicked() {
				v.hideDr = !v.hideDr
			}
			if kit.Ib(c, "gear", kit.IbOpt{Tip: "浏览器设置"}).Clicked() {
				a.openSettings("browser")
			}
		})
		var drawer func()
		if !v.hideDr && pg.URL != "" {
			drawer = func() { a.browserDrawer(c, b, pg) }
		}
		a.pageSplitW(c, "br", drawerW, &v.st, func() {
			ui.Column(c).Grow(1).MinHeight(0).Children(func() {
				a.browserReader(c, b, pg)
				a.browserReaderSelectLayer(c, pg)
			})
		}, drawer)
	})
}

// browserReader 是阅读模式正文:居中一列,每节自己量位置供大纲定位。
func (a *App) browserReader(c *ui.Context, b *browser.Browser, pg browser.Page) {
	t := a.tk
	v := &a.brs
	ui.Row(c).Justify(ui.Center).Children(func() {
		col := ui.Column(c).Key("br-col").Width(640).Shrink(1).MinWidth(0).Padding(14, 0, 24, 0)
		col.Children(func() {
			top := col.Bounds().Y
			if b.Err != nil {
				a.box(c).Padding(12, 14).Gap(4).Margin(0, 0, 14, 0).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
						ui.Text(c, "无法打开 "+v.url).FontWeight(500).SingleLine()
					})
					ui.Text(c, b.Err.Error()+"。可尝试在系统浏览器中打开,或到 设置 → 浏览器 检查抓取开关与超时。").TextColor(t.Tx3).LineHeight(1.5)
					kit.Btn(c, "在系统浏览器打开", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).OnClick(func() { a.openExtBrowser(v.url) })
				})
			}
			if pg.URL == "" {
				if b.Err == nil {
					ui.Column(c).AlignItems(ui.Center).Gap(6).Padding(80, 0, 0, 0).Children(func() {
						kit.Ic(c, "globe", kit.S20).TextColor(t.Tx4)
						ui.Text(c, "阅读模式浏览器").FontWeight(500)
						ui.Text(c, "在上方地址栏输入网址并回车,正文会以干净的版式显示,还能就这个页面向 Claude 提问。").
							FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.5).TextAlign(ui.Center)
					})
				}
				return
			}
			if pageHasNoText(pg) {
				// 空态:抓到了页面但没有正文(纯脚本渲染 / 非 HTML / 空页面),给原因与下一步
				a.box(c).Padding(16, 14).Gap(6).Margin(0, 0, 14, 0).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						kit.Ic(c, "file", kit.S12).TextColor(t.Tx3)
						ui.Text(c, "这个页面没有可读取的正文").FontWeight(500).SingleLine()
					})
					ui.Text(c, pg.Site+" 返回了内容,但阅读模式没找到文字:可能是纯文本 / 图片、需要脚本渲染的页面,或本身就是空页面。").
						TextColor(t.Tx3).LineHeight(1.5)
					kit.Btn(c, "在系统浏览器打开", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).OnClick(func() { a.openExtBrowser(v.url) })
				})
				return
			}
			ui.Text(c, pg.Site+" · 阅读约 "+itoa(pg.ReadMin)+" 分钟").FontSize(theme.FsCap).TextColor(t.Tx3).Margin(0, 0, 6, 0)
			for i, s := range pg.Sections {
				el := ui.Column(c.Key("brs-" + itoa(i)))
				el.Children(func() {
					switch s.Level {
					case 0:
						ui.Text(c, s.Heading).FontSize(theme.Px(26)).FontWeight(600).Margin(2, 0, 14, 0)
					case 1:
						ui.Text(c, s.Heading).FontSize(theme.Px(15)).FontWeight(600).Padding(16, 0, 8, 0)
					default:
						ui.Text(c, s.Heading).FontSize(theme.Px(13)).FontWeight(650).Padding(12, 0, 6, 0)
					}
					markdown.View(c, padCode(s.Body), a.mdOptions())
				})
				v.pos[i] = el.Bounds().Y - top
			}
		})
	})
}

func (a *App) browserDrawer(c *ui.Context, b *browser.Browser, pg browser.Page) {
	t := a.tk
	v := &a.brs
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "与此页面对话").FontWeight(500).Grow(1).SingleLine()
		kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭"}).OnClick(func() { v.hideDr = true })
	})
	if id := kit.Tabs(c, []kit.TabItem{{ID: "sum", Text: "摘要"}, {ID: "out", Text: "大纲"}, {ID: "lnk", Text: "链接"}}, v.tab, 30, nil); id != "" {
		v.tab = id
	}
	switch v.tab {
	case "sum":
		ui.Column(c).Gap(7).Children(func() {
			for _, s := range pg.Summary {
				ui.Row(c).Gap(7).AlignItems(ui.Start).Children(func() {
					ui.Box(c).Size(6, 6).Radius(3).Background(t.Ac).Margin(7, 0, 0, 0).Shrink(0)
					ui.Text(c, s).LineHeight(1.55).Grow(1).MinWidth(0)
				})
			}
		})
	case "out":
		ui.Column(c).Margin(0, -6).Children(func() {
			for _, o := range pg.Outline() {
				r := kit.RowEl(c, false).Key("bo-" + itoa(o.Index)).PaddingStart(8 + float32(o.Level)*14).Justify(ui.Start)
				r.OnClick(func() {
					if y, ok := v.pos[o.Index]; ok {
						v.st.Y = max(0, y-4)
					}
				})
				r.Children(func() { ui.Text(c, o.Text).SingleLine() })
			}
		})
	case "lnk":
		ui.Column(c).Margin(0, -6).Children(func() {
			for i, l := range pg.Links {
				r := kit.RowEl(c, false).Key("bl-"+itoa(i)).Justify(ui.Start).Padding(5, 8).Height(40)
				r.OnClick(func() {
					if l.URL == "" {
						a.toast = "此链接(" + l.Host + ")暂不能在内置浏览器里打开"
						return
					}
					a.browseTo(l.URL)
				})
				r.Children(func() {
					ui.Column(c).Grow(1).MinWidth(0).Children(func() {
						ui.Text(c, l.Title).SingleLine()
						ui.Text(c, l.Host).FontSize(theme.FsMicro).TextColor(t.Tx4)
					})
					kit.Ic(c, "ext", kit.S12).TextColor(t.Tx4)
				})
			}
		})
	}
	ui.Column(c).Gap(6).Children(func() {
		kit.Grp(c, "快捷提示")
		ui.Row(c).Gap(6).Wrap().Children(func() {
			for _, q := range []string{"总结要点", "提取代码示例", "翻译成中文", "列出相关链接"} {
				if kit.Chip(c, q, kit.ChipOpt{Line: true}).Clicked() {
					a.fillDraftWithPage(pg, q)
				}
			}
		})
	})
	ui.Spacer(c).Grow(1)
	a.miniComposer(c, "br-ask", "就这个页面提问…", &v.ask, "", func(text string) {
		if text != "" {
			a.fillDraftWithPage(pg, text)
		}
	})
}

// dim 给不可用的图标按钮淡色(可用返回零值 = 默认色)。
func dim(ok bool, t *theme.Tokens) ui.Color {
	if ok {
		return ui.Color{}
	}
	return t.Tx4
}

// pageHasNoText 页面是否没有任何可显示的文字(标题与正文都为空)。
func pageHasNoText(pg browser.Page) bool {
	for _, s := range pg.Sections {
		if strings.TrimSpace(s.Heading) != "" || strings.TrimSpace(s.Body) != "" {
			return false
		}
	}
	return true
}
