package mermaid

import (
	"strings"
	"unicode"
)

type Tip int

const (
	TipArrow Tip = iota
	TipNone
	TipCross
	TipAsync
)

type NoteAt struct {
	Over            bool
	Left, Right     bool
	A, B            int
	ParticipantOnly int
}

type SeqEv struct {
	Msg        *SeqMsg
	Note       *SeqNote
	Activate   *int
	Deactivate *int
	Open       *SeqOpen
	Else       *string
	Close      bool
}

type SeqMsg struct {
	From, To             int
	Text                 []string
	Dotted               bool
	Tip                  Tip
	Activate, Deactivate bool
}

type SeqNote struct {
	At   NoteAt
	Text []string
}

type SeqOpen struct {
	Kind, Label string
}

type Part struct {
	ID, Label []string
	IDStr     string
	Actor     bool
}

type Seq struct {
	Parts      []Part
	Evs        []SeqEv
	Title      *string
	Autonumber bool
}

func (s *Seq) ensure(id string) int {
	id = strings.TrimSpace(id)
	for i, p := range s.Parts {
		if p.IDStr == id {
			return i
		}
	}
	s.Parts = append(s.Parts, Part{IDStr: id, Label: []string{id}})
	return len(s.Parts) - 1
}

var seqOps = []struct {
	op     string
	dotted bool
	tip    Tip
}{
	{"-->>", true, TipArrow},
	{"->>", false, TipArrow},
	{"--x", true, TipCross},
	{"-x", false, TipCross},
	{"--)", true, TipAsync},
	{"-)", false, TipAsync},
	{"-->", true, TipNone},
	{"->", false, TipNone},
}

var seqBlocks = []string{"loop", "alt", "opt", "par", "critical", "break", "rect", "box"}

func parseSeq(src string) (*Seq, error) {
	seq := &Seq{}
	stmts := Statements(src)
	if len(stmts) == 0 {
		return nil, RenderError{Kind: ErrEmpty}
	}
	var blocks []bool
	for _, line := range stmts[1:] {
		word, rest, _ := strings.Cut(line, " ")
		rest = strings.TrimSpace(rest)
		lw := strings.ToLower(word)
		switch lw {
		case "participant", "actor":
			id, label := rest, rest
			if a, b, ok := strings.Cut(rest, " as "); ok {
				id, label = a, b
			}
			i := seq.ensure(id)
			seq.Parts[i].Label = SplitLines(label)
			seq.Parts[i].Actor = lw == "actor"
		case "autonumber":
			seq.Autonumber = true
		case "title":
			t := strings.TrimSpace(strings.TrimPrefix(rest, ":"))
			seq.Title = &t
		case "activate", "deactivate":
			i := seq.ensure(rest)
			if lw == "activate" {
				seq.Evs = append(seq.Evs, SeqEv{Activate: &i})
			} else {
				seq.Evs = append(seq.Evs, SeqEv{Deactivate: &i})
			}
		case "else", "and", "option":
			seq.Evs = append(seq.Evs, SeqEv{Else: &rest})
		case "end":
			if len(blocks) > 0 {
				blocks = blocks[:len(blocks)-1]
			}
			seq.Evs = append(seq.Evs, SeqEv{Close: true})
		default:
			if containsStr(seqBlocks, lw) {
				draw := lw != "box" && lw != "rect"
				blocks = append(blocks, draw)
				if draw {
					seq.Evs = append(seq.Evs, SeqEv{Open: &SeqOpen{Kind: lw, Label: rest}})
				}
				continue
			}
			if lw == "note" {
				head, text, ok := strings.Cut(rest, ":")
				if !ok {
					return nil, RenderError{Kind: ErrSyntax, Arg: line}
				}
				lh := strings.ToLower(head)
				var at NoteAt
				if names, ok := strings.CutPrefix(lh, "over "); ok {
					parts := strings.Split(head[len(head)-len(names):], ",")
					a := seq.ensure(strings.TrimSpace(parts[0]))
					b := a
					if len(parts) > 1 {
						b = seq.ensure(strings.TrimSpace(parts[1]))
					}
					if a > b {
						a, b = b, a
					}
					at = NoteAt{Over: true, A: a, B: b}
				} else if strings.HasPrefix(lh, "left of ") {
					at = NoteAt{Left: true, ParticipantOnly: seq.ensure(strings.TrimSpace(head[8:]))}
				} else if strings.HasPrefix(lh, "right of ") {
					at = NoteAt{Right: true, ParticipantOnly: seq.ensure(strings.TrimSpace(head[9:]))}
				} else {
					return nil, RenderError{Kind: ErrSyntax, Arg: line}
				}
				seq.Evs = append(seq.Evs, SeqEv{Note: &SeqNote{At: at, Text: SplitLines(text)}})
				continue
			}
			if lw == "create" || lw == "destroy" || lw == "link" || lw == "links" || lw == "properties" || lw == "details" {
				if r, ok := strings.CutPrefix(rest, "participant "); ok {
					parseParticipant(seq, r, false)
				} else if r, ok := strings.CutPrefix(rest, "actor "); ok {
					parseParticipant(seq, r, true)
				}
				continue
			}
			if err := parseSeqMessage(seq, line); err != nil {
				return nil, err
			}
		}
	}
	if len(seq.Parts) == 0 {
		return nil, RenderError{Kind: ErrEmpty}
	}
	return seq, nil
}

func parseParticipant(seq *Seq, rest string, actor bool) {
	id, label := rest, rest
	if a, b, ok := strings.Cut(rest, " as "); ok {
		id, label = a, b
	}
	i := seq.ensure(id)
	seq.Parts[i].Label = SplitLines(label)
	seq.Parts[i].Actor = actor
}

func parseSeqMessage(seq *Seq, line string) error {
	left, text, ok := strings.Cut(line, ":")
	if !ok {
		text = ""
	}
	var opIdx int = -1
	var op *struct {
		op     string
		dotted bool
		tip    Tip
	}
	for i, c := range left {
		if c == '-' {
			for _, candidate := range seqOps {
				if strings.HasPrefix(left[i:], candidate.op) {
					opIdx = i
					op = &struct {
						op     string
						dotted bool
						tip    Tip
					}{candidate.op, candidate.dotted, candidate.tip}
					break
				}
			}
			if op != nil {
				break
			}
		}
	}
	if op == nil {
		return RenderError{Kind: ErrSyntax, Arg: line}
	}
	from := strings.TrimSpace(left[:opIdx])
	to := strings.TrimSpace(left[opIdx+len(op.op):])
	act, deact := false, false
	if strings.HasPrefix(to, "+") {
		act = true
		to = strings.TrimSpace(to[1:])
	} else if strings.HasPrefix(to, "-") {
		deact = true
		to = strings.TrimSpace(to[1:])
	}
	if from == "" || to == "" {
		return RenderError{Kind: ErrSyntax, Arg: line}
	}
	a, b := seq.ensure(from), seq.ensure(to)
	seq.Evs = append(seq.Evs, SeqEv{Msg: &SeqMsg{
		From: a, To: b, Text: SplitLines(text), Dotted: op.dotted, Tip: op.tip, Activate: act, Deactivate: deact,
	}})
	return nil
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

var _ = unicode.IsSpace
