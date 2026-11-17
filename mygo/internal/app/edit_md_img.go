package app

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

const pMdAssets = "editor.mdAssets" // 拖入 / 粘贴图片的默认相对目录

var mdImgLog = diag.L("edit.mdimg")

// mdAssetDir 返回当前项目里存放 Markdown 图片的相对目录。
func (a *App) mdAssetDir() string {
	d := strings.Trim(a.prefs.Str(pMdAssets, "docs/assets"), "/")
	if d == "" {
		d = "docs/assets"
	}
	return d
}

// saveImageForMD 把绝对路径的图片复制进工作区并返回相对 Markdown 路径。
func (a *App) saveImageForMD(abs string) (string, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(abs))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
	default:
		return "", os.ErrInvalid
	}
	dir := a.mdAssetDir()
	name := filepath.Base(abs)
	dest := path.Join(dir, name)
	outAbs := a.ide.fs.Abs(dest)
	if err := os.MkdirAll(filepath.Dir(outAbs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(outAbs, data, 0o644); err != nil {
		return "", err
	}
	a.ide.invalidate()
	mdImgLog.Info("保存图片", "dest", dest)
	return dest, nil
}

// insertMdImage 在光标处插入 Markdown 图片语法。
func (a *App) insertMdImage(tb *ideTab, rel string, alt string) {
	if tb == nil || tb.buf == nil || tb.st == nil {
		return
	}
	if alt == "" {
		alt = path.Base(rel)
	}
	snip := "![" + alt + "](" + rel + ")\n"
	off := tb.st.Cursor()
	tb.buf.Edit(editor.Edit{Start: off, End: off, Text: snip})
	tb.st.Select(off+len(snip), off+len(snip))
}

// handleDroppedImages 处理访达拖入的图片(对话栏与 Markdown 编辑器)。
func (a *App) handleDroppedImages(paths []string) {
	var rest []string
	for _, abs := range paths {
		ext := strings.ToLower(filepath.Ext(abs))
		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" || ext == ".svg" {
			tb := a.ide.cur()
			if tb != nil && isMarkdownFile(tb.path) && !a.mdSource(tb) {
				rel, err := a.saveImageForMD(abs) // 只有插进 Markdown 文档才复制进工作区
				if err != nil {
					a.toast = "无法保存图片:" + err.Error()
					mdImgLog.Warn("保存图片失败", "src", abs, "err", err)
					continue
				}
				a.insertMdImage(tb, rel, "")
				a.toast = "已插入图片 " + rel
			} else {
				a.addDraftImage(abs) // 对话附件:直接用原文件,不往项目里复制
			}
			continue
		}
		rest = append(rest, abs)
	}
	if len(rest) > 0 {
		a.dropPaths(rest)
	}
}
