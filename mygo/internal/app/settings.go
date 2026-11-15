package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 设置页(设计稿 v3:4 组 14 类)。内容由 settings_tabs.go 里的声明式定义给出,这里是引擎:
// 导航与搜索、行与控件的渲染、已修改标记与「恢复默认」、需重启条。
// 每个控件对应偏好里的一个键(prefs.Store),只存与默认不同的项;
// 会立刻影响界面的项(主题、布局、强调色、密度、圆角)在 applyPrefs 里生效。

type ctlKind int

const (
	kNone  ctlKind = iota // 右侧没有标准控件(自定义或纯说明行)
	kSeg                  // 分段:值是选中的下标
	kTog                  // 开关
	kStep                 // 步进器
	kChips                // 单选胶囊:值是选中的下标
	kField                // 单行输入
)

// setRow 是设置里的一行。
type setRow struct {
	key, title, desc string
	kind             ctlKind
	opts             []string
	def              any // int / bool / string,决定默认值与类型
	min, max, step   int
	unit             string
	restart          bool // 改动后需要重启才生效
	mono             bool
	w                float32 // 输入框宽度
	ph               string
	right            func(a *App, c *ui.Context) // kNone 时的自定义右侧内容
}

// setGroup 是一个分组卡:rows 声明式的行,或 body 自定义整块内容。
type setGroup struct {
	title string
	rows  []setRow
	body  func(a *App, c *ui.Context)
}

// setTabDef 是一个设置分类。
type setTabDef struct {
	id, title, desc, icon string
	groups                []setGroup
	note                  string // 分组之后的一行灰色说明
}

// 设置页的分组导航(设计稿 GROUPS)。
var setNavGroups = []struct {
	name string
	ids  []string
}{
	{"客户端", []string{"general", "appearance", "shortcuts"}},
	{"Claude Code", []string{"models", "chat", "permissions", "runtime"}},
	{"工作台", []string{"editor", "terminal", "scm", "browser"}},
	{"其他", []string{"privacy", "advanced", "about"}},
}

// settingsState 是设置页自己的临时状态。
type settingsState struct {
	query   string
	bufs    map[string]*string // 输入框的编辑缓冲(键 → 文本)
	restart map[string]bool    // 改过且需要重启的项
	mainSt  ui.ScrollState
	lastTab string
	// 搜索跳转高亮
	jumpKey    string
	flashKey   string
	flashUntil time.Time
	// 快捷键录制
	recID  string
	kbWarn string
	// 服务商(数据与编辑区状态见 prov_state.go) / 连通性
	pv        provState
	providers []claudecfg.Provider // 设置页列表(演示与旧 CRUD 路径)
	provBusy  map[string]int
	provEdit  bool
	provDraft claudecfg.Provider
	provKey   string
	net       map[string]int
	netMs     map[string]int64
	// 删除撤销(kit.UndoSlot)
	undo kit.UndoSlot
	// 输入校验抖动
	fieldShake map[string]time.Time
	enterAt    time.Time // 分类切换时分组 stagger 起点
	navDrop    bool      // 窄窗分类下拉展开
	// 权限 / 环境变量表单
	ruleDraft string
	ruleKind  int
	envName   string
	envVal    string
	envLayer  int
	// claude 版本检测
	claudeVer       string
	claudeVerErr    string
	claudeDetecting bool
	// claude doctor
	doctorOpen    bool
	doctorRunning bool
	doctorLines   []string
	// settings 导出
	settingsImport []byte
	// 项目信任(真实列表缓存)
	trustRows    []claudecfg.TrustEntry
	trustErr     string
	trustLoading bool
	trustRowErr  map[string]string // 路径 → 写失败文案
	// 语言服务器检测
	lspRows     []lspSettingRow
	lspScanning bool
	lspScanned  bool
	rulesErr    string
	wt          []worktreeRow // worktree 列表缓存(后台刷新,渲染里不跑 git)
	wtAt        time.Time
	wtBusy      bool
	cr          []crashBrief // 崩溃记录缓存(后台刷新)
	crAt        time.Time
	crBusy      bool
	envErr      string
}

