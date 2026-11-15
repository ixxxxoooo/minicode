package mermaid

import (
	"fmt"
	"sort"
)

const seqFont = 13.5
const seqLH = 17

func drawSeq(seq *Seq) Scene {
	n := len(seq.Parts)
	widths := make([]float32, n)
	heights := make([]float32, n)
	for i, p := range seq.Parts {
		widths[i] = LinesW(p.Label, seqFont) + 30
		if widths[i] < 84 {
			widths[i] = 84
		}
		heights[i] = float32(len(p.Label))*seqLH + 16
	}
	boxH := float32(34)
	for _, h := range heights {
		if h > boxH {
			boxH = h
		}
	}
	xs := seqColumns(seq, widths)
	xmin, xmax := xs[0], xs[n-1]
	var back, front []Item
	top := float32(0)
	if seq.Title != nil {
		top = 34
		front = append(front, itemText((xmin+xmax)/2, 10, *seq.Title, 15, true, AnchorMiddle, PaintRole(RoleText)))
	}
	y := top + boxH + 24
	counter := 0
	type frame struct {
		y0    float32
		kind  string
		label string
		divs  [][2]float32 // y, label
	}
	var frames []frame
	acts := make([][]float32, n)
	var bars [][4]int // i, depth, start y as float stored separately
	var barF [][4]float32

	pushLine := func(items *[]Item, pts []P, dotted bool) {
		stroke := PaintRole(RoleLine)
		if dotted {
			for _, seg := range Dashes(pts, 5, 4) {
				*items = append(*items, itemPoly(seg, false, nil, &stroke, 1.3))
			}
		} else {
			*items = append(*items, itemPoly(pts, false, nil, &stroke, 1.5))
		}
	}

	for _, ev := range seq.Evs {
		switch {
		case ev.Msg != nil:
			m := ev.Msg
			lines := append([]string(nil), m.Text...)
			if seq.Autonumber && len(lines) > 0 {
				counter++
				lines[0] = fmt.Sprintf("%d. %s", counter, lines[0])
			}
			xa, xb := xs[m.From], xs[m.To]
			labelH := float32(0)
			if len(lines) > 0 && lines[0] != "" {
				labelH = float32(len(lines)) * seqLH
			}
			ay := y + labelH + 8
			barA := float32(len(acts[m.From])) * 4
			barB := float32(len(acts[m.To]))*4 + boolF(m.Activate)*4
			if m.From == m.To {
				x := xa + 5 + barA
				pts := []P{{x, ay}, {x + 34, ay}, {x + 34, ay + 22}, {x, ay + 22}}
				pushLine(&front, pts[:3], m.Dotted)
				front = append(front, itemPoly([]P{pts[2], pts[3]}, false, nil, paintPtr(RoleLine), 1.5))
				seqHead(&front, m.Tip, pts[3], pts[2])
				seqBlock(&front, x+42, ay-4, lines, AnchorStart, PaintRole(RoleText))
				y = ay + 22 + 16
			} else {
				dir := float32(1)
				if xb < xa {
					dir = -1
				}
				sx, ex := xa+dir*(5+barA), xb-dir*(5+barB)
				cx := (sx + ex) / 2
				seqBlock(&front, cx, y, lines, AnchorMiddle, PaintRole(RoleText))
				endLine := ex
				if m.Tip == TipArrow || m.Tip == TipAsync {
					endLine = ex - dir*8
				}
				pushLine(&front, []P{{sx, ay}, {endLine, ay}}, m.Dotted)
				seqHead(&front, m.Tip, P{ex, ay}, P{sx, ay})
				y = ay + 18
			}
			if m.Activate {
				acts[m.To] = append(acts[m.To], ay)
			}
			if m.Deactivate && len(acts[m.From]) > 0 {
				s := acts[m.From][len(acts[m.From])-1]
				acts[m.From] = acts[m.From][:len(acts[m.From])-1]
				barF = append(barF, [4]float32{float32(m.From), float32(len(acts[m.From])), s, ay})
			}
		case ev.Note != nil:
			lines := ev.Note.Text
			w := LinesW(lines, seqFont) + 24
			h := float32(len(lines))*seqLH + 12
			x0 := float32(0)
			at := ev.Note.At
			if at.Over {
				w = maxf(w, xs[at.B]-xs[at.A]+28)
				x0 = (xs[at.A]+xs[at.B])/2 - w/2
			} else if at.Left {
				x0 = xs[at.ParticipantOnly] - 14 - w
			} else if at.Right {
				x0 = xs[at.ParticipantOnly] + 14
			}
			stroke := PaintRole(RoleBorder)
			fill := PaintRole(RoleNote)
			front = append(front, itemRect(x0, y, w, h, 3, &fill, &stroke, 1))
			seqBlock(&front, x0+w/2, y+6, lines, AnchorMiddle, PaintRole(RoleText))
			y += h + 14
		case ev.Activate != nil:
			acts[*ev.Activate] = append(acts[*ev.Activate], y-8)
		case ev.Deactivate != nil:
			i := *ev.Deactivate
			if len(acts[i]) > 0 {
				s := acts[i][len(acts[i])-1]
				acts[i] = acts[i][:len(acts[i])-1]
				barF = append(barF, [4]float32{float32(i), float32(len(acts[i])), s, y - 8})
			}
		case ev.Open != nil:
			frames = append(frames, frame{y0: y, kind: ev.Open.Kind, label: ev.Open.Label})
			y += 28
		case ev.Else != nil:
			if len(frames) > 0 {
				f := &frames[len(frames)-1]
				f.divs = append(f.divs, [2]float32{y, float32(0)}) // label stored separately - simplified
				_ = ev.Else
			}
			y += 24
		case ev.Close:
			if len(frames) > 0 {
				f := frames[len(frames)-1]
				frames = frames[:len(frames)-1]
				d := float32(len(frames))
				x0 := xmin - 34 + d*8 - widths[0]/2 + 40
				x1 := xmax + 34 - d*8 + widths[n-1]/2 - 40
				stroke := PaintRole(RoleBorder)
				fillG := PaintRole(RoleGroup)
				back = append(back, itemRect(x0, f.y0, x1-x0, y-f.y0, 3, nil, &stroke, 1.2))
				tab := TextW(f.kind, 12) + 18
				back = append(back, itemRect(x0, f.y0, tab, 20, 3, &fillG, &stroke, 1.2))
				back = append(back, itemText(x0+tab/2, f.y0+10, f.kind, 12, true, AnchorMiddle, PaintRole(RoleText)))
				if f.label != "" {
					back = append(back, itemText(x0+tab+8, f.y0+10, "["+f.label+"]", 12.5, false, AnchorStart, PaintRole(RoleMuted)))
				}
				y += 12
			}
		}
	}
	end := y + 6
	for i, stack := range acts {
		for d, s := range stack {
			barF = append(barF, [4]float32{float32(i), float32(d), s, end - 14})
		}
	}
	_ = bars
	var lifelines []Item
	for i := 0; i < n; i++ {
		pushLine(&lifelines, []P{{xs[i], top + boxH}, {xs[i], end + 10}}, true)
	}
	var boxes []Item
	for i, p := range seq.Parts {
		for _, by := range []float32{top, end + 10} {
			r := float32(5)
			if p.Actor {
				r = boxH / 2
			}
			stroke := PaintRole(RoleBorder)
			fill := PaintRole(RoleNode)
			boxes = append(boxes, itemRect(xs[i]-widths[i]/2, by, widths[i], boxH, r, &fill, &stroke, 1.3))
			t := by + boxH/2 - float32(len(p.Label))*seqLH/2
			seqBlock(&boxes, xs[i], t, p.Label, AnchorMiddle, PaintRole(RoleText))
		}
	}
	var actItems []Item
	for _, b := range barF {
		i, d, s, e := int(b[0]), b[1], b[2], b[3]
		fill := PaintRole(RoleGroup)
		stroke := PaintRole(RoleBorder)
		actItems = append(actItems, itemRect(xs[i]-5+d*4, s, 10, maxf(e-s, 10), 1, &fill, &stroke, 1.2))
	}
	var items []Item
	items = append(items, lifelines...)
	items = append(items, back...)
	items = append(items, actItems...)
	items = append(items, boxes...)
	items = append(items, front...)
	return Scene{Items: items}
}

