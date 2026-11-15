package core

// CmdEditUser 就地编辑用户消息后从该条起重发。
// MsgUUID 是对应的 claude 消息 uuid(无 uuid 时真实后端会拒绝)。
type CmdEditUser struct {
	UserID, Text, MsgUUID string
}

func (CmdEditUser) command() {}

// CmdCompact 请求压缩上下文(真实后端一般发 /compact 用户消息)。
type CmdCompact struct{}

func (CmdCompact) command() {}

// CmdRestartTask 请求重新运行一个后台任务(演示后端模拟;真实模式由界面引导)。
type CmdRestartTask struct{ TaskID string }

func (CmdRestartTask) command() {}
