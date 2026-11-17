package app

import "testing"

func TestLightboxFitScale(t *testing.T) {
	k := lightboxFitScale(2000, 1000, 800, 600)
	if k > 1 || k <= 0 {
		t.Fatalf("适应系数应在 (0,1],得 %v", k)
	}
	w, h := lightboxDisplaySize(1000, 800, 800, 600, 1)
	if w > 800*lbFitW+1 || h > 600*lbFitH+1 {
		t.Fatalf("适应尺寸不应超过上限: %v×%v", w, h)
	}
	pct := lightboxZoomPct(1000, 800, 800, 600, 2)
	if pct < 100 {
		t.Fatalf("2× 缩放百分比应 ≥100,得 %d", pct)
	}
}

func TestPinnedEditorBodyMax(t *testing.T) {
	h := pinnedEditorBodyMax(600, 40)
	if h < pinnedEditorMinBody {
		t.Fatalf("正文最大高度不应小于最小值: %v", h)
	}
	if h > 600 {
		t.Fatalf("不应超过视口: %v", h)
	}
}
