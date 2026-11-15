package mermaid

import (
	"strings"
	"unicode"
)

type Dir int

const (
	DirTB Dir = iota
	DirBT
	DirLR
	DirRL
)

type Shape int

const (
	ShapeRect Shape = iota
	ShapeRound
	ShapeStadium
	ShapeSubroutine
	ShapeCylinder
	ShapeCircle
	ShapeDiamond
	ShapeHexagon
	ShapeParaR
	ShapeParaL
	ShapeTrapU
	ShapeTrapD
	ShapeFlag
	ShapeDot
)

type LineKind int

const (
	LineSolid LineKind = iota
	LineDotted
	LineThick
)

type Head int

const (
	HeadNone Head = iota
	HeadArrow
	HeadCircle
	HeadCross
)

type NodeStyle struct {
	Fill, Stroke, Color *uint32
}

type Node struct {
	Label []string
	Shape Shape
	Style NodeStyle
}

type Edge struct {
	From, To int
	Label    []string
	Line     LineKind
	End      Head
	Start    Head
}

type Sub struct {
	Title  string
	Parent *int
	Nodes  []int
}

type classPair struct {
	i    int
	name string
}

type Flow struct {
	Dir   Dir
	Nodes []Node
	Edges []Edge
	Subs  []Sub
	Index map[string]int
}

func newFlow(dir Dir) *Flow {
	return &Flow{Dir: dir, Index: map[string]int{}}
}

func (f *Flow) ensure(id string) int {
	if i, ok := f.Index[id]; ok {
		return i
	}
	f.Nodes = append(f.Nodes, Node{Label: []string{id}, Shape: ShapeRect})
	i := len(f.Nodes) - 1
	f.Index[id] = i
	return i
}

type link struct {
	line  LineKind
	start Head
	end   Head
	label *string
}

func headOf(c rune) Head {
	switch c {
	case '>':
		return HeadArrow
	case 'o':
		return HeadCircle
	case 'x':
		return HeadCross
	default:
		return HeadNone
	}
}

