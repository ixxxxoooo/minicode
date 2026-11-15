package app

import "testing"

// 整页功能要有出口:再点一次入口图标、或点导航分段,都回到工作台;设置页不受影响。
func TestPageExit(t *testing.T) {
	a, _ := newTestApp(t)
	for _, p := range pages {
		a.screen = ScreenWork
		a.togglePage(p.screen)
		if a.screen != p.screen {
			t.Fatalf("%s:第一次点应进入", p.name)
		}
		a.togglePage(p.screen)
		if a.screen != ScreenWork {
			t.Fatalf("%s:再点一次应回到工作台", p.name)
		}
		a.togglePage(p.screen)
		a.leavePage()
		if a.screen != ScreenWork {
			t.Fatalf("%s:点导航分段应回到工作台", p.name)
		}
	}
	a.screen = ScreenUsage
	a.leavePage()
	if a.screen != ScreenWork {
		t.Fatal("用量整页也应能离开")
	}
	a.screen = ScreenSettings
	a.leavePage()
	if a.screen != ScreenSettings {
		t.Fatal("设置页有自己的「返回应用」,leavePage 不该动它")
	}
}
