package core

// EvForkSession 通知界面打开一个已配置好的分叉会话(由后端或演示层发出)。
type EvForkSession struct {
	ParentID, NewID string
	Cwd             string
	AtUser          string
}

func (EvForkSession) event() {}
