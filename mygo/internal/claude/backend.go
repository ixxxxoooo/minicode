package claude

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

var log = diag.L("claude")

// Config 是一个会话后端的配置。
type Config struct {
	Cwd string
	// SessionID 非空 = 恢复这个已有会话(并回放 HistoryPath);空 = 新会话(自动分配 id)。
	SessionID          string
	HistoryPath        string
	ClaudePath         string
	Model              string // 空 = default
	Mode               *core.PermissionMode
	Effort             *core.Effort
	AddDirs            []string
	AppendSystemPrompt string
	Env                []core.EnvVar
	// InitTimeout 握手超时,零值用 60 秒(测试里调短)。
	InitTimeout time.Duration
	// ProjectsDir 会话记录目录,空 = ~/.claude/projects(重启时判断会话是否已落盘)。
	ProjectsDir string
	// Fork 非空 = 从已有会话分叉出新会话(不走 HistoryPath)。
	Fork *ForkConfig
}

type msgKind int

const (
	mLine msgKind = iota
	mStderr
	mEOF
	mInitTimeout
)

type msg struct {
	p    *proc
	kind msgKind
	text string
}

// proc 是一个运行中的 claude 子进程。只有 run 循环读写它的可变字段;读线程只往 msgs 投递。
type proc struct {
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	pid          int
	started      time.Time
	initAt       time.Time     // 握手完成的时刻(零值 = 还没完成)
	stop         chan struct{} // 关闭 = 不再投递消息
	timer        *time.Timer
	stderrTail   []string
	shuttingDown bool
	badLines     int
}

type reqKind int

const (
	rkInit reqKind = iota
	rkSetModel
	rkSetMode
	rkOther
	rkUsage
	rkContext
	rkMcpStatus
	rkMcpToggle
	rkRewind
	rkResumeAt
)

type req struct {
	kind   reqKind
	model  string
	mode   core.PermissionMode
	what   string
	uuid   string
	dryRun bool
}

// Backend 是真实 Claude 后端(惰性):构造后不起进程,收到第一条提示 / 回退请求才启动 claude;
// 进程退出(崩溃、握手超时、被杀)后句柄依然有效,下一条提示会重新启动(会话已落盘就 --resume 接着聊)。
//
// 线程模型:读 stdout / stderr 的 goroutine 与界面命令都只往通道投递;run 循环独占 stdin、Translator
// 与全部状态,顺序消费,所以没有锁。
type Backend struct {
	cfg  Config
	sink backend.Sink
	cmds chan core.Command
	msgs chan msg
	done chan struct{}
	once sync.Once

	// 以下只在 run 循环里访问
	p          *proc
	tr         *Translator
	turnActive bool
	reqs       map[string]req
	// 本回合的实测耗时(墙钟):发送时刻、第一个响应、发送时进程是否还没起(冷启动)
	turnT0        time.Time
	turnFirst     time.Duration
	turnCold      bool
	carry         []time.Time // 收尾期间发出的提示的发送时刻:result 到达后无缝接力成下一回合
	stages        stageClock  // 本回合的分阶段计时(阶段切换时向界面发 EvTurnPhase)
	reqSeq        int
	usageAsked    time.Time
	confirmedMode core.PermissionMode
	sessionID     string
	resume        string
	fork          bool
	started       bool
	released      bool      // 上一个进程是被空闲回收的(重新启动时不提示「进程已重新启动」)
	warmFailAt    time.Time // 最近一次预热失败的时间,60 秒内不再自动重试
	models        []core.ModelInfo
	pendingEdit   *pendingUserEdit
}

var idGen atomic.Int64

// New 创建后端。
func New(cfg Config) *Backend {
	b := &Backend{
		cfg: cfg, cmds: make(chan core.Command, 1024), msgs: make(chan msg, 256), done: make(chan struct{}),
		reqs: map[string]req{},
	}
	switch {
	case cfg.Fork != nil && cfg.Fork.ParentSessionID != "":
		b.resume = cfg.Fork.ParentSessionID
		b.fork = true
		if cfg.Fork.NewSessionID != "" {
			b.sessionID = cfg.Fork.NewSessionID
		} else {
			b.sessionID = newUUID()
		}
	case cfg.SessionID != "":
		b.sessionID, b.resume = cfg.SessionID, cfg.SessionID
	default:
		b.sessionID = newUUID()
	}
	if cfg.Mode != nil {
		b.confirmedMode = *cfg.Mode
	}
	b.models = append([]core.ModelInfo{{ID: "default", Name: "默认", Description: "跟随 claude 的设置"}}, core.DefaultModels()...)
	return b
}

func (b *Backend) Name() string { return "claude" }

// SessionID 当前会话 id。
func (b *Backend) SessionID() string { return b.sessionID }

