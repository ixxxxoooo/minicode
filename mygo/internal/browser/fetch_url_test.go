package browser

import (
	"strings"
	"testing"
)

// 地址补全:本机默认 http,公网默认 https,明显敲错的不请求;httpbin.org 这类以 http 开头的域名不能被当成带协议。
func TestFullURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"example.com", "https://example.com", true},
		{"httpbin.org/get", "https://httpbin.org/get", true},
		{"localhost:3000", "http://localhost:3000", true},
		{"localhost:1/x", "http://localhost:1/x", true},
		{"127.0.0.1:8080/a", "http://127.0.0.1:8080/a", true},
		{"foo.local", "http://foo.local", true},
		{"foo", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := fullURL(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("fullURL(%q) = %q,%v 期望 %q,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestFriendlyNetErr(t *testing.T) {
	cases := map[string]string{
		`dial tcp 127.0.0.1:1: connect: connection refused`: "连接被拒绝",
		`dial tcp: lookup x.invalid: no such host`:          "找不到这个域名",
		`context deadline exceeded (Client.Timeout)`:        "连接超时",
		`x509: certificate signed by unknown authority`:     "证书校验失败",
		`weird`: "无法连接",
	}
	for in, want := range cases {
		if got := friendlyNetErr(errString(in)); !strings.Contains(got, want) {
			t.Errorf("%q → %q,应含 %q", in, got, want)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }
