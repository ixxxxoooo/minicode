package app

import "testing"

// 文件树标题行:有展开的文件夹时「全部收起」可点,点后全部收起;没有展开时置灰。
func TestTreeCollapseAll(t *testing.T) {
	a, tt := demoIDE(t)
	d := a.ide
	d.open = map[string]bool{"src": true, "docs": true}
	tt.Frame()
	if a.treeOpenCount() != 2 {
		t.Fatalf("准备工作:应有 2 个展开的文件夹,实际 %d", a.treeOpenCount())
	}
	if err := tt.Click("全部收起"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.treeOpenCount() != 0 {
		t.Fatalf("点「全部收起」后应没有展开的文件夹,实际 %d", a.treeOpenCount())
	}
	d.sel = "src/lib/a.rs"
	d.open = map[string]bool{"src": true, "src/lib": true}
	a.collapseTree()
	if d.sel != "src" {
		t.Fatalf("选中项落在被收起的文件夹里时应改选最外层文件夹,实际 %q", d.sel)
	}
}
