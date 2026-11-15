package demo

import (
	"fmt"
	"strings"
)

// TermLine / TermSeg 描述演示终端的一行(由 app 包转成 termLine)。
type TermLine struct {
	Segs []TermSeg
}
type TermSeg struct {
	Text string
	Kind byte // 0 默认 / p 成功 / e 错误 / d 暗 / b 粗
}

// TermDemoLines 演示模式底部终端的预设输出(非真 shell)。
func TermDemoLines() []TermLine {
	prompt := func(cmd string) TermLine {
		return TermLine{Segs: []TermSeg{{"➜ ", 'p'}, {"router-demo ", 'b'}, {"git:(", 'd'}, {"main", 'e'}, {") ", 'd'}, {cmd, 0}}}
	}
	var lines []TermLine
	lines = append(lines,
		prompt("cargo test router"),
		TermLine{Segs: []TermSeg{{"    Finished", 'd'}, {" test profile in 1.92s", 0}}},
		TermLine{Segs: []TermSeg{{"test matches_ignoring_case ... ", 0}, {"ok", 'p'}}},
		TermLine{Segs: []TermSeg{{"test matches_trailing_slash ... ", 0}, {"ok", 'p'}}},
		TermLine{Segs: []TermSeg{{"test result: ", 0}, {"ok", 'p'}, {". 12 passed; 0 failed", 0}}},
	)
	if !sandboxExtras { // 默认只留设计稿里的一次测试输出 + 光标行;颜色 / 宽字符 / 长输出压力数据随压力开关
		last := prompt("")
		last.Segs[len(last.Segs)-1].Text = ""
		return append(lines, last)
	}
	lines = append(lines, prompt("echo $'ANSI 颜色演示'"))
	lines = append(lines, TermLine{Segs: []TermSeg{{"ANSI 绿", 'p'}, {" ", 0}, {"黄", 'd'}}})
	lines = append(lines, TermLine{Segs: []TermSeg{{"进度 ", 'd'}, {"[", 0}, {"████████░░░░░░░░", 'p'}, {"] 52%", 0}}})
	lines = append(lines, TermLine{Segs: []TermSeg{{"超链接(演示数据;静态面板不支持点击): ", 'd'}, {"https://example.com", 'p'}}})
	lines = append(lines, TermLine{Segs: []TermSeg{{"路径链接: ", 'd'}, {"src/router.rs:16:10", 'p'}}})
	lines = append(lines, TermLine{Segs: []TermSeg{{"中文宽字符: ", 0}, {"汉字ＡＢ", 'b'}, {" 全角", 'd'}}})
	lines = append(lines, TermLine{Segs: []TermSeg{{"[清屏后]", 'd'}, {" 上一屏输出已滚出回看区(静态演示无法真 ESC[2J)", 0}}})
	lines = append(lines, prompt("yes | head -n 50000"))
	for i := 0; i < 80; i++ {
		lines = append(lines, TermLine{Segs: []TermSeg{{fmt.Sprintf("normalize output line %d", i+1), 0}}})
	}
	lines = append(lines, TermLine{Segs: []TermSeg{{"… 共 50000 行(演示静态终端仅渲染前 80 行)", 'd'}}})
	last := prompt("")
	last.Segs[len(last.Segs)-1].Text = ""
	lines = append(lines, last)
	return lines
}

// CmdView 是「在终端面板查看」的只读命令标签。
type CmdView struct {
	ID, Title, Output string
	Exit              *int
}

// CmdViews 额外只读命令标签(退出码、长输出、失败命令)。
func CmdViews() []CmdView {
	if !sandboxExtras { // 默认不额外开命令标签(设计稿只有 zsh + 工具卡里查看的那一个)
		return nil
	}
	fail := 101
	ok := 0
	long := strings.Builder{}
	long.WriteString("npm run build\n")
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&long, "chunk %d normalize\n", i)
	}
	long.WriteString("Done.\n")
	return []CmdView{
		{
			ID: "cmd:normalize-scan", Title: "rg normalize samples/",
			Output: "samples/langs/go_demo.go:6:func Normalize\nsamples/langs/main.ts:4:export function normalizePath\n… (演示)",
			Exit:   &ok,
		},
		{
			ID: "cmd:long-build", Title: "npm run build",
			Output: long.String(),
			Exit:   &ok,
		},
		{
			ID: "cmd:flaky-test", Title: "cargo test edge::syntax",
			Output: "error: could not compile `router-demo`\n\nerror: expected `)`, found `{`",
			Exit:   &fail,
		},
	}
}
