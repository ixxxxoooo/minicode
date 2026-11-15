package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 文件名以 zz 开头,保证在本包其它界面测试之后运行:它们渲染过的界面里,
// 凡是请求了图标表里没有的图标名(会被画成 ✕),这里统一报出来。
func TestZZNoUnknownIcons(t *testing.T) {
	if u := kit.UnknownIcons(); len(u) > 0 {
		t.Fatalf("界面请求了不存在的图标(会显示成 ✕):%v", u)
	}
}
