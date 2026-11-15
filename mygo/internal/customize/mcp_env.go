package customize

import (
	"fmt"
	"strings"
)

// ParseEnvLines 解析多行 KEY=VALUE;值可留空。
func ParseEnvLines(text string) (map[string]string, error) {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i <= 0 {
			return nil, fmt.Errorf("环境变量格式错误(需要 KEY=VALUE):%s", line)
		}
		k := strings.TrimSpace(line[:i])
		if k == "" {
			return nil, fmt.Errorf("环境变量名不能为空")
		}
		out[k] = strings.TrimSpace(line[i+1:])
	}
	return out, nil
}
