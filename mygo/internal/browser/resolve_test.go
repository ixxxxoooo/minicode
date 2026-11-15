package browser

import "testing"

// 相对链接按页面所在目录解析;不能打开的协议与纯锚点返回空串。
func TestResolveURL(t *testing.T) {
	cases := []struct{ base, href, want string }{
		{"example.com/docs/a.html", "b.html", "https://example.com/docs/b.html"},
		{"example.com/docs/a.html", "../x", "https://example.com/x"},
		{"example.com/docs/a.html", "/root", "https://example.com/root"},
		{"example.com/docs/a.html", "?q=1", "https://example.com/docs/a.html?q=1"},
		{"example.com/docs/a.html", "//cdn.com/x", "https://cdn.com/x"},
		{"example.com/a", "http://other.org/p#frag", "http://other.org/p"},
		{"localhost:8765", "b.html", "https://localhost:8765/b.html"},
		{"example.com/a", "#top", ""},
		{"example.com/a", "mailto:x@y.z", ""},
		{"example.com/a", "javascript:void(0)", ""},
	}
	for _, c := range cases {
		if got := resolveURL(c.base, c.href); got != c.want {
			t.Errorf("resolveURL(%q, %q) = %q, 期望 %q", c.base, c.href, got, c.want)
		}
	}
}
