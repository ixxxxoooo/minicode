package claude

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// TestRealSteerAndQueue 回合进行中发引导与普通排队,确认协议带 priority:next(无后端显式接收确认)。
func TestRealSteerAndQueue(t *testing.T) {
	realSkip(t)
	mode := core.ModePlan
	r := newRealRig(t, Config{Mode: &mode, Cwd: t.TempDir()})

	r.b.Send(core.CmdPrompt{Text: "先输出正在思考三个字(不要工具),然后 sleep 用 Bash 前台执行 sleep 8 秒(若不能则只输出思考三字)。"})
	r.waitTimeout(30*time.Second, "回合开始", func() bool {
		return r.st.Phase == core.PhaseRunning || r.turnsEndedLocked() >= 1
	})

	// 若回合已结束则无法测 steer
	r.mu.Lock()
	running := r.turnsEndedLocked() == 0
	r.mu.Unlock()
	if !running {
		t.Skip("回合结束太快,无法测引导/排队")
	}

	r.b.Send(core.CmdPrompt{Text: "引导:改成只回复 steer", Steer: true})
	r.b.Send(core.CmdPrompt{Text: "排队:稍后回复 queue"})
	time.Sleep(2 * time.Second)
	r.mu.Lock()
	users := 0
	for _, it := range r.st.Items {
		if _, ok := it.(*core.User); ok {
			users++
		}
	}
	r.mu.Unlock()
	if users < 3 {
		t.Errorf("应有原消息+引导+排队,实际用户条数 %d", users)
	}
	t.Log("结论:后端无 steer 接收确认事件;界面需用启发式(下一条非用户条目出现后判接收)")

	r.b.Send(core.CmdCancel{})
	r.waitTimeout(60*time.Second, "中断后收尾", func() bool { return r.turnsEndedLocked() >= 1 })
}

// TestRealCrashResume kill 子进程后下一条消息应自动 --resume 且历史仍在。
func TestRealCrashResume(t *testing.T) {
	realSkip(t)
	cwd := t.TempDir()
	r, _ := realRigWithCleanup(t, Config{Cwd: cwd, Mode: modePtr(core.ModePlan)})

	r.b.Send(core.CmdPrompt{Text: "只回复 marker1"})
	r.waitTimeout(90*time.Second, "首回合", func() bool { return r.turnsEndedLocked() >= 1 })
	skipIfLimitReply(t, r)

	marker := replyOf(r)
	r.b.KillChildForTest()
	time.Sleep(800 * time.Millisecond)

	r.b.Send(core.CmdPrompt{Text: "只回复 marker2"})
	r.waitTimeout(90*time.Second, "恢复后第二回合", func() bool { return r.turnsEndedLocked() >= 2 })

	r.mu.Lock()
	defer r.mu.Unlock()
	if !strings.Contains(marker, "marker1") && !strings.Contains(strings.ToLower(marker), "ok") {
		t.Logf("首回合回复:%q", marker)
	}
	assist := 0
	for _, it := range r.st.Items {
		if _, ok := it.(*core.Assistant); ok {
			assist++
		}
	}
	if assist < 2 {
		t.Errorf("恢复后应至少两条助手回复,实际 %d", assist)
	}
}

