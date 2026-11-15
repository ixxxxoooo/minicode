package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// Phase 与界面 update_progress 四态对齐。
type Phase int

const (
	PhaseNone Phase = iota
	PhaseDownload
	PhaseVerify
	PhaseUnpack
	PhaseReady
)

type Progress struct {
	Phase    Phase
	Percent  int
	Version  string
	Err      string
	UpToDate bool
}

type Fetcher interface {
	Check(ctx context.Context) (version string, pkgURL, sha string, ok bool, err error)
}

type GitHubAppcastFetcher struct {
	URL string
}

func (g GitHubAppcastFetcher) Check(ctx context.Context) (string, string, string, bool, error) {
	end := diag.SpanSlow("update", "appcast", 3*time.Second)
	defer end()
	url := g.URL
	if url == "" {
		url = "https://github.com/ixxxxoooo/minicode/releases/latest/download/appcast.json"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", "", false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", "", false, fmt.Errorf("appcast HTTP %d", resp.StatusCode)
	}
	var ac struct {
		Version string `json:"version"`
		URL     string `json:"url"`
		Sha256  string `json:"sha256"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ac); err != nil {
		return "", "", "", false, err
	}
	if ac.Version == "" || ac.URL == "" {
		return "", "", "", false, nil
	}
	return ac.Version, ac.URL, ac.Sha256, true, nil
}

type DemoFetcher struct{ Version string }

func (d DemoFetcher) Check(ctx context.Context) (string, string, string, bool, error) {
	v := d.Version
	if v == "" {
		v = "0.2.2"
	}
	return v, "", "", true, nil
}

// Runner 执行下载 → 校验 → 解包 → 可替换。
type Runner struct {
	Fetch       Fetcher
	CurrentVer  string
	InstallFunc func(staging, pkgPath string) error
	OnProgress  func(Progress)
	workDir     string
}

func (r *Runner) WorkDir() string {
	if r.workDir != "" {
		return r.workDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".minicode", "update")
}

func (r *Runner) Start(ctx context.Context) {
	r.emit(Progress{Phase: PhaseDownload, Percent: 0, Version: r.CurrentVer})
	go r.run(ctx)
}

func (r *Runner) run(ctx context.Context) {
	ver, url, sha, ok, err := r.Fetch.Check(ctx)
	if err != nil {
		r.emit(Progress{Err: err.Error()})
		return
	}
	if !ok || ver == r.CurrentVer {
		r.emit(Progress{Version: r.CurrentVer, UpToDate: true})
		return
	}
	r.emit(Progress{Phase: PhaseDownload, Percent: 5, Version: ver})
	path, err := r.download(ctx, url, func(p int) { r.emit(Progress{Phase: PhaseDownload, Percent: p, Version: ver}) })
	if err != nil {
		r.emit(Progress{Err: err.Error()})
		return
	}
	r.emit(Progress{Phase: PhaseVerify, Percent: 100, Version: ver})
	if sha != "" {
		if err := verifySHA(path, sha); err != nil {
			r.emit(Progress{Err: err.Error()})
			return
		}
	}
	r.emit(Progress{Phase: PhaseUnpack, Percent: 100, Version: ver})
	stage := filepath.Join(r.WorkDir(), "staging")
	if path != "" {
		if err := UnpackPackage(path, stage); err != nil {
			r.emit(Progress{Err: "解包失败: " + err.Error()})
			return
		}
	} else if err := os.MkdirAll(filepath.Join(stage, "MiniCode.app"), 0o755); err != nil {
		r.emit(Progress{Err: err.Error()})
		return
	}
	_ = os.WriteFile(filepath.Join(stage, "Meta.json"), []byte(`{"version":"`+ver+`"}`), 0o644)
	if r.InstallFunc != nil {
		if err := r.InstallFunc(stage, path); err != nil {
			r.emit(Progress{Err: err.Error()})
			return
		}
	}
	r.emit(Progress{Phase: PhaseReady, Percent: 100, Version: ver})
}

func (r *Runner) emit(p Progress) {
	if r.OnProgress != nil {
		r.OnProgress(p)
	}
}

func (r *Runner) download(ctx context.Context, url string, pct func(int)) (string, error) {
	if url == "" {
		// 演示:无 URL 时模拟进度
		for i := 10; i <= 100; i += 15 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(120 * time.Millisecond):
				pct(i)
			}
		}
		return "", nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	path := filepath.Join(r.WorkDir(), "download.pkg")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var total int64 = resp.ContentLength
	var n int64
	buf := make([]byte, 64*1024)
	for {
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			if _, ew := f.Write(buf[:nr]); ew != nil {
				return "", ew
			}
			n += int64(nr)
			if total > 0 {
				pct(int(n * 100 / total))
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			return "", er
		}
	}
	pct(100)
	return path, nil
}

func verifySHA(path, want string) error {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("校验和不匹配")
	}
	return nil
}