func at(s []rune, i int) rune {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

func parseLink(s []rune, i int) (link, int, bool) {
	start := HeadNone
	if at(s, i) == '<' {
		start = HeadArrow
		i++
	} else if (at(s, i) == 'o' || at(s, i) == 'x') && (at(s, i+1) == '-' || at(s, i+1) == '=') {
		start = headOf(at(s, i))
		i++
	}
	runStart := i
	isRun := func(c rune) bool { return c == '-' || c == '=' || c == '.' }
	for isRun(at(s, i)) {
		i++
	}
	run := string(s[runStart:i])
	if len(run) < 2 {
		return link{}, i, false
	}
	line := LineSolid
	if strings.ContainsRune(run, '=') {
		line = LineThick
	} else if strings.ContainsRune(run, '.') {
		line = LineDotted
	}
	var label *string
	end := HeadNone
	boundary := func(c rune) bool { return c == 0 || unicode.IsSpace(c) || c == '|' }
	if at(s, i) == '>' {
		end = HeadArrow
		i++
	} else if (at(s, i) == 'o' || at(s, i) == 'x') && boundary(at(s, i+1)) && len(run) >= 2 {
		end = headOf(at(s, i))
		i++
	} else if at(s, i) == ' ' && len(run) == 2 {
		j := i
		for {
			for j < len(s) && !isRun(s[j]) {
				j++
			}
			if j >= len(s) {
				return link{}, i, false
			}
			k0 := j
			for isRun(at(s, j)) {
				j++
			}
			if j-k0 >= 2 {
				lbl := strings.TrimSpace(string(s[i:k0]))
				label = &lbl
				if at(s, j) == '>' {
					j++
					end = HeadArrow
				} else if (at(s, j) == 'o' || at(s, j) == 'x') && boundary(at(s, j+1)) {
					end = headOf(at(s, j))
					j++
				}
				i = j
				break
			}
		}
	} else if len(run) >= 3 {
		end = HeadNone
	} else {
		return link{}, i, false
	}
	k := i
	for at(s, k) == ' ' {
		k++
	}
	if at(s, k) == '|' && label == nil {
		from := k + 1
		to := -1
		for j := from; j < len(s); j++ {
			if s[j] == '|' {
				to = j
				break
			}
		}
		if to < 0 {
			return link{}, i, false
		}
		lbl := string(s[from:to])
		label = &lbl
		i = to + 1
	}
	return link{line: line, start: start, end: end, label: label}, i, true
}

var shapeTable = []struct {
	open   string
	closes []string
	shape  Shape
}{
	{"(((", []string{")))"}, ShapeCircle},
	{"((", []string{"))"}, ShapeCircle},
	{"([", []string{"])"}, ShapeStadium},
	{"(", []string{")"}, ShapeRound},
	{"[[", []string{"]]"}, ShapeSubroutine},
	{"[(", []string{"])"}, ShapeCylinder},
	{`[/`, []string{"/]"}, ShapeParaR},
	{`[/`, []string{`\]`}, ShapeTrapU},
	{`[\`, []string{`\]`}, ShapeParaL},
	{`[\`, []string{"/]"}, ShapeTrapD},
	{"[", []string{"]"}, ShapeRect},
	{"{{", []string{"}}"}, ShapeHexagon},
	{"{", []string{"}"}, ShapeDiamond},
	{">", []string{"]"}, ShapeFlag},
}

func starts(s []rune, i int, pat string) bool {
	pr := []rune(pat)
	for k, c := range pr {
		if i+k >= len(s) || s[i+k] != c {
			return false
		}
	}
	return true
}

func parseNode(flow *Flow, s []rune, i int, classes *[]classPair) (int, int, bool) {
	idStart := i
	for i < len(s) && (unicode.IsLetter(s[i]) || unicode.IsDigit(s[i]) || s[i] == '_') {
		i++
	}
	if i == idStart {
		return 0, i, false
	}
	id := string(s[idStart:i])
	var best *struct {
		k, clen, olen int
		shape         Shape
		body          int
	}
	for _, sh := range shapeTable {
		if !starts(s, i, sh.open) {
			continue
		}
		olen := len([]rune(sh.open))
		body := i + olen
		search := body
		if body < len(s) && s[body] == '"' {
			for q := body + 1; q < len(s); q++ {
				if s[q] == '"' {
					search = q + 1
					break
				}
			}
		}
		for _, c := range sh.closes {
			clen := len([]rune(c))
			for k := search; k < len(s); k++ {
				if starts(s, k, c) {
					cand := struct {
						k, clen, olen int
						shape         Shape
						body          int
					}{k, clen, olen, sh.shape, body}
					if best == nil || cand.k < best.k || (cand.k == best.k && cand.olen > best.olen) {
						cp := cand
						best = &cp
					}
					break
				}
			}
		}
	}
	idx := flow.ensure(id)
	if best != nil {
		label := string(s[best.body:best.k])
		flow.Nodes[idx].Shape = best.shape
		flow.Nodes[idx].Label = SplitLines(label)
		i = best.k + best.clen
	}
	if starts(s, i, ":::") {
		st := i + 3
		e := st
		for e < len(s) && (unicode.IsLetter(s[e]) || unicode.IsDigit(s[e]) || s[e] == '_' || s[e] == '-') {
			e++
		}
		*classes = append(*classes, classPair{i: idx, name: string(s[st:e])})
		i = e
	}
	return idx, i, true
}

func parseStyle(spec string) NodeStyle {
	var st NodeStyle
	for _, kv := range strings.Split(spec, ",") {
		parts := strings.SplitN(kv, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if c, ok := ParseColor(parts[1]); ok {
			k := strings.TrimSpace(parts[0])
			v := c
			switch k {
			case "fill":
				st.Fill = &v
			case "stroke":
				st.Stroke = &v
			case "color":
				st.Color = &v
			}
		}
	}
	return st
}

// Statements 按行拆分语句。
func Statements(src string) []string {
	var out []string
	for _, line := range strings.Split(src, "\n") {
		var cur strings.Builder
		depth := 0
		quote := false
		for _, c := range line {
			switch c {
			case '"':
				quote = !quote
			case '[', '(', '{':
				if !quote {
					depth++
				}
			case ']', ')', '}':
				if !quote {
					depth--
				}
			case ';':
				if !quote && depth <= 0 {
					out = append(out, strings.TrimSpace(cur.String()))
					cur.Reset()
					continue
				}
			}
			cur.WriteRune(c)
		}
		out = append(out, strings.TrimSpace(cur.String()))
	}
	var fin []string
	for _, l := range out {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "%%") {
			continue
		}
		fin = append(fin, l)
	}
	return fin
}

func parseFlow(src string) (*Flow, error) {
	stmts := Statements(src)
	if len(stmts) == 0 {
		return nil, RenderError{Kind: ErrEmpty}
	}
	head := stmts[0]
	if head == "---" {
		found := false
		for i := 1; i < len(stmts); i++ {
			if stmts[i] == "---" && i+1 < len(stmts) {
				head = stmts[i+1]
				stmts = stmts[i+2:]
				found = true
				break
			}
		}
		if !found {
			return nil, RenderError{Kind: ErrEmpty}
		}
	} else {
		stmts = stmts[1:]
	}
	parts := strings.Fields(head)
	dir := DirTB
	if len(parts) > 1 {
		switch strings.ToUpper(parts[1]) {
		case "LR":
			dir = DirLR
		case "RL":
			dir = DirRL
		case "BT":
			dir = DirBT
		}
	}
	flow := newFlow(dir)
	var stack []int
	classDefs := map[string]NodeStyle{}
	var classes []classPair
	for _, line := range stmts {
		lower := strings.ToLower(line)
		if lower == "end" {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		} else if strings.HasPrefix(lower, "subgraph ") || lower == "subgraph" {
			rest := strings.TrimSpace(line[8:])
			title := rest
			if a := strings.Index(rest, "["); a >= 0 {
				if b := strings.LastIndex(rest, "]"); b > a {
					title = strings.Trim(strings.TrimSpace(rest[a+1:b]), "\"")
				}
			} else {
				title = strings.Trim(title, "\"")
			}
			var parent *int
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				parent = &p
			}
			flow.Subs = append(flow.Subs, Sub{Title: title, Parent: parent})
			stack = append(stack, len(flow.Subs)-1)
		} else if strings.HasPrefix(lower, "direction ") || strings.HasPrefix(lower, "linkstyle ") ||
			strings.HasPrefix(lower, "click ") || strings.HasPrefix(lower, "acctitle") || strings.HasPrefix(lower, "accdescr") {
		} else if rest, ok := strings.CutPrefix(line, "classDef "); ok {
			it := strings.Fields(strings.TrimSpace(rest))
			if len(it) >= 2 {
				spec := strings.TrimSpace(strings.TrimPrefix(rest, it[0]))
				st := parseStyle(spec)
				for _, n := range strings.Split(it[0], ",") {
					classDefs[strings.TrimSpace(n)] = st
				}
			}
		} else if rest, ok := strings.CutPrefix(line, "class "); ok {
			idx := strings.LastIndexByte(rest, ' ')
			if idx > 0 {
				ids, name := rest[:idx], strings.TrimSpace(rest[idx+1:])
				for _, id := range strings.Split(ids, ",") {
					i := flow.ensure(strings.TrimSpace(id))
					classes = append(classes, classPair{i: i, name: name})
				}
			}
		} else if rest, ok := strings.CutPrefix(line, "style "); ok {
			it := strings.SplitN(strings.TrimSpace(rest), " ", 2)
			if len(it) == 2 {
				i := flow.ensure(it[0])
				flow.Nodes[i].Style = parseStyle(it[1])
			}
		} else {
			if err := parseChain(flow, line, stack, &classes); err != nil {
				return nil, err
			}
		}
		if len(flow.Nodes) > MaxNodes {
			return nil, RenderError{Kind: ErrTooLarge, N: MaxNodes}
		}
	}
	for _, cp := range classes {
		if st, ok := classDefs[cp.name]; ok {
			flow.Nodes[cp.i].Style = st
		}
	}
	return flow, nil
}

func subHasNode(flow *Flow, idx int) bool {
	for _, s := range flow.Subs {
		for _, n := range s.Nodes {
			if n == idx {
				return true
			}
		}
	}
	return false
}

func parseChain(flow *Flow, line string, stack []int, classes *[]classPair) error {
	s := []rune(line)
	bad := func() error { return RenderError{Kind: ErrSyntax, Arg: line} }
	i := 0
	var prev *struct {
		group []int
		link  link
	}
	for {
		var group []int
		for {
			for i < len(s) && s[i] == ' ' {
				i++
			}
			idx, next, ok := parseNode(flow, s, i, classes)
			if !ok {
				return bad()
			}
			group = append(group, idx)
			if len(stack) > 0 {
				sub := stack[len(stack)-1]
				if !subHasNode(flow, idx) {
					flow.Subs[sub].Nodes = append(flow.Subs[sub].Nodes, idx)
				}
			}
			i = next
			for i < len(s) && s[i] == ' ' {
				i++
			}
			if i < len(s) && s[i] == '&' {
				i++
				continue
			}
			break
		}
		if prev != nil {
			for _, a := range prev.group {
				for _, b := range group {
					var lbl []string
					if prev.link.label != nil {
						lbl = SplitLines(*prev.link.label)
					}
					flow.Edges = append(flow.Edges, Edge{
						From: a, To: b, Label: lbl,
						Line: prev.link.line, End: prev.link.end, Start: prev.link.start,
					})
				}
			}
		}
		for i < len(s) && s[i] == ' ' {
			i++
		}
		if i >= len(s) {
			return nil
		}
		lk, next, ok := parseLink(s, i)
		if !ok {
			return bad()
		}
		i = next
		prev = &struct {
			group []int
			link  link
		}{append([]int(nil), group...), lk}
	}
}
