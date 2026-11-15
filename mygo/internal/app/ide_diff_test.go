package app

import (
	"strings"
	"testing"
)

func TestHunkStageAndRevert(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	if a.repo == nil {
		t.Fatal("演示应有仓库")
	}
	a.openDiffKind(dkWork, "src/router.rs", true)
	tb := a.ide.tab(diffID(dkWork, "src/router.rs"))
	if tb == nil || tb.dv == nil || !tb.dv.Editable || tb.dk != dkWork {
		t.Fatalf("差异标签没打开 / 应可编辑:%+v", tb)
	}
	hunks := tb.dv.Hunks()
	if len(hunks) == 0 {
		t.Fatal("router.rs 相对暂存区应有改动块")
	}
	work := tb.dv.New.String()
	// 暂存第一块:暂存区的版本 = 旧文本 + 这一块
	a.stageHunk(tb, 0)
	idx, _ := a.repo.Index("src/router.rs")
	if idx != tb.dv.Old.String() || idx == "" || idx == work && len(hunks) > 1 {
		t.Fatalf("暂存这一块后,暂存区应是旧文本 + 该块;得到 %d 字节(工作区 %d)", len(idx), len(work))
	}
	if left := len(tb.dv.Hunks()); left != len(hunks)-1 {
		t.Fatalf("暂存一块后应少一块:%d → %d", len(hunks), left)
	}
	// 还原一块:工作区文件被改回去
	if left := tb.dv.Hunks(); len(left) > 0 {
		before, _ := a.ide.fs.Read("src/router.rs")
		a.revertDiffHunk(tb, 0)
		after, _ := a.ide.fs.Read("src/router.rs")
		if before == after {
			t.Fatal("还原一块后文件内容应变化")
		}
	}
	// 补丁文本
	if hs := tb.dv.Hunks(); len(hs) > 0 {
		if p := a.hunkPatch(tb, hs[0]); !strings.HasPrefix(p, "--- a/src/router.rs") || !strings.Contains(p, "@@") {
			t.Fatalf("补丁格式不对:%q", p)
		}
	}
}

func TestStagedDiffUnstageHunk(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	_ = a.repo.StageContent("src/router.rs", "pub struct Router;\n")
	a.openDiffKind(dkStaged, "src/router.rs", true)
	tb := a.ide.tab(diffID(dkStaged, "src/router.rs"))
	if tb == nil || tb.dv.Editable {
		t.Fatal("暂存区视图应只读")
	}
	if len(tb.dv.Hunks()) == 0 {
		t.Fatal("HEAD ↔ 暂存区应有改动块")
	}
	a.unstageHunk(tb, 0)
	if idx, _ := a.repo.Index("src/router.rs"); idx != tb.dv.New.String() {
		t.Fatal("取消暂存后暂存区应等于还原后的暂存区一侧")
	}
}
