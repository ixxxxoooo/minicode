package lsp

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 托管 Node:应用本身不内置 Node(否则安装包翻倍),依赖 Node 的语言服务器在本机找不到合适的 Node 时,
// 由用户点「下载安装」后才下载一份到 <数据目录>/runtime/node-<版本>/。
// 选哪个 Node:优先本机(PATH 与常见目录里版本够新的),没有才用托管的。

var rtLog = diag.L("lsp")

// ManagedNodeVersion 是托管的 Node 版本(固定一个 LTS,随应用版本更新)。
const ManagedNodeVersion = "v24.11.1"

// Progress 报告下载 / 安装进度:stage 是人读的阶段名;total ≤ 0 表示不知道总量(转圈)。
type Progress func(stage string, done, total int64)

func dataRoot() string { return diag.Home() }

func managedNodeHome() string {
	return filepath.Join(dataRoot(), "runtime", "node-"+ManagedNodeVersion)
}

// ManagedNodeDir 是托管 Node 的 bin 目录(里面有 node / npm)。
func ManagedNodeDir() string { return filepath.Join(managedNodeHome(), "bin") }

// ───────── 版本比较 ─────────

type nodeVer [3]int

// parseNodeVer 解析 "v24.11.1" / "24.11" / "18";缺的部分当 0。
func parseNodeVer(s string) (nodeVer, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if s == "" {
		return nodeVer{}, false
	}
	var v nodeVer
	for i, p := range strings.SplitN(s, ".", 3) {
		end := 0
		for end < len(p) && p[end] >= '0' && p[end] <= '9' {
			end++
		}
		if end == 0 {
			return nodeVer{}, false
		}
		n, err := strconv.Atoi(p[:end])
		if err != nil {
			return nodeVer{}, false
		}
		v[i] = n
	}
	return v, true
}