func (b *Backend) Start(sink backend.Sink) {
	b.sink = sink
	go func() {
		defer diag.Recover("claude 后端循环")
		b.run()
	}()
}

func (b *Backend) Send(cmd core.Command) {
	select {
	case b.cmds <- cmd:
	default:
		log.Warn("命令队列已满,丢弃命令", "cmd", fmt.Sprintf("%T", cmd))
	}
}

func (b *Backend) Close() { b.once.Do(func() { close(b.done) }) }

// KillChildForTest 结束当前 claude 子进程(仅集成测试验证崩溃恢复)。
func (b *Backend) KillChildForTest() {
	if b.p != nil && b.p.cmd != nil && b.p.cmd.Process != nil {
		_ = b.p.cmd.Process.Kill()
	}
}

func (b *Backend) emit(evs ...core.Event) {
	for _, e := range evs {
		b.sink(e)
	}
}

func (b *Backend) model() string {
	if b.cfg.Model == "" {
		return "default"
	}
	return b.cfg.Model
}

func (b *Backend) mode() core.PermissionMode {
	if b.cfg.Mode != nil {
		return *b.cfg.Mode
	}
	return b.confirmedMode
}

func (b *Backend) run() {
	// 先回放历史,再报告就绪(此时还没有进程,模型列表是内置的)
	if b.cfg.HistoryPath != "" {
		end := diag.SpanSlow("claude", "回放历史", 300*time.Millisecond, "path", b.cfg.HistoryPath)
		evs, err := LoadSession(b.cfg.HistoryPath, b.cfg.Cwd)
		end()
		if err != nil {
			log.Warn("读取历史失败", "err", err)
			b.emit(notice(core.NoticeWarning, "读取历史会话失败:"+err.Error()))
		}
		log.Info("历史已回放", "events", len(evs))
		b.emit(evs...)
	}
	b.emit(core.EvReady{SessionID: b.sessionID, Models: b.models, Model: b.model(), Mode: b.mode()})
	for {
		select {
		case <-b.done:
			b.shutdown("界面关闭")
			return
		case c := <-b.cmds:
			b.onCommand(c)
		case m := <-b.msgs:
			if m.p != b.p || b.p == nil { // 旧进程的迟到消息
				continue
			}
			b.onMsg(m)
		}
	}
}

// ---------- 进程 ----------

func newUUID() string {
	var u [16]byte
	_, _ = rand.Read(u[:])
	u[6] = u[6]&0x0f | 0x40
	u[8] = u[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

func (b *Backend) options() Options {
	o := Options{
		Cwd: b.cfg.Cwd, Model: b.cfg.Model, Mode: b.cfg.Mode, Effort: b.cfg.Effort, AddDirs: b.cfg.AddDirs,
		AppendSystemPrompt: b.cfg.AppendSystemPrompt, ExtraEnv: b.cfg.Env,
	}
	if b.resume != "" {
		o.Resume, o.Fork = b.resume, b.fork
		if b.fork {
			o.SessionID = b.sessionID
		}
	} else {
		o.SessionID = b.sessionID
	}
	return o
}

func (b *Backend) startProc() error {
	path, ok := FindClaude(b.cfg.ClaudePath)
	if !ok {
		return fmt.Errorf("没有找到 claude 命令。请先安装 Claude Code(npm i -g @anthropic-ai/claude-code),或设置环境变量 MINI_CLAUDE_PATH 指向它")
	}
	opts := b.options()
	cmd := exec.Command(path, opts.Args()...)
	env := make([]string, 0, len(os.Environ())+8)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "CLAUDECODE=") { // 在 Claude Code 会话里启动本程序时,避免继承其嵌套标记
			env = append(env, e)
		}
	}
	cmd.Env = append(env, opts.Env()...)
	if b.cfg.Cwd != "" {
		cmd.Dir = b.cfg.Cwd
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 claude 失败(%s):%w", path, err)
	}
	p := &proc{cmd: cmd, stdin: stdin, pid: cmd.Process.Pid, started: time.Now(), stop: make(chan struct{})}
	b.p = p
	b.reqs = map[string]req{}
	b.tr = NewTranslator(b.cfg.Cwd)
	b.tr.IDGen = int(idGen.Add(1)) - 1
	b.tr.SessionID = opts.EffectiveSessionID()
	if b.started && !b.released {
		log.Info("claude 进程已重新启动", "resume", b.resume)
		b.emit(notice(core.NoticeInfo, "claude 进程已重新启动,继续当前会话"))
	} else if b.released {
		log.Info("空闲回收后重新连接", "resume", b.resume)
	}
	b.started, b.released = true, false
	b.emit(core.EvConn{State: core.ConnStarting})
	log.Info("claude 已启动", "pid", p.pid, "path", path, "cwd", b.cfg.Cwd, "resume", opts.Resume != "")

	send := func(m msg) bool {
		select {
		case b.msgs <- m:
			return true
		case <-p.stop:
			return false
		case <-b.done:
			return false
		}
	}
	go func() { // stdout:按行读取,不限行长(单行工具结果可能很大)
		defer diag.Recover("claude stdout")
		r := bufio.NewReaderSize(stdout, 256*1024)
		for {
			line, err := r.ReadString('\n')
			if s := strings.TrimSpace(line); s != "" {
				if !send(msg{p: p, kind: mLine, text: s}) {
					return
				}
			}
			if err != nil {
				break
			}
		}
		send(msg{p: p, kind: mEOF})
	}()
	go func() { // stderr:只保留尾部,进程异常退出时用于诊断
		defer diag.Recover("claude stderr")
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			if !send(msg{p: p, kind: mStderr, text: sc.Text()}) {
				return
			}
		}
	}()
	// 握手超时:claude 卡在启动阶段(既不应答也不退出)时,不能让界面一直停在「处理中」
	to := b.cfg.InitTimeout
	if to <= 0 {
		to = 60 * time.Second
	}
	p.timer = time.AfterFunc(to, func() { send(msg{p: p, kind: mInitTimeout}) })

	b.request(req{kind: rkInit}, Initialize)
	return nil
}

