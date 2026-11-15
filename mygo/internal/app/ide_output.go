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
	ui.Row(c).Height(26).Shrink(0).PaddingX(10).Gap(6).AlignItems(ui.Center).Children(func() {
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
	ui.Scroll(c).Key("out-scroll").TrackScroll(&o.st).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(2, 12).Children(func() {
			for i, ln := range lines {
				col := t.Tx2
				switch {
				case strings.Contains(ln, "ERROR") || strings.Contains(ln, "失败"):
					col = t.Err
				case strings.Contains(ln, "WARN"):
					col = t.Warn
				}
				ui.Text(c, ln).Key("o" + itoa(i)).Font("monospace").FontSize(11.5).LineHeight(1.5).TextColor(col).SingleLine()
			}
		})
	})
	if len(lines) != o.last && (atEnd || o.last == 0) {
		o.st.Y = 1 << 30
	}
	o.last = len(lines)
}
