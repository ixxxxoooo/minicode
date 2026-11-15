package customize

import "testing"

func TestParseEnvLines(t *testing.T) {
	m, err := ParseEnvLines("FOO=bar\n# comment\nBAZ=\n")
	if err != nil || m["FOO"] != "bar" || m["BAZ"] != "" {
		t.Fatalf("parse: %v %+v", err, m)
	}
	if _, err := ParseEnvLines("badline"); err == nil {
		t.Fatal("应拒绝无等号行")
	}
}