// killProc 结束当前进程;不再把它的退出当作异常。
func (b *Backend) killProc() {
	p := b.p
	if p == nil {
		return
	}
	p.shuttingDown = true
	if p.timer != nil {
		p.timer.Stop()
	}
	close(p.stop)
	_ = p.stdin.Close()
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	go func() { _ = p.cmd.Wait() }()
	b.p = nil
	b.emit(core.EvConn{State: core.ConnNone})
}

// shutdown 先把挂起的请求都拒绝掉,让 claude 干净退出,再结束进程。
func (b *Backend) shutdown(why string) {
	if b.p == nil {
		return
	}
	log.Info("claude 由界面关闭", "pid", b.p.pid, "uptime", time.Since(b.p.started).Round(time.Second), "why", why)
	for _, p := range b.tr.DrainPending() {
		b.write(p.RejectResponse())
	}
	b.killProc()
}

// restartConfig 进程退出后:会话已落盘就恢复它(分叉已经发生过,不再分叉);没落盘就沿用原配置重新开始。
func (b *Backend) restartConfig() {
	projects := b.cfg.ProjectsDir
	if projects == "" {
		projects = ProjectsDir()
	}
	if b.sessionID != "" && SessionOnDisk(projects, b.cfg.Cwd, b.sessionID) {
		b.resume, b.fork = b.sessionID, false
	}
}

func (b *Backend) write(v obj) {
	if b.p == nil {
		return
	}
	data, err := json.Marshal(v)
	if err != nil {
		log.Error("序列化失败", "err", err)
		return
	}
	data = append(data, '\n')
	if _, err := b.p.stdin.Write(data); err != nil { // 进程已死时写入会失败,随后 EOF 会处理善后
		log.Debug("写入 claude 失败", "err", err)
	}
}

func (b *Backend) request(r req, build func(id string) obj) {
	b.reqSeq++
	id := fmt.Sprintf("req_%d", b.reqSeq)
	if r.kind == rkInit {
		id = "req_init"
	}
	b.reqs[id] = r
	b.write(build(id))
}

func (b *Backend) hasReq(k reqKind) bool {
	for _, r := range b.reqs {
		if r.kind == k {
			return true
		}
	}
	return false
}

func (b *Backend) refreshMcpStatus() {
	if !b.hasReq(rkMcpStatus) {
		b.request(req{kind: rkMcpStatus}, McpStatusReq)
	}
}

// ensureProc 确保进程在运行;启动失败时发提示并结束回合。
func (b *Backend) ensureProc() bool {
	if b.p != nil {
		return true
	}
	if err := b.startProc(); err != nil {
		log.Error("启动 claude 失败", "err", err)
		b.emit(notice(core.NoticeError, err.Error()), core.EvConn{State: core.ConnFailed, Why: err.Error()})
		if b.turnActive {
			b.turnActive = false
			b.stages.off()
			b.carry = nil
			b.emit(core.EvTurnEnded{Reason: core.StopError})
		}
		return false
	}
	return true
}

// StaleTaskStopped 一个已经不在运行的任务(上一个进程留下的)被用户「停止」时,本地补的终态通知。
func StaleTaskStopped(id string) core.Event {
	return core.EvTaskNotification{TaskID: id, Status: core.TaskStopped, Summary: "任务已不在运行(上次的进程已退出)"}
}

// ---------- 界面命令 ----------

