package app

import (
	"os"
	"path"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

var extmodLog = diag.L("ide.extmod")

type diskSnap struct {
	mod  time.Time
	size int64
}

const extmodPoll = 2 * time.Second

// diskBacked 是否值得对磁盘做 Stat(演示内存、Multi 里含 OS 根时仍轮询)。
func diskBacked(fs workspace.FS) bool {
	switch fs.(type) {
	case *workspace.OS, *workspace.Multi:
		return true
	default:
		return false
	}
}

func (a *App) recordDiskSnap(tb *ideTab) {
	if tb == nil || tb.diff || tb.virt != "" || tb.buf == nil || !diskBacked(a.ide.fs) {
		return
	}
	st, err := os.Stat(a.ide.fs.Abs(tb.path))
	if err != nil {
		return
	}
	if a.ide.diskSnaps == nil {
		a.ide.diskSnaps = map[string]diskSnap{}
	}
	a.ide.diskSnaps[tb.path] = diskSnap{mod: st.ModTime(), size: st.Size()}
}

func (a *App) extmodTick() {
	if a.win == nil || !a.win.IsFocused() || a.mode != ModeIDE || a.screen != ScreenWork {
		return
	}
	if !diskBacked(a.ide.fs) {
		return
	}
	now := a.now()
	if !a.ide.extmodLast.IsZero() && now.Sub(a.ide.extmodLast) < extmodPoll {
		return
	}
	a.ide.extmodLast = now
	if len(a.ide.tabs) == 0 && len(a.ide.tabs2) == 0 {
		return
	}
	for _, tb := range a.allFileTabs() {
		if tb != nil && tb.buf != nil {
			if _, ok := a.ide.diskSnaps[tb.path]; !ok {
				a.recordDiskSnap(tb)
			}
		}
	}
	a.bg("extmod-poll", func() func() {
		type hit struct {
			tb    *ideTab
			snap  diskSnap
			dirty bool
		}
		var hits []hit
		for _, tb := range a.allFileTabs() {
			if tb == nil || tb.diff || tb.virt != "" || tb.buf == nil {
				continue
			}
			st, err := os.Stat(a.ide.fs.Abs(tb.path))
			if err != nil {
				continue
			}
			cur := diskSnap{mod: st.ModTime(), size: st.Size()}
			prev, ok := a.ide.diskSnaps[tb.path]
			if !ok {
				continue
			}
			if cur.mod.Equal(prev.mod) && cur.size == prev.size {
				continue
			}
			hits = append(hits, hit{tb: tb, snap: cur, dirty: tb.dirty()})
		}
		return func() {
			for _, h := range hits {
				a.applyExtmod(h.tb, h.snap, h.dirty)
			}
		}
	})
}

func (a *App) allFileTabs() []*ideTab {
	d := a.ide
	out := append([]*ideTab(nil), d.tabs...)
	out = append(out, d.tabs2...)
	return out
}

func (a *App) applyExtmod(tb *ideTab, snap diskSnap, dirty bool) {
	name := path.Base(tb.path)
	if !dirty {
		src, err := a.ide.fs.Read(tb.path)
		if err != nil {
			extmodLog.Warn("外部修改后重载失败", "path", tb.path, "err", err)
			return
		}
		setKeepCursor(tb, src)
		tb.saved = src
		tb.buf.MarkSaved()
		tb.extBanner = false
		a.ide.diskSnaps[tb.path] = snap
		a.noticeInfo("已重新载入", name+":磁盘上的文件已被外部修改,已自动同步。")
		extmodLog.Info("外部修改自动重载", "path", tb.path)
		return
	}
	tb.extBanner = true
	a.ide.diskSnaps[tb.path] = snap
	body := name + ":磁盘上的文件已被外部修改。"
	a.noticeWarn("文件已在外部修改", body, []noticeAction{
		{label: "重新载入", run: func() { a.reloadTabFromDisk(tb, true) }},
		{label: "保留我的", run: func() { tb.extBanner = false; a.recordDiskSnap(tb) }},
	})
	extmodLog.Warn("外部修改与本地未保存冲突", "path", tb.path)
}

// reloadTabFromDisk 丢弃缓冲区,读回磁盘(用户点「重新载入」或调试)。
func (a *App) reloadTabFromDisk(tb *ideTab, user bool) {
	if tb == nil || tb.buf == nil {
		return
	}
	src, err := a.ide.fs.Read(tb.path)
	if err != nil {
		a.toast = "重新载入失败:" + err.Error()
		return
	}
	setKeepCursor(tb, src)
	tb.saved = src
	tb.buf.MarkSaved()
	tb.extBanner = false
	a.recordDiskSnap(tb)
	a.ide.invalidate()
	if user {
		a.toast = "已从磁盘重新载入"
	}
	extmodLog.Info("用户重新载入", "path", tb.path)
}

// extmodBanner 有未保存修改且磁盘已变时在编辑器顶部显示。
func (a *App) extmodBanner(c *ui.Context, tb *ideTab) {
	if tb == nil || !tb.extBanner {
		return
	}
	t := a.tk
	ui.Row(c).Shrink(0).Gap(8).Padding(6, 12).Background(t.WarnSoft).BorderWidth(0, 0, 1, 0).BorderColor(t.Warn).
		AlignItems(ui.Center).FontSize(theme.FsCap).Children(func() {
		kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
		ui.Text(c, "文件已在磁盘上被修改").TextColor(t.Warn).Grow(1)
		kit.Btn(c, "重新载入", kit.BtnOpt{Small: true}).OnClick(func() { a.reloadTabFromDisk(tb, true) })
		kit.Btn(c, "保留我的", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
			tb.extBanner = false
			a.recordDiskSnap(tb)
		})
	})
}

// dbgExtmod 演示 / 截图:强制展示外部修改横幅(内存 FS 不调轮询)。
func (a *App) dbgExtmod(arg string) {
	tb := a.ide.cur()
	if tb == nil || tb.buf == nil {
		a.ide.openFile("src/router.rs", true)
		tb = a.ide.cur()
	}
	if tb == nil {
		return
	}
	switch arg {
	case "auto", "clean":
		if diskBacked(a.ide.fs) {
			a.recordDiskSnap(tb)
			_ = a.ide.fs.Write(tb.path, tb.saved+"\n// extmod demo\n")
			a.extmodTick()
			a.extmodLastForce()
			return
		}
		fallthrough
	default:
		tb.extBanner = true
		a.noticeWarn("文件已在外部修改", path.Base(tb.path)+":磁盘上的文件已被外部修改。", []noticeAction{
			{label: "重新载入", run: func() { a.reloadTabFromDisk(tb, true) }},
			{label: "保留我的", run: func() { tb.extBanner = false }},
		})
	}
}

func (a *App) extmodLastForce() { a.ide.extmodLast = time.Time{} }
