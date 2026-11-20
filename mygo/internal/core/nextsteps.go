package core

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// 下一步建议随回答一起给出(不另发请求):系统提示词里要求 Claude 在回答末尾输出一个 <next_steps>…</next_steps> 块,
// 翻译器在流式增量里把它从正文中截走、解析成 EvNextSteps;界面在回合结束时把它显示成输入框上方的建议胶囊。
// 历史回放也走同一套剥离,所以会话文件里留着的块不会出现在界面上。
//
// @author ixxxxoooo

const (
	NextStepsOpen  = "<next_steps>"
	NextStepsClose = "</next_steps>"
	// NextStepsMax 每轮最多几条;NextStepsMaxRunes 单条上限(界面再按宽度省略)。
	NextStepsMax      = 3
	NextStepsMaxRunes = 40
)

// NextStepsPrompt 是追加到系统提示词末尾的说明(设置 → 对话 → 回答后给出下一步建议 开着时才追加)。
const NextStepsPrompt = `# 下一步建议(界面约定)
当你完成了本次任务、正在等用户的下一步指示时,在最后一条回复的末尾(正文之后、不放进代码块)另起一行,按下面的格式给出 3 条用户最可能想接着说的话:
<next_steps>
建议一
建议二
建议三
</next_steps>
要求:
- 每条一行,口吻是用户对你说的话(祈使句),用和用户相同的语言,不超过 24 个字,能直接执行;不要「继续」「好的」这类空话,也不要编号或引号;
- 3 条方向不同(例如:验证 / 收尾 / 深入),按推荐度从高到低;
- 如果你在向用户提问或需要用户做决定,建议就写成对你的几种不同回答;
- 只在整个回合的最后一条回复里输出一次,工具调用之间的文字里不要输出;纯闲聊或一句话就答完的简单问题可以省略;
- 不要在这个块之外提到它,也不要解释它。`

// EvNextSteps 是回答里带的下一步建议(只来自主对话,子 Agent 的不发)。
type EvNextSteps struct{ Items []string }

func (EvNextSteps) event() {}

var listMarker = regexp.MustCompile(`^(?:[-*•]|\d{1,2}[.、)])\s*`)

// ParseNextSteps 把块里的正文解析成建议:按行,去掉序号 / 项目符号 / 引号,去重,限长,最多 NextStepsMax 条。
func ParseNextSteps(body string) []string {
	var res []string
	seen := map[string]bool{}
	for _, ln := range strings.Split(body, "\n") {
		ln = strings.TrimSpace(ln)
		ln = listMarker.ReplaceAllString(ln, "") // 只去掉「1.」「2)」「-」这样的序号 / 符号,不动正文里的数字
		ln = strings.Trim(ln, "\"“”「」`")
		ln = strings.TrimSpace(ln)
		if ln == "" || seen[ln] {
			continue
		}
		seen[ln] = true
		if utf8.RuneCountInString(ln) > NextStepsMaxRunes {
			ln = string([]rune(ln)[:NextStepsMaxRunes]) + "…"
		}
		res = append(res, ln)
		if len(res) == NextStepsMax {
			break
		}
	}
	return res
}

// NextStepsFilter 是流式文字里的块截取器:喂入一段段增量,吐出应显示的正文;块内的字被扣下,闭合时解析。
// 开标签可能被切在两段增量之间(如 "<next_" + "steps>"),所以末尾像开标签前缀的几个字先扣住,下一段来了再判断。
type NextStepsFilter struct {
	hold  string // 末尾可能是开标签前缀的字
	buf   string // 块内已收到的字
	inTag bool
}

// Push 喂入一段增量,返回应显示的文字,以及(块在这一段里闭合时)解析出的建议。
func (f *NextStepsFilter) Push(s string) (show string, items []string) {
	s = f.hold + s
	f.hold = ""
	var out strings.Builder
	for s != "" {
		if f.inTag {
			f.buf += s
			s = ""
			if j := strings.Index(f.buf, NextStepsClose); j >= 0 {
				if it := ParseNextSteps(f.buf[:j]); len(it) > 0 {
					items = it
				}
				s = f.buf[j+len(NextStepsClose):]
				f.buf, f.inTag = "", false
			}
			continue
		}
		if i := strings.Index(s, NextStepsOpen); i >= 0 {
			out.WriteString(s[:i])
			s = s[i+len(NextStepsOpen):]
			f.inTag = true
			continue
		}
		k := openPrefixLen(s)
		out.WriteString(s[:len(s)-k])
		f.hold = s[len(s)-k:]
		s = ""
	}
	return out.String(), items
}

// Flush 在文本块结束时调用:返回被扣住、其实不是开标签的字;块没闭合(输出被截断)就整块丢弃,不让半截标签露出来。
func (f *NextStepsFilter) Flush() string {
	h := f.hold
	f.hold, f.buf, f.inTag = "", "", false
	return h
}

// openPrefixLen 是 s 末尾与开标签前缀重合的最长长度(不含整个开标签)。
func openPrefixLen(s string) int {
	for k := min(len(NextStepsOpen)-1, len(s)); k > 0; k-- {
		if strings.HasSuffix(s, NextStepsOpen[:k]) {
			return k
		}
	}
	return 0
}

// StripNextSteps 从一整段文字里剥掉块(完整消息 / 历史回放用):返回正文与建议;没有块就原样返回。
func StripNextSteps(text string) (clean string, items []string) {
	if !strings.Contains(text, "<next_") {
		return text, nil
	}
	var f NextStepsFilter
	show, items := f.Push(text)
	show += f.Flush()
	if items != nil || strings.Contains(text, NextStepsOpen) {
		show = strings.TrimRight(show, " \t\r\n")
	}
	return show, items
}
