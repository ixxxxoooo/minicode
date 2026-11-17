package app

import "testing"

func TestParseFontFamiliesAndEnv(t *testing.T) {
	if got := parseTermEnv("FOO=bar\n#cmt\nBAD\nBAR=baz"); len(got) != 2 || got[0] != "FOO=bar" {
		t.Fatalf("env parse: %v", got)
	}
	if args := parseShellArgs("-l -i"); len(args) != 2 {
		t.Fatalf("args: %v", args)
	}
}

func TestTermCloseConfirmPref(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a"})
	a.prefs.Set(pTermCloseConfirm, 2)
	if a.termCloseNeedsConfirm(true) {
		t.Fatal("永不确认")
	}
	a.prefs.Set(pTermCloseConfirm, 1)
	if !a.termCloseNeedsConfirm(false) {
		t.Fatal("始终确认")
	}
}
