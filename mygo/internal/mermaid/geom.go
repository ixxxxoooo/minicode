package mermaid

import "math"

// P 坐标点。
type P = [2]float32

func Chaikin(pts []P, iters int) []P {
	cur := append([]P(nil), pts...)
	for k := 0; k < iters; k++ {
		if len(cur) < 3 {
			break
		}
		next := []P{cur[0]}
		for i := 0; i+1 < len(cur); i++ {
			a, b := cur[i], cur[i+1]
			next = append(next,
				P{a[0]*0.75 + b[0]*0.25, a[1]*0.75 + b[1]*0.25},
				P{a[0]*0.25 + b[0]*0.75, a[1]*0.25 + b[1]*0.75},
			)
		}
		next = append(next, cur[len(cur)-1])
		cur = next
	}
	return cur
}

func Length(pts []P) float32 {
	var sum float32
	for i := 0; i+1 < len(pts); i++ {
		sum += Dist(pts[i], pts[i+1])
	}
	return sum
}

func Dist(a, b P) float32 {
	return float32(math.Hypot(float64(a[0]-b[0]), float64(a[1]-b[1])))
}

func PointAt(pts []P, d float32) P {
	left := d
	for i := 0; i+1 < len(pts); i++ {
		l := Dist(pts[i], pts[i+1])
		if left <= l && l > 0 {
			t := left / l
			return P{pts[i][0] + (pts[i+1][0]-pts[i][0])*t, pts[i][1] + (pts[i+1][1]-pts[i][1])*t}
		}
		left -= l
	}
	if len(pts) == 0 {
		return P{0, 0}
	}
	return pts[len(pts)-1]
}

func TrimEnd(pts []P, d float32) []P {
	total := Length(pts)
	if total <= d*1.5 || len(pts) < 2 {
		out := make([]P, len(pts))
		copy(out, pts)
		return out
	}
	keep := total - d
	out := []P{pts[0]}
	var acc float32
	for i := 0; i+1 < len(pts); i++ {
		l := Dist(pts[i], pts[i+1])
		if acc+l >= keep {
			out = append(out, PointAt([]P{pts[i], pts[i+1]}, keep-acc))
			return out
		}
		acc += l
		out = append(out, pts[i+1])
	}
	return out
}

func Dashes(pts []P, on, off float32) [][]P {
	total := Length(pts)
	var out [][]P
	for d := float32(0); d < total; {
		end := d + on
		if end > total {
			end = total
		}
		seg := []P{PointAt(pts, d), PointAt(pts, end)}
		out = append(out, seg)
		d += on + off
	}
	return out
}

func ArrowHead(tip, from P, length, halfW float32) []P {
	d := Dist(tip, from)
	if d < 0.001 {
		d = 0.001
	}
	ux, uy := (tip[0]-from[0])/d, (tip[1]-from[1])/d
	base := P{tip[0] - ux*length, tip[1] - uy*length}
	return []P{
		tip,
		{base[0] - uy*halfW, base[1] + ux*halfW},
		{base[0] + uy*halfW, base[1] - ux*halfW},
	}
}

func Arc(cx, cy, rx, ry, a0, a1 float32, steps int) []P {
	out := make([]P, 0, steps+1)
	for i := 0; i <= steps; i++ {
		a := a0 + (a1-a0)*float32(i)/float32(steps)
		out = append(out, P{cx + rx*float32(math.Cos(float64(a))), cy + ry*float32(math.Sin(float64(a)))})
	}
	return out
}
