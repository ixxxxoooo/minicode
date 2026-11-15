package app

import (
	"errors"
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

const (
	extmodPollFast = time.Second
	extmodPollSlow = 5 * time.Second
)

// diskBacked 是否值得对磁盘做 Stat(演示内存、Multi 里含 OS 根时仍轮询)。
func diskBacked(fs workspace.FS) bool {
	switch fs.(type) {
	case *workspace.OS, *workspace.Multi:
		return true
	default:
		return false
	}
}

// extmodParticipates 差异 / 虚拟 / 无缓冲区 / 无路径的标签不参与轮询。
func extmodParticipates(tb *ideTab) bool {
	if tb == nil || tb.diff || tb.virt != "" || tb.buf == nil || tb.path == "" || tb.err != "" {
		return false
	}
	return true
}

func (a *App) recordDiskSnap(tb *ideTab) {
	if tb == nil || !extmodParticipates(tb) || !diskBacked(a.ide.fs) {
		return
	}
	st, err := os.Stat(a.ide.fs.Abs(tb.path))
	if err != nil {
		return
	}
	a.setDiskSnap(tb.path, diskSnap{mod: st.ModTime(), size: st.Size()})
}

func (a *App) setDiskSnap(p string, s diskSnap) {
	d := a.ide
	if d.diskSnaps == nil {
		d.diskSnaps = map[string]diskSnap{}
	}
	d.diskSnaps[p] = s
}

func (a *App) extmodTick() {
	if a.win == nil || a.mode != ModeIDE || a.screen != ScreenWork {
		return
	}
	if !diskBacked(a.ide.fs) {
		return
	}
	focused := a.win.IsFocused()
	d := a.ide
	if focused && !extHubFocus {
		d.extmodLast = time.Time{}
	}
	extHubFocus = focused
	now := a.now()
	interval := extmodPollSlow
	if focused {
		interval = extmodPollFast
	}
	if !d.extmodLast.IsZero() && now.Sub(d.extmodLast) < interval {
		return
	}
	d.extmodLast = now
	paths := a.extmodPaths()
	if len(paths) == 0 {
		return
	}
	for p := range paths {
		if _, ok := d.diskSnaps[p]; !ok {
			if tb := a.ide.tabByPath(p); tb != nil {
				a.recordDiskSnap(tb)
			}
		}
	}
	a.bg("extmod-poll", func() func() {
		type hit struct {
			path    string
			snap    diskSnap
			dirty   bool
			deleted bool
		}
		var hits []hit
		for p := range paths {
			tb := a.ide.tabByPath(p)
			if tb == nil || !extmodParticipates(tb) {
				continue
			}
			abs := a.ide.fs.Abs(p)
			st, err := os.Stat(abs)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					if _, ok := d.diskSnaps[p]; ok && !extFlags(p).orphan {
						hits = append(hits, hit{path: p, dirty: a.anyDirtyForPath(p), deleted: true})
					}
				}
				continue
			}
			cur := diskSnap{mod: st.ModTime(), size: st.Size()}
			prev, ok := d.diskSnaps[p]
			if !ok {
				continue
			}
			if cur.mod.Equal(prev.mod) && cur.size == prev.size {
				continue
			}
			if extAlerted(p, cur) {
				continue
			}
			hits = append(hits, hit{path: p, snap: cur, dirty: a.anyDirtyForPath(p)})
		}
		return func() {
			for _, h := range hits {
				if tb := a.ide.tabByPath(h.path); tb != nil {
					if h.deleted {
						a.applyExtmodDeleted(tb, h.dirty)
					} else {
						a.applyExtmod(tb, h.snap, h.dirty)
					}
				}
			}
		}
	})
}

// extmodPaths 返回当前应轮询的唯一文件路径(多拆分共享缓冲区只算一次)。
func (a *App) extmodPaths() map[string]bool {
	out := map[string]bool{}
	for _, tb := range a.allFileTabs() {
		if extmodParticipates(tb) {
			out[tb.path] = true
		}
	}
	return out
}

func (a *App) allFileTabs() []*ideTab {
	d := a.ide
	out := append([]*ideTab(nil), d.tabs...)
	out = append(out, d.tabs2...)
	return out
}

func (a *App) tabsForPath(p string) []*ideTab {
	var out []*ideTab
	for _, tb := range a.allFileTabs() {
		if tb != nil && tb.path == p && !tb.diff && tb.virt == "" {
			out = append(out, tb)
		}
	}
	return out
}

