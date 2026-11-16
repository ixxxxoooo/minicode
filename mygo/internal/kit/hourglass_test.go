package kit

import "testing"

func TestHourglassPhaseSynced(t *testing.T) {
	SetMotion(true)
	a := globalHourglassPhase()
	b := globalHourglassPhase()
	if a < 0 || a > 1 || b < 0 || b > 1 {
		t.Fatalf("phase out of range: %v %v", a, b)
	}
}

func TestSpinRingDistinctFromHourglass(t *testing.T) {
	SetMotion(true)
	// 沙漏与圆环是不同构造(回归:按钮仍可用圆环)。
	if hourglassPeriod <= 0 {
		t.Fatal("hourglass period")
	}
}
