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

// TruncateFromUser 去掉该用户消息之后的条目,保留该用户消息(就地编辑用)。
func (s *State) TruncateFromUser(userID string) (text string, ok bool) {
	for i, it := range s.Items {
		if u, y := it.(*User); y && u.ID == userID {
			text = u.Text
			s.Items = s.Items[:i+1]
			s.cleanupAfterTruncate(userID)
			s.Rev++
			return text, true
		}
	}
	return "", false
}

// TruncateBeforeUser 从该用户消息起截断(含该条),此前内容保留(分叉等)。
func (s *State) TruncateBeforeUser(userID string) (text string, ok bool) {
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
	for id := range s.Timings {
		if id != keepUser {
			delete(s.Timings, id)
		}
	}
	s.Rewind = nil
	s.Phase = PhaseIdle
}
