// Package projects 保存用户手动添加的项目与工作区(不依赖界面)。
//
//	项目   = 一个文件夹(Claude 会话的工作目录)
//	工作区 = 一组命名的文件夹:第一个是主文件夹(会话 cwd),其余是附加文件夹(--add-dir)
//
// 数据放在 ~/.minicode/workspaces.json,每次修改原子落盘;不往项目里写任何文件。
package projects

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var log = diag.L("projects")

// MaxFolders 是一个工作区最多的文件夹数。
const MaxFolders = 12

// Folder 是工作区里的一个文件夹。Name 为空时取目录名。
type Folder struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

// Label 是显示名。
func (f Folder) Label() string {
	if f.Name != "" {
		return f.Name
	}
	return filepath.Base(f.Path)
}

// Workspace 是一个工作区。Folders[0] 是主文件夹。
type Workspace struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Folders []Folder `json:"folders"`
	// LoadMD 为真时让 Claude 同时读取附加文件夹里的 CLAUDE.md 与规则。
	LoadMD bool `json:"loadMD,omitempty"`
}

// Primary 是主文件夹路径。
func (w Workspace) Primary() string {
	if len(w.Folders) == 0 {
		return ""
	}
	return w.Folders[0].Path
}

// Extra 是附加文件夹路径(给 --add-dir)。
func (w Workspace) Extra() []string {
	var out []string
	for i, f := range w.Folders {
		if i > 0 {
			out = append(out, f.Path)
		}
	}
	return out
}

// Store 是项目与工作区的持久化存储。path 为空时只在内存里(演示 / 测试)。
type Store struct {
	mu         sync.Mutex
	path       string
	projects   []string
	workspaces []Workspace
	names      map[string]string // 项目显示名(路径 → 名称);没有 = 用文件夹名
	// Exists 判断路径是不是存在的目录;测试 / 演示可替换。
	Exists func(string) bool
}

type disk struct {
	Projects   []string          `json:"projects"`
	Workspaces []Workspace       `json:"workspaces"`
	Names      map[string]string `json:"names,omitempty"`
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// Open 读取存储文件;不存在 / 损坏时从空开始(损坏的文件改名留底,不丢用户数据)。
func Open(path string) *Store {
	s := &Store{path: path, Exists: isDir}
	if path == "" {
		return s
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	var d disk
	if err := json.Unmarshal(b, &d); err != nil {
		log.Warn("项目存储损坏,已改名留底并从空开始", "err", err)
		_ = os.Rename(path, path+".bad-"+time.Now().Format("20060102-150405"))
		return s
	}
	s.projects, s.workspaces, s.names = d.Projects, d.Workspaces, d.Names
	return s
}

// NewMem 是只在内存里的存储(演示用)。
func NewMem() *Store { return &Store{Exists: func(string) bool { return true }} }

func (s *Store) save() error {
	if s.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(disk{Projects: s.projects, Workspaces: s.workspaces, Names: s.names}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// 校验错误:界面按种类给出「下一步」。
var (
	ErrNotFound = errors.New("找不到这个文件夹")
	ErrNotDir   = errors.New("这不是文件夹")
	ErrDup      = errors.New("已经添加过了")
	ErrEmpty    = errors.New("请输入路径")
	ErrTooMany  = fmt.Errorf("一个工作区最多 %d 个文件夹", MaxFolders)
)

// Clean 展开 ~、转绝对路径并去掉末尾斜杠。
func Clean(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(h, strings.TrimPrefix(p, "~"))
		}
	}
	if !filepath.IsAbs(p) {
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
	}
	return filepath.Clean(p)
}

// Check 校验一个路径能不能作为项目 / 工作区文件夹。
func (s *Store) Check(p string) (string, error) {
	c := Clean(p)
	if c == "" {
		return "", ErrEmpty
	}
	if !s.Exists(c) {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, ErrNotDir
		}
		return c, ErrNotFound
	}
	return c, nil
}

// Projects 是手动添加的项目(路径,按添加顺序)。
func (s *Store) Projects() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.projects...)
}

// HasProject 路径是否已是手动添加的项目。
func (s *Store) HasProject(p string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := Clean(p)
	for _, x := range s.projects {
		if x == c {
			return true
		}
	}
	return false
}

// AddProject 添加项目,返回规范化后的路径。
func (s *Store) AddProject(p string) (string, error) {
	c, err := s.Check(p)
	if err != nil {
		return c, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.projects {
		if x == c {
			return c, ErrDup
		}
	}
	s.projects = append(s.projects, c)
	log.Info("添加项目", "path", c)
	return c, s.save()
}

// RemoveProject 从列表移除项目(不碰磁盘)。
func (s *Store) RemoveProject(p string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := Clean(p)
	out := s.projects[:0:0]
	for _, x := range s.projects {
		if x != c {
			out = append(out, x)
		}
	}
	s.projects = out
	delete(s.names, c)
	log.Info("移除项目", "path", c)
	return s.save()
}

// ProjectName 返回项目的自定义显示名;没有设置返回 ""。
func (s *Store) ProjectName(p string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.names[Clean(p)]
}

// SetProjectName 设置项目的显示名(只存在本机整理信息里,不改文件夹);name 为空 = 恢复文件夹名。
func (s *Store) SetProjectName(p, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, name := Clean(p), strings.TrimSpace(name)
	if name == "" || name == filepath.Base(c) {
		delete(s.names, c)
	} else {
		if s.names == nil {
			s.names = map[string]string{}
		}
		s.names[c] = name
	}
	log.Info("项目显示名", "path", c, "name", name)
	return s.save()
}

// Workspaces 是全部工作区(拷贝)。
func (s *Store) Workspaces() []Workspace {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Workspace, len(s.workspaces))
	for i, w := range s.workspaces {
		w.Folders = append([]Folder(nil), w.Folders...)
		out[i] = w
	}
	return out
}

