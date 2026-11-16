package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WithExternal 在原有 FS 之上,允许用绝对磁盘路径读写项目外的文件( List 仍只走 base)。
type WithExternal struct {
	base FS
}

// NewWithExternal 包装 base;已包装则原样返回。
func NewWithExternal(base FS) FS {
	if base == nil {
		return nil
	}
	if _, ok := base.(*WithExternal); ok {
		return base
	}
	return &WithExternal{base: base}
}

// Base 返回被包装的原始文件系统。
func (w *WithExternal) Base() FS { return w.base }

// Unwrap 拆掉 WithExternal 包装,返回底层文件系统(没包装则原样返回)。
// 凡是要按具体类型判断能力(Multi / Mutable / OS)的地方,都要先 Unwrap,否则包装层会遮住底层能力。
func Unwrap(fs FS) FS {
	for {
		w, ok := fs.(*WithExternal)
		if !ok {
			return fs
		}
		fs = w.base
	}
}

// IsExternalPath 是否应按磁盘绝对路径打开(演示虚拟路径 /demo/... 不算)。
func IsExternalPath(p string) bool {
	if !strings.HasPrefix(p, "/") {
		return false
	}
	return !strings.HasPrefix(p, "/demo/")
}

func (w *WithExternal) Name() string { return w.base.Name() }

func (w *WithExternal) List(dir string) ([]Entry, error) { return w.base.List(dir) }

func (w *WithExternal) Read(p string) (string, error) {
	if IsExternalPath(p) {
		return readAbsFile(p)
	}
	return w.base.Read(p)
}

func (w *WithExternal) Write(p, content string) error {
	if IsExternalPath(p) {
		return writeAbsFile(p, content)
	}
	return w.base.Write(p, content)
}

func (w *WithExternal) Abs(p string) string {
	if IsExternalPath(p) {
		return filepath.Clean(p)
	}
	return w.base.Abs(p)
}

func readAbsFile(p string) (string, error) {
	abs := filepath.Clean(p)
	st, err := os.Stat(abs)
	if err != nil {
		return "", friendlyIOErr("读取", abs, err)
	}
	if st.IsDir() {
		return "", errors.New("这是文件夹,不能用编辑器打开")
	}
	if st.Size() > MaxRead {
		return "", ErrTooLarge
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", friendlyIOErr("读取", abs, err)
	}
	return string(b), nil
}

func writeAbsFile(p, content string) error {
	abs := filepath.Clean(p)
	if st, err := os.Stat(abs); err == nil && st.IsDir() {
		return errors.New("不能向文件夹写入")
	}
	mode := os.FileMode(0o644)
	if st, err := os.Stat(abs); err == nil {
		mode = st.Mode() & 0o777
	}
	tmp := abs + ".minicode-tmp"
	if err := os.WriteFile(tmp, []byte(content), mode); err != nil {
		return friendlyIOErr("写入", abs, err)
	}
	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		return friendlyIOErr("写入", abs, err)
	}
	return nil
}

func friendlyIOErr(verb, abs string, err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return err
	}
	var pe *os.PathError
	if errors.As(err, &pe) && errors.Is(pe.Err, os.ErrPermission) {
		return fmt.Errorf("%s %s:没有权限", verb, filepath.Base(abs))
	}
	return err
}
