package core

// 会话流相关的界面命令(重试回合、分叉会话、编辑后重发等)。

// CmdRetryTurn 用同一条用户消息重新发起回合(等于再发一次该提示)。
type CmdRetryTurn struct{ UserID string }

func (CmdRetryTurn) command() {}

// CmdForkSession 从当前会话分叉出一个新会话(真实后端用 claude --fork-session)。
// ParentID 为被分叉的会话 id;NewID 为新会话 id(调用方生成);AtUser 为用户消息条目 id(预留,供后续按消息分叉)。
type CmdForkSession struct {
	ParentID, NewID, AtUser string
}

func (CmdForkSession) command() {}

// TruncateFromUser 从某条用户消息起截断时间线(就地编辑 / 分叉演示用)。返回是否找到该用户消息。
func (s *State) TruncateFromUser(userID string) (text string, ok bool) {
	for i, it := range s.Items {
		if u, y := it.(*User); y && u.ID == userID {
			text = u.Text
			s.Items = s.Items[:i]
			s.cleanupAfterTruncate(userID)
			s.Rev++
			return text, true
		}
	}
	return "", false
}

// cleanupAfterTruncate 截断后清理悬空统计与回退状态。
func (s *State) cleanupAfterTruncate(keepUser string) {
	for id := range s.TurnStats {
		if id != keepUser {
			delete(s.TurnStats, id)
		}
	}
	for id := range s.Durations {
		if id != keepUser {
			delete(s.Durations, id)
		}
	}
	s.Rewind = nil
	s.Phase = PhaseIdle
}
