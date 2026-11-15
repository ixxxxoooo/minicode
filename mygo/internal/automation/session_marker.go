package automation

import "strings"

const sessionMarkerPrefix = "\n\x1eMC_SESS:"

// EmbedSession 在输出末尾附加会话 id 标记(仅运行器使用,界面落盘前须去掉)。
func EmbedSession(output, sessionID string) string {
	if sessionID == "" {
		return output
	}
	return output + sessionMarkerPrefix + sessionID + "\x1e"
}

// SplitSession 从运行输出中拆出正文与会话 id(兼容无标记的旧记录)。
func SplitSession(output string) (text, sessionID string) {
	i := strings.LastIndex(output, sessionMarkerPrefix)
	if i < 0 {
		return output, ""
	}
	text = output[:i]
	rest := output[i+len(sessionMarkerPrefix):]
	if j := strings.IndexByte(rest, '\x1e'); j >= 0 {
		sessionID = rest[:j]
	}
	return text, sessionID
}
