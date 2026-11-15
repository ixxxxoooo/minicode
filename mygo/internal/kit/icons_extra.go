package kit

// 设计稿图标表之外补充的图标(同样 24×24 画布、描边 1.7),init 时并入 iconPaths。
func init() {
	for k, v := range map[string]string{
		"min":       `<path d="M9 4v5H4M15 4v5h5M9 20v-5H4M15 20v-5h5"/>`,
		"dockR":     `<rect x="4" y="5" width="16" height="14" rx="2"/><path d="M14 5v14"/>`,
		"dockB":     `<rect x="4" y="5" width="16" height="14" rx="2"/><path d="M4 14h16"/>`,
		"download":  `<path d="M12 4v11M7.5 11 12 15.5 16.5 11M5 19.5h14"/>`,
		"package":   `<path d="M4 8 12 4l8 4v8l-8 4-8-4zM4 8l8 4 8-4M12 12v8"/>`,
		"caseSens":  `<path d="M3.5 17 8 6l4.5 11M5 13.5h6M15 17v-5.5a2.5 2.5 0 0 1 5 0V17M15 15h5"/>`,
		"regex":     `<path d="M6 17h.01M12 5v8M8.5 7l7 4M15.5 7l-7 4"/>`,
		"wholeWord": `<path d="M3 17h18M7 13V7h3a2 2 0 0 1 0 4H7M14 13V7"/>`,
	} {
		if _, ok := iconPaths[k]; !ok {
			iconPaths[k] = v
		}
	}
}
