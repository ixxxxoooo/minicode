package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

var attachLog = diag.L("composer.attach")

var draftImgRE = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)

type draftAttach struct {
	Path string // 草稿里的原始路径(多为相对路径)
	Span [2]int // 在 draft 中的起止下标,用于移除
}

func parseDraftAttaches(draft string) []draftAttach {
	var out []draftAttach
	for _, m := range draftImgRE.FindAllStringSubmatchIndex(draft, -1) {
		if len(m) < 4 {
			continue
		}
		out = append(out, draftAttach{Path: draft[m[2]:m[3]], Span: [2]int{m[0], m[1]}})
	}
	return out
}

func splitDraftImages(draft string) (string, []string) {
	var imgs []string
	text := draftImgRE.ReplaceAllStringFunc(draft, func(m string) string {
		sub := draftImgRE.FindStringSubmatch(m)
		if len(sub) > 1 {
			imgs = append(imgs, strings.TrimSpace(sub[1]))
		}
		return ""
	})
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n\n\n", "\n\n"))
	return text, imgs
}

func (a *App) absAttach(rel string) string {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return ""
	}
	if filepath.IsAbs(rel) {
		return rel
	}
	if a.ide.fs != nil {
		return a.ide.fs.Abs(rel)
	}
	if a.proj != "" {
		return filepath.Join(a.proj, rel)
	}
	return rel
}

func (a *App) relRefForAttach(abs string) string {
	abs = filepath.Clean(abs)
	if a.proj != "" {
		if rel, err := filepath.Rel(a.proj, abs); err == nil && rel != "" && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return abs
}

// plusAttachMenu 「+」附件菜单。
func (a *App) plusAttachMenu(c *ui.Context, anchor ui.Element) {
	b := anchor
	a.popOpen("cmp-attach", func(open *bool) {
		ui.PopoverBase(c, b, open, func(panel ui.Element) {
			a.popPanel(panel, 200).AttachTo(b, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 6, 0)
			pick := func(key, icon, label string, fn func()) {
				if a.mi(c, key, icon, label, "", false).Clicked() {
					a.popup = ""
					fn()
				}
			}
			pick("img", "image", "添加图片…", a.pickAttachImages)
			pick("ref", "at", "添加文件引用…", a.pickAttachRefs)
		})
	})
}

func (a *App) pickAttachImages() {
	if a.win == nil || a.demoWs() {
		a.toast = "演示模式没有文件选择器"
		return
	}
	a.bg("pick-attach-img", func() func() {
		defer diag.SpanSlow("composer", "选择图片", time.Minute)()
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title: "选择图片", Multiple: true, ButtonLabel: "添加", Parent: a.win,
		})
		return func() {
			if err != nil {
				attachLog.Warn("图片选择器失败", "err", err)
				return
			}
			if len(paths) > 0 {
				a.handleDroppedImages(paths)
			}
		}
	})
}

func (a *App) pickAttachRefs() {
	if a.win == nil || a.demoWs() {
		a.toast = "演示模式没有文件选择器"
		return
	}
	a.bg("pick-attach-ref", func() func() {
		defer diag.SpanSlow("composer", "选择文件引用", time.Minute)()
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title: "选择文件", Multiple: true, ButtonLabel: "引用", Parent: a.win,
		})
		return func() {
			if err != nil {
				attachLog.Warn("文件选择器失败", "err", err)
				return
			}
			if len(paths) == 0 {
				return
			}
			var refs []string
			for _, p := range paths {
				refs = append(refs, refText(a.relRefForAttach(p), false))
			}
			a.draft = appendRefs(a.draft, refs...)
			a.ux.focusDraft = true
			attachLog.Info("添加文件引用", "n", len(refs))
		}
	})
}

// queueShowText 排队卡正文:去掉图片语法后的摘要,纯图时显示「(图片)」。
func queueShowText(draft string) string {
	body, _ := splitDraftImages(draft)
	body = strings.TrimSpace(body)
	if body != "" {
		return body
	}
	if len(parseDraftAttaches(draft)) > 0 {
		return "(图片)"
	}
	return draft
}

