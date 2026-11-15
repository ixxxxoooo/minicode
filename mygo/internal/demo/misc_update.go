// 演示数据索引(应用更新)
//
//   - UpdatePhases:命令面板「演示:更新阶段…」与 -ed upd:<phase> 截图。
//   - 展示:侧栏胶囊(若接线)、状态栏更新项、设置 → 关于、弹层 upd。
package demo

// UpdatePhases 全部可切换的演示阶段(id, 中文说明, 下载百分比)。
func UpdatePhases() []struct {
	ID, Label string
	Pct       int
} {
	return []struct {
		ID, Label string
		Pct       int
	}{
		{"check", "检查中…", 0},
		{"latest", "已是最新", 0},
		{"found", "发现新版本", 0},
		{"download", "下载中", 42},
		{"download", "下载中 78%", 78},
		{"verify", "校验中…", 100},
		{"unpack", "解包中…", 100},
		{"ready", "可重启", 100},
		{"fail", "下载失败", 0},
		{"neterr", "网络错误", 0},
	}
}
