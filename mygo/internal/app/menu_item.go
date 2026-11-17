package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 菜单项的统一入口(设计稿 menus 规则:③ 同一动作在不同对象里叫法与图标一致;④ 有快捷键就显示)。
//
// 所有右键 / 「···」菜单一律用 menuItem(m, "文字") 起项:图标按文字查同一张表,所以「复制路径」
// 在文件树、标签、SCM、问题行里永远是同一个图标;文字里「两个空格 + 快捷键」(如 "转到定义  F12")
// 会自动拆成右侧的快捷键提示,选项本身的文字不带快捷键,菜单重建后仍稳定。

// menuIconTable 是「菜单文字 → 设计稿图标名」。末尾的「…」查表前先去掉。
var menuIconTable = map[string]string{
	// 打开 / 导航
	"打开": "code", "打开文件": "code", "在工作区打开": "code", "用编辑器打开": "edit",
	"打开更改": "diff", "打开该次提交里的改动": "diff", "查看差异": "diff",
	"在审查标签中查看": "eye", "在终端面板查看": "terminal",
	"在侧边打开": "split", "在侧边预览": "eye", "向右拆分": "split", "向下拆分": "split",
	"在浏览器中打开": "globe", "在新窗口中打开": "arrow", "在此处打开终端": "terminal",
	"在文件树中定位": "tree", "全部折叠": "collapseAll", "全部展开": "tree", "折叠其它": "collapseAll",
	"复制该行": "copy", "复制全部引用": "copy",
	"转到定义": "arrow", "转到类型定义": "arrow", "转到实现": "arrow", "转到位置": "arrow",
	"预览定义": "eye", "查找所有引用": "search",
	"在文件夹中搜索": "search", "仅在此文件夹搜索": "search", "查找": "search", "在终端中查找": "search",
	// 编辑
	"重命名": "edit", "重命名符号": "edit", "格式化文档": "align", "格式化选定内容": "align",
	"源代码操作": "spark", "自动换行": "wrap", "开启自动换行": "wrap", "关闭自动换行": "wrap",
	"显示迷你地图": "panelR", "默认以预览打开 Markdown": "code", "切换为源码": "code", "切换为预览": "code",
	"命令面板": "cmd", "全选": "list", "剪切": "copy", "粘贴": "copy",
	"新建文件": "filePlus", "新建文件夹": "folderPlus", "新建终端": "plus",
	// 会话 / Claude
	"问 Claude": "spark", "行内编辑": "spark", "让 Claude": "spark", "让 Claude 解释": "spark", "让 Claude 修复": "spark",
	"让 Claude 修改这段代码": "spark", "引用到对话": "chat",
	"置顶": "pin", "取消置顶": "pin", "固定标签": "pin", "归档": "archive", "取消归档": "archive",
	"分叉会话": "fork", "复制会话链接": "link", "历史会话": "history",
	"显示文件历史": "clock", "重新打开已关闭的标签": "clock",
	// 复制 / 访达
	"复制": "copy", "复制路径": "copy", "复制相对路径": "copy", "复制全部": "copy", "复制内容": "copy",
	"复制名称": "copy", "复制标题": "copy", "复制消息": "copy", "复制此行": "copy", "复制匹配行": "copy",
	"复制链接": "copy", "复制哈希": "copy", "复制会话 ID": "copy", "复制为 Markdown": "copy",
	"复制路径与行号": "copy", "复制所有标签路径": "copy",
	"在访达中显示": "folder", "在访达中显示文件": "folder", "在访达中显示项目": "folder",
	"在访达中打开当前目录": "folder",
	// 关闭 / 危险
	"关闭": "x", "关闭其他": "x", "关闭右侧": "x", "关闭已保存": "x", "关闭全部": "x", "关闭终端": "x",
	"忽略此结果": "x", "恢复此结果": "x", "从工作区移出": "x",
	"删除": "trash", "删除会话": "trash", "移到废纸篓": "trash", "清空输出": "trash", "清屏": "trash",
	// Git
	"暂存更改": "plus", "全部暂存": "plus", "取消暂存": "minus", "全部取消暂存": "minus",
	"放弃更改": "undo", "放弃全部更改": "undo", "还原(revert)": "undo", "回退到这里": "rewind",
	"储藏所有更改": "layers", "弹出最近的储藏": "layers", "刷新": "refresh",
	"拉取": "download", "全部拉取": "download", "推送": "upload", "同步": "sync", "全部同步": "sync",
	"切换分支": "branch", "从此提交创建分支": "branch", "在源代码管理中查看": "branch",
	"在图谱中查看": "fork", "拣选(cherry-pick)": "fork", "检出此提交": "commit",
	"与 HEAD 比较": "diff", "与工作区比较": "diff", "比较这两个提交": "diff",
	"切换左右 / 统一": "swap", "忽略空白差异": "diff", "显示空白差异": "diff",
	"开启行内 Git 追溯": "commit", "关闭行内 Git 追溯": "commit",
	"切换为树形视图": "list", "切换为列表视图": "list", "设为活动仓库": "arrow",
	"终端设置": "gear", "导出为 .mccanvas": "download",
}

// menuIconFor 按文字找图标名;表里没有就按开头的动词兜底,再不行返回空(该项不显示图标)。
func menuIconFor(label string) string {
	label = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(label), "…"))
	if n, ok := menuIconTable[label]; ok {
		return n
	}
	switch {
	case strings.HasPrefix(label, "复制"):
		return "copy"
	case strings.HasPrefix(label, "在访达"):
		return "folder"
	case strings.HasPrefix(label, "关闭"):
		return "x"
	case strings.HasPrefix(label, "让 Claude"), strings.HasPrefix(label, "问 Claude"):
		return "spark"
	case strings.HasPrefix(label, "转到"):
		return "arrow"
	case strings.HasPrefix(label, "删除"), strings.HasPrefix(label, "清空"):
		return "trash"
	}
	return ""
}

// menuItem 起一个带统一图标的菜单项;「文字  快捷键」自动拆成文字 + 右侧快捷键提示。
// 之后照常链 Disabled / Tooltip / Danger / Trailing / Chosen(Trailing 会覆盖拆出来的快捷键)。
func menuItem(m *ui.Menu, label string) *ui.MenuItem {
	name, key := label, ""
	if i := strings.Index(label, "  "); i > 0 {
		name, key = label[:i], strings.TrimSpace(label[i:])
	}
	it := m.Item(name)
	if n := menuIconFor(name); n != "" {
		it.Icon(kit.MenuIcon(n))
	}
	if key != "" {
		it.Trailing(key)
	}
	return it
}