func (a *App) setTabByID(id string) *setTabDef {
	for i := range settingsTabs {
		if settingsTabs[i].id == id {
			return &settingsTabs[i]
		}
	}
	return &settingsTabs[1] // 外观
}

// ───────── 偏好读写(带默认值) ─────────

func (a *App) prefInt(r setRow) int {
	d, _ := r.def.(int)
	return a.prefs.Int(r.key, d)
}

func (a *App) prefBool(r setRow) bool {
	d, _ := r.def.(bool)
	return a.prefs.Bool(r.key, d)
}

func (a *App) prefStr(r setRow) string {
	d, _ := r.def.(string)
	return a.prefs.Str(r.key, d)
}

// setPref 写入一项:等于默认值就删键(「已修改」标记随之消失),然后重新应用会影响界面的偏好。
func (a *App) setPref(r setRow, v any) {
	if !a.persistPrefs() {
		return
	}
	if v == r.def {
		a.prefs.Reset(r.key)
	} else {
		a.prefs.Set(r.key, v)
	}
	if r.restart {
		if a.set.restart == nil {
			a.set.restart = map[string]bool{}
		}
		if a.prefs.Has(r.key) {
			a.set.restart[r.key] = true
		} else {
			delete(a.set.restart, r.key)
		}
	}
	a.applyPrefs()
}

func (a *App) resetPref(r setRow) {
	a.prefs.Reset(r.key)
	delete(a.set.restart, r.key)
	delete(a.set.bufs, r.key)
	a.applyPrefs()
}

// 外观偏好的键。
const (
	kScheme  = "appearance.scheme"
	kTheme   = "appearance.theme"
	kAccent  = "appearance.accent"
	kDensity = "appearance.density"
	kRadius  = "appearance.radius"
)

// accentOf 返回强调色选项 i 的颜色(0 = 沿用主题自带);选项顺序与设计稿一致。
func accentOf(p theme.Palette, i int) string {
	switch i {
	case 1:
		return p.Info
	case 2:
		return p.Ok
	case 3:
		return p.Cyan
	case 4:
		return p.Vio
	case 5:
		return p.Pink
	case 6:
		return p.Err
	case 7:
		return p.Warn
	}
	return ""
}

// applyPrefs 把偏好里与外观相关的项读进应用状态并重算令牌。
// 偏好里没有的项保持创建应用时给定的值(截图工具与演示用)。
func (a *App) applyPrefs() {
	if a.prefs.Has(kTheme) {
		a.pal = theme.Find(a.prefs.Str(kTheme, a.pal.ID))
	}
	if a.prefs.Has(kScheme) {
		a.sch = theme.FindScheme(a.prefs.Str(kScheme, a.sch.ID))
	}
	a.accent = accentOf(a.pal, a.prefs.Int(kAccent, 0))
	a.density = a.prefs.Int(kDensity, 1)
	a.radius = a.prefs.Int(kRadius, 1)
	a.ApplySystemAppearance()
	a.applyBehavior()
	a.retheme()
}

// ───────── 页面 ─────────

// settingsPage 是整页设置:左侧导航(196)+ 右侧内容(620 居中)。
func (a *App) settingsPage(c *ui.Context) {
	t := a.tk
	a.initSettingsConfig()
	a.shortcutRecordCapture(c)
	ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Background(t.BG).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd2).Children(func() {
		if !a.settingsNarrow() {
			a.settingsNav(c)
		}
		a.settingsMain(c)
	})
}