// Workspace 按 id 取工作区。
func (s *Store) Workspace(id string) (Workspace, bool) {
	for _, w := range s.Workspaces() {
		if w.ID == id {
			return w, true
		}
	}
	return Workspace{}, false
}

// normalize 校验并规整一个工作区(去重、规范路径、限制数量),返回错误时不修改任何东西。
func (s *Store) normalize(w Workspace) (Workspace, error) {
	w.Name = strings.TrimSpace(w.Name)
	if len(w.Folders) > MaxFolders {
		return w, ErrTooMany
	}
	seen := map[string]bool{}
	var fs []Folder
	for _, f := range w.Folders {
		c, err := s.Check(f.Path)
		if err != nil {
			return w, fmt.Errorf("%s:%w", f.Path, err)
		}
		if seen[c] {
			continue
		}
		seen[c] = true
		fs = append(fs, Folder{Path: c, Name: f.Name})
	}
	if len(fs) == 0 {
		return w, errors.New("工作区至少要有一个文件夹")
	}
	w.Folders = fs
	if w.Name == "" {
		w.Name = fs[0].Label() + " 工作区"
	}
	return w, nil
}

// SaveWorkspace 新建(ID 为空)或更新工作区,返回保存后的值。
func (s *Store) SaveWorkspace(w Workspace) (Workspace, error) {
	w, err := s.normalize(w)
	if err != nil {
		return w, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if w.ID == "" {
		w.ID = "ws-" + strconv.FormatInt(time.Now().UnixNano(), 36)
		s.workspaces = append(s.workspaces, w)
		log.Info("新建工作区", "id", w.ID, "name", w.Name, "folders", len(w.Folders))
		return w, s.save()
	}
	for i := range s.workspaces {
		if s.workspaces[i].ID == w.ID {
			s.workspaces[i] = w
			log.Info("更新工作区", "id", w.ID, "folders", len(w.Folders))
			return w, s.save()
		}
	}
	// 指定了 ID 但还不存在(演示数据 / 从别处导入):按这个 ID 新建
	s.workspaces = append(s.workspaces, w)
	return w, s.save()
}

// RemoveWorkspace 删除工作区(只删列表项)。
func (s *Store) RemoveWorkspace(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.workspaces[:0:0]
	for _, w := range s.workspaces {
		if w.ID != id {
			out = append(out, w)
		}
	}
	s.workspaces = out
	log.Info("移除工作区", "id", id)
	return s.save()
}

// MoveWorkspace 把工作区 id 放到工作区 before 之前(before 为空 = 放到最后);顺序写盘。
func (s *Store) MoveWorkspace(id, before string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	from := -1
	for i, w := range s.workspaces {
		if w.ID == id {
			from = i
		}
	}
	if from < 0 || id == before {
		return nil
	}
	w := s.workspaces[from]
	rest := append(append([]Workspace(nil), s.workspaces[:from]...), s.workspaces[from+1:]...)
	at := len(rest)
	for i, x := range rest {
		if x.ID == before {
			at = i
		}
	}
	s.workspaces = append(rest[:at], append([]Workspace{w}, rest[at:]...)...)
	log.Info("工作区换序", "id", id, "before", before)
	return s.save()
}

// MoveBefore 把 list 里第 from 项放到第 before 项之前(before == len = 放到最后),返回新顺序的拷贝与它落下的位置。
func MoveBefore[T any](list []T, from, before int) ([]T, int) {
	out := append([]T(nil), list...)
	if from < 0 || from >= len(out) || before < 0 || before > len(out) || before == from || before == from+1 {
		return out, from
	}
	x := out[from]
	out = append(out[:from], out[from+1:]...)
	if before > from {
		before--
	}
	out = append(out[:before], append([]T{x}, out[before:]...)...)
	return out, before
}

// WorkspacesOf 引用了某个文件夹的工作区(移除项目前提示用)。
func (s *Store) WorkspacesOf(path string) []Workspace {
	c := Clean(path)
	var out []Workspace
	for _, w := range s.Workspaces() {
		for _, f := range w.Folders {
			if f.Path == c {
				out = append(out, w)
				break
			}
		}
	}
	return out
}

// LoadMDEnv 是让 Claude 读取附加文件夹里 CLAUDE.md 的环境变量名。
const LoadMDEnv = "CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD"

// ByPrimary 找主文件夹是 dir 的第一个工作区(启动会话时据此追加 --add-dir)。
func (s *Store) ByPrimary(dir string) (Workspace, bool) {
	c := Clean(dir)
	for _, w := range s.Workspaces() {
		if w.Primary() == c {
			return w, true
		}
	}
	return Workspace{}, false
}

// Has 路径是不是这个工作区的成员文件夹。
func (w Workspace) Has(path string) bool {
	c := Clean(path)
	for _, f := range w.Folders {
		if f.Path == c {
			return true
		}
	}
	return false
}

// Persistent 是否落盘(演示 / 测试的内存存储返回 false)。
func (s *Store) Persistent() bool { return s.path != "" }
