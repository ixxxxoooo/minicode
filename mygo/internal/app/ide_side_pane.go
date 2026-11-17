package app

// sideSecHeight 左栏折叠区块内容区高度(持久化;窗口变窄时上限缩小)。
func (a *App) sideSecHeight(key string, def, outerH float32) float32 {
	cur := a.ly.sideOutline
	if key == lySideTimeline {
		cur = a.ly.sideTimeline
	}
	if cur <= 0 {
		cur = def
	}
	return clampF(cur, sideSecMin, a.sideSecMax(outerH))
}

func (a *App) sideSecMax(outerH float32) float32 {
	if outerH <= 0 {
		return sideSecMax
	}
	hi := outerH * 0.55
	if hi < sideSecMin {
		hi = sideSecMin
	}
	if hi > sideSecMax {
		hi = sideSecMax
	}
	return hi
}
