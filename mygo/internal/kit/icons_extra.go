package kit

// 设计稿图标表之外补充的图标(同样 24×24 画布、描边 1.7),init 时并入 iconPaths。
func init() {
	for k, v := range map[string]string{
		"merge":     `<circle cx="6" cy="5.5" r="2"/><circle cx="6" cy="18.5" r="2"/><circle cx="18" cy="11" r="2"/><path d="M6 7.5v9M18 13c0 3-4 3-8 3"/>`,
		"rows":      `<rect x="3.5" y="4.5" width="17" height="15" rx="2.5"/><path d="M3.5 12h17"/>`,
		"msg":       `<path d="M5 5h14a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1h-8l-4.5 3.5V16H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1zM8.5 9.5h7M8.5 12.5h4"/>`,
		"reply":     `<path d="M10 6 4.5 11.5 10 17M5 11.5h8a6 6 0 0 1 6 6V19"/>`,
		"warn":      `<path d="M12 4 3 19.5h18zM12 10v4.5M12 17h.01"/>`,
		"arrowDown": `<path d="M12 5v14M6 13l6 6 6-6"/>`,
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
