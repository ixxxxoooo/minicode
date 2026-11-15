package app

import (
	"path/filepath"

	"github.com/ixxxxoooo/minicode/internal/customize"
)

func (a *App) cuSaveSkillRename(st *customize.Store, it *customize.Item) {
	s := &a.cu
	oldPath := it.Path
	if err := st.Save(customize.Skills, it, s.draft); err != nil {
		s.err = err.Error()
		return
	}
	if ds := a.cuDisk(); ds != nil {
		if err := ds.Persist(customize.Skills, it, s.draft); err != nil {
			s.err = "保存失败: " + err.Error()
			log.Warn("Skill 重命名失败", "name", it.Name, "err", err)
			return
		}
	}
	if oldPath != "" && it.Path != "" && oldPath != it.Path {
		oldAbs, newAbs := oldPath, it.Path
		if !filepath.IsAbs(oldAbs) && a.proj != "" {
			oldAbs = filepath.Join(a.proj, oldPath)
		}
		if !filepath.IsAbs(newAbs) && a.proj != "" {
			newAbs = filepath.Join(a.proj, it.Path)
		}
		a.renameTabs(filepath.Join(oldAbs, "SKILL.md"), filepath.Join(newAbs, "SKILL.md"))
	}
	s.err, s.dOf = "", nil
	a.toast = "已保存 Skill「" + it.Name + "」"
	log.Info("保存 Skill", "name", it.Name, "path", it.Path)
}
