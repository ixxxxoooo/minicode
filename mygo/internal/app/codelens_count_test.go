package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/workspace"
)

func TestCountGoTests(t *testing.T) {
	a := &App{ide: newIDE()}
	fs := workspace.NewMem("demo", map[string]string{
		"pkg/foo.go":      "package pkg\nfunc FooBar() {}\n",
		"pkg/foo_test.go": "package pkg\nfunc TestFooBar(t *testing.T) {}\nfunc TestOther(t *testing.T) {}\n",
	})
	a.ide.useFS(fs, nil, nil)
	if n := a.countGoTests("pkg/foo.go", "FooBar"); n != 1 {
		t.Fatalf("期望 1 个 TestFooBar, 得 %d", n)
	}
}
