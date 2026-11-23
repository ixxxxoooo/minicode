// Package cmdtitle 把一条 shell 命令提炼成一句简短的中文步骤标题。纯函数,不依赖界面。
//
// Claude 调用 Bash 时不一定写 description(有的会话、有的调用没有),这时步骤行标题退回命令本身,
// 一长串 cd /private/tmp/... && ls ... 很难读。Title 先去掉开头的 cd / 变量赋值 / export 之类的「铺垫」,
// 再按第一条真正的命令套规则(查看 / 列出 / 搜索 / Git / 编译 / 测试…);
// 认不出来的返回清理掉铺垫后的命令本身。完整命令始终在悬停提示、展开的终端卡头部和复制里。
//
// @author ixxxxoooo
package cmdtitle

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Title 返回命令 cmd 的步骤行标题。参数 cmd 是完整命令(可多行、可 && 串联)。
// 返回:标题文字,永不为空(cmd 为空才返回空串)。
func Title(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	stages := cmdStages(cmd)
	var real [][]string
	for _, st := range stages {
		if toks := cmdTokens(st); len(toks) > 0 && !cmdPrelude(toks) {
			real = append(real, toks)
		}
	}
	if len(real) == 0 {
		return shortLine(cmd, 80)
	}
	title := cmdRule(real[0])
	if title == "" {
		title = shortLine(cmdCleanStage(stages, real[0]), 80)
	}
	if extra := len(real) - 1; extra > 0 {
		title += fmt.Sprintf(" 等 %d 步", len(real))
	}
	return title
}

