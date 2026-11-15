package app

import (
	"github.com/egoist/mygo/ui"
)

// tabDrag 是编辑器标签跨组拖动的载荷。
type tabDrag struct {
	ID   string
	From int8 // 来源编辑器组 0 / 1
}

func (d *ide) paneTabSlice(pane int8) *[]*ideTab {
	if pane == 1 && d.splitDir != 0 {
		return &d.tabs2
	}
	return &d.tabs
}

// tabDrop 处理标签落在某一组标签条上的换位 / 跨组移动。
func (a *App) tabDrop(pane int8, beforeID string, d tabDrag) {
	dide := a.ide
	if dide.splitDir == 0 || d.ID == beforeID {
		return
	}
	src := dide.paneTabSlice(d.From)
	dst := dide.paneTabSlice(pane)
	var moved *ideTab
	sl := *src
	for i, t := range sl {
		if t.id == d.ID {
			moved = t
			*src = append(sl[:i], sl[i+1:]...)
			break
		}
	}
	if moved == nil {
		return
	}
	dl := *dst
	at := len(dl)
	if beforeID != "" {
		for i, t := range dl {
			if t.id == beforeID {
				at = i
				break
			}
		}
	}
	*dst = append(dl, nil)
	copy((*dst)[at+1:], (*dst)[at:])
	(*dst)[at] = moved
	*dide.paneActive(pane) = moved.id
	dide.paneSel = pane
	if moved.path != "" {
		dide.sel = moved.path
	}
	uxLog.Info("标签跨组移动", "id", moved.id, "from", d.From, "to", pane, "before", beforeID)
}

// etabDrag 给标签条元素接上拖动与跨组落点。
func (a *App) etabDrag(el ui.Element, tb *ideTab, pane int8) {
	el.Drag(tabDrag{ID: tb.id, From: pane})
	if d, ok := ui.Drop[tabDrag](el); ok && d.ID != tb.id {
		a.tabDrop(pane, tb.id, d)
	}
}
