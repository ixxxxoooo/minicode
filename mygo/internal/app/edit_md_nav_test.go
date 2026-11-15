package app

import (
	"testing"

	"github.com/egoist/mygo/ui/richtext"
)

func TestMdHeadingOffset(t *testing.T) {
	body := "# One\n\n## Two\n\npara\n"
	doc := richtext.ParseMarkdown(body)
	off := mdHeadingOffset(body, 2, doc) // line 2 = ## Two
	if off < 0 {
		t.Fatal("应找到第二个标题")
	}
	st := doc.ParagraphStyleAt(off)
	if st.Heading != 2 {
		t.Fatalf("want h2 at %d got %d", off, st.Heading)
	}
}
