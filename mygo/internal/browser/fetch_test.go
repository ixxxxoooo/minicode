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

// 只有 <p> 的极简页面(example.com):标题取 <title>,正文不能丢。
func TestExtractHTMLBareParagraph(t *testing.T) {
	body := []byte(`<!doctype html><html><head><title>Example Domain</title></head><body><p>This domain is for use in documentation examples.</p></body></html>`)
	pg, err := extractHTML(body, "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if pg.Title() != "Example Domain" {
		t.Fatalf("标题应取 <title>:%q", pg.Title())
	}
	var all string
	for _, s := range pg.Sections {
		all += s.Body
	}
	if !strings.Contains(all, "documentation examples") {
		t.Fatalf("正文丢失:%+v", pg.Sections)
	}
}
