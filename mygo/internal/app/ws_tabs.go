package app

import (
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 虚拟标签的种类。它们是工作区(编辑器标签组)里的标签,但不对应磁盘文件:
// 计划、子 Agent、后台任务输出、本轮审查、合并冲突。每种内容只有这一个「家」,
// 会话流里的卡片、任务坞、SCM 只是打开它的入口。
const (
	vReview   = "review"
	vPlan     = "plan"
	vSub      = "sub"
	vTask     = "task"
	vConflict = "conflict"
)

// virtID 虚拟标签的 id:同一种内容(同一个 ref)只会有一个标签。
func virtID(kind, ref string) string {
	if ref == "" {
		return "v:" + kind
	}
	return "v:" + kind + ":" + ref
}

// isVirt 是否虚拟标签。
func (t *ideTab) isVirt() bool { return t.virt != "" }

// openVirtual 打开(或激活)一个虚拟标签。
func (d *ide) openVirtual(kind, ref string, pin bool) *ideTab {
	id := virtID(kind, ref)
	if t := d.tab(id); t != nil {
		if pin {
			t.preview = false
		}
		d.active = id
		return t
	}
	t := &ideTab{id: id, virt: kind, ref: ref, preview: !pin}
	d.addTab(t)
	d.sel = ""
	return t
}

// OpenWorkspaceTab 打开工作区里的虚拟标签:Agent 形态下工作区收起时先展开并闪一下边框,
// IDE 形态下对话栏占满时先恢复。所有入口(会话流卡片、任务坞、SCM、命令面板)都走这里。
func (a *App) OpenWorkspaceTab(kind, ref string) {
	a.ide.openVirtual(kind, ref, true)
	a.revealWorkspace()
}

// revealWorkspace 保证工作区可见,并让它的边框闪一下。
func (a *App) revealWorkspace() {
	if a.mode == ModeAgent && !a.rightOn {
		a.rightOn = true
	}
	if a.mode == ModeIDE {
		a.chatMax = false
	}
	a.screen = ScreenWork
	a.flashAt = time.Now()
}

// toggleRight 切换右侧栏(Agent = 工作区,IDE = 对话栏)。
func (a *App) toggleRight() {
	a.rightOn = !a.rightOn
	if a.mode == ModeIDE && !a.rightOn {
		a.chatMax = false
	}
}

// tabTitle 是标签上显示的名字。
func (a *App) tabTitle(t *ideTab) string {
	switch t.virt {
	case vReview:
		return "审查 · 本轮改动"
	case vPlan:
		return "计划"
	case vSub:
		if s := a.findSub(t.ref); s != nil {
			return core.ShortTitle(s.Description)
		}
		return "子 Agent"
	case vTask:
		if tk := a.findTask(t.ref); tk != nil {
			return core.ShortTitle(tk.Description)
		}
		return "任务输出"
	case vConflict:
		return strings.TrimPrefix(t.ref, "") + " (合并冲突)"
	}
	return t.title()
}

// tabBadge 标签上的数字角标(审查 = 待处理的文件数)。
func (a *App) tabBadge(t *ideTab) string {
	if t.virt == vReview {
		if n := a.pendingReviewFiles(); n > 0 {
			return itoa(n)
		}
	}
	return ""
}

// tabIcon 标签图标。
func (a *App) tabIcon(t *ideTab) string {
	switch t.virt {
	case vReview:
		return "eye"
	case vPlan:
		return "list"
	case vSub:
		return "bot"
	case vTask:
		return "terminal"
	case vConflict:
		return "fork"
	}
	switch {
	case t.diff:
		return "diff"
	case strings.HasSuffix(t.path, ".toml"):
		return "gear"
	case strings.HasSuffix(t.path, ".md"):
		return "fileText"
	}
	return "code"
}

// findSub 在时间线里按 id 找子 Agent 卡片(含嵌套)。
func (a *App) findSub(id string) *core.SubagentCard {
	var walk func(items []core.Item) *core.SubagentCard
	walk = func(items []core.Item) *core.SubagentCard {
		for _, it := range items {
			if s, ok := it.(*core.SubagentCard); ok {
				if s.ID == id {
					return s
				}
				if r := walk(s.Children); r != nil {
					return r
				}
			}
		}
		return nil
	}
	return walk(a.st.Items)
}

// findTask 按 id 找后台任务。
func (a *App) findTask(id string) *core.BackgroundTask {
	for _, tk := range a.st.Tasks {
		if tk.TaskID == id {
			return tk
		}
	}
	return nil
}

// latestPlan 时间线里最新的一份计划。
func (a *App) latestPlan() *core.PlanApproval {
	for i := len(a.st.Items) - 1; i >= 0; i-- {
		if p, ok := a.st.Items[i].(*core.PlanApproval); ok {
			return p
		}
	}
	return nil
}

// seedWorkspace 演示 / 截图:摆出设计稿里工作区的初始状态(router.rs / router_test.rs / Cargo.toml / 审查)。
func (a *App) seedWorkspace() {
	if n := len(a.ide.tabs); n > 0 {
		a.ide.active = a.ide.tabs[n-1].id // 审查标签排在末尾(设计稿的默认顺序)
	}
	a.ide.openVirtual(vReview, "", true)
	if a.mode == ModeAgent {
		a.ide.active = virtID(vReview, "")
	} else {
		a.ide.active = "src/router.rs"
	}
	a.memBy[ModeAgent].right = true
	if a.mode == ModeAgent {
		a.rightOn = true
	}
}

// conflictView 合并冲突标签(三方解决,见 conflict.go)。
