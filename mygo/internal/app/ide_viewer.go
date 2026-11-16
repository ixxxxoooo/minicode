package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/tabular"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// 非文本文件的查看器(设计稿 ide-md 页「非文本文件 · 图片与表格」):
// 图片 → 图片查看器(适应 / 缩放);csv · tsv · xlsx → 只读表格查看器(虚拟滚动,最多 5000 行 × 100 列);
// 其余二进制 → 提示 + 「在访达中显示 / 用默认应用打开」两个出口。读盘与解码都在后台,界面线程只查结果。

var viewLog = diag.L("viewer")

const (
	viewImageMax = 64 << 20 // 图片超过这个大小不预览
	viewXlsxMax  = 64 << 20
	viewCSVMax   = 32 << 20 // csv 只读开头这一段
)

// viewerKind 按扩展名决定用哪种查看器,空 = 当文本打开。
func viewerKind(p string) string {
	switch strings.ToLower(path.Ext(p)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tif", ".tiff", ".svg":
		return "image"
	case ".csv", ".tsv", ".xlsx":
		return "table"
	}
	return ""
}

// binaryHint 是读到二进制内容时的说明;旧版表格格式单独指路。
func binaryHint(p string) string {
	switch strings.ToLower(path.Ext(p)) {
	case ".xls", ".ods", ".numbers":
		return "暂不支持直接查看 " + path.Ext(p) + " 文件,请先另存为 .xlsx 或 .csv"
	}
	return "这是二进制文件,不能用文本方式打开"
}

func friendlyReadErr(err error) string {
	if errors.Is(err, workspace.ErrTooLarge) {
		return "文件太大(超过 8 MB),不适合在编辑器里打开"
	}
	return err.Error()
}