// cmdStages 把命令按 && ; 换行 切成几段(管道 | 之后的部分丢掉:管道是同一个动作的后处理)。
func cmdStages(cmd string) []string {
	var out []string
	var cur strings.Builder
	q := rune(0)
	rs := []rune(cmd)
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case q != 0:
			if r == q {
				q = 0
			}
			cur.WriteRune(r)
		case r == '\'' || r == '"':
			q = r
			cur.WriteRune(r)
		case r == '\n' || r == ';':
			flush()
		case r == '&' && i+1 < len(rs) && rs[i+1] == '&':
			flush()
			i++
		case r == '|' && i+1 < len(rs) && rs[i+1] == '|':
			flush()
			i++
		case r == '|':
			// 管道:这一段到此为止,后面的部分(head / grep 过滤)是后处理
			for i+1 < len(rs) && rs[i+1] != '\n' && rs[i+1] != ';' && !(rs[i+1] == '&' && i+2 < len(rs) && rs[i+2] == '&') {
				i++
			}
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

// cmdTokens 按空白切词,尊重单 / 双引号(引号本身去掉),开头的 VAR=value 赋值丢掉。
func cmdTokens(s string) []string {
	var toks []string
	var cur strings.Builder
	q := rune(0)
	has := false
	for _, r := range s {
		switch {
		case q != 0:
			if r == q {
				q = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			q, has = r, true
		case r == ' ' || r == '\t':
			if has || cur.Len() > 0 {
				toks = append(toks, cur.String())
				cur.Reset()
				has = false
			}
		default:
			cur.WriteRune(r)
		}
	}
	if has || cur.Len() > 0 {
		toks = append(toks, cur.String())
	}
	for len(toks) > 0 && isAssign(toks[0]) {
		toks = toks[1:]
	}
	return toks
}

func isAssign(t string) bool {
	i := strings.IndexByte(t, '=')
	if i <= 0 {
		return false
	}
	for _, r := range t[:i] {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// cmdPrelude 报告这一段是不是「铺垫」(不算动作):cd / export / set / source / 纯赋值等。
func cmdPrelude(toks []string) bool {
	switch toks[0] {
	case "cd", "export", "set", "unset", "source", ".", "true", "false", ":", "pushd", "popd", "umask", "alias", "local", "readonly":
		return true
	}
	return false
}

// cmdCleanStage 取原命令里第一条真实命令那一段的原文(去掉铺垫后的样子)。
func cmdCleanStage(stages []string, real []string) string {
	for _, st := range stages {
		if t := cmdTokens(st); len(t) > 0 && len(t) == len(real) && t[0] == real[0] && !cmdPrelude(t) {
			return st
		}
	}
	return strings.Join(real, " ")
}

func shortLine(s string, n int) string {
	s = strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// firstArg 取第一个不以 - 开头的参数。
func firstArg(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

// lastArg 取最后一个不以 - 开头的参数。
func lastArg(args []string) string {
	for i := len(args) - 1; i >= 0; i-- {
		if !strings.HasPrefix(args[i], "-") {
			return args[i]
		}
	}
	return ""
}

func baseOf(p string) string {
	p = strings.TrimRight(p, "/")
	if p == "" {
		return "/"
	}
	return filepath.Base(p)
}

// cmdRule 按命令名套规则返回标题;认不出来返回空串。参数 t 是已切好词的一条命令(t[0] 是命令名)。
func cmdRule(t []string) string {
	name := filepath.Base(t[0])
	args := t[1:]
	sub := firstArg(args)
	switch name {
	case "cat", "head", "tail", "less", "more", "bat":
		if f := lastArg(args); f != "" {
			return "查看 " + baseOf(f)
		}
		return "查看文件"
	case "ls", "ll", "la", "tree", "exa", "eza":
		if d := lastArg(args); d != "" {
			return "列出 " + baseOf(d)
		}
		return "列出目录"
	case "grep", "egrep", "fgrep", "rg", "ag", "ack":
		if sub != "" {
			return "搜索 " + shortLine(sub, 30)
		}
		return "搜索"
	case "find", "fd", "locate", "mdfind":
		return "查找文件"
	case "wc":
		return "统计行数"
	case "diff", "cmp":
		return "对比文件"
	case "sed", "awk", "tr", "cut", "sort", "uniq", "xargs":
		return "处理文本"
	case "jq", "yq":
		return "解析 JSON / YAML"
	case "echo", "printf":
		return "输出文字"
	case "pwd":
		return "查看当前目录"
	case "date":
		return "查看时间"
	case "which", "type", "command", "whereis":
		return "查找命令 " + sub
	case "env", "printenv":
		return "查看环境变量"
	case "sleep":
		if sub != "" {
			return "等待 " + sub + " 秒"
		}
	case "rm", "rmdir", "trash":
		if f := lastArg(args); f != "" {
			return "删除 " + baseOf(f)
		}
		return "删除文件"
	case "mkdir":
		if f := lastArg(args); f != "" {
			return "创建目录 " + baseOf(f)
		}
	case "touch":
		if f := lastArg(args); f != "" {
			return "创建文件 " + baseOf(f)
		}
	case "cp":
		return "复制 " + baseOf(firstArg(args))
	case "mv":
		return "移动 " + baseOf(firstArg(args))
	case "chmod", "chown", "chgrp", "xattr":
		return "修改文件属性"
	case "tar", "zip", "unzip", "gzip", "gunzip", "7z":
		return "打包 / 解压"
	case "curl", "wget", "http", "https":
		if u := firstURL(args); u != "" {
			return "请求 " + u
		}
		return "发送网络请求"
	case "open":
		if f := lastArg(args); f != "" {
			return "打开 " + baseOf(f)
		}
	case "ps", "pgrep", "lsof", "top", "pstree":
		return "查看进程"
	case "kill", "pkill", "killall":
		return "结束进程"
	case "git":
		return gitTitle(args)
	case "go":
		return toolTitle("Go", sub, map[string]string{"build": "编译", "test": "测试", "vet": "检查", "run": "运行", "mod": "管理依赖", "fmt": "格式化", "install": "安装", "get": "获取依赖", "generate": "生成代码"})
	case "gofmt":
		return "Go 格式化"
	case "cargo":
		return toolTitle("Rust", sub, map[string]string{"build": "编译", "test": "测试", "check": "检查", "run": "运行", "clippy": "Clippy 检查", "fmt": "格式化", "add": "添加依赖", "update": "更新依赖"})
	case "npm", "pnpm", "yarn", "bun":
		switch sub {
		case "install", "i", "add", "ci":
			return "安装依赖"
		case "test", "t":
			return "运行测试"
		case "run", "run-script":
			if s := nth(args, 1); s != "" {
				return name + " run " + s
			}
		case "build", "dev", "start", "lint":
			return name + " " + sub
		}
		return name + " " + sub
	case "make", "just", "gradle", "mvn", "cmake", "ninja":
		return name + " " + sub
	case "python", "python3", "python2", "node", "deno", "ruby", "perl", "bash", "sh", "zsh", "swift", "php", "lua":
		for _, a := range args {
			if a == "-c" || a == "-e" {
				return "运行 " + name + " 代码"
			}
		}
		if sub != "" {
			return "运行 " + baseOf(sub)
		}
	case "xcodebuild":
		return "Xcode 构建"
	case "brew":
		return "Homebrew " + sub
	case "docker", "podman":
		return "Docker " + sub
	case "ssh", "scp", "rsync":
		return name + " 远程操作"
	case "osascript":
		return "运行 AppleScript"
	case "codesign", "hdiutil", "lipo", "otool", "install_name_tool", "iconutil", "sips":
		return name
	}
	if strings.HasPrefix(t[0], "./") || strings.HasPrefix(t[0], "/") {
		return "运行 " + baseOf(t[0])
	}
	return ""
}

func nth(a []string, i int) string {
	if i < len(a) {
		return a[i]
	}
	return ""
}

func firstURL(args []string) string {
	for _, a := range args {
		if strings.HasPrefix(a, "http://") || strings.HasPrefix(a, "https://") {
			a = strings.TrimPrefix(strings.TrimPrefix(a, "https://"), "http://")
			if i := strings.IndexAny(a, "/?"); i >= 0 {
				a = a[:i]
			}
			return a
		}
	}
	return ""
}

func toolTitle(lang, sub string, m map[string]string) string {
	if v, ok := m[sub]; ok {
		return lang + " " + v
	}
	if sub == "" {
		return lang
	}
	return lang + " " + sub
}

func gitTitle(args []string) string {
	// 跳过 git 的全局选项(-C dir、-c k=v)
	for i := 0; i < len(args); i++ {
		if args[i] == "-C" || args[i] == "-c" {
			i++
			continue
		}
		if strings.HasPrefix(args[i], "-") {
			continue
		}
		switch sub := args[i]; sub {
		case "status":
			return "查看 Git 状态"
		case "diff":
			return "查看 Git 差异"
		case "log":
			return "查看提交历史"
		case "show":
			return "查看提交"
		case "add":
			return "暂存改动"
		case "commit":
			return "提交改动"
		case "push":
			return "推送到远端"
		case "pull":
			return "拉取远端"
		case "fetch":
			return "获取远端"
		case "checkout", "switch":
			return "切换分支 / 文件"
		case "branch":
			return "管理分支"
		case "merge", "rebase":
			return "Git " + sub
		case "stash":
			return "Git 储藏"
		case "reset", "restore", "clean":
			return "Git 回退 / 清理"
		case "ls-files":
			return "列出 Git 跟踪的文件"
		case "grep":
			return "Git 搜索"
		case "worktree", "remote", "tag", "blame":
			return "Git " + sub
		default:
			return "git " + sub
		}
	}
	return "git"
}
