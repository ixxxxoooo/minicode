package app

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// 文件树右键菜单补项:让 Claude… / 在侧边打开 / 复制·剪切·粘贴(设计稿 menus 页)。

type fileTreeClip struct {
	rel string
	cut bool
	dir bool
}

type treeClaudeDlg struct {
	rel   string
	draft string
}

// treeClaudeAsk 打开「让 Claude…」小输入框。
func (a *App) treeClaudeAsk(rel string) {
	a.treeAsk = treeClaudeDlg{rel: rel, draft: ""}
}

func (a *App) closeTreeClaudeAsk() { a.treeAsk = treeClaudeDlg{} }

func (a *App) submitTreeClaudeAsk() {
	d := a.treeAsk
	if d.rel == "" {
		return
	}
	instr := strings.TrimSpace(d.draft)
	line := strings.TrimSpace(instr + " " + refText(d.rel, false))
	if instr == "" {
		line = refText(d.rel, false)
	}
	a.draft = strings.TrimSpace(a.draft + "\n" + line)
	a.ux.focusDraft = true
	a.revealChat()
	a.closeTreeClaudeAsk()
	uxLog.Info("让 Claude…", "path", d.rel)
}

// treeClaudeDialog 画「让 Claude…」居中浮层。
func (a *App) treeClaudeDialog(c *ui.Context) {
	d := &a.treeAsk
	if d.rel == "" {
		return
	}
	t := a.tk
	a.dialogOpen("tree-claude", true, func(open *bool) {
		ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
			backdrop.Background(t.Scrim).Justify(ui.Center).AlignItems(ui.Center)
			panel.Width(420).Background(t.Raised).Radius(t.RXl()).Column().Border(1, t.Bd1).Padding(14, 16).Gap(6)
			for _, sh := range t.Shadow3() {
				panel.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
			}
			kit.Float(panel)
			a.dialogIn("tree-claude", backdrop, panel)
			panel.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "让 Claude…").FontSize(theme.FsHead).FontWeight(600).Grow(1)
					kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭 Esc"}).OnClick(func() { *open = false; a.closeTreeClaudeAsk() })
				})
				ui.Text(c, "输入要对 "+d.rel+" 说的话(回车确认):").FontSize(theme.FsCap).TextColor(t.Tx3)
				in := ui.TextInput(c.Key("tree-claude-in"), &d.draft).Grow(1).AutoFocus().Placeholder("例如:解释一下这段代码").PlaceholderColor(t.Tx4)
				in.OnTextKey(func(k ui.TextKey) bool {
					if k.Key == ui.KeyEnter && k.Mods == 0 {
						a.submitTreeClaudeAsk()
						return true
					}
					return false
				})
				ui.Row(c).Gap(8).Justify(ui.End).Margin(8, 0, 0, 0).Children(func() {
					kit.Btn(c, "取消", kit.BtnOpt{Variant: kit.BtnGhost}).OnClick(func() { *open = false; a.closeTreeClaudeAsk() })
					kit.Btn(c, "填入对话", kit.BtnOpt{Variant: kit.BtnPri}).OnClick(a.submitTreeClaudeAsk)
				})
			})
		})
		if !*open {
			a.closeTreeClaudeAsk()
		}
	})
}

func (a *App) treeCopyFile(rel string, cut bool) {
	a.treeClip = fileTreeClip{rel: rel, cut: cut, dir: false}
	a.toast = map[bool]string{true: "已剪切,可在目标文件夹右键粘贴", false: "已复制,可在目标文件夹右键粘贴"}[cut]
}

