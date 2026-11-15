package claude

// ForkConfig 从已有会话分叉出新会话的启动参数,对应 claude --fork-session --resume <父> --session-id <新>。
type ForkConfig struct {
	ParentSessionID string
	NewSessionID    string
}

// Apply 写入 Options(Resume + Fork + 新 session id)。
func (f ForkConfig) Apply(o *Options) {
	if f.ParentSessionID == "" {
		return
	}
	o.Resume = f.ParentSessionID
	o.Fork = true
	if f.NewSessionID != "" {
		o.SessionID = f.NewSessionID
	}
}

// ApplyConfig 写入 Backend 构造用的 Config:新会话 id、无历史回放路径。
func (f ForkConfig) ApplyConfig(cfg *Config) {
	if f.ParentSessionID == "" {
		return
	}
	if f.NewSessionID != "" {
		cfg.SessionID = f.NewSessionID
	}
	cfg.HistoryPath = ""
	cfg.Fork = &f
}

// NewSessionID 生成 UUID 风格的新会话 id(与 backend.newUUID 一致)。
func NewSessionID() string { return newUUID() }
