package app

// 外部修改检测的辅助状态(不挂在 ide / ideTab 上,避免与禁改文件交叉依赖)。
type extPathFlags struct {
	keep, deleted, orphan bool
}

var (
	extHubFocus  bool
	extHubAlert  map[string]diskSnap
	extPathState map[string]extPathFlags
)

func extAlerted(p string, cur diskSnap) bool {
	if extHubAlert == nil {
		return false
	}
	alerted, ok := extHubAlert[p]
	return ok && alerted.mod.Equal(cur.mod) && alerted.size == cur.size
}

func extMarkAlerted(p string, s diskSnap) {
	if extHubAlert == nil {
		extHubAlert = map[string]diskSnap{}
	}
	extHubAlert[p] = s
}

func extClearAlerted(p string) {
	if extHubAlert != nil {
		delete(extHubAlert, p)
	}
}

func extFlags(p string) extPathFlags {
	if extPathState == nil {
		return extPathFlags{}
	}
	return extPathState[p]
}

func extSetFlags(p string, f extPathFlags) {
	if extPathState == nil {
		extPathState = map[string]extPathFlags{}
	}
	extPathState[p] = f
}

func extResetFlags(p string) {
	extSetFlags(p, extPathFlags{})
}
