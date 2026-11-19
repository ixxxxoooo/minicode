package app

import (
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 底部「输出」面板:按通道(下拉)分开显示各来源的文字——
// MiniCode(应用日志文件的尾部)、Git(本应用执行的 Git 操作)、语言服务(服务器启动 / 诊断 / 错误)。
// 对标 VS Code 的「输出」:只读、可清空、可复制、自动贴底,新内容到来时在底部才跟随。

const (
	outApp = "MiniCode"
	outGit = "Git"
	outLSP = "语言服务"
)

var outChannels = []string{outApp, outGit, outLSP}

type outState struct {
	ch     string
	lines  map[string][]string
	tail   []string // MiniCode 通道:日志尾部缓存
	read   time.Time
	reding bool
	st     ui.ScrollState
	last   int // 上次画的行数(变多时贴底)
}

const outMax = 500

// outAppend 往一个通道追加一行(带时间),最多留 outMax 行。
func (a *App) outAppend(ch, line string) {
	o := &a.out
	if o.lines == nil {
		o.lines = map[string][]string{}
	}
	ls := append(o.lines[ch], time.Now().Format("15:04:05")+"  "+line)
	if len(ls) > outMax {
		ls = ls[len(ls)-outMax:]
	}
	o.lines[ch] = ls
}

// outTail 读应用日志最后 n 行(只读文件尾部 64 KB)。
func outTail(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil
	}
	const win = 64 << 10
	off := max(0, st.Size()-win)
	buf := make([]byte, st.Size()-off)
	if _, err := f.ReadAt(buf, off); err != nil && len(buf) == 0 {
		return nil
	}
	ls := strings.Split(strings.TrimRight(string(buf), "\n"), "\n")
	if off > 0 && len(ls) > 0 {
		ls = ls[1:] // 第一行可能被截断
	}
	if len(ls) > n {
		ls = ls[len(ls)-n:]
	}
	return ls
}

// selectedString 把 Selectable 容器的选区取成纯文本:每段取被选中的字符,段间换行。
func selectedString(parts []ui.SelectedText) string {
	var out []string
	for _, p := range parts {
		r := []rune(p.Text)
		s, e := max(0, min(p.Start, len(r))), max(0, min(p.End, len(r)))
		if e > s {
			out = append(out, string(r[s:e]))
		}
	}
	return strings.Join(out, "\n")
}

func (a *App) outPanel(c *ui.Context) {
	t := a.tk
	o := &a.out
	if o.ch == "" {
		o.ch = outApp
	}
	if o.ch == outApp && !o.reding && time.Since(o.read) > 2*time.Second {
		o.reding, o.read = true, time.Now()
		a.bg("out.tail", func() func() {
			ls := outTail(diag.LogPath(), 300)
			return func() { o.tail, o.reding = ls, false }
		})
	}
	lines := o.lines[o.ch]
	if o.ch == outApp {
		lines = o.tail
	}
	ui.Row(c).Height(theme.Px(26)).Shrink(0).PaddingX(10).Gap(6).AlignItems(ui.Center).Children(func() {
		for _, ch := range outChannels {
			ch := ch
			kit.Chip(c, ch, kit.ChipOpt{On: o.ch == ch}).OnClick(func() { o.ch, o.last = ch, 0 })
		}
		ui.Spacer(c).Grow(1)
		ui.Text(c, itoa(len(lines))+" 行").FontSize(theme.FsMicro).TextColor(t.Tx4)
		if kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制全部"}).Clicked() {
			a.copyText = strings.Join(lines, "\n")
			a.toast = "已复制 " + o.ch + " 输出"
		}
		if o.ch != outApp && kit.Ib(c, "trash", kit.IbOpt{Small: true, Tip: "清空"}).Clicked() {
			o.lines[o.ch] = nil
		}
	})
	if len(lines) == 0 {
		ui.Column(c).Grow(1).Padding(6, 12).Children(func() {
			ui.Text(c, map[string]string{outApp: "还没有日志", outGit: "还没有 Git 操作", outLSP: "语言服务器尚未启动"}[o.ch]).TextColor(t.Tx4).FontSize(theme.FsCap)
		})
		return
	}
	atEnd := o.st.Y >= o.st.MaxY-24
	kit.VScroll(c).Key("out-scroll").TrackScroll(&o.st).Grow(1).MinHeight(0).Children(func() {
		// 整列文字共用一个选区:鼠标拖选 / 双击选词 / 三击选行 / ⇧+方向键扩选,⌘C 复制(多行以换行连接)。
		var sel ui.Element
		sel = ui.Column(c).Padding(2, 12).ClipX().Selectable()
		sel.Children(func() {
			// 日志行用 NoWrap 而不是 SingleLine:SingleLine 带 MaxLines=1,每换一个宽度整段都要重排;
			// 工作区从折叠展开的过渡里宽度从 ≈0 起逐帧变化,300 行长日志在窄宽度下每行 5~8 ms,
			// 一帧就卡 400~570 ms(切到 IDE 时底部停在「输出」就会发生)。NoWrap 的排版与宽度无关,换宽度直接复用。
			for i, ln := range lines {
				col := t.Tx2
				switch {
				case strings.Contains(ln, "ERROR") || strings.Contains(ln, "失败"):
					col = t.Err
				case strings.Contains(ln, "WARN"):
					col = t.Warn
				}
				ln := ln
				ui.Text(c, ln).Key("o" + itoa(i)).Font(theme.CodeFont()).FontSize(theme.Px(11.5)).LineHeight(1.5).TextColor(col).NoWrap().
					ContextMenu(func(m *ui.Menu) {
						// 行上的菜单盖过了选区容器自带的菜单,所以「复制所选」要自己补(没有选区时置灰并写明原因)。
						picked := selectedString(sel.SelectedTexts())
						if menuItem(m, "复制所选  ⌘C").Disabled(picked == "").Tooltip(map[bool]string{true: "先用鼠标拖选要复制的文字"}[picked == ""]).Chosen() {
							a.copyOut(picked, "所选文字")
						}
						m.Separator()
						a.outputMenu(m, ln, lines)
					})
			}
		})
	})
	if len(lines) != o.last && (atEnd || o.last == 0) {
		o.st.Y = 1 << 30
	}
	o.last = len(lines)
}
