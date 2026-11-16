package app

import (
	"fmt"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

// clickOne 在全新应用里点第 idx 个可见控件,返回崩溃信息与总数。
func clickOne(mode Mode, sess string, w, h float32, idx int) (panicMsg string, total int, name string) {
	defer func() {
		if r := recover(); r != nil {
			panicMsg = fmt.Sprint(r)
		}
	}()
	a := New(Options{Palette: "paper", Scheme: "a", Mode: mode, FakeLights: true})
	a.SetProjects(demo.Projects(), sess)
	tt := ui.NewTester(a.View, int(w), int(h))
	tt.Frame()
	tt.Frame()
	var cs []ui.AccessInfo
	for _, i := range tt.AccessAll() {
		if i.Offscreen || i.Disabled || i.Bounds.W < 6 || i.Bounds.H < 6 {
			continue
		}
		switch i.Role {
		case ui.RoleButton, ui.RoleTab, ui.RoleMenuItem, ui.RoleCheckBox, ui.RoleSwitch:
			cs = append(cs, i)
		}
	}
	total = len(cs)
	if idx >= total {
		return
	}
	c := cs[idx]
	name = c.Name
	tt.ClickAt(c.Bounds.X+c.Bounds.W/2, c.Bounds.Y+c.Bounds.H/2)
	tt.Frame()
	tt.Frame()
	tt.Frame()
	return
}

// 冒烟:每个可见按钮点一下都不能让界面线程崩溃(曾有「拆分编辑器」在没有标签时空指针)。
// 数量较多,默认只在 MINI_CLICK_ALL=1 时跑。
func TestClickEverythingNoPanic(t *testing.T) {
	if os.Getenv("MINI_CLICK_ALL") != "1" {
		t.Skip("设 MINI_CLICK_ALL=1 运行")
	}
	for _, mode := range []Mode{ModeAgent, ModeIDE} {
		for _, wh := range [][2]float32{{1280, 800}, {800, 640}} {
			for _, sess := range []string{"fixbug", "ci"} {
				_, total, _ := clickOne(mode, sess, wh[0], wh[1], 1<<30)
				for i := 0; i < total; i++ {
					if msg, _, name := clickOne(mode, sess, wh[0], wh[1], i); msg != "" {
						t.Errorf("mode=%d %vx%v %s 点第 %d 个「%s」崩溃:%s", mode, wh[0], wh[1], sess, i, name, msg)
					}
				}
			}
		}
	}
}
