package kit

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestBtnLoadingDelay(t *testing.T) {
	SetMotion(true)
	defer SetMotion(true)
	b := ui.ButtonBase(nil)
	if btnLoadingSpin(b, ui.Hex("#000")) {
		t.Fatal("未过 400ms 不应显示转圈")
	}
	_ = b.AnimateTo("loadDly", 1, ui.Animation{Duration: loadingDelay, Ease: ui.Linear})
	time.Sleep(5 * time.Millisecond)
}

func TestClickPressNoPanic(t *testing.T) {
	SetMotion(false)
	e := ui.Box(nil)
	clickPressScale(e)
	clickFocusRing(e)
	_ = e
}