func (b *Backend) onCommand(c core.Command) {
	switch c := c.(type) {
	case core.CmdPrompt:
		b.onPrompt(c)
	case core.CmdCancel:
		if b.p != nil && b.turnActive {
			b.tr.Interrupted = true
			b.request(req{kind: rkOther, what: "中断"}, Interrupt)
		}
	case core.CmdRelease:
		b.release(c.Why)
	case core.CmdWarmup:
		b.warmup()
	case core.CmdSetModel:
		b.cfg.Model = c.Model
		if c.Model == "default" {
			b.cfg.Model = ""
		}
		if b.p != nil {
			b.request(req{kind: rkSetModel, model: c.Model}, func(id string) obj { return SetModel(id, c.Model) })
		} else {
			b.emit(core.EvModelChanged{Model: c.Model})
		}
	case core.CmdSetMode:
		m := c.Mode
		b.cfg.Mode = &m
		if b.p != nil {
			b.request(req{kind: rkSetMode, mode: m}, func(id string) obj { return SetPermissionMode(id, m) })
		} else {
			b.confirmedMode = m
			b.emit(core.EvModeChanged{Mode: m})
		}
	case core.CmdSetEffort:
		e := c.Effort
		b.cfg.Effort = &e
		if b.p != nil {
			b.request(req{kind: rkOther, what: "切换思考强度"}, func(id string) obj { return SetEffort(id, e) })
		}
	case core.CmdResolvePermission:
		if p := b.takePending(c.ID); p != nil {
			b.write(p.PermissionResponse(c.Option))
			b.emit(core.EvPermissionResolved{ID: c.ID, Option: c.Option})
			b.stageAnswered()
			if c.Option == "allow_always" {
				if m, ok := p.SuggestedMode(); ok {
					b.emit(core.EvModeChanged{Mode: m})
				}
			}
		}
	case core.CmdAnswerQuestion:
		if p := b.takePending(c.ID); p != nil {
			b.write(p.QuestionResponse(c.Answers))
			b.emit(core.EvQuestionAnswered{ID: c.ID, Answers: c.Answers})
			b.stageAnswered()
		}
	case core.CmdResolvePlan:
		if p := b.takePending(c.ID); p != nil {
			b.write(p.PlanResponse(c.Approve, c.Mode))
			b.emit(core.EvPlanResolved{ID: c.ID, Approved: c.Approve})
			b.stageAnswered()
			if c.Approve {
				m, ok := core.PermissionMode(0), false
				if c.Mode != nil {
					m, ok = *c.Mode, true
				} else {
					m, ok = p.SuggestedMode()
				}
				if ok {
					b.emit(core.EvModeChanged{Mode: m})
				}
			}
		}
	case core.CmdRewindFiles:
		if !b.ensureProc() {
			return
		}
		b.request(req{kind: rkRewind, uuid: c.UUID, dryRun: c.DryRun}, func(id string) obj { return RewindFiles(id, c.UUID, c.DryRun) })
	case core.CmdStopTask:
		// 本进程不认识的任务(恢复会话时历史里遗留的,属于已退出的上一个进程):
		// claude 对未知 id 也回「成功」却什么都不发,界面会永远停在运行中,这里直接在本地收尾
		if b.p == nil || !b.tr.TaskIsLive(c.TaskID) {
			log.Info("停止任务:本进程没有这个任务,按已停止处理", "task", c.TaskID)
			b.emit(StaleTaskStopped(c.TaskID))
			return
		}
		b.request(req{kind: rkOther, what: "停止任务"}, func(id string) obj { return StopTask(id, c.TaskID) })
	case core.CmdBackgroundCurrent:
		if b.p != nil {
			b.request(req{kind: rkOther, what: "转入后台"}, func(id string) obj { return BackgroundTasks(id, c.ToolUseID) })
		}
	case core.CmdRefreshUsage:
		wantCtx, wantLim := c.Context, c.Limits
		if wantCtx {
			if b.p == nil {
				if !c.StartProc || !b.ensureProc() {
					return
				}
			}
			if !b.hasReq(rkContext) {
				b.request(req{kind: rkContext}, ContextUsageReq)
			}
		}
		if !wantLim {
			return
		}
		if b.p == nil {
			return
		}
		fresh := !c.Force && !b.usageAsked.IsZero() && time.Since(b.usageAsked) < core.LimitsTTLSecs*time.Second
		if fresh || b.hasReq(rkUsage) {
			return
		}
		b.usageAsked = time.Now()
		b.emit(b.tr.UsageChecking())
		b.request(req{kind: rkUsage}, UsageReq)
	case core.CmdRefreshMcp:
		if !c.Runtime {
			return
		}
		if b.p == nil {
			if !c.StartProc || !b.ensureProc() {
				return
			}
		}
		b.refreshMcpStatus()
	case core.CmdMcpToggle:
		if b.p != nil {
			b.request(req{kind: rkMcpToggle}, func(id string) obj { return McpToggleReq(id, c.Name, c.Enabled) })
		}
	case core.CmdSetEnv:
		b.cfg.Env = mergeManagedEnv(b.cfg.Env, c.Env)
		if b.p != nil { // 环境变量只能在启动时注入:结束当前进程,下一条提示用新环境恢复会话
			b.shutdown("环境变量变化")
			b.restartConfig()
		}
	case core.CmdForkSession:
		log.Info("分叉会话", "parent", c.ParentID, "new", c.NewID, "at", c.AtUser)
		b.emit(core.EvForkSession{ParentID: c.ParentID, NewID: c.NewID, Cwd: b.cfg.Cwd, AtUser: c.AtUser})
	case core.CmdRetryTurn:
		log.Info("重试回合(由界面发 CmdPrompt)", "user", c.UserID)
	case core.CmdEditUser:
		b.onEditUser(c)
	case core.CmdCompact:
		b.onCompact()
	}
}

