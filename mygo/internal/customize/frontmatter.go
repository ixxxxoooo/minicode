package customize

import (
	"bytes"
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
