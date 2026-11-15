package app

// askCloseTermTab 关闭第 i 个终端标签;有前台命令时先确认。
func (a *App) askCloseTermTab(i int) {
	if i < 0 || i >= len(a.ide.tps) {
		return
	}
	tp := a.ide.tps[i]
	if tp.t != nil && tp.t.CmdRunning() {
		a.confirmDlg("结束正在运行的命令?", "这个终端里还有前台命令在跑,结束 shell 会一并终止。", "结束 shell", true, func() {
			a.closeTermTab(i)
		})
		return
	}
	a.closeTermTab(i)
}