func (b *Backend) takePending(id string) *Pending {
	if b.tr == nil {
		return nil
	}
	return b.tr.TakePending(id)
}

func (b *Backend) onPrompt(c core.CmdPrompt) {
	steering := c.Steer && b.turnActive
	settled := !steering && b.turnActive && b.stages.settled() // 上一回合的回复已出完,claude 还在收尾
	if !b.turnActive {
		b.turnActive = true
		if b.tr != nil {
			b.tr.Interrupted = false
		}
		b.turnT0, b.turnFirst = time.Now(), 0
		b.turnCold = b.p == nil
		b.emit(core.EvTurnStarted{})
		b.stages.begin(b.turnT0, b.turnCold)
		b.emitStage(nil)
	}
	// claude 不回显用户输入,这里补上时间线条目(引导消息同样)。先于进程启动发出:冷启动要几百毫秒,
	// 用户消息不能等进程起来才出现(界面不能有空白)。id 不依赖翻译器(翻译器在进程启动时才重建)。
	uuid := newUUID()
	uid := "lp" + uuid[:8]
	b.emit(core.EvAppend{Item: &core.User{ID: uid, Text: c.Text, Images: c.Images}}, core.EvUserUUID{ID: uid, UUID: uuid})
	if settled { // 记下发送时刻;result 到达后接力成下一回合(claude 串行处理,收尾结束才会轮到这条)
		now := time.Now()
		b.carry = append(b.carry, now)
		b.emit(core.EvTurnStarted{}, core.EvTurnPhase{Kind: core.StageQueued, At: now, TurnStart: now})
	}
	if !b.ensureProc() {
		return
	}
	if !steering && b.stages.connected(time.Now()) { // 冷启动的握手结束:「连接」→「本地准备」
		b.emitStage(nil)
	}
	b.write(UserMessage(c.Text, c.Images, steering, uuid))
	if !steering {
		log.Info("提示已写入 claude", "冷启动", b.turnCold, "写入耗时", time.Since(b.turnT0).Round(time.Millisecond), "pid", b.p.pid)
	}
}

// warmup 预热:用户还在输入框里打字时先把进程拉起来,点发送后就省掉进程启动与握手。
func (b *Backend) warmup() {
	if b.p != nil || b.turnActive {
		return
	}
	if !b.warmFailAt.IsZero() && time.Since(b.warmFailAt) < 60*time.Second { // 刚失败过:别每次打字都重试
		return
	}
	if err := b.startProc(); err != nil {
		b.warmFailAt = time.Now()
		log.Warn("预热 claude 失败(发送时会再试)", "err", err)
		b.emit(core.EvConn{State: core.ConnFailed, Why: err.Error()})
		return
	}
	log.Info("预热 claude 进程", "pid", b.p.pid)
}

// release 空闲回收:进程闲着时结束它省内存,会话保留(下次 --resume)。有回合 / 未完成请求 / 后台任务时拒绝。
func (b *Backend) release(why string) {
	if b.p == nil {
		return
	}
	if b.turnActive {
		log.Info("拒绝空闲回收:回合进行中")
		return
	}
	if n := len(b.reqs); n > 1 || (n == 1 && !b.hasReq(rkInit)) { // 只允许还在握手的 init 请求存在
		log.Info("拒绝空闲回收:还有未完成的请求", "n", n)
		return
	}
	if b.tr != nil && len(b.tr.liveTasks) > 0 {
		log.Info("拒绝空闲回收:还有后台任务在跑", "n", len(b.tr.liveTasks))
		return
	}
	uptime := time.Since(b.p.started).Round(time.Second)
	pid := b.p.pid
	b.restartConfig() // 已落盘就 --resume,没落盘(从没发过消息)就用同一个 session id 新开
	b.released = true
	b.killProc()
	log.Info("空闲回收 claude 进程", "pid", pid, "原因", why, "存活", uptime)
}

