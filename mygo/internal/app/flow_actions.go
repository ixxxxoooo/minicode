package app

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 会话流里的「跳转 / 复制 / 打开」动作:步骤行悬停出现的小按钮、命令卡与差异卡的复制、
// 助手正文里的链接。入口一律落到工作区(文件 → 编辑器标签,差异 → 差异标签,命令 → 底部终端面板),
// 找不到目标时用提示说明,不静默失败。

// copyOut 把文字写进剪贴板并提示(下一帧由 root 写入)。
func (a *App) copyOut(s, what string) {
	if s == "" {
		a.toast = "没有可复制的内容"
		return
	}
	a.copyText = s
	a.toast = "已复制" + what
}

// openPathRef 打开会话流里提到的文件引用:`路径[:行[:列]]`,优先工作区内路径,否则尝试磁盘绝对路径。
func (a *App) openPathRef(ref string) bool {
	p, line, col := splitPathRef(ref)
	return a.openAnyFile(p, line, col)
}

// splitPathRef 拆 `a/b.rs:12:3` 成路径、行、列(后缀不是数字就不拆)。
func splitPathRef(ref string) (p string, line, col int) {
	p = ref
	for i := 0; i < 2; i++ {
		k := strings.LastIndexByte(p, ':')
		if k < 0 {
			break
		}
		n, err := strconv.Atoi(p[k+1:])
		if err != nil {
			break
		}
		p = p[:k]
		if i == 0 {
			line = n
		} else {
			col, line = line, n // 倒着拆:先拿到的是列
		}
	}
	return p, line, col
}

// openURL 用系统浏览器打开 http(s) 链接(其它协议一律不开,链接可能来自模型输出)。
func (a *App) openURL(u string) bool {
	if !ui.SafeToOpen(u) || strings.HasPrefix(strings.ToLower(u), "mailto:") {
		return false
	}
	if err := exec.Command("open", u).Start(); err != nil {
		a.toast = "打不开链接:" + err.Error()
		return false
	}
	return true
}

// openMdLink 是 Markdown 链接的点击处理:网址交给系统,路径在工作区打开。
func (a *App) openMdLink(dest string) {
	dest = strings.TrimPrefix(dest, pathLinkScheme)
	if strings.HasPrefix(dest, "#") {
		return
	}
	if a.openURL(dest) {
		return
	}
	if a.openDirRef(dest) {
		return
	}
	if !a.openPathRef(dest) {
		a.toast = "找不到 " + dest
	}
}

// linkMenu 是链接的右键菜单:路径 → 打开 / 复制;网址 → 打开 / 复制。
func (a *App) linkMenu(m *ui.Menu, dest string) {
	dest = strings.TrimPrefix(dest, pathLinkScheme)
	if strings.HasPrefix(dest, "http") {
		if menuItem(m, "在浏览器中打开").Chosen() {
			a.openURL(dest)
		}
		if menuItem(m, "复制链接").Chosen() {
			a.copyOut(dest, "链接")
		}
		return
	}
	if menuItem(m, "在工作区打开").Chosen() {
		a.openMdLink(dest)
	}
	if p, _, _ := splitPathRef(dest); p != "" {
		if abs, ok := a.resolveUserPath(p); ok && menuItem(m, "在访达中显示").Chosen() {
			a.revealInFinder(abs)
		}
	}
	if menuItem(m, "复制路径").Chosen() {
		a.copyOut(dest, "路径")
	}
}

// stepActions 是步骤行悬停时右侧出现的小按钮(只列这一行真正能做的事)。
func (a *App) stepActions(c *ui.Context, sp step) {
	if sp.imgAbs != "" { // 读图步骤:原文件可能早被清理,这里打开的是步骤里显示的那张图
		if kit.Ib(c, "ext", kit.IbOpt{Small: true, Tip: "查看图片"}).Clicked() {
			a.openImageLightboxPath(sp.imgAbs)
		}
	} else if sp.path != "" {
		if kit.Ib(c, "ext", kit.IbOpt{Small: true, Tip: "在工作区打开文件"}).Clicked() && !a.openPathRef(joinRef(sp.path, sp.line)) {
			a.toast = "文件已不存在:" + baseName(sp.path)
		}
		if sp.diffAs && kit.Ib(c, "diff", kit.IbOpt{Small: true, Tip: "在工作区查看差异"}).Clicked() {
			a.OpenDiffTab(sp.path)
		}
	}
	if sp.term != nil && kit.Ib(c, "terminal", kit.IbOpt{Small: true, Tip: "在终端面板查看输出"}).Clicked() {
		a.viewInPanel(sp.term)
	}
	if sp.url != "" && kit.Ib(c, "globe", kit.IbOpt{Small: true, Tip: "在浏览器中打开"}).Clicked() {
		a.openURL(sp.url)
	}
	if sp.copyStr != "" && kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制"}).Clicked() {
		a.copyOut(sp.copyStr, "")
	}
}

func joinRef(p string, line int) string {
	if line > 0 {
		return p + ":" + strconv.Itoa(line)
	}
	return p
}
