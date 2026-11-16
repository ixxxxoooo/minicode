package core

import (
	"strings"
	"testing"
)

func TestFormatRespAndTiming(t *testing.T) {
	cases := map[int64]string{420: "420 毫秒", 1800: "1.8 秒", 18400: "18 秒", 75000: "1 分 15 秒"}
	for ms, want := range cases {
		if got := FormatResp(ms); got != want {
			t.Errorf("FormatResp(%d)=%q,想要 %q", ms, got, want)
		}
	}
	d := TurnTiming{StartupMs: 600, FirstMs: 980, TotalMs: 2000}.DetailText()
	for _, s := range []string{"启动 claude 进程并握手 600 毫秒", "第一个响应 980 毫秒", "回合结束 2.0 秒"} {
		if !strings.Contains(d, s) {
			t.Errorf("明细缺少 %q:\n%s", s, d)
		}
	}
}

func TestApplyTurnTiming(t *testing.T) {
	st := NewState()
	st.Items = append(st.Items, &User{ID: "u1", Text: "hi"})
	st.Apply(EvTurnTiming{Timing: TurnTiming{TotalMs: 1500}}) // 空 ID = 最近一条用户消息
	if st.Timings["u1"].TotalMs != 1500 {
		t.Fatalf("应记到 u1:%v", st.Timings)
	}
	st.Apply(EvTurnTiming{ID: "u9", Timing: TurnTiming{}}) // 总耗时为 0 不记
	if _, ok := st.Timings["u9"]; ok {
		t.Fatal("TotalMs=0 不应记录")
	}
}