func seqColumns(seq *Seq, widths []float32) []float32 {
	n := len(seq.Parts)
	need := make([]float32, max(0, n-1))
	for k := 0; k < n-1; k++ {
		need[k] = (widths[k] + widths[k+1]) / 2
		need[k] += 30
	}
	type span struct {
		i, j int
		r    float32
	}
	var spans []span
	for _, ev := range seq.Evs {
		if ev.Msg != nil {
			m := ev.Msg
			w := LinesW(m.Text, seqFont) + 40
			if m.From == m.To {
				if m.From+1 < n {
					spans = append(spans, span{m.From, m.From + 1, w + 50})
				}
			} else {
				i, j := m.From, m.To
				if i > j {
					i, j = j, i
				}
				spans = append(spans, span{i, j, w})
			}
		}
	}
	sort.Slice(spans, func(a, b int) bool { return spans[a].j-spans[a].i < spans[b].j-spans[b].i })
	for _, sp := range spans {
		var cur float32
		for k := sp.i; k < sp.j; k++ {
			cur += need[k]
		}
		if cur < sp.r {
			add := (sp.r - cur) / float32(sp.j-sp.i)
			for k := sp.i; k < sp.j; k++ {
				need[k] += add
			}
		}
	}
	xs := []float32{widths[0] / 2}
	for k := 0; k < n-1; k++ {
		xs = append(xs, xs[k]+need[k])
	}
	return xs
}

func seqHead(items *[]Item, tip Tip, at, from P) {
	stroke := PaintRole(RoleLine)
	switch tip {
	case TipArrow:
		*items = append(*items, itemPoly(ArrowHead(at, from, 9, 4.5), true, &stroke, &stroke, 1))
	case TipAsync:
		h := ArrowHead(at, from, 9, 5)
		*items = append(*items, itemPoly([]P{h[1], h[0], h[2]}, false, nil, &stroke, 1.5))
	case TipCross:
		for _, sgn := range []float32{1, -1} {
			*items = append(*items, itemPoly([]P{{at[0] - 4, at[1] - 4*sgn}, {at[0] + 4, at[1] + 4*sgn}}, false, nil, &stroke, 1.8))
		}
	}
}

func seqBlock(items *[]Item, x, top float32, lines []string, anchor Anchor, paint Paint) float32 {
	for i, l := range lines {
		if l != "" {
			*items = append(*items, itemText(x, top+seqLH/2+float32(i)*seqLH, l, seqFont, false, anchor, paint))
		}
	}
	return float32(len(lines)) * seqLH
}

func paintPtr(r Role) *Paint {
	p := PaintRole(r)
	return &p
}

func boolF(v bool) float32 {
	if v {
		return 4
	}
	return 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
