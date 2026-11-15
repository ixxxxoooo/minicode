package kit

// 设计稿 v3.1(docs/design/js/ws.js)新增的图标;icons_gen.go 由脚本生成,这里手工补充,避免被覆盖。
func init() {
	for k, v := range map[string]string{
		"arrowL": `<path d="M19 12H5M11 6l-6 6 6 6"/>`,
		"merge":  `<circle cx="6" cy="5.5" r="2"/><circle cx="6" cy="18.5" r="2"/><circle cx="18" cy="11" r="2"/><path d="M6 7.5v9M18 13c0 3-4 3-8 3"/>`,
		"bulb":   `<path d="M9 18h6M10 21h4M12 3a6 6 0 0 0-3.5 10.9c.6.5 1 1.2 1 2.1h5c0-.9.4-1.6 1-2.1A6 6 0 0 0 12 3z"/>`,
		"unfold": `<path d="M8 9l4-4 4 4M8 15l4 4 4-4"/>`,
		"quote":  `<path d="M6 17c0-4 1-6 4-7M14 17c0-4 1-6 4-7"/>`,
	} {
		iconPaths[k] = v
	}
}
