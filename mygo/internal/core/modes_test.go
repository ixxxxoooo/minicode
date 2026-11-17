package core

import "testing"

func TestParseModeAliases(t *testing.T) {
	cases := []struct {
		in   string
		want PermissionMode
		ok   bool
	}{
		{"manual", ModeDefault, true},
		{"default", ModeDefault, true},
		{"acceptEdits", ModeAcceptEdits, true},
		{"bypassPermissions", ModeBypassPermissions, true},
		{"nope", ModeDefault, false},
	}
	for _, c := range cases {
		got, ok := ParseMode(c.in)
		if ok != c.ok || got != c.want {
			t.Fatalf("ParseMode(%q) = %v,%v 要 %v,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestCLILiveDefaultIsManual(t *testing.T) {
	if ModeDefault.CLI() != "default" || ModeDefault.CLILive() != "manual" {
		t.Fatalf("default CLI=%q live=%q", ModeDefault.CLI(), ModeDefault.CLILive())
	}
	if ModeBypassPermissions.CLI() != ModeBypassPermissions.CLILive() {
		t.Fatal("bypass 启动与运行中应一致")
	}
}
