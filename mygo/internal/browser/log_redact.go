package browser

import (
	"net/url"
	"strings"
)

// RedactURL 去掉查询串与用户信息,供日志与诊断输出(不记录 token)。
func RedactURL(raw string) string {
	u := Normalize(raw)
	if u == "" {
		return ""
	}
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.User = nil
	s := parsed.Host + parsed.Path
	return strings.TrimRight(s, "/")
}
