package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/browser"
)

// 没有任何文字的页面要走空态,而不是一片空白。
func TestPageHasNoText(t *testing.T) {
	cases := []struct {
		name string
		pg   browser.Page
		want bool
	}{
		{"无任何节", browser.Page{URL: "x"}, true},
		{"只有空白", browser.Page{URL: "x", Sections: []browser.Section{{Heading: " ", Body: "\n"}}}, true},
		{"有标题", browser.Page{URL: "x", Sections: []browser.Section{{Heading: "标题"}}}, false},
		{"有正文", browser.Page{URL: "x", Sections: []browser.Section{{Body: "正文"}}}, false},
	}
	for _, c := range cases {
		if got := pageHasNoText(c.pg); got != c.want {
			t.Errorf("%s: 得到 %v,期望 %v", c.name, got, c.want)
		}
	}
}
