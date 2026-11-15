package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{ID: "c-todo-all", Title: "待办 · 全部待办", Label: "刚刚", Build: buildTodoAll})
	Register(Scenario{ID: "c-todo-active", Title: "待办 · 进行中 ActiveForm", Label: "刚刚", Build: buildTodoActive})
	Register(Scenario{ID: "c-todo-part", Title: "待办 · 部分完成", Label: "刚刚", Build: buildTodoPartial})
	Register(Scenario{ID: "c-todo-done", Title: "待办 · 全部完成", Label: "刚刚", Build: buildTodoDone})
	Register(Scenario{ID: "c-todo-long", Title: "待办 · 很长清单 30 项", Label: "刚刚", Build: buildTodoLong})
	Register(Scenario{ID: "c-todo-replace", Title: "待办 · 清单多次替换", Label: "刚刚", Live: true, Build: buildTodoReplace})
}

func buildTodoAll() []Step {
	s := NewScript("c-todo-all", "待办 · 全部待办")
	uid := s.User("初始清单")
	tid := s.ID("td")
	s.Add(ms(120), core.EvAppend{Item: &core.Todos{ID: tid}})
	s.Add(ms(80), core.EvTodos{ID: tid, Items: []core.TodoItem{
		{Content: "读 router.rs", Status: core.TodoPending},
		{Content: "统一 normalize", Status: core.TodoPending},
		{Content: "补测试", Status: core.TodoPending},
	}})
	finishLab(s, uid, 12)
	return s.Steps()
}

func buildTodoActive() []Step {
	s := NewScript("c-todo-active", "待办 · 进行中 ActiveForm")
	uid := s.User("进行中")
	s.Add(ms(120), core.EvTodos{ID: s.ID("td"), Items: []core.TodoItem{
		{Content: "定位大小写敏感的来源", Status: core.TodoCompleted},
		{Content: "实现 normalize", Status: core.TodoInProgress, ActiveForm: "编写 normalize 函数"},
		{Content: "跑通测试", Status: core.TodoPending},
	}})
	finishLab(s, uid, 13)
	return s.Steps()
}

func buildTodoPartial() []Step {
	s := NewScript("c-todo-part", "待办 · 部分完成")
	uid := s.User("部分完成")
	s.Add(ms(120), core.EvTodos{ID: s.ID("td"), Items: []core.TodoItem{
		{Content: "读文件", Status: core.TodoCompleted},
		{Content: "改路由", Status: core.TodoCompleted},
		{Content: "写 CHANGELOG", Status: core.TodoPending},
		{Content: "截图对比", Status: core.TodoPending},
	}})
	finishLab(s, uid, 14)
	return s.Steps()
}

func buildTodoDone() []Step {
	s := NewScript("c-todo-done", "待办 · 全部完成")
	uid := s.User("全部完成")
	s.Add(ms(120), core.EvTodos{ID: s.ID("td"), Items: []core.TodoItem{
		{Content: "定位问题", Status: core.TodoCompleted},
		{Content: "提交修复", Status: core.TodoCompleted},
		{Content: "合并 PR", Status: core.TodoCompleted},
	}})
	finishLab(s, uid, 15)
	return s.Steps()
}

func buildTodoLong() []Step {
	s := NewScript("c-todo-long", "待办 · 很长清单 30 项")
	uid := s.User("30 项")
	s.Add(ms(120), core.EvTodos{ID: s.ID("td"), Items: todoLong30()})
	finishLab(s, uid, 16)
	return s.Steps()
}

func buildTodoReplace() []Step {
	s := NewScript("c-todo-replace", "待办 · 清单多次替换")
	uid := s.User("替换")
	tid := s.ID("td")
	s.Add(ms(120), core.EvTodos{ID: tid, Items: []core.TodoItem{
		{Content: "草案 A:只改 router", Status: core.TodoInProgress, ActiveForm: "评估草案 A"},
		{Content: "草案 B:加开关", Status: core.TodoPending},
	}})
	s.Add(ms(900), core.EvTodos{ID: tid, Items: []core.TodoItem{
		{Content: "采用草案 A", Status: core.TodoCompleted},
		{Content: "删除废弃分支", Status: core.TodoInProgress, ActiveForm: "清理分支"},
	}})
	s.Add(ms(900), core.EvTodos{ID: tid, Items: []core.TodoItem{
		{Content: "合并主分支", Status: core.TodoCompleted},
		{Content: "发布补丁", Status: core.TodoCompleted},
	}})
	s.Say("清单已随决策多次替换。")
	finishLab(s, uid, 17)
	return s.Steps()
}
