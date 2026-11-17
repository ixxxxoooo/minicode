package update

import "testing"

// 版本比较:只在远端严格更新时才算有新版本;后缀(-mygo)不参与比较。
func TestIsNewer(t *testing.T) {
	cases := []struct {
		remote, cur string
		want        bool
	}{
		{"0.3.1", "0.3.0", true},
		{"0.10.0", "0.9.9", true},
		{"1.0", "0.9.9", true},
		{"0.3.0", "0.3.0-mygo", false},
		{"0.3.0", "0.3.0", false},
		{"0.1.22", "0.3.0-mygo", false},
		{"v0.4.0", "0.3.0", true},
		{"0.3.0", "0.3", false},
		{"abc", "0.3.0", true},
	}
	for _, c := range cases {
		if got := IsNewer(c.remote, c.cur); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v,想要 %v", c.remote, c.cur, got, c.want)
		}
	}
}
