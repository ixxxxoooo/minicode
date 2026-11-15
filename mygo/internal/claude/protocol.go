package claude

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 发往 claude 标准输入的消息构造:全是纯函数,便于单测。

type obj = map[string]any

// UserMessage 是用户消息。steer 为真时带 priority:next,让 claude 在当前回合两次工具调用之间就地吸收
// (实测:now 会提前收尾当前回合,不是引导)。uuid 是这条消息的 claude 消息 id:文件检查点按它记录。
func UserMessage(text string, images []string, steer bool, uuid string) obj {
	var content []any
	for _, p := range images {
		b, err := os.ReadFile(p)
		if err != nil { // 读不到的图片退化成文字说明,避免整条消息发送失败
			content = append(content, obj{"type": "text", "text": "[图片读取失败:" + p + "]"})
			continue
		}
		content = append(content, obj{"type": "image", "source": obj{"type": "base64", "media_type": imageMediaType(p), "data": base64.StdEncoding.EncodeToString(b)}})
	}
	content = append(content, obj{"type": "text", "text": text})
	m := obj{"type": "user", "message": obj{"role": "user", "content": content}, "parent_tool_use_id": nil, "session_id": "", "uuid": uuid}
	if steer {
		m["priority"] = "next"
	}
	return m
}

func imageMediaType(p string) string {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(p), ".")) {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	}
	return "image/png"
}

func controlRequest(id string, req obj) obj {
	return obj{"type": "control_request", "request_id": id, "request": req}
}

// Initialize 是初始化握手。perTaskStopAffordance:声明界面有逐任务的停止按钮,中断只中止当前回合,
// 不连带杀掉后台子 Agent;agentProgressSummaries:task_progress 带一句话进度;forwardSubagentText:转发前台子 Agent 的文本。
func Initialize(id string) obj {
	return controlRequest(id, obj{"subtype": "initialize", "perTaskStopAffordance": true, "agentProgressSummaries": true, "forwardSubagentText": true})
}

func RewindFiles(id, uuid string, dry bool) obj {
	return controlRequest(id, obj{"subtype": "rewind_files", "user_message_id": uuid, "dry_run": dry})
}
func McpStatusReq(id string) obj { return controlRequest(id, obj{"subtype": "mcp_status"}) }
func McpToggleReq(id, name string, on bool) obj {
	return controlRequest(id, obj{"subtype": "mcp_toggle", "serverName": name, "enabled": on})
}
func StopTask(id, task string) obj {
	return controlRequest(id, obj{"subtype": "stop_task", "task_id": task})
}
func BackgroundTasks(id, toolUse string) obj {
	r := obj{"subtype": "background_tasks"}
	if toolUse != "" {
		r["tool_use_id"] = toolUse
	}
	return controlRequest(id, r)
}
func Interrupt(id string) obj { return controlRequest(id, obj{"subtype": "interrupt"}) }

// SetModel 切换模型;default 表示恢复默认,此时不带 model 字段。
func SetModel(id, model string) obj {
	r := obj{"subtype": "set_model"}
	if model != "default" {
		r["model"] = model
	}
	return controlRequest(id, r)
}
func SetPermissionMode(id string, m core.PermissionMode) obj {
	return controlRequest(id, obj{"subtype": "set_permission_mode", "mode": m.CLI()})
}
func SetEffort(id string, e core.Effort) obj {
	return controlRequest(id, obj{"subtype": "apply_flag_settings", "settings": obj{"effortLevel": e.CLI()}})
}

func successResponse(reqID string, body obj) obj {
	return obj{"type": "control_response", "response": obj{"subtype": "success", "request_id": reqID, "response": body}}
}
func errorResponse(reqID, msg string) obj {
	return obj{"type": "control_response", "response": obj{"subtype": "error", "request_id": reqID, "error": msg}}
}

// PendingKind 是挂起的控制请求类型。
type PendingKind int

const (
	PendPermission PendingKind = iota
	PendQuestion
	PendPlan
)

// Pending 是一个等待用户决定的 can_use_tool 请求。
type Pending struct {
	RequestID   string
	ToolUseID   string
	ToolName    string
	Input       obj
	Suggestions []any // claude 给出的权限更新建议
	Kind        PendingKind
}

