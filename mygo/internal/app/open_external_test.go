package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/canvas"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

func TestIsSecretPrefKey(t *testing.T) {
	for _, k := range []string{"provider.apiKey", "x.secret", "auth.token", "a.password", "provider.key", "my_key"} {
		if !isSecretPrefKey(k) {
			t.Errorf("%q 应当算密钥键", k)
		}
	}
	for _, k := range []string{"general.sendKey", "appearance.theme", "editor.fontSize", "shortcuts.palette", "keyboard.layout"} {
		if isSecretPrefKey(k) {
			t.Errorf("%q 不该算密钥键", k)
		}
	}
	got, dropped := stripSecretPrefs(map[string]json.RawMessage{"a.token": []byte(`"x"`), "editor.fontSize": []byte(`14`)})
	if dropped != 1 || len(got) != 1 || got["editor.fontSize"] == nil {
		t.Fatalf("剔除结果不对: %v dropped=%d", got, dropped)
	}
}

func TestOpenExternalFileMissingAndQueue(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.OpenExternalFile(filepath.Join(t.TempDir(), "nope.mccanvas"))
	if !strings.Contains(a.toast, "找不到文件") {
		t.Fatalf("toast: %q", a.toast)
	}
	// 窗口没出来:先排队,放开后再处理
	a.toast = ""
	a.HoldExternalOpen(true)
	a.OpenExternalFile(filepath.Join(t.TempDir(), "later.mcsettings"))
	if len(a.extQueue) != 1 || a.toast != "" {
		t.Fatalf("应排队而不是立刻处理: queue=%v toast=%q", a.extQueue, a.toast)
	}
	a.HoldExternalOpen(false)
	if len(a.extQueue) != 0 || !strings.Contains(a.toast, "找不到文件") {
		t.Fatalf("放开后应处理排队文件: queue=%v toast=%q", a.extQueue, a.toast)
	}
	// 从属窗口不接管
	b := New(Options{Palette: "paper", Scheme: "a", FakeLights: true, Secondary: true})
	b.OpenExternalFile("/不存在")
	if b.toast != "" {
		t.Fatalf("从属窗口不应处理: %q", b.toast)
	}
}

func TestOpenExternalCanvas(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	src, err := canvas.OpenLibrary(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, id, _ := src.New("Imported", canvas.Doc{Title: "Imported", Status: "草稿", Blocks: []canvas.Block{{ID: "p1", Kind: canvas.Para, Text: "内容"}}}, "你", "新建", time.Now())
	file := filepath.Join(t.TempDir(), "x.mccanvas")
	if err := src.Export(id, file); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.OpenExternalFile(file)
	if a.screen != ScreenCanvas || a.canv == nil || a.canv.Doc.Title != "Imported" {
		t.Fatalf("应打开画布页并选中导入的画布: screen=%v canv=%v toast=%q", a.screen, a.canv, a.toast)
	}
	if !strings.Contains(a.toast, "已导入画布") {
		t.Fatalf("toast: %q", a.toast)
	}
	// 坏文件:不动现有数据,给原因
	bad := filepath.Join(t.TempDir(), "bad.mccanvas")
	_ = os.WriteFile(bad, []byte("not json"), 0o644)
	before := a.canv
	a.screen = ScreenWork
	a.OpenExternalFile(bad)
	if a.canv != before || a.screen != ScreenWork || !strings.Contains(a.toast, "无法导入画布") {
		t.Fatalf("坏文件应只提示: screen=%v toast=%q", a.screen, a.toast)
	}
}

func TestOpenExternalWorkspace(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	// 演示模式(注册表在内存里)不导入
	a.OpenExternalFile(writeWsFile(t, projects.Workspace{Name: "demo", Folders: []projects.Folder{{Path: t.TempDir()}}}))
	if !strings.Contains(a.toast, "演示模式") {
		t.Fatalf("演示模式应拒绝: %q", a.toast)
	}
	a.reg = projects.Open(filepath.Join(t.TempDir(), "ws.json"))
	d1, d2 := t.TempDir(), t.TempDir()
	// 全部文件夹都不存在
	a.OpenExternalFile(writeWsFile(t, projects.Workspace{Name: "gone", Folders: []projects.Folder{{Path: "/不存在/a"}, {Path: "/不存在/b"}}}))
	if !strings.Contains(a.toast, "都不存在") || len(a.reg.Workspaces()) != 0 {
		t.Fatalf("全缺失应拒绝: toast=%q ws=%d", a.toast, len(a.reg.Workspaces()))
	}
	// 一半存在:跳过缺失的
	f := writeWsFile(t, projects.Workspace{Name: "全栈", Folders: []projects.Folder{{Path: d1}, {Path: d2}, {Path: "/不存在/c"}}})
	a.OpenExternalFile(f)
	ws := a.reg.Workspaces()
	if len(ws) != 1 || ws[0].Name != "全栈" || len(ws[0].Folders) != 2 {
		t.Fatalf("应导入并跳过缺失文件夹: %+v toast=%q", ws, a.toast)
	}
	if !strings.Contains(a.toast, "跳过 1 个") {
		t.Fatalf("toast 应说明跳过: %q", a.toast)
	}
	// 同一组文件夹再打开:不重复建
	a.OpenExternalFile(f)
	if len(a.reg.Workspaces()) != 1 {
		t.Fatalf("不应重复创建: %d", len(a.reg.Workspaces()))
	}
	// 坏文件
	bad := filepath.Join(t.TempDir(), "bad.mcworkspace")
	_ = os.WriteFile(bad, []byte("{}"), 0o644)
	a.OpenExternalFile(bad)
	if !strings.Contains(a.toast, "无法打开工作区文件") {
		t.Fatalf("坏文件 toast: %q", a.toast)
	}
}

func writeWsFile(t *testing.T, w projects.Workspace) string {
	t.Helper()
	b, err := projects.MarshalWorkspaceFile(w)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "x"+projects.WorkspaceFileExt)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSettingsBackupRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	_ = os.MkdirAll(filepath.Join(home, "Downloads"), 0o755)
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.prefs.Set("editor.fontSize", 17)
	a.prefs.Set("provider.apiKey", "sk-不该出现")
	a.exportSettingsJSON()
	ents, _ := os.ReadDir(filepath.Join(home, "Downloads"))
	if len(ents) != 1 || !strings.HasSuffix(ents[0].Name(), settingsBackupExt) {
		t.Fatalf("应导出一个 .mcsettings: %v toast=%q", ents, a.toast)
	}
	path := filepath.Join(home, "Downloads", ents[0].Name())
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), "sk-") || strings.Contains(string(b), "apiKey") {
		t.Fatalf("备份里不应有密钥: %s", b)
	}
	if !strings.Contains(string(b), "editor.fontSize") {
		t.Fatalf("备份里应有偏好: %s", b)
	}
	// 清掉后导入:先弹确认,确认后才合并
	a.prefs.Reset("editor.fontSize")
	a.OpenExternalFile(path)
	if a.pd.kind == "" {
		t.Fatalf("导入前应先弹确认框: toast=%q", a.toast)
	}
	if a.prefs.Has("editor.fontSize") {
		t.Fatal("确认前不应改偏好")
	}
	if a.pd.ok == nil {
		t.Fatal("确认框应带确认动作")
	}
	a.pd.ok()
	if a.prefs.Int("editor.fontSize", 0) != 17 {
		t.Fatalf("确认后应导入: %d toast=%q", a.prefs.Int("editor.fontSize", 0), a.toast)
	}
}
