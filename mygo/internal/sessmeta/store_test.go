package sessmeta

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "meta.json")
	s := Open(p)
	_ = s.SetTitle("a", "  新标题 ")
	_ = s.SetPinned("a", true)
	_ = s.SetArchived("b", true)
	_ = s.SetPinned("new:/x", true) // 新会话没有 id,忽略
	r := Open(p)
	if m := r.Get("a"); m.Title != "新标题" || !m.Pinned || m.Archived {
		t.Fatalf("a:%+v", m)
	}
	if !r.Get("b").Archived || r.Len() != 2 {
		t.Fatalf("b 或条数不对:%d", r.Len())
	}
	// 全部还原后记录被清掉
	_ = r.SetTitle("a", "")
	_ = r.SetPinned("a", false)
	if r.Len() != 1 {
		t.Fatalf("还原后应只剩 b:%d", r.Len())
	}
	_ = r.Forget("b")
	if Open(p).Len() != 0 {
		t.Fatal("删除会话后记录应清空")
	}
}

func TestStoreCorruptKept(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "meta.json")
	_ = os.WriteFile(p, []byte("{坏的"), 0o600)
	s := Open(p)
	if s.Len() != 0 {
		t.Fatal("损坏应从空开始")
	}
	ents, _ := os.ReadDir(dir)
	bad := 0
	for _, e := range ents {
		if len(e.Name()) > 9 && e.Name()[:9] == "meta.json" && e.Name() != "meta.json" {
			bad++
		}
	}
	if bad != 1 {
		t.Fatalf("应留底一份:%v", ents)
	}
}

func TestMemNotPersistent(t *testing.T) {
	s := NewMem()
	_ = s.SetPinned("a", true)
	if s.Persistent() || !s.Get("a").Pinned {
		t.Fatal("内存存储应可读写且不落盘")
	}
}
