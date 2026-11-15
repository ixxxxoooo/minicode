package browser

import "testing"

func TestRedactURL(t *testing.T) {
	if got := RedactURL("https://user:pass@example.com/path?q=1#frag"); got != "example.com/path" {
		t.Fatalf("got %q", got)
	}
}
