package demo

// OutputLine 预置输出面板一行(不含时间前缀,由 app.outAppend 添加)。
type OutputChannel struct {
	Channel string
	Lines   []string
}

// OutputPresets 三个输出通道的演示日志。
func OutputPresets() []OutputChannel {
	return []OutputChannel{
		{Channel: "Git", Lines: []string{
			"git status --porcelain 完成 · 3 个路径已更改",
			"git fetch origin 完成",
			"WARN 演示:远端暂不可用,仅本地状态",
		}},
		{Channel: "语言服务", Lines: []string{
			"rust-analyzer 已启动(演示模式未连接真实服务器)",
			"gopls 未安装 · 安装:go install golang.org/x/tools/gopls@latest",
			"documentSymbol 退回本地索引(演示)",
		}},
		{Channel: "MiniCode", Lines: []string{
			"INFO 演示模式 · router-demo 沙盒已加载",
			"DEBUG 输出面板:MiniCode 通道在演示下仍读日志文件尾部(若存在)",
		}},
	}
}

// SeedOutput 把预置行写入宿主提供的追加函数(ch 为 app 包里的通道名)。
func SeedOutput(append func(ch, line string)) {
	for _, p := range OutputPresets() {
		for _, ln := range p.Lines {
			append(p.Channel, ln)
		}
	}
}
