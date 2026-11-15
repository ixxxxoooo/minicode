package mermaid

import (
	"math"
	"sort"
)

const (
	nodeSep    float32 = 36
	dummySep   float32 = 14
	rankSep    float32 = 52
	labelExtra float32 = 22
)

type edgeIn struct {
	from, to int
	hasLabel bool
}

type layoutResult struct {
	centers [][2]float32
	routes  [][][2]float32
}

func isotonic(z []float32) []float32 {
	type block struct {
		m float32
		n int
	}
	var blocks []block
	for _, v := range z {
		blocks = append(blocks, block{v, 1})
		for len(blocks) > 1 {
			a, b := blocks[len(blocks)-2], blocks[len(blocks)-1]
			if a.m <= b.m {
				break
			}
			n := a.n + b.n
			blocks = blocks[:len(blocks)-2]
			blocks = append(blocks, block{(a.m*float32(a.n) + b.m*float32(b.n)) / float32(n), n})
		}
	}
	var out []float32
	for _, b := range blocks {
		for i := 0; i < b.n; i++ {
			out = append(out, b.m)
		}
	}
	return out
}

func crossings(a, b []int, down [][]int, pos []int) int {
	var es [][2]int
	for _, u := range a {
		for _, v := range down[u] {
			for _, bv := range b {
				if bv == v {
					es = append(es, [2]int{pos[u], pos[v]})
				}
			}
		}
	}
	c := 0
	for i := 0; i < len(es); i++ {
		for j := i + 1; j < len(es); j++ {
			if (es[i][0] < es[j][0] && es[i][1] > es[j][1]) || (es[i][0] > es[j][0] && es[i][1] < es[j][1]) {
				c++
			}
		}
	}
	return c
}

