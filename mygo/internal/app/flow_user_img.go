package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) resolveUserImage(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		return ""
	}
	if strings.HasPrefix(p, "/demo/") {
		rel := strings.TrimPrefix(p, "/demo/")
		if a.ide != nil && a.ide.fs != nil {
			abs := a.ide.fs.Abs(rel)
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	if a.proj != "" {
		abs := filepath.Join(a.proj, p)
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}
	return ""
}

func (a *App) userMsgImages(c *ui.Context, u *core.User) {
	if len(u.Images) == 0 {
		return
	}
	t := a.tk
	expKey := "userimg:" + u.ID
	exp := a.foldExpanded(expKey)
	ui.Row(c).Gap(6).Margin(6, 0, 0, 0).Children(func() {
		for i, raw := range u.Images {
			abs := a.resolveUserImage(raw)
			if abs == "" {
				ui.Row(c).Height(56).Padding(0, 10).Radius(t.RSm()).Background(t.Fill1).AlignItems(ui.Center).Children(func() {
					kit.Ic(c, "image", kit.S12).TextColor(t.Tx4)
					ui.Text(c, baseName(raw)).FontSize(theme.FsMicro).TextColor(t.Tx4).Margin(0, 0, 0, 6)
				})
				continue
			}
			btn := ui.ButtonBase(c).Key("uimg-" + u.ID + "-" + itoa(i)).Height(56).MinWidth(56).MaxWidth(160).
				Radius(t.RSm()).Clip().Cursor(ui.CursorPointer).Background(t.Fill1)
			btn.OnClick(func() { a.setFold(expKey, !exp) })
			btn.Children(func() {
				if src := a.localImage(abs); src != nil {
					ui.Image(c, src).Height(56).Grow(1).Fit(ui.Cover)
				} else {
					kit.Ic(c, "image", kit.S12).TextColor(t.Tx4).Margin(0, 20)
				}
			})
		}
	})
	if exp {
		for i, raw := range u.Images {
			abs := a.resolveUserImage(raw)
			if abs == "" {
				continue
			}
			ui.Column(c).Key("uimg-exp-"+itoa(i)).Margin(4, 0, 0, 0).Gap(4).Children(func() {
				if src := a.localImage(abs); src != nil {
					ui.Image(c, src).MaxWidth(480).MaxHeight(360).Fit(ui.Contain).Radius(t.RMd()).Border(1, t.Bd1)
				}
				ui.Row(c).Gap(6).Children(func() {
					kit.Btn(c, "收起", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.setFold(expKey, false) })
					kit.Btn(c, "在预览中打开", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "ext"}).OnClick(func() {
						_ = exec.Command("open", abs).Start()
					})
				})
			})
		}
	}
}
