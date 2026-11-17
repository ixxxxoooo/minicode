package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	ErrStage string // 出错的阶段:check / download / verify / unpack / install
	UpToDate bool
	// Found 表示发现了新版本但按偏好没有自动下载(Runner.CheckOnly),等用户点「下载并安装」。
	Found bool
}

type Fetcher interface {
	Check(ctx context.Context) (version string, pkgURL, sha string, ok bool, err error)
}

type GitHubAppcastFetcher struct {
	URL     string
	Channel string // 空 = 稳定;「beta」= 测试通道(追加查询参数,服务器忽略亦无害)
}

func (g GitHubAppcastFetcher) Check(ctx context.Context) (string, string, string, bool, error) {
	end := diag.SpanSlow("update", "appcast", 3*time.Second)
	defer end()
	// 只限制「查询发布信息」这一步;下载是否超时由调用方的 ctx 决定
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	url := g.URL
	if url == "" {
		url = "https://github.com/ixxxxoooo/minicode/releases/latest/download/appcast.json"
	}
	if g.Channel != "" {
		if strings.Contains(url, "?") {
			url += "&channel=" + g.Channel
		} else {
			url += "?channel=" + g.Channel
		}
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
	ver, pkgURL, sha, err := parseAppcast(resp.Body, g.Channel)
	if err != nil {
		return "", "", "", false, err
	}
	return ver, pkgURL, sha, ver != "" && pkgURL != "", nil
}

type DemoFetcher struct{ Version string }

func (d DemoFetcher) Check(ctx context.Context) (string, string, string, bool, error) {
	v := d.Version
	if v == "" {
		v = "0.9.0"
	}
	return v, "", "", true, nil
}

// Runner 执行下载 → 校验 → 解包 → 可替换。
type Runner struct {
	Fetch       Fetcher
	CurrentVer  string
	InstallFunc func(staging, pkgPath string) error
	OnProgress  func(Progress)
	// CheckOnly 为真时,发现新版本只上报 Progress.Found 并停下(偏好「自动下载更新」关闭);
	// 之后由 Download 接着把已发现的版本下载下来。
	CheckOnly bool
	workDir   string

	mu    sync.Mutex
	found foundPkg
}

// foundPkg 是 CheckOnly 模式下记住的、已发现但未下载的版本。
type foundPkg struct {
	ver, url, sha string
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

// Run 同步执行整个流程(调用方已在后台线程里):返回时 ctx 才可以取消。
// 用 Start 的调用方若在返回后立刻 cancel(),整个检查会被自己取消。
func (r *Runner) Run(ctx context.Context) {
	// 开头只表示「在检查」:不能发下载进度,否则界面会把它当成「发现新版本」(版本号还是当前版本)
	r.emit(Progress{Phase: PhaseNone, Version: r.CurrentVer})
	r.run(ctx)
}

func (r *Runner) run(ctx context.Context) {
	ver, url, sha, ok, err := r.Fetch.Check(ctx)
	if err != nil {
		r.emit(Progress{Err: err.Error(), ErrStage: "check"})
		return
	}
	if !ok || !IsNewer(ver, r.CurrentVer) {
		r.emit(Progress{Version: r.CurrentVer, UpToDate: true})
		return
	}
	if r.CheckOnly {
		r.mu.Lock()
		r.found = foundPkg{ver: ver, url: url, sha: sha}
		r.mu.Unlock()
		r.emit(Progress{Version: ver, Found: true})
		return
	}
	r.fetchAndStage(ctx, ver, url, sha)
}

// Download 下载并准备「已发现但按偏好没有自动下载」的版本(同步执行,调用方已在后台线程)。
// 没有已发现的版本(没检查过 / 已经在下载)时报 check 阶段的错误。
func (r *Runner) Download(ctx context.Context) {
	r.mu.Lock()
	f := r.found
	r.found = foundPkg{}
	r.mu.Unlock()
	if f.ver == "" {
		r.emit(Progress{Err: "没有可下载的新版本,请先检查更新", ErrStage: "check"})
		return
	}
	r.fetchAndStage(ctx, f.ver, f.url, f.sha)
}

// FoundVersion 返回已发现但尚未下载的版本号(没有则为空)。
func (r *Runner) FoundVersion() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.found.ver
}

// fetchAndStage 下载 → 校验 → 解包 → 交给 InstallFunc,各阶段都上报进度。
func (r *Runner) fetchAndStage(ctx context.Context, ver, url, sha string) {
	r.emit(Progress{Phase: PhaseDownload, Percent: 5, Version: ver})
	path, err := r.download(ctx, url, func(p int) { r.emit(Progress{Phase: PhaseDownload, Percent: p, Version: ver}) })
	if err != nil {
		r.emit(Progress{Err: err.Error(), ErrStage: "download"})
		return
	}
	r.emit(Progress{Phase: PhaseVerify, Percent: 100, Version: ver})
	if sha != "" {
		if err := verifySHA(path, sha); err != nil {
			r.emit(Progress{Err: err.Error(), ErrStage: "verify"})
			return
		}
	}
	r.emit(Progress{Phase: PhaseUnpack, Percent: 100, Version: ver})
	stage := filepath.Join(r.WorkDir(), "staging")
	if path != "" {
		if err := UnpackPackage(path, stage); err != nil {
			r.emit(Progress{Err: "解包失败: " + err.Error(), ErrStage: "unpack"})
			return
		}
	} else if err := os.MkdirAll(filepath.Join(stage, "MiniCode.app"), 0o755); err != nil {
		r.emit(Progress{Err: err.Error(), ErrStage: "unpack"})
		return
	}
	_ = os.WriteFile(filepath.Join(stage, "Meta.json"), []byte(`{"version":"`+ver+`"}`), 0o644)
	if r.InstallFunc != nil {
		if err := r.InstallFunc(stage, path); err != nil {
			r.emit(Progress{Err: err.Error(), ErrStage: "install"})
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
	if resp.StatusCode != http.StatusOK { // 404 / 5xx 的错误页不能当安装包存下来
		return "", fmt.Errorf("下载失败 HTTP %d", resp.StatusCode)
	}
	if err := os.MkdirAll(r.WorkDir(), 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(r.WorkDir(), pkgFileName(url))
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
