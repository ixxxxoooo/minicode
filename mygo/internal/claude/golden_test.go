package claude

// 回放快照(golden)测试:把真实协议形状的 jsonl 样本喂给翻译器,再经 core.State 归约,
// 把最终状态打成确定性文本,与 testdata 里的 .golden 比对。
//
//	testdata/live/*.jsonl    实时 stream-json(Translator.Handle / ControlRequest 路径)
//	testdata/replay/*.jsonl  会话历史文件(ReplayLines 路径,带 timestamp,算用时)
//
// 样本里以 # 开头的行是注释;live 样本还支持两条指令:
//
//	#user <文字>      模拟用户发出提示(Translator.UserPrompt)
//	#cancel <请求 id> 模拟 claude 取消一个挂起的权限请求
//
// 改动翻译器 / 归约器后快照对不上时:先看 diff 判断是改坏了还是有意改变;
// 确认是有意改变,再 `go test ./internal/claude -run Golden -update` 重新生成并一并提交。

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

var update = flag.Bool("update", false, "重新生成 testdata 里的 .golden 快照")

// ───────── 确定性 dump ─────────

var timeType = reflect.TypeOf(time.Time{})

// goldenSkip 是 State 里不进快照的字段:噪声(默认模型表)或依赖挂钟时间的字段。
var goldenSkip = map[string]bool{"Models": true, "Rev": true, "Now": true, "TurnAt": true, "Live": true, "LiveDone": true, "Timings": true}