func (a *App) anyDirtyForPath(p string) bool {
	for _, tb := range a.tabsForPath(p) {
		if tb.buf != nil && tb.dirty() {
			return true
		}
	}
	return false
}

func (a *App) markExtmodAlerted(p string, s diskSnap) {
	extMarkAlerted(p, s)
}

func (a *App) setExtBanner(p string, on bool) {
	for _, tb := range a.tabsForPath(p) {
		tb.extBanner = on
	}
}

func (a *App) setExtDeleted(p string, on bool) {
	f := extFlags(p)
	f.deleted = on
	if on {
		f.orphan = false
		extSetFlags(p, f)
		for _, tb := range a.tabsForPath(p) {
			tb.extBanner = false
		}
		return
	}
	extSetFlags(p, f)
}

func (a *App) applyExtmod(tb *ideTab, snap diskSnap, dirty bool) {
	name := path.Base(tb.path)
	if !dirty {
		src, err := a.ide.fs.Read(tb.path)
		if err != nil {
			extmodLog.Warn("外部修改后重载失败", "path", tb.path, "err", err)
			return
		}
		for _, t := range a.tabsForPath(tb.path) {
			if t.buf != nil {
				setKeepCursor(t, src)
				t.saved = src
				t.buf.MarkSaved()
				t.extBanner = false
				extResetFlags(tb.path)
			}
		}
		a.setDiskSnap(tb.path, snap)
		a.markExtmodAlerted(tb.path, snap)
		a.toast = "已从磁盘重新载入 " + name
		extmodLog.Info("外部修改自动重载", "path", tb.path)
		return
	}
	a.setExtBanner(tb.path, true)
	a.setDiskSnap(tb.path, snap)
	a.markExtmodAlerted(tb.path, snap)
	extmodLog.Warn("外部修改与本地未保存冲突", "path", tb.path)
}

func (a *App) applyExtmodDeleted(tb *ideTab, dirty bool) {
	name := path.Base(tb.path)
	a.setExtDeleted(tb.path, true)
	a.markExtmodAlerted(tb.path, diskSnap{})
	extmodLog.Warn("磁盘文件已删除", "path", tb.path, "dirty", dirty)
	_ = name
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
	for _, t := range a.tabsForPath(tb.path) {
		if t.buf == nil {
			continue
		}
		setKeepCursor(t, src)
		t.saved = src
		t.buf.MarkSaved()
		t.extBanner = false
		extResetFlags(tb.path)
	}
	a.recordDiskSnap(tb)
	a.ide.invalidate()
	if user {
		name := path.Base(tb.path)
		a.toast = "已从磁盘重新载入 " + name
	}
	extmodLog.Info("用户重新载入", "path", tb.path)
}

func (a *App) keepMineExtmod(tb *ideTab) {
	if tb == nil {
		return
	}
	for _, t := range a.tabsForPath(tb.path) {
		t.extBanner = false
	}
	f := extFlags(tb.path)
	f.keep = true
	extSetFlags(tb.path, f)
	a.recordDiskSnap(tb)
	extmodLog.Info("保留本地版本", "path", tb.path)
}

func (a *App) keepOrphanExtmod(tb *ideTab) {
	if tb == nil {
		return
	}
	for _, t := range a.tabsForPath(tb.path) {
		t.extBanner = false
	}
	f := extFlags(tb.path)
	f.deleted, f.orphan = false, true
	extSetFlags(tb.path, f)
	extClearAlerted(tb.path)
	extmodLog.Info("删除后保留在编辑器", "path", tb.path)
}

func (a *App) openExtmodDiff(tb *ideTab) {
	if tb == nil || tb.buf == nil {
		return
	}
	disk, err := a.ide.fs.Read(tb.path)
	if err != nil {
		disk = ""
	}
	buf := tb.buf.String()
	id := "diff:extmod:" + tb.path
	if t := a.ide.tab(id); t != nil {
		a.ide.active, a.ide.sel = id, tb.path
		return
	}
	a.ide.addDiffTab(id, tb.path, "磁盘 ↔ 编辑器", disk, buf, true)
	if t := a.ide.tab(id); t != nil && t.dv != nil {
		t.dv.Editable = false
	}
	a.ide.active = id
	extmodLog.Info("打开外部修改差异", "path", tb.path)
}