// turnTiming 回合结束时汇总本回合的实测耗时并清掉计时;不是本程序发出的回合(claude 自行开启的)返回 false。
func (b *Backend) turnTiming() (core.TurnTiming, bool) {
	if b.turnT0.IsZero() {
		return core.TurnTiming{}, false
	}
	t := core.TurnTiming{TotalMs: time.Since(b.turnT0).Milliseconds(), FirstMs: b.turnFirst.Milliseconds()}
	if b.turnCold && b.p != nil && !b.p.initAt.IsZero() {
		t.StartupMs = b.p.initAt.Sub(b.p.started).Milliseconds()
	}
	t.Phases, t.Calls = b.stages.finish(time.Now())
	b.turnT0, b.turnFirst, b.turnCold = time.Time{}, 0, false
	return t, true
}

// ---------- claude 输出 ----------

func (b *Backend) onMsg(m msg) {
	p := b.p
	switch m.kind {
	case mStderr:
		if len(p.stderrTail) >= 20 {
			p.stderrTail = p.stderrTail[1:]
		}
		p.stderrTail = append(p.stderrTail, m.text)
	case mEOF:
		b.onExit()
	case mInitTimeout:
		b.onInitTimeout()
	case mLine:
		b.onLine(m.text)
	}
}

func (b *Backend) onLine(line string) {
	p := b.p
	v := parseLine(line)
	if v == nil { // 非 JSON 行(例如启动横幅)当作诊断信息保留
		p.badLines++
		if p.badLines <= 3 || p.badLines%100 == 0 {
			log.Warn("claude 输出了非 JSON 行", "pid", p.pid, "n", p.badLines, "line", truncate(line, 120))
		}
		p.stderrTail = append(p.stderrTail, truncate(line, 200))
		return
	}
	ty := str(v, "type")
	switch ty {
	case "control_response":
		b.onControlResponse(v)
		return
	case "control_request":
		evs, reply := b.tr.ControlRequest(v)
		if reply != nil {
			b.write(reply)
		}
		b.emit(evs...)
		return
	case "control_cancel_request":
		if b.stages.answered(time.Now()) {
			b.emitStage(nil)
		}
		b.emit(b.tr.CancelRequest(str(v, "request_id"))...)
		return
	}
	// claude 自行开启的回合(排队消息、后台任务通知)也要让界面进入运行态
	if !b.turnActive && (ty == "stream_event" || ty == "assistant") {
		b.turnActive = true
		b.emit(core.EvTurnStarted{})
	}
	if changed, cache := b.stages.line(v, time.Now()); changed {
		b.emitStage(cache)
		if b.stages.settled() { // 最后一段回复出完:界面立刻结束,先给一份临时用时,result 到达再替换
			b.emitProvisionalTiming()
		}
	}
	evs := b.tr.Handle(v)
	if b.turnFirst == 0 && !b.turnT0.IsZero() && len(evs) > 0 && (ty == "stream_event" || ty == "assistant") {
		b.turnFirst = max(time.Since(b.turnT0), time.Millisecond)
		log.Info("收到第一个响应", "首响应", b.turnFirst.Round(time.Millisecond), "冷启动", b.turnCold, "类型", ty)
	}
	if ty == "result" {
		b.turnActive = false
		if tm, ok := b.turnTiming(); ok {
			log.Info("回合完成", "总耗时", time.Duration(tm.TotalMs)*time.Millisecond, "首响应", time.Duration(tm.FirstMs)*time.Millisecond,
				"启动握手", time.Duration(tm.StartupMs)*time.Millisecond, "阶段", tm.PhaseSummary(), "调用次数", len(tm.Calls), "缓存", callsCacheSummary(tm.Calls))
			b.emit(core.EvTurnTiming{Timing: tm})
		}
	}
	b.emit(evs...)
	if ty == "result" && len(b.carry) > 0 { // 收尾期间发的消息:紧接着开始它的回合,界面不留空档
		t0 := b.carry[0]
		b.carry = b.carry[1:]
		b.turnActive = true
		b.turnT0, b.turnFirst, b.turnCold = t0, 0, false
		b.emit(core.EvTurnStarted{})
		b.stages.begin(t0, false)
		b.emitStage(nil)
	}
}

