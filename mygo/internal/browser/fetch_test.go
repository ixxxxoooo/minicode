package browser

import (
	"strings"
	"testing"
)

func TestExtractHTML(t *testing.T) {
	body := []byte(`<!doctype html><html><head><title>Hello</title></head><body><article>
	<h1>Main</h1><p>First paragraph here.</p><pre><code>fn main(){}</code></pre>
	<h2>Sub</h2><p>Second.</p><a href="/next">Next</a></article></body></html>`)
	pg, err := extractHTML(body, "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pg.Title(), "Main") && pg.Title() != "Hello" {
		t.Fatalf("title %q", pg.Title())
	}
	if len(pg.Sections) < 2 {
		t.Fatalf("sections %d", len(pg.Sections))
	}
	if len(pg.Summary) == 0 {
		t.Fatal("summary empty")
	}
}

func TestNormalizeFetchKey(t *testing.T) {
	u := Normalize("https://example.com/path/")
	if u != "example.com/path" {
		t.Fatalf("got %q", u)
	}
}
