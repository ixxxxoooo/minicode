// Package mermaid 把 Mermaid 源码排版成矩形 / 折线 / 文字组成的 Scene,与界面无关。
package mermaid

import (
	"fmt"
	"strings"
)

// Role 图元语义色,由界面映射到主题。
type Role int

const (
	RoleSurface Role = iota
	RoleNode
	RoleBorder
	RoleLine
	RoleText
	RoleMuted
	RoleNote
	RoleGroup
)

// RoleAccent 强调色槽(饼图扇区等),n 取 0..7。
func RoleAccent(n uint8) Paint { return Paint{accent: n + 1} }

// Paint 语义角色或 RGB(#rrggbb)。
type Paint struct {
	role   Role
	rgb    uint32
	accent uint8 // 0=无,>0 为 Accent 序号-1
}

func (p Paint) Role() (Role, bool) {
	if p.accent > 0 {
		return 0, false
	}
	if p.rgb != 0 {
		return 0, false
	}
	return p.role, true
}

func (p Paint) RGB() (uint32, bool) {
	if p.rgb != 0 {
		return p.rgb, true
	}
	return 0, false
}

func (p Paint) Accent() (uint8, bool) {
	if p.accent > 0 {
		return p.accent - 1, true
	}
	return 0, false
}

func PaintRole(r Role) Paint  { return Paint{role: r} }
func PaintRGB(v uint32) Paint { return Paint{rgb: v} }
func rolePaint(r Role) Paint  { return Paint{role: r} }

// Anchor 文字锚点。
type Anchor int

const (
	AnchorStart Anchor = iota
	AnchorMiddle
	AnchorEnd
)

// Item 一个图元。
type Item struct {
	Rect *RectItem
	Poly *PolyItem
	Text *TextItem
}

type RectItem struct {
	X, Y, W, H, R float32
	Fill, Stroke  *Paint
	Width         float32
}

type PolyItem struct {
	Pts          [][2]float32
	Closed       bool
	Fill, Stroke *Paint
	Width        float32
}

type TextItem struct {
	X, Y   float32
	Text   string
	Size   float32
	Bold   bool
	Anchor Anchor
	Paint  Paint
}

// Scene 排版结果。
type Scene struct {
	Width, Height float32
	Items         []Item
}

// RenderError 渲染失败原因。
type RenderError struct {
	Kind RenderErrKind
	Arg  string
	N    int
}

type RenderErrKind int

const (
	ErrUnsupported RenderErrKind = iota
	ErrSyntax
	ErrTooLarge
	ErrEmpty
)

func (e RenderError) Error() string {
	switch e.Kind {
	case ErrUnsupported:
		return fmt.Sprintf("暂不支持 %s 图表", e.Arg)
	case ErrSyntax:
		return "语法错误:" + e.Arg
	case ErrTooLarge:
		return fmt.Sprintf("图太大(超过 %d 个节点)", e.N)
	case ErrEmpty:
		return "内容为空"
	default:
		return "未知错误"
	}
}

// MaxNodes 单张图的节点数上限。
const MaxNodes = 300

// Fit 按图元范围收紧画布。
func (s Scene) Fit(margin float32) Scene {
	x0, y0, x1, y1 := float32(1e30), float32(1e30), float32(-1e30), float32(-1e30)
	grow := func(a, b, c, d float32) {
		if a < x0 {
			x0 = a
		}
		if b < y0 {
			y0 = b
		}
		if c > x1 {
			x1 = c
		}
		if d > y1 {
			y1 = d
		}
	}
	for _, it := range s.Items {
		switch {
		case it.Rect != nil:
			r := it.Rect
			grow(r.X, r.Y, r.X+r.W, r.Y+r.H)
		case it.Poly != nil:
			for _, p := range it.Poly.Pts {
				grow(p[0], p[1], p[0], p[1])
			}
		case it.Text != nil:
			t := it.Text
			w := TextW(t.Text, t.Size)
			left := t.X
			switch t.Anchor {
			case AnchorMiddle:
				left = t.X - w/2
			case AnchorEnd:
				left = t.X - w
			}
			grow(left, t.Y-t.Size, left+w, t.Y+t.Size)
		}
	}
	if x0 > x1 {
		s.Width = margin * 2
		s.Height = margin * 2
		return s
	}
	dx, dy := margin-x0, margin-y0
	for i := range s.Items {
		switch {
		case s.Items[i].Rect != nil:
			s.Items[i].Rect.X += dx
			s.Items[i].Rect.Y += dy
		case s.Items[i].Text != nil:
			s.Items[i].Text.X += dx
			s.Items[i].Text.Y += dy
		case s.Items[i].Poly != nil:
			for j := range s.Items[i].Poly.Pts {
				s.Items[i].Poly.Pts[j][0] += dx
				s.Items[i].Poly.Pts[j][1] += dy
			}
		}
	}
	s.Width = x1 - x0 + margin*2
	s.Height = y1 - y0 + margin*2
	return s
}

func diagramKind(src string) (string, bool) {
	var lines []string
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "", false
	}
	first := lines[0]
	if first == "---" {
		found := false
		for i := 1; i < len(lines); i++ {
			if lines[i] == "---" {
				if i+1 < len(lines) {
					first = lines[i+1]
					found = true
				}
				break
			}
		}
		if !found {
			return "", false
		}
	}
	for i, r := range first {
		if r == ' ' || r == ';' || r == '\t' {
			first = first[:i]
			break
		}
	}
	return first, first != ""
}

// Render 把 Mermaid 源码排版成场景。
func Render(src string) (Scene, error) {
	kind, ok := diagramKind(src)
	if !ok {
		return Scene{}, RenderError{Kind: ErrEmpty}
	}
	kl := strings.ToLower(kind)
	var scene Scene
	var err error
	switch kl {
	case "graph", "flowchart", "flowchart-elk":
		scene, err = renderFlow(src)
	case "sequencediagram":
		scene, err = renderSeq(src)
	case "pie":
		scene, err = renderPie(src)
	case "statediagram", "statediagram-v2":
		scene, err = renderState(src)
	default:
		return Scene{}, RenderError{Kind: ErrUnsupported, Arg: kind}
	}
	if err != nil {
		return Scene{}, err
	}
	return scene.Fit(16), nil
}

// KindTitle 图表类型中文名。
func KindTitle(src string) (string, bool) {
	kind, ok := diagramKind(src)
	if !ok {
		return "", false
	}
	switch strings.ToLower(kind) {
	case "graph", "flowchart", "flowchart-elk":
		return "流程图", true
	case "sequencediagram":
		return "时序图", true
	case "pie":
		return "饼图", true
	case "statediagram", "statediagram-v2":
		return "状态图", true
	default:
		return "", false
	}
}

func itemRect(x, y, w, h, r float32, fill, stroke *Paint, width float32) Item {
	return Item{Rect: &RectItem{X: x, Y: y, W: w, H: h, R: r, Fill: fill, Stroke: stroke, Width: width}}
}

func itemPoly(pts [][2]float32, closed bool, fill, stroke *Paint, width float32) Item {
	cp := make([][2]float32, len(pts))
	copy(cp, pts)
	return Item{Poly: &PolyItem{Pts: cp, Closed: closed, Fill: fill, Stroke: stroke, Width: width}}
}

func itemText(x, y float32, text string, size float32, bold bool, anchor Anchor, paint Paint) Item {
	return Item{Text: &TextItem{X: x, Y: y, Text: text, Size: size, Bold: bold, Anchor: anchor, Paint: paint}}
}
