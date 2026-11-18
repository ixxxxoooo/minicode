package customize

import (
	"bytes"
	"os"
	"strings"
)

// splitFrontmatter 拆 YAML 头与正文;没有头时正文=全文。
func splitFrontmatter(s string) (fm, body string) {
	s = strings.TrimPrefix(s, "\ufeff")
	if !strings.HasPrefix(s, "---") {
		return "", s
	}
	rest := s[3:]
	if i := strings.Index(rest, "\n---"); i < 0 {
		return "", s
	} else {
		fm = strings.TrimSpace(rest[:i])
		body = strings.TrimPrefix(rest[i+4:], "\n")
		return fm, body
	}
}

func joinFrontmatter(fm, body string) string {
	fm = strings.TrimSpace(fm)
	body = strings.TrimRight(body, " \t")
	if fm == "" {
		return body
	}
	var b bytes.Buffer
	b.WriteString("---\n")
	b.WriteString(fm)
	b.WriteString("\n---\n")
	if body != "" {
		b.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// fmLine 从 YAML 头取 key: value(只处理简单一行,无引号嵌套)。
func fmLine(fm, key string) string {
	prefix := key + ":"
	for _, ln := range strings.Split(fm, "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(ln, prefix))
		}
	}
	return ""
}

func setFmLine(fm, key, val string) string {
	prefix := key + ":"
	lines := strings.Split(fm, "\n")
	found := false
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), prefix) {
			lines[i] = key + ": " + val
			found = true
			break
		}
	}
	if !found {
		if fm != "" {
			lines = append(lines, key+": "+val)
		} else {
			lines = []string{key + ": " + val}
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// delFmLine 删掉 YAML 头里的 key 行(没有就原样返回)。
func delFmLine(fm, key string) string {
	prefix := key + ":"
	var out []string
	for _, ln := range strings.Split(fm, "\n") {
		if strings.HasPrefix(strings.TrimSpace(ln), prefix) {
			continue
		}
		out = append(out, ln)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// existingFm 读磁盘上原文件的 YAML 头(文件不存在返回空):保存时以它为底只改界面编辑的字段,
// 其余字段(name / model / color / disallowedTools / 自定义键……)原样保留。此前是从空头重写,保存一次就丢光。
func existingFm(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	fm, _ := splitFrontmatter(string(b))
	return fm
}

// 子 Agent 头里 model / color 与界面下标的对应。
var agentColorNames = []string{"blue", "green", "purple", "cyan", "pink", "orange"}

func agentModelIndex(v string) int {
	v = strings.ToLower(strings.Trim(v, `"' `))
	switch {
	case strings.Contains(v, "sonnet"):
		return 1
	case strings.Contains(v, "opus"):
		return 2
	case strings.Contains(v, "haiku"):
		return 3
	}
	return 0 // 空 / inherit / 具体型号 id:按「继承会话」显示
}

func agentColorIndex(v string) int {
	v = strings.ToLower(strings.Trim(v, `"' `))
	for i, n := range agentColorNames {
		if v == n {
			return i
		}
	}
	switch v {
	case "red", "yellow":
		return 5
	}
	return 0
}
