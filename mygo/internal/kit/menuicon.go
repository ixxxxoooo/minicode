package kit

import (
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// 右键菜单(窗口内自绘,见 App.View 的 SetMenuPresentation)里的图标:
// 复用设计稿的 24 格线性图标,按「模板图标」处理,跟随菜单文字色(危险项再用 IconColor 上色)。

var (
	menuIconMu    sync.Mutex
	menuIconCache = map[string]ui.MenuIcon{}
)

// MenuIcon 返回名为 name 的菜单图标(进程内缓存,每个只生成一次);生成失败返回 nil(菜单项不显示图标)。
func MenuIcon(name string) ui.MenuIcon {
	menuIconMu.Lock()
	defer menuIconMu.Unlock()
	if ic, ok := menuIconCache[name]; ok {
		return ic
	}
	var out ui.MenuIcon
	if ic, err := mygo.NewMenuIconSVG(svgOf(name), true); err == nil && ic != nil {
		out = ic
	}
	menuIconCache[name] = out
	return out
}
