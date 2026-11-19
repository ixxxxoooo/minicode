// Package matico 提供按文件名 / 扩展名 / 文件夹名取彩色图标的能力,图标取自 Material Icon Theme(MIT,许可见 assets/LICENSE),
// 映射规则与 VS Code 的图标主题一致:文件先按文件名,再按最长的点扩展名;文件夹按文件夹名,分闭合与展开。
// 参照 BaoCode 的做法(它也捆绑了同一份图标)。
//
// @author ixxxxoooo
package matico

import (
	"embed"
	"encoding/json"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

//go:embed assets
var assets embed.FS

// manifest 是图标主题的映射表(assets/manifest.json)。
type manifest struct {
	File                string            `json:"file"`
	Folder              string            `json:"folder"`
	FolderExpanded      string            `json:"folderExpanded"`
	FileNames           map[string]string `json:"fileNames"`
	FileExtensions      map[string]string `json:"fileExtensions"`
	FolderNames         map[string]string `json:"folderNames"`
	FolderNamesExpanded map[string]string `json:"folderNamesExpanded"`
}

var (
	once sync.Once
	mf   manifest

	mu    sync.Mutex
	cache = map[string]*ui.SVG{}
)

func load() {
	once.Do(func() {
		raw, err := assets.ReadFile("assets/manifest.json")
		if err == nil {
			err = json.Unmarshal(raw, &mf)
		}
		if err != nil || mf.File == "" { // 映射读不出来:退回最基本的几个名字
			mf = manifest{File: "file", Folder: "folder", FolderExpanded: "folder-open"}
		}
	})
}

func baseName(p string) string {
	p = strings.TrimRight(p, "/\\")
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		p = p[i+1:]
	}
	return strings.ToLower(p)
}

// FileID 返回文件 path(或纯文件名)对应的图标 id。
func FileID(path string) string {
	load()
	name := baseName(path)
	if id, ok := mf.FileNames[name]; ok {
		return id
	}
	// 最长的点扩展名优先(a.test.ts 先试 test.ts 再试 ts);开头的点不算扩展名
	dot := -1
	if name != "" {
		dot = strings.IndexByte(name[1:], '.') + 1
		if dot == 0 {
			dot = -1
		}
	}
	for dot >= 0 && dot < len(name)-1 {
		if id, ok := mf.FileExtensions[name[dot+1:]]; ok {
			return id
		}
		next := strings.IndexByte(name[dot+1:], '.')
		if next < 0 {
			break
		}
		dot += next + 1
	}
	return mf.File
}

// FolderID 返回文件夹 path 对应的图标 id;open 为 true 取展开的样子。
func FolderID(path string, open bool) string {
	load()
	name := baseName(path)
	if open {
		if id, ok := mf.FolderNamesExpanded[name]; ok {
			return id
		}
		return mf.FolderExpanded
	}
	if id, ok := mf.FolderNames[name]; ok {
		return id
	}
	return mf.Folder
}

// svg 取(并缓存)图标 id 对应的 SVG;读不出来返回 nil。
func svg(id string) *ui.SVG {
	mu.Lock()
	defer mu.Unlock()
	if s, ok := cache[id]; ok {
		return s
	}
	var s *ui.SVG
	if raw, err := assets.ReadFile("assets/icons/" + id + ".svg"); err == nil {
		if p, err := ui.ParseSVG(raw); err == nil {
			s = p
		}
	}
	cache[id] = s
	return s
}

// File 返回文件的彩色图标(SVG 自带颜色,用 Painter.Image 画)。
func File(path string) *ui.SVG {
	s := svg(FileID(path))
	if s == nil {
		s = svg("file")
	}
	return s
}

// Folder 返回文件夹的彩色图标。
func Folder(path string, open bool) *ui.SVG {
	s := svg(FolderID(path, open))
	if s == nil {
		s = svg("folder")
	}
	return s
}
