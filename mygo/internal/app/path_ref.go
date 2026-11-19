package app

import (
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
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

// pathRefCap 是缓存条数上限,超了整表清空(会话很长时防止无限增长)。
const pathRefCap = 4000

// pathRefCache 缓存「路径文字 → 探测结果」,只在界面线程读写。
type pathRefCache struct {
	m map[string]pathRefState
	// ver 每次有新结果落地就加一,进 Markdown 选项缓存键,让段落按新答案重建。
	ver int
}

var (
	pathExtRe  = regexp.MustCompile(`^[\w.\-@+]+\.[A-Za-z0-9]{1,10}$`)
	pathCharRe = regexp.MustCompile(`^[\w.\-@+~/\\()\[\]:#]+$`)
)

// looksLikePath 判断一段行内代码是否「长得像路径」,没通过就不去查磁盘。
// 条件:无空白、不是网址、不太长,且含 / 或形如 name.ext。参数 code 是行内代码原文。
func looksLikePath(code string) bool {
	if code == "" || len(code) > 240 || strings.Contains(code, "://") || strings.HasPrefix(code, "-") {
		return false
	}
	if !pathCharRe.MatchString(code) {
		return false
	}
	p, _, _ := splitPathRef(code)
	if i := strings.IndexByte(p, '#'); i >= 0 {
		p = p[:i]
	}
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

// codeLink 是 markdown.Options.CodeLink:本机存在的路径 → 链接(目的地就是原文,点击交给 openMdLink)。
func (a *App) codeLink(code string) (string, bool) {
	code = strings.TrimSpace(code)
	if st := a.pathRefOf(code); st == pathRefFile || st == pathRefDir {
		return code, true
	}
	return "", false
}

// linkIcon 是 markdown.Options.LinkIcon:路径链接前面一枚文件 / 文件夹小图标(别的链接不画)。
// 参数 dest 是链接目的地,size 是周围文字大小。
func (a *App) linkIcon(dest string, size float32) (markdown.InlineObject, bool) {
	st, ok := a.pathRefs.m[strings.TrimSpace(dest)]
	if !ok || (st != pathRefFile && st != pathRefDir) {
		return markdown.InlineObject{}, false
	}
	name := "folder"
	if st == pathRefFile {
		p, _, _ := splitPathRef(dest)
		name = fileIcon(baseName(p))
	}
	svg := kit.IconSVG(name)
	sz := size * .95
	t := a.tk
	return markdown.InlineObject{
		Width: sz + 3, Height: sz*.82 + 1, Depth: sz * .18,
		Draw: func(p *ui.Painter, r ui.Rect) {
			p.Icon(svg, ui.Rect{X: r.X, Y: r.Y + (r.H-sz)/2, W: sz, H: sz}, t.Tx3)
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
