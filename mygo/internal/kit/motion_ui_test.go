package kit

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestDrawerSlideMotionOff(t *testing.T) {
	SetMotion(false)
	e := ui.Box(nil)
	DrawerSlide(e, time.Now(), true, nil)
}

func TestTextShimmerOff(t *testing.T) {
	SetShimmer(2)
	e := ui.Text(nil, "x")
	if RunText(e) != e {
		t.Fatal("关闭动效时不应包装")
	}
}