func (a *App) treePasteFile(into string) {
	clip := a.treeClip
	if clip.rel == "" || clip.dir {
		return
	}
	mu, why := a.mutable()
	if why != "" {
		a.toast = why
		return
	}
	if into == "" && a.multiRoot() {
		into = a.repos[0].name
	}
	base := path.Base(clip.rel)
	dest := path.Join(into, base)
	if dest == clip.rel {
		a.toast = "不能粘贴到同一位置"
		return
	}
	if clip.cut {
		if err := mu.Rename(clip.rel, dest); err != nil {
			a.toast = err.Error()
			return
		}
		a.renameTabs(clip.rel, dest)
		a.treeClip = fileTreeClip{}
		a.ide.listCache, a.ide.filesCache = nil, nil
		a.toast = "已移动"
		return
	}
	data, err := a.ide.fs.Read(clip.rel)
	if err != nil {
		a.toast = err.Error()
		return
	}
	if _, err := a.ide.fs.Read(dest); err == nil {
		dest = uniqueTreeDest(into, base, a.ide.fs)
	}
	if err := mu.Create(dest, false); err != nil {
		if errors.Is(err, workspace.ErrExist) {
			dest = uniqueTreeDest(into, base, a.ide.fs)
			if err2 := mu.Create(dest, false); err2 != nil {
				a.toast = err2.Error()
				return
			}
		} else {
			a.toast = err.Error()
			return
		}
	}
	if err := a.ide.fs.Write(dest, data); err != nil {
		a.toast = err.Error()
		_ = mu.Remove(dest)
		return
	}
	a.ide.listCache, a.ide.filesCache = nil, nil
	a.toast = "已粘贴"
}

// uniqueTreeDest 粘贴复制时若重名,自动加「副本」后缀。
func uniqueTreeDest(into, base string, fs workspace.FS) string {
	ext := path.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	try := path.Join(into, stem+" 副本"+ext)
	if _, err := fs.Read(try); err != nil {
		return try
	}
	for i := 2; i < 1000; i++ {
		try = path.Join(into, fmt.Sprintf("%s 副本 %d%s", stem, i, ext))
		if _, err := fs.Read(try); err != nil {
			return try
		}
	}
	return path.Join(into, stem+" 副本"+ext)
}

// treeFileActions 文件树「主要动作 / 导航 / 剪贴板」段(菜单 build 里调用)。
func (a *App) treeFileActions(m *ui.Menu, rel string, dir bool) {
	if !dir {
		if menuItem(m, "打开").Trailing("⏎").Chosen() {
			a.ide.openFile(rel, true)
		}
		if menuItem(m, "在侧边打开").Trailing("⌘⏎").Chosen() {
			a.openSide(rel, true)
		}
		m.Separator()
		clean := a.ide.fileGit(rel) == ""
		if menuItem(m, "打开更改").Disabled(clean).Tooltip(map[bool]string{true: "这个文件没有未提交的更改"}[clean]).Chosen() {
			a.openDiffKind(dkWork, rel, true)
		}
		if menuItem(m, "显示文件历史").Chosen() {
			a.ide.sel = rel
			a.ide.sections["时间线"] = true
			a.nav, a.sideOpen = "files", true
		}
		if menuItem(m, "让 Claude…").Chosen() {
			a.treeClaudeAsk(rel)
		}
	}
}

func (a *App) treeClipMenu(m *ui.Menu, rel string, dir bool) {
	_, ro := a.mutable()
	if !dir {
		if menuItem(m, "复制").Disabled(ro != "").Tooltip(ro).Chosen() {
			a.treeCopyFile(rel, false)
		}
		if menuItem(m, "剪切").Disabled(ro != "").Tooltip(ro).Chosen() {
			a.treeCopyFile(rel, true)
		}
	}
	canPaste := a.treeClip.rel != "" && !a.treeClip.dir && (dir || !dir)
	pasteWhy := ""
	if a.treeClip.rel == "" {
		pasteWhy = "剪贴板里没有文件"
	} else if a.treeClip.dir {
		pasteWhy = "暂不支持粘贴文件夹"
	} else if ro != "" {
		pasteWhy = ro
	}
	if menuItem(m, "粘贴").Disabled(!canPaste || ro != "").Tooltip(pasteWhy).Chosen() {
		into := rel
		if !dir {
			into = parentDir(rel)
		}
		a.treePasteFile(into)
	}
}
