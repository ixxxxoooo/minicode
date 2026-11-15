package mermaid

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

const pieR float32 = 110

func renderPie(src string) (Scene, error) {
	stmts := Statements(src)
	if len(stmts) == 0 {
		return Scene{}, RenderError{Kind: ErrEmpty}
	}
	head := stmts[0]
	showData := strings.Contains(head, "showData")
	var title *string
	if i := strings.Index(head, "title "); i >= 0 {
		t := strings.TrimSpace(head[i+6:])
		title = &t
	}
	type slice struct {
		label string
		value float64
	}
	var slices []slice
	for _, line := range stmts[1:] {
		if t, ok := strings.CutPrefix(line, "title "); ok {
			t = strings.TrimSpace(t)
			title = &t
			continue
		}
		label, value, ok := strings.Cut(line, ":")
		if !ok {
			return Scene{}, RenderError{Kind: ErrSyntax, Arg: line}
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return Scene{}, RenderError{Kind: ErrSyntax, Arg: line}
		}
		if v < 0 {
			return Scene{}, RenderError{Kind: ErrSyntax, Arg: line}
		}
		slices = append(slices, slice{strings.Join(SplitLines(label), " "), v})
	}
	var total float64
	for _, d := range slices {
		total += d.value
	}
	if len(slices) == 0 || total <= 0 {
		return Scene{}, RenderError{Kind: ErrEmpty}
	}
	sort.Slice(slices, func(i, j int) bool { return slices[i].value > slices[j].value })
	var items []Item
	cx, cy := pieR+10, pieR+40
	if title != nil {
		items = append(items, itemText(cx, 12, *title, 16, true, AnchorMiddle, PaintRole(RoleText)))
	}
	start := float32(-math.Pi / 2)
	for i, d := range slices {
		frac := float32(d.value / total)
		end := start + frac*2*float32(math.Pi)
		color := RoleAccent(uint8(i))
		var pts []P
		if frac >= 0.9999 {
			pts = Arc(cx, cy, pieR, pieR, 0, 2*float32(math.Pi), 72)
		} else {
			pts = append(pts, P{cx, cy})
			steps := int(frac * 72)
			if steps < 2 {
				steps = 2
			}
			pts = append(pts, Arc(cx, cy, pieR, pieR, start, end, steps)...)
		}
		surf := PaintRole(RoleSurface)
		items = append(items, itemPoly(pts, true, &color, &surf, 2))
		if frac >= 0.04 {
			mid := (start + end) / 2
			items = append(items, itemText(cx+pieR*0.68*float32(math.Cos(float64(mid))), cy+pieR*0.68*float32(math.Sin(float64(mid))),
				fmt.Sprintf("%.0f%%", frac*100), 12.5, true, AnchorMiddle, PaintRGB(0xffffff)))
		}
		ly := float32(50 + i*24)
		lx := cx + pieR + 40
		items = append(items, itemRect(lx, ly-7, 14, 14, 3, &color, nil, 0))
		shown := d.label
		if showData {
			shown = fmt.Sprintf("%s [%s]", d.label, fmtPieVal(d.value))
		}
		items = append(items, itemText(lx+22, ly, shown, 13.5, false, AnchorStart, PaintRole(RoleText)))
		_ = TextW(shown, 13.5)
		start = end
	}
	return Scene{Items: items}, nil
}

func fmtPieVal(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%v", v)
}
