package projects

import (
	"os"
	"path/filepath"
	"testing"
)

func mk(t *testing.T, names ...string) []string {
	t.Helper()
	root := t.TempDir()
	var out []string
	for _, n := range names {
		p := filepath.Join(root, n)
		if err := os.Mkdir(p, 0o755); err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

func TestAddProjectValidation(t *testing.T) {
	dirs := mk(t, "a")
	s := Open(filepath.Join(t.TempDir(), "ws.json"))
	if _, err := s.AddProject(""); err != ErrEmpty {
		t.Fatalf("空路径:%v", err)
	}
	if _, err := s.AddProject(filepath.Join(dirs[0], "nope")); err != ErrNotFound {
		t.Fatalf("不存在:%v", err)
	}
	f := filepath.Join(dirs[0], "f.txt")
	_ = os.WriteFile(f, []byte("x"), 0o644)
	if _, err := s.AddProject(f); err != ErrNotDir {
		t.Fatalf("文件不是目录:%v", err)
	}
	if _, err := s.AddProject(dirs[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddProject(dirs[0] + "/"); err != ErrDup {
		t.Fatalf("重复(带斜杠)应识别:%v", err)
	}
	if !s.HasProject(dirs[0]) || len(s.Projects()) != 1 {
		t.Fatal("应记下项目")
	}
}

func TestWorkspaceCRUDAndPersist(t *testing.T) {
	d := mk(t, "api", "web", "docs")
	file := filepath.Join(t.TempDir(), "ws.json")
	s := Open(file)
	w, err := s.SaveWorkspace(Workspace{Folders: []Folder{{Path: d[0]}, {Path: d[1]}, {Path: d[0]}}})
	if err != nil {
		t.Fatal(err)
	}
	if w.ID == "" || len(w.Folders) != 2 || w.Name != "api 工作区" {
		t.Fatalf("去重 + 默认名 + 分配 id:%+v", w)
	}
	if w.Primary() != d[0] || len(w.Extra()) != 1 || w.Extra()[0] != d[1] {
		t.Fatalf("主文件夹与附加文件夹:%+v", w)
	}
	w.Folders = append(w.Folders, Folder{Path: d[2], Name: "文档"})
	w.LoadMD = true
	if _, err := s.SaveWorkspace(w); err != nil {
		t.Fatal(err)
	}
	// 重新打开:内容还在
	s2 := Open(file)
	got, ok := s2.Workspace(w.ID)
	if !ok || len(got.Folders) != 3 || got.Folders[2].Label() != "文档" || !got.LoadMD {
		t.Fatalf("落盘后重读:%+v", got)
	}
	if n := len(s2.WorkspacesOf(d[1])); n != 1 {
		t.Fatalf("被 1 个工作区引用:%d", n)
	}
	if err := s2.RemoveWorkspace(w.ID); err != nil {
		t.Fatal(err)
	}
	if len(Open(file).Workspaces()) != 0 {
		t.Fatal("删除后应为空")
	}
}

func TestWorkspaceErrors(t *testing.T) {
	d := mk(t, "a")
	s := NewMem()
	s.Exists = isDir
	if _, err := s.SaveWorkspace(Workspace{}); err == nil {
		t.Fatal("没有文件夹应报错")
	}
	if _, err := s.SaveWorkspace(Workspace{Folders: []Folder{{Path: d[0]}, {Path: d[0] + "/missing"}}}); err == nil {
		t.Fatal("有不存在的文件夹应报错")
	}
	var many []Folder
	for i := 0; i < MaxFolders+1; i++ {
		many = append(many, Folder{Path: d[0]})
	}
	if _, err := s.SaveWorkspace(Workspace{Folders: many}); err != ErrTooMany {
		t.Fatalf("超过上限:%v", err)
	}
}

func TestCorruptFileIsKept(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "ws.json")
	_ = os.WriteFile(file, []byte("{not json"), 0o600)
	s := Open(file)
	if len(s.Workspaces()) != 0 {
		t.Fatal("损坏时从空开始")
	}
	es, _ := os.ReadDir(dir)
	found := false
	for _, e := range es {
		if len(e.Name()) > 8 && e.Name()[:8] == "ws.json." {
			found = true
		}
	}
	if !found {
		t.Fatal("损坏的文件应改名留底")
	}
}

func TestClean(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := Clean("~/x/"); got != filepath.Join(home, "x") {
		t.Fatalf("~ 展开:%s", got)
	}
	if Clean("  ") != "" {
		t.Fatal("空白应为空")
	}
}

func TestByPrimary(t *testing.T) {
	d := mk(t, "a", "b")
	s := NewMem()
	s.Exists = isDir
	w, err := s.SaveWorkspace(Workspace{Folders: []Folder{{Path: d[0]}, {Path: d[1]}}, LoadMD: true})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s.ByPrimary(d[0] + "/")
	if !ok || got.ID != w.ID || !got.LoadMD || len(got.Extra()) != 1 {
		t.Fatalf("按主文件夹找工作区:%+v %v", got, ok)
	}
	if _, ok := s.ByPrimary(d[1]); ok {
		t.Fatal("附加文件夹不是主文件夹")
	}
}
