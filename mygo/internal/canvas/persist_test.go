package canvas

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLibraryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	l, err := OpenLibrary(dir)
	if err != nil {
		t.Fatal(err)
	}
	doc := Doc{Title: "测试", Status: "草稿", Blocks: []Block{{ID: "p", Kind: Para, Text: "hi"}}}
	_, id, err := l.New("测试", doc, "你", "新建", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	l2, err := OpenLibrary(dir)
	if err != nil || len(l2.IDs()) != 1 {
		t.Fatal("应能重新打开")
	}
	s, err := l2.Store(id)
	if err != nil || s.Doc.Title != "测试" {
		t.Fatalf("%+v", s)
	}
}

func TestLibraryImportExport(t *testing.T) {
	dir := t.TempDir()
	l, err := OpenLibrary(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 16, 8, 0, 0, 0, time.UTC)
	doc := Doc{Title: "Plan", Status: "草稿", Blocks: []Block{{ID: "p1", Kind: Para, Text: "你好"}}}
	_, id, err := l.New("Plan", doc, "你", "新建", now)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "plan.mccanvas")
	if err := l.Export(id, out); err != nil {
		t.Fatal(err)
	}
	// 导入到同一个库:不覆盖原画布,换新 id,并设为当前
	id2, err := l.Import(out, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if id2 == id {
		t.Fatalf("导入不应覆盖已有画布: %s", id2)
	}
	if l.Active() != id2 || len(l.IDs()) != 2 {
		t.Fatalf("导入后应有 2 张且当前为新画布: active=%s ids=%v", l.Active(), l.IDs())
	}
	s, err := l.Store(id2)
	if err != nil || s.Doc.Title != "Plan" || len(s.Doc.Blocks) != 1 || len(s.Versions) == 0 {
		t.Fatalf("内容不一致: %+v err=%v", s, err)
	}
	// 再开一次库:两张都在
	l2, _ := OpenLibrary(dir)
	if len(l2.IDs()) != 2 {
		t.Fatalf("重新打开应有 2 张: %v", l2.IDs())
	}
	// 非画布文件
	bad := filepath.Join(t.TempDir(), "bad.mccanvas")
	_ = os.WriteFile(bad, []byte("hello"), 0o644)
	if _, err := l.Import(bad, now); err == nil {
		t.Fatal("非 JSON 应报错")
	}
	empty := filepath.Join(t.TempDir(), "e.mccanvas")
	_ = os.WriteFile(empty, []byte("{}"), 0o644)
	if _, err := l.Import(empty, now); err == nil {
		t.Fatal("空对象应报错")
	}
}
