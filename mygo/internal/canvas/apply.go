package canvas

import (
	"encoding/json"
	"strings"
	"time"
)

// ApplyClaudeReply 从 Claude 回复里提取画布 JSON 并合并进当前文档;成功则提交新版本。
// 支持整份 {"title","status","blocks":[…]} 或仅 blocks 数组。
func (s *Store) ApplyClaudeReply(text, by, note string, at time.Time) (bool, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return false, "回复为空"
	}
	// 围栏代码块里的 JSON
	if i := strings.Index(text, "{"); i >= 0 {
		if j := strings.LastIndex(text, "}"); j > i {
			text = text[i : j+1]
		}
	}
	var raw struct {
		Title  string          `json:"title"`
		Status string          `json:"status"`
		Blocks json.RawMessage `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return false, "未找到可解析的画布 JSON"
	}
	if raw.Title != "" {
		s.Doc.Title = raw.Title
	}
	if raw.Status != "" {
		s.Doc.Status = raw.Status
	}
	if len(raw.Blocks) > 0 {
		var blocks []Block
		if err := json.Unmarshal(raw.Blocks, &blocks); err == nil && len(blocks) > 0 {
			s.Doc.Blocks = blocks
		}
	}
	if !s.Dirty() {
		return false, "内容与当前版本相同"
	}
	s.Commit(by, note, at)
	return true, ""
}
