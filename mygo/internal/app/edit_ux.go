package app

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 设计稿 ide-md / drag-more:拖拽引用到对话、Markdown 所见即所得、自动换行与格式化、各处「···」菜单。
// 这一批能力的界面状态与拖拽落点都放在这个文件。

var uxLog = diag.L("edit")

// uxState 汇总这一批编辑体验用到的界面状态。
type uxState struct {
	focusDraft    bool            // 下一帧把键盘焦点给输入框(拖入引用之后接着写)
	mdNoteOff     map[string]bool // 已点掉「写法会被规范化」提示条的文件
	demoHint      string          // 截图 / 调试:强制显示「可放」态的提示文字(ide_debug 的 -ed drag)
	rt            *richtext.Theme // 所见即所得的外观(换主题时清掉)
	blockCmtHint  string          // 块注释退化为行注释时的顶栏提示
	blockCmtUntil time.Time
}

// fileRef 是从文件树 / 编辑器标签 / 源代码管理里拖出来的一个文件或文件夹。
type fileRef struct {
	Path string // 相对项目根(多根工作区带文件夹名前缀)
	Dir  bool
}

// refText 把路径写成输入框里的 @ 引用;文件夹以 / 结尾。
func refText(p string, dir bool) string {
	if dir && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return "@" + p
}

// appendRefs 把引用插到输入框末尾:与已有文字之间补一个空格,已经引用过的不重复,末尾留一个空格接着写。
func appendRefs(draft string, refs ...string) string {
	var add []string
	for _, r := range refs {
		if r == "" || strings.Contains(" "+strings.ReplaceAll(draft, "\n", " ")+" ", " "+r+" ") {
			continue
		}
		dup := false
		for _, x := range add {
			dup = dup || x == r
		}
		if !dup {
			add = append(add, r)
		}
	}
	if len(add) == 0 {
		return draft
	}
	tail := strings.Join(add, " ") + " "
	switch {
	case draft == "", strings.HasSuffix(draft, " "), strings.HasSuffix(draft, "\n"):
		return draft + tail
	}
	return draft + " " + tail
}

// dropRefs 是落到对话栏的结果:「@路径」直接写进输入框正文(与打的字在一起),并把焦点交给它。
// 只有图片不一样(走输入框顶部的缩略图附件),不经过这里。
func (a *App) dropRefs(refs ...string) {
	next := appendRefs(a.draft, refs...)
	if next == a.draft {
		a.toast = "已经引用过了"
		return
	}
	a.draft = next
	a.ux.focusDraft = true
	uxLog.Info("拖入引用", "n", len(refs))
	a.toast = "已引用 " + itoa(len(refs)) + " 项到对话"
}

// dropPaths 处理从访达等其它应用拖进来的文件:项目内的写相对路径,项目外写绝对路径。
func (a *App) dropPaths(paths []string) {
	var refs []string
	for _, p := range paths {
		dir := false
		if strings.HasSuffix(p, "/") {
			dir = true
		}
		rel := p
		if a.proj != "" {
			if r, err := filepath.Rel(a.proj, p); err == nil && !strings.HasPrefix(r, "..") {
				rel = r
			}
		}
		refs = append(refs, refText(rel, dir))
	}
	a.dropRefs(refs...)
}

// chatDrop 把对话栏变成拖拽落点,返回此刻该显示的提示文字(空 = 没有拖拽经过)。
func (a *App) chatDrop(p ui.Element) string {
	if r, ok := ui.Drop[fileRef](p); ok {
		a.dropRefs(refText(r.Path, r.Dir))
	}
	if ps := p.DroppedFiles(); len(ps) > 0 {
		a.handleDroppedImages(ps)
	}
	if r, ok := ui.DragOver[fileRef](p); ok {
		return refText(r.Path, r.Dir)
	}
	if p.FileDragOver() {
		return "文件"
	}
	return a.ux.demoHint
}

// dropOverlay 是「可放」态:整栏强调色描边 + 中央提示。
func (a *App) dropOverlay(c *ui.Context, what string) {
	t := a.tk
	ui.Column(c).Key("dropzone").Absolute().Left(4).Right(4).Top(4).Bottom(4).PassThrough().Justify(ui.Center).AlignItems(ui.Center).
		Background(t.Ac.Alpha(.08)).Radius(t.RLg()).Border(2, t.Ac).Children(func() {
		pill := ui.Row(c).Padding(8, 14).Gap(8).AlignItems(ui.Center).Background(t.Raised).Radius(t.RMd()).Border(1, t.AcLine)
		for _, s := range t.Shadow2() {
			pill.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
		pill.Children(func() {
			kit.Ic(c, "link", kit.S12).TextColor(t.AcTx)
			ui.Text(c, "松手插入").FontWeight(500)
			ui.Text(c, what).Font("monospace").FontSize(theme.FsCap).TextColor(t.Tx2).SingleLine()
		})
	})
}

// dragRef 让一个元素成为文件引用的拖拽源(元素副本会跟着指针)。
func dragRef(e ui.Element, p string, dir bool) ui.Element {
	if p == "" {
		return e
	}
	return e.Drag(fileRef{Path: p, Dir: dir})
}