func layoutFlow(dir Dir, sizes [][2]float32, edges []edgeIn, rectlike []bool, groups []int) layoutResult {
	n := len(sizes)
	horiz := dir == DirLR || dir == DirRL
	vw := make([]float32, n)
	vh := make([]float32, n)
	for i, s := range sizes {
		if horiz {
			vw[i], vh[i] = s[1], s[0]
		} else {
			vw[i], vh[i] = s[0], s[1]
		}
	}
	group := append([]int(nil), groups...)

	adj := make([][]struct{ to, k int }, n)
	for k, e := range edges {
		if e.from != e.to {
			adj[e.from] = append(adj[e.from], struct{ to, k int }{e.to, k})
		}
	}
	state := make([]int, n)
	rev := make([]bool, len(edges))
	var dfs func(u int)
	dfs = func(u int) {
		state[u] = 1
		for _, x := range adj[u] {
			switch state[x.to] {
			case 0:
				dfs(x.to)
			case 1:
				rev[x.k] = true
			}
		}
		state[u] = 2
	}
	for u := 0; u < n; u++ {
		if state[u] == 0 {
			dfs(u)
		}
	}
	type dagEdge struct{ u, v, k int }
	var dag []dagEdge
	for k, e := range edges {
		if e.from == e.to {
			continue
		}
		if rev[k] {
			dag = append(dag, dagEdge{e.to, e.from, k})
		} else {
			dag = append(dag, dagEdge{e.from, e.to, k})
		}
	}
	indeg := make([]int, n)
	out := make([][]int, n)
	for _, e := range dag {
		indeg[e.v]++
		out[e.u] = append(out[e.u], e.v)
	}
	rank := make([]int, n)
	var queue []int
	for i := 0; i < n; i++ {
		if indeg[i] == 0 {
			queue = append(queue, i)
		}
	}
	for head := 0; head < len(queue); head++ {
		u := queue[head]
		for _, v := range out[u] {
			if rank[v] < rank[u]+1 {
				rank[v] = rank[u] + 1
			}
			indeg[v]--
			if indeg[v] == 0 {
				queue = append(queue, v)
			}
		}
	}
	vrank := append([]int(nil), rank...)
	type chainRec struct {
		k     int
		chain []int
	}
	var chains []chainRec
	for _, e := range dag {
		chain := []int{e.u}
		for r := rank[e.u] + 1; r < rank[e.v]; r++ {
			vrank = append(vrank, r)
			vw = append(vw, 0)
			vh = append(vh, 0)
			group = append(group, math.MaxInt)
			chain = append(chain, len(vrank)-1)
		}
		chain = append(chain, e.v)
		chains = append(chains, chainRec{k: e.k, chain: chain})
	}
	total := len(vrank)
	depth := 0
	for _, r := range vrank {
		if r+1 > depth {
			depth = r + 1
		}
	}
	up := make([][]int, total)
	down := make([][]int, total)
	for _, ch := range chains {
		for w := 0; w+1 < len(ch.chain); w++ {
			down[ch.chain[w]] = append(down[ch.chain[w]], ch.chain[w+1])
			up[ch.chain[w+1]] = append(up[ch.chain[w+1]], ch.chain[w])
		}
	}
	layers := make([][]int, depth)
	for v := 0; v < total; v++ {
		layers[vrank[v]] = append(layers[vrank[v]], v)
	}
	pos := make([]int, total)
	indexLayers := func() {
		for li := range layers {
			for i, v := range layers[li] {
				pos[v] = i
			}
		}
	}
	indexLayers()
	score := func() int {
		s := 0
		for l := 0; l+1 < depth; l++ {
			s += crossings(layers[l], layers[l+1], down, pos)
		}
		return s
	}
	bestScore, bestLayers := score(), cloneLayers(layers)
	for round := 0; round < 10; round++ {
		var order []int
		if round%2 == 0 {
			for l := 1; l < depth; l++ {
				order = append(order, l)
			}
		} else {
			for l := depth - 2; l >= 0; l-- {
				order = append(order, l)
			}
		}
		for _, l := range order {
			var nb [][]int
			if round%2 == 0 {
				nb = up
			} else {
				nb = down
			}
			type kvPair struct {
				k float32
				v int
			}
			var pairs []kvPair
			for _, v := range layers[l] {
				k := float32(pos[v])
				if len(nb[v]) > 0 {
					var sum float32
					for _, u := range nb[v] {
						sum += float32(pos[u])
					}
					k = sum / float32(len(nb[v]))
				}
				pairs = append(pairs, kvPair{k, v})
			}
			sort.Slice(pairs, func(i, j int) bool { return pairs[i].k < pairs[j].k })
			var ord []int
			for _, kv := range pairs {
				ord = append(ord, kv.v)
			}
			gkey := func(v int) float32 {
				if group[v] >= 1<<29 {
					for i, x := range ord {
						if x == v {
							return float32(i)
						}
					}
					return 0
				}
				var m []int
				for i, x := range ord {
					if group[x] == group[v] {
						m = append(m, i)
					}
				}
				if len(m) == 0 {
					return 0
				}
				sum := 0
				for _, i := range m {
					sum += i
				}
				return float32(sum) / float32(len(m))
			}
			type gk struct {
				k    float32
				v, i int
			}
			var keys []gk
			for i, v := range ord {
				keys = append(keys, gk{gkey(v) + float32(i)*1e-3, v, i})
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i].k < keys[j].k })
			ord = ord[:0]
			for _, x := range keys {
				ord = append(ord, x.v)
			}
			layers[l] = ord
			for i, v := range layers[l] {
				pos[v] = i
			}
		}
		if s := score(); s < bestScore {
			bestScore, bestLayers = s, cloneLayers(layers)
		}
	}
	layers = bestLayers
	indexLayers()
	sep := func(a, b int) float32 {
		g := (vw[a] + vw[b]) / 2
		if vw[a] == 0 || vw[b] == 0 {
			return g + dummySep
		}
		return g + nodeSep
	}
	x := make([]float32, total)
	for _, layer := range layers {
		cur := float32(0)
		for i, v := range layer {
			if i > 0 {
				cur += sep(layer[i-1], v)
			}
			x[v] = cur
		}
	}
	for round := 0; round < 8; round++ {
		var order []int
		if round%2 == 0 {
			for l := 0; l < depth; l++ {
				order = append(order, l)
			}
		} else {
			for l := depth - 1; l >= 0; l-- {
				order = append(order, l)
			}
		}
		for _, l := range order {
			var nb [][]int
			if round%2 == 0 {
				nb = up
			} else {
				nb = down
			}
			layer := layers[l]
			offs := make([]float32, len(layer))
			for i := 1; i < len(layer); i++ {
				offs[i] = offs[i-1] + sep(layer[i-1], layer[i])
			}
			var z []float32
			for i, v := range layer {
				var target float32
				if len(nb[v]) == 0 {
					target = x[v]
				} else {
					var sum float32
					for _, u := range nb[v] {
						sum += x[u]
					}
					target = sum/float32(len(nb[v])) - offs[i]
				}
				z = append(z, target)
			}
			iso := isotonic(z)
			for i, m := range iso {
				x[layer[i]] = m + offs[i]
			}
		}
	}
	labeledGap := make([]bool, depth)
	for _, ch := range chains {
		if ch.k < len(edges) && edges[ch.k].hasLabel {
			mid := (len(ch.chain) - 1) / 2
			labeledGap[vrank[ch.chain[mid]]] = true
		}
	}
	layerH := make([]float32, depth)
	for l, layer := range layers {
		var maxH float32
		for _, v := range layer {
			if vh[v] > maxH {
				maxH = vh[v]
			}
		}
		layerH[l] = maxH
	}
	yc := make([]float32, depth)
	var y float32
	for l := 0; l < depth; l++ {
		yc[l] = y + layerH[l]/2
		y += layerH[l] + rankSep
		if labeledGap[l] {
			y += labelExtra
		}
	}
	totalH := y - rankSep
	if totalH < 0 {
		totalH = 0
	}
	outPorts := make([][][2]float32, n)
	inPorts := make([][][2]float32, n)
	for ci, ch := range chains {
		outPorts[ch.chain[0]] = append(outPorts[ch.chain[0]], [2]float32{x[ch.chain[1]], float32(ci)})
		inPorts[ch.chain[len(ch.chain)-1]] = append(inPorts[ch.chain[len(ch.chain)-1]], [2]float32{x[ch.chain[len(ch.chain)-2]], float32(ci)})
	}
	offsetPort := func(ports [][2]float32, node int, ci int, rect bool, vwNode float32) float32 {
		if len(ports) < 2 || !rect {
			return 0
		}
		sort.Slice(ports, func(i, j int) bool {
			if ports[i][0] == ports[j][0] {
				return ports[i][1] < ports[j][1]
			}
			return ports[i][0] < ports[j][0]
		})
		k := len(ports)
		idx := 0
		for i, p := range ports {
			if int(p[1]) == ci {
				idx = i
				break
			}
		}
		spread := vwNode * 0.7
		if spread > 16*float32(k-1) {
			spread = 16 * float32(k-1)
		}
		return -spread/2 + spread*float32(idx)/float32(k-1)
	}
	routes := make([][][2]float32, len(edges))
	for ci, ch := range chains {
		chain := ch.chain
		a, b := chain[0], chain[len(chain)-1]
		var pts [][2]float32
		outOff := offsetPort(outPorts[a], a, ci, a < len(rectlike) && rectlike[a], vw[a])
		pts = append(pts, P{x[a] + outOff, yc[vrank[a]] + vh[a]/2})
		for _, d := range chain[1 : len(chain)-1] {
			pts = append(pts, P{x[d], yc[vrank[d]]})
		}
		inOff := offsetPort(inPorts[b], b, ci, b < len(rectlike) && rectlike[b], vw[b])
		pts = append(pts, P{x[b] + inOff, yc[vrank[b]] - vh[b]/2})
		var skel []P
		skel = append(skel, pts[0])
		for w := 0; w+1 < len(pts); w++ {
			if abs(pts[w][0]-pts[w+1][0]) > 1 {
				mid := (pts[w][1] + pts[w+1][1]) / 2
				skel = append(skel, P{pts[w][0], mid}, P{pts[w+1][0], mid})
			}
			skel = append(skel, pts[w+1])
		}
		smooth := Chaikin(skel, 3)
		if rev[ch.k] {
			for i, j := 0, len(smooth)-1; i < j; i, j = i+1, j-1 {
				smooth[i], smooth[j] = smooth[j], smooth[i]
			}
		}
		route := make([][2]float32, len(smooth))
		for i, p := range smooth {
			route[i] = toReal(dir, p, totalH)
		}
		routes[ch.k] = route
	}
	centers := make([][2]float32, n)
	for v := 0; v < n; v++ {
		centers[v] = toReal(dir, P{x[v], yc[vrank[v]]}, totalH)
	}
	return layoutResult{centers: centers, routes: routes}
}

func cloneLayers(in [][]int) [][]int {
	out := make([][]int, len(in))
	for i, l := range in {
		out[i] = append([]int(nil), l...)
	}
	return out
}

func toReal(dir Dir, p P, totalH float32) [2]float32 {
	switch dir {
	case DirTB:
		return p
	case DirBT:
		return [2]float32{p[0], totalH - p[1]}
	case DirLR:
		return [2]float32{p[1], p[0]}
	case DirRL:
		return [2]float32{totalH - p[1], p[0]}
	default:
		return p
	}
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
