package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/matico"
	"github.com/ixxxxoooo/minicode/internal/mdcompat"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// @author ixxxxoooo

// 会话里出现的路径(行内代码写的文件 / 文件夹)可以直接点开,参照 BaoCode:
// 只有「本机确实存在」的路径才成为链接并带文件 / 文件夹图标,不存在的仍是普通行内代码。
// 存在性在后台查一次并缓存(界面线程不做磁盘 IO),查完下一帧才出现链接。

// pathRefState 是一个路径的探测结果。
type pathRefState int

const (
	pathRefUnknown pathRefState = iota // 还没查
	pathRefPending                     // 后台查询中
	pathRefNone                        // 不存在 / 不是可点的路径
	pathRefFile                        // 存在的文件
	pathRefDir                         // 存在的文件夹
)

// pathLinkScheme 是路径链接目的地的前缀。打了 MyGo 补丁(scripts/mygo-overlay.sh)时,带它的链接悬停不画下划线,
// 改成手型光标 + 淡底;没打补丁时就是普通链接(悬停有下划线,功能一样)。
const pathLinkScheme = "mc-path:"

// pathRefCap 是缓存条数上限,超了整表清空(会话很长时防止无限增长)。
const pathRefCap = 4000

// pathChipMaxText 是路径胶囊里文字的最大宽度(DIP),超出从中间省略。
const pathChipMaxText = 380

// pathRefCache 缓存「路径文字 → 探测结果」,只在界面线程读写。
type pathRefCache struct {
	m map[string]pathRefState
	// ver 每次有新结果落地就加一,进 Markdown 选项缓存键,让段落按新答案重建。
	ver int
}

// pathExtRe 匹配「没有目录、只有文件名」的写法(名字里可以有空格和中文):name.ext。
var pathExtRe = regexp.MustCompile(`^[^/\\]+\.[A-Za-z0-9]{1,10}$`)

// looksLikePath 判断一段行内代码是否「长得像路径」,没通过就不去查磁盘(查盘本身不贵,但没必要)。
// 条件:不是网址、不太长、不含换行 / 尖括号 / 反引号,且含 /(绝对路径、~/、相对路径)或形如 name.ext;
// 路径里允许空格、中文与各种标点(如「~/Downloads/MiniPost 界面重设计 — 组件 · 标题栏.html」)。参数 code 是行内代码原文。
func looksLikePath(code string) bool {
	if code == "" || len(code) > 240 || strings.Contains(code, "://") || strings.HasPrefix(code, "-") {
		return false
	}
	if strings.ContainsAny(code, "<>\n\r\t`") {
		return false
	}
	p, _, _ := splitPathRef(code)
	if i := strings.IndexByte(p, '#'); i >= 0 {
		p = p[:i]
	}
	p = strings.TrimSpace(p)
	if p == "" || p == "/" || p == "." || p == ".." {
		return false
	}
	return strings.Contains(p, "/") || pathExtRe.MatchString(p)
}

// pathRefOf 返回行内代码 code 的探测结果;没查过就排一个后台查询并先当作「无」。
func (a *App) pathRefOf(code string) pathRefState {
	if a.pathRefs.m == nil || len(a.pathRefs.m) > pathRefCap {
		a.pathRefs.m = map[string]pathRefState{}
	}
	if st, ok := a.pathRefs.m[code]; ok {
		if st == pathRefPending {
			return pathRefNone
		}
		return st
	}
	if !looksLikePath(code) {
		a.pathRefs.m[code] = pathRefNone
		return pathRefNone
	}
	a.pathRefs.m[code] = pathRefPending
	p, _, _ := splitPathRef(code)
	if i := strings.IndexByte(p, '#'); i >= 0 {
		p = p[:i]
	}
	abs, ok := a.resolveUserPath(p)
	a.bg("path-ref", func() func() {
		st := pathRefNone
		if ok {
			done := diag.SpanSlow("app", "探测会话里的路径", 100*time.Millisecond)
			if fi, err := os.Stat(abs); err == nil {
				st = pathRefFile
				if fi.IsDir() {
					st = pathRefDir
				}
			}
			done()
		}
		return func() {
			a.pathRefs.m[code] = st
			if st != pathRefNone {
				a.pathRefs.ver++
				a.frameSoon()
			}
		}
	})
	return pathRefNone
}

// mygoPatched 报告当前二进制是不是链到了打过补丁的 MyGo(scripts/mygo-overlay.sh 用 -modfile 把依赖 replace 到 target/mygo-patched,
// 构建信息里能看到)。补丁让路径链接的 LinkIcon 对象成为整枚胶囊;没打补丁时只画图标,链接文字照常显示。
var mygoPatched = sync.OnceValue(func() bool {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}
	for _, d := range bi.Deps {
		if d.Path == "github.com/egoist/mygo" && d.Replace != nil && strings.Contains(d.Replace.Path, "mygo-patched") {
			return true
		}
	}
	return false
})