func (a *App) settingsNav(c *ui.Context) {
	t := a.tk
	q := strings.TrimSpace(a.set.query)
	counts := map[string]int{}
	if q != "" {
		for _, h := range a.settingsHits(q) {
			counts[h.tab]++
		}
	}
	ui.Column(c).Width(196).Shrink(0).Padding(10, 8).Gap(1).Background(t.Chrome).BorderWidth(0, 1, 0, 0).BorderColor(t.Bd2).Children(func() {
		// 顶部「返回应用」:离开设置的出口放在最显眼的位置(与红绿灯同一视线),Esc 同效
		back := kit.RowEl(c, false).Key("set-back").Gap(6).Margin(0, 0, 6, 0).TextColor(t.Tx2)
		back.OnClick(a.closeSettings)
		back.Children(func() {
			kit.Ic(c, "arrowL", kit.S12)
			ui.Text(c, "返回应用").Grow(1)
			kit.Kbd(c, "esc")
		})
		// 搜索框
		ui.Row(c).Height(24).PaddingX(8).Gap(6).Radius(t.RSm()).Background(t.Fill1).AlignItems(ui.Center).Margin(0, 0, 6, 0).Children(func() {
			kit.Ic(c, "search", kit.S12).TextColor(t.Tx3)
			in := ui.TextInputBase(c.Key("set-q"), &a.set.query).Grow(1).FontSize(theme.FsCap + .5).Placeholder("搜索设置").PlaceholderColor(t.Tx4)
			in.OnShortcut(0, ui.KeyEscape, func() { a.set.query = "" })
		})
		ui.Scroll(c).Key("set-nav").Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Gap(1).Children(func() {
				for gi, g := range setNavGroups {
					h := kit.Grp(c, g.name)
					if gi > 0 {
						h.Margin(4, 0, 0, 0)
					}
					for _, id := range g.ids {
						tab := a.setTabByID(id)
						on := q == "" && a.setTab == id
						row := kit.RowEl(c, on).Key("setnav-" + id)
						if q != "" && counts[id] == 0 {
							row.Opacity(.42)
						}
						n := counts[id]
						row.OnClick(func() { a.setTab, a.set.query = id, "" })
						row.Children(func() {
							kit.Ic(c, tab.icon, kit.S12)
							ui.Text(c, tab.title).Grow(1).SingleLine()
							if n > 0 {
								kit.Cnt(c, itoa(n), true)
							}
						})
					}
				}
			})
		})
	})
}

// closeSettings 回到工作台。
func (a *App) closeSettings() {
	a.screen = ScreenWork
	a.set.query = ""
}

func (a *App) settingsMain(c *ui.Context) {
	t := a.tk
	q := strings.TrimSpace(a.set.query)
	tab := a.setTabByID(a.setTab)
	if a.set.lastTab != a.setTab+"|"+q {
		a.set.mainSt.Y = 0 // 换分类 / 搜索词回到顶部
		a.set.lastTab = a.setTab + "|" + q
		a.set.enterAt = time.Now()
	}
	ui.Column(c).Grow(1).MinWidth(0).Children(func() {
		ui.Scroll(c).Key("set-main").TrackScroll(&a.set.mainSt).Grow(1).MinHeight(0).Children(func() {
			ui.Row(c).Justify(ui.Center).Padding(18, 0, 28, 0).Children(func() {
				col := ui.Column(c).Width(620).Shrink(1).MinWidth(0).Gap(14)
				if a.settingsNarrow() {
					col.Width(0).Grow(1)
				}
				col.Children(func() {
					if a.settingsNarrow() && q == "" {
						a.settingsTabPicker(c)
					}
					a.settingsUndoToast(c)
					if q != "" {
						a.settingsSearchView(c, q)
						return
					}
					ui.Column(c).Gap(2).Children(func() {
						ui.Text(c, tab.title).FontSize(theme.FsHead).FontWeight(700)
						if tab.desc != "" {
							ui.Text(c, tab.desc).TextColor(t.Tx3).LineHeight(theme.LhBody)
						}
					})
					for gi, g := range tab.groups {
						a.settingsGroup(c, g, gi)
					}
					if tab.note != "" {
						ui.Text(c, tab.note).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(theme.LhBody).Padding(0, 4)
					}
					if a.setTab == "shortcuts" {
						a.shortcutWarnBar(c)
						kit.Btn(c, "恢复默认", kit.BtnOpt{Small: true, Icon: "undo"}).OnClick(a.resetShortcuts)
					}
					a.syncSettingJump()
				})
			})
		})
		a.restartBar(c)
	})
}