// humanBytes 把字节数写成「38 KB」「2.4 MB」。
func humanBytes(n int64) string {
	switch {
	case n < 1<<10:
		return fmt.Sprintf("%d B", n)
	case n < 1<<20:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	case n < 1<<30:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
}

// viewerState 是一个查看器标签的状态。
type viewerState struct {
	loading, loaded bool
	err             string
	info            string // 顶部信息:「PNG · 512 × 512 · 38 KB」
	note            string // 附注:编码 / 截断

	// 图片
	img    ui.ImageSource
	iw, ih float32
	zoom   float32 // 0 = 适应窗口
	bw, bh float32 // 上一帧内容区尺寸(算适应比例用)

	// 表格
	wb       *tabular.Workbook
	sheet    int
	noHeader bool // 首行不当表头
	tv       *tableView
}

// tableView 是某张表在某种表头设置下的列定义与列表状态(切换时重建)。
type tableView struct {
	sheet  int
	header bool
	start  int // 数据从第几行开始(首行作表头时为 1)
	cols   []ui.TableColumn
	list   *ui.ListState
}

type viewerResult struct {
	err, info, note string
	img             ui.ImageSource
	iw, ih          float32
	wb              *tabular.Workbook
}

// readViewerBytes 读文件;真实文件走磁盘,演示(内存文件系统)退回 fs.Read。
// 超过 limit 时:partial=true 只取开头 limit 字节,否则返回「太大」。
func readViewerBytes(fs workspace.FS, p string, limit int64, partial bool) (data []byte, size int64, err error) {
	if f, oerr := os.Open(fs.Abs(p)); oerr == nil {
		defer f.Close()
		if st, serr := f.Stat(); serr == nil {
			size = st.Size()
		}
		if size > limit && !partial {
			return nil, size, fmt.Errorf("文件太大(%s),超过 %s 的上限,不在编辑器里预览", humanBytes(size), humanBytes(limit))
		}
		data, err = io.ReadAll(io.LimitReader(f, limit))
		return data, size, err
	}
	s, rerr := fs.Read(p)
	if rerr != nil {
		return nil, 0, rerr
	}
	return []byte(s), int64(len(s)), nil
}

func loadViewer(fs workspace.FS, p, kind string) viewerResult {
	ext := strings.ToLower(path.Ext(p))
	var res viewerResult
	switch kind {
	case "image":
		data, size, err := readViewerBytes(fs, p, viewImageMax, false)
		if err != nil {
			res.err = "无法打开:" + err.Error()
			return res
		}
		img, err := ui.DecodeImage(data)
		if err != nil || img == nil {
			res.err = "图片无法解码(文件损坏或格式不受支持)"
			return res
		}
		w, h := ui.ImageSize(img)
		res.img, res.iw, res.ih = img, w, h
		res.info = fmt.Sprintf("%s · %d × %d · %s", strings.ToUpper(strings.TrimPrefix(ext, ".")), int(w+.5), int(h+.5), humanBytes(size))
	case "table":
		var wb *tabular.Workbook
		var size int64
		if ext == ".xlsx" {
			data, sz, err := readViewerBytes(fs, p, viewXlsxMax, false)
			if err != nil {
				res.err = "无法打开:" + err.Error()
				return res
			}
			size = sz
			wb, err = tabular.ParseXLSX(data)
			if err != nil {
				res.err = "无法解析表格:" + err.Error()
				return res
			}
		} else {
			data, sz, err := readViewerBytes(fs, p, viewCSVMax, true)
			if err != nil {
				res.err = "无法打开:" + err.Error()
				return res
			}
			size = sz
			wb, err = tabular.ParseCSV(data, ext, sz > int64(len(data)))
			if err != nil {
				res.err = "无法解析表格:" + err.Error()
				return res
			}
		}
		res.wb, res.note = wb, wb.Note
		res.info = strings.ToUpper(strings.TrimPrefix(ext, ".")) + " · " + humanBytes(size)
	}
	return res
}

// viewerLoad 第一次显示时在后台载入;完成后经 a.bg 回到界面线程。
func (a *App) viewerLoad(tb *ideTab) {
	v := tb.vw
	if v == nil || v.loading || v.loaded {
		return
	}
	v.loading = true
	fs, p, kind := a.ide.fs, tb.path, tb.viewer
	viewLog.Info("打开文件(查看器)", "path", p, "kind", kind)
	a.bg("viewer-load", func() func() {
		start := time.Now()
		end := diag.SpanSlow("viewer", "载入"+kind, 300*time.Millisecond, "path", p)
		res := loadViewer(fs, p, kind)
		end()
		if res.err != "" {
			viewLog.Warn("查看器载入失败", "path", p, "kind", kind, "err", res.err)
		} else {
			viewLog.Info("查看器载入完成", "path", p, "kind", kind, "耗时", time.Since(start))
		}
		return func() {
			v.loading, v.loaded = false, true
			v.err, v.info, v.note = res.err, res.info, res.note
			v.img, v.iw, v.ih, v.wb = res.img, res.iw, res.ih, res.wb
		}
	})
}

// viewerTab 是查看器标签的内容(顶部面包屑由调用方先画)。
func (a *App) viewerTab(c *ui.Context, tb *ideTab) {
	a.viewerLoad(tb)
	v, t := tb.vw, a.tk
	switch {
	case v == nil:
	case v.err != "":
		a.viewerFail(c, tb, v.err)
	case !v.loaded:
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(8).Children(func() {
			kit.Spin(c, t.Tx3)
			ui.Text(c, "正在载入…").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
	case tb.viewer == "image":
		a.imageView(c, tb)
	default:
		a.tableView(c, tb)
	}
}

// viewerAsText 把查看器标签改成普通文本标签(csv / svg 都是文本,用户可能要直接编辑)。
func (a *App) viewerAsText(tb *ideTab) {
	viewLog.Info("改用文本方式打开", "path", tb.path, "kind", tb.viewer)
	tb.viewer, tb.vw = "", nil
	a.ide.loadText(tb)
	if a.ide.onDiskSnap != nil && tb.buf != nil {
		a.ide.onDiskSnap(tb)
	}
}

func (a *App) openDefault(rel string) {
	abs := a.absPath(rel)
	viewLog.Info("用默认应用打开", "path", abs)
	if testing.Testing() {
		return
	}
	if err := exec.Command("open", abs).Start(); err != nil {
		viewLog.Warn("默认应用打开失败", "path", abs, "err", err)
		a.toast = "无法用默认应用打开:" + err.Error()
	}
}

// errView 是读不出来 / 太大 / 二进制文件的提示,带两个出口(原来只有一句灰字)。
func (a *App) errView(c *ui.Context, tb *ideTab) { a.viewerFail(c, tb, tb.err) }

func (a *App) viewerFail(c *ui.Context, tb *ideTab, msg string) {
	t := a.tk
	demo := a.proj == ""
	tip := ""
	if demo {
		tip = "演示模式没有真实文件"
	}
	ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(10).Children(func() {
		kit.Ic(c, "file", kit.S20).TextColor(t.Tx4)
		ui.Text(c, msg).TextColor(t.Tx3)
		ui.Row(c).Gap(8).Children(func() {
			kit.Btn(c, "在访达中显示", kit.BtnOpt{Small: true, Icon: "folder", Disabled: demo, Tip: tip}).OnClick(func() { a.revealInFinder(a.absPath(tb.path)) })
			kit.Btn(c, "用默认应用打开", kit.BtnOpt{Small: true, Icon: "ext", Disabled: demo, Tip: tip}).OnClick(func() { a.openDefault(tb.path) })
		})
	})
}

// viewerBar 是查看器顶部的工具条:左信息、右操作。
func (a *App) viewerBar(c *ui.Context, left, right func()) {
	t := a.tk
	ui.Row(c).Height(30).Shrink(0).PaddingX(10).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		left()
		ui.Box(c).Grow(1)
		right()
	})
}

// ───────── 图片 ─────────

func (v *viewerState) fitScale() float32 {
	if v.iw <= 0 || v.ih <= 0 || v.bw <= 0 || v.bh <= 0 {
		return 1
	}
	s := (v.bw - 48) / v.iw
	if h := (v.bh - 48) / v.ih; h < s {
		s = h
	}
	if s > 1 {
		s = 1
	}
	if s < .02 {
		s = .02
	}
	return s
}

func (v *viewerState) effScale() float32 {
	if v.zoom > 0 {
		return v.zoom
	}
	return v.fitScale()
}

func (v *viewerState) zoomBy(f float32) {
	z := v.effScale() * f
	if z < .05 {
		z = .05
	}
	if z > 16 {
		z = 16
	}
	v.zoom = z
}

func (a *App) imageView(c *ui.Context, tb *ideTab) {
	v, t := tb.vw, a.tk
	eff := v.effScale()
	a.viewerBar(c, func() {
		ui.Text(c, v.info).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
	}, func() {
		kit.Ib(c, "minus", kit.IbOpt{Small: true, Tip: "缩小", Disabled: eff <= .05}).OnClick(func() { v.zoomBy(1 / 1.25) })
		ui.Text(c, fmt.Sprintf("%d%%", int(eff*100+.5))).FontSize(theme.FsCap).TextColor(t.Tx3)
		kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "放大", Disabled: eff >= 16}).OnClick(func() { v.zoomBy(1.25) })
		if v.zoom == 0 {
			kit.Btn(c, "100%", kit.BtnOpt{Small: true, Tip: "按原始大小显示"}).OnClick(func() { v.zoom = 1 })
		} else {
			kit.Btn(c, "适应窗口", kit.BtnOpt{Small: true}).OnClick(func() { v.zoom = 0 })
		}
		if strings.EqualFold(path.Ext(tb.path), ".svg") {
			kit.Btn(c, "以文本打开", kit.BtnOpt{Small: true, Icon: "fileText"}).OnClick(func() { a.viewerAsText(tb) })
		}
		kit.Ib(c, "folder", kit.IbOpt{Small: true, Tip: "在访达中显示", Disabled: a.proj == ""}).OnClick(func() { a.revealInFinder(a.absPath(tb.path)) })
	})
	body := ui.Column(c).Key("imgbody").Grow(1).MinHeight(0).MinWidth(0).Background(t.Fill1)
	if b := body.Bounds(); b.W > 0 {
		v.bw, v.bh = b.W, b.H
	} else {
		a.frameSoon() // 第一帧还没有尺寸:下一帧再按窗口大小适应
	}
	w, h := v.iw*eff, v.ih*eff
	body.Children(func() {
		if v.zoom == 0 {
			ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Image(c, v.img).Size(w, h)
			})
			return
		}
		ui.Scroll(c).Key("imgscroll-" + tb.id).Grow(1).MinHeight(0).Children(func() {
			ui.ScrollHorizontal(c).Key("imgscrollx-" + tb.id).Children(func() {
				ui.Column(c).Padding(24).Children(func() {
					ui.Image(c, v.img).Size(w, h)
				})
			})
		})
	})
}

