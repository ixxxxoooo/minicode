package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// 合并冲突标签(三方合并):把文件里的 <<<<<<< / ======= / >>>>>>> 区段解析出来,
// 每处冲突并排给「当前」与「传入」两侧,逐处选择接受当前 / 接受传入 / 接受双方,
// 底部是合并结果预览;全部处理完后「标记为已解决」写回文件并暂存。

const (
	chNone = iota
	chCur
	chInc
	chBoth
)

// confSeg 是文件里的一段:普通文本(plain)或一处冲突。
type confSeg struct {
	plain          []string
	conflict       bool
	cur, inc       []string
	curTag, incTag string
	line           int // 冲突起始行(1 起)
}

// parseConflicts 把带冲突标记的文件拆成段;diff3 的 ||||||| 基底部分会被丢弃。
func parseConflicts(src string) []confSeg {
	lines := strings.Split(src, "\n")
	var segs []confSeg
	var plain []string
	flush := func() {
		if len(plain) > 0 {
			segs = append(segs, confSeg{plain: plain})
			plain = nil
		}
	}
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if !strings.HasPrefix(l, "<<<<<<<") {
			plain = append(plain, l)
			continue
		}
		// 找到闭合的 >>>>>>>,不完整的标记当普通文本
		mid, end := -1, -1
		for j := i + 1; j < len(lines); j++ {
			switch {
			case strings.HasPrefix(lines[j], "=======") && mid < 0:
				mid = j
			case strings.HasPrefix(lines[j], ">>>>>>>") && mid >= 0:
				end = j
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			plain = append(plain, l)
			continue
		}
		flush()
		curEnd := mid
		for j := i + 1; j < mid; j++ {
			if strings.HasPrefix(lines[j], "|||||||") {
				curEnd = j
				break
			}
		}
		if curEnd == mid {
			curEnd = mid
		}
		seg := confSeg{conflict: true, line: i + 1,
			cur: lines[i+1 : minInt(curEnd, mid)], inc: lines[mid+1 : end],
			curTag: strings.TrimSpace(strings.TrimPrefix(l, "<<<<<<<")), incTag: strings.TrimSpace(strings.TrimPrefix(lines[end], ">>>>>>>"))}
		segs = append(segs, seg)
		i = end
	}
	flush()
	return segs
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// composeConflicts 按各处的选择拼出结果;未选择的保留成冲突标记原样。
func composeConflicts(segs []confSeg, choice map[int]int) string {
	var out []string
	k := 0
	for _, s := range segs {
		if !s.conflict {
			out = append(out, s.plain...)
			continue
		}
		switch choice[k] {
		case chCur:
			out = append(out, s.cur...)
		case chInc:
			out = append(out, s.inc...)
		case chBoth:
			out = append(out, s.cur...)
			out = append(out, s.inc...)
		default:
			out = append(out, "<<<<<<< "+s.curTag)
			out = append(out, s.cur...)
			out = append(out, "=======")
			out = append(out, s.inc...)
			out = append(out, ">>>>>>> "+s.incTag)
		}
		k++
	}
	return strings.Join(out, "\n")
}

// confState 是一个冲突文件的界面状态。
type confState struct {
	src    string // 解析时的原文(检测磁盘是否变过)
	segs   []confSeg
	choice map[int]int // 第几处冲突 → 选择
}

func (s *confState) count() (total, done int) {
	for _, g := range s.segs {
		if g.conflict {
			total++
		}
	}
	for _, v := range s.choice {
		if v != chNone {
			done++
		}
	}
	return
}

// conflictOf 取(必要时载入)某个文件的冲突状态。
func (a *App) conflictOf(p string) *confState {
	if a.cf == nil {
		a.cf = map[string]*confState{}
	}
	src, err := a.ide.fs.Read(p)
	if err != nil {
		return nil
	}
	if s := a.cf[p]; s != nil && s.src == src {
		return s
	}
	s := &confState{src: src, segs: parseConflicts(src), choice: map[int]int{}}
	a.cf[p] = s
	return s
}

// resolveAll 把所有未处理的冲突按同一种方式处理。
func (s *confState) resolveAll(ch int) {
	k := 0
	for _, g := range s.segs {
		if g.conflict {
			s.choice[k] = ch
			k++
		}
	}
}

// finishConflict 把结果写回文件、暂存并关闭标签。
func (a *App) finishConflict(tb *ideTab, s *confState) {
	p := tb.ref
	text := composeConflicts(s.segs, s.choice)
	if err := a.ide.fs.Write(p, text); err != nil {
		a.toast = "写入失败:" + err.Error()
		return
	}
	a.ide.invalidate()
	delete(a.cf, p)
	a.scmAct("标记为已解决", func(r vcs.Repo) error { return r.Stage(p) }, func() {
		a.toast = "已解决冲突并暂存 " + p
		a.ide.closeTab(tb.id)
	})
}

// conflictView 画合并冲突标签。
func (a *App) conflictView(c *ui.Context, tb *ideTab) {
	t := a.tk
	s := a.conflictOf(tb.ref)
	if s == nil {
		ui.Text(c, "读取不到这个文件").TextColor(t.Tx3).Padding(16)
		return
	}
	total, done := s.count()
	narrow := a.wsNarrow()
	a.wsHead(c, "fork", t.Err, tb.ref, func() {
		kk := kit.KWarn
		if total > 0 && done == total {
			kk = kit.KOk
		}
		kit.Tag(c, itoa(done)+" / "+itoa(total)+" 已处理", kk)
	}, func() {
		if !narrow {
			if kit.Btn(c, "全部接受当前", kit.BtnOpt{Small: true}).Clicked() {
				s.resolveAll(chCur)
			}
			if kit.Btn(c, "全部接受传入", kit.BtnOpt{Small: true}).Clicked() {
				s.resolveAll(chInc)
			}
		}
		if kit.Btn(c, "让 Claude 解决", kit.BtnOpt{Small: true, Icon: "star"}).Clicked() {
			a.draft = strings.TrimSpace(a.draft + "\n请帮我解决 " + tb.ref + " 里的合并冲突,说明每处你选择的理由。")
			a.rightOn = a.rightOn || a.mode == ModeIDE
			a.toast = "已把冲突文件引用到对话"
		}
		ready := total > 0 && done == total
		b := kit.Btn(c, "标记为已解决", kit.BtnOpt{Small: true, Variant: kit.BtnPri, Icon: "check"})
		if !ready {
			b.Opacity(.5)
		} else if b.Clicked() {
			a.finishConflict(tb, s)
		}
	})
	ui.Scroll(c).Key("conf-" + tb.ref).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(12).Gap(12).Children(func() {
			k := 0
			for _, g := range s.segs {
				if !g.conflict {
					continue
				}
				a.conflictCard(c, s, g, k, narrow)
				k++
			}
			if total == 0 {
				ui.Text(c, "这个文件里没有冲突标记。").TextColor(t.Tx3)
			}
			a.conflictResult(c, s)
		})
	})
}