// settingsGroup 是一张分组卡(小标题 + 白底卡片)。
func (a *App) settingsGroup(c *ui.Context, g setGroup, index int) {
	t := a.tk
	grp := ui.Column(c).Gap(5)
	kit.StaggerRise(grp, index, a.set.enterAt, a.frameSoon)
	grp.Children(func() {
		if g.title != "" {
			ui.Text(c, g.title).FontSize(theme.FsCap).FontWeight(700).TextColor(t.Tx3).LetterSpacing(.2).Padding(0, 4)
		}
		card := ui.Column(c).Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd1)
		for _, s := range t.Shadow1() {
			card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
		card.Children(func() {
			// 同一分组既有行又有自定义块时,先画行再画块(原先有块就丢掉了行)。
			for i, r := range g.rows {
				a.settingsRow(c, r, i == 0)
			}
			if g.body != nil {
				g.body(a, c)
			}
		})
	})
}

// settingsRow 画一行:标题 + 说明在左,控件在右;改过的行左侧有强调色小点,悬停时出现「恢复默认」。
func (a *App) settingsRow(c *ui.Context, r setRow, first bool) {
	t := a.tk
	changed := r.key != "" && a.prefs.Has(r.key)
	row := ui.Row(c.Key("sr-"+r.key+r.title)).MinHeight(38).Padding(6, 12).Gap(12).AlignItems(ui.Center)
	if !first {
		row.BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1)
	}
	if a.settingRowFlash(r) {
		row.Background(t.AcSoft)
	} else if a.set.jumpKey == rowFlashID(r) {
		row.Background(t.AcSoft.Alpha(.6))
	}
	hov := row.Hovered()
	row.Children(func() {
		if changed {
			ui.Box(c).Absolute().Left(4).Top(16).Size(5, 5).Radius(3).Background(t.Ac)
		}
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, r.title).FontWeight(500)
				if r.restart {
					kit.Tag(c, "需重启", kit.KWarn)
				}
			})
			if r.desc != "" {
				ui.Text(c, r.desc).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(theme.LhUI)
			}
		})
		a.settingsControl(c, r)
		if changed && hov {
			kit.Ib(c, "undo", kit.IbOpt{Small: true, Tip: "恢复默认"}).OnClick(func() { a.resetPref(r) })
		}
	})
}

// settingsControl 画右侧的标准控件。
func (a *App) settingsControl(c *ui.Context, r setRow) {
	t := a.tk
	switch r.kind {
	case kSeg:
		items := make([]kit.SegItem, len(r.opts))
		for i, o := range r.opts {
			items[i] = kit.SegItem{Text: o}
		}
		if i := kit.Seg(c, items, a.prefInt(r), kit.SegOpt{Key: "seg-" + r.key}); i >= 0 {
			a.setPref(r, i)
		}
	case kTog:
		if kit.Tog(c, a.prefBool(r), "tog-"+r.key) {
			a.setPref(r, !a.prefBool(r))
		}
	case kStep:
		v := a.prefInt(r)
		ui.Row(c).Height(24).PaddingX(2).Gap(2).Radius(t.RSm()).Background(t.Raised).Border(1, t.Bd2).AlignItems(ui.Center).Shrink(0).Children(func() {
			btn := func(label string, d int) {
				b := ui.ButtonBase(c).Key("stp-"+r.key+label).Size(20, 20).Radius(t.RXs()).Justify(ui.Center).AlignItems(ui.Center).TextColor(t.Tx3).Transition(kit.Fade)
				if b.Hovered() {
					b.Background(t.Fill2).TextColor(t.Tx1)
				}
				if b.Clicked() {
					nv := v + d*max(r.step, 1)
					nv = min(max(nv, r.min), r.max)
					a.setPref(r, nv)
				}
				b.Children(func() { ui.Text(c, label) })
			}
			btn("−", -1)
			ui.Text(c, itoa(v)).FontWeight(500).MinWidth(26).FontFeatures("tnum").TextAlign(ui.Center)
			if r.unit != "" {
				ui.Text(c, r.unit).TextColor(t.Tx3)
			}
			btn("+", 1)
		})
	case kChips:
		cur := a.prefInt(r)
		ui.Row(c).Gap(8).Shrink(0).Children(func() {
			for i, o := range r.opts {
				ch := kit.Chip(c, o, kit.ChipOpt{Line: cur != i, On: cur == i}).Key("chip-" + r.key + itoa(i))
				if cur == i {
					ch.Background(t.AcSoft).TextColor(t.AcTx)
				}
				if ch.Clicked() {
					a.setPref(r, i)
				}
			}
		})
	case kField:
		a.settingsField(c, r)
	case kNone:
		if r.right != nil {
			r.right(a, c)
		}
	}
}

