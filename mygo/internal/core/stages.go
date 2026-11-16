package core

import (
	"fmt"
	"strings"
	"time"
)

// 回合分阶段耗时(设计稿 resp-time 页)。阶段全部来自 claude 的真实事件:
// system status(requesting)= 请求已发出,message_start = 第一个字节,content_block_start 区分思考 / 正文 / 工具,
// message_stop + stop_reason 区分「去执行工具」与「回合收尾」,result = 回合结束。

// StageKind 是回合里的一个阶段。
type StageKind uint8

const (
	StageNone   StageKind = iota
	StageConn             // 连接 Claude:进程冷启动 + 握手(本机)
	StagePrep             // 本地准备:写入提示 → 请求发出(本机)
	StageWait             // 等待模型:请求发出 → 第一个字节(上游),每次模型调用一段
	StageThink            // 思考
	StageWrite            // 生成:正文 / 组织工具调用
	StageTool             // 工具执行(本机)
	StageAsk              // 等你确认:权限 / 提问 / 计划,不计入模型耗时
	StageFin              // 收尾:最后一个 message_stop → result
	StageQueued           // 收尾期间又发了一条:等上一轮收尾结束 claude 才会处理它
)

// Label 阶段的中文名。
func (k StageKind) Label() string {
	switch k {
	case StageConn:
		return "连接 Claude"
	case StagePrep:
		return "本地准备"
	case StageWait:
		return "等待模型"
	case StageThink:
		return "思考"
	case StageWrite:
		return "生成"
	case StageTool:
		return "工具"
	case StageAsk:
		return "等你确认"
	case StageFin:
		return "收尾"
	case StageQueued:
		return "等待上一轮收尾"
	}
	return ""
}

// TimingPhase 是回合里连续的一段。
type TimingPhase struct {
	Kind  StageKind
	Ms    int64
	Call  int    // 第几次模型调用(等待 / 思考 / 生成 / 收尾),0 = 与调用无关
	Label string // 工具阶段写工具名
}

// CallInfo 是一次模型调用在第一个字节到达时就知道的信息。
type CallInfo struct {
	Cache CacheTokens // message_start.usage:命中 / 写入 / 未缓存;三项全 0 = 服务商没给缓存字段
}

// HasCache 服务商是否返回了缓存字段。
func (c CallInfo) HasCache() bool { return c.Cache.Total() > 0 }

// CacheVerdict 本次调用的缓存结论,用于等待行 chip 与弹卡备注。
type CacheVerdict uint8

const (
	CacheUnknown CacheVerdict = iota // 没有缓存字段(第三方代理):不显示
	CacheHit                         // 读 ≥ 80%
	CachePartial                     // 20% ~ 80%
	CacheWritten                     // 写入占主导:第一次带这段上下文,慢是正常的
	CacheMiss                        // 读 = 0 且上下文大
)

// Verdict 按设计稿规则判定。
func (c CallInfo) Verdict() CacheVerdict {
	total := c.Cache.Total()
	if total == 0 {
		return CacheUnknown
	}
	hit := c.Cache.HitPercent()
	switch {
	case hit >= 80:
		return CacheHit
	case hit >= 20:
		return CachePartial
	case c.Cache.Written*2 > total:
		return CacheWritten
	case c.Cache.Read == 0 && total > 20_000:
		return CacheMiss
	}
	return CachePartial
}

// Text 缓存结论的文案:`缓存命中 98%` / `缓存写入 26.2k(首次)` / `缓存未命中`。没有字段返回空串。
func (c CallInfo) Text() string {
	switch c.Verdict() {
	case CacheHit:
		return fmt.Sprintf("缓存命中 %d%%", c.Cache.HitPercent())
	case CachePartial:
		return fmt.Sprintf("缓存部分命中 %d%%", c.Cache.HitPercent())
	case CacheWritten:
		return "缓存写入 " + FormatTokens(c.Cache.Written) + "(首次)"
	case CacheMiss:
		return "缓存未命中 " + FormatTokens(c.Cache.Total())
	}
	return ""
}

// Slow 这次缓存结论是否值得用琥珀色提醒(写入 / 未命中)。
func (c CallInfo) Slow() bool { v := c.Verdict(); return v == CacheWritten || v == CacheMiss }

// LiveStage 是进行中回合当前所处的阶段(界面的「等待行」读它)。
type LiveStage struct {
	Kind      StageKind
	Since     time.Time // 进入当前阶段的时刻
	TurnStart time.Time // 回合开始(点发送)的时刻
	Call      int       // 当前是第几次模型调用
	Label     string    // 工具名等
	Cache     *CacheTokens
}

// 慢的阈值(设计稿「慢因规则」)。
const (
	SlowWait = 8 * time.Second
	SlowFin  = 3 * time.Second
	SlowConn = 3 * time.Second
)

// ModelMs 扣掉「等你确认」后的耗时(标题里的「模型 x s」)。
func (t TurnTiming) ModelMs() int64 {
	var ask int64
	for _, p := range t.Phases {
		if p.Kind == StageAsk {
			ask += p.Ms
		}
	}
	return t.TotalMs - ask
}

// SlowestWait 最慢的一次「等待模型」(没有返回 0)。
func (t TurnTiming) SlowestWait() (ms int64, call int) {
	for _, p := range t.Phases {
		if p.Kind == StageWait && p.Ms > ms {
			ms, call = p.Ms, p.Call
		}
	}
	return
}

// PhaseSummary 给日志用的一行摘要:`等待模型#1 5.7s · 生成 0.5s · 工具 0.8s …`。
func (t TurnTiming) PhaseSummary() string {
	parts := make([]string, 0, len(t.Phases))
	for _, p := range t.Phases {
		name := p.Kind.Label()
		if p.Label != "" {
			name += " " + p.Label
		}
		if p.Kind == StageWait && p.Call > 0 {
			name += fmt.Sprintf("#%d", p.Call)
		}
		parts = append(parts, fmt.Sprintf("%s %.1fs", name, float64(p.Ms)/1000))
	}
	return strings.Join(parts, " · ")
}
