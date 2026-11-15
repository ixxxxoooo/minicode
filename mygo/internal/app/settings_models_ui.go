package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

func (a *App) defaultModelRow(c *ui.Context) {
	t := a.tk
	cur := modelChipIndex(a.prefs.Str(prefModel, "sonnet"))
	ui.Row(c).Gap(8).Shrink(0).Children(func() {
		for i, label := range defaultModelLabels {
			i, label := i, label
			ch := kit.Chip(c, label, kit.ChipOpt{Line: cur != i, On: cur == i}).Key("chip-model-" + itoa(i))
			if cur == i {
				ch.Background(t.AcSoft).TextColor(t.AcTx)
			}
			if ch.Clicked() {
				id := defaultModelIDs[i]
				a.prefs.Set(prefModel, id)
				if a.be != nil {
					a.pickModel(id)
				}
				a.applyPrefs()
			}
		}
	})
}