// ───────── 表格 ─────────

// textWidthUnits 估算文本宽度:中日韩字符算 2,其余算 1。
func textWidthUnits(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x2E80 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

func clipRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// buildTableView 按当前表与表头设置算列定义:列宽取前 200 行的最宽内容(60–320px)。
func buildTableView(sh *tabular.Sheet, sheet int, header bool) *tableView {
	tv := &tableView{sheet: sheet, header: header, list: &ui.ListState{}}
	if header && len(sh.Rows) > 0 {
		tv.start = 1
	}
	tv.cols = append(tv.cols, ui.TableColumn{Title: "#", ID: "#", Width: 52, Fixed: true, Align: ui.End})
	sample := len(sh.Rows)
	if sample > 200 {
		sample = 200
	}
	for j := 0; j < sh.Cols; j++ {
		title := tabular.ColName(j)
		if tv.start == 1 {
			if h := strings.TrimSpace(sh.Cell(0, j)); h != "" {
				title = clipRunes(h, 40)
			}
		}
		u := textWidthUnits(title)
		for i := 0; i < sample; i++ {
			if n := textWidthUnits(sh.Cell(i, j)); n > u {
				u = n
			}
		}
		w := float32(u)*7 + 28
		if w < 60 {
			w = 60
		}
		if w > 320 {
			w = 320
		}
		tv.cols = append(tv.cols, ui.TableColumn{Title: title, ID: "c" + strconv.Itoa(j), Width: w, MinWidth: 40, MaxWidth: 800})
	}
	return tv
}

func (a *App) tableView(c *ui.Context, tb *ideTab) {
	v, t := tb.vw, a.tk
	wb := v.wb
	if wb == nil || len(wb.Sheets) == 0 {
		a.viewerFail(c, tb, "这个文件里没有可显示的内容")
		return
	}
	if v.sheet < 0 || v.sheet >= len(wb.Sheets) {
		v.sheet = 0
	}
	sh := &wb.Sheets[v.sheet]
	if v.tv == nil || v.tv.sheet != v.sheet || v.tv.header != !v.noHeader {
		v.tv = buildTableView(sh, v.sheet, !v.noHeader)
	}
	tv := v.tv
	n := len(sh.Rows) - tv.start
	if n < 0 {
		n = 0
	}
	a.viewerBar(c, func() {
		if k := len(wb.Sheets); k > 1 {
			if k <= 5 {
				items := make([]kit.SegItem, k)
				for i := range wb.Sheets {
					items[i] = kit.SegItem{Text: clipRunes(wb.Sheets[i].Name, 12)}
				}
				if i := kit.Seg(c, items, v.sheet, kit.SegOpt{Key: "tbl-sheets"}); i >= 0 && i != v.sheet {
					v.sheet = i
				}
			} else {
				kit.Btn(c, clipRunes(sh.Name, 16)+"  ▾", kit.BtnOpt{Small: true, Tip: "切换工作表"}).Menu(func(m *ui.Menu) {
					for i := range wb.Sheets {
						if menuItem(m, wb.Sheets[i].Name).Chosen() {
							v.sheet = i
						}
					}
				})
			}
		}
		stat := fmt.Sprintf("%d 行 × %d 列", len(sh.Rows), sh.Cols)
		if sh.Truncated {
			stat = fmt.Sprintf("仅显示前 %d 行 × %d 列", tabular.MaxRows, tabular.MaxCols)
		}
		ui.Text(c, stat+" · "+v.info).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
		if v.note != "" {
			ui.Text(c, "· "+v.note).FontSize(theme.FsCap).TextColor(t.Warn).SingleLine()
		}
	}, func() {
		ui.Text(c, "首行作表头").FontSize(theme.FsCap).TextColor(t.Tx3)
		if kit.Tog(c, !v.noHeader, "tbl-hdr") {
			v.noHeader = !v.noHeader
		}
		if ext := strings.ToLower(path.Ext(tb.path)); ext == ".csv" || ext == ".tsv" {
			kit.Btn(c, "以文本打开", kit.BtnOpt{Small: true, Icon: "fileText", Tip: "改用文本编辑器打开,可以编辑与保存"}).OnClick(func() { a.viewerAsText(tb) })
		}
		kit.Ib(c, "folder", kit.IbOpt{Small: true, Tip: "在访达中显示", Disabled: a.proj == ""}).OnClick(func() { a.revealInFinder(a.absPath(tb.path)) })
	})
	if n == 0 {
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "这张表是空的").TextColor(t.Tx3)
		})
		return
	}
	ui.Table(c, tv.list, tv.cols, n, func(row, col int) {
		if col == 0 {
			ui.Text(c, strconv.Itoa(row+tv.start+1)).FontSize(theme.FsCap).TextColor(t.Tx4).SingleLine()
			return
		}
		ui.Text(c, sh.Cell(row+tv.start, col-1)).FontSize(12).TextColor(t.Tx1).SingleLine()
	}).Key("tbl-" + tb.id).Grow(1).MinHeight(0)
}