// extmodBanner 磁盘变化时在编辑器顶部显示(修改冲突或文件已删除)。
func (a *App) extmodBanner(c *ui.Context, tb *ideTab) {
	if tb == nil {
		return
	}
	t := a.tk
	name := path.Base(tb.path)
	ef := extFlags(tb.path)
	if ef.deleted && !ef.orphan {
		ui.Row(c).Shrink(0).Gap(8).Padding(6, 12).Background(t.ErrSoft).BorderWidth(0, 0, 1, 0).BorderColor(t.Err).
			AlignItems(ui.Center).FontSize(theme.FsCap).Children(func() {
			kit.Ic(c, "errc", kit.S12).TextColor(t.Err)
			ui.Text(c, name+" 已在磁盘上被删除").TextColor(t.Err).Grow(1)
			kit.Btn(c, "保留在编辑器", kit.BtnOpt{Small: true}).OnClick(func() { a.keepOrphanExtmod(tb) })
			kit.Btn(c, "关闭标签", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.closeExtmodTabs(tb.path) })
		})
		return
	}
	if !tb.extBanner {
		return
	}
	ui.Row(c).Shrink(0).Gap(8).Padding(6, 12).Background(t.WarnSoft).BorderWidth(0, 0, 1, 0).BorderColor(t.Warn).
		AlignItems(ui.Center).FontSize(theme.FsCap).Children(func() {
		kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
		ui.Text(c, "磁盘上的 "+name+" 已被其它程序修改").TextColor(t.Warn).Grow(1)
		kit.Btn(c, "重新载入", kit.BtnOpt{Small: true}).OnClick(func() { a.reloadTabFromDisk(tb, true) })
		kit.Btn(c, "保留我的", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.keepMineExtmod(tb) })
		kit.Btn(c, "查看差异", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.openExtmodDiff(tb) })
	})
}

func (a *App) closeExtmodTabs(p string) {
	ids := []string{}
	for _, tb := range a.allFileTabs() {
		if tb != nil && tb.path == p {
			ids = append(ids, tb.id)
		}
	}
	for _, id := range ids {
		a.ide.closeTab(id)
	}
	extmodLog.Info("关闭已删除文件的标签", "path", p)
}

// extmodSaveIntercept 在 ⌘S / 保存命令前调用;需要确认时弹框并返回 true(中止同步保存)。
func (a *App) extmodSaveIntercept() bool {
	tb := a.ide.cur()
	if tb == nil || tb.diff || tb.virt != "" || tb.buf == nil {
		return false
	}
	name := path.Base(tb.path)
	ef := extFlags(tb.path)
	if ef.orphan {
		a.confirmDlg("保存 "+name+"?",
			"文件已在磁盘上被删除;保存将按当前路径重新创建该文件。",
			"保存", false, func() {
				ef.orphan = false
				extSetFlags(tb.path, ef)
				if err := a.ide.save(); err != nil {
					a.toast = "保存失败:" + err.Error()
				}
			})
		return true
	}
	if !ef.keep {
		return false
	}
	a.confirmDlg("用编辑器里的内容覆盖磁盘上的 "+name+"?",
		"外部版本将被替换;可在源代码管理或访达历史里找回(若仓库有提交)。",
		"覆盖保存", false, func() {
			ef.keep = false
			extSetFlags(tb.path, ef)
			if err := a.ide.save(); err != nil {
				a.toast = "保存失败:" + err.Error()
			}
		})
	return true
}

// dbgExtmod 演示 / 截图: -ed extmod:dirty|clean|deleted|auto
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
			a.ide.extmodLast = time.Time{}
			a.extmodTick()
			return
		}
		fallthrough
	case "dirty":
		for _, t := range a.tabsForPath(tb.path) {
			if t.buf != nil && t.saved == t.buf.String() {
				t.buf.Set(t.buf.String() + "\n// local edit")
			}
			t.extBanner = true
		}
		a.setExtDeleted(tb.path, false)
		a.markExtmodAlerted(tb.path, diskSnap{mod: time.Now(), size: 1})
	case "deleted":
		a.setExtDeleted(tb.path, true)
	default:
		tb.extBanner = true
	}
}

func (a *App) extmodLastForce() { a.ide.extmodLast = time.Time{} }