// emitProvisionalTiming 最后一段回复出完时先给界面一份用时(到此刻为止),脚注立刻出现。
func (b *Backend) emitProvisionalTiming() {
	if b.turnT0.IsZero() {
		return
	}
	phases, calls := b.stages.snapshot(time.Now())
	t := core.TurnTiming{TotalMs: time.Since(b.turnT0).Milliseconds(), FirstMs: b.turnFirst.Milliseconds(), Phases: phases, Calls: calls}
	if b.turnCold && b.p != nil && !b.p.initAt.IsZero() {
		t.StartupMs = b.p.initAt.Sub(b.p.started).Milliseconds()
	}
	b.emit(core.EvTurnTiming{Timing: t})
}

func (b *Backend) onControlResponse(v obj) {
	resp := sub(v, "response")
	id := str(resp, "request_id")
	r, ok := b.reqs[id]
	if !ok {
		return
	}
	delete(b.reqs, id)
	success := str(resp, "subtype") == "success"
	errText := str(resp, "error")
	if errText == "" {
		errText = "未知错误"
	}
	body := sub(resp, "response")
	warn := func(what string) core.Event { return notice(core.NoticeWarning, what+"失败:"+errText) }

	switch r.kind {
	case rkInit:
		if b.p != nil && b.p.timer != nil {
			b.p.timer.Stop()
		}
		if b.p != nil {
			b.p.initAt = time.Now()
			log.Info("claude 握手完成", "耗时", b.p.initAt.Sub(b.p.started).Round(time.Millisecond), "pid", b.p.pid)
		}
		var models []core.ModelInfo
		for _, m := range arr(body, "models") {
			mm := asObj(m)
			val := str(mm, "value")
			if val == "" {
				continue
			}
			name := str(mm, "displayName")
			if name == "" {
				name = val
			}
			models = append(models, core.ModelInfo{ID: val, Name: name, Description: str(mm, "description")})
		}
		if len(models) > 0 {
			b.models = models
		}
		mode := core.ModeDefault
		if b.cfg.Mode != nil {
			mode = *b.cfg.Mode
		} else if m, ok := core.ParseMode(str(body, "current_permission_mode")); ok {
			mode = m
		}
		b.confirmedMode = mode
		evs := []core.Event{core.EvReady{SessionID: b.sessionID, Models: b.models, Model: b.model(), Mode: mode}}
		if !success {
			evs = append(evs, warn("初始化"))
		}
		if success {
			evs = append(evs, core.EvConn{State: core.ConnReady})
		} else {
			evs = append(evs, core.EvConn{State: core.ConnFailed, Why: errText})
		}
		b.emit(evs...)
	case rkSetModel:
		if success {
			b.emit(core.EvModelChanged{Model: r.model})
		} else {
			b.emit(warn("切换模型"))
		}
	case rkSetMode:
		if success {
			b.confirmedMode = r.mode
			b.emit(core.EvModeChanged{Mode: r.mode})
		} else { // 没切成功:提示原因,并把界面退回到仍然生效的那个模式
			m := b.confirmedMode
			b.cfg.Mode = &m
			b.emit(warn("切换权限模式"), core.EvModeChanged{Mode: m})
		}
	case rkMcpStatus:
		if ev, ok := b.tr.McpStatusEvent(body, success); ok {
			b.emit(ev)
		}
	case rkMcpToggle:
		if !success {
			log.Warn("mcp_toggle 失败", "err", errText)
		}
		b.refreshMcpStatus() // 不管成败都刷新一次:列表以后端实际状态为准
	case rkContext:
		if ev, ok := b.tr.ContextBreakdown(body, success); ok {
			b.emit(ev)
		} else {
			msg := strings.TrimSpace(errText)
			if success {
				if msg == "" {
					msg = "当前后端未返回上下文分类明细"
				}
			} else if msg == "" {
				msg = "控制请求失败"
			}
			u := b.tr.UsageSnapshot()
			u.ContextDetailErr = msg
			b.emit(core.EvUsage{Usage: u})
		}
	case rkRewind:
		o := ParseRewind(r.uuid, r.dryRun, success, body, errText)
		log.Info("回退文件", "dry", r.dryRun, "can", o.CanRewind, "files", len(o.Files), "err", o.Error)
		b.emit(core.EvRewind{Outcome: o})
	case rkResumeAt:
		b.finishResumeAt(success, errText)
	case rkUsage:
		b.emit(b.tr.AccountUsage(body, success))
	case rkOther:
		if !success && r.what == "中断" { // 中断失败时撤销取消标记,避免下一个回合被误判为已取消
			b.tr.Interrupted = false
		}
		if !success {
			b.emit(warn(r.what))
		}
	}
}

