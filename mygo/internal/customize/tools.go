package customize

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SplitTools 把 frontmatter 里的 allowed-tools 拆成一条条工具规则。
// 逗号与空白都是分隔符,但括号里的不拆:`Bash(git log --oneline), Read` → [Bash(git log --oneline) Read]。
func SplitTools(s string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	flush := func() {
		if t := strings.TrimSpace(cur.String()); t != "" {
			out = append(out, t)
		}
		cur.Reset()
	}
	for _, r := range s {
		switch {
		case r == '(':
			depth++
			cur.WriteRune(r)
		case r == ')':
			if depth > 0 {
				depth--
			}
			cur.WriteRune(r)
		case depth == 0 && (r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'):
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

var toolRuleRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.:\-]*(\(.+\))?$`)

// ValidateToolRule 检查一条工具规则的写法(与权限规则同语法):
// `Read`、`Bash(git status:*)`、`mcp__github`、`mcp__github__create_issue`。返回给人看的原因,合法返回空串。
func ValidateToolRule(rule string) string {
	rule = strings.TrimSpace(rule)
	switch {
	case rule == "":
		return "规则不能为空"
	case strings.ContainsAny(rule, ",\n"):
		return "一次只能添加一条规则,逗号请分开添加"
	case strings.Count(rule, "(") != strings.Count(rule, ")"):
		return "括号没有配对,例如 Bash(git status:*)"
	case strings.Contains(rule, "()"):
		return "括号里不能为空;不限定参数请直接写工具名"
	case !toolRuleRe.MatchString(rule):
		return "写法不对:工具名只能是字母 / 数字 / 下划线,参数写在括号里,例如 Bash(git diff:*)"
	}
	return ""
}

// ToolBase 取规则的工具名部分(`Bash(git:*)` → `Bash`)。
func ToolBase(rule string) string {
	if i := strings.IndexByte(rule, '('); i > 0 {
		return rule[:i]
	}
	return rule
}

// HasAllow 是否已允许了这条规则。
func (it *Item) HasAllow(rule string) bool {
	for _, t := range it.Allow {
		if t == rule {
			return true
		}
	}
	return false
}

// AddAllow 添加一条工具规则(已存在返回 false)。内置工具按固定顺序排前面,其余按添加顺序跟在后面。
func (it *Item) AddAllow(rule string) bool {
	rule = strings.TrimSpace(rule)
	if rule == "" || it.HasAllow(rule) {
		return false
	}
	it.Allow = append(it.Allow, rule)
	sort.SliceStable(it.Allow, func(a, b int) bool { return toolIx(it.Allow[a]) < toolIx(it.Allow[b]) })
	return true
}

// MCPToolRules 列出已知 MCP 服务器可选的工具规则:每台服务器一条整台(`mcp__名`),再加上它已列出的各个工具。
func (s *Store) MCPToolRules() []string {
	var out []string
	for _, it := range s.List(MCP, All, "") {
		base := "mcp__" + sanitizeMCPName(it.Name)
		out = append(out, base)
		for _, tl := range it.Tools {
			out = append(out, base+"__"+tl.Name)
		}
	}
	return out
}

// sanitizeMCPName 与 claude 的命名一致:非字母数字下划线连字符的字符换成下划线。
func sanitizeMCPName(n string) string {
	var b strings.Builder
	for _, r := range n {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// ToolSearchValue 读取按需加载工具(ENABLE_TOOL_SEARCH)的当前取值与它定义在哪一层(0 用户 / 1 项目 / 2 本地,-1 没设置)。
// 优先级与 Claude 一致:本地 > 项目 > 用户。没设置按「开」(自动按需)处理。
func (ds *DiskStore) ToolSearchValue() (on bool, layer int) {
	on, layer = true, -1
	paths := []string{filepath.Join(ds.home, ".claude", "settings.json")}
	if ds.proj != "" {
		paths = append(paths, filepath.Join(ds.proj, ".claude", "settings.json"), filepath.Join(ds.proj, ".claude", "settings.local.json"))
	}
	for i, p := range paths {
		var v map[string]any
		if !readJSON(p, &v) {
			continue
		}
		env, _ := v["env"].(map[string]any)
		raw, ok := env["ENABLE_TOOL_SEARCH"]
		if !ok {
			continue
		}
		layer = i
		switch strings.ToLower(strings.TrimSpace(fmt.Sprint(raw))) {
		case "false", "0", "off", "no":
			on = false
		default:
			on = true
		}
	}
	return on, layer
}
