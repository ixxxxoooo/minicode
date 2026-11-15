package claudecfg

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeProvider(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"m1"}]}`))
	}))
	defer okSrv.Close()

	p := Provider{ID: "t", BaseURL: okSrv.URL + "/anthropic"}
	res := ProbeProvider(p, "good")
	if !res.OK || res.Kind != ProbeOK {
		t.Fatalf("应连通:%+v", res)
	}
	res = ProbeProvider(p, "bad")
	if res.OK || res.Kind != ProbeAuth {
		t.Fatalf("坏密钥应 401:%+v", res)
	}

	nf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer nf.Close()
	res = ProbeProvider(Provider{ID: "t", BaseURL: nf.URL}, "k")
	if res.OK || res.Kind != ProbeNotFound {
		t.Fatalf("全 404 应归类 not_found:%+v", res)
	}

	res = ProbeProvider(Provider{BaseURL: okSrv.URL}, "")
	if res.Kind != ProbeNoKey {
		t.Fatalf("无密钥:%+v", res)
	}
}