// conflictCard 是一处冲突:当前 / 传入两栏 + 三个接受按钮。
func (a *App) conflictCard(c *ui.Context, s *confState, g confSeg, k int, narrow bool) {
	t := a.tk
	ch := s.choice[k]
	ui.Column(c).Key("cf-"+itoa(k)).Radius(t.RMd()).Border(1, t.Bd2).Background(t.Raised).Children(func() {
		ui.Row(c).Height(30).PaddingX(10).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
			ui.Text(c, "冲突 "+itoa(k+1)).FontWeight(700)
			ui.Text(c, "第 "+itoa(g.line)+" 行").FontSize(theme.FsCap).TextColor(t.Tx4)
			ui.Spacer(c).Grow(1)
			if ch != chNone {
				kit.Tag(c, map[int]string{chCur: "已接受当前", chInc: "已接受传入", chBoth: "已接受双方"}[ch], kit.KOk)
			} else {
				kit.Tag(c, "待处理", kit.KWarn)
			}
		})
		side := func(title, tag string, lines []string, tint ui.Color, picked bool, key string) {
			col := ui.Column(c).Key(key).Grow(1).Basis(0).MinWidth(0)
			if picked {
				col.Background(tint.Alpha(.10))
			}
			col.Children(func() {
				ui.Row(c).Height(22).PaddingX(10).Gap(6).AlignItems(ui.Center).Background(tint.Alpha(.12)).Children(func() {
					ui.Text(c, title).FontSize(theme.FsCap).FontWeight(600).TextColor(tint)
					ui.Text(c, tag).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().MinWidth(0)
				})
				ui.Column(c).Padding(6, 10).Children(func() {
					if len(lines) == 0 {
						ui.Text(c, "(空)").FontSize(theme.FsCap).TextColor(t.Tx4)
					}
					for _, l := range lines {
						txt := l
						if txt == "" {
							txt = " "
						}
						ui.Text(c, txt).Font("monospace").FontSize(12).LineHeight(1.6).TextColor(t.Tx1)
					}
				})
			})
		}
		layout := ui.Row(c)
		if narrow {
			layout = ui.Column(c)
		}
		layout.Children(func() {
			side("当前", g.curTag, g.cur, t.Info, ch == chCur || ch == chBoth, "cf-c"+itoa(k))
			side("传入", g.incTag, g.inc, t.Ok, ch == chInc || ch == chBoth, "cf-i"+itoa(k))
		})
		ui.Row(c).Height(34).PaddingX(8).Gap(6).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Children(func() {
			for _, o := range []struct {
				label string
				ch    int
			}{{"接受当前", chCur}, {"接受传入", chInc}, {"接受双方", chBoth}} {
				v := kit.BtnDefault
				if ch == o.ch {
					v = kit.BtnSoft
				}
				if kit.Btn(c, o.label, kit.BtnOpt{Small: true, Variant: v}).Key("cfa-" + itoa(k) + "-" + itoa(o.ch)).Clicked() {
					s.choice[k] = o.ch
				}
			}
			if ch != chNone {
				ui.Spacer(c).Grow(1)
				if kit.Btn(c, "撤销选择", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).Clicked() {
					s.choice[k] = chNone
				}
			}
		})
	})
}