// settingsField 是设置里的单行输入(kit.Field:错误描边 + 抖动 + 说明)。
func (a *App) settingsField(c *ui.Context, r setRow) {
	if a.set.bufs == nil {
		a.set.bufs = map[string]*string{}
	}
	if a.set.fieldShake == nil {
		a.set.fieldShake = map[string]time.Time{}
	}
	buf, ok := a.set.bufs[r.key]
	if !ok {
		s := a.prefStr(r)
		buf = &s
		a.set.bufs[r.key] = buf
	}
	w := r.w
	if w == 0 {
		w = 160
	}
	shake := a.set.fieldShake[r.key]
	kit.Field(c, r.key, kit.FieldOpt{ShakeAt: &shake}, func(box ui.Element) {
		box.BorderWidth(0).Background(ui.Transparent).PaddingX(0).Height(24).Width(w).Shrink(0)
		in := ui.TextInputBase(c.Key("fld-"+r.key), buf).Grow(1).FontSize(theme.FsCap + .5).Placeholder(r.ph).PlaceholderColor(a.tk.Tx4)
		if r.mono {
			in.Font("monospace")
		}
		in.OnChange(func() { a.setPref(r, *buf) })
	})
}

// restartBar 是「需重启」条:有需要重启的项被改过时浮在内容底部。
func (a *App) restartBar(c *ui.Context) {
	n := len(a.set.restart)
	if n == 0 {
		return
	}
	t := a.tk
	ui.Row(c).Justify(ui.Center).Padding(0, 0, 14, 0).Children(func() {
		bar := ui.Row(c).Gap(10).Padding(7, 8, 7, 14).Radius(t.RXl()).Background(t.Raised).Border(1, t.Warn).AlignItems(ui.Center)
		for _, s := range t.Shadow3() {
			bar.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
		bar.Children(func() {
			kit.Ic(c, "refresh", kit.S12).TextColor(t.Warn)
			ui.Text(c, itoa(n)+" 项更改需要重启后生效")
			kit.Btn(c, "稍后", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.set.restart = nil })
			kit.Btn(c, "立即重启", kit.BtnOpt{Small: true, Variant: kit.BtnPri}).OnClick(func() {
				a.restartForSettings("设置项")
				a.set.restart = nil
			})
		})
	})
}

// ───────── 搜索 ─────────

type setHit struct {
	tab, tabTitle, group string
	row                  setRow
}

// settingsHits 在所有分类的行里按关键字(标题 + 说明 + 分组 + 分类名,不区分大小写)找命中。
func (a *App) settingsHits(q string) []setHit {
	k := strings.ToLower(q)
	var out []setHit
	for _, tab := range settingsTabs {
		for _, g := range tab.groups {
			rows := g.rows
			if g.body != nil && g.title != "" {
				rows = []setRow{{title: g.title}} // 自定义块按分组标题可被搜到
			}
			for _, r := range rows {
				if r.title == "" {
					continue
				}
				if strings.Contains(strings.ToLower(r.title+" "+r.desc+" "+g.title+" "+tab.title), k) {
					out = append(out, setHit{tab.id, tab.title, g.title, r})
				}
			}
		}
	}
	return out
}

