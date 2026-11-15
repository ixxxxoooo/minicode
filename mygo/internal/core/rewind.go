package core

import (
	"fmt"
	"strings"
)

// RewindOutcome 是一次回退请求的结果(预览或实际执行)。
type RewindOutcome struct {
	UUID       string   // 目标用户消息的 uuid
	DryRun     bool     // 是否只是预览
	CanRewind  bool     // 能否回退(没有检查点、或者请求失败时为假)
	Files      []string // 会被恢复的文件(只有预览时 claude 才给出)
	Insertions int      // 预览:回退会恢复的行数
	Deletions  int      // 预览:回退会去掉的行数
	Error      string
}

// RewindFailed 构造失败结果。
func RewindFailed(uuid string, dryRun bool, err string) RewindOutcome {
	return RewindOutcome{UUID: uuid, DryRun: dryRun, Error: err}
}

// Reason 给用户看的失败原因(把 claude 的英文原因翻成人话,其余原样)。
func (o RewindOutcome) Reason() string {
	switch {
	case strings.Contains(o.Error, "No file checkpoint"):
		return "这条消息没有文件检查点(可能没改过文件,或会话不是在开启检查点后发出的)"
	case o.Error != "":
		return o.Error
	}
	return "没有可回退的改动"
}

// HasChanges 预览是否有实际可恢复的改动。
func (o RewindOutcome) HasChanges() bool {
	return o.CanRewind && (len(o.Files) > 0 || o.Insertions > 0 || o.Deletions > 0)
}

// RewindStage 是回退确认卡所处的阶段。
type RewindStage int

const (
	RewindChecking RewindStage = iota // 已发出预览请求,等待结果
	RewindReady                       // 拿到预览结果,等用户确认
	RewindApplying                    // 已确认,正在回退
)

// RewindPrompt 是回退确认卡的状态(界面一次只确认一条)。
type RewindPrompt struct {
	Stage   RewindStage
	ItemID  string
	UUID    string
	Outcome RewindOutcome // Ready 阶段的预览
	Files   int           // Applying 阶段:预览里的文件数
}

// RewindBegin 用户点了某条消息的「回退文件」:进入预览状态。返回要发预览请求的 uuid;不能回退时返回原因。
func (s *State) RewindBegin(itemID string) (string, error) {
	if s.Phase == PhaseRunning {
		return "", fmt.Errorf("回合进行中,结束或中断后再回退")
	}
	uuid, ok := s.UserUUIDs[itemID]
	if !ok {
		return "", fmt.Errorf("这条消息没有检查点记录")
	}
	s.Rewind = &RewindPrompt{Stage: RewindChecking, ItemID: itemID, UUID: uuid}
	return uuid, nil
}

// RewindConfirm 用户在确认卡上点了「回退」。返回要执行回退的 uuid;预览没有改动或状态不对时返回 false。
func (s *State) RewindConfirm() (string, bool) {
	r := s.Rewind
	if r == nil || r.Stage != RewindReady || !r.Outcome.HasChanges() || s.Phase == PhaseRunning {
		return "", false
	}
	s.Rewind = &RewindPrompt{Stage: RewindApplying, ItemID: r.ItemID, UUID: r.Outcome.UUID, Files: len(r.Outcome.Files)}
	return r.Outcome.UUID, true
}

// RewindCancel 关掉确认卡(取消,或看完「没有可回退的改动」)。
func (s *State) RewindCancel() { s.Rewind = nil }

// rewindResult 归约后端的回退结果:预览 → 等确认;执行完 → 关卡并在时间线末尾留一条提示。过期的结果忽略。
func (s *State) rewindResult(o RewindOutcome) {
	r := s.Rewind
	switch {
	case r != nil && r.Stage == RewindChecking && o.DryRun && r.UUID == o.UUID:
		s.Rewind = &RewindPrompt{Stage: RewindReady, ItemID: r.ItemID, UUID: o.UUID, Outcome: o}
	case r != nil && r.Stage == RewindApplying && !o.DryRun && r.UUID == o.UUID:
		s.Rewind = nil
		what := s.userExcerpt(r.ItemID)
		level, text := NoticeInfo, fmt.Sprintf("已把 %d 个文件回退到「%s」发出之前(对话记录保留不变)", r.Files, what)
		if !o.CanRewind || o.Error != "" {
			level, text = NoticeWarning, "回退失败:"+o.Reason()
		}
		s.Apply(EvNotice{Level: level, Text: text})
	}
}

// userExcerpt 用户消息的前 24 个字符(提示里指代这条消息)。
func (s *State) userExcerpt(itemID string) string {
	text := "这条消息"
	for _, it := range s.Items {
		if u, ok := it.(*User); ok && u.ID == itemID {
			text = u.Text
			break
		}
	}
	one := strings.Join(strings.Fields(text), " ")
	if r := []rune(one); len(r) > 24 {
		return string(r[:24]) + "…"
	}
	return one
}
