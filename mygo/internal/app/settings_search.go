package app

import (
	"strings"
	"time"
)

const setFlashDur = 1400 * time.Millisecond

// rowFlashID 是设置行的定位键(优先偏好键,否则标题)。
func rowFlashID(r setRow) string {
	if r.key != "" {
		return r.key
	}
	return r.title
}

// jumpSettingRow 从搜索结果跳进分类并准备滚动与高亮。
func (a *App) jumpSettingRow(tab string, r setRow) {
	a.setTab, a.set.query = tab, ""
	a.set.jumpKey = rowFlashID(r)
	a.set.flashKey = a.set.jumpKey
	a.set.flashUntil = time.Now().Add(setFlashDur)
	a.set.mainSt.Y = a.settingsScrollY(tab, a.set.jumpKey)
	a.set.lastTab = tab + "|"
}

// syncSettingJump 渲染后应用待滚动位置。
func (a *App) syncSettingJump() {
	if a.set.jumpKey == "" {
		return
	}
	y := a.settingsScrollY(a.setTab, a.set.jumpKey)
	if y >= 0 {
		a.set.mainSt.Y = y
	}
	a.set.jumpKey = ""
}

// settingsScrollY 估算目标行在滚动区内的 Y(用于 scrollIntoView)。
func (a *App) settingsScrollY(tabID, rowKey string) float32 {
	tab := a.setTabByID(tabID)
	const (
		header   = float32(52) // 分类标题区
		grpTitle = float32(22) // 分组小标题 + 间距
		rowH     = float32(38)
		grpPad   = float32(14)
	)
	y := header
	for _, g := range tab.groups {
		if g.title != "" {
			y += grpTitle
		}
		// 先是行,后是自定义块(与渲染顺序一致)
		for _, r := range g.rows {
			if rowFlashID(r) == rowKey {
				return y
			}
			if strings.Contains(rowKey, r.title) && r.title != "" && rowKey == r.title {
				return y
			}
			y += rowH
		}
		if g.body != nil {
			// 自定义块:按分组标题匹配时滚到块头
			if rowKey == g.title {
				return y
			}
			y += 120 // 估算自定义块高度
		}
		y += grpPad
	}
	return 0
}

// settingRowFlash 是否对该行画高亮底。
func (a *App) settingRowFlash(r setRow) bool {
	if a.set.flashKey == "" {
		return false
	}
	if rowFlashID(r) != a.set.flashKey {
		return false
	}
	if time.Now().After(a.set.flashUntil) {
		a.set.flashKey = ""
		return false
	}
	return true
}
