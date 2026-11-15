package canvas

import (
	"testing"
	"time"
)

func TestLibraryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	l, err := OpenLibrary(dir)
	if err != nil {
		t.Fatal(err)
	}
	doc := Doc{Title: "测试", Status: "草稿", Blocks: []Block{{ID: "p", Kind: Para, Text: "hi"}}}
	_, id, err := l.New("测试", doc, "你", "新建", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	l2, err := OpenLibrary(dir)
	if err != nil || len(l2.IDs()) != 1 {
		t.Fatal("应能重新打开")
	}
	s, err := l2.Store(id)
	if err != nil || s.Doc.Title != "测试" {
		t.Fatalf("%+v", s)
	}
}
