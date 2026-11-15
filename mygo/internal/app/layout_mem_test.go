package app

import "testing"

func TestResolveStartupMode(t *testing.T) {
	pr := NewPrefsReader(func(key string, def int) int {
		switch key {
		case kStartMode:
			return 2
		case kLastMode:
			return 1
		default:
			return def
		}
	})
	if ResolveStartupMode(pr, false) != ModeIDE {
		t.Fatal("上次=IDE")
	}
	pr2 := NewPrefsReader(func(string, int) int { return 0 })
	if ResolveStartupMode(pr2, false) != ModeAgent {
		t.Fatal("general.start=Agent")
	}
	if ResolveStartupMode(pr2, true) != ModeIDE {
		t.Fatal("MINI_MODE=ide")
	}
}
