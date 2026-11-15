package mermaid

import "math"

const fontSize = 14
const lineH = 19

func nodeSize(n *Node) (float32, float32) {
	tw := LinesW(n.Label, fontSize)
	th := float32(len(n.Label)) * lineH
	w, h := tw+28, th+18
	switch n.Shape {
	case ShapeRect, ShapeRound:
		if w < 44 {
			w = 44
		}
		if h < 36 {
			h = 36
		}
	case ShapeStadium:
		w = w + 16
		if w < 60 {
			w = 60
		}
		if h < 36 {
			h = 36
		}
	case ShapeSubroutine:
		w = w + 18
		if w < 60 {
			w = 60
		}
		if h < 36 {
			h = 36
		}
	case ShapeCylinder:
		if w < 54 {
			w = 54
		}
		h = th + 32
		if h < 50 {
			h = 50
		}
	case ShapeCircle:
		d := tw + 16
		if th+12 > d {
			d = th + 12
		}
		d *= 1.15
		if d < 48 {
			d = 48
		}
		return d, d
	case ShapeDiamond:
		if w = tw*1.7 + 36; w < 70 {
			w = 70
		}
		if h = th*1.8 + 20; h < 60 {
			h = 60
		}
	case ShapeHexagon:
		w += 24
		if w < 70 {
			w = 70
		}
		if h < 40 {
			h = 40
		}
	case ShapeParaR, ShapeParaL, ShapeTrapU, ShapeTrapD, ShapeFlag:
		w += 26
		if w < 70 {
			w = 70
		}
		if h < 36 {
			h = 36
		}
	case ShapeDot:
		return 16, 16
	}
	return w, h
}

func rectLike(s Shape) bool {
	switch s {
	case ShapeRect, ShapeRound, ShapeStadium, ShapeSubroutine, ShapeCylinder, ShapeHexagon:
		return true
	default:
		return false
	}
}

func topGroups(flow *Flow) []int {
	g := make([]int, len(flow.Nodes))
	for i := range g {
		g[i] = 1<<30 - 1
	}
	for si, sub := range flow.Subs {
		top := si
		for {
			if flow.Subs[top].Parent == nil {
				break
			}
			top = *flow.Subs[top].Parent
		}
		for _, n := range sub.Nodes {
			if n >= 0 && n < len(g) {
				g[n] = top
			}
		}
	}
	return g
}

func subNodes(flow *Flow, si int) []int {
	v := append([]int(nil), flow.Subs[si].Nodes...)
	for ci, c := range flow.Subs {
		if c.Parent != nil && *c.Parent == si {
			v = append(v, subNodes(flow, ci)...)
		}
	}
	return v
}

func subDepth(flow *Flow, si int) int {
	m := 0
	for ci, c := range flow.Subs {
		if c.Parent != nil && *c.Parent == si {
			if d := 1 + subDepth(flow, ci); d > m {
				m = d
			}
		}
	}
	return m
}

func drawFlow(flow *Flow) Scene {
	sizes := make([][2]float32, len(flow.Nodes))
	rect := make([]bool, len(flow.Nodes))
	for i, n := range flow.Nodes {
		sizes[i][0], sizes[i][1] = nodeSize(&n)
		rect[i] = rectLike(n.Shape)
	}
	var edgesIn []edgeIn
	for _, e := range flow.Edges {
		edgesIn = append(edgesIn, edgeIn{e.From, e.To, len(e.Label) > 0})
	}
	lay := layoutFlow(flow.Dir, sizes, edgesIn, rect, topGroups(flow))
	var items []Item
	order := make([]int, len(flow.Subs))
	for i := range order {
		order[i] = i
	}
	sortIntsByDepth := func() {
		for i := 0; i < len(order); i++ {
			for j := i + 1; j < len(order); j++ {
				if subDepth(flow, order[j]) > subDepth(flow, order[i]) {
					order[i], order[j] = order[j], order[i]
				}
			}
		}
	}
	sortIntsByDepth()
	for _, si := range order {
		members := subNodes(flow, si)
		if len(members) == 0 {
			continue
		}
		pad := 16 + 14*float32(subDepth(flow, si))
		x0, y0, x1, y1 := float32(1e30), float32(1e30), float32(-1e30), float32(-1e30)
		for _, m := range members {
			if m < 0 || m >= len(sizes) {
				continue
			}
			c := lay.centers[m]
			s := sizes[m]
			x0 = minf(x0, c[0]-s[0]/2)
			x1 = maxf(x1, c[0]+s[0]/2)
			y0 = minf(y0, c[1]-s[1]/2)
			y1 = maxf(y1, c[1]+s[1]/2)
		}
		title := flow.Subs[si].Title
		w := x1 - x0 + pad*2
		if tw := TextW(title, 13) + 24; tw > w {
			w = tw
		}
		stroke := PaintRole(RoleBorder)
		fill := PaintRole(RoleGroup)
		items = append(items, itemRect((x0+x1)/2-w/2, y0-pad-20, w, y1-y0+pad*2+20, 8, &fill, &stroke, 1))
		items = append(items, itemText((x0+x1)/2-w/2+12, y0-pad-8, title, 13, true, AnchorStart, PaintRole(RoleMuted)))
	}
	var late []Item
	for k, e := range flow.Edges {
		var route [][2]float32
		if e.From == e.To {
			route = selfLoop(lay.centers[e.From], sizes[e.From])
		} else if k < len(lay.routes) {
			route = lay.routes[k]
		}
		drawEdge(e, route, &items, &late)
	}
	for i, n := range flow.Nodes {
		drawNode(&n, lay.centers[i], sizes[i], &items)
	}
	items = append(items, late...)
	return Scene{Items: items}
}

