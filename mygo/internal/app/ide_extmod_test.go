package app

import (
	"testing"

	"github.com/egoist/mygo/ui/editor"
)

func TestExtmodParticipates(t *testing.T) {
	cases := []struct {
		tb   *ideTab
		want bool
	}{
		{nil, false},
		{&ideTab{path: "a.go", diff: true, buf: editorBuf("x")}, false},
		{&ideTab{path: "a.go", virt: "plan", buf: editorBuf("x")}, false},
		{&ideTab{path: "", buf: editorBuf("x")}, false},
		{&ideTab{path: "a.go", err: "fail"}, false},
		{&ideTab{path: "a.go", buf: editorBuf("x")}, true},
	}
	for _, c := range cases {
		if got := extmodParticipates(c.tb); got != c.want {
			t.Fatalf("participates %+v: got %v want %v", c.tb, got, c.want)
		}
	}
}

func editorBuf(s string) *editor.Buffer { return editor.NewBuffer(s) }
