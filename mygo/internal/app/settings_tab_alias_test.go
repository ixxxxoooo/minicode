package app

import "testing"

func TestResolveSetTabID(t *testing.T) {
	cases := map[string]string{
		"perm":        "permissions",
		"intel":       "lsp",
		"appearance":  "appearance",
		"bogus":       "appearance",
		"":            "appearance",
		"prompts":     "chat",
		"diagnostics": "privacy",
	}
	for in, want := range cases {
		if got := resolveSetTabID(in); got != want {
			t.Fatalf("resolveSetTabID(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestOpenSettingsResolvesAlias(t *testing.T) {
	a, _, _ := settingsApp(t, "appearance")
	a.openSettings("perm")
	if a.setTab != "permissions" {
		t.Fatalf("openSettings(perm) 应落到 permissions,得到 %q", a.setTab)
	}
}
