package update

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// appcast 平铺与按通道嵌套两种格式都能读;没有 beta 时退回 stable。
func TestParseAppcast(t *testing.T) {
	flat := `{"version":"0.3.0","url":"https://x/a.dmg","sha256":"ab"}`
	nested := `{"stable":{"version":"0.3.0","url":"https://x/a.dmg","sha256":"ab"},"beta":{"version":"0.4.0-b1","url":"https://x/b.dmg","sha256":"cd"}}`
	both := `{"version":"0.3.0","url":"https://x/a.dmg","sha256":"ab","stable":{"version":"0.3.0","url":"https://x/a.dmg","sha256":"ab"}}`
	cases := []struct{ name, body, ch, wantVer string }{
		{"平铺", flat, "", "0.3.0"},
		{"嵌套 stable", nested, "", "0.3.0"},
		{"嵌套 beta", nested, "beta", "0.4.0-b1"},
		{"平铺请求 beta 退回平铺", flat, "beta", "0.3.0"},
		{"两种格式并存", both, "", "0.3.0"},
		{"空对象", `{}`, "", ""},
	}
	for _, c := range cases {
		ver, _, _, err := parseAppcast(strings.NewReader(c.body), c.ch)
		if err != nil || ver != c.wantVer {
			t.Errorf("%s:得到 %q / %v,想要 %q", c.name, ver, err, c.wantVer)
		}
	}
}

func TestPkgFileName(t *testing.T) {
	for in, want := range map[string]string{
		"https://github.com/o/r/releases/download/v0.3.0/MiniCode-0.3.0.dmg":     "download.dmg",
		"https://github.com/o/r/releases/download/v0.3.0/MiniCode-0.3.0.ZIP?x=1": "download.zip",
		"https://example.com/latest":                                             "download.pkg",
	} {
		if got := pkgFileName(in); got != want {
			t.Errorf("pkgFileName(%q) = %q,想要 %q", in, got, want)
		}
	}
}

// 端到端:本地服务器给出 appcast 与 zip 包 → 检查 → 下载 → 校验 → 解包 → 生成替换脚本。
// 回归:下载文件原先固定叫 download.pkg,解包按扩展名分派会走错分支,更新永远失败。
func TestRunnerEndToEnd(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "pkg.zip")
	f, _ := os.Create(zipPath)
	w := zip.NewWriter(f)
	fw, _ := w.Create("MiniCode.app/Contents/Info.plist")
	_, _ = fw.Write([]byte("plist"))
	_ = w.Close()
	_ = f.Close()
	body, _ := os.ReadFile(zipPath)
	sum := sha256.Sum256(body)

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/appcast.json":
			_, _ = rw.Write([]byte(`{"version":"0.3.1","url":"` + srv.URL + `/MiniCode-0.3.1.zip","sha256":"` + hex.EncodeToString(sum[:]) + `"}`))
		case "/MiniCode-0.3.1.zip":
			_, _ = rw.Write(body)
		default:
			http.NotFound(rw, r)
		}
	}))
	defer srv.Close()

	var last Progress
	installed := false
	r := &Runner{
		Fetch:      GitHubAppcastFetcher{URL: srv.URL + "/appcast.json"},
		CurrentVer: "0.3.0-mygo",
		workDir:    filepath.Join(dir, "work"),
		OnProgress: func(p Progress) { last = p },
		InstallFunc: func(staging, pkg string) error {
			installed = true
			return WriteReplaceScript(staging, filepath.Join(dir, "Target.app"), filepath.Join(dir, "replace.sh"))
		},
	}
	r.Run(context.Background())
	if last.Err != "" || last.Phase != PhaseReady || !installed {
		t.Fatalf("更新流程没走完:%+v installed=%v", last, installed)
	}

	// 已是最新:同版本(带后缀)与更旧的版本都不算有更新
	for _, cur := range []string{"0.3.1", "0.3.1-mygo", "0.4.0"} {
		r2 := &Runner{Fetch: GitHubAppcastFetcher{URL: srv.URL + "/appcast.json"}, CurrentVer: cur, workDir: t.TempDir(), OnProgress: func(p Progress) { last = p }}
		r2.Run(context.Background())
		if !last.UpToDate {
			t.Errorf("当前 %s 应判为已是最新:%+v", cur, last)
		}
	}

	// 404 要报错,不能把错误页当包存下来
	r3 := &Runner{Fetch: GitHubAppcastFetcher{URL: srv.URL + "/nope.json"}, CurrentVer: "0.3.0", workDir: t.TempDir(), OnProgress: func(p Progress) { last = p }}
	r3.Run(context.Background())
	if last.Err == "" || last.ErrStage != "check" {
		t.Errorf("appcast 404 应报 check 阶段错误:%+v", last)
	}
}
