package workspace

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// Mutable 是可以新建 / 改名 / 删除文件的文件系统(文件树的「新建… / 重命名… / 移到废纸篓…」用)。
// 路径与 FS 一样是相对项目根的斜杠路径。
type Mutable interface {
	// Create 新建文件或文件夹;缺的中间文件夹一并建;已存在返回 ErrExist。
	Create(p string, dir bool) error
	// Rename 改名 / 移动;目标已存在返回 ErrExist;源不存在返回 os.ErrNotExist。
	Rename(from, to string) error
	// Remove 删除文件或文件夹(磁盘上是移到废纸篓)。
	Remove(p string) error
	// Count 返回 p 里的文件数(文件 = 1),确认框写「连同里面的 N 个文件」用。
	Count(p string) int
}

// ErrExist 表示目标已存在。
var ErrExist = os.ErrExist

// ErrName 表示名字不合法。
var ErrName = errors.New("名称不合法")

// ValidName 检查「新建 / 重命名」输入的一段名字(不含路径分隔符):不能空、不能是 . / ..、不能含 / 与控制字符。
func ValidName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w:不能为空", ErrName)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("%w:不能是 %s", ErrName, name)
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("%w:不能含 /", ErrName)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%w:含有不可见字符", ErrName)
		}
	}
	return nil
}

// ValidPath 检查新建时输入的相对路径(可含 / 以连带建文件夹):每一段都合法,不能以 / 开头。
func ValidPath(p string) error {
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("%w:不能以 / 开头", ErrName)
	}
	for _, seg := range strings.Split(strings.TrimSuffix(p, "/"), "/") {
		if err := ValidName(seg); err != nil {
			return err
		}
	}
	return nil
}

func under(p, dir string) bool { return p == dir || strings.HasPrefix(p, dir+"/") }

// ───────── 内存(演示) ─────────

func (m *Mem) Create(p string, dir bool) error {
	p = strings.Trim(p, "/")
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.existsLocked(p) {
		return ErrExist
	}
	// 已有同名文件挡住了某一级目录
	for q := path.Dir(p); q != "." && q != "/"; q = path.Dir(q) {
		if _, isFile := m.files[q]; isFile {
			return ErrExist
		}
	}
	if dir {
		m.dirs[p] = true
	} else {
		m.files[p] = ""
	}
	return nil
}

func (m *Mem) existsLocked(p string) bool {
	if _, ok := m.files[p]; ok {
		return true
	}
	if m.dirs[p] {
		return true
	}
	for f := range m.files {
		if strings.HasPrefix(f, p+"/") {
			return true
		}
	}
	return false
}

func (m *Mem) Rename(from, to string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.existsLocked(from) {
		return os.ErrNotExist
	}
	if m.existsLocked(to) {
		return ErrExist
	}
	if under(to, from) {
		return fmt.Errorf("%w:不能移到自己里面", ErrName)
	}
	if c, ok := m.files[from]; ok {
		delete(m.files, from)
		m.files[to] = c
		return nil
	}
	for f, c := range m.files {
		if strings.HasPrefix(f, from+"/") {
			delete(m.files, f)
			m.files[to+strings.TrimPrefix(f, from)] = c
		}
	}
	for d := range m.dirs {
		if under(d, from) {
			delete(m.dirs, d)
			m.dirs[to+strings.TrimPrefix(d, from)] = true
		}
	}
	return nil
}

func (m *Mem) Remove(p string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.existsLocked(p) {
		return os.ErrNotExist
	}
	delete(m.files, p)
	for f := range m.files {
		if strings.HasPrefix(f, p+"/") {
			delete(m.files, f)
		}
	}
	for d := range m.dirs {
		if under(d, p) {
			delete(m.dirs, d)
		}
	}
	return nil
}

func (m *Mem) Count(p string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.files[p]; ok {
		return 1
	}
	n := 0
	for f := range m.files {
		if strings.HasPrefix(f, p+"/") {
			n++
		}
	}
	return n
}

