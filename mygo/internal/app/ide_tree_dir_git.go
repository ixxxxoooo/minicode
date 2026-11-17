package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// dirGitAgg 是某目录下后代改动的聚合(由文件路径向上累计)。
type dirGitAgg struct {
	worst     byte
	modified  int // M / R
	added     int // A
	deleted   int // D
	untracked int // U
	conflict  int // C
}

func gitStatusRank(st byte) int {
	switch st {
	case vcs.StatusConflict:
		return 5
	case 'D':
		return 4
	case 'M', 'R':
		return 3
	case 'A':
		return 2
	case 'U':
		return 1
	default:
		return 0
	}
}

func bumpDirGit(m map[string]dirGitAgg, filePath string, st byte) {
	if st == 0 {
		return
	}
	dir := parentDir(filePath)
	for dir != "." && dir != "" {
		a := m[dir]
		switch st {
		case vcs.StatusConflict:
			a.conflict++
		case 'D':
			a.deleted++
		case 'M', 'R':
			a.modified++
		case 'A':
			a.added++
		case 'U':
			a.untracked++
		}
		if gitStatusRank(st) > gitStatusRank(a.worst) {
			a.worst = st
		}
		m[dir] = a
		dir = parentDir(dir)
	}
}

// rebuildTreeDirGit 从文件级 git 状态表向上聚合到各级目录(多根路径已带前缀)。
func (a *App) rebuildTreeDirGit(files map[string]string) {
	end := diag.SpanSlow("ide", "目录 Git 聚合", 50*time.Millisecond, "files", len(files))
	defer end()
	m := map[string]dirGitAgg{}
	for p, letter := range files {
		if letter == "" {
			continue
		}
		bumpDirGit(m, p, letter[0])
	}
	a.ide.dirGit = m
}

func (agg dirGitAgg) hasChange() bool {
	return agg.worst != 0
}

func (agg dirGitAgg) changeCount() int {
	return agg.modified + agg.added + agg.deleted + agg.conflict
}

func (agg dirGitAgg) tooltip() string {
	var parts []string
	if n := agg.changeCount(); n > 0 {
		parts = append(parts, fmt.Sprintf("内含 %d 个已修改", n))
	}
	if agg.untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d 个未跟踪", agg.untracked))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " · ")
}

func treeGitColor(t *theme.Tokens, st byte) ui.Color {
	switch st {
	case 'A', 'U':
		return t.Ok
	case 'D', vcs.StatusConflict:
		return t.Err
	case 'M', 'R':
		return t.Warn
	default:
		return t.Tx2
	}
}

// treeDirMuted 目录无 Git 后代且为常见产物 / 忽略目录时灰显(与资源管理器一致)。
func treeDirMuted(name string, dir string, agg dirGitAgg) bool {
	if agg.hasChange() {
		return false
	}
	switch name {
	case "node_modules", "target", "dist", "build", "vendor", ".next", "out", ".gradle", "coverage":
		return true
	}
	if strings.HasPrefix(name, ".") && name != ".github" && name != ".vscode" && name != ".cursor" {
		return true
	}
	_ = dir
	return false
}
