package app

import "time"

// gitDebug 处理 -ed 中与 Git / 差异相关的调试动作。
func (a *App) gitDebug(name, arg string) bool {
	switch name {
	case "pushreject":
		a.dbgSCM()
		a.sc.pushGuide = "远端 origin/main 比本地多 2 个提交(non-fast-forward)"
		return true
	case "cmp2":
		a.dbgSCM()
		if len(a.sc.log) >= 2 {
			a.graphTab.picks = []string{a.sc.log[1].Hash, a.sc.log[0].Hash}
			a.scmCompareTwo(a.sc.log[1].Hash, a.sc.log[0].Hash)
		}
		return true
	case "cmphead":
		a.dbgSCM()
		if len(a.sc.log) > 0 {
			a.scmCompareHead(a.sc.log[1%len(a.sc.log)])
		}
		return true
	case "stageaccepted":
		a.OpenReview()
		a.reviewStageAccepted()
		return true
	case "blamecard":
		a.SetMode(ModeIDE)
		if arg != "" {
			a.ide.openFile(arg, true)
		} else if len(a.ide.tabs) == 0 {
			a.ide.openFile("src/router.rs", true)
		}
		if tb := a.ide.cur(); tb != nil {
			tb.st.GoTo(tb.st.CursorPos())
			a.ide.blameLine = tb.st.CursorPos().Line
			a.ide.blameHoverAt = time.Now().Add(-400 * time.Millisecond)
			a.ide.blameCardShow = true
		}
		return true
	case "stashfiles":
		a.dbgSCM()
		a.scmStashToggle(0)
		return true
	}
	return false
}
