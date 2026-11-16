package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/projects"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 工作区附加文件夹权限提示(设计稿 proj-ws「权限提示」)。

type wsPermPending struct {
	when        string // save / session
	pendingSess string
	extraRO     bool
}

// maybeWsPermBeforeSession 首次在工作区启动会话时再确认一次附加目录权限。
func (a *App) maybeWsPermBeforeSession(id string) bool {
	if a.ws == nil || len(a.ws.Extra()) == 0 || a.ws.PermSessionAck {
		return false
	}
	a.wsPerm = wsPermPending{when: "session", pendingSess: id}
	a.pd.kind = "ws-perm"
	return true
}

func (a *App) wsExtraNames() string {
	if a.ws == nil {
		return ""
	}
	return strings.Join(a.ws.ExtraDirLabels(), "、")
}

func (a *App) wsPermChoose(readOnly bool) {
	when := a.wsPerm.when
	sid := a.wsPerm.pendingSess
	a.wsPerm.extraRO = readOnly
	if when == "session" && a.ws != nil {
		w := *a.ws
		w.ExtraReadOnly = readOnly
		w.PermSessionAck = true
		if nw, err := a.registry().SaveWorkspace(w); err == nil {
			*a.ws = nw
		}
		a.pd.kind = ""
		a.wsPerm = wsPermPending{}
		log.Info("工作区会话启动附加目录授权", "readOnly", readOnly)
		if sid != "" {
			a.OpenSession(sid)
		}
		return
	}
	if when == "save" {
		a.pd.extraReadOnly = readOnly
		a.pd.kind = "ws"
		a.wsPerm = wsPermPending{}
		a.submitWorkspaceAfterPerm()
		log.Info("工作区保存附加目录授权", "readOnly", readOnly)
	}
}

func (a *App) wsPermDialog(c *ui.Context) {
	t := a.tk
	n, labels := 0, ""
	if a.ws != nil {
		n = len(a.ws.Extra())
		labels = a.wsExtraNames()
	} else if len(a.pd.folders) >= 2 {
		n = len(a.pd.folders) - 1
		var ls []string
		for i, f := range a.pd.folders {
			if i > 0 {
				ls = append(ls, f.Label())
			}
		}
		labels = strings.Join(ls, "、")
	}
	if n == 0 {
		return
	}
	sub := map[string]string{
		"session": "第一次在本工作区启动会话时再确认一次。",
		"save":    "保存工作区前请确认一次。",
	}[a.wsPerm.when]
	a.dlgFrame(c, "ws-perm", 440, "附加文件夹权限", func() {
		ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
			kit.Ic(c, "shield", 20).TextColor(t.Ac).Shrink(0)
			ui.Column(c).Grow(1).Gap(6).Children(func() {
				ui.Text(c, "Claude 将获得 "+itoa(n)+" 个附加文件夹的读写权限").FontWeight(600)
				ui.Text(c, labels+"。可在 设置 → 权限与信任 → 附加目录 里随时撤销;"+sub).FontSize(theme.FsCap).TextColor(t.Tx2).LineHeight(theme.LhBody)
				ui.Text(c, "只读授权 = 仍以 --add-dir 加入,并在系统提示里声明不要修改这些目录内的文件。").FontSize(theme.FsMicro).TextColor(t.Tx3)
				ui.Row(c).Gap(8).Margin(4, 0, 0, 0).Children(func() {
					kit.Btn(c, "我知道了", kit.BtnOpt{Variant: kit.BtnPri, Small: true}).OnClick(func() { a.wsPermChoose(false) })
					kit.Btn(c, "只读授权", kit.BtnOpt{Small: true}).OnClick(func() { a.wsPermChoose(true) })
				})
			})
		})
	})
}

func (a *App) submitWorkspaceAfterPerm() {
	d := &a.pd
	w, err := a.registry().SaveWorkspace(projects.Workspace{
		ID: d.wsID, Name: d.name, Folders: d.folders, LoadMD: d.loadMD,
		ExtraReadOnly: d.extraReadOnly, PermSaveAck: true,
	})
	if err != nil {
		d.err = err.Error()
		return
	}
	a.closeProjDlg()
	a.projOpen[w.ID] = true
	if a.ws != nil && a.ws.ID == w.ID {
		id := w.ID
		a.leaveWorkspace()
		a.ActivateWorkspace(id)
	}
	a.toast = map[bool]string{true: "已更新工作区「" + w.Name + "」", false: "已创建工作区「" + w.Name + "」"}[d.wsID != ""]
}

// OpenSessionChecked 打开会话;工作区首次启动会先出权限卡。
func (a *App) OpenSessionChecked(id string) {
	if a.maybeWsPermBeforeSession(id) {
		return
	}
	a.OpenSession(id)
}