// ParseRewind 解析 rewind_files 的应答。已对真实 claude 验证:预览成功带 canRewind / filesChanged /
// insertions / deletions;执行成功只有 canRewind;没有检查点时 canRewind=false 带 error
// (预览仍是 success 应答,执行是 error 应答)。
func ParseRewind(uuid string, dry, ok bool, body obj, errText string) core.RewindOutcome {
	if !ok {
		return core.RewindFailed(uuid, dry, errText)
	}
	o := core.RewindOutcome{
		UUID: uuid, DryRun: dry, CanRewind: boolOf(body, "canRewind", false),
		Insertions: int(u64(body, "insertions")), Deletions: int(u64(body, "deletions")), Error: str(body, "error"),
	}
	for _, f := range arr(body, "filesChanged") {
		if s, ok := f.(string); ok {
			o.Files = append(o.Files, s)
		}
	}
	return o
}

func (b *Backend) tailText() string { return strings.Join(b.p.stderrTail, "\n") }

// onInitTimeout 握手计时到点:仍未应答则提示原因、结束回合并关闭进程(下一条提示时重新启动)。
func (b *Backend) onInitTimeout() {
	if _, waiting := b.reqs["req_init"]; !waiting {
		return
	}
	tail := b.tailText()
	log.Warn("claude 握手超时,结束进程", "pid", b.p.pid, "stderr", tail)
	text := "claude 启动超时:没有响应。请在终端运行 claude 确认能正常启动(登录、信任目录、网络或代理)。"
	if tail != "" {
		text += "\n" + tail
	}
	if b.turnActive {
		b.turnActive = false
		b.stages.off()
		b.carry = nil
		b.emit(core.EvTurnEnded{Reason: core.StopError})
	}
	b.emit(notice(core.NoticeError, text))
	b.killProc() // 进程随后被 kill,它的退出不再另报一次
	b.emit(core.EvConn{State: core.ConnFailed, Why: "握手超时"})
	b.restartConfig()
}

func (b *Backend) onExit() {
	p := b.p
	if p.timer != nil {
		p.timer.Stop()
	}
	_ = p.stdin.Close()
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill() // stdout 已关,进程若还在也不再有用
	}
	err := p.cmd.Wait()
	code, sig := 0, ""
	switch e := err.(type) {
	case nil:
	case *exec.ExitError:
		code = e.ExitCode()
		if ws, ok := e.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			sig = ws.Signal().String()
		}
	default:
		code = -1
	}
	tail := p.stderrTail
	if len(tail) > 5 {
		tail = tail[len(tail)-5:]
	}
	log.Warn("claude 进程退出", "pid", p.pid, "code", code, "signal", sig, "uptime", time.Since(p.started).Round(time.Second),
		"turn", b.turnActive, "stderr", strings.Join(tail, " | "))
	text := "claude 进程已退出"
	switch {
	case sig != "":
		text = "claude 进程被信号终止(" + sig + ")"
	case code != 0:
		text = fmt.Sprintf("claude 进程异常退出(退出码 %d)", code)
	}
	if t := b.tailText(); t != "" {
		text += ":\n" + t
	}
	close(p.stop)
	b.p = nil
	b.restartConfig()
	if b.turnActive {
		b.turnActive = false
		b.stages.off()
		b.carry = nil
		b.emit(core.EvTurnEnded{Reason: core.StopError})
	}
	b.emit(notice(core.NoticeError, text), core.EvConn{State: core.ConnNone, Why: "进程退出"})
}

// mergeManagedEnv 用 next 替换模型映射管理的那几个环境变量(见 claudecfg.ManagedEnvKeys),其余原样保留。
func mergeManagedEnv(cur, next []core.EnvVar) []core.EnvVar {
	managed := map[string]bool{}
	for _, k := range claudecfg.ManagedEnvKeys() {
		managed[k] = true
	}
	out := make([]core.EnvVar, 0, len(cur)+len(next))
	for _, e := range cur {
		if !managed[e.Key] {
			out = append(out, e)
		}
	}
	return append(out, next...)
}

// emitStage 向界面报告当前阶段(阶段切换时调用)。
func (b *Backend) emitStage(cache *core.CacheTokens) {
	b.emit(b.stages.live(time.Now(), cache))
}

// stageAnswered 权限 / 提问 / 计划被作答:「等你确认」结束,回到请求到达前的阶段。
func (b *Backend) stageAnswered() {
	if b.stages.answered(time.Now()) {
		b.emitStage(nil)
	}
}

// callsCacheSummary 日志用:每次调用的缓存结论,`#1 缓存写入 26.2k(首次) #2 缓存命中 98%`。
func callsCacheSummary(calls []core.CallInfo) string {
	var sb strings.Builder
	for i, c := range calls {
		if i > 0 {
			sb.WriteByte(' ')
		}
		t := c.Text()
		if t == "" {
			t = "无缓存字段"
		}
		fmt.Fprintf(&sb, "#%d %s", i+1, t)
	}
	return sb.String()
}
