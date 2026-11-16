package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// 欢迎页起手卡(对齐设计稿 .starters):卡片固定 150 宽;宽档一排四张;中档 flex-wrap 居中,
// 放不下就 3+1 折行(「审查改动」居中落在第二行);起手卡在「最近会话」之上。
func TestWelcomeStarterLayout(t *testing.T) {
	a, tt := newTestApp(t)
	a.NewSession()
	a.rightOn = false
	wide := ui.NewTester(a.View, 1800, 900) // 对话栏独占时 ≥ 640 → 宽档
	titles := []string{"修复一个 Bug", "解释这个项目", "补充测试", "审查改动"}

	// chatW 每帧按实测宽度刷新:大窗口 = 宽档,默认窗口 = 中档
	find := func(wantTier int) (xs, ys [4]float32, recentY float32) {
		tt := tt
		if wantTier == tierWide {
			tt = wide
		}
		tt.Frame()
		tt.Frame()
		if a.chatTier() != wantTier {
			t.Skipf("测试窗口下对话栏宽 %v,不是目标档 %d", a.chatW, wantTier)
		}
		for i, s := range titles {
			r, ok := tt.Find(s)
			if !ok {
				t.Fatalf("没找到起手卡「%s」", s)
			}
			xs[i], ys[i] = r.X, r.Y
		}
		if r, ok := tt.Find("最近会话"); ok {
			recentY = r.Y
		}
		return
	}

	xs, ys, ry := find(tierWide)
	if !(ys[0] == ys[1] && ys[1] == ys[2] && ys[2] == ys[3]) {
		t.Errorf("宽档四张应在同一排:%v", ys)
	}
	if ry != 0 && ry <= ys[0] {
		t.Errorf("宽档:最近会话(y=%v)应在起手卡(y=%v)下面", ry, ys[0])
	}

	// 中档:Agent 形态右侧工作区展开(设计稿截图的场景),对话栏约 488 宽
	a.rightOn = true
	xs, ys, ry = find(tierMid)
	if ys[0] != ys[1] || ys[1] != ys[2] || ys[3] <= ys[0] {
		t.Errorf("中档应是 3+1 折行:%v", ys)
	}
	if !(xs[1] > xs[0] && xs[2] > xs[1]) {
		t.Errorf("中档第一排三张应从左到右排开:%v", xs)
	}
	// 第二排只有一张,居中:它的左边缘应与中间那张对齐(三张居中时中间张就是整排的中线)
	if d := xs[3] - xs[1]; d < -1 || d > 1 {
		t.Errorf("中档第二排单张应居中(与中间张左缘对齐):%v", xs)
	}
	if ry != 0 && ry <= ys[3] {
		t.Errorf("中档:最近会话(y=%v)应在起手卡末行(y=%v)下面", ry, ys[3])
	}
}
