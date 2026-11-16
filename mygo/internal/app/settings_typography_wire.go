package app

var settingsTypographyPatched bool

func ensureTypographySettingsRows() {
	if settingsTypographyPatched {
		return
	}
	settingsTypographyPatched = true
	patchAppearanceTypography()
}

// patchAppearanceTypography 在同事合并 typography 分类前,把排版相关行挂到「外观」。
func patchAppearanceTypography() {
	for ti := range settingsTabs {
		if settingsTabs[ti].id != "appearance" {
			continue
		}
		// 扩展「字号与大小」
		for gi := range settingsTabs[ti].groups {
			if settingsTabs[ti].groups[gi].title != "字号与大小" {
				continue
			}
			extra := []setRow{
				rSeg(pFontPreset, "字号预设", "同时调整界面大小与会话正文字号;滑块偏离后自动显示「自定义」", 1, "紧凑", "标准", "舒适", "大字", "自定义"),
				rSeg(pLineHeight, "正文行距", "会话流、用户消息与输入框的行距;不影响编辑器与终端", 1, "紧凑", "标准", "宽松"),
			}
			settingsTabs[ti].groups[gi].rows = append(settingsTabs[ti].groups[gi].rows, extra...)
		}
		// 替换 Agent 会话里的旧三档内容宽度,并加设置页列宽
		for gi := range settingsTabs[ti].groups {
			if settingsTabs[ti].groups[gi].title != "Agent 会话" {
				continue
			}
			var rows []setRow
			for _, r := range settingsTabs[ti].groups[gi].rows {
				if r.key == pWidthLegacy || r.key == "appearance.width" {
					continue
				}
				rows = append(rows, r)
			}
			rows = append(rows,
				rSeg(pFlowWidth, "会话内容宽度", "时间线、排队卡、任务坞与输入框同宽;窄屏自动收窄", 2, "窄", "紧凑", "标准", "宽", "特宽", "占满"),
				rSeg(pSetWidth, "设置页内容宽度", "设置主栏内容列;窗口较窄时自动铺满", 1, "窄", "标准", "宽", "特宽", "占满"),
			)
			settingsTabs[ti].groups[gi].rows = rows
		}
		break
	}
}