// dumpValue 只按 Kind 读取(不调 Interface),所以未导出字段也能打;零值字段省略,指针不打地址,
// map 按键排序,函数 / 通道 / 时间一律跳过,保证同一输入永远得到同一输出。
func dumpValue(b *strings.Builder, v reflect.Value, ind int, depth int) {
	pad := strings.Repeat("  ", ind)
	if depth > 40 {
		b.WriteString("<太深>")
		return
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		dumpValue(b, v.Elem(), ind, depth+1)
	case reflect.Struct:
		if v.Type() == timeType {
			b.WriteString("<时间>")
			return
		}
		fmt.Fprintf(b, "%s{", v.Type().Name())
		n := 0
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			fv := v.Field(i)
			if fv.IsZero() || skipKind(fv.Kind()) || (ind == 0 && goldenSkip[f.Name]) {
				continue
			}
			fmt.Fprintf(b, "\n%s  %s: ", pad, f.Name)
			dumpValue(b, fv, ind+1, depth+1)
			n++
		}
		if n > 0 {
			fmt.Fprintf(b, "\n%s", pad)
		}
		b.WriteString("}")
	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteString("[")
		for i := 0; i < v.Len(); i++ {
			fmt.Fprintf(b, "\n%s  - ", pad)
			dumpValue(b, v.Index(i), ind+1, depth+1)
		}
		fmt.Fprintf(b, "\n%s]", pad)
	case reflect.Map:
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j]) })
		b.WriteString("map{")
		for _, k := range keys {
			fmt.Fprintf(b, "\n%s  %v: ", pad, k)
			dumpValue(b, v.MapIndex(k), ind+1, depth+1)
		}
		if len(keys) > 0 {
			fmt.Fprintf(b, "\n%s", pad)
		}
		b.WriteString("}")
	case reflect.String:
		fmt.Fprintf(b, "%q", v.String())
	case reflect.Bool:
		fmt.Fprintf(b, "%v", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if name, ok := enumNames[v.Type().Name()][v.Int()]; ok {
			fmt.Fprintf(b, "%s(%d)", name, v.Int())
		} else {
			fmt.Fprintf(b, "%d", v.Int())
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fmt.Fprintf(b, "%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		fmt.Fprintf(b, "%g", v.Float())
	default:
		fmt.Fprintf(b, "<%s>", v.Kind())
	}
}

// enumNames 给 core 里没有 String() 的枚举补名字,让快照读起来是 ToolFailed(3) 而不是裸的 3。
// 用真实常量建表:常量被改号或重排,快照里的名字会跟着变,评审时一眼能看到。新增常量没登记时回退成裸数字。
var enumNames = map[string]map[int64]string{
	"ToolStatus": {int64(core.ToolPending): "ToolPending", int64(core.ToolRunning): "ToolRunning", int64(core.ToolCompleted): "ToolCompleted",
		int64(core.ToolFailed): "ToolFailed", int64(core.ToolDenied): "ToolDenied", int64(core.ToolInterrupted): "ToolInterrupted"},
	"ToolKind": {int64(core.KindRead): "KindRead", int64(core.KindEdit): "KindEdit", int64(core.KindWrite): "KindWrite", int64(core.KindDelete): "KindDelete",
		int64(core.KindMove): "KindMove", int64(core.KindSearch): "KindSearch", int64(core.KindExecute): "KindExecute", int64(core.KindThink): "KindThink",
		int64(core.KindFetch): "KindFetch", int64(core.KindBrowser): "KindBrowser", int64(core.KindMCP): "KindMCP", int64(core.KindSubagent): "KindSubagent"},
	"Phase":       {int64(core.PhaseStarting): "PhaseStarting", int64(core.PhaseIdle): "PhaseIdle", int64(core.PhaseRunning): "PhaseRunning"},
	"StopReason":  {int64(core.StopEndTurn): "StopEndTurn", int64(core.StopCancelled): "StopCancelled", int64(core.StopRefusal): "StopRefusal", int64(core.StopMaxTokens): "StopMaxTokens", int64(core.StopError): "StopError"},
	"NoticeLevel": {int64(core.NoticeInfo): "NoticeInfo", int64(core.NoticeWarning): "NoticeWarning", int64(core.NoticeError): "NoticeError"},
	"SubagentStatus": {int64(core.SubStarting): "SubStarting", int64(core.SubRunning): "SubRunning", int64(core.SubBackground): "SubBackground",
		int64(core.SubCompleted): "SubCompleted", int64(core.SubFailed): "SubFailed", int64(core.SubStopped): "SubStopped"},
	"TaskKind": {int64(core.TaskOther): "TaskOther", int64(core.TaskShell): "TaskShell", int64(core.TaskAgent): "TaskAgent",
		int64(core.TaskMonitor): "TaskMonitor", int64(core.TaskWorkflow): "TaskWorkflow"},
	"TaskStatus": {int64(core.TaskRunning): "TaskRunning", int64(core.TaskCompleted): "TaskCompleted", int64(core.TaskFailed): "TaskFailed", int64(core.TaskStopped): "TaskStopped"},
	"Effort":     {int64(core.EffortLow): "EffortLow", int64(core.EffortMedium): "EffortMedium", int64(core.EffortHigh): "EffortHigh", int64(core.EffortXHigh): "EffortXHigh", int64(core.EffortMax): "EffortMax"},
}

func skipKind(k reflect.Kind) bool {
	return k == reflect.Func || k == reflect.Chan || k == reflect.UnsafePointer
}

// dumpState 打出 State 的快照:先是一段摘要,再是整棵状态树。
func dumpState(st *core.State) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# 摘要:条目 %d 个 · 阶段 %d · 会话 %q · 标题 %q\n", len(st.Items), st.Phase, st.SessionID, st.Title)
	dumpValue(&b, reflect.ValueOf(st), 0, 0)
	b.WriteString("\n")
	return b.String()
}

// ───────── 样本驱动 ─────────

func fixedState() *core.State {
	st := core.NewState()
	st.Now = func() int64 { return 1_700_000_000_000 }
	return st
}

// sampleLines 读样本,去掉空行与 # 注释;# 指令行原样保留(以 "#user " / "#cancel " 开头)。
func sampleLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, ln := range strings.Split(string(data), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || (strings.HasPrefix(ln, "#") && !strings.HasPrefix(ln, "#user ") && !strings.HasPrefix(ln, "#cancel ")) {
			continue
		}
		out = append(out, ln)
	}
	return out
}

