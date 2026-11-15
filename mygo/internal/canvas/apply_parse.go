package canvas

import (
	"encoding/json"
	"regexp"
	"strings"
)

var fenceCanvas = regexp.MustCompile("(?s)```canvas\\s*\n(.*?)```")

// ExtractCanvasJSON 从助手回复里取出画布 JSON(```canvas 围栏或裸 JSON)。
func ExtractCanvasJSON(text string) string {
	text = strings.TrimSpace(text)
	if m := fenceCanvas.FindStringSubmatch(text); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if i := strings.Index(text, "{"); i >= 0 {
		if j := strings.LastIndex(text, "}"); j > i {
			cand := text[i : j+1]
			var probe struct {
				Blocks json.RawMessage `json:"blocks"`
			}
			if json.Unmarshal([]byte(cand), &probe) == nil && len(probe.Blocks) > 0 {
				return cand
			}
		}
	}
	return text
}