// SuggestedMode 从建议里取出会改变权限模式的那一项。
func (p *Pending) SuggestedMode() (core.PermissionMode, bool) {
	for _, s := range p.Suggestions {
		if m, ok := s.(obj); ok && str(m, "type") == "setMode" {
			if pm, ok := core.ParseMode(str(m, "mode")); ok {
				return pm, true
			}
		}
	}
	return 0, false
}

func (p *Pending) allow(updated obj, perms any) obj {
	b := obj{"behavior": "allow", "updatedInput": updated}
	if p.ToolUseID != "" {
		b["toolUseID"] = p.ToolUseID
	}
	if perms != nil {
		b["updatedPermissions"] = perms
	}
	return b
}

func (p *Pending) deny(msg string) obj {
	b := obj{"behavior": "deny", "message": msg}
	if p.ToolUseID != "" {
		b["toolUseID"] = p.ToolUseID
	}
	return b
}

// AlwaysAllowPermissions 是「总是允许」写入的权限更新:优先采用 claude 的建议,否则按工具名生成本地规则。
func (p *Pending) AlwaysAllowPermissions() any {
	if len(p.Suggestions) > 0 {
		return p.Suggestions
	}
	rule := obj{"toolName": p.ToolName}
	if p.ToolName == "Bash" {
		if f := strings.Fields(str(p.Input, "command")); len(f) > 0 {
			rule["ruleContent"] = f[0] + ":*"
		}
	}
	return []any{obj{"type": "addRules", "rules": []any{rule}, "behavior": "allow", "destination": "localSettings"}}
}

// PermissionResponse 是权限请求的应答。option:allow_once / allow_always / 其它视为拒绝。
func (p *Pending) PermissionResponse(option string) obj {
	var b obj
	switch option {
	case "allow_once":
		b = p.allow(p.Input, nil)
	case "allow_always":
		b = p.allow(p.Input, p.AlwaysAllowPermissions())
	default:
		b = p.deny("用户拒绝了此次操作")
	}
	return successResponse(p.RequestID, b)
}

// QuestionResponse 是 AskUserQuestion 的应答:把答案以「问题 → 选中标签」写回入参的 answers。
func (p *Pending) QuestionResponse(answers [][]string) obj {
	in := obj{}
	for k, v := range p.Input {
		in[k] = v
	}
	m := obj{}
	for i, q := range arr(p.Input, "questions") {
		qm, _ := q.(obj)
		picked := ""
		if i < len(answers) {
			picked = strings.Join(answers[i], ", ")
		}
		m[str(qm, "question")] = picked
	}
	in["answers"] = m
	return successResponse(p.RequestID, p.allow(in, nil))
}

// PlanResponse 是 ExitPlanMode 的应答:批准时切到用户选的模式(没选沿用 claude 建议),拒绝时让它继续规划。
func (p *Pending) PlanResponse(approve bool, mode *core.PermissionMode) obj {
	if !approve {
		return successResponse(p.RequestID, p.deny("用户没有批准该计划,请根据反馈继续完善计划"))
	}
	var perms any
	if mode != nil {
		perms = []any{obj{"type": "setMode", "mode": mode.CLI(), "destination": "session"}}
	} else if len(p.Suggestions) > 0 {
		perms = p.Suggestions
	}
	return successResponse(p.RequestID, p.allow(p.Input, perms))
}

// RejectResponse 按类型生成「默认拒绝」应答(进程退出前清理用)。
func (p *Pending) RejectResponse() obj {
	switch p.Kind {
	case PendQuestion:
		return successResponse(p.RequestID, p.deny("用户取消了提问"))
	case PendPlan:
		return p.PlanResponse(false, nil)
	}
	return p.PermissionResponse("reject")
}

// ContextUsageReq 查询上下文明细(本地请求)。
func ContextUsageReq(id string) obj { return controlRequest(id, obj{"subtype": "get_context_usage"}) }

// UsageReq 查询账户用量与限额。
func UsageReq(id string) obj { return controlRequest(id, obj{"subtype": "get_usage"}) }
