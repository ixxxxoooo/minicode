package kit

import (
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// 流光(液态玻璃)主题的材质入口:界面代码只调这里,不直接依赖 glass 插件,
// 以后要换实现(例如给低配机器一个「简化玻璃」)只改这一个文件。
//
// 分工(与设计稿 glow-spec 一致):
//   - 大面积、背后固定只有壁纸的面板 / 浮层 / 输入框:用插件的真玻璃(模糊 + 折射 + 高光边);
//   - 按钮、选中行、输入框这类小控件:用 theme.GlassTone 的半透明色 + 描边,不走着色器
//     (每块真玻璃都要抓一次背后画面,小控件成十上百,代价不划算)。
//
// @author ixxxxoooo

// GlassMat 返回面板用的玻璃材质。clarity:0 = 最磨砂(对话 / 整页 / 浮层,文字多),
// 约 0.35 = 更通透(导航栏、工作区)。系统「降低透明度」时插件自己退成不透明的表面色。
func GlassMat(clarity float32) ui.Material { return glass.Glass{Clarity: clarity} }

// GlassTinted 返回带着色的玻璃(主要按钮、发送键):tint 的透明度决定染得多重。
func GlassTinted(tint ui.Color) ui.Material { return glass.Glass{Tint: tint, Interactive: true} }

// GlassControl 返回可交互的小块玻璃(悬停亮边、按下微胀),给标题栏搜索胶囊、分段滑块这类少量大控件用。
func GlassControl() ui.Material { return glass.Glass{Interactive: true} }