// ───────── 磁盘 ─────────

// TrashFunc 把磁盘上的路径移到废纸篓(测试里可替换);默认经 Finder,以便在访达里「放回原处」。
var TrashFunc = func(abs string) error {
	script := `tell application "Finder" to delete POSIX file ` + appleQuote(abs)
	if out, err := exec.Command("/usr/bin/osascript", "-e", script).CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return fmt.Errorf("无法移到废纸篓:%v %s", err, msg)
	}
	return nil
}

// appleQuote 把路径写成 AppleScript 字符串字面量(转义反斜杠与双引号)。
func appleQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// safe 路径必须落在项目根里(防 .. 逃出去),返回绝对路径。
func (o *OS) safe(p string) (string, error) {
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", fmt.Errorf("%w:不在项目里", ErrName)
		}
	}
	abs := o.Abs(path.Clean("/" + p)[1:])
	rel, err := filepath.Rel(o.root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w:不在项目里", ErrName)
	}
	return abs, nil
}

func (o *OS) Create(p string, dir bool) error {
	abs, err := o.safe(p)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(abs); err == nil {
		return ErrExist
	}
	if dir {
		return os.MkdirAll(abs, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

func (o *OS) Rename(from, to string) error {
	a, err := o.safe(from)
	if err != nil {
		return err
	}
	b, err := o.safe(to)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(a); err != nil {
		return err
	}
	if _, err := os.Lstat(b); err == nil && !strings.EqualFold(a, b) { // 仅大小写不同的改名(大小写不敏感的磁盘上)放行
		return ErrExist
	}
	if under(b, a) {
		return fmt.Errorf("%w:不能移到自己里面", ErrName)
	}
	return os.Rename(a, b)
}

func (o *OS) Remove(p string) error {
	abs, err := o.safe(p)
	if err != nil {
		return err
	}
	if abs == o.root {
		return fmt.Errorf("%w:不能删除项目根", ErrName)
	}
	if _, err := os.Lstat(abs); err != nil {
		return err
	}
	return TrashFunc(abs)
}

func (o *OS) Count(p string) int {
	abs, err := o.safe(p)
	if err != nil {
		return 0
	}
	st, err := os.Lstat(abs)
	if err != nil {
		return 0
	}
	if !st.IsDir() {
		return 1
	}
	n := 0
	_ = filepath.WalkDir(abs, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}

// ───────── 多根 ─────────

func (m *Multi) mut(p string) (Mutable, string, error) {
	i, rest, ok := m.Split(p)
	if !ok || rest == "" {
		return nil, "", fmt.Errorf("%w:不能改动工作区的顶层文件夹", ErrName)
	}
	mu, ok := m.roots[i].FS.(Mutable)
	if !ok {
		return nil, "", errors.New("这个文件夹是只读的")
	}
	return mu, rest, nil
}

func (m *Multi) Create(p string, dir bool) error {
	mu, rest, err := m.mut(p)
	if err != nil {
		return err
	}
	return mu.Create(rest, dir)
}

func (m *Multi) Rename(from, to string) error {
	i, a, ok1 := m.Split(from)
	j, b, ok2 := m.Split(to)
	if !ok1 || !ok2 || a == "" || b == "" {
		return fmt.Errorf("%w:不能改动工作区的顶层文件夹", ErrName)
	}
	if i != j {
		return fmt.Errorf("%w:不能移到别的文件夹里", ErrName)
	}
	mu, ok := m.roots[i].FS.(Mutable)
	if !ok {
		return errors.New("这个文件夹是只读的")
	}
	return mu.Rename(a, b)
}

func (m *Multi) Remove(p string) error {
	mu, rest, err := m.mut(p)
	if err != nil {
		return err
	}
	return mu.Remove(rest)
}

func (m *Multi) Count(p string) int {
	i, rest, ok := m.Split(p)
	if !ok || rest == "" {
		return 0
	}
	if mu, ok := m.roots[i].FS.(Mutable); ok {
		return mu.Count(rest)
	}
	return 0
}