func (a nodeVer) cmp(b nodeVer) int {
	for i := range a {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// ───────── 找 Node ─────────

// NodeInfo 是选中的 Node:Dir 是含 node(和 npm)的目录。
type NodeInfo struct {
	Dir     string
	Version string // 如 "v24.11.1"
	Managed bool   // true = 托管的(下载的),false = 本机的
}

// 路径里带版本号的目录(nvm 的 .../node/v24.11.1/bin、托管的 .../node-v24.11.1/bin)不必起子进程问版本。
var nodePathVerRe = regexp.MustCompile(`/node[-/](v\d+\.\d+\.\d+)/`)

var nodeVerCache sync.Map // node 路径 → 版本串(只缓存成功的)

func isExec(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0
}

// nodeVersionAt 取某个目录里 node 的版本。
func nodeVersionAt(dir string) (string, bool) {
	if m := nodePathVerRe.FindStringSubmatch(dir + "/"); m != nil {
		return m[1], true
	}
	p := filepath.Join(dir, "node")
	if v, ok := nodeVerCache.Load(p); ok {
		return v.(string), true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, p, "--version").Output()
	s := strings.TrimSpace(string(out))
	if err != nil || !strings.HasPrefix(s, "v") {
		return "", false
	}
	nodeVerCache.Store(p, s)
	return s, true
}

// FindNode 在本机找版本 ≥ min 的 Node(min 为空 = 不限);needNPM 要求同目录有 npm(安装语言包要用)。
// 顺序:PATH → 常见目录 → nvm 各版本(新的在前),所以用户 shell 里在用的那个优先。
func FindNode(min string, needNPM bool) (NodeInfo, bool) {
	want, _ := parseNodeVer(min)
	for _, d := range searchDirs() {
		if !isExec(filepath.Join(d, "node")) || (needNPM && !isExec(filepath.Join(d, "npm"))) {
			continue
		}
		vs, ok := nodeVersionAt(d)
		if !ok {
			continue
		}
		if v, ok := parseNodeVer(vs); !ok || v.cmp(want) < 0 {
			continue
		}
		return NodeInfo{Dir: d, Version: vs}, true
	}
	return NodeInfo{}, false
}

// managedNode 是已下载好的托管 Node(没下载过返回 false)。
func managedNode(min string, needNPM bool) (NodeInfo, bool) {
	dir := ManagedNodeDir()
	if !isExec(filepath.Join(dir, "node")) || (needNPM && !isExec(filepath.Join(dir, "npm"))) {
		return NodeInfo{}, false
	}
	have, _ := parseNodeVer(ManagedNodeVersion)
	if want, _ := parseNodeVer(min); have.cmp(want) < 0 {
		return NodeInfo{}, false
	}
	return NodeInfo{Dir: dir, Version: ManagedNodeVersion, Managed: true}, true
}

// ResolveNode 选 Node:优先本机,其次已下载的托管 Node;都没有返回 false(要先 InstallManagedNode)。
func ResolveNode(min string, needNPM bool) (NodeInfo, bool) {
	if n, ok := FindNode(min, needNPM); ok {
		return n, true
	}
	return managedNode(min, needNPM)
}

// nvmBins 列出 ~/.nvm/versions/node/*/bin,新版本在前(字典序 v9 会排在 v24 后面,不能直接用 Glob 的顺序)。
func nvmBins(home string) []string {
	m, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin"))
	ver := func(p string) nodeVer {
		mm := nodePathVerRe.FindStringSubmatch(p + "/")
		if mm == nil {
			return nodeVer{}
		}
		v, _ := parseNodeVer(mm[1])
		return v
	}
	sort.SliceStable(m, func(i, j int) bool { return ver(m[i]).cmp(ver(m[j])) > 0 })
	return m
}

// ───────── 下载托管 Node ─────────

var httpClient = &http.Client{Transport: &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	TLSHandshakeTimeout:   15 * time.Second,
	ResponseHeaderTimeout: 30 * time.Second,
	IdleConnTimeout:       30 * time.Second,
}}

const stallTimeout = 60 * time.Second // 超过这么久一个字节也没收到就当网络停了

var errStalled = errors.New("下载停滞(60 秒没有收到数据)")

// httpGet 发 GET 并交给 fn 处理响应体;stallTimeout 内没有数据到达就中断。
func httpGet(ctx context.Context, url string, fn func(body io.Reader, total int64) error) error {
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	var stalled atomic.Bool
	wd := time.AfterFunc(stallTimeout, func() { stalled.Store(true); cancel() })
	defer wd.Stop()
	req, err := http.NewRequestWithContext(ctx2, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "MiniCode")
	resp, err := httpClient.Do(req)
	if err != nil {
		if stalled.Load() {
			return errStalled
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d(%s)", resp.StatusCode, url)
	}
	err = fn(&watchReader{r: resp.Body, wd: wd}, resp.ContentLength)
	if err != nil && stalled.Load() {
		return errStalled
	}
	return err
}

// watchReader 每次读到数据就把看门狗往后推。
type watchReader struct {
	r  io.Reader
	wd *time.Timer
}

func (w *watchReader) Read(p []byte) (int, error) {
	n, err := w.r.Read(p)
	if n > 0 {
		w.wd.Reset(stallTimeout)
	}
	return n, err
}

func httpGetString(ctx context.Context, url string, limit int64) (string, error) {
	var out string
	err := httpGet(ctx, url, func(body io.Reader, _ int64) error {
		b, err := io.ReadAll(io.LimitReader(body, limit))
		out = string(b)
		return err
	})
	return out, err
}

// sumFor 从 SHASUMS256.txt(每行「sha256  文件名」)里取某个文件的校验和。
func sumFor(sums, name string) (string, bool) {
	for _, ln := range strings.Split(sums, "\n") {
		f := strings.Fields(ln)
		if len(f) == 2 && f[1] == name {
			return strings.ToLower(f[0]), true
		}
	}
	return "", false
}

// InstallManagedNode 下载托管 Node:取同源 SHASUMS256.txt → 下载 tar.gz 并边下边算 sha256 →
// 校验通过才解压到临时目录 → 试跑 node --version → 原子改名。任何一步失败都不留半成品。
func InstallManagedNode(ctx context.Context, m Mirror, prog Progress) error {
	defer diag.SpanSlow("lsp", "下载托管 Node", 60*time.Second, "mirror", string(m))()
	arch := map[string]string{"arm64": "arm64", "amd64": "x64"}[runtime.GOARCH]
	if arch == "" {
		return fmt.Errorf("不支持的 CPU 架构:%s", runtime.GOARCH)
	}
	name := fmt.Sprintf("node-%s-darwin-%s.tar.gz", ManagedNodeVersion, arch)
	base := m.NodeDist() + "/" + ManagedNodeVersion
	rtDir := filepath.Join(dataRoot(), "runtime")
	if err := os.MkdirAll(rtDir, 0o755); err != nil {
		return err
	}
	rtLog.Info("下载托管 Node", "version", ManagedNodeVersion, "mirror", string(m), "file", name)

	stage := "下载 Node " + ManagedNodeVersion
	prog(stage, 0, 0)
	sums, err := httpGetString(ctx, base+"/SHASUMS256.txt", 1<<20)
	if err != nil {
		return fmt.Errorf("取 Node 校验文件失败:%w", err)
	}
	want, ok := sumFor(sums, name)
	if !ok {
		return fmt.Errorf("校验文件里没有 %s", name)
	}

	tmp, err := os.CreateTemp(rtDir, ".dl-*.tgz")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	h := sha256.New()
	err = httpGet(ctx, base+"/"+name, func(body io.Reader, total int64) error {
		var done int64
		var last time.Time
		buf := make([]byte, 128<<10)
		for {
			n, rerr := body.Read(buf)
			if n > 0 {
				h.Write(buf[:n])
				if _, werr := tmp.Write(buf[:n]); werr != nil {
					return werr
				}
				done += int64(n)
				if time.Since(last) > 120*time.Millisecond {
					last = time.Now()
					prog(stage, done, total)
				}
			}
			if rerr == io.EOF {
				prog(stage, done, total)
				return nil
			}
			if rerr != nil {
				return rerr
			}
		}
	})
	if err != nil {
		return fmt.Errorf("下载 Node 失败:%w", err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return fmt.Errorf("Node 校验失败(sha256 不一致):期望 %s…,实际 %s…", want[:12], got[:12])
	}

	prog("解压 Node", 0, 0)
	ex, err := os.MkdirTemp(rtDir, ".extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(ex)
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}
	gz, err := gzip.NewReader(tmp)
	if err != nil {
		return fmt.Errorf("解压 Node 失败:%w", err)
	}
	defer gz.Close()
	// 只要 bin / lib(含自带 npm)和许可证;include / share 用不上
	skip := func(rel string) bool {
		return rel == "include" || strings.HasPrefix(rel, "include/") || rel == "share" || strings.HasPrefix(rel, "share/")
	}
	if err := extractTarGz(gz, ex, skip); err != nil {
		return fmt.Errorf("解压 Node 失败:%w", err)
	}

	ctx3, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx3, filepath.Join(ex, "bin", "node"), "--version").Output()
	if got := strings.TrimSpace(string(out)); err != nil || got != ManagedNodeVersion {
		return fmt.Errorf("下载的 Node 无法运行(期望 %s,得到 %q):%v", ManagedNodeVersion, got, err)
	}
	home := managedNodeHome()
	if err := os.RemoveAll(home); err != nil {
		return err
	}
	if err := os.Rename(ex, home); err != nil {
		return err
	}
	rtLog.Info("托管 Node 已就绪", "dir", home)
	return nil
}

// within 判断 p 是否在 root 之内(防压缩包里的 ../ 与越界符号链接)。
func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// extractTarGz 解开 tar 流到 dest,去掉最外层目录(node-v24.x-darwin-arm64/);skip 返回 true 的相对路径跳过。
// 只处理目录 / 普通文件 / 符号链接;任何越出 dest 的路径或链接都直接报错。
func extractTarGz(r io.Reader, dest string, skip func(rel string) bool) error {
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := path.Clean(h.Name)
		i := strings.IndexByte(name, '/')
		if i < 0 {
			continue // 最外层目录本身
		}
		rel := name[i+1:]
		if rel == "" || rel == "." || (skip != nil && skip(rel)) {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if !within(dest, target) {
			return fmt.Errorf("压缩包里有越界路径:%s", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, h.FileInfo().Mode().Perm()|0o200)
			if err != nil {
				return err
			}
			_, cerr := io.CopyN(f, tr, h.Size)
			if err := f.Close(); cerr == nil {
				cerr = err
			}
			if cerr != nil {
				return cerr
			}
		case tar.TypeSymlink:
			if filepath.IsAbs(h.Linkname) || !within(dest, filepath.Join(filepath.Dir(target), h.Linkname)) {
				return fmt.Errorf("压缩包里有越界的符号链接:%s → %s", h.Name, h.Linkname)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(h.Linkname, target); err != nil {
				return err
			}
		}
	}
}
