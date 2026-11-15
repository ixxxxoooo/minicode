// 默认模型 / 思考强度:claude.model|effort 为唯一来源,兼容旧的 models.model|effort 索引键。
package app

import "github.com/ixxxxoooo/minicode/internal/core"

// 旧版设置键(仅迁移与对外导出兼容);界面与 FillClaudeConfig 只读写 prefModel / prefEffort。
const (
	legacyModelIx = "models.model"
	legacyEffort  = "models.effort"
)

// syncModelEffortPrefs 双向对齐旧键与新键,对齐后删掉旧键避免再次分叉。
func (a *App) syncModelEffortPrefs() {
	if a == nil || a.prefs == nil {
		return
	}
	// 模型:旧索引 → 新字符串 id
	if a.prefs.Has(legacyModelIx) {
		ix := a.prefs.Int(legacyModelIx, -1)
		if ix >= 0 && ix < len(defaultModelIDs) {
			if !a.prefs.Has(prefModel) {
				a.prefs.Set(prefModel, defaultModelIDs[ix])
			}
		}
		a.prefs.Reset(legacyModelIx)
	}
	// 若只有新键,补写旧索引供外部脚本读(马上再删,保持磁盘上只剩新键)
	if id := a.prefs.Str(prefModel, ""); id != "" {
		if ix := modelChipIndex(id); ix >= 0 && ix < len(defaultModelIDs) {
			// 旧键已废弃,不再写回
			_ = ix
		}
	}
	if a.prefs.Has(legacyEffort) {
		if !a.prefs.Has(prefEffort) {
			a.prefs.Set(prefEffort, a.prefs.Int(legacyEffort, int(core.EffortMedium)))
		}
		a.prefs.Reset(legacyEffort)
	}
}