func selfLoop(c [2]float32, s [2]float32) [][2]float32 {
	x, y0, y1 := c[0]+s[0]/2, c[1]-s[1]/4, c[1]+s[1]/4
	return Chaikin([]P{{x, y0}, {x + 24, y0}, {x + 24, y1}, {x, y1}}, 2)
}

func drawEdge(e Edge, route [][2]float32, items, late *[]Item) {
	if len(route) < 2 {
		return
	}
	width := float32(1.5)
	if e.Line == LineThick {
		width = 3
	}
	headLen, half := float32(9), float32(4.5)
	if e.Line == LineThick {
		headLen, half = 11, 5.5
	}
	body := append([]P(nil), toPs(route)...)
	if e.End == HeadArrow {
		body = TrimEnd(body, headLen*0.8)
	}
	if e.Start == HeadArrow {
		for i, j := 0, len(body)-1; i < j; i, j = i+1, j-1 {
			body[i], body[j] = body[j], body[i]
		}
		body = TrimEnd(body, headLen*0.8)
		for i, j := 0, len(body)-1; i < j; i, j = i+1, j-1 {
			body[i], body[j] = body[j], body[i]
		}
	}
	stroke := PaintRole(RoleLine)
	pushLine := func(pts []P) {
		*items = append(*items, itemPoly(pts, false, nil, &stroke, width))
	}
	if e.Line == LineDotted {
		for _, seg := range Dashes(body, 4, 4) {
			pushLine(seg)
		}
	} else {
		pushLine(body)
	}
	marker := func(tip, from P, head Head) {
		switch head {
		case HeadArrow:
			*late = append(*late, itemPoly(ArrowHead(tip, from, headLen, half), true, &stroke, &stroke, 1))
		case HeadCircle:
			d := Dist(tip, from)
			if d < 0.001 {
				d = 0.001
			}
			cx := tip[0] + (from[0]-tip[0])/d*5
			cy := tip[1] + (from[1]-tip[1])/d*5
			surf := PaintRole(RoleSurface)
			*late = append(*late, itemPoly(Arc(cx, cy, 5, 5, 0, 2*math.Pi, 16), true, &surf, &stroke, 1.5))
		case HeadCross:
			d := Dist(tip, from)
			if d < 0.001 {
				d = 0.001
			}
			ux, uy := (from[0]-tip[0])/d, (from[1]-tip[1])/d
			c := P{tip[0] + ux*6, tip[1] + uy*6}
			for _, sgn := range []float32{1, -1} {
				*late = append(*late, itemPoly([]P{
					{c[0] - 4*(ux-sgn*uy), c[1] - 4*(uy+sgn*ux)},
					{c[0] + 4*(ux-sgn*uy), c[1] + 4*(uy-sgn*ux)},
				}, false, nil, &stroke, 1.8))
			}
		}
	}
	n := len(route)
	marker(P(route[n-1]), P(route[n-2]), e.End)
	marker(P(route[0]), P(route[1]), e.Start)
	if len(e.Label) > 0 {
		mid := PointAt(toPs(route), Length(toPs(route))/2)
		w := LinesW(e.Label, 12.5) + 12
		h := float32(len(e.Label))*16 + 6
		surf := PaintRole(RoleSurface)
		*late = append(*late, itemRect(mid[0]-w/2, mid[1]-h/2, w, h, 4, &surf, nil, 0))
		textLines(late, mid[0], mid[1], e.Label, 12.5, 16, PaintRole(RoleMuted))
	}
}

