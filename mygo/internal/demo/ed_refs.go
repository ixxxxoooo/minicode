package demo

// RefHit 是演示模式「引用」面板的一行(LSP 未就绪或 -ed refs:demo 时用)。
type RefHit struct {
	Path      string
	Line, Col int
	Text      string
}

// RefPanelDemo 返回对标设计稿的引用列表演示数据。
func RefPanelDemo(word string) ([]RefHit, bool) {
	if word != "normalize" {
		return nil, false
	}
	long := "let payload = \"" + refPadRepeat("scroll", 80) + "\"; let _ = normalize(payload); // 尾注释"
	return []RefHit{
		{"src/router.rs", 11, 18, "        let key = normalize(path);"},
		{"src/router.rs", 17, 18, "        let key = normalize(path);"},
		{"src/router.rs", 22, 3, "fn normalize(path: &str) -> String {"},
		{"tests/router_test.rs", 7, 15, "    assert_eq!(normalize(\"/A/\"), \"/a\");"},
		{"edge/wrap_scroll.ts", 0, 38, long},
		{"/opt/demo/stdlib/unicode.go", 320, 12, "return strings.Contains(s, sub)"},
	}, true
}

func refPadRepeat(seed string, n int) string {
	var b []byte
	for len(b) < n {
		b = append(b, seed...)
	}
	return string(b[:n])
}
