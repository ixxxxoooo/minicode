package browser

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 纯 http 站点:自动补的 https 握手失败后退回 http;显式 http:// 直接走 http。
func TestFetchPlainHTTPFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body><main><h1>你好</h1><p>纯 http 站点的正文内容。</p></main></body></html>"))
	}))
	defer srv.Close()
	hostPort := strings.TrimPrefix(srv.URL, "http://")
	// 0.0.0.0 不在「本机」白名单里,所以会先被补成 https
	addr := strings.Replace(hostPort, "127.0.0.1", "0.0.0.0", 1)
	src := HTTPSource{Opts: FetchOpts{Enabled: true, TimeoutSec: 5}}
	for _, in := range []string{addr, "http://" + addr} {
		pg, err := src.Fetch(in)
		if err != nil {
			t.Fatalf("%q 应能打开:%v", in, err)
		}
		if len(pg.Sections) == 0 {
			t.Errorf("%q 应有正文", in)
		}
		if !pg.Insecure {
			t.Errorf("%q 是明文 http 抓取的,Page.Insecure 应为 true(地址栏不能显示安全锁)", in)
		}
	}
}

func TestHTTPSUnsupported(t *testing.T) {
	if !httpsUnsupported(errString("http: server gave HTTP response to HTTPS client")) {
		t.Error("应识别为不支持 https")
	}
	if !httpsUnsupported(errString("dial tcp 1.2.3.4:443: connect: connection refused")) {
		t.Error("443 被拒绝(只开了 80)应退回 http")
	}
	if httpsUnsupported(errString("dial tcp: lookup x: no such host")) {
		t.Error("域名错误不该退回 http")
	}
}
