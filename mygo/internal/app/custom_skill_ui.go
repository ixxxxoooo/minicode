package app

import (
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const skillPreviewMax = 256 << 10 // 256 KiB

func humanSkillSize(n int64) string {
	if n < 1024 {
		return itoa(int(n)) + " B"
	}
	if n < 1024*1024 {
		return itoa(int(n/1024)) + " KB"
	}
	return itoa(int(n/(1024*1024))) + " MB"
}

// cuEnsureSkillFile 选中 Skill 时默认高亮 SKILL.md。
func (a *App) cuEnsureSkillFile(it *customize.Item) {
	if it == nil || a.cu.skillOf == it.Path && a.cu.skillFile != "" {
		return
	}
	a.cu.skillOf = it.Path
	pick := ""
	for _, f := range it.Files {
		if f.Name == "SKILL.md" {
			pick = f.Name
			break
		}
	}
	if pick == "" && len(it.Files) > 0 {
		pick = it.Files[0].Name
	}
	if pick != "" {
		a.cuPickSkillFile(it, pick)
	}
}

func (a *App) cuPickSkillFile(it *customize.Item, rel string) {
	if it == nil || rel == "" {
		return
	}
	a.cu.skillFile, a.cu.skillPrev, a.cu.skillPrevErr = rel, "", ""
	a.cu.skillLoad++
	load := a.cu.skillLoad
	a.bg("skill.preview", func() func() {
		defer diag.Recover("skill.preview")
		text, errText := a.cuReadSkillFile(it, rel)
		return func() {
			if load != a.cu.skillLoad || a.cu.skillFile != rel {
				return
			}
			a.cu.skillPrev, a.cu.skillPrevErr = text, errText
		}
	})
}

func (a *App) cuReadSkillFile(it *customize.Item, rel string) (string, string) {
	for _, f := range it.Files {
		if f.Name == rel && f.Bytes > skillPreviewMax {
			return "", "文件超过 256 KB,未加载预览"
		}
	}
	if strings.TrimSpace(it.Body) != "" && rel == "SKILL.md" && strings.HasPrefix(it.Path, ".") {
		return it.Body, ""
	}
	abs, ok := a.cuSkillAbs(it, rel)
	if !ok {
		return "", "找不到文件路径"
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return "", "无法读取:" + err.Error()
	}
	if fi.IsDir() {
		return "", "这是文件夹,请选择具体文件"
	}
	if fi.Size() > skillPreviewMax {
		return "", "文件超过 256 KB,未加载预览"
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", "无法读取:" + err.Error()
	}
	if !utf8.Valid(b) {
		return "", "二进制或非 UTF-8 文件,无法预览"
	}
	return string(b), ""
}

func (a *App) cuSkillAbs(it *customize.Item, rel string) (string, bool) {
	root, ok := a.resolveUserPath(it.Path)
	if !ok {
		return "", false
	}
	abs := filepath.Join(root, rel)
	if !strings.HasPrefix(abs, root+string(os.PathSeparator)) && abs != root {
		return "", false
	}
	return abs, true
}

func (a *App) cuSkillFilePath(it *customize.Item) string {
	if it == nil || a.cu.skillFile == "" {
		return it.Path
	}
	if abs, ok := a.cuSkillAbs(it, a.cu.skillFile); ok {
		return abs
	}
	return filepath.Join(it.Path, a.cu.skillFile)
}

func (a *App) cuSkillPreview(c *ui.Context) {
	t := a.tk
	if a.cu.skillPrevErr != "" {
		ui.Text(c, a.cu.skillPrevErr).FontSize(theme.FsCap).TextColor(t.Warn).Padding(8, 10)
		return
	}
	if a.cu.skillPrev == "" {
		ui.Text(c, "加载中…").FontSize(theme.FsCap).TextColor(t.Tx3).Padding(8, 10)
		return
	}
	ui.Scroll(c).MaxHeight(180).Children(func() {
		kit.Mono(c, a.cu.skillPrev, t.Tx2).Padding(8, 10).LineHeight(1.45)
	})
}

func (a *App) cuSkillFileList(c *ui.Context, it *customize.Item) {
	t := a.tk
	a.cuEnsureSkillFile(it)
	ui.Scroll(c).MaxHeight(160).Children(func() {
		for _, f := range it.Files {
			f := f
			sel := a.cu.skillFile == f.Name
			row := kit.RowEl(c, sel).Key("sk-f-" + f.Name).Height(t.Scheme.RowH).PaddingStart(8 + float32(f.Indent)*14).PaddingEnd(8).Gap(8).AlignItems(ui.Center).Radius(t.RSm()).MinWidth(0)
			row.OnClick(func() { a.cuPickSkillFile(it, f.Name) })
			row.Children(func() {
				icon := "fileText"
				if f.Dir {
					icon = "folder"
				}
				kit.Ic(c, icon, kit.S12).TextColor(t.Tx3).Shrink(0)
				kit.Mono(c, f.Name, map[bool]ui.Color{true: t.Tx1, false: t.Tx2}[sel]).Grow(1).MinWidth(0).SingleLine()
				if f.Bytes > 0 {
					ui.Text(c, humanSkillSize(f.Bytes)).FontSize(theme.FsMicro).TextColor(t.Tx4).Shrink(0)
				}
			})
		}
	})
}
