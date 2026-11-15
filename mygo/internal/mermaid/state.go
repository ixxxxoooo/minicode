package mermaid

import (
	"strings"
)

func renderState(src string) (Scene, error) {
	stmts := Statements(src)
	if len(stmts) == 0 {
		return Scene{}, RenderError{Kind: ErrEmpty}
	}
	flow := newFlow(DirTB)
	var stack []int
	starts, ends := 0, 0
	stateNode := func(name string, isTarget bool) int {
		name = strings.TrimSpace(name)
		if name == "[*]" {
			var id string
			if isTarget {
				ends++
				id = fmtStar("__end", ends)
			} else {
				starts++
				id = fmtStar("__start", starts)
			}
			i := flow.ensure(id)
			flow.Nodes[i].Shape = ShapeDot
			flow.Nodes[i].Label = []string{""}
			return i
		}
		i := flow.ensure(name)
		flow.Nodes[i].Shape = ShapeRound
		return i
	}
	inNote := false
	for _, line := range stmts[1:] {
		if inNote {
			inNote = strings.ToLower(line) != "end note"
			continue
		}
		if line == "}" {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		} else if strings.HasPrefix(line, "direction ") {
			switch strings.TrimSpace(line[10:]) {
			case "LR":
				flow.Dir = DirLR
			case "RL":
				flow.Dir = DirRL
			case "BT":
				flow.Dir = DirBT
			default:
				flow.Dir = DirTB
			}
		} else if strings.HasPrefix(line, "state ") {
			rest := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line[6:]), "{"))
			if desc, id, ok := strings.Cut(rest, " as "); ok {
				i := stateNode(id, false)
				flow.Nodes[i].Label = SplitLines(desc)
			} else if strings.HasSuffix(rest, "<<choice>>") {
				name := strings.TrimSpace(strings.TrimSuffix(rest, "<<choice>>"))
				i := stateNode(name, false)
				flow.Nodes[i].Shape = ShapeDiamond
				flow.Nodes[i].Label = []string{""}
			} else if strings.HasSuffix(line, "{") {
				var parent *int
				if len(stack) > 0 {
					p := stack[len(stack)-1]
					parent = &p
				}
				flow.Subs = append(flow.Subs, Sub{Title: rest, Parent: parent})
				stack = append(stack, len(flow.Subs)-1)
			} else {
				stateNode(rest, false)
			}
		} else if l, r, ok := strings.Cut(line, "-->"); ok {
			target, label := r, (*string)(nil)
			if t, lab, ok2 := strings.Cut(r, ":"); ok2 {
				target, label = t, &lab
			}
			a := stateNode(l, false)
			b := stateNode(target, true)
			for _, i := range []int{a, b} {
				if len(stack) > 0 {
					sub := stack[len(stack)-1]
					if !subHasNode(flow, i) {
						flow.Subs[sub].Nodes = append(flow.Subs[sub].Nodes, i)
					}
				}
			}
			var lbl []string
			if label != nil {
				lbl = SplitLines(strings.TrimSpace(*label))
			}
			flow.Edges = append(flow.Edges, Edge{From: a, To: b, Label: lbl, Line: LineSolid, End: HeadArrow, Start: HeadNone})
		} else if id, desc, ok := strings.Cut(line, ":"); ok {
			i := stateNode(id, false)
			flow.Nodes[i].Label = SplitLines(desc)
		} else if strings.HasPrefix(line, "note ") {
			inNote = !strings.Contains(line, ":")
		} else {
			return Scene{}, RenderError{Kind: ErrSyntax, Arg: line}
		}
	}
	if len(flow.Nodes) == 0 {
		return Scene{}, RenderError{Kind: ErrEmpty}
	}
	return drawFlow(flow), nil
}

func fmtStar(prefix string, n int) string {
	return prefix + itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
