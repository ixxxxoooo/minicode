package core

import "strings"

// ProjectInfo 是侧栏里的一个项目(工作目录)及其会话列表。
type ProjectInfo struct {
	ID, Name, Path string
	Sessions       []SessionInfo
	Count          int // 会话总数(Sessions 可能只是前几条)
}

// SessionInfo 是侧栏里的一条会话:只有标题与时间,不带运行状态(设计稿 v3 的约定)。
type SessionInfo struct {
	ID, Title string
	Updated   int64  // unix 毫秒
	Label     string // 已格式化的时间文案(演示用;真实模式由 RelTime 生成)
	Group     string // 分段:今天 / 昨天 / 更早(方案 B 按它分段)
	Clock     string // 方案 B 行尾时间(今天的用相对时间,其余用时刻 / 星期 / 日期)
}

// ShortTitle 是面包屑里用的短标题:冒号后的部分(「完整修复一个 Bug:路由大小写匹配」→「路由大小写匹配」)。
func ShortTitle(title string) string {
	for _, sep := range []string{":", ":"} {
		if i := strings.Index(title, sep); i >= 0 && i+len(sep) < len(title) {
			return strings.TrimSpace(title[i+len(sep):])
		}
	}
	return title
}
