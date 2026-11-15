package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/format"
	"path"
	"regexp"
	"strings"
)

// 内置格式化器:语言服务器没有(没装 / 这类文件没有)时的兜底,设计稿 ide-md「格式化的三种结果」。
// Go 用 go/format,JSON 重新缩进,Markdown 与其余文本整理空白;都是纯函数,便于测试。

// builtinFormat 按文件名挑格式化器;by 是显示给用户看的名字,出错时 out 原样返回。
// width 是缩进宽度(只有 JSON 用)。
func builtinFormat(name, src string, width int) (out, by string, err error) {
	if width <= 0 {
		width = 2
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".go":
		b, e := format.Source([]byte(src))
		if e != nil {
			return src, "Go", e
		}
		return string(b), "Go", nil
	case ".json":
		if !json.Valid([]byte(src)) {
			return src, "JSON", errors.New("不是合法的 JSON(带注释的 jsonc 暂不支持)")
		}
		var buf bytes.Buffer
		if e := json.Indent(&buf, []byte(src), "", strings.Repeat(" ", width)); e != nil {
			return src, "JSON", e
		}
		return strings.TrimRight(buf.String(), " \t\r\n") + "\n", "JSON", nil
	case ".md", ".markdown", ".mdc":
		return formatMarkdown(src), "Markdown", nil
	}
	return formatText(src), "文本", nil
}

// formatText 去掉行尾空白与文末多余空行,文末恰好一个换行(空文件保持为空)。保留 CRLF。
func formatText(src string) string {
	if strings.TrimSpace(src) == "" {
		if src == "" {
			return ""
		}
		return "\n"
	}
	lines := strings.Split(src, "\n")
	for i, ln := range lines {
		cr := strings.HasSuffix(ln, "\r")
		ln = strings.TrimRight(strings.TrimSuffix(ln, "\r"), " \t")
		if cr {
			ln += "\r"
		}
		lines[i] = ln
	}
	out := strings.Join(lines, "\n")
	out = strings.TrimRight(out, "\r\n")
	if strings.Contains(src, "\r\n") {
		return out + "\r\n"
	}
	return out + "\n"
}

var hardBreak = regexp.MustCompile(`\S {2,}$`)

// formatMarkdown 整理 Markdown 的空白:围栏代码块里一字不动;其余去行尾空白(保留行尾两个空格的硬换行)、
// 连续空行压成一行、去掉开头空行、文末恰好一个换行。
func formatMarkdown(src string) string {
	if strings.TrimSpace(src) == "" {
		return formatText(src)
	}
	var out []string
	fence := ""
	blank := 0
	all := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	for i, ln := range all {
		trim := strings.TrimSpace(ln)
		if fence != "" {
			out = append(out, ln)
			if strings.HasPrefix(trim, fence) && strings.Trim(trim, fence[:1]) == "" {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			fence = trim[:3]
			blank = 0
			out = append(out, strings.TrimRight(ln, " \t"))
			continue
		}
		if trim == "" {
			blank++
			if blank > 1 || len(out) == 0 {
				continue
			}
			out = append(out, "")
			continue
		}
		blank = 0
		// 行尾两个空格是硬换行:只在后面还有同段文字时才保留(标题、段末的没有意义)
		if hardBreak.MatchString(ln) && !strings.HasPrefix(trim, "#") && i+1 < len(all) && strings.TrimSpace(all[i+1]) != "" {
			ln = strings.TrimRight(ln, " \t") + "  "
		} else {
			ln = strings.TrimRight(ln, " \t")
		}
		out = append(out, ln)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n") + "\n"
}

// formatBuiltin 用内置格式化器整理当前标签并反馈结果;返回是否改了文本。
func (a *App) formatBuiltin(tb *ideTab) bool {
	if tb == nil || tb.buf == nil {
		return false
	}
	width := 2
	if a.prefs.Int(pEdIndent, 1) == 1 {
		width = 4
	}
	src := tb.buf.String()
	out, by, err := builtinFormat(tb.path, src, width)
	switch {
	case err != nil:
		a.toast = "格式化失败(" + by + "):" + firstLine(err.Error())
		uxLog.Warn("内置格式化失败", "path", tb.path, "by", by, "err", err)
		return false
	case out == src:
		a.toast = "已经是格式化好的"
		return false
	}
	setKeepCursor(tb, out)
	a.toast = "已格式化(内置 · " + by + ")"
	uxLog.Info("内置格式化", "path", tb.path, "by", by, "bytes", len(out))
	return true
}

// setKeepCursor 整体替换缓冲区文本并尽量保留光标位置。
func setKeepCursor(tb *ideTab, out string) {
	cur := 0
	if tb.st != nil {
		cur = tb.st.Cursor()
	}
	tb.buf.Set(out)
	if tb.st != nil {
		tb.st.Select(min(cur, len(out)), min(cur, len(out)))
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// beforeSave 是保存前的整理(设置:保存时格式化 / 保存时去掉行尾空白)。
// 所见即所得里的 Markdown 以文档为准,保存前不再整理,免得自动保存把正在输入的光标冲掉。
func (a *App) beforeSave(tb *ideTab) {
	if tb == nil || tb.diff || tb.buf == nil || tb.virt != "" {
		return
	}
	if isMarkdownFile(tb.path) && !a.mdSource(tb) {
		return
	}
	fmtOn, trimOn := a.prefs.Bool(pFmtSave, false), a.prefs.Bool(pTrimSave, false)
	if !fmtOn && !trimOn {
		return
	}
	src := tb.buf.String()
	out := src
	if fmtOn {
		width := 2
		if a.prefs.Int(pEdIndent, 1) == 1 {
			width = 4
		}
		if o, _, err := builtinFormat(tb.path, src, width); err == nil {
			out = o
		}
	} else {
		out = formatText(src)
	}
	if out != src {
		setKeepCursor(tb, out)
	}
}
