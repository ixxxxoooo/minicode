package update

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BundlePath 从当前可执行文件推断 .app 路径;非包内运行时返回空。
func BundlePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return ""
	}
	dir := exe
	for i := 0; i < 6; i++ {
		if strings.HasSuffix(dir, ".app") {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return ""
}

// UnpackPackage 把下载的 zip 或已挂载的目录解到 staging(期望 staging 内有一个 .app)。
func UnpackPackage(pkgPath, staging string) error {
	_ = os.RemoveAll(staging)
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	low := strings.ToLower(pkgPath)
	switch {
	case strings.HasSuffix(low, ".zip"):
		return unzipTo(pkgPath, staging)
	case strings.HasSuffix(low, ".dmg"):
		return unpackDMG(pkgPath, staging)
	default:
		// 已是目录或未知:尝试复制 *.app
		return copyAppInDir(pkgPath, staging)
	}
}

func unzipTo(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		fp := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(filepath.Clean(fp), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("非法路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fp, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.OpenFile(fp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(w, rc)
		w.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func unpackDMG(dmgPath, dest string) error {
	mount := filepath.Join(dest, "mnt")
	if err := os.MkdirAll(mount, 0o755); err != nil {
		return err
	}
	if out, err := exec.Command("hdiutil", "attach", "-nobrowse", "-mountpoint", mount, dmgPath).CombinedOutput(); err != nil {
		return fmt.Errorf("挂载 dmg 失败: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	defer exec.Command("hdiutil", "detach", mount, "-quiet").Run()
	return copyAppInDir(mount, dest)
}

func copyAppInDir(srcDir, dest string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	var appName string
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), ".app") {
			appName = e.Name()
			break
		}
	}
	if appName == "" {
		return fmt.Errorf("未找到 .app")
	}
	src := filepath.Join(srcDir, appName)
	dst := filepath.Join(dest, appName)
	out, err := exec.Command("ditto", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ditto 失败: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// WriteReplaceScript 生成退出后替换 .app 并重启的 shell 脚本。
func WriteReplaceScript(staging, targetBundle, scriptPath string) error {
	entries, err := os.ReadDir(staging)
	if err != nil {
		return err
	}
	var appName string
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), ".app") {
			appName = e.Name()
			break
		}
	}
	if appName == "" {
		return fmt.Errorf("暂存目录里没有 .app")
	}
	src := filepath.Join(staging, appName)
	if targetBundle == "" {
		targetBundle = filepath.Join("/Applications", appName)
	}
	// 先等当前进程退出(最多约 10 秒)再覆盖:正在运行的包被替换会导致新旧文件混用;
	// 覆盖后清掉下载带来的隔离属性,避免「已损坏 / 无法验证开发者」的拦截,最后重新打开。
	body := fmt.Sprintf(`#!/bin/sh
i=0
while kill -0 %d 2>/dev/null && [ $i -lt 100 ]; do sleep 0.1; i=$((i+1)); done
ditto %q %q || exit 1
xattr -dr com.apple.quarantine %q 2>/dev/null
open %q
`, os.Getpid(), src, targetBundle, targetBundle, targetBundle)
	return os.WriteFile(scriptPath, []byte(body), 0o755)
}
