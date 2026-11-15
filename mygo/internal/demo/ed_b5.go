package demo

// B5EditorExtras 返回演示沙盒里便于测 B5 编辑能力的文件(外部修改横幅走 -ed extmod,内存 FS 不轮询)。
func B5EditorExtras() map[string]string {
	return map[string]string{
		"docs/extmod-hint.md": "# 外部修改演示\n演示模式不轮询磁盘;用 `-ed extmod` 或 `extmod:auto`(真实项目目录) 到达横幅 / 自动重载态。\n",
	}
}
