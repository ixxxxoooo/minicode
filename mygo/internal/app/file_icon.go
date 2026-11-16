package app

import (
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
)

// 文件类型图标(对标 Cursor / VS Code 的图标主题):已知语言用「语言缩写 + 主题色」的小圆角徽标
// (js 黄 / ts 蓝 / py 绿 …),其余退回线条图标。设计稿:docs/design/js/ui.js 的 U.fi,两边的扩展名表保持一致。
// 颜色取主题令牌,换主题自动跟着变;设置 → 外观「彩色文件图标」关闭时一律是灰色线条图标。

// fileKindTone 是徽标取哪一种主题色。
type fileKindTone int

const (
	toneOk fileKindTone = iota
	toneErr
	toneWarn
	toneInfo
	toneVio
	toneMuted
)

type fileKind struct {
	label string
	tone  fileKindTone
}

// fileKinds 按小写扩展名(不带点)查徽标。
var fileKinds = map[string]fileKind{
	"js": {"JS", toneWarn}, "mjs": {"JS", toneWarn}, "cjs": {"JS", toneWarn}, "jsx": {"JSX", toneWarn},
	"ts": {"TS", toneInfo}, "tsx": {"TSX", toneInfo}, "go": {"GO", toneInfo}, "rs": {"RS", toneWarn}, "py": {"PY", toneOk},
	"swift": {"SW", toneErr}, "java": {"JV", toneErr}, "kt": {"KT", toneVio},
	"c": {"C", toneInfo}, "h": {"H", toneInfo}, "cpp": {"C++", toneInfo}, "hpp": {"H++", toneInfo},
	"rb": {"RB", toneErr}, "php": {"PHP", toneVio}, "lua": {"LU", toneInfo},
	"sh": {"SH", toneOk}, "zsh": {"SH", toneOk}, "bash": {"SH", toneOk},
	"html": {"HT", toneErr}, "css": {"CS", toneInfo}, "scss": {"SC", toneVio}, "vue": {"VU", toneOk},
	"md": {"MD", toneMuted}, "mdx": {"MD", toneMuted}, "json": {"{}", toneWarn},
	"toml": {"TM", toneMuted}, "yaml": {"YM", toneErr}, "yml": {"YM", toneErr},
	"sql": {"SQ", toneVio}, "csv": {"CS", toneOk}, "xml": {"XM", toneWarn},
}

// fileKindOf 返回文件名对应的徽标;没有扩展名(Makefile、.gitignore)或不认识的返回 false。
func fileKindOf(name string) (fileKind, bool) {
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(name)), ".")
	if ext == "" {
		return fileKind{}, false
	}
	k, ok := fileKinds[ext]
	return k, ok
}

func (a *App) fileKindColor(tone fileKindTone) ui.Color {
	t := a.tk
	switch tone {
	case toneOk:
		return t.Ok
	case toneErr:
		return t.Err
	case toneWarn:
		return t.Warn
	case toneInfo:
		return t.Info
	case toneVio:
		return t.Vio
	}
	return t.Tx2
}

// fileBadge 画语言徽标:14×14 圆角底(主题色 18% 透明度)+ 居中的缩写字母。
func (a *App) fileBadge(c *ui.Context, k fileKind) ui.Element {
	col := a.fileKindColor(k.tone)
	size := float32(8)
	if len([]rune(k.label)) >= 3 {
		size = 6.5
	}
	return ui.Row(c).Size(14, 14).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Radius(3).
		Background(col.Alpha(.18)).Children(func() {
		ui.Text(c, k.label).FontSize(size).FontWeight(700).LineHeight(1).TextColor(col).NoWrap()
	})
}
