package kit

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestBtnLoadingDelayGate(t *testing.T) {
	SetMotion(true)
	b := ui.ButtonBase(nil)
	if btnLoadingSpin(b, ui.Hex("#000")) {
		t.Fatal("未过 400ms 不应转圈")
	}
	_ = b.AnimateTo("loadDly", 1, ui.Animation{Duration: loadingDelay, Ease: ui.Linear})
	time.Sleep(5 * time.Millisecond)
}

func TestTogDisabledNoPanic(t *testing.T) {
	SetMotion(false)
	if Tog(nil, false, "t", TogOpt{Disabled: true}) {
		t.Fatal("禁用开关不应返回 true")
	}
}