// runLive 让样本走实时路径:与后端读 stdout 的分派一致(control_request 走 ControlRequest,其余走 Handle)。
func runLive(t *testing.T, lines []string) *core.State {
	t.Helper()
	tr := NewTranslator("/work/demo")
	tr.Clock = func() time.Time { return time.UnixMilli(1_700_000_000_000) }
	st := fixedState()
	apply := func(evs []core.Event) {
		for _, e := range evs {
			st.Apply(e)
		}
	}
	n := 0
	for _, l := range lines {
		n++
		switch {
		case strings.HasPrefix(l, "#user "):
			apply(tr.UserPrompt(strings.TrimPrefix(l, "#user "), nil, fmt.Sprintf("uuid-%d", n)))
		case strings.HasPrefix(l, "#cancel "):
			apply(tr.CancelRequest(strings.TrimPrefix(l, "#cancel ")))
		default:
			v := parseLine(l)
			if v == nil {
				t.Fatalf("第 %d 行不是合法 JSON:%s", n, l)
			}
			if str(v, "type") == "control_request" {
				evs, _ := tr.ControlRequest(v)
				apply(evs)
			} else {
				apply(tr.Handle(v))
			}
		}
	}
	return st
}

// runReplay 让样本走历史回放路径,和后端打开旧会话一样用 EvReplay 包起来。
func runReplay(t *testing.T, lines []string) *core.State {
	t.Helper()
	st := fixedState()
	st.Apply(core.EvReplay{On: true})
	for _, e := range ReplayLines(lines, "/work/demo") {
		st.Apply(e)
	}
	st.Apply(core.EvReplay{On: false})
	return st
}

// checkGolden 与 .golden 比对;-update 时改写文件。快照缺失视为失败并提示怎么生成。
func checkGolden(t *testing.T, golden, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("缺少快照 %s(用 -update 生成后检查内容再提交):%v", golden, err)
	}
	if string(want) != got {
		t.Errorf("快照不一致:%s\n%s\n(确认是有意改变后,用 -update 重新生成)", golden, firstDiff(string(want), got))
	}
}

// firstDiff 给出第一处不同的行附近的上下文,省得整份快照刷屏。
func firstDiff(want, got string) string {
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < max(len(wl), len(gl)); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			lo := max(0, i-3)
			return fmt.Sprintf("第 %d 行起不同\n  期望:%s\n  实际:%s\n  上文:\n    %s", i+1, w, g, strings.Join(wl[lo:i], "\n    "))
		}
	}
	return "(长度不同)"
}

func goldenSuite(t *testing.T, dir string, run func(*testing.T, []string) *core.State) {
	files, _ := filepath.Glob(filepath.Join("testdata", dir, "*.jsonl"))
	if len(files) == 0 {
		t.Fatalf("testdata/%s 里没有样本", dir)
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".jsonl")
		t.Run(name, func(t *testing.T) {
			lines := sampleLines(t, f)
			got := dumpState(run(t, lines))
			// 同一输入必须得到同一输出,否则快照没有意义(通常是漏掉了依赖挂钟的字段)
			time.Sleep(5 * time.Millisecond) // 隔开几毫秒,才抓得到残留的挂钟依赖(同一毫秒内两次读数相同,会被蒙混过去)
			if again := dumpState(run(t, lines)); again != got {
				t.Fatalf("同一样本两次回放结果不同,快照不稳定:\n%s", firstDiff(got, again))
			}
			checkGolden(t, strings.TrimSuffix(f, ".jsonl")+".golden", got)
		})
	}
}

func TestGoldenLive(t *testing.T)   { goldenSuite(t, "live", runLive) }
func TestGoldenReplay(t *testing.T) { goldenSuite(t, "replay", runReplay) }