// conflictResult 是合并结果预览(未处理的冲突仍显示标记,行号随之变化)。
func (a *App) conflictResult(c *ui.Context, s *confState) {
	t := a.tk
	text := composeConflicts(s.segs, s.choice)
	ui.Column(c).Radius(t.RMd()).Border(1, t.Bd2).Children(func() {
		ui.Row(c).Height(26).PaddingX(10).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
			ui.Text(c, "结果").FontSize(theme.FsCap).FontWeight(600).TextColor(t.Tx3)
		})
		ui.Column(c).Padding(6, 10).Children(func() {
			for i, l := range strings.Split(text, "\n") {
				col := t.Tx1
				bg := ui.Transparent
				if strings.HasPrefix(l, "<<<<<<<") || strings.HasPrefix(l, "=======") || strings.HasPrefix(l, ">>>>>>>") {
					col, bg = t.Warn, t.Warn.Alpha(.10)
				}
				ui.Row(c).Key("cr-" + itoa(i)).Background(bg).Gap(10).Children(func() {
					ui.Text(c, itoa(i+1)).Font("monospace").FontSize(11).TextColor(t.Tx4).Width(28).Justify(ui.End)
					txt := l
					if txt == "" {
						txt = " "
					}
					ui.Text(c, txt).Font("monospace").FontSize(12).LineHeight(1.6).TextColor(col)
				})
			}
		})
	})
}

// demoConflict 演示用:把 src/server.rs 写成带冲突标记的内容并打开冲突标签。
func (a *App) demoConflict() {
	const p = "src/server.rs"
	const src = `use axum::Router;

pub fn build() -> Router {
    let app = Router::new();
<<<<<<< HEAD (当前:main)
    let app = app.route("/health", get(health));
    let limit = 64 * 1024;
=======
    let app = app.route("/healthz", get(health));
    let limit = 256 * 1024;
>>>>>>> feat/auth (传入)
    let app = app.layer(trace());
<<<<<<< HEAD (当前:main)
    app.with_state(State::default())
=======
    app.with_state(State::new(auth()))
>>>>>>> feat/auth (传入)
}
`
	if err := a.ide.fs.Write(p, src); err == nil {
		a.ide.invalidate()
		delete(a.cf, p)
		a.OpenWorkspaceTab(vConflict, p)
	}
}