// linkIcon 是 markdown.Options.LinkIcon:路径链接。打了补丁时它画整枚胶囊(参照会话输入框里的胶囊:淡灰底 + 细边 + 圆角,
// 左边是按文件名 / 扩展名 / 文件夹名取的彩色类型图标 Material Icon Theme(matico 包),右边是路径文字);
// 没打补丁时只在链接文字前画一枚图标。参数 dest 是链接目的地,size 是周围正文的字号;别的链接不画。
func (a *App) linkIcon(dest string, size float32) (markdown.InlineObject, bool) {
	dest = strings.TrimPrefix(strings.TrimSpace(dest), pathLinkScheme)
	st, ok := a.pathRefs.m[dest]
	if !ok || (st != pathRefFile && st != pathRefDir) {
		return markdown.InlineObject{}, false
	}
	p, _, _ := splitPathRef(dest)
	if i := strings.IndexByte(p, '#'); i >= 0 {
		p = p[:i]
	}
	var svg *ui.SVG
	if st == pathRefDir {
		svg = matico.Folder(p, false)
	} else {
		svg = matico.File(p)
	}
	if svg == nil {
		return markdown.InlineObject{}, false
	}
	cs := size * 11.5 / theme.FsRead // 行内代码的字号(mdTheme 的 CodeSize 是它与正文的比)
	ic := cs * 1.15
	c := a.buildCtx
	if !mygoPatched() || c == nil {
		return markdown.InlineObject{
			Width: ic + 6, Height: cs * .98, Depth: cs * .34,
			Draw: func(pt *ui.Painter, r ui.Rect) {
				pt.Image(svg, ui.Rect{X: r.X + 2, Y: r.Y + (r.H-ic)/2, W: ic, H: ic}, ui.Contain)
			},
		}, true
	}
	font := theme.CodeFont()
	span := func(s string) float32 {
		w, _ := c.MeasureText(0, ui.Span{Text: s, Font: font, Size: cs})
		return w
	}
	label := clipMiddle(dest, pathChipMaxText, span)
	_, lineH := c.MeasureText(0, ui.Span{Text: "Ag", Font: font, Size: cs}) // 这个字体这个字号的真实行高:文字按它在胶囊里垂直居中
	const padL, gap, padR, margin = 6, 4, 7, 1.5
	w := padL + ic + gap + span(label) + padR
	h := cs * 1.5 // 比行内代码底色(1.32 倍)略高一点,算上细边
	// 胶囊中线放在基线上方约 0.31 倍正文字号处(介于小写字母与大写字母的视觉中心之间),与周围文字对齐
	up := size * .31
	// 配色跟主题强调色走,与旁边的行内代码同一族:淡强调色底 + 稍深一点的强调色细边 + 强调色文字
	t := a.tk
	ink, fill := a.inlineCodeColors()
	line := t.Ac.Alpha(.26)
	return markdown.InlineObject{
		Width: w + 2*margin, Height: h/2 + up, Depth: h/2 - up,
		Draw: func(pt *ui.Painter, r ui.Rect) {
			box := ui.Rect{X: r.X + margin, Y: r.Y, W: w, H: r.H}
			pt.Fill(box, fill, 4)
			pt.Stroke(box, line, 4, 1)
			cy := box.Y + box.H/2 // 图标与文字共用这条水平中线
			pt.Image(svg, ui.Rect{X: box.X + padL, Y: cy - ic/2, W: ic, H: ic}, ui.Contain)
			pt.RichText(box.X+padL+ic+gap, cy-lineH/2, 0, ui.Span{Text: label, Font: font, Size: cs, Color: ink})
		},
	}, true
}

// openDirRef 处理「点了一个文件夹路径」:在左栏文件树里展开并选中(项目内),项目外的在访达里显示。
// 返回是否已处理(不是文件夹返回 false,由调用方走打开文件的路径)。参数 ref 是路径原文。
func (a *App) openDirRef(ref string) bool {
	p, _, _ := splitPathRef(ref)
	abs, ok := a.resolveUserPath(p)
	if !ok {
		return false
	}
	fi, err := os.Stat(abs)
	if err != nil || !fi.IsDir() {
		return false
	}
	var rel string
	var in bool
	if a.multiRoot() {
		rel, in = a.wsRelMulti(abs)
	} else {
		rel, in = a.underProj(abs)
	}
	if in && a.ide != nil && a.ide.fs != nil {
		a.screen = ScreenWork
		a.nav, a.sideOpen = "files", true
		if rel != "" { // 空串 = 项目根本身,只切到「文件」
			parts := strings.Split(rel, "/")
			for i := 1; i <= len(parts); i++ {
				a.ide.open[strings.Join(parts[:i], "/")] = true
			}
			a.ide.sel = rel
			a.ide.revealTo = rel
		}
		log.Info("会话里的文件夹路径:在文件树定位", "path", rel)
		return true
	}
	a.revealInFinder(abs)
	return true
}