func (a *App) queueAttachChips(c *ui.Context, draft string) {
	atts := parseDraftAttaches(draft)
	if len(atts) == 0 {
		return
	}
	t := a.tk
	ui.Row(c).Gap(4).Shrink(0).Children(func() {
		for _, at := range atts {
			chip := ui.Row(c).Height(22).PaddingX(3).Gap(3).Radius(t.RXs()).AlignItems(ui.Center).Border(1, t.Bd2).Background(t.Fill1).Shrink(0)
			chip.Children(func() {
				if abs := a.absAttach(at.Path); abs != "" {
					if _, err := os.Stat(abs); err == nil {
						if src := a.localImage(abs); src != nil {
							ui.Image(c, src).Size(16, 16).Radius(t.RXs()).Fit(ui.Cover).Shrink(0)
						} else {
							kit.Ic(c, "image", kit.S12).Shrink(0)
						}
					} else {
						kit.Ic(c, "image", kit.S12).TextColor(t.Tx3).Shrink(0)
					}
				} else {
					kit.Ic(c, "image", kit.S12).Shrink(0)
				}
			})
			_ = chip
		}
	})
}

func (a *App) removeDraftAttachPath(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	pat := `!\[[^\]]*\]\(` + regexp.QuoteMeta(path) + `\)`
	a.draft = strings.TrimSpace(regexp.MustCompile(pat).ReplaceAllString(a.draft, ""))
}

func (a *App) composerAttachRow(c *ui.Context) {
	atts := parseDraftAttaches(a.draft)
	if len(atts) == 0 {
		return
	}
	t := a.tk
	ui.Row(c).Padding(6, 12, 0, 12).Gap(6).Wrap().Children(func() {
		for _, at := range atts {
			at := at
			chip := ui.Row(c).Height(24).PaddingX(4).Gap(4).Radius(t.RSm()).AlignItems(ui.Center).Border(1, t.Bd2).Background(t.Fill1).Cursor(ui.CursorDefault)
			chip.Children(func() {
				if abs := a.absAttach(at.Path); abs != "" {
					if _, err := os.Stat(abs); err == nil {
						if src := a.localImage(abs); src != nil {
							ui.Image(c, src).Size(18, 18).Radius(t.RXs()).Fit(ui.Cover).Shrink(0)
						} else {
							a.fileIc(c, baseName(at.Path))
						}
					} else {
						a.fileIc(c, baseName(at.Path))
					}
				} else {
					a.fileIc(c, baseName(at.Path))
				}
				ui.Text(c, baseName(at.Path)).FontSize(theme.FsCap).TextColor(t.Tx2).SingleLine().MaxWidth(120)
				kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "移除"}).OnClick(func() { a.removeDraftAttachPath(at.Path) })
			})
			_ = chip
		}
	})
}

// submitPrompt 发送输入框内容(拆分图片附件、处理内置斜杠)。
func (a *App) submitPrompt(text string, steer bool) {
	body, rels := splitDraftImages(text)
	var imgs []string
	for _, rel := range rels {
		if abs := a.absAttach(rel); abs != "" {
			imgs = append(imgs, abs)
		}
	}
	body = strings.TrimSpace(a.expandPasteRefs(body))
	if body == "" && len(imgs) == 0 {
		return
	}
	if !steer && a.tryBuiltInSlash(body) {
		a.draft = ""
		return
	}
	attachLog.Info("发送提示", "len", len(body), "images", len(imgs), "steer", steer)
	a.send(core.CmdPrompt{Text: body, Images: imgs, Steer: steer})
	a.draft = ""
}

func (a *App) tryBuiltInSlash(text string) bool {
	low := strings.ToLower(strings.TrimSpace(text))
	switch {
	case low == "/clear":
		a.toast = "输入已清空(/clear 仅演示提示,不会发给 Claude)"
		a.draft = ""
		return true
	case low == "/compact" || strings.HasPrefix(low, "/compact "):
		a.send(core.CmdCompact{})
		a.draft = ""
		return true
	}
	return false
}
