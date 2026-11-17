package app

import (
	"testing"
)

// TestSidePaneSashAboveHeader 展开时标题行应位于内容区上方(sash 在标题上缘,不在内容区顶)。
func TestSidePaneSashAboveHeader(t *testing.T) {
	a, tt := demoIDE(t)
	a.nav = "files"
	a.sideOpen = true
	a.ide.openFile("src/router.rs", true)
	a.ide.sections["大纲"] = true
	a.ide.sections["时间线"] = false
	tt.Frame()
	tt.Frame()
	tt.Frame()
	hdr, ok := tt.Find("大纲")
	if !ok {
		t.Fatal("找不到大纲标题")
	}
	body, ok := tt.Find("match_path")
	if !ok {
		t.Fatal("找不到大纲内容区")
	}
	if body.Y <= hdr.Y {
		t.Fatalf("内容区应在标题下方,hdr.Y=%v body.Y=%v", hdr.Y, body.Y)
	}
	if body.Y-hdr.Y < 20 {
		t.Fatalf("标题行高度异常,hdr.Y=%v body.Y=%v", hdr.Y, body.Y)
	}
}

func TestFileIconIDMatchesTree(t *testing.T) {
	cases := []struct{ name, want string }{
		{"main.go", "badge:GO"},
		{"router.rs", "badge:RS"},
		{"config.json", "badge:{}"},
		{"readme.md", "badge:MD"},
		{"unknown", "fileText"},
	}
	for _, tc := range cases {
		if got := fileIconID(tc.name); got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}