// padCodeLinks 在 padCode 的基础上,把「本机存在的路径」行内代码改写成指向自己的链接 [`路径`](<路径>):
// 这样框架会给链接前面画文件 / 文件夹图标(LinkIcon),点击走 openMdLink。框架自带的 CodeLink 保留代码原样,
// 画不了图标,所以改在源码这一层做;源码前面的部分变了,框架会整篇重解析,流式输出不受影响。
// 已经在链接里的代码([`x`](..))、围栏代码块里的、还没查出来的路径都不动(查出来后下一帧才变成链接)。
func (a *App) padCodeLinks(src string) string {
	src = mdcompat.Prepare(src)
	const thin = "\u2009"
	var b strings.Builder
	fenced := false
	for _, line := range strings.SplitAfter(src, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " "), "```") {
			fenced = !fenced
			b.WriteString(line)
			continue
		}
		if fenced || !strings.Contains(line, "`") {
			b.WriteString(line)
			continue
		}
		rs := []rune(line)
		for i := 0; i < len(rs); i++ {
			if rs[i] != '`' {
				b.WriteRune(rs[i])
				continue
			}
			// 找同一行里配对的结束反引号(只处理单反引号;没配上的原样写出)
			end := -1
			for k := i + 1; k < len(rs); k++ {
				if rs[k] == '`' {
					end = k
					break
				}
			}
			if end < 0 {
				b.WriteRune('`')
				continue
			}
			code := string(rs[i+1 : end])
			inLink := (i > 0 && rs[i-1] == '[') || (end+2 < len(rs) && rs[end+1] == ']' && rs[end+2] == '(')
			if !inLink {
				if st := a.pathRefOf(strings.TrimSpace(code)); st == pathRefFile || st == pathRefDir {
					b.WriteString("[`" + thin + code + thin + "`](<" + pathLinkScheme + strings.TrimSpace(code) + "> \"点击打开\")")
					i = end
					continue
				}
			}
			b.WriteString("`" + thin + code + thin + "`")
			i = end
		}
	}
	return b.String()
}

// openPathSmart 打开会话里点中的路径(文件):图片开灯箱;PDF / Office / 压缩包 / 音视频等不是文本的文件交给系统默认程序;
// 文本文件在工作区编辑器里打开(带 :行:列 就跳过去)。参数 ref 是路径原文。返回是否已处理(路径不存在返回 false)。
func (a *App) openPathSmart(ref string) bool {
	p, _, _ := splitPathRef(ref)
	if i := strings.IndexByte(p, '#'); i >= 0 {
		p = p[:i]
	}
	abs, ok := a.resolveUserPath(p)
	if !ok {
		return a.openPathRef(ref)
	}
	fi, err := os.Stat(abs)
	if err != nil || fi.IsDir() {
		return a.openPathRef(ref)
	}
	if isImageExt(strings.ToLower(filepath.Ext(abs))) {
		a.openImageLightboxPath(abs)
		return true
	}
	// 网页(.html / .htm)交给系统默认浏览器;PDF / Office / 压缩包 / 音视频等非文本文件交给系统默认程序
	if ext := strings.ToLower(filepath.Ext(abs)); ext == ".html" || ext == ".htm" || ext == ".xhtml" || !isTextFile(abs) {
		log.Info("会话里的路径:交给系统默认程序打开", "path", abs)
		if testing.Testing() { // 单测里不要真的拉起外部程序
			return true
		}
		if err := exec.Command("open", abs).Start(); err != nil {
			a.toast = "无法打开:" + err.Error()
		}
		return true
	}
	return a.openPathRef(ref)
}

// binaryExts 是一定不是文本、交给系统打开的常见扩展名。
var binaryExts = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true, ".key": true, ".pages": true, ".numbers": true,
	".zip": true, ".gz": true, ".tgz": true, ".tar": true, ".7z": true, ".rar": true, ".dmg": true, ".pkg": true, ".app": true,
	".mp4": true, ".mov": true, ".mkv": true, ".avi": true, ".mp3": true, ".wav": true, ".flac": true, ".m4a": true, ".aac": true,
	".psd": true, ".sketch": true, ".fig": true, ".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".sqlite": true, ".db": true,
}

// isTextFile 判断文件能不能当文本在编辑器里打开:已知二进制扩展名直接否;其余读开头 8KB,含 NUL 或不是合法 UTF-8 就否。
func isTextFile(abs string) bool {
	if binaryExts[strings.ToLower(filepath.Ext(abs))] {
		return false
	}
	f, err := os.Open(abs)
	if err != nil {
		return true // 读不了就交给编辑器路径去报具体原因
	}
	defer f.Close()
	buf := make([]byte, 8192)
	n, _ := f.Read(buf)
	buf = buf[:n]
	if bytes.IndexByte(buf, 0) >= 0 {
		return false
	}
	if len(buf) == cap(buf) { // 可能切在一个多字节字符中间:去掉最后几个字节再判
		buf = buf[:len(buf)-3]
	}
	return utf8.Valid(buf)
}
