package app

import "testing"

// 设置 → 源代码管理:第一项是主工作树,只显示标签、不给删除入口;
// 渲染路径走缓存(演示模式直接读演示数据)。
func TestSettingsWorktreeMainProtected(t *testing.T) {
	a, tt := newTestApp(t)
	a.openSettings("scm")
	tt.Frame()
	if !tt.HasText("主工作树") {
		t.Fatalf("第一项应标注主工作树,实际文本: %v", tt.Texts())
	}
}
