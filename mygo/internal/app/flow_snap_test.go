package app

import "testing"

// 吸附宽度:向下取整到台阶、不超过原宽;窄于两个台阶时原样返回。
func TestSnapDown(t *testing.T) {
	cases := []struct{ in, want float32 }{
		{760, 720}, {720, 720}, {719.9, 640}, {352, 320}, {160, 160}, {100, 100}, {0, 0},
	}
	for _, c := range cases {
		got := snapDown(c.in, morphSnapStep)
		if got != c.want || got > c.in {
			t.Errorf("snapDown(%v)=%v,期望 %v", c.in, got, c.want)
		}
	}
}

// 没有真实窗口(测试 / 截图)时不吸附,直达终态。
func TestFlowSnapWidthNoWindow(t *testing.T) {
	a := &App{}
	a.chatW = 700
	if _, ok := a.flowSnapWidth(); ok {
		t.Fatal("无窗口时不应吸附")
	}
}
