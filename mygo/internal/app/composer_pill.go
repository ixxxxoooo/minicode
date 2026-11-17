package app

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 正文里的胶囊(文件 / 文件夹 / 折叠的长文本):圆角底 + 描边 + 类型图标 + 名称,不带 @,
// 和文字在同一行、随文字一起换行;是编辑器里的一个原子字符(光标跳过、退格整块删)。
//
// @author ixxxxoooo

const (
	pillMaxText = 190 // 名称最宽(超出省略)
	pillMargin  = 2   // 胶囊左右留的空隙,相邻胶囊 / 文字不贴死
)

// pillLabel 取胶囊上显示的名称与图标。
// 参数:kind 原子类型(file / dir / paste:字数);path 路径。返回:名称、图标名、图标取名用的文件名。
func pillLabel(kind, path string) (label, icon, name string) {
	name = filepath.Base(strings.TrimSuffix(path, "/"))
	switch {
	case kind == atomDir:
		return name, "folder", name
	case strings.HasPrefix(kind, atomPaste):
		n := strings.TrimPrefix(strings.TrimPrefix(kind, atomPaste), ":")
		if n == "" {
			return "粘贴的文本", "fileText", name
		}
		return "粘贴的文本 · " + n + " 字", "fileText", name
	}
	return name, fileIcon(name), name
}

// clipMiddle 名称放不下 max 宽时从中间删字补「…」(保留头尾,扩展名不丢)。span 量一段文字的宽度。
func clipMiddle(s string, max float32, span func(string) float32) string {
	if span(s) <= max {
		return s
	}
	rs := []rune(s)
	head, tail := rs[:(len(rs)+1)/2], rs[(len(rs)+1)/2:]
	for len(head)+len(tail) > 4 && span(string(head)+"…"+string(tail)) > max {
		if len(head) > len(tail) {
			head = head[:len(head)-1]
		} else {
			tail = tail[1:]
		}
	}
	return string(head) + "…" + string(tail)
}

// pillAttachment 造出一枚胶囊的富文本附件。尺寸在这里量好(附件要先给定宽高),绘制时再按当前主题取色。
// 参数:c 构建界面期间的上下文(量字用);kind / path 同 pillLabel。
func (a *App) pillAttachment(c *ui.Context, kind, path string) *richtext.Attachment {
	label, icon, name := pillLabel(kind, path)
	fs := a.FlowFontSize() * 0.92
	span := func(s string) float32 {
		w, _ := c.MeasureText(0, ui.Span{Text: s, Size: fs})
		return w
	}
	label = clipMiddle(label, pillMaxText, span)
	iconSz := fs
	padL, gap, padR := float32(6), float32(4), float32(7)
	w := padL + iconSz + gap + span(label) + padR
	// 附件框的底边落在文字基线上,胶囊要和文字「视觉中心」对齐,就得在基线下面再伸出一截(字的下沿):
	// 附件框只占基线以上的部分,绘制时把胶囊画得比框高 down,向下伸出去。
	pillH := a.FlowFontSize() * 1.4
	down := a.FlowFontSize() * 0.3
	h := pillH - down
	svg := kit.IconSVG(icon)
	return &richtext.Attachment{
		Width: w + 2*pillMargin, Height: h,
		Name: label, URL: path,
		Data: pillData{Kind: kind, Path: path},
		Draw: func(p *ui.Painter, r ui.Rect) {
			t := a.tk
			box := ui.Rect{X: r.X + pillMargin, Y: r.Y, W: r.W - 2*pillMargin, H: pillH}
			p.Fill(box, t.Fill2, 5)
			p.Stroke(box, t.Bd2, 5, 1)
			tint := t.Tx3
			if icon != "folder" {
				tint = a.fileTint(name, t.Tx3)
			}
			p.Icon(svg, ui.Rect{X: box.X + padL, Y: box.Y + (box.H-iconSz)/2, W: iconSz, H: iconSz}, tint)
			p.Text(box.X+padL+iconSz+gap, box.Y+(box.H-fs*1.25)/2, label, fs, t.Tx1)
		},
	}
}
