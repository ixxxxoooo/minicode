package demo

// QueuePromptSamples 演示排队卡:带图片、多项、图片缺失占位。
func QueuePromptSamples() []string {
	return []string{
		"请对照截图里的报错修路由\n![](demo/queue-screenshot.png)",
		"再补一条单元测试",
		"![缺失文件](demo/queue-missing.png)",
	}
}
