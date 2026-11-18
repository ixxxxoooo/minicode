package app

import (
	"os"
	"strings"
	"sync"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// userRefChip 用户消息里的 @ 文件引用胶囊:和输入框里的胶囊(composer_pill.go)同一个样子——
// Fill2 底 + Bd2 描边 + 5 圆角 + 类型图标 + 名称(Tx1),跟着当前主题配色走;文件夹用文件夹图标。
func (a *App) userRefChip(c *ui.Context, path string) {
	t := a.tk
	kind := atomFile
	if strings.HasSuffix(path, "/") {
		kind = atomDir
	}
	label, icon, name := pillLabel(kind, path)
	fs := a.FlowFontSize() * 0.92
	label = clipMiddle(label, pillMaxText, func(s string) float32 {
		w, _ := c.MeasureText(0, ui.Span{Text: s, Size: fs})
		return w
	})
	ui.Row(c).Height(a.FlowFontSize()*1.4).PaddingX(6).Gap(4).Radius(5).Background(t.Fill2).Border(1, t.Bd2).AlignItems(ui.Center).Shrink(0).Children(func() {
		if icon == "folder" {
			kit.Ic(c, "folder", kit.S12).TextColor(t.Tx3)
		} else {
			a.fileIc(c, name)
		}
		ui.Text(c, label).FontSize(fs).TextColor(t.Tx1).SingleLine()
	})
}

// refExistsCache 缓存「这条路径在磁盘上是否存在」:用户消息每帧重排,不能每帧都 stat。
var refExistsCache sync.Map

func refExists(p string) bool {
	if v, ok := refExistsCache.Load(p); ok {
		return v.(bool)
	}
	_, err := os.Stat(p)
	ok := err == nil
	refExistsCache.Store(p, ok)
	return ok
}

// userTokMatches 找出用户文字里要特殊渲染的片段(见 userTok):
//   - @路径 前面必须是空白或行首,邮箱之类不算;
//   - 旧版本发出去的 @路径 不带引号,路径里有空格(如「acceleration-task (9).log」)会被正则在空格处截断:
//     截断后的路径不存在时,往后一个词一个词地并,并到存在的那条为止(最多 6 个词)。
//
// 返回的下标与 FindAllStringSubmatchIndex 一致(分组 4 = @路径,已按上面延长)。
func userTokMatches(txt string) [][]int {
	var out [][]int
	for _, m := range userTok.FindAllStringSubmatchIndex(txt, -1) {
		if m[8] >= 0 { // 分组 4:@路径
			if m[0] > 0 && !unicode.IsSpace(rune(txt[m[0]-1])) {
				continue
			}
			if !refExists(txt[m[8]:m[9]]) {
				end := m[9]
				for i := 0; i < 6; i++ {
					rest := txt[end:]
					ws := len(rest) - len(strings.TrimLeft(rest, " "))
					if ws == 0 || ws == len(rest) {
						break
					}
					wEnd := end + ws
					for wEnd < len(txt) && txt[wEnd] != ' ' && txt[wEnd] != '\n' {
						wEnd++
					}
					if strings.ContainsAny(txt[end:wEnd], "`@") {
						break
					}
					end = wEnd
					if refExists(txt[m[8]:end]) {
						m[1], m[9] = end, end
						break
					}
				}
			}
		}
		out = append(out, m)
	}
	return out
}