func textLines(items *[]Item, cx, cy float32, lines []string, size, lh float32, paint Paint) {
	top := cy - lh*float32(len(lines))/2 + lh/2
	for i, l := range lines {
		if l != "" {
			*items = append(*items, itemText(cx, top+float32(i)*lh, l, size, false, AnchorMiddle, paint))
		}
	}
}

func drawNode(n *Node, c [2]float32, wh [2]float32, items *[]Item) {
	w, h := wh[0], wh[1]
	fill := PaintRole(RoleNode)
	if n.Style.Fill != nil {
		fill = PaintRGB(*n.Style.Fill)
	}
	stroke := PaintRole(RoleBorder)
	if n.Style.Stroke != nil {
		stroke = PaintRGB(*n.Style.Stroke)
	}
	textPaint := PaintRole(RoleText)
	if n.Style.Color != nil {
		textPaint = PaintRGB(*n.Style.Color)
	} else if n.Style.Fill != nil {
		textPaint = PaintRGB(Contrast(*n.Style.Fill))
	}
	x0, y0, x1, y1 := c[0]-w/2, c[1]-h/2, c[0]+w/2, c[1]+h/2
	k := float32(11)
	poly := func(pts []P) {
		*items = append(*items, itemPoly(pts, true, &fill, &stroke, 1.3))
	}
	switch n.Shape {
	case ShapeRect:
		*items = append(*items, itemRect(x0, y0, w, h, 3, &fill, &stroke, 1.3))
	case ShapeRound:
		*items = append(*items, itemRect(x0, y0, w, h, 10, &fill, &stroke, 1.3))
	case ShapeStadium:
		*items = append(*items, itemRect(x0, y0, w, h, h/2, &fill, &stroke, 1.3))
	case ShapeSubroutine:
		*items = append(*items, itemRect(x0, y0, w, h, 3, &fill, &stroke, 1.3))
		for _, x := range []float32{x0 + 8, x1 - 8} {
			*items = append(*items, itemPoly([]P{{x, y0}, {x, y1}}, false, nil, &stroke, 1.3))
		}
	case ShapeCircle:
		poly(Arc(c[0], c[1], w/2, h/2, 0, 2*math.Pi, 40))
	case ShapeDot:
		line := PaintRole(RoleLine)
		*items = append(*items, itemPoly(Arc(c[0], c[1], w/2, h/2, 0, 2*math.Pi, 24), true, &line, &line, 1))
	case ShapeDiamond:
		poly([]P{{c[0], y0}, {x1, c[1]}, {c[0], y1}, {x0, c[1]}})
	case ShapeHexagon:
		poly([]P{{x0 + k, y0}, {x1 - k, y0}, {x1, c[1]}, {x1 - k, y1}, {x0 + k, y1}, {x0, c[1]}})
	case ShapeParaR:
		poly([]P{{x0 + k, y0}, {x1, y0}, {x1 - k, y1}, {x0, y1}})
	case ShapeParaL:
		poly([]P{{x0, y0}, {x1 - k, y0}, {x1, y1}, {x0 + k, y1}})
	case ShapeTrapU:
		poly([]P{{x0 + k, y0}, {x1 - k, y0}, {x1, y1}, {x0, y1}})
	case ShapeTrapD:
		poly([]P{{x0, y0}, {x1, y0}, {x1 - k, y1}, {x0 + k, y1}})
	case ShapeFlag:
		poly([]P{{x0 + k, y0}, {x1, y0}, {x1, y1}, {x0 + k, y1}, {x0, c[1]}})
	case ShapeCylinder:
		ry := float32(7)
		var pts []P
		pts = append(pts, Arc(c[0], y0+ry, w/2, ry, math.Pi, 2*math.Pi, 20)...)
		pts = append(pts, Arc(c[0], y1-ry, w/2, ry, 0, math.Pi, 20)...)
		poly(pts)
		*items = append(*items, itemPoly(Arc(c[0], y0+ry, w/2, ry, 0, math.Pi, 20), false, nil, &stroke, 1.3))
	}
	if n.Shape != ShapeDot {
		off := float32(0)
		if n.Shape == ShapeCylinder {
			off = 5
		}
		textLines(items, c[0], c[1]+off, n.Label, fontSize, lineH, textPaint)
	}
}

func toPs(route [][2]float32) []P {
	out := make([]P, len(route))
	for i, p := range route {
		out[i] = P{p[0], p[1]}
	}
	return out
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
