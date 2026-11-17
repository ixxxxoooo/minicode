package app

import (
	"testing"
	"time"
)

func TestFlowRiseDelay(t *testing.T) {
	prev := time.Duration(-1)
	for i := 0; i < 80; i++ {
		d := flowRiseDelay(i)
		if d < prev {
			t.Fatalf("stagger %d: 延迟 %v 小于上一档 %v", i, d, prev)
		}
		if d > flowRiseCap {
			t.Fatalf("stagger %d: 延迟 %v 超过封顶 %v", i, d, flowRiseCap)
		}
		prev = d
	}
	if flowRiseDelay(100) != flowRiseCap {
		t.Fatalf("长列表应封顶在 %v", flowRiseCap)
	}
}

func TestFlowRiseViewport(t *testing.T) {
	const viewH float32 = 400
	const scrollMid float32 = 2000
	cases := []struct {
		name     string
		row      int
		first    int
		last     int
		total    int
		laidOut  bool
		scroll   float32
		view     float32
		wantIns  bool
		wantSt   int
		checkRow int // 非零时用此行代替 row
	}{
		{name: "顶部首行", row: 0, first: 0, last: 5, total: 100, laidOut: true, wantSt: 0},
		{name: "中间视口", row: 50, first: 48, last: 55, total: 100, laidOut: true, wantSt: 2},
		{name: "视口上方", row: 10, first: 20, last: 30, total: 100, laidOut: true, wantIns: true},
		{name: "视口下方", row: 90, first: 20, last: 30, total: 100, laidOut: true, wantIns: true},
		{name: "视口外上行", row: 5, first: 80, last: 90, total: 100, laidOut: true, wantIns: true},
		{name: "视口外下行", row: 99, first: 80, last: 90, total: 100, laidOut: true, wantIns: true},
		{name: "单条行", row: 0, first: 0, last: 0, total: 1, laidOut: true, wantSt: 0},
		{name: "空列表", row: 0, first: 0, last: -1, total: 0, laidOut: false, wantIns: true},
		{name: "布局未知无滚动", row: 3, first: 0, last: -1, total: 20, laidOut: false, wantIns: true},
		{name: "布局未知有滚动", row: 30, first: 0, last: -1, total: 100, laidOut: false, scroll: scrollMid, view: viewH},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := tc.row
			if tc.checkRow != 0 {
				row = tc.checkRow
			}
			st, ins := flowRiseViewport(row, tc.first, tc.last, tc.total, tc.laidOut, tc.scroll, tc.view)
			if ins != tc.wantIns {
				t.Fatalf("instant=%v 想要 %v (stagger=%d)", ins, tc.wantIns, st)
			}
			if !tc.wantIns && st != tc.wantSt {
				t.Fatalf("stagger=%d 想要 %d", st, tc.wantSt)
			}
		})
	}
}

func TestFlowRiseEstimateRows(t *testing.T) {
	if g := flowRiseEstimateFirst(720, 400, 72, 50); g < 8 || g > 12 {
		t.Fatalf("中间滚动估算首行异常: %d", g)
	}
	if flowRiseEstimateFirst(0, 400, 72, 50) != 0 {
		t.Fatal("顶部应为 0")
	}
	last := flowRiseEstimateLast(720, 400, 72, 50, 10)
	if last < 10 || last > 25 {
		t.Fatalf("估算末可见行异常: %d", last)
	}
}
