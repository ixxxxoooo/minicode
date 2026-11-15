package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) userRewindBegin(userID string) {
	uuid, err := a.st.RewindBegin(userID)
	if err != nil {
		a.toast = err.Error()
		return
	}
	log.Info("回退文件预览", "user", userID, "uuid", uuid)
	if a.be != nil && a.be.Name() == "demo" {
		out := demoRewindPreview(uuid)
		apply := func() { a.st.Apply(core.EvRewind{Outcome: out}) }
		if a.win != nil {
			a.win.Update(apply)
		} else {
			apply()
		}
		return
	}
	a.send(core.CmdRewindFiles{UUID: uuid, DryRun: true})
}

func demoRewindPreview(uuid string) core.RewindOutcome {
	if strings.HasPrefix(uuid, "no-ckpt") {
		return core.RewindFailed(uuid, true, "No file checkpoint found for this message.")
	}
	return core.RewindOutcome{
		UUID: uuid, DryRun: true, CanRewind: true,
		Files:      []string{"src/router.rs", "tests/router_test.rs"},
		Insertions: 18, Deletions: 4,
	}
}

func (a *App) rewindConfirmApply() {
	uuid, ok := a.st.RewindConfirm()
	if !ok {
		a.toast = "没有可回退的改动"
		return
	}
	if a.be != nil && a.be.Name() == "demo" {
		apply := func() {
			a.st.Apply(core.EvRewind{Outcome: core.RewindOutcome{UUID: uuid, CanRewind: true, Files: []string{"src/router.rs", "tests/router_test.rs"}}})
		}
		if a.win != nil {
			a.win.Update(apply)
		} else {
			apply()
		}
		return
	}
	a.send(core.CmdRewindFiles{UUID: uuid, DryRun: false})
	log.Info("确认回退文件", "uuid", uuid)
}

func (a *App) flowRewindCard(c *ui.Context) {
	r := a.st.Rewind
	if r == nil {
		return
	}
	t := a.tk
	box := ui.Column(c).Key("rewind-card").Background(t.Raised).Radius(t.RMd()).Padding(10, 12).Gap(8).Margin(8, 0).Border(1, t.Bd1)
	box.Children(func() {
		switch r.Stage {
		case core.RewindChecking:
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				kit.Spin(c, t.Ac)
				ui.Text(c, "正在预览可回退的文件…").TextColor(t.Tx2)
			})
		case core.RewindReady:
			ui.Text(c, "回退到这条消息之前?").FontWeight(600)
			o := r.Outcome
			if !o.HasChanges() {
				ui.Text(c, o.Reason()).TextColor(t.Tx3).FontSize(theme.FsCap)
				ui.Row(c).Gap(6).Justify(ui.End).Children(func() {
					kit.Btn(c, "知道了", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.st.RewindCancel() })
				})
				return
			}
			n := len(o.Files)
			ui.Text(c, "以下 "+itoa(n)+" 个文件会恢复到发送这条消息之前的内容:").FontSize(theme.FsCap).TextColor(t.Tx3)
			ui.Column(c).Background(t.Sunken).Radius(t.RSm()).Padding(2, 4).Gap(0).Border(1, t.Bd1).Children(func() {
				show := o.Files
				if len(show) > 6 {
					show = show[:6]
				}
				for i, p := range show {
					ui.Row(c).Key(i).Gap(6).Padding(3, 4).AlignItems(ui.Center).Children(func() {
						kit.Ic(c, "code", kit.S12).TextColor(t.Tx3)
						ui.Text(c, baseName(p)).SingleLine().Shrink(0)
						ui.Text(c, trimSlash(dirOf(p))).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Grow(1)
					})
				}
				if len(o.Files) > 6 {
					ui.Text(c, "…还有 "+itoa(len(o.Files)-6)+" 个").FontSize(theme.FsMicro).TextColor(t.Tx4).Padding(4, 6)
				}
			})
			ui.Row(c).Gap(6).Justify(ui.End).Children(func() {
				kit.Btn(c, "取消", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.st.RewindCancel() })
				kit.Btn(c, "回退文件", kit.BtnOpt{Small: true, Variant: kit.BtnPri, Icon: "undo"}).OnClick(a.rewindConfirmApply)
			})
		case core.RewindApplying:
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				kit.Spin(c, t.Ac)
				ui.Text(c, "正在回退 "+itoa(r.Files)+" 个文件…").TextColor(t.Tx2)
			})
		}
	})
}
