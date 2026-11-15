package canvas

import "testing"

func TestExtractCanvasFence(t *testing.T) {
	raw := "说明\n```canvas\n{\"blocks\":[{\"id\":\"a\",\"kind\":0,\"text\":\"hi\"}]}\n```"
	got := ExtractCanvasJSON(raw)
	if got == "" || !contains(got, `"blocks"`) {
		t.Fatalf("want json inside fence, got %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || findSub(s, sub))
}

func findSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
