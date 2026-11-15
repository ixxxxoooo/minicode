package app

import (
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 块级「撤销」之后可以「重新应用」;文件在撤销后又被改过就如实拒绝,不乱改别处。
func TestReviewReapplyHunk(t *testing.T) {
	a, tt := newTestApp(t)
	tt.Frame()
	files := a.changedFiles()
	var f chFile
	var d *core.DiffBlock
	var idx int
	for _, cf := range files {
		for i, b := range cf.blocks {
			if b.New != "" && b.Old != "" && b.Old != b.New {
				if cur, err := a.ide.fs.Read(a.wsRel(b.Path)); err == nil && strings.Contains(cur, b.New) {
					f, d, idx = cf, b, i
				}
			}
		}
	}
	if d == nil {
		t.Skip("演示数据里没有可往返的改动块")
	}
	rel := a.wsRel(d.Path)
	before, _ := a.ide.fs.Read(rel)
	a.rv.init()
	a.revertHunk(f, idx)
	k := hunkKey(f.path, idx)
	if a.rv.hunk[k] != "rev" {
		t.Fatalf("撤销后应标 rev,得到 %q(toast=%q)", a.rv.hunk[k], a.toast)
	}
	mid, _ := a.ide.fs.Read(rel)
	if mid == before {
		t.Fatal("撤销应改动文件")
	}
	a.reapplyHunk(f, idx)
	after, _ := a.ide.fs.Read(rel)
	if after != before || a.rv.hunk[k] != "" {
		t.Fatalf("重新应用应还原文件并清除状态:state=%q toast=%q", a.rv.hunk[k], a.toast)
	}
	// 撤销后文件被改:拒绝
	a.revertHunk(f, idx)
	cur, _ := a.ide.fs.Read(rel)
	_ = a.ide.fs.Write(rel, "// 外部改动\n"+strings.Replace(cur, d.Old, "被改过", 1))
	a.reapplyHunk(f, idx)
	if a.rv.hunk[k] != "rev" || !strings.Contains(a.toast, "无法重新应用") {
		t.Fatalf("文件被改过应拒绝:state=%q toast=%q", a.rv.hunk[k], a.toast)
	}
}