func (a *App) settingsSearchView(c *ui.Context, q string) {
	t := a.tk
	hs := a.settingsHits(q)
	if len(hs) == 0 {
		ui.Column(c).AlignItems(ui.Center).Gap(8).Padding(48, 12).Children(func() {
			kit.Ic(c, "search", kit.S20).TextColor(t.Tx4)
			ui.Text(c, "没有匹配「"+q+"」的设置").FontWeight(600)
			ui.Text(c, "换个关键字试试,例如「主题」「代理」「快捷键」「服务商」。\n功能内容(MCP、Skills、命令……)在侧栏的「自定义」页。").
				FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(theme.LhMono).TextAlign(ui.Center)
			kit.Btn(c, "清除搜索", kit.BtnOpt{Small: true, Icon: "x"}).OnClick(func() { a.set.query = "" })
		})
		return
	}
	order := []string{}
	by := map[string][]setHit{}
	for _, h := range hs {
		if _, ok := by[h.tab]; !ok {
			order = append(order, h.tab)
		}
		by[h.tab] = append(by[h.tab], h)
	}
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, "搜索结果").FontSize(theme.FsHead).FontWeight(700)
		ui.Text(c, itoa(len(hs))+" 项,分布在 "+itoa(len(order))+" 个分类里 · 点一项直接跳过去 · Esc 清除").TextColor(t.Tx3)
	})
	for _, id := range order {
		tab := a.setTabByID(id)
		ui.Column(c).Gap(5).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(0, 4).TextColor(t.Tx3).Children(func() {
				kit.Ic(c, tab.icon, kit.S12)
				ui.Text(c, tab.title).FontSize(theme.FsCap).FontWeight(700)
			})
			ui.Column(c).Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd1).Children(func() {
				for i, h := range by[id] {
					h := h
					r := ui.ButtonBase(c).Key("hit-"+id+itoa(i)).MinHeight(38).Padding(6, 12).Gap(12).AlignItems(ui.Center).Cursor(ui.CursorPointer).Transition(kit.Fade)
					if i > 0 {
						r.BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1)
					}
					if r.Hovered() {
						r.Background(t.Fill1)
					}
					r.OnClick(func() { a.jumpSettingRow(h.tab, h.row) })
					r.Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
							a.highlight(c, h.row.title, q, 500, theme.FsBody, t.Tx1)
							if h.row.desc != "" {
								a.highlight(c, h.row.desc, q, 400, theme.FsCap, t.Tx3)
							}
						})
						if h.group != "" {
							ui.Text(c, h.group).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine()
						}
						kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
					})
				}
			})
		})
	}
}

// highlight 画一段文字,把其中命中查询的部分用强调色淡底标出。
func (a *App) highlight(c *ui.Context, s, q string, weight int, size float32, col ui.Color) {
	t := a.tk
	ls, lq := strings.ToLower(s), strings.ToLower(q)
	i := strings.Index(ls, lq)
	if i < 0 || lq == "" {
		ui.Text(c, s).FontSize(size).FontWeight(weight).TextColor(col)
		return
	}
	// 小写化可能改变字节长度(极少数字符),此时放弃高亮以免切坏 UTF-8
	if len(ls) != len(s) {
		ui.Text(c, s).FontSize(size).FontWeight(weight).TextColor(col)
		return
	}
	ui.RichText(c,
		ui.Span{Text: s[:i], Color: col},
		ui.Span{Text: s[i : i+len(q)], Color: t.AcTx, Background: t.AcSoft, Radius: 2},
		ui.Span{Text: s[i+len(q):], Color: col},
	).FontSize(size).FontWeight(weight)
}

// SetSettings 直接打开设置页的某个分类并填入搜索词(截图 / 调试用)。
func (a *App) SetSettings(tab, query string) {
	a.openSettings(tab)
	a.set.query = query
}
