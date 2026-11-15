// 演示数据索引(通知中心)
//
//   - NoticeSeeds:IDE 右下角历史 + 铃铛未读(上限 100,种子 120 条测溢出)。
//   - 入口:IDE 形态状态栏铃铛;Agent 仍走顶部 toast。
//   - 调试:-ed pop:notice 或 MINI_NOTICES=1。
package demo

import "time"

// NoticeSeed 是一条可写入通知中心的历史记录。
type NoticeSeed struct {
	Level   string // info / ok / warn / err
	Title   string
	Body    string
	Read    bool
	Ago     time.Duration // 相对当前时间的偏移
	Actions bool
}

// NoticeSeeds 各级别 × 动作/已读/长文案 × 溢出。
func NoticeSeeds() []NoticeSeed {
	var out []NoticeSeed
	add := func(s NoticeSeed) { out = append(out, s) }

	add(NoticeSeed{Level: "info", Title: "发现新版本 0.3.1", Body: "正在后台下载,完成后可在关于页重启。", Ago: 30 * time.Second})
	add(NoticeSeed{Level: "ok", Title: "回合结束", Body: "路由大小写匹配", Read: true, Ago: 2 * time.Minute})
	add(NoticeSeed{Level: "warn", Title: "未安装 rust-analyzer", Body: "语言智能暂不可用。", Actions: true, Ago: 5 * time.Minute})
	add(NoticeSeed{Level: "err", Title: "下载更新失败", Body: "网络连接丢失,稍后在关于页重试。", Actions: true, Ago: 8 * time.Minute})
	add(NoticeSeed{Level: "warn", Title: "系统通知权限未授予", Body: "MiniCode 无法发送系统通知。请在「系统设置 → 通知」中为本应用开启通知。", Actions: true, Ago: 10 * time.Minute})
	add(NoticeSeed{Level: "info", Title: "后台任务完成", Body: "cargo watch -x test", Read: true, Ago: 12 * time.Minute})

	long := "这是一条刻意写得很长的通知正文,用来测试通知面板里的换行、省略与滚动。"
	long += "当文案超过两行时仍应完整展示,并保留动作按钮与相对时间戳,便于与设计稿 overlays 页对照。"
	add(NoticeSeed{Level: "info", Title: "长文案通知示例", Body: long, Ago: 15 * time.Minute})

	levels := []struct{ lv, title, body string }{
		{"info", "索引构建完成", "router-demo 符号索引已更新。"},
		{"ok", "提交成功", "main · 3 files changed"},
		{"warn", "合并冲突", "docs-site/config.yaml 需要手动处理。"},
		{"err", "拉取失败", "non-fast-forward,先拉取再推送。"},
	}
	for i := 0; i < 120; i++ {
		x := levels[i%len(levels)]
		read := i%3 == 0
		act := i%5 == 0
		ago := time.Duration(20+i) * time.Minute
		title := x.title
		if i%11 == 0 {
			title += " (同类 #" + itoa(i/11+1) + ")"
		}
		add(NoticeSeed{Level: x.lv, Title: title, Body: x.body, Read: read, Ago: ago, Actions: act})
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
