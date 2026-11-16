package theme

import "testing"

func TestTypographyBasesAtDefaultUIScale(t *testing.T) {
	SetUIScale(DefaultUIScale)
	if FsBody < 14 || FsRead < 16 {
		t.Fatalf("110%% 下 body/read 应更大: body=%v read=%v", FsBody, FsRead)
	}
	if FsCap < minCapPx {
		t.Fatalf("cap 不应低于 %v,得 %v", minCapPx, FsCap)
	}
	ratio := FsMono / FsRead
	if ratio < 0.85 || ratio > 0.92 {
		t.Fatalf("mono/read 比例异常: %v", ratio)
	}
}
