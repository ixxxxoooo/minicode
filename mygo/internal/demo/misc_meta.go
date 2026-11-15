package demo

import "github.com/ixxxxoooo/minicode/internal/sessmeta"

// SessMetaPreset 是一条会话整理信息的预置。
type SessMetaPreset struct {
	ID       string
	Title    string
	Pinned   bool
	Archived bool
}

// SessMetaPresets 演示用置顶 / 归档 / 自定义标题(见导航栏会话行与历史弹层)。
func SessMetaPresets() []SessMetaPreset {
	return []SessMetaPreset{
		{ID: "files", Pinned: true},
		{ID: "perm", Pinned: true},
		{ID: "fixbug", Pinned: true},
		{ID: "axum", Archived: true},
		{ID: "long", Archived: true},
		{ID: "refactor", Archived: true},
		{ID: "ci", Archived: true},
		{ID: "browser", Archived: true},
		{ID: "md", Title: "【演示】发布说明终稿"},
		{ID: "side-unread", Title: "✅ 合并冲突已处理(未读)"},
		{ID: "vault-1", Archived: true},
		{ID: "vault-2", Archived: true},
		{ID: "vault-3", Archived: true},
	}
}

// ApplySessMeta 把预置写入 sessmeta 存储。
func ApplySessMeta(m *sessmeta.Store) {
	for _, p := range SessMetaPresets() {
		if p.Title != "" {
			_ = m.SetTitle(p.ID, p.Title)
		}
		if p.Pinned {
			_ = m.SetPinned(p.ID, true)
		}
		if p.Archived {
			_ = m.SetArchived(p.ID, true)
		}
	}
}
