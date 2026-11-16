package theme

import "testing"

// v4 字号阶梯:默认界面大小 100%,界面 13 / 阅读 14 / 代码 12。
func TestTypographyBasesAtDefaultUIScale(t *testing.T) {
	SetUIScale(DefaultUIScale)
	if DefaultUIScale != 100 {
		t.Fatalf("默认界面大小应为 100,得 %d", DefaultUIScale)
	}
	if FsBody != 13 || FsRead != 14 || FsMono != 12 || FsCap != 12 {
		t.Fatalf("默认阶梯应为 body 13 / read 14 / mono 12 / cap 12,得 %v / %v / %v / %v", FsBody, FsRead, FsMono, FsCap)
	}
	if FsRead-FsBody > 1.5 {
		t.Fatalf("阅读正文只应比界面正文大约 1px,得 %v vs %v", FsRead, FsBody)
	}
	if FsCap < minCapPx {
		t.Fatalf("cap 不应低于 %v,得 %v", minCapPx, FsCap)
	}
	ratio := FsMono / FsRead
	if ratio < 0.82 || ratio > 0.9 {
		t.Fatalf("mono/read 比例异常: %v", ratio)
	}
}

// 缩放到最小 / 最大后阶梯仍然递增,cap 不低于下限。
func TestTypographyScaleKeepsOrder(t *testing.T) {
	defer SetUIScale(DefaultUIScale)
	for _, pct := range []int{MinUIScale, DefaultUIScale, MaxUIScale} {
		SetUIScale(pct)
		if !(FsMicro <= FsCtl && FsCtl <= FsCap && FsCap <= FsBody && FsBody <= FsRead && FsRead <= FsHead && FsHead <= FsDisplay) {
			t.Fatalf("%d%% 下阶梯应递增: %v %v %v %v %v %v %v", pct, FsMicro, FsCtl, FsCap, FsBody, FsRead, FsHead, FsDisplay)
		}
		if FsCap < minCapPx {
			t.Fatalf("%d%% 下 cap=%v 低于下限 %v", pct, FsCap, minCapPx)
		}
	}
}