// TestRealEditUserAndFork 两条消息后编辑与分叉:检查 uuid、resume_session_at、--fork-session。
func TestRealEditUserAndFork(t *testing.T) {
	realSkip(t)
	cwd := t.TempDir()
	parent, _ := realRigWithCleanup(t, Config{Cwd: cwd, Mode: modePtr(core.ModePlan)})

	parent.b.Send(core.CmdPrompt{Text: "只回复 one"})
	parent.waitTimeout(90*time.Second, "回合1", func() bool { return parent.turnsEndedLocked() >= 1 })
	parent.b.Send(core.CmdPrompt{Text: "只回复 two"})
	parent.waitTimeout(90*time.Second, "回合2", func() bool { return parent.turnsEndedLocked() >= 2 })
	skipIfLimitReply(t, parent)

	uuids := parent.userUUIDs()
	if len(uuids) == 0 {
		parent.b.Send(core.CmdEditUser{UserID: "u1", Text: "edited", MsgUUID: ""})
		parent.waitTimeout(5*time.Second, "无 uuid 编辑", func() bool {
			return parent.hasEventLocked(func(e core.Event) bool {
				n, ok := e.(core.EvNotice)
				return ok && strings.Contains(n.Text, "检查点")
			})
		})
	} else {
		parent.b.Send(core.CmdEditUser{UserID: uuids[0].ID, Text: "edited-one", MsgUUID: uuids[0].UUID})
		parent.waitTimeout(120*time.Second, "编辑后重发", func() bool { return parent.turnsEndedLocked() >= 3 })
	}

	parentID := parent.b.SessionID()
	newID := NewSessionID()
	child := New(Config{
		Cwd:  cwd,
		Fork: &ForkConfig{ParentSessionID: parentID, NewSessionID: newID},
	})
	var childRig rig
	childRig.t, childRig.st = t, core.NewState()
	childRig.cond = sync.NewCond(&childRig.mu)
	childRig.b = child
	childRig.b.Start(func(e core.Event) {
		childRig.mu.Lock()
		childRig.st.Apply(e)
		childRig.evs = append(childRig.evs, e)
		childRig.cond.Broadcast()
		childRig.mu.Unlock()
	})
	t.Cleanup(child.Close)

	child.Send(core.CmdPrompt{Text: "只回复 forked"})
	childRig.waitTimeout(90*time.Second, "分叉会话首回合", func() bool {
		return childRig.turnsEndedLocked() >= 1
	})

	if child.SessionID() != newID {
		t.Errorf("分叉会话 id=%s want %s", child.SessionID(), newID)
	}
	proj := ProjectsDir()
	if !SessionOnDisk(proj, cwd, parentID) {
		t.Error("父会话 jsonl 应仍在磁盘")
	}
	if !SessionOnDisk(proj, cwd, newID) {
		t.Log("新会话可能尚未落盘(首回合后常见)")
	}
	_ = sessionJSONL(proj, cwd, parentID)
	_ = filepath.Join(proj, EncodeProjectDir(cwd), newID+".jsonl")
}

// TestRealPermissionAndQuestion 触发写文件权限与 AskUserQuestion(依赖模型配合,失败则 Skip)。
func TestRealPermissionAndQuestion(t *testing.T) {
	realSkip(t)
	cwd := t.TempDir()
	r := newRealRig(t, Config{Cwd: cwd, Mode: modePtr(core.ModeDefault)})

	r.b.Send(core.CmdPrompt{Text: `先调用 AskUserQuestion 问我「颜色」单选:红/蓝;
然后用 Write 工具在 cwd 创建 answer.txt 内容为 red。`})

	deadline := time.Now().Add(180 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		if r.turnsEndedLocked() >= 1 {
			r.mu.Unlock()
			break
		}
		for _, it := range r.st.Items {
			if q, ok := it.(*core.QuestionRequest); ok && !q.Answered {
				id := q.ID
				r.mu.Unlock()
				r.b.Send(core.CmdAnswerQuestion{ID: id, Answers: [][]string{{"红"}}})
				goto nextPoll
			}
			if p, ok := it.(*core.PermissionRequest); ok && p.Resolved == "" {
				id := p.ID
				r.mu.Unlock()
				r.b.Send(core.CmdResolvePermission{ID: id, Option: "allow_once"})
				goto nextPoll
			}
		}
		r.mu.Unlock()
	nextPoll:
		time.Sleep(200 * time.Millisecond)
	}
	r.mu.Lock()
	ended := r.turnsEndedLocked()
	evn := len(r.evs)
	r.mu.Unlock()
	if ended < 1 {
		t.Fatalf("等待回合结束超时,事件数 %d", evn)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	var perm, plan, ask int
	for _, it := range r.st.Items {
		switch it.(type) {
		case *core.PermissionRequest:
			perm++
		case *core.PlanApproval:
			plan++
		case *core.QuestionRequest:
			ask++
		}
	}
	if perm == 0 && ask == 0 {
		t.Skip("模型未触发权限或问答")
	}
	t.Logf("权限请求 %d, 计划卡 %d", perm, plan)
}

func modePtr(m core.PermissionMode) *core.PermissionMode { return &m }
