package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// slowHintText 只给「最可能的一条」,按设计稿「慢因规则」的优先级匹配;文案写事实 + 下一步,不猜测。
// 返回的 action 非空时,提示卡尾部带一个动作按钮。
func (a *App) slowHintText(ls core.LiveStage) (text, action string) {
	el := a.now().Sub(ls.Since).Seconds()
	switch ls.Kind {
	case core.StageFin:
		return "最后一段回复已经出完,Claude 还没返回「完成」。常见于上游迟迟不关流或有后台任务在收尾;可以直接发新消息,不用等。", ""
	case core.StageConn:
		return fmt.Sprintf("Claude 进程启动较慢(已等 %.0f 秒)。设置 → 对话 → 连接预热选「打开会话时」可以提前启动。", el), ""
	}
	used := a.st.Usage.ContextUsed
	if n := a.rtLoadedFolders(); n > 0 && used > 20_000 {
		return fmt.Sprintf("比平时慢。这次请求带着约 %s tokens 的上下文;工作区有 %d 个附加文件夹并读取了它们的 CLAUDE.md,这部分每次都计入。", fmtTok(used), n), "编辑工作区…"
	}
	if used > 40_000 {
		return fmt.Sprintf("比平时慢。这次请求带着约 %s tokens 的上下文,模型要先读完才会出第一个字;长会话可以压缩上下文。", fmtTok(used)), "压缩上下文"
	}
	prov, _ := a.pvModelLabel()
	if prov == "" {
		prov = "上游"
	}
	return fmt.Sprintf("%s响应较慢(已等 %.0f 秒)。这段时间在等模型的第一个字,不是本地卡住;结束后点回复下方的「用时」可看各阶段与缓存。", prov, el), ""
}

// slowHint 等待行下面的提示卡(琥珀淡底,与会话流里的警告提示同款)。
func (a *App) slowHint(c *ui.Context, ls core.LiveStage) {
	t := a.tk
	text, action := a.slowHintText(ls)
	ui.Row(c).Gap(8).AlignItems(ui.Start).Margin(2, 0, 4, 16).Padding(7, 10).Radius(t.RMd()).Background(t.WarnSoft).Children(func() {
		kit.Ic(c, "alert", kit.S12).TextColor(t.Warn).Margin(3, 0, 0, 0)
		ui.Text(c, text).Grow(1).MinWidth(0).LineHeight(1.55).FontSize(a.FlowCapFont()).TextColor(t.Tx2)
		if action != "" && kit.Btn(c, action, kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).Clicked() {
			switch action {
			case "编辑工作区…":
				if a.ws != nil {
					a.openWsDialog(a.ws.ID, "")
				}
			case "压缩上下文":
				a.send(core.CmdCompact{})
				log.Info("慢响应提示:压缩上下文")
			}
		}
	})
}
