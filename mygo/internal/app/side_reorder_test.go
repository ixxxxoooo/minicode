package app

import (
	"strings"
	"testing"
)

func wsNames(a *App) string {
	var s []string
	for _, w := range a.registry().Workspaces() {
		s = append(s, w.Name)
	}
	return strings.Join(s, ",")
}

func folderNames(a *App, id string) string {
	w, _ := a.registry().Workspace(id)
	var s []string
	for _, f := range w.Folders {
		s = append(s, f.Label())
	}
	return strings.Join(s, ",")
}

// 工作区拖拽换序:真拖拽(按下 → 移动 → 松手)+ 菜单「上移 / 下移」走同一个存储函数。
func TestWorkspaceDragReorder(t *testing.T) {
	a, tt := newTestApp(t)
	if wsNames(a) != "全栈工作区,文档与素材" {
		t.Fatalf("演示应有两个工作区:%s", wsNames(a))
	}
	from, ok1 := tt.Find("文档与素材")
	to, ok2 := tt.Find("全栈工作区")
	if !ok1 || !ok2 {
		t.Fatal("侧栏应显示两个工作区行")
	}
	fx, fy := from.X+from.W/2, from.Y+from.H/2
	tx, ty := to.X+to.W/2, to.Y+to.H/2
	tt.Press(fx, fy)
	for i := 1; i <= 6; i++ {
		tt.Move(fx+(tx-fx)*float32(i)/6, fy+(ty-fy)*float32(i)/6)
		tt.Frame()
	}
	tt.Release(tx, ty)
	tt.Frame()
	if wsNames(a) != "文档与素材,全栈工作区" {
		t.Fatalf("拖到第一个之前后顺序不对:%s", wsNames(a))
	}
	// 菜单:下移回去;边界置灰
	a.moveWorkspace("ws-fullstack", "ws-content")
	if wsNames(a) != "全栈工作区,文档与素材" {
		t.Fatalf("还原:%s", wsNames(a))
	}
	if err := tt.RightClick("文档与素材"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.ChooseMenuItem("上移"); err != nil {
		t.Fatal(err)
	}
	if wsNames(a) != "文档与素材,全栈工作区" {
		t.Fatalf("菜单上移:%s", wsNames(a))
	}
}

// 成员文件夹换序:第一个是主文件夹不参与,拖到主文件夹上给出说明。
func TestMemberFolderReorder(t *testing.T) {
	a, _ := newTestApp(t)
	a.moveFolder("ws-fullstack", 2, 1) // docs-site 放到 web-dashboard 之前
	if folderNames(a, "ws-fullstack") != "router-demo,docs-site,web-dashboard" {
		t.Fatalf("换序:%s", folderNames(a, "ws-fullstack"))
	}
	a.moveFolder("ws-fullstack", 1, 3) // 放到最后
	if folderNames(a, "ws-fullstack") != "router-demo,web-dashboard,docs-site" {
		t.Fatalf("放到最后:%s", folderNames(a, "ws-fullstack"))
	}
	a.toast = ""
	a.moveFolder("ws-fullstack", 0, 2) // 拖主文件夹
	a.moveFolder("ws-fullstack", 2, 0) // 放到主文件夹之前
	if folderNames(a, "ws-fullstack") != "router-demo,web-dashboard,docs-site" || !strings.Contains(a.toast, "主文件夹") {
		t.Fatalf("主文件夹应固定:%s %q", folderNames(a, "ws-fullstack"), a.toast)
	}
	// 正在使用的工作区换序后文件树跟着变
	a.ActivateWorkspace("ws-fullstack")
	a.moveFolder("ws-fullstack", 2, 1)
	if a.repos[1].name != "docs-site" {
		t.Fatalf("使用中的工作区换序应立即生效:%s", a.repos[1].name)
	}
}
