package app

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/lsp"
)

// lspInlayTick 请求并应用内联提示(仅当前活动文档)。
func (a *App) lspInlayTick(tb *ideTab) {
	if tb.st == nil || tb.buf == nil || !a.prefs.Bool("editor.inlay", true) {
		return
	}
	s := a.lspReadyFor(tb.path)
	if s == nil {
		return
	}
	pos := tb.st.CursorPos()
	line := tb.buf.Line(pos.Line)
	start := lsp.Position{Line: pos.Line, Character: 0}
	end := lsp.Position{Line: pos.Line, Character: lsp.ColToUTF16(line, len(line))}
	ver := tb.buf.Version()
	if tb.inlayVer == ver {
		return
	}
	tb.inlayVer = ver
	uri := lsp.PathToURI(filepath.Join(a.proj, tb.path))
	a.lspJob("inlay", s, func() func() {
		hints, err := s.c.InlayHint(uri, lsp.Range{Start: start, End: end})
		return func() {
			if err != nil || tb.buf == nil || tb.buf.Version() != ver {
				return
			}
			var ins []editor.Inlay
			for _, h := range hints {
				off, ok := lspOffset(tb, h.Position)
				if !ok {
					continue
				}
				txt := strings.TrimSpace(h.Label)
				if txt == "" {
					continue
				}
				ins = append(ins, editor.Inlay{Offset: off, Text: " " + txt, Trailing: false})
			}
			if len(ins) > 24 {
				ins = ins[:24]
			}
			tb.st.SetInlays(ins)
		}
	})
}
