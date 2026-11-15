package app

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

type slashRow struct {
	name    string
	desc    string
	builtin string // compact / clear / model
}

func (a *App) slashPrefix() string {
	s := strings.TrimLeft(a.draft, " \t")
	if !strings.HasPrefix(s, "/") {
		return ""
	}
	if i := strings.IndexAny(s, " \n\t"); i >= 0 {
		return "" // 已有参数,不再补全
	}
	return strings.TrimPrefix(s, "/")
}

func (a *App) slashRows() []slashRow {
	prefix := strings.ToLower(a.slashPrefix())
	var rows []slashRow
	add := func(name, desc, builtin string) {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), "/"+prefix) {
			return
		}
		rows = append(rows, slashRow{name: name, desc: desc, builtin: builtin})
	}
	add("/compact", "压缩上下文", "compact")
	add("/clear", "清空输入(仅提示)", "clear")
	add("/model", "打开运行配置…", "model")
	for _, it := range a.slashCommandItems() {
		if !it.On {
			continue
		}
		nm := it.Name
		if !strings.HasPrefix(nm, "/") {
			nm = "/" + nm
		}
		desc := it.Note
		if desc == "" {
			desc = it.Desc
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(nm), "/"+prefix) {
			continue
		}
		rows = append(rows, slashRow{name: nm, desc: desc})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	return rows
}

func (a *App) slashCommandItems() []*customize.Item {
	if a.demoWs() {
		return a.cuStore().List(customize.Commands, customize.All, "")
	}
	if a.slashProj != a.proj {
		a.slashProj = a.proj
		a.slashDisk = nil
		if a.proj != "" {
			if ds, err := customize.OpenDisk(a.proj); err == nil {
				a.slashDisk = ds
			}
		}
	}
	if a.slashDisk == nil {
		return nil
	}
	return a.slashDisk.List(customize.Commands, customize.All, "")
}

func (a *App) slashOpen() bool { return a.slashPrefix() != "" }

func (a *App) slashPopup(c *ui.Context, anchor ui.Element) {
	if !a.slashOpen() {
		if a.popup == "slash" {
			a.popup = ""
		}
		return
	}
	rows := a.slashRows()
	if len(rows) == 0 {
		return
	}
	if a.slashSel >= len(rows) {
		a.slashSel = len(rows) - 1
	}
	if a.slashSel < 0 {
		a.slashSel = 0
	}
	key := "slash"
	if a.popup != "" && a.popup != key {
		return
	}
	a.popup = key
	a.popOpen(key, func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 280).Padding(0).Clip().AttachTo(anchor, ui.AnchorBottomLeft, ui.AnchorTopLeft).Margin(0, 0, 6, 0).MaxHeight(220)
			ui.Column(c).Children(func() {
				kit.Cap(c, "斜杠命令")
				for i, row := range rows {
					i, row := i, row
					if a.mi(c, "slash-"+itoa(i), "slash", row.name, row.desc, i == a.slashSel).Clicked() {
						a.slashApply(row)
					}
				}
			})
		})
	})
}

func (a *App) slashApply(row slashRow) {
	a.popup = ""
	switch row.builtin {
	case "compact":
		a.draft = "/compact "
	case "clear":
		a.toast = "输入已清空(/clear 不会发给 Claude)"
		a.draft = ""
	case "model":
		a.draft = strings.TrimLeft(a.draft, "/") // 去掉半截命令
		if i := strings.IndexAny(a.draft, " \n"); i >= 0 {
			a.draft = strings.TrimSpace(a.draft[i:])
		} else {
			a.draft = ""
		}
		a.togglePop("cfg")
		return
	default:
		a.draft = row.name + " "
	}
	a.ux.focusDraft = true
}

func (a *App) slashTextKey(k ui.TextKey) bool {
	if !a.slashOpen() {
		a.slashSel = 0
		return false
	}
	rows := a.slashRows()
	if len(rows) == 0 {
		return false
	}
	switch k.Key {
	case ui.KeyUp:
		a.slashSel = (a.slashSel - 1 + len(rows)) % len(rows)
		return true
	case ui.KeyDown:
		a.slashSel = (a.slashSel + 1) % len(rows)
		return true
	case ui.KeyTab, ui.KeyEnter:
		if k.Key == ui.KeyEnter && k.Mods != 0 {
			return false
		}
		a.slashApply(rows[a.slashSel])
		if k.Key == ui.KeyEnter {
			return true
		}
		return true
	case ui.KeyEscape:
		a.draft = ""
		return true
	}
	return false
}
