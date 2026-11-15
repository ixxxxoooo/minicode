package app

import "testing"

func TestParsePathLine(t *testing.T) {
	cases := []struct {
		q    string
		p    string
		line int
		col  int
		ok   bool
	}{
		{":42", "", 41, 0, true},
		{":10:5", "", 9, 4, true},
		{"src/router.rs:42", "src/router.rs", 41, 0, true},
		{"router.rs:3:8", "router.rs", 2, 7, true},
		{"foo", "", 0, 0, false},
		{":", "", 0, 0, false},
	}
	for _, c := range cases {
		p, line, col, ok := parsePathLine(c.q)
		if ok != c.ok || p != c.p || line != c.line || col != c.col {
			t.Fatalf("%q => (%q,%d,%d,%v) want (%q,%d,%d,%v)", c.q, p, line, col, ok, c.p, c.line, c.col, c.ok)
		}
	}
}

func TestGotoLineClamp(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	a.ide.openFile("src/router.rs", true)
	tb := a.ide.cur()
	lines := tb.buf.LineCount()
	a.gotoLineInCurrent(9999)
	if p := tb.st.CursorPos(); p.Line != lines-1 {
		t.Fatalf("超界应夹到末行 %d, got %d", lines-1, p.Line)
	}
}
