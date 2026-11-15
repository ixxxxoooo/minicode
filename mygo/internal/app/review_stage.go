package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// reviewStageAcceptedPaths 返回可暂存的文件路径及块数(已接受,或已查看且非撤销的块)。
func (a *App) reviewStageAcceptedPaths(files []chFile) (paths []string, nHunks int) {
	a.rv.init()
	for _, f := range files {
		hasOK := false
		for i := range f.blocks {
			k := hunkKey(f.path, i)
			st := a.rv.hunk[k]
			if st == "ok" {
				hasOK = true
				nHunks++
				continue
			}
			// 已查看且该块未决:不计入;已查看且全部为 ok/rev 时把仍为空的视为「已查看」保留现状(不单独暂存该块)
			if a.rv.viewed[f.path] && st == "" && a.fileDecided(f) && hasOK {
				nHunks++
			}
		}
		if hasOK {
			paths = append(paths, f.path)
		}
	}
	return paths, nHunks
}

func (a *App) reviewStageAcceptedReason(files []chFile) (n int, why string) {
	paths, n := a.reviewStageAcceptedPaths(files)
	n = len(paths)
	if n == 0 {
		return 0, "没有已接受或已查看并完成处理的改动块"
	}
	return n, ""
}

// stageAcceptedContent 把审查中已接受的块合成暂存区内容(未接受的块保持 HEAD 侧)。
func (a *App) stageAcceptedContent(f chFile) (string, bool) {
	rel := a.wsRel(f.path)
	repo, rp := a.repoAt(a.treePath(rel))
	if repo == nil {
		return "", false
	}
	head, ok := repo.Show("HEAD", rp)
	if !ok {
		head, _ = a.ide.fs.Read(a.treePath(rel))
	}
	cur := head
	for i, d := range f.blocks {
		k := hunkKey(f.path, i)
		st := a.rv.hunk[k]
		eligible := st == "ok"
		if !eligible {
			continue
		}
		if d.New != "" && strings.Contains(cur, d.Old) {
			cur = strings.Replace(cur, d.Old, d.New, 1)
		} else if d.New != "" {
			cur = d.New
		}
	}
	return cur, true
}

// reviewStageAccepted 暂存已接受的改动块。
func (a *App) reviewStageAccepted() {
	files := a.changedFiles()
	paths, _ := a.reviewStageAcceptedPaths(files)
	if len(paths) == 0 {
		a.toast = "没有可暂存的已接受改动"
		return
	}
	type job struct {
		rel, content string
	}
	var jobs []job
	for _, f := range files {
		okFile := false
		for _, p := range paths {
			if p == f.path {
				okFile = true
				break
			}
		}
		if !okFile {
			continue
		}
		content, ok := a.stageAcceptedContent(f)
		if !ok {
			a.toast = "无法读取 " + f.path
			return
		}
		jobs = append(jobs, job{rel: a.wsRel(f.path), content: content})
	}
	type stagedBefore struct {
		repo    vcs.Repo
		relPath string
		content string
	}
	var prev []stagedBefore
	var partialMsg string
	var okN int
	a.scmAct("暂存已接受的改动", func(_ vcs.Repo) error {
		var firstErr error
		fail := 0
		okN = 0
		for _, j := range jobs {
			repo, rp := a.repoAt(a.treePath(j.rel))
			if repo == nil {
				fail++
				if firstErr == nil {
					firstErr = fmt.Errorf("无法找到仓库:%s", j.rel)
				}
				continue
			}
			if s, ok := repo.Index(rp); ok {
				prev = append(prev, stagedBefore{repo: repo, relPath: rp, content: s})
			} else {
				prev = append(prev, stagedBefore{repo: repo, relPath: rp, content: ""})
			}
			if err := repo.StageContent(rp, j.content); err != nil {
				fail++
				if firstErr == nil {
					firstErr = err
				}
				diag.L("review").Warn("暂存块失败", "path", rp, "err", err)
				continue
			}
			okN++
		}
		if fail > 0 && okN > 0 {
			partialMsg = fmt.Sprintf("%d 个成功,%d 个失败", okN, fail)
			return nil
		}
		if fail > 0 {
			if firstErr != nil {
				return firstErr
			}
			return fmt.Errorf("暂存失败")
		}
		return nil
	}, func() {
		staged := len(jobs)
		if partialMsg != "" {
			staged = okN
			a.toast = "部分暂存:" + partialMsg
		}
		a.rvStageUndo.Begin("已暂存 "+itoa(staged)+" 个文件", func() {
			a.scmAct("撤销暂存", func(_ vcs.Repo) error {
				var err error
				for _, p := range prev {
					if p.repo == nil {
						continue
					}
					if p.content == "" {
						if e := p.repo.Unstage(p.relPath); e != nil && err == nil {
							err = e
						}
						continue
					}
					if e := p.repo.StageContent(p.relPath, p.content); e != nil && err == nil {
						err = e
					}
				}
				return err
			})
		})
		if partialMsg == "" {
			a.toast = "已暂存 " + itoa(staged) + " 个文件"
		}
		diag.L("review").Info("暂存已接受的改动", "files", staged, "partial", partialMsg != "")
	})
}

// reviewStageUndoBar 审查页脚撤销条。
func (a *App) reviewStageUndoBar(c *ui.Context) {
	kit.UndoBar(c, &a.rvStageUndo)
}
