# MiniCode 开发笔记(归档)

> 这是原来塞在 `AGENTS.md` 里的全部内容:各功能的实现要点、踩坑记录、已知缺口。**不会被自动加载进 Claude 的上下文**,要查某个功能的历史做法或踩过的坑时再翻(搜关键字即可)。日常必须遵守的规则只在 `AGENTS.md`(精简版)。

---

# MiniCode 开发约定
> **冲突裁决:本文件与 `docs/design` 设计稿不一致时,一律以设计稿为准**,并同步修改本文件。已知以设计稿为准的点:窗口断点 1040 / 900 / 720、窗口最小宽 720、IDE 状态栏不放通知铃铛(铃铛只在标题栏)、IDE 对话栏收成图标条只有一个「对话」气泡图标(没有机器人)。
> **分支 `feat/mygo`:界面用** [**MyGo**](https://mygo.run/zh)**(Go 原生 UI,无 WebView),代码在 `mygo/`。** 旧 Rust 实现(`crates/`、`vendor/`、Cargo 配置、CI)已从本分支删除,需要时从 git 历史取(删除前最后一个提交的父提交是 `ddd7452`)。
## 开发工作流(强制:按「补缺口 → 逐项审查 → 集中测试(computer use 实测)」三阶段走,不能跳步)
1. **先看设计稿**:动手前先读 `docs/design` 里对应的页面(入口见 `js/app.js` 的 `PAGES`)。设计稿没有、或缺状态 / 右键菜单 / 响应式 / 动效 / 空态 / 错误态 / 演示数据时,**先补设计稿的 UI**(`docs/design` 先改),并在页面的「实现状态」列如实标注,然后才写代码。不允许先写功能、事后补稿。
2. **集中补缺口(阶段一)**:先不跑测试、不截图。从设计稿「缺口总表」(`js/status.js`)集中盘点所有缺口,列成清单,然后尽快补齐;缺口互相独立时**用子代理并行补**(各管不同文件,避免冲突)。演示模式(`internal/demo`)必须有覆盖该功能各种状态的模拟数据(正常 / 空 / 多项 / 失败 / 进行中);入口、出口、取消、撤销都要有,不留「只进不出」。
3. **逐项审查(阶段二)**:每补完一个缺口,就**对着设计稿从代码逻辑上审查一遍**(状态、文案、菜单项、置灰原因、空态 / 错误态、可观测性清单),没问题再补下一个。这一步只做静态审查与 `gofmt` / `go build ./...` 保证能编译,**不跑 `go vet` / `go test`、不出截图、不重启开发版**。**每改完一处代码就立刻提交并 `git push origin HEAD`(中文 Conventional Commits,一个功能一个提交),不等测试、不等「全部补完」**;提交不以 `go vet` / `go test` 为前提,测试阶段发现的问题到时候再单独修复、再提交一个 `fix:`。提交前至少 `gofmt`,尽量保证 `go build` 通过;工作区里有别人 / 别的子代理尚未完成的文件时,只按路径 `git add` 自己改的文件,不要因为别处编不过就不提交,也不要顺手提交别人的半成品。设计稿「实现状态」列与 `status.js` 同步更新,并随同一个提交。
4. **集中测试(阶段三,只在用户明确说「开始测」时才做)**:全部缺口补完后,由用户手动通知才进入。**所有测试一律用 computer use 在真实运行的开发版里实测**,这是验收的唯一依据:
   1. 先 `cd mygo && unset GOROOT && gofmt -l . ; go vet ./... && go test ./...` 全绿(纯逻辑的回归单测照常保留,每个新行为补 `go test`,界面用 `ui.Tester`),再 `scripts/dev.sh [--demo|--real]` 重启开发版(始终只留一个实例)。
   2. 用 computer use 操作 `MiniCode Dev` 窗口:按设计稿逐项走一遍——**真点击、真输入、真快捷键**(入口、出口、取消、撤销、右键 / ··· 菜单、空态 / 错误态 / 进行中、窄档与宽档、浅色 / 深色),每个行为至少截一张图;同时在浏览器里打开 `docs/design/index.html` 对应页面,与之对照间距、图标、文案、状态、动效。差异写成清单并补齐,补完再测一遍。
   3. `cmd/shot` 的无窗口截图与 `go test` 都**不能代替** computer use 实测,只作辅助排查;没有用 computer use 走过的功能,一律写「未测试」。
   4. 测试中发现的问题,单独 `fix:` 提交并推送,再用 computer use 复测。**没有用户的通知,不自动测试、不自动截图。**
   5. **修复后立即重启开发版(用户已授权的常规动作)**:每次修完一处问题(`gofmt` + `go build` 通过、已提交并推送)就直接 `scripts/dev.sh` 重启开发版,让用户马上看到效果,不用再问;这只是重启,不等于测试——仍不自动用 computer use 操作、不自动截图。重启前要确认编译通过(`dev.sh` 会先跑 `go vet`,失败时旧版本继续运行,先修好再重试)。
   6. **工具使用**:读文件用 Read、改文件用 Edit / Write、搜索用 Grep / Glob(不可用时才退回 Bash 里的 `grep`);Bash 只用来跑 `go build` / `gofmt` / `git` / `scripts/dev.sh` 这类必须走命令行的事,不要用 `sed` / `cat` / Python 脚本代替读写文件(脚本替换没有「先读后改」的校验,出过把文件截断的事故)。
5. **汇报要诚实**:列出已实现 / 部分 / 未做,不把「演示数据」说成「真实能力」;还没测过的明确写「未测试」,测过的写明是否「已用 computer use 实测」及走过的操作路径;已知缺口同步写进下面的「已知缺口」与设计稿「实现状态」列。

### 可观测性清单(方便后期排查,新增代码必须满足)
- **日志**:一律 `diag.L("包名")`,不用 `fmt.Println` / `log`。用户动作(打开 / 切换 / 提交 / 安装 / 跳转)写 `Info` 并带关键字段(路径、仓库、会话 id、耗时);降级与可恢复错误写 `Warn`;不可恢复写 `Error`;逐帧 / 高频事件只写 `Debug`。不输出密钥与环境变量值,只写变量名。
- **性能**:新增的耗时操作(磁盘遍历、git / LSP / 子进程、解析、大列表构建)用 `diag.SpanSlow(模块, 名称, 阈值)` 包起来;热路径统计用 `diag.Span` / `diag.NewCounter`;界面构建超过 33 ms 会自动记慢帧。**所有后台任务经 `a.bg(name, work)`**(内置 `diag.Recover`,崩溃不拖垮窗口)。
- **外部进程 / IO**:启动、退出码、耗时、stderr 摘要都要记录;失败时界面给出可行动的提示(下一步按钮),日志给出原因。
- **崩溃与卡死**:用 `diag.Guard` / `diag.Recover` 包住新的 goroutine;界面线程卡死由看门狗记录,不要在界面线程里做阻塞 IO。
- **可复现**:新增界面状态要能用 `cmd/shot` 参数或 `ide_debug.go` 的 `-ed` 动作直接到达,便于截图与回归;调试开关走环境变量并写进本文件。
- 排查入口:`~/.minicode/logs/minicode.log`(日志)、`~/.minicode/crashes/`(崩溃与卡死记录)、设置 → 隐私与诊断(诊断包 / 资源曲线 / 慢操作统计)。

## MyGo 重写(`mygo/`)

Go 1.27、`github.com/egoist/mygo` v0.9.0,模块名 `github.com/ixxxxoooo/minicode`。界面对标 `docs/design` 设计稿(Agent / IDE 是同一个窗口的两种形态,共用一份会话 / 编辑器 / 终端)。

| 包 | 职责 |
| --- | --- |
| `internal/core` | 与界面无关的数据模型:时间线条目、`Event` / `Command`、`State.Apply` 归约器、模式 / 用量 / 回合统计 |
| `internal/backend` | `Backend` 抽象(`Start(sink)` / `Send(cmd)` / `Close()`) |
| `internal/claude` | 真实后端:stream-json 协议(`protocol.go`)、翻译器(`translate.go`)、惰性启动与崩溃自动 `--resume` 重启(`backend.go`)、历史回放(`history.go`)、会话索引与增量磁盘缓存(`index.go`) |
| `internal/demo` | 演示后端:脚本化场景、沙盒项目(`sandbox.go`)、演示仓库 |
| `internal/vcs` | Git 访问(`CLI` 走 git 命令行、`Mem` 供测试 / 演示),所有读命令带 `--no-optional-locks` |
| `internal/workspace` | 文件系统抽象(`OS` / `Mem` / `Multi`),IDE 文件树与搜索读它;`Multi` 把多个文件夹拼成一棵树(顶层 = 文件夹名,重名加后缀) |
| `internal/projects` | 手动添加的项目与工作区存储(`~/.minicode/workspaces.json`,原子落盘,损坏改名留底):校验、去重、`ByPrimary`、`Extra()`(给 `--add-dir`) |
| `internal/lsp` | 语言服务器客户端内核(不依赖界面):Content-Length 帧、JSON-RPC 配对、服务端请求自动应答、UTF-16 ↔ 字节列换算(`position.go`)、按扩展名选服务器与查找安装(`servers.go`,rust-analyzer 跳过 rustup 代理)、Hover / Definition / References / Completion / Rename / Formatting、`ApplyEdits` |
| `internal/term` | 终端内核:`creack/pty` 起登录 shell + `charmbracelet/x/vt` 仿真器,输出画面快照(样式段、光标、历史),不依赖界面 |
| `internal/app` | 全部界面:`app.go` 状态与事件、`shell.go` 外壳、`chat/flow/composer/dock` 会话、`ide*.go` IDE、`term_panel.go` 终端面板、`project.go` 真实项目上下文、`scm*.go` 源代码管理、`settings*.go` 设置、`palette.go` 命令面板等 |
| `internal/kit`、`internal/theme` | 控件与设计令牌(6 套主题 × 3 种布局) |
| `internal/diag` | 日志(`~/.minicode/logs/minicode.log`)、慢操作计时、崩溃与卡死看门狗 |
| `cmd/shot` | 无窗口截图(`go run ./cmd/shot -scheme a -theme paper -mode agent -screen work -o /tmp/x.png`),与 `/tmp/proto-*.png` 的设计稿截图对照 |

数据流:`Backend` → `core.Event` → `core.State.Apply` → 界面;界面交互转成 `core.Command` 交回后端。演示与真实后端共用同一套界面。

### 设计稿 v3.1 的区域模型(`docs/design`,Go 实现以此为准)

- 窗口只有四块区域:**导航栏**(会话 / 文件 / 搜索 / 源代码管理)、**对话栏**、**工作区**(编辑器标签组 + 底部面板)、**状态栏**。**没有「检查器」**,每个内容只有一个「家」,别处出现的同一内容只是跳转。
- Agent 与 IDE 只是对话栏与工作区谁在主位:Agent = 对话居中、工作区在右侧(默认收起,开标签时自动展开并闪一下边框);IDE = 工作区占主区、对话栏收到右侧 352 宽。会话 / 编辑器 / 终端全窗口各一份。
- 计划、子 Agent、后台任务输出、本轮审查、差异、合并冲突都是工作区的**虚拟标签**(不落盘);入口:会话流卡片的「查看 / 审查」、任务坞列表、命令卡「在终端面板查看」、SCM 单击文件(预览标签)。
- 对话栏按**自己的宽度**三档响应(CSS 容器查询):宽 ≥640 / 中 420–639 / 窄 <420;窄档头部带会话下拉、新会话、历史、占满;工作区三档:宽 ≥600 / 中 520–599(隐藏迷你地图)/ 窄 <520(强制统一 diff、隐藏面包屑)。
- 编辑器能力清单见设计稿 `ide-nav`(迷你地图与概览标尺高亮、查找替换、悬停 / 预览定义 / 引用、补全、代码操作、重命名、问 Claude、行内评审、前进后退),Git / Diff 清单见 `ide-git`(块级暂存 / 还原、行内左右切换、词级高亮、三方冲突、本轮审查、文件历史 / 追溯、储藏、图谱右键);总览见 `flows` 与 `resp` 页。

### 运行与开发流程

```sh
cd mygo && unset GOROOT          # 每次执行 Go 命令前(环境里残留的 GOROOT 会让工具链对不上)
gofmt -l . ; go vet ./... && go test ./...
scripts/dev.sh [--demo|--real]   # 编译成功后杀旧实例并在 target/dev/MiniCode Dev.app 里启动唯一的新实例
```

- 运行模式优先级:`--demo` / `--real` / `MINI_DEMO=1|0` > 开发版记住的上次选择(`~/.minicode/dev-mode`)> 真实。开发版左下角徽章:演示模式显示「演示模式」,真实模式显示「开发版」。
- 调试环境变量:`MINI_SHORTCUT_DEMO=1`(演示模式预置「快捷键冲突」假数据;默认不写,因为演示与真实共用 `~/.minicode/settings.json`,会污染真实偏好)、`MINI_TRAY=waiting|error|unread|update|paused|running|demo|idle`(固定菜单栏图标状态,用 `open -n "target/dev/MiniCode Dev.app" --env MINI_TRAY=… --env MINI_DEV=1` 启动后 `screencapture` 看菜单栏;裁剪区域在屏幕右半部分顶部 70px)、`MINI_MODE=ide`(启动进 IDE)、`MINI_PROJECT=<目录>`(真实模式的初始项目)、`MINI_THEME` / `MINI_SCHEME`、`MINI_SCREEN`、`MINI_WIN_W/H`、`MINI_DEMO_SLOW`、`MINI_MOTION_SCALE=8`(把所有动效时长放慢 8 倍,方便逐帧截图核对形态切换 / 开合 / a-swap;1–50,默认 1)、`MINI_ANIMLOG=1`(每帧把导航栏 / 舞台 / 对话栏 / 工作区 / 底部面板的 X+宽 写进 `~/.minicode/logs/minicode.log` 的 `[anim]` 模块;不设时只在动画结束写一行汇总,异常帧写 Warn)、`MINI_CLAUDE_PATH`;真实 claude 集成测试用 `MINI_REAL_CLAUDE=1 go test ./internal/claude -run Real`。
- **真实场景测试用 DeepSeek**:`scripts/with-deepseek.sh <命令…>` 运行时从 `~/.minicode/provider-keys.json`(0600,仓库之外)读取 DeepSeek 密钥,注入 `ANTHROPIC_BASE_URL=https://api.deepseek.com/anthropic` 等环境变量后执行命令(默认模型 `deepseek-chat`,`DEEPSEEK_MODEL=deepseek-flash|deepseek-v4-pro` 带思考)。例:`scripts/with-deepseek.sh scripts/dev.sh --real`、`cd mygo && MINI_REAL_CLAUDE=1 ../scripts/with-deepseek.sh go test ./internal/claude -run Real`。**密钥只存在那份本机文件里,不进仓库 / 日志 / 文档 / 提交;脚本与文档里不出现密钥值。**
- 提交:中文 Conventional Commits,一个功能一个提交,**改完立刻提交并 `git push origin HEAD`**(本分支推到 `feat/mygo`),不等 `go vet` / `go test`;测试出问题再修、再提交 `fix:`。不提交 `target` / `dist`;不读取、不输出密钥,环境变量只展示名称。
- 日志一律用 `diag.L("包名")`,不用 `fmt.Println` / `log`;新增耗时操作用 `diag.SpanSlow` 包起来。

### MyGo 使用要点(踩过的坑)

- `ButtonBase` 与 `kit.RowEl` 默认居中且固定高度;`c.Key(...)` 要在构造输入控件之前;`Element.Bounds()` 是**上一帧**的矩形(第一帧为空);`ui.ScrollState` 用 `TrackScroll(&st)` 绑定,滚动会被夹在内容范围内,`st.Y = math.MaxFloat32` = 滚到底。
- 异步一律 `a.bg(name, work)`:后台线程算完返回「应用结果」闭包,经 `a.win.Update` 回主线程执行;`a.win == nil`(测试、启动阶段)时同步执行。后端事件同理,且会丢弃已被替换的旧后端的事件。
- `go vet` 不接受其他包结构体的无键字面量,一律写键名。
- 截图对照:Read 工具会缓存图片,每次换新文件名。

### 真实模式的数据来源

- 会话:`claude.BuildIndex` 扫 `~/.claude/projects`(增量 + 磁盘缓存 `session-index-go.json`),侧栏每 20 秒重扫;打开会话 = 关掉旧后端(结束它的 claude 进程)、`claude.New` 造惰性后端(第一次发消息才启动进程)、回放历史。
- 项目:打开会话时 `App.SetProject(cwd)` 切项目——IDE 文件树读磁盘(`workspace.NewOS`)、仓库换成 `vcs.Open(cwd)`(非仓库则分支为空)、旧终端结束。新会话的目录 = 最近打开的会话所在目录(`MINI_PROJECT` 优先)。
- 终端:IDE 底部「终端」面板首次显示时在项目目录起 `$SHELL -l`;面板量出列 / 行数后调 `Resize`;按键经 `uv.KeyPressEvent` 交给仿真器编码(应用光标键等);输入法组字显示在光标处(组字期间的按键留给输入法);shell 退出后按回车重开;切项目与退出时结束整个进程组。

### 工作区与 Git 的实现要点

- **工作区**(`workspace.go` / `ws_tabs.go`):编辑器标签组 + 底部面板;虚拟标签 `vReview / vPlan / vSub / vTask / vConflict`,id 形如 `v:<kind>[:ref]`,入口都走 `openVirtual`,开标签时工作区自动展开并闪一下边框(`flashOverlay`)。对话栏三档、工作区三档按**自己的宽度**判断。窗口最小宽 720(以设计稿 `resp` 页为准)。
- **编辑器**:符号 / 引用索引(`symbols.go`,本地正则 + 导航历史,不是 LSP)、迷你地图与概览标尺(`ide_overlay.go`)、查找替换、悬停 / 预览定义 / 引用 / 重命名 / 代码操作、问 Claude、行内评审、前进后退(`ide_nav.go`)。
- **Diff**(`ide_diff.go`):三种标签 `HEAD ↔ 工作区` / `暂存区 ↔ 工作区` / `HEAD ↔ 暂存区`,块级暂存 / 还原(`StageContent`,悬停显示块头动作,这是 MyGo `DiffState.HunkActions` 的行为)、行内 / 左右切换、词级高亮。**合并冲突**(`conflict.go`):`parseConflicts` / `composeConflicts`,逐处「接受当前 / 传入 / 双方」,全部处理后「标记为已解决」写盘并暂存;调试 `-ed conflict`、命令面板「演示:打开合并冲突」。
- **Git**(`vcs`):除状态 / 暂存 / 提交外有 Blame、FileLog、Stash(推入 / 应用 / 弹出 / 删除)、Checkout、CherryPick、Revert、CreateAt、UndoCommit;`Mem` 给演示与测试。行内追溯 `ide_git.go`(按「路径@HEAD」缓存)、文件时间线、SCM 储藏区与提交右键菜单、「审查本轮改动」入口。
- **后台保活**:`App.live` 里停放正在运行的非演示会话,切走不杀进程,回合结束再关;重新打开时接回。新会话按项目开(`NewSessionIn`,id `new:<路径>`);模型 / 思考强度选择写偏好(`claude.model` / `claude.effort`)。
- **用量**:`claude.UsageScanner` 增量扫 `~/.claude/projects`(按文件大小 + 修改时间缓存,按 message.id 去重,价格表在 `UsageRec.Cost`),main.go 每分钟后台重扫;**账户限额(`get_usage`)与上下文分类明细仍无数据,界面如实说明**。
- **自动化**:`automation.OpenStore`(`~/.minicode/automations.json`,每次修改原子落盘),`claude.RunPrompt` 以 `claude -p` 运行,main.go 每 30 秒 `AutomationTick` 触发到期项,「立即运行」同一路径。
- **设置驱动行为**(`prefs_apply.go`,键与默认值必须与 `settings_tabs.go` 一致):动效开关(`kit.SetMotion` 把时长令牌清零)/ 过渡 / 流光、编辑器字号 / 缩进 / 自动保存(延时 1.5 s 或失焦)/ 迷你地图 / 行号 / 折叠 / 换行 / 行内追溯、终端字号、思考默认展开 / 回合统计 / 探索组折叠、发送键(Enter 或 ⌘Enter)与「回合中再发」(排队或引导)、内容宽度。其余设置页控件仍只存偏好。
- **动效**:弹层入场(`popIn`:透明度 + 4px 升起,120 ms)、居中浮层(`dialogIn`:遮罩淡入 + 卡片落下放大,180 ms)、步骤行展开(淡入 + 下落)、工作区 / 对话栏宽度过渡(440 ms)、边框闪、旋转 / 呼吸 / 光标闪烁、开关滑块与标签下划线弹簧、悬停颜色过渡;`appearance.motion` 关闭或系统「减弱动态」时全部直达终态。需要持续帧的入场动画用 `frameSoon()`。
- **本地化**:`main.go` 与 `cmd/shot` 都 `import _ ".../locales/zh-Hans"` 并 `SetLocale("zh-Hans")`,控件自带文案(差异折叠条、菜单)才是中文。**坑**:应用自己没有翻译目录时,库默认「支持的语言」只有英文,单独 `SetLocale` 会被静默忽略(真实应用里菜单仍是 Copy / Select All,而 Tester 里是中文);`main.go` 必须先 `mygo.App.SetLanguages("zh-Hans")` 再 `SetLocale`。

### 新增模块与约定(v3.2 补充)

- **语言服务**(`lsp_app.go` / `lsp_problems.go` / `lsp_actions.go`):每个「项目根 + 服务器」一个会话,串行队列 + `barrier()`,编辑后去抖发整篇 `didChange`;诊断进编辑器波浪线、问题面板、状态栏;未安装时发一条带安装命令的提示,「安装」在可见终端里预填命令(`termPanel.initCmd`);失败退回本地符号索引。测试里 `a.win == nil` 走 `onUI` 队列 + `drainUI()`。真实 gopls 的端到端测试在 `lsp_app_test.go`(没装 gopls 自动跳过)。
- **终端**(`term_sel.go`):鼠标选择 / 复制、⌘F 搜索条、⌘点击超链接(URL 与 `path:line:col`)、字体 / 光标设置弹层;颜色一律取当前主题令牌,不单独切换。无鼠标上报。
- **Git**:`scm_ai.go` AI 提交说明(`claude -p` Haiku,失败退回启发式 `draftCommitMsg`);`scm_graph.go` 图谱(整页虚拟标签 `vGraph`,`-ed graph[:hash]` 调试);`ide_git.go` 时间线(顶部「本地更改」条目 + 提交行右键菜单)。
- **布局**(`layout.go`):左 / 右栏与底部面板分隔线可拖,双击还原;面板拖到顶部吸附成「占满」,拖离还原;对话栏「占满」有明确出口。
- **输出面板**(`ide_output.go`):`outApp / outGit / outLSP` 三个通道,语言服务与 Git 操作写入。
- **彩色文件图标**:`ide_tree.go` 的 `fileTint` / `fileIc`,偏好 `appearance.fileIcons`(默认开)。
- 设计稿 `docs/design` v3.2 新增五页(`js/specs2.js`):布局 · 拖拽与面板停靠、终端 · 能力与主题、语言服务 · 识别与安装、画布 / 浏览器 / 自定义 · 业界做法、会话流 · 待办 / 审批 / 后台任务;每页都有「实现状态」列,与代码保持一致。
- 设计稿 `docs/design` v3.3 新增「图标与品牌 · 应用 / 托盘」页(`js/brand-spec.js`,规范组首项):图标生成器 `js/brand.js`(`U.brand`:五种应用图标样式(默认「提示符」,织纹已删除)× 浅 / 深 / 透明 / 着色外观 × 六套主题配色,四种菜单栏模板图标 × 八种状态 × 彩色徽标,「经纬」按「压 / 穿」切段、不靠背景色挖洞)、样式 `css/brand.css`、交互 `js/brand-act.js`;设置 → 外观新增「应用图标 / 菜单栏图标」两组(`settings.js::brandGroups`),选择实时刷新全稿里的图标(Dock、⌘Tab、通知、关于、查看器标题栏等)。Go 分支尚未实现 Dock 图标换装与托盘,实现状态表如实标注;`assets/icon/` 目前只有浅 / 深两套 `.icns` 与四张托盘 PNG。

### 项目与工作区、多仓库(v3.3 补充,设计稿 `proj-ws` / `multi-git` / `menus` / `gaps` 页)

- **概念**:项目 = 一个文件夹;工作区 = 一组命名文件夹(第一个是主文件夹 = 会话 cwd,其余经 `--add-dir` 授权,`LoadMD` 开时设 `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1`);仓库 = 文件夹里的 Git 仓库(可没有)。数据 `App.reg *projects.Store`(真实落盘,演示 `projects.NewMem()` + `seedDemoRegistry`)。
- **侧栏**(`side_ws.go`):标题行「+」(添加项目… / 新建工作区…);工作区行置顶,展开 = 只列主文件夹的会话(成员文件夹不占行,在「编辑工作区」里管理);成员文件夹不再单独列;手动添加但没有会话的项目补在列表里。工作区 / 项目两种行有右键菜单(工作区 = 新对话 · 编辑工作区… · 添加文件夹… · 上移 / 下移 · 访达 / 复制 · 移除;项目 = 新对话 · 编辑项目…(`proj_edit.go`,显示名存 `workspaces.json` 的 `names`)· 加入工作区 · 访达 / 复制 · 移除并确认);全应用右键菜单经 `App.View` 的 `SetMenuPresentation(ui.PopoverMenus)` 画在窗口内,图标用 `kit.MenuIcon`。吸顶的用户消息卡背景必须不透明(`Fill1.Over(BG)`),固定时抬阴影 + 描边 + 下沿渐隐(`flow_user.go`)。方案 B 的项目下拉暂未接工作区。
- **对话框**(`proj_dialogs.go`):添加项目(路径输入 + 浏览 + 最近 / 发现 + 校验:找不到 → 相近路径建议、是文件 → 用所在文件夹、重复 → 切换过去)、新建 / 编辑工作区(≥2 个文件夹、设为主、移出、从已有项目添加、路径 / 浏览、CLAUDE.md 开关、权限提示)、通用确认框。系统文件夹选择器 `mygo.Dialog.Open`,演示模式没有。入场动画按**真实时钟**计时,截图 / 测试要 `time.Sleep` 一下再 `Frame`。
- **多根**(`workspaces.go`):`ActivateWorkspace` 把 `a.ide.fs` 换成 `workspace.Multi`,`a.repos` 每个文件夹一项(`repoEntry`:分支 / 领先落后 / 最近状态 / 提交说明草稿);**`a.repo` / `a.sc` 始终是「活动仓库」的**,`setActiveRepo` 切换并重读;文件树路径 = `文件夹名/相对路径`,`a.treePath(rel)` 是仓库内 → 树路径,`a.repoAt(p)` 是树路径 → 仓库 + 相对路径(HEAD / Index / 追溯 / 暂存块都走它),`a.focusRepoOf(p)` 在会改仓库的操作前切到该文件所属仓库;`syncTreeGit` 合并所有仓库的改动(带前缀);非活动仓库每 8 秒后台轻量刷新(`reposTick`)。真实模式 `a.proj = 主文件夹`,每个文件夹后台探测是不是仓库;非仓库在总览里有「初始化仓库」(`vcs.Init`)。`leaveWorkspace` 还原进入前的单项目现场;打开不属于成员的会话会自动退出(演示在 `OpenSession`,真实在 `SetProject`)。
- **源代码管理**(`scm_repos.go`):多根时顶部「仓库」总览(分支 · ↑↓ · 改动数,点击切换活动仓库,全部拉取 / 同步),提交框上方写明「提交到 <仓库> · <分支>」,状态栏显示「仓库 · 分支」。
- **调试**:`MINI_WS=1` 启动即进入演示工作区;`cmd/shot -ws [-repo N]`、`-dlg add|ws|wsedit|confirm|add:<路径>`。演示数据 `internal/demo/wsdemo.go`(`WorkspaceFolders` / `Discovered`)。
- **多仓库 B1 补充**(设计稿 `multi-git` 页):审查页按仓库分组 + 「逐个提交」向导(`review_repos.go`:`reviewGroups` / `commitWizard`,提交说明按仓库预填,`wizardAfterCommit` 串起下一个);图谱标题行与分支选择器的仓库下拉 / chip(`repo_pick.go` 的 `folderPicker`,切换 = 切换活动仓库);搜索「范围」chip(`searchState.scope`,`Multi.Only(i)` 限定文件系统,文件夹右键「仅在此文件夹搜索」,切工作区清除);总览行(`scm_repos.go`)有悬停动作、右键菜单、逐行转圈 / 失败重试 / 冲突徽标 / 已暂存圆点,`syncAllRepos` 顺序执行并跳过有冲突的仓库;`vcs.Mem.SetFail / SetDelay` 给演示造失败与慢操作。**嵌套仓库 / 子模块**(`repo_nested.go`):`repoEntry.kind`(子模块 / 嵌套)、`prefix`(树路径前缀)、`parent`,追加在根条目之后;`repoIndexOf` 按最长前缀归属(最近的 `.git`),`repoOrder` 让总览里嵌套行缩进在父文件夹后面;真实模式 `discoverNested` 后台向下 2 层找 `.git`(`findNested`,读 `.gitmodules` 区分子模块)。调试 `-ed`:`review` / `wizard[:1|done]` / `pop:<key>` / `search:<q>` / `scope:<文件夹>` / `scm` / `syncall` / `repobusy:N` / `expand:<树路径>[;…]`。缺口:「子模块有未提交改动」提示行与父仓库的指针变更条目;嵌套仓库没有「在父仓库里排除」的提交范围控制;真实模式的嵌套发现只有单测,没有对真实多仓库工作区端到端验证。
- **会话行整理(B2)**(`session_meta.go`,`internal/sessmeta`,设计稿 `menus`「会话行」一节):置顶 / 归档 / 自定义标题存在 `~/.minicode/session-meta.json`(原子落盘,损坏改名留底;演示用内存并预置 `files` 置顶、`axum` / `long` 归档),不改 Claude 的会话文件。`a.rawProj` 是索引原样,`a.projects = applyMeta(rawProj)`(标题已套用、带 `Pinned / Archived`,时间顺序不变),置顶只在侧栏渲染时提前(`sessionsSplit`),归档的收进项目底部「已归档 N」折叠行(`projOpen["arch:"+项目 id]`);行内重命名(`a.ren`,⏎ 保存 / Esc 或点别处取消 / 与原标题相同或留空 = 恢复);删除走确认框 → `a.sessDel`(真实 = `claude.DeleteSession`,经 `SessionDeleter` 按索引找目录;演示只改内存)→ `dropSession` 清整理信息并切到同项目下一个会话,运行中的会话置灰;「复制为 Markdown」(`core.ExportMarkdown`)只对已打开的会话可用。调试 `-ed rename:<会话 id>` / `archived:<项目 id>`。缺口:「分叉会话」「复制会话链接」后端未接,置灰;历史会话选择器与命令面板仍列出已归档的会话。
- **拖拽排序(B2)**(`side_ws.go` 末尾,设计稿 `proj-ws`「拖拽排序」):工作区行与成员文件夹行是 `wsDrag` / `memDrag` 拖拽源 + 落点(`ui.Drag` / `ui.Drop` / `ui.DragOver`,经过的行顶部 2px 插入线),松手 = 放到那一行之前;`Store.MoveWorkspace` / `projects.MoveBefore` 落盘;主文件夹(第一个)不参与换位,拖到它上面提示;右键菜单有等价的「上移 / 下移」;使用中的工作区换序后经 `wsMutate` 重新进入使文件树生效。项目行按最近活动排序,不手动排。演示新增第二个工作区「文档与素材」(`ws-content`)。
- **未做**:见设计稿「缺口总表」页。

- **拖拽引用 / Markdown 所见即所得 / 换行与格式化 / 「···」菜单**(`edit_ux.go` / `edit_md.go` / `edit_format.go` / `edit_tools.go` / `edit_menus.go`,设计稿 `ide-md`、`drag-more` 两页):文件树行、编辑器标签(含差异)、SCM 文件行用 `dragRef` 成为拖拽源(`fileRef{Path,Dir}`),对话栏 `chatPane` 里 `chatDrop` 接住站内拖拽与访达拖入(`DroppedFiles`),落下 = 输入框末尾插入 `@路径`(`appendRefs` 去重)并聚焦输入框,可放态 `dropOverlay`(`-ed drag:@路径` 可直接截图);Markdown 标签默认用 `ui/richtext` 所见即所得(`mdView`):文档与缓冲区两向同步(每次修改 `mdWrite` 序列化回缓冲区,缓冲区版本变了 `mdEnsure` 重新读入),YAML 头单独保留并显示只读属性条,含表格 / HTML / 脚注(`mdUnsafe`)只读预览,写法规范化只提示(`mdNormalized`);标签上的分段 `tb.mdMode` 优先于偏好 `editor.mdView`,要定位到行(`tb.reveal`)自动切源码;格式化 `formatDoc` 先试语言服务器,没有就走内置(`builtinFormat`:Go / JSON / Markdown / 文本),偏好 `editor.formatOnSave` / `editor.trimOnSave` 经 `ide.beforeSave` 在保存前整理;自动换行 `toggleWrap`(⌥Z)与差异视图共用 `editor.wrap`;菜单:标签条 ··· / 标签右键 / 文件树右键 / 树标题 ··· / 会话 ···(`edit_menus.go`),后端没有的动作置灰并写明原因。调试 `-ed md:src|wysiwyg`、`wrap`、`format`、`drag:@路径`、`mdedit:内容`;演示沙盒新增 `docs/guide.md`(正常)、`docs/api.md`(含表格)、`.cursor/rules/style.mdc`(带 YAML 头)。已知缺口:所见即所得往返会丢代码块语言标注、并掉段内换行;图片 / 附件、按标题滚动、向右拆分 / 固定标签、会话重命名 / 置顶 / 归档 / 删除 / 分叉(后端未接)、差异 / SCM / 终端 / 自定义页的 ··· 菜单未做。

- **文件新建 / 重命名 / 删除**(`file_ops.go`,`workspace/mutate.go`,设计稿 `drag-more`「文件操作」一节):`workspace.Mutable`(`Mem` / `OS` / `Multi` 实现;`OS` 拒绝 `..`、不删根,删除经 `TrashFunc` = Finder 移到废纸篓,失败不退回永久删除)。树里行内输入 `a.fo`(`fileOp`):新建插在文件夹子项最前、重命名换掉该行,⏎ 提交,失败留着并红字提示,Esc / 点别处取消;重命名后 `renameTabs` 让标签改路径(保留未保存修改,差异标签关闭);删除走 `confirmDlg`(文件夹写明文件数、有未保存标签提示)。多根时顶层文件夹行只有「新建…」。调试 `-ed newfile|newdir:文件夹:名字`、`renamef:路径:名字`、`foerr`、`delask:路径`。缺口:F2 / ⌘⌫ 快捷键、只选不含扩展名的部分、拖动移动文件。

- **SCM / 图谱 / 搜索菜单**(`scm_menus.go`,设计稿 `menus`「具体行为」一节):`scmDiscardAsk`(放弃更改统一确认,悬停按钮同走)、`scmFileMenu` / `scmGroupMenu`、`commitMenu`(图谱与 SCM 共用:与工作区比较 = 快速选择 `cmpfiles` 列文件 → `openCompareWork` 只读差异标签 `diff:cmp-<短哈希>:<路径>`;回退到这里 = `vcs.Repo.ResetKeep`(`git reset --keep`,不丢工作区,与回退内容冲突则中止;游离 HEAD / 当前提交 / 不在当前分支上置灰,`resetInfo` 数被移除的提交数)、`searchMatchMenu` / `searchFileMenu`。搜索列表行回调要防越界(菜单里折叠 / 忽略会在本帧中途重排 `rows`)。调试 `-ed discardask:路径`、`cmpfiles:提交`、`resetask:序号`。

- **其余「···」与右键菜单**(`more_menus.go`,设计稿 `menus` / `drag-more` 页):SCM 标题 ···(`scmHeadMenu`:拉取 / 推送 / 同步 / 储藏所有更改 / 弹出最近的储藏 / 刷新)、差异标签 ···(`diffMoreMenu`)、终端标签条 ···(`termMoreMenu`)、问题行与输出行右键(`problemMenu` / `outputMenu`,「让 Claude 修复」把诊断填进输入框)、计划 / 子 Agent / 任务虚拟标签右键(`virtMenu`:复制内容 / 引用到对话)、自定义详情 ···(`cuMoreMenu`:复制名称 / 在访达中显示 / 删除带确认)。测试用 `clickMore` 按位置点同名「更多操作」按钮。缺口:SCM 标题的列表 / 树切换(B3)、「在新窗口中打开」、自定义详情「用编辑器打开」。

### 集中补缺口一轮(B3–B7,代码已提交但**尚未测试**,待用户通知后集中测)

- **Git / 差异**:冲突标签按「冲突 k/N」单卡展示,结果区可编辑(`conflict.go`);差异注解条常驻块头 + 行级暂存(`ide_diff_extra.go`、`vcs/lines.go`);`scm.viewMode` 列表 / 树(`scm_tree.go`)、`scm.ignoreWs` 忽略空白;提交并同步、推送被拒引导(`PushForce` = `--force-with-lease`)、切分支储藏三选一(`git_dialogs.go`);审查页 `-ed review:empty`、页脚回退本回合(`review_turn.go`);子模块提示行(`scm_nested_hint.go`)。
- **会话流**:`flow_user.go` 按回合渲染(用户卡吸顶 / 折叠 / 就地编辑,`appearance.flowStyle`);`flow_fold.go` 的 `foldBody` 统一长输出限高;`flow_group.go` 探索组「最新一步」;回合操作条 `flow_turn.go`,分叉 `claude/fork.go`(`OpenSession("fork:<父>:<新>")`);改动卡撤销 `chg_undo.go`;`flow_tail.go` 尾读任务 `output_file` 进 `termLive`。
- **输入框 / 任务 / 通知**:`composer_mcp.go`(`CmdRefreshMcp`)、`ctx_panel.go`(输入框圆环与状态栏共用)、`task_dock.go` + `core.GroupTasks` 六类胶囊、`notice_*.go`(IDE 通知中心 + `general.notify*` 系统通知)、`update_progress.go`、`proj_pick.go`(方案 B 项目下拉)、`session_fork.go`(`minicode://session/<id>` 链接与分叉入口)。
- **编辑器**:Git 行号竖条靠 `syncDiffBase`(`ide_gutter.go`);`editor.minimapSlider/Width/Git`;CodeLens 为自绘叠层(测试数是占位);括号着色 `ide_brackets.go`;拆分 `ide_split.go`(`tabs2` / `splitDir` / `splitFrac`,同路径共享 `*editor.Buffer`);`lsp_codeaction.go` ⌘.、`ide_inline.go` ⌘I、`ide_suggest.go` ✦(启发式占位);文件树 F2 / ⌘⌫ / 拖移。
- **终端 / 语言服务**:多终端 `ide.tps/tpSel`,关到最后一个留空白壳;搜索 `Aa/ab/.*` + 刻度;`term.scrollback`;鼠标上报 `term_mouse.go`(按住 ⌥ 回选择,滚轮未编码);OSC 133 集成脚本在 `~/.minicode/shell/`;LSP 大纲 / inlayHint / 崩溃退避重启 / 工作区轻量诊断(`lsp_outline.go` 等)。语义高亮无库 API,未做。
- **设置**:`jumpSettingRow`、`restartForSettings`(自启 + 退出)、`shortcuts.<id>` 录制(`settings_shortcuts.go`)、`internal/claudecfg` 读写服务商 / 规则 / 信任 / 环境变量(密钥 0600 文件)、`kit/feedback.go`(`UndoSlot` / `ShakeInput`)、`kit.SystemReduceMotion()`。
- **外壳**:`responsive.go` + `overlays.go`(1040 / 900 / 720 断点、浮层 / 抽屉 / 图标条;右侧栏在 1040 以上随窗口连续缩小,主区 ≥ 440,拖窗口边缘期间面板尺寸直接跟随、不做过渡,只有某一栏被收起 / 展开时才带 440 ms 动画)、`layout_mem.go`(`layout.memAgent/memIDE`)、`morph.go`、`newwindow.go`(`--new-window`,从属进程不落盘)、`brand.go` + `mac_brand_darwin.go`(Dock 图标 / 角标 / 托盘,cgo 用 `-x objective-c`);main 经 `InstallAppMenu / InitBrand / SaveLayMemToPrefs` 接入。
- **整页**:自动化 `automation.ParseCron` + 运行记录;画布 `canvas.OpenLibrary(~/.minicode/canvases)` + `ApplyClaudeReply`;浏览器 `browser.HTTPSource`(`browser.fetch` 可关);自定义 `customize.OpenDisk`(原子写 + 备份)、`claude mcp/plugin` 子进程;用量限额经活动会话 `CmdRefreshUsage`。
- **本轮已知缺口**:⌥点击多光标、标签跨组拖动、真实 Haiku 行内补全、MD 图片附件;用户消息编辑只做本地截断、计划选中批注未接选区;演示 Bash 无增量输出剧本;MCP 工具级热开关、系统通知点击回会话、真实 updater 重启;终端滚轮上报、Nerd Font 缺字形检测;画布全屏无平移与列表删除 / 重命名;自动化运行中无流式输出、取消不杀子进程;浏览器框选元素;MCP / 插件表单未写回 JSON;Dock / 托盘缺动画帧与 error/update/paused/demo 图;`shortcutRecordCapture` 透明层可能挡点击(待测)。

- **二期与梳理补充(同样未测试)**:`kit` 新增 `Btn` 的 `Disabled/Loading`、`Field`(错误描边 + `ShakeInput`)、`popshell.go`(`PopIn/PopOut/StaggerRise/CollapseHide`)、`Ring`/`SkelRows`、`DurLoop`(1600 ms)、`UndoSlot`+`UndoBar`(会话删除先折叠再延迟落盘);用户消息编辑 `CmdEditUser`(`resume_session_at`,无 uuid 则置灰);计划段落「批注」(`plan_annotate.go`);MCP 工具级开关写本地 `permissions.deny`;`CmdCompact` 发 `/compact`;`internal/update` 状态机 + GitHub appcast(开发版重启走 `scripts/dev.sh`,正式包替换仍是占位);跨组拖标签 `ide_split_drag.go`;Claude 行内补全 / ⌘I 走 `claude.RunPromptModelCtx`(Haiku,可取消,失败退回占位);CodeLens 测试数来自本地符号启发式;终端 SGR 滚轮上报与 Nerd Font 缺字形提示(`term_nerd.go`);自动化 `RunPromptStreamCtx` 流式 + 取消杀进程组;画布 ` ```canvas ` 围栏回合结束自动写回;浏览器阅读区框选引用(`browser_select.go`);`DiskStore.SaveMCP`;服务商 CRUD(`providers.json` + 0600 `provider-keys.json`);Dock 图标读 `appearance.appIcon*` 并随系统深浅轮询;对话栏「分叉会话…」与回合条共用 `forkTurn`;文件标签右键「在新窗口中打开」= `SpawnNewWindow`。
- **二期仍未做**:语义高亮(无库 API)、剪贴板贴图与 WYSIWYG 内嵌图片块、系统通知点击回会话(缺 UN 委托)、正式包更新替换、托盘 error / update / paused / demo 图与动画帧、欢迎页 64px 品牌徽标、MCP 详情「编辑配置」表单、服务商行内「编辑」入口、Skill 重命名目录、文字流光 / 流底部渐隐 / 抽屉 / ok-pop 等剩余动效、`kit.Field` 铺到设置单行。

- **三期补充(未测试,仅编译通过)**:
  - 品牌:`assets/icon/gen.py all` + `build_all.sh` 生成六样式 × 浅 / 深 / 透明 / 着色 `.icns`、托盘 11 态 PNG 与动画帧、文档图标;`brand_tray.go`(托盘状态机)、`brand_dock.go` / `mac_dock_darwin.go`(Dock 弹跳 / 进度)、`brand_logo.go`(欢迎页徽标)、`mac_a11y_*`(减弱动态 / 高对比读取与 VoiceOver 播报);`scripts/dev.sh` 注册 UTType。
  - cgo 注意:带 `//export` 的 Go 文件序言里不能放非 static 定义(会被编译两次导致重复符号),ObjC 实现放 `.m` + `.h`(见 `mac_notify_darwin.*`)。
  - 真实模式:`claude/mcp_status.go`(`mcp_status`)、`claude/context_usage.go`(`get_context_usage`)、任务事件翻译与 `task_restart.go`、`welcome_recent.go`、通知点击经 UN 委托回 `OpenSessionFromNotify`、`internal/update/install.go`(替换脚本 + 重启)。
  - 动效 / 控件:`kit/motion_ui.go`(DrawerSlide / DragLift / ToastSlide / TextShimmer)、`kit/connectivity.go`(ConnTestRow)、`chat_fade.go`(流底部渐隐)、`dialogOpen`(对话框退场);Markdown 大纲定位 / 粘贴图片 / 往返保留(`edit_md_nav|paste|preserve.go`);OSC 133 注入 bash / zsh(`term/integration.go`);`--virt` 虚拟标签新窗口(`newwindow_virt.go`);MCP 编辑配置、Skill 重命名、服务商编辑、画布全屏坐标 / 适应、演示自动化运行器、用量重试。
- **三期仍未做**:语义高亮(MyGo 无 API,文档已写明取舍)、`resume_session_at` 分叉的真实后端核验、Mermaid 完整引擎、fish 的 OSC 133、JPEG 剪贴板、工作区 / 诊断 pull、`.icon` 打包与 Finder 图标换装、托盘按样式分图、规则 / 环境变量行与 quickpick 接 `Field` / `dialogOpen`、ok-pop / 折叠高度插值、侧栏会话行错峰入场、插件表单回写、会话内画布版本步进。**以上全部未做过截图 / `go test` 验证,等用户通知再进入阶段三。**

### 合并中止 / 追溯卡延迟 / 文件类型关联(已提交;单测 + 部分开发版实测)

- **合并中止**:`vcs.Repo` 新增 `MergeAbort()` 与 `OpState()`(`""` / `merge` / `rebase`,CLI 读 `rev-parse --absolute-git-dir` 下的 `rebase-merge` / `rebase-apply` / `MERGE_HEAD`);`scmRefresh` 据此置 `sc.rebase / sc.merging`,推送引导条(`scmPushGuide`)写「正在合并 / 变基」并给「中止合并 / 中止变基」,不再露出推送按钮。调试 `-ed mergeconflict`。仅单测(真实 git + Mem),未用 Computer Use 走过。
- **追溯悬停卡**:离开文字与卡片 250 ms 后才收(`blameCloseDelay` / `blameKeepOpen`,需 `frameSoon()` 续帧)。仅单测。
- **文件类型关联**(`open_external.go`,`settings_backup.go`):`mygo.App.OnOpenFile(a.OpenExternalFile)` 必须在 `Run` 前注册(Dock 拖放 / 最近打开 / 命令行同入口),`main.go` 先 `HoldExternalOpen(true)` 冷启动排队、`InitBrand()` 后放行;`.mccanvas`(= 画布库文件 JSON,`canvas.Library.Import/Export`)/ `.mcworkspace`(`projects.MarshalWorkspaceFile` / `ParseWorkspaceFile`,清掉 ID 与权限确认,缺失文件夹跳过并提示,全缺则拒绝,演示模式拒绝)/ `.mcsettings`(弹确认再合并,`isSecretPrefKey` 剔除密钥类键,导出同样剔除);出口:画布行 ··· 「导出为 .mccanvas」、工作区右键「导出为工作区文件…」、设置 → 高级。**实机踩坑**:`open -a` 必须用 `.app` 的绝对路径;Computer Use 的 app 目标点击需要最新截图的 `coordinate_token`,没有时可用 System Events 坐标点击(会卡住要及时 kill)。已用 `open -a` 在开发版实测画布 / 设置 / 工作区三种导入;Finder 里真双击图标与冷启动排队未实测。

- **托盘菜单**(`brand_tray.go` / `brand_tray_icons.go`):会话按项目分组(灰色禁用项当组名),行首文字标记(`trayMarkText`:⁘ 运行 / 🟠 待确认 / 🔵 未读,看过的会话用全角空格占位对齐),标题截 24 字,整体窄;「重启 MiniCode」走 `restartApp(reason, false)`(不改真实 / 演示记忆),有运行中会话或未保存文件时先弹确认。**库的 `MenuItem.Icon` 在托盘菜单里实机不渲染**(两次实测,生成与传入都正确),`MINI_TRAY_ICON=1` 可强制改回图标复测。**`syncBrand` 的 Dock 图标要按「样式 + 外观后缀 + 深浅」签名判断是否重设**,只比深浅会让设置里改样式不生效。**`scripts/dev.sh` 先跑 `go vet ./...`**,别的会话留下的半成品测试文件会让开发版起不来;还有 `~/.claude.json` 项目条目缺 `disabledMcpServers` 曾让界面构建 panic(已修)。

### 关闭 ⌘W 按所在位置分流(`close_ctx.go`,设计稿「窗口形态 · 关闭 ⌘W」,仅 `go build` 通过,**未测试**)

- **规则**:⌘W / 文件 → 关闭 走 `closeHere`:在工作区的编辑器标签组里只关当前标签(拆分时只关焦点组的;未保存且别处没开着同一缓冲区时先 `confirmDlg`);在底部面板的终端里关当前终端标签(只剩一个 = 收起面板,不结束 shell);拆分后空着的组 = 合并拆分;其它地方(导航栏 / 对话栏 / 设置与整页)或标签已关完 = 关窗口。**关窗口不退出**:主窗口 `Hide()`(不看 `general.close`,那个偏好只管红色按钮),从属窗口(⇧⌘N)`Close()` 真关闭。
- **「你在哪」**(`ide.cz`,`zoneTrack`):① 根 / 工作区 / 底部面板三层 `ObserveInput`(捕获阶段、外层先收到,所以最里面的区域胜出)记最近一次指针按下 / 按键;② `zoneSync` 每帧比较键盘焦点所在区域,换了就跟过去;③ 焦点组的当前标签在最近一次输入后 800 ms 内变了(文件树单击、会话流「查看」)= 人过去了,Claude 自己开的标签不算,换项目带来的变化不算。新增多标签区域时照 `zoneMarkEditor / zoneMarkPanel` 接一层并在 `closeTarget` 里加分支。
- **菜单**:「文件 → 关闭」是自定义项 + `CmdOrCtrl+W`(不用 `RoleClose`,它直接关窗口);菜单快捷键先于窗口收到按键,所以应用内不用再判 ⌘W。从属窗口现在也装应用菜单(否则用库默认菜单,⌘W 不会先关标签),只是不放「新建窗口 / 最近项目」。
- 缺口:菜单文案固定叫「关闭」,不随位置变成「关闭标签 / 关闭窗口」;标签上的 × 关未保存标签仍不确认(只有 ⌘W 确认);全屏下隐藏主窗口的表现未验证;隐藏后从 Dock / 托盘回来依赖 `showMainWindow`(另一处改动)。

### 排队引导 / 问答卡 / 回到最新(设计稿 `flow-queue` 页,`js/specs5.js`,仅 `go build` 通过,**未测试**)

- **待发送区**(`dock.go::queueCard`、`queue.go`、`composer_queue.go`):排队(⏎,偏好 `general.busy` 可对调)与引导(⌘⏎)都先落在输入框正上方。引导 = `steerSend`:立刻 `CmdPrompt{Steer}` 发给 Claude,但登记成 `promptQueue.steers`,行内显示「引导 · 等 Claude 接收」,会话流里用 `visibleItems()` 藏起后端回显的那条用户消息;`syncSteers` 每帧启发式判定接收(其后出现新的非用户条目且停留 ≥ 700 ms,或回合结束)后才放进会话流(后端没有显式确认信号)。排队行悬停:改为引导 / 停止并发送 / 删除,点文字退回编辑。输入框占位随回合状态写明 ⏎ / ⌘⏎。
- **问答卡**(`dock_ask.go`):多题时顶部步骤条可直接跳任意一题(已答带对勾)、「上一题 / 下一题(最后是提交)」、← → 切换(无文字输入框时);提交前校验,前面有题没答会跳回去。回答后会话流里是回答卡片(`askEcho`:问题 + 回答胶囊,多选多枚,自由输入方角,跳过灰色)。
- **回到最新**(`flow_scroll.go`):输入框正上方居中的 30px 圆形向下箭头(替换右下角「↓ 回到底部」),距底 > 32px 才出现,离开期间有新内容右上角亮点(`flowTailMark`);手动滚回底部自动恢复跟随;子 Agent / 任务输出同形态。
- Markdown 标签的「所见即所得」界面文案已改叫「预览」(功能不变),预览顶部格式条默认不显示(`mdShowFormatBar`)。

### IDE 细节缺口补齐(代码已提交,**未测试**,待用户说「开始测」)

- **编辑器**:多光标 `ide_multicursor.go`(状态栏「N 个光标」芯片 / 1000 上限 / 只读置灰,库自带 ⌘D ⇧⌘L ⌥⌘↑↓ 键位,`-ed multicursor`)、转到行浮条 `ide_goto.go`(五态,与 ⌘P `:行号` 共用 `parseLineCol`,`-ed gotoline:<文本>`)、块注释退化提示(`-ed blockcomment`)、外部修改 `ide_extmod.go`(三态 + 删除态 + 查看差异,`-ed extmod:auto|clean|dirty|deleted`)、面包屑同级下拉 `ide_crumb.go`(`-ed crumb:file|sym`)。缺口:Esc 回单光标 / 一次撤销整批 / ⌥点击上限未实测;「保留在编辑器」没有另存为;面包屑弹层标题行与符号取本地索引。
- **问题面板**:`internal/lsp` 新增 `workspace/diagnostic` / `textDocument/diagnostic`(`previousResultId` + 流式片段),`lsp_workspace.go` 按能力选 pull / 逐文件拉取 / didOpen 扫盘并显示来源;分段「已打开 | 工作区」、进度 / 取消 / 截断(`lsp.workspaceDiagLimit` 默认 2000)/ 空 / 错误态,`-ed wsdiag:scan|done|trunc|empty|err`。阶段三实测:本机 gopls 0.22 / pyright 都不声明 `diagnosticProvider`,真实环境目前走 didOpen 扫盘兜底,pull 路径无真实服务器验证(`internal/lsp/real_gopls_pull_test.go` 可复测);`.gitignore` 只跳过硬编码目录。
- **Markdown**:预览写回只重写被编辑的块(`edit_md_preserve.go`,LCS 对齐),`-ed mdkeep`;⌘V 粘贴 JPEG(10 MB 上限、去重文件名)。缺口:跨多块选区的合并写回。
- **Git**:追溯卡 / 追溯栏、储藏展开、推送被拒「拉取并变基 / 合并」、图谱两种比较、审查一键暂存都按设计稿审查补齐(`-ed blamecard|stashfiles|cmp2|cmphead`)。缺口:合并冲突无 `git merge --abort`、追溯卡移出无延迟收起、储藏父行无右键、分组审查页按仓库的暂存钮。
- 设计稿 `ide-nav` 新增「多光标 / 转到行 / 块注释 / 磁盘外部修改」,`ide-problems` 新增工作区分段,`ide-md` 新增「写回策略」,`ide-git` 补变基冲突 / 储藏展开 / 追溯栏 / 比较两个提交画面。

### 终端实测踩坑(Computer Use,真实 zsh,已修)

- **叠在终端正文上的控件**(Nerd 提示条按钮、⌘F 搜索条、「↓ 回到底部」):正文的 `HandleInput` 会吃掉按下事件(用来开始选择),控件永远点不到。这些控件构造时要 `tp.addOverlay(el.Bounds())`,`termInput` 对落在覆盖区里的指针事件直接 `return false`;新增同类控件照此办。Tester 里这类按钮「没有可见区域」点不了,只能测输入逻辑(`TestTermOverlayPassesPointer`),真点击靠实机。
- **vt 仿真器的锁**:`SafeEmulator.Write` 持写锁,OSC 等处理器里只能用内嵌的无锁 `*Emulator`(`t.em.Emulator.CursorPosition()` 等),调带锁的包装会死锁(测试会卡住 120 s)。OSC 处理器的 `data` 带命令编号前缀(`"133;B"`)。OSC 133 的 zsh 脚本需要 `autoload -Uz add-zsh-hook`。命令块语义:B = 输出首行,C;<退出码> = 提示符返回,清屏丢弃旧块。
- **命令装饰点**画在行左边距(`padXTerm` = 10px,Absolute),点击由 `termInput` 判断 `ev.X < padXTerm` 复制输出,不要再用覆盖整行的热区(会抢提示符行的点击)。
- **鼠标上报**:vt 的 `SendMouse` 不按模式过滤移动事件,`internal/term/mouse.go` 的 `mouseWanted` 负责过滤(只要点击的程序不收移动)。
- **Nerd Font**:`term_fonts.go` 后台扫 `/Library/Fonts`、`~/Library/Fonts`;`termFontFamily` 只在「选了 Nerd 字体且确认未安装」时回退等宽;提示条 `nerdWarning` 分「没装 / 已装未选用」两种文案。
- 调试:`MINI_TERM_CMD='clear; seq 1 120; ls'` 启动后在第一个终端自动输入命令;Computer Use 的字母键会走输入法(首次回车被组字吃掉),终端正文不能 `computer_type`,滚轮可用,不能带修饰键点击(⌘点击链接只能靠单测)。

### 全流程实测踩坑(Computer Use,真实 DeepSeek 回路,已修)

- **`EvPromoteToPermission` 会把时间线里的 `*core.ToolCall` 整个换成 `*core.PermissionRequest{Tool}`**。凡是遍历 `a.st.Items` 找工具的代码(改动统计、回合折叠摘要、审查入口……)必须同时处理这两种条目,统一用 `doneToolOf(it)`(只认已处理的权限请求),否则「允许后的编辑」整轮都不出现在改动卡 / 审查页 / 「本轮」里。`ToolStatus.Unsuccessful()`(失败 / 拒绝 / 中断)的编辑不计入改动。
- **被拒绝的命令没有执行过**:后端回的「退出码 1」只是拒绝信号。实时路径是 `ToolDenied`,**历史回放没有权限事件,状态只是 `Failed`**,所以命令卡用 `deniedOutput(输出)` 按拒绝文案兜底显示「未执行」。
- **文件树缓存**:Claude 新建文件后树里不出现(要手点刷新)。`syncTreeGit` 发现改动路径集合变化(`treeKeysChanged`)就清 `listCache / filesCache`,回合结束(`EvTurnEnded`)也清一次。
- **CodeLens 与行内追溯同在光标行右端会叠成乱码**:光标行正显示追溯(`tb.blameShown`)时不画该行的 CodeLens(追溯优先)。
- **SIGTERM / SIGINT / SIGHUP 要当正常退出**(`diag.WatchSignals`):系统关机 / 登出 / `pkill` 重启开发版都发这些信号,不处理会让「崩溃与卡顿」列表被「未正常退出」淹没;SIGKILL 与真崩溃仍留运行标记。macOS 输入法框架的 `TSM AdjustCapsLockLED…` stderr 提示是无害噪音,已降为调试级。
- **`kit.Ib(...).ContextMenu(fn)` 只响应右键**:要「左键点 ··· 弹菜单」必须用 `.Menu(fn)`(画布列表行 ··· 曾因此点了没反应)。`ui.Tester.AccessAll` 里一行的可访问项会把子按钮并进行名(如「第二张 刚刚 更多」),子按钮不能单独按名字找,测试按位置点。
- **`update.Runner.Start` 是异步 goroutine,`Run` 才是同步**:在 `a.bg` 里要用 `Run`(用 `Start` + `defer cancel()` 会让请求立刻 `context canceled`,界面永远「下载失败」)。`Run` 开头只能发 `PhaseNone`(检查中);发下载进度会被界面当成「发现新版本」(版本号还是当前版本)。失败按阶段(`ErrStage`:check / download / verify / unpack / install)翻成人话,「重试」走 `retryUpdate`。版本号唯一来源是 `main.go` 的 `version`,启动时写进 `app.Version`。
- **`diag.SpanSlow` 的阈值不能写 0,也不要多传 `nil`**:0 阈值每次调用都记 WARN,多出的 `nil` 让 slog 打出 `!BADKEY=<nil>`。
- **方案 B 的会话列表(`sessionsB`)与方案 A / C 的 `sessionRows` 是两条渲染路径**:置顶 / 归档逻辑都经 `sessionsSplit`,「已归档 N」折叠行共用 `archiveFold`;新增会话列表变体不要漏掉归档入口(否则归档后无处找回)。点导航「搜索」分段(侧栏分段与图标条两处)经 `focusSearchOnEnter` 让输入框自动聚焦。
- **浏览器页**:没有可读正文(`pageHasNoText`)时显示空态卡 +「在系统浏览器打开」,不留空白;相对链接用 `resolveURL` 对当前页解析(`#锚点` / `javascript:` / `mailto:` 丢弃);地址栏图标:http 与本机地址为警示、https 为锁、空页为地球。开发验证用本地 `python3 -m http.server`,Shell 工具结束时会连带结束后台进程,要 `start_new_session=True` 脱离。
- Computer Use 实机边界:悬停才出现的按钮(差异块「暂存 / 还原此块」)与弹出菜单项 AX 点不到、编辑器里放不下光标、不能打字进编辑器;这些只靠单测,实机用「搜索结果跳行」之类的入口间接到达。

### 品牌图标 / 托盘 / 应用菜单(Computer Use + 菜单栏截图实测,已修)

- **素材内嵌**:`assets/icon/build_all.sh` 生成后同步到 `mygo/internal/app/brandassets/`,`brand_embed.go` 用 `go:embed` 读取(开发版从访达 / `open` 启动时工作目录是 `/`,靠文件路径永远找不到素材,托盘曾因此整个不出现)。命名:应用图标 `app-style<N>[-dark|-clear|-tint].png`(N 与 `appStyles` 同序:提示符(默认)/ 经纬 / 棋盘 / 行 / 花括号),托盘 `tray[-s<N>]-<状态>.png`(**N 与应用图标下标一一对应**,0 省略;经纬 = 2×2 织纹缩小版:经线常亮、纬线两行轮流点亮),运行中三帧 `-running-0..2`。**应用图标与菜单栏图标只有一个样式偏好 `appearance.appStyle`**(旧 `appearance.trayStyle` 由 `migrateIconStyles` 迁移后清除,`iconStyleIndex()` 是唯一出口);织纹样式已删除。改 `gen.py` 后重跑 `build_all.sh` 并把 `brandassets` 一起提交。
- **`mygo.NewTray` 在应用就绪前会直接 panic**(无任何输出、进程静默退出 2,日志停在「载入设置」后)。`syncBrand` 开头必须 `if !mygo.App.IsReady() { return }`;`New()` 里的 `retheme` 也会走到它,就绪后 `InitBrand` 会补一次。
- **托盘模板图 vs 彩色徽标**:纯黑 + alpha 的模板图随菜单栏明暗自动着色;开「彩色徽标」时合成非模板图,必须用 `Tray.Dark()`(macOS 26 的菜单栏明暗跟壁纸走,不等于 `AppleInterfaceStyle`)并在 `OnAppearanceChange` 里重画。状态优先级见 `brandSummarize`:待确认 > 出错 > 已完成 > 运行中 > 有更新 > 暂停 > 空闲;出错 / 已完成只记「不在前台时发生」(`brand.trayErr/trayUnread`),回前台或新回合开始清除。
- **库的菜单图标(`MenuItem.Icon`)在托盘菜单里实机不渲染**(PNG 正确、`NewMenuIcon` 成功,仍不显示;带 `Subtitle` 的项也一样),托盘会话行的状态点改用 🟠🔵🟢 字符。`System Events` 能点托盘:`click menu bar item 1 of menu bar 2`(`process "minicode"`),子菜单用 `menu 1 of menu item "…"`;`screencapture -x` 后用 PIL 裁剪看菜单栏。
- **应用菜单栏**(`newwindow.go::installAppMenu`):自定义项不挂 Accelerator(快捷键由应用内按用户设置分发,可改键),标准项走 Role(`RoleEditMenu` / `RoleWindowMenu` / `RoleHide` / `RoleQuit`…);只放 MiniCode / 文件 两项会丢掉编辑、窗口、帮助与退出。
- **暂停通知**:`general.pauseNotify`(开关)+ `general.pauseUntil`(限时到期 Unix 秒),`notifyPaused()` 懒清理到期项,`pauseNotifyFor(d)`:`d>0` 限时 / `0` 一直 / `<0` 恢复;托盘菜单是二级(1 小时 / 到明天 9:00 / 一直 / 恢复)。
- **访达换装**(`brand_finder.go`,`appearance.fileIcon`):`NSWorkspace setIcon:forFile:` 给 `.app` 包写自定义图标(包根多一个隐藏的 `Icon\r`,传 nil 即还原);实测可用,但**开启期间 `codesign --verify --deep --strict` 会报 resource fork**(浅层校验、启动与自动更新不受影响);**应用更新会替换整个 .app,图标随之丢失**,所以每次启动 `finderSig` 为空会自动补套。按「开关 + 样式 + 外观」签名去重、后台执行;不是从 `.app` 启动(`go run` / 测试)直接跳过。失败不清签名(避免每个事件都重试),状态写进 `brand.finderState` 显示在设置里,「重新应用」(`reapplyBrand`)清签名强制重试。开发版 `dev.sh` 每次重新打包会还原。
- **设置「不生效」的几个根因(已修)**:① 托盘单击「打开窗口」从不生效——菜单一直挂着,系统先弹菜单(现在只有「弹出菜单」模式才挂菜单,右键用 `PopUpMenu`);② 改「运行中动画」后计时器不重新对账(`reconcileTrayAnim`);③ 彩色徽标没看 `trayDyn`;④ Dock / 访达失败静默。现在每次 `syncBrand` 都会刷新图标 / 标题 / 提示,设置页「当前生效」三行状态 + 「重新应用」。
- **动态预览**(`brand_live.go`):Dock + 浅 / 深菜单栏三处用同一个 `brandLiveClock` 驱动(2.8 s 一个状态,180 ms 一帧,33 ms 重绘),`brandBounceDy` 复现 Dock 弹跳的抛物线;样式卡悬停播放「运行中」。设计稿对应 `brand.js` 的 `live()` / `lockup()`。
- **图标名必须存在**:`kit.Ic/Ib/Btn{Icon}` 用不存在的名字会画成 ✕。图标表由 `scripts/gen-icons.mjs` 从 `docs/design/js/icons.js` 生成(先加设计稿再生成),`TestIconNamesExist` 静态扫源码、`zz_icons_test.go` 在 app 包所有界面测试之后检查 `kit.UnknownIcons()`。
- **设置行 `setRow` 带键必须写 `def`**:自定义行(`kind: kNone`)没写默认值,`rowChanged` 一律认为「改过」,默认值也亮红点。

### 性能排查踩坑(形态切换 / 启动卡顿,已修)

- **看门狗会记栈**:主线程停滞超过 400 ms,`diag.Watchdog` 每 100 ms 记一次「主线程停滞时的调用栈」(最多 3 次,内层 10 帧 + 外层 `app.` / `kit.` / `main.` 帧,取 1 号 goroutine;`runtime.Stack(all=true)` 第一块是看门狗自己,要按 `goroutine 1 [` 找)。若栈里只有框架帧(`text.layout ← main.main`),说明卡在框架的布局阶段,看不出节点——把框架拷到 `/tmp`、在 `ui/layoutcache.go` 的 `textLayout` 和 `internal/text` 的 `layout` 里按耗时写日志,用 `go build -modfile=… ` + `replace` 临时构建(不进仓库)。
- **`ElementTransition{Size}` 会在每个中间尺寸重排内容**:工作区从折叠展开时起始宽度 ≈ 0~1 px。`SingleLine()` 带 `MaxLines=1`,`LayoutFrom` 因此不能复用旧排版,**每换一个宽度整段重排**(长文本在窄宽度下每行 5~8 ms);`NoWrap()` 的排版与宽度无关、换宽度直接复用。所以会被放进动画面板的长列表(输出面板最多 300 行日志)一律 `NoWrap` + 容器 `ClipX`,不要用 `SingleLine`。底部面板停在「输出」时,切到 IDE 曾每次卡 400~570 ms。
- **形态切换卡顿与项目大小有关**:对话栏宽度做 440 ms 过渡,库会在每个中间宽度重排整棵子树;会话流没有虚拟化,大项目里会话长、文字多,每帧 CoreText 断行几十毫秒(CPU profile:一次 Agent→IDE 布局 390 ms,其中 `textLayout` 320 ms)。**历史做法已撤销**:曾在过渡期间把 `flowCol` 内容列宽度吸附到 80px 台阶(`flowSnapWidth`),代价是切换结束那一刻才跳回精确宽度并整体重排(「最后一瞬间才重绘」)。会话流后来改成虚拟列表(只构建视口内的行),吸附已不需要,现在宽度全程跟随容器、逐帧动态重排;无窗口计时(宽度 700↔1280 共 194 帧):默认演示会话平均 7 ms / 最大 15 ms,放大 30 倍后中位 8 ms / p95 31 ms / 最大 40 ms。别再加「钉在终态宽度 / 吸附台阶」这类只在结束时重排的办法;要提速优先减少每个中间宽度要排的行数。剩余主要是工作区(编辑器标签)按动画宽度排版。**排查方法**:临时在 `main` 包加个收到 SIGUSR1 就 `pprof.StartCPUProfile` 3 秒的文件(不提交),发信号后立刻切形态,`go tool pprof -top -cum` 看——Go 帧完整可见,比看门狗栈和 `sample` 都准。
- **自动最小宽度**:行内 flex 子项没有显式 `MinWidth` / `ClipX` 时,自动最小宽度 = 子树最小内容宽度(把文字按宽度 1 逐词折行排一遍);长文本放进 `Row`(尤其 `Wrap`)要给 `MinWidth(0)`。
- **`customize.OpenDisk` 不再同步起子进程**:它只给兜底市场数据,真实列表由 `cuEnsureDisk` 放 `a.bg` 里 `RunMarketplaceList`,回主线程 `SetMarket`。凡在界面线程每帧可达的路径(输入框 MCP 芯片、斜杠命令)都不能 `exec`。
- **`kit.SystemReduceMotion` 走 cgo 的 NSWorkspace**,不再每次 `defaults read` 起进程。
- 验证方式:`MINI_WIN_W/H` 把窗口开到用户同样大小;`MINI_SESSION` 会让 `ApplySavedRuntime` 提前返回、**不恢复编辑器标签与底部面板标签**,复现 IDE 形态问题要去掉它。向开发版发键盘事件前必须校验前台就是它(否则按键会落到别的应用)。

### 连接预热与空闲释放(`app/warm.go`、`claude/backend.go`,设计稿 `settings.js`「连接」组 + 状态栏「已连接」芯片)

- **结论(真实 claude 2.1.325 + DeepSeek 实测,N=3~5)**:拉起进程 + `initialize` 握手稳态 **205~280 ms**(空目录 224 / 真实项目 249 / 恢复 0.5 MB 会话 245 / 恢复 10.6 MB 会话 276 ms,与会话大小几乎无关);首次冷磁盘缓存可到 600 ms。预热把首响应从中位 548 ms 降到 244 ms。常驻成本:每个空闲进程 **约 180~250 MB RSS、空闲 CPU ≈ 1.5%**。所以预热「值得做但不能滥」:只给前台会话、闲置要回收。模型本身的首 token(用户真实会话记录:发出 → 第一条回复,中位 1.5~3 s)才是「等反馈」的大头,客户端管不了。
- **策略**(偏好 `chat.warm` 默认 0 打开会话时 / 1 输入时 / 2 不预热;`chat.idle` 默认 1 = 10 分钟,0 = 5 分钟 / 2 = 30 分钟 / 3 = 不释放):触发 = 会话成为前台且窗口在前台,去抖 350 ms;打字 / 窗口回前台 / 释放后用户回来都补预热;数量 = 只有前台会话持有空闲进程(切走时空闲的结束,运行中的转后台保活);回收 = 前台闲置超时(窗口在后台 ⅓)发 `core.CmdRelease`,会话保留、下次发送自动 `--resume`;后端拒绝回收的条件:回合进行中 / 有待处理请求 / 有存活后台任务;预热失败 60 s 退避。**输入框永远不禁用**(Zed 的做法会让用户干等 6~10 s)。
- **状态**:`core.EvConn` → `State.Conn`(未连接 / 连接中 / 已连接 / 连接失败),状态栏芯片(`warm_ui.go`,点未连接 / 失败 = 立即连接),只对真实 claude 后端显示。`warmTick` 每帧 + 15 s 节拍 + 去抖定时器调用;测试用 `a.warm.fgHook` 注入前台状态、`a.SetClock` 拨时钟。
- **业界做法**:官方 Agent SDK `startup()` 预先 spawn + 握手(首问快一个数量级);Zed 打开项目就给恢复线程起 agent(~335 MB / 线程,输入禁用);VS Code 扩展每个面板一个进程且不回收(有人 29 个进程 7.4 GB);T3 Code 30 分钟闲置回收 + resume。**已知缺口**:恢复时 MCP 服务器可能还没连好(官方 `MCP_CONNECTION_NONBLOCKING=0` 可让恢复等 MCP,上限 `MCP_CONNECT_TIMEOUT_MS` 5 s;本机没配 MCP,未做);恢复场景下「首响应」指标没采到(总耗时可用);Anthropic 5 分钟提示缓存过期后的首问慢是服务端预填充,预热管不了。**Computer Use 在开发版(真实模式 + DeepSeek)实测**:启动恢复会话即预热(开发版刚编完时握手 1.84 s,之后 386 / 323 ms)、切会话时旧的空闲进程立即结束并给新会话预热、状态栏芯片四态(已连接 / 未连接已见)、空闲 5 分钟整释放(进程数归 0,芯片转「未连接」)、点芯片后以 `--resume` 重连(握手 323 ms)、设置 → 通用 → 连接两行可调;未实测:窗口在后台时 ⅓ 时限(仅单测)、连接失败态的真实画面、释放后直接发送(仅后端单测)。

### 故障隔离 / ⌘ 跳转 / 折叠(本轮,单测通过,实机部分验证)

- **界面分区故障隔离**(`safe_view.go`):`a.guard(c, 名称, key, kind, build)` 包住一块界面,构建 panic 只让这一块显示「出错了」卡片(重试 / 复制错误信息),日志 `guard` 与 `crashes/` 留栈;`guardChip`(标题栏 / 状态栏小红条)、`guardCard`(导航栏 / 整页 / 对话 / 每个编辑标签 / 底部面板)、`guardSilent`(浮层:不画,5 秒后自动重试并提示)。**测试里(`a.win == nil`)默认不吞 panic**,要测拦截设 `a.guardAlways`。新增大块界面照 `shell.go` / `workspace.go` 的写法包一层;MyGo 在嵌套 `Children` 里 recover 后控件树不会错乱(`kit/safe_probe_test.go` 验证)。
- **⌘ 悬停可跳转**(`ide_link.go`):按住 ⌘,指针下「本地符号索引有定义,或语言服务器就绪」的标识符加下划线 + 手型,点击走原 `OnDefinition`;位置 = `edGutterW(tb)`(复现库的行号栏宽:行号至少 3 位 + 折叠 + Git 竖条 + 4px)+ `edTextPad`(≈6pt)+ 显示列 × `edChar × edScale`(Tab / 宽字符已换算),自动换行与差异标签不画。`cursorXY` 同样改用 `edGutterW`。**实机验证用 `/tmp/cmdhover` 一类 CGEvent 脚本;合成的鼠标移动不会触发列表行的悬停态(`Hovered()` 在 Tester 里正常),悬停底色只能靠单测。**
- **折叠**:箭头 / 占位 / 快捷键是库自带(`NoFolding` 由 `editor.folding` 控制),`ide_fold.go` 补右键菜单与命令面板;状态栏缩进菜单 `ide_indent_menu.go`;文件树自动定位 `ide_reveal.go`(`explorer.autoReveal`)。
- **去噪**:滚动行范围气泡、迷你地图源码摘要已删;CodeLens 默认关(`editor.codelens`);列表行悬停改 `Fill2`(8.5%),SCM 提交行选中改 `Sel`。

### 已知缺口(真实模式)

- **真实模式实测第五轮(同上环境,Computer Use,SCM 提交回路)**:**开会话会往用户仓库里写 `.claude/settings.json` 与 `.mini-backup`**(`SyncClaudeSettingsFromPrefs` 每次都重写 + 换备份),现在内容没变不写,项目里没有该文件且用户没改过相关偏好时不创建;图谱 HEAD 提交右键的「检出此提交 / 与 HEAD 比较」置灰(检出当前提交会无故让 HEAD 游离);Git 操作失败的 toast / 提交框 / 推送被拒引导条改用 `friendlyGitErr`(无远程 / 无上游 / 认证 / 网络 / 被拒 / 未设用户 / 锁文件翻成一句中文加下一步,认不出取首行并截断),完整原文仍进输出面板与日志;**AI 提交说明等一次性 `claude -p`(`RunPromptModelCtx`)此前会在 `~/.claude/projects` 留下「你是 Git 提交说明助手…」会话污染历史列表**,现加 `--no-session-persistence`(自动化流式运行故意保留会话,「查看会话」要用);**真实后端的编辑块 `DiffBlock.Line` 一直是 0**(审查页「第 1–4 行」、编辑器「Claude 改动」叠层都标在第 1 行),现 `BuildTool` 按文件内容定位(实时找 old、回放找 new,>1 MiB 放弃);审查页文件标题不再露绝对目录、改动已提交后说明写「已经提交到 Git」。**均有 go test,并已在开发版实机走过**(暂存 → AI 提交说明 → 提交 → 图谱右键 → 推送无远程 → 审查页行号 → 命令面板 / 主题预览与切换)。已知:Computer Use 的 AX 点击对弹出菜单项无效(菜单项没有 press 动作),菜单只能点头部等价按钮或用坐标点击;「审查本轮改动」入口在改动已提交后仍显示「待审查」(审查页内已如实说明);此前已生成的一次性会话不会自动清理。
- **Worktree 新建(设置 → 源代码管理,`worktree_add.go`,已用 Computer Use 实测)**:分支名输入 + 「新建 worktree」,目录 = 偏好 `scm.worktrees`(默认 `~/.minicode/worktrees`,支持 `~`)下的 `<项目名>-<分支名>`(`/` 换成 `-`);分支已存在则直接检出、否则 `-b` 基于当前 HEAD 新建;分支名先过 `git check-ref-format`,目录已存在先拦;错误显示在输入框下方并抖动,删除失败的 toast 带原因(`friendlyWorktreeRmErr`:有未提交改动 / 被锁定 / 主工作树)。演示模式按钮置灰(不真的建目录);删除 worktree 不会删分支。
- **真实模式实测第五轮补充(自动化页,Computer Use)**:新建对话框提示词为空时输入框描边变红,错误只在该字段值与报错时快照一致时显示(改了就消失);「下次运行」统一经 `automation.WhenNext`(今天 / 明天 / 周几 / M 月 D 日,总带时间,不再出现「今天 09:00 09:00」);**自动化流式运行缺 `--verbose`**(stream-json 在 `-p` 下必须带,否则 stderr 报错但退出码 0,运行被记成「成功」且无输出),现已补上,并把「没有文本也没有会话」判为失败(带 stderr 摘要);`assistant` 文本与 `result` 事件是同一段话,此前输出写两遍,现 `result` 只在没收到文本时兜底;运行详情在运行结束后自动跟到最新一条记录(原来停在空的实时视图,显示「运行失败」);「立即运行」→「查看会话」可直接回放该次会话;**删除自动化原先点一下就删(运行记录一并丢失、无确认无撤销)**,现走 `confirmDlg`。**均有 go test,并已在开发版实机走过**(新建 → 错误态 → 创建 → 立即运行 → 查看会话 → 删除确认)。已知:自动化运行中输出只在结束时一次出现(无逐段流式)。
- **真实模式实测第五轮补充二(浏览器页 / 审查入口,Computer Use)**:内置浏览器 `Fetch` 原先要求主机名含「.」且 `Normalize` 丢协议,`localhost` / `localhost:3000` 一律报「网址格式不正确」(预览本地开发服务器的主要场景),现 `fullURL` 解析主机名:本机地址(localhost / 127.x / ::1 / `*.local` / 192.168.x / 10.x)默认 http,其余 https,`httpbin.org` 这类以 http 开头的域名也不再被当成带协议;抓取失败经 `friendlyNetErr` 翻成中文加下一步(连接被拒绝 / 找不到域名 / 超时 / 证书),原文只进日志。工作区空态的「审查本轮改动」入口在改动已提交后写「已提交到 Git,可回看」(`changesCommitted`,与审查页页脚说明共用判断)。**均有 go test,浏览器已在开发版实机走过**(localhost 抓取成功 / 连接被拒绝提示)。`http://` 显式协议现被保留(`Fetch` 看原始输入);自动补的 https 握手失败(`httpsUnsupported`:对方给的是 HTTP 响应 / 非 TLS / 连接被重置)时自动退回 http 重试一次,从历史记录重开纯 http 站点也能打开;`Browser.Go` 记住「显式 http」的主机(`plain`)并把原始协议交给 `Fetch`。**导航失败时地址不进历史,原先「重新载入」什么也不做**,现 `Browser.failed` 记住失败地址、`Reload` 优先重试它(成功才入历史)。均有 go test,并已在开发版实机走过(站点停掉 → 失败提示 → 站点恢复 → 重新载入 → 自动退回 http 显示正文)。 历史列表不再展示旧版本一次性 `claude -p`(AI 提交说明 / 行内补全 / 行内编辑)留下的内部会话(`claude/index.go` 的 `isInternalPrompt` 按提示词开头过滤,仅隐藏、不删文件;已在开发版实机确认列表里消失)。 设置页:「默认模型」第一项由虚构的「Fable 5.2」改叫「默认」;权限规则输入框不再预填 `Bash(git status)`(手滑点「添加」会静默写入规则);**「语法高亮」开关此前只存偏好,现已生效**(`applyEditorPrefs` 把主题里每种词法着色显式写成正文色;**坑:不能把 `Tokens` 置零,库会把零值颜色补成默认主题色,实机一开始关键字反而变红**),均有 go test,语法高亮已在开发版实机开 / 关各走一遍。 **设置行「恢复默认」按钮悬停才出现,会把右侧控件整体左推(实机点击两次落空才发现)**,现所有带键的行统一预留 20px 槽位(`settingsRow`),改过与没改过的行右侧控件对齐、悬停不跳动。注:Computer Use 的点击偶尔需要点两次(第一次只触发悬停),不是产品问题;我早先实测时遗留在真实设置里的终端行高 / 光标形状已手工还原。
- **真实模式实测第四轮(同上环境,Computer Use,Agent / IDE 会话回路 + 审查页)**:**点会话里的「审查」整个应用闪退**(`reviewFooterText` 直接调 `a.clock()`,真实应用没注入时钟 → 空指针;测试 / 截图都注入了时钟所以从未暴露),已改 `a.now()` 并补无时钟回归测试;**界面线程 panic 此前什么都不留**(日志 / `crashes/` 都空,只剩「上次运行异常退出」),现在 `App.View` 先 `diag.Report` 记调用栈再原样抛出(不吞,控件树做到一半会错乱);`SetLocale("zh-Hans")` 在真实应用里被静默忽略(库文案如代码块「Copy」、右键菜单「Select All」仍是英文),需先 `SetLanguages`;设置页单行输入框恢复描边与聚焦环;移除高级页无效的「一次测完行高」开关;权限卡右上角工具标签中文化;正在看会话时权限请求不再弹浮层遮住权限卡(只记进通知中心,已读);第三方模型(DeepSeek 等)回合不再显示按 Claude 价格估算的费用;审查页块级「撤销」后补「重新应用」(按撤销位置校验)。**均有 go test,并已在开发版实机走过**(权限卡 → 改动卡 → 命令输出 → 审查 → 撤销 / 重新应用)。仍未实机验证:窄窗口下审查页页脚(文案被按钮挤截断,仅可见未改)、IDE 形态下审查入口。
- **真实模式实测第三轮(同上环境,Computer Use,设置各页 + 终端 / 搜索 / SCM)**:修掉「界面显示的默认值与真实生效值不一致」一类问题(权限页额外目录、运行环境代理地址 / 绕过代理、终端 Shell 程序现在都是空值 + 占位提示;**追加系统提示词默认为空,不再给所有会话静默注入示例文案**);演示模式预置的假快捷键会写进共用的 `settings.json` 污染真实模式(⌘E 被改成命令面板),现只在 `MINI_SHORTCUT_DEMO=1` 时预置;关于页未检查更新时写「尚未检查更新」而不是「已是最新」;外观文案去掉开发术语;隐私页崩溃记录类别中文化、worktree 列表与崩溃列表改后台缓存(原来每帧起 git 进程 / 读盘),主工作树不给删除入口;高级页 Claude 配置摘要接真实数据(三层规则数 + 环境变量数);终端启动不再回显 `source …/integration.zsh`(注入后清屏);搜索结果行文本过长时行号被挤出面板已修。**均有 `go test` 并已在开发版实机看过**。未处理:设置里单行输入框(如「回看行数」「Diff 大小上限」)没有输入框描边、不像可编辑(待 `kit.Field` 铺到设置单行);「会话流一次测完行高」开关仍无效(行内已写明);规则 / 环境变量页每帧读小 JSON(未缓存)。注:Computer Use 的字母键走系统输入法,第一次回车会被组字吃掉,不是产品问题。
- **真实模式实测第二轮(同上环境,Computer Use)**:命令面板 IDE 形态有查询时文件排在会话前(`palItems`);差异标签可见时也定时刷新仓库状态(`diffTab` 调 `scmTick`);**居中浮层入场计时被外壳每帧清掉(`app.go` 无条件 `dlgSeen = ""`),新建自动化 / 自定义新建 / 文件树「让 Claude」对话框永远透明却拦截点击,已修(回归测试 `dialog_anim_test.go`)**;对话框输入框恢复 `kit.Field` 自带边框;新建自动化错误按字段定位、重开清除;浏览器 `extractHTML` 对只有 `<p>` 的页面丢正文(`findMain` 找不到 `body`)已修,写死的「对比 actix」快捷提示改通用文案;画布页行容器补 `Stretch`。实机走通:gopls 诊断波浪线 / 问题面板 / 右键「让 Claude 修复」、磁盘外部修改自动重载、浏览器真实抓取 example.com。**未能实机验证**:编辑器内 F2 / F12 / 悬停(Computer Use 的 AX 点击放不下编辑器光标,只靠单测)。全量 `go vet` / `go test ./...` 通过。
- **真实模式端到端实测一轮(DeepSeek,`/tmp/mc-e2e`,Computer Use 操作开发版)**:读 → 改 → 权限卡 → 运行 → 总结的核心回路走通。本轮修掉:项目根经软链(`/tmp` ↔ `/private/tmp`)时 `wsRel` 换不出相对路径(改动卡 / 「本轮」/ 打开文件全受影响,现用 `underProj` 同时认原样与解开软链的写法,界面显示一律 `shortPath`);改动卡只放最后一轮;失败 / 被拒绝 / 被中断的编辑不计入改动统计;编辑权限卡预览改动(`permDiffPreview`,超 14 行只给摘要);欢迎页「审查改动」取仓库真实改动数;后端没给标题时用第一条提问当会话标题(`core.TitleFromPrompt`)。**这些均有 `go test`,并已在开发版实机看过;其余 74 个「部分」项仍未逐项实机验证。**

- **逐页盘点(设计稿 v3.6,缺口总表以 `docs/design/js/status.js` 扫描结果为准,下面旧条目有的已过时)**:340 个功能点,已实现 220 / 部分 82 / 待做 38(**数字是设计稿状态表自评,未经阶段三验证**)。新增登记的缺口,按批次:
 - B3 Git:推送被拒「拉取并变基 / 合并」分按钮、审查一键暂存已接受改动、追溯悬停提交卡、储藏内文件差异、图谱「与 HEAD 比较 / 比较两个提交」。
 - B4 会话流:`core.TaskNote` 已有但流里不渲染、并行子 Agent 汇总行、回合折叠(`chat.fold` 没人读)、用户消息悬停动作条与「回退文件」预览入口、用户消息图片、输入框「+」附件与斜杠命令、审批排队「1 / 3」、排队卡「停止并发送」、提示卡动作按钮。
 - B5 编辑器:转到行 ⌃G(及 ⌘P 的 `:行号`)、块注释 ⌥⇧A、磁盘外部修改检测(没有文件监视)、追溯栏、面包屑同级下拉、多光标未在应用层验证、终端标签重命名、语义高亮、Markdown 粘贴 JPEG。
 - B6 设置:`chat.sys` / `runtime.bin` / `models.model|effort`(与 `claude.model|effort` 不是同一个键)/ `chat.fold` / `perm.*` / `term.login|cwd` / `scm.diffMax|fetch|worktrees` / `privacy.level` 等**只存偏好不生效**;设计稿有而设置页没有的行(等待动画、滚动条、诊断包导出、claude doctor、settings 导入导出、开源致谢等);Go 里仍叫「检查器」,设计稿 v3.1 已没有这个概念;命令面板缺高频命令;账号菜单用户名 / 用量是写死的。
 - B2 菜单:文件树右键缺「让 Claude…」「在侧边打开」、侧栏会话行缺「分叉 / 复制会话链接」、搜索「在侧边预览」仍置灰(编辑器拆分已有,可接通)、附加文件夹权限提示卡缺「只读授权」。
 - B1 多仓:「全部展开」模式未做。B7:自动化运行不能「查看会话」(只有只读输出浮层)、会话流内 Mermaid 图表卡与排版引擎。B8:`kit.OkPop` / `IndetMeter` / `FoldHeight` 定义了但没有调用点,图标按钮 / 开关缺 disabled、统一按下态与键盘聚焦环未做。
 - 状态降级:多光标、工作区级诊断(`didOpen` 扫盘代替 pull)、终端标签重命名、task-spec 首行等原先标「已实现」的,已改为「部分 / 待做」。
 - 设计稿脚本:`providers.js` 的 `ensure()` 在 `APP` 就绪前返回 `null`,导致 `overlays.js` 加载期崩溃、「浮层与面板」页整页空白,已用临时状态兜底修复。
- 注:上面 B3–B8 的旧条目里,第一波已写代码的项以下面一段为准(它们现在是「部分,未测试」,不再是「待做」)。
- **第一波补缺口(代码已写、仅 `go build` 通过,**未测试**,等用户说「开始测」)**:缺口总表现为 344 点 / 已实现 218 / 部分 118 / 待做 8;下面各项在 `status.js` 里一律标「部分」并注明未测试。
  - 会话流:`flow_note.go`(任务通知行)、`flow_parallel.go`(并行子 Agent 汇总)、`flow_turn_fold.go`(`chat.fold` 回合折叠)、`flow_user_actions.go`(悬停动作条)、`flow_rewind.go`(回退文件对话框)、`flow_user_img.go` + `img_cache.go`(用户图片;本地图片经 `a.localImage` 后台解码,界面线程只查表);输入框:`composer_attach.go`(「+」附件)、`composer_slash.go`(/ 命令补全)、`composer_queue.go`(停止并发送)、`dock_pending.go`(审批 1/N)。
  - 编辑器 / 终端:`ide_goto.go`(转到行 ⌃G、⌘P `:行号`)、块注释 ⌥⇧A、`ide_extmod.go`(外部修改)、`ide_crumb.go`(面包屑同级下拉)、`term_rename.go`;`term.Start` 现在经 `App.TermStartOpts` 读 `term.login / cwd / shell / scrollback`。
  - Git:`scm_compare.go` / `scm_graph_picks.go`(与 HEAD / 两提交比较)、`git_push_actions.go`(拉取并变基 / 合并,`vcs/sync.go`)、`review_stage.go`(一键暂存已接受改动)、`ide_blame_ui.go`(追溯栏 + 悬停卡)、`scm_stash_expand.go`(储藏展开,`vcs/stash_diff.go`)、`scm_expand_all.go`(多仓库全部展开,设置 → 源代码管理 `scm.expandAll`);Diff 大小上限在 `openDiffKind` 弹确认(「仍然显示」,会话内记住)。
  - 设置:`settings_wire.go` 等接了运行环境 / 代理 / 隐私诊断 / 高级项;`scm.fetch` 节拍在 main.go 常开、由 `ScmFetchTick` 按偏好执行。`adv.flowMeasure`(一次测完行高)MyGo 没有对应 API,**只存偏好**。
  - kit:`Btn / Ib / Tog` 的 Disabled / Loading / Error、按下缩放、聚焦环(近似)、加载 400 ms 延迟、`Card`、`SpinStyled`;`OkPop / IndetMeter / FoldHeight` 已在自动化页接线,`GrowY / Crossfade` 仍无调用点。自动化「查看会话」`auto_runner.go`。
  - **仍未做(第一波时的清单,第二波已处理其中大部分,见下)**:Mermaid 图表卡与排版引擎、语义高亮(无库 API)、定时自动化回写会话 id、文件夹剪切 / 粘贴、图片排队、账号菜单真实数据。
- **第二波补缺口(代码已写、仅 `go build` 通过,**未测试**)**:缺口总表 359 点 / 已实现 238 / 部分 120 / 待做 1(只剩语义高亮,MyGo 无前景着色 API,明确不做)。
  - Mermaid:`internal/mermaid`(解析 + 分层布局 + Scene,移植自旧 `mini-mermaid`,支持流程 / 时序 / 饼 / 状态图,其余返回 `ErrUnsupported` 回退源码卡)、`flow_mermaid.go`(图表卡,后台排版按源码哈希缓存)、`mermaid_viewer.go`(全屏:适应 / 100% / ± / 拖拽 / Esc);演示会话 `mermaid-samples`。未做:gantt / class / ER 等、画布页接新引擎、滚轮以光标为中心缩放。
  - 输入框与账号:排队卡保留并发送图片(`CmdPrompt.Images`),`claudecfg.ReadOAuthProfile` 只读 `~/.claude.json` 的 `oauthAccount` 非密钥字段,账号菜单用量取 `usageSrc` 与 `FiveHourPct`,没数据显示「暂无数据」。
  - 自动化与动效:定时 / 立即运行统一走 `runAutomationCore`,成功 / 失败 / 取消都回写 `Run.SessionID`,「查看会话」可用(索引未命中先 `BuildIndex` 重扫);`kit.GrowY` 接用量柱状图、`kit.Crossfade` 接用量骨架 → 内容。
  - 偏好接线:`general.restore` / `general.close` / `general.sound` / `chat.ding` / `chat.queueAtt` / `chat.queuePause` / `models.thinking`(写回 `alwaysThinkingEnabled`)/ `models.compact|compactAt`(应用内 `CmdCompact`,**不写** claude 设置)/ `general.paste`(超长粘贴存 `~/.minicode/paste/*.txt` 以 `@` 引用,发送时展开,无「展开为文本」chip);`perm.sandbox` 仍只存偏好;`adv.flowMeasure` 在设置行里写明「Go 版暂不生效」。
  - 设置真实化:项目信任读写 `~/.claude.json` 的 `hasTrustDialogAccepted`(原子写 + 备份)、语言服务器行后台检测(`settings_lsp.go`,偏好 `lsp.server.<id>`)、规则 / 环境变量去掉演示回退(环境变量只显示名称)。已知:`writeJSON` 不保证 `~/.claude.json` 顶层键顺序;环境变量删除撤销固定走用户层。
  - 外壳与命令:整页窄窗(自定义 / 用量 / 自动化,断点读 `winW` 而不是容器宽)、命令面板高频命令 + ⇧Tab 循环权限模式(`perm_cycle.go`)、时间线字号 `FlowFontSize` 等、形态切换分轨淡入(`morph_content.go`,未做标题栏形态滑块分轨)。
  - 提醒:第二波中途曾有提交引用了未提交的 `pageNarrowMid` 定义使远端 HEAD 编不过(`2ab7164` 已补),`specs2.js` 缺括号使缺口总表脚本整页不加载(`71be730` 已修)。**并行子代理 / 多会话提交务必用干净检出验证 HEAD:`git worktree add --detach /tmp/wt HEAD && cd /tmp/wt/mygo && go build ./...`。**

- **服务商与模型映射(`prov_*.go`,`claudecfg/providers*.go`,设计稿 `js/providers.js`,未测试)**:`~/.minicode/providers.json` = `{providers, mapping}`(兼容旧数组格式),密钥在 0600 的 `provider-keys.json`;设置 → 服务商可新建自定义 / 预设、「获取模型列表」(`FetchModels` 依次试 `/v1/models` 等候选,全 404 = 不支持,可手动添加)、勾选要用的模型;模型映射 5 个槽位(默认 / Opus / Sonnet / Haiku / 子 Agent)只能从该服务商已勾选的模型里下拉选;输入框配置弹层(`cfgModels`)按「Claude 官方 + 各服务商」分组直接选模型,切换经 `core.CmdSetEnv`(后端只合并受管理的环境变量,回合进行中不推送),新会话启动时由 `claudecfg.ProviderEnv()` 注入。演示模式只改内存、不落盘。缺口:真实连通性只做 HEAD 探测;未做 go test。
- 设置页:快捷键录制、规则 / 环境变量 / LSP 行是演示数据;「项目信任」只列项目,不读写信任状态;除上面列出的偏好外,其它控件只存偏好。
- 语言服务器(LSP)已接入诊断 / 悬停 / 定义 / 引用 / 补全 / 重命名 / 格式化;大纲、代码操作、内联提示、语义高亮仍未接,诊断只覆盖已打开的文档,没装服务器时退回本地符号索引。
- 终端:有多标签、选择复制、搜索、超链接、字体设置,但没有鼠标上报与命令装饰;Nerd Font 图标字形无字体时显示成方框;改项目会结束旧终端。
- 用量页:没有账户限额与上下文分类明细(后端未接 `get_usage` / 分类);前台命令的实时输出尾读未接入;右栏「改动」仍由会话 diff 推导。
- Diff / 合并:块头动作只在悬停时出现(库的行为,设计稿是常驻);冲突视图把所有冲突按卡片列出,没有设计稿里「冲突 1/2」的逐个切换与三方「结果」可编辑区(diff3 的 base 被丢弃);审查页的行内评论串只有演示。
- 画布「让 Claude 修改」只把提示填进输入框,没有图表全屏查看器;浏览器页只有演示源,没有真实抓取;自定义页只在内存里。
- 动效:文字流光(用呼吸代替)、折叠展开的高度插值(只做淡入下落,避免挤动下方内容)、流底部渐隐遮罩未做。

Rust + gpui-kit 0.8 的 Claude Code 工作台。只支持 Claude Code,直连本机 `claude` 的 stream-json 控制协议(不走 ACP、不依赖 Node)。

### 布局动画模型(左栏 / 右栏 / 底部面板 / 形态切换 / 窗口缩放)

- **动画的是「开合因子」(0..1),不是像素宽度**:`paneOpen`(motion_layout.go)每帧返回因子,宽度 = 因子 × 本帧重算的满开宽度(`wsRW` / `dockW` / `sideW`,`sideOcc` 取导航栏**本帧**的动画宽)。动像素宽度的话,目标(窗口宽、另一块面板)在动,动画值会滞后,出现「先涨后缩」「露出空白」。
- IDE 舞台 = `[工作区 Grow(1)][对话栏 Width = dockW×fDock×(1−fMax) + 舞台宽×fMax]`,两者加间距恒等于舞台,任何一帧都不会有空白;Agent⇄IDE 的传送带(morph_belt.go)是围绕对话栏的一条连续链。
- 窄窗断点(<900 / <1040)导航栏改抽屉:并排列照样 260 ms 收到 0(`side()` 不再提前 return),抽屉里的内容在 overlayLayer 画。舞台那一行有固定 `Key("stage")`,导航栏有无不会改变它的身份(否则整个舞台状态重置)。
- **形态切换时导航栏不能「直接到位」**:`paneSnap()` 在形态切换(传送带)期间返回 true,但导航栏在传送带之外,若也直接到位,会一帧从 228 跳到 0,而传送带的几何仍按旧舞台宽算 → 右侧空白。导航栏用 `paneOpenFree`(只有拖分隔条 / 连续缩窗才直接到位)。
- 底部面板收起:高度 `AnimateWith("h")`,透明度跟随「当前高度 / 满高」(前一半高度内淡出),不再单独计时,避免内容比外框先消失(设计稿 `.bpanel` 同步改为 height + opacity 同曲线过渡)。
- 拖分隔条 / 拖窗口边缘走 `instantLayout()`(逐帧跟手,不做动画);终端 pty 尺寸在面板尺寸稳定 90 ms 后才改(`realTerminal`),动画里不逐帧重排。
- **探针**(anim_probe.go):每帧读各区矩形,检查 空白 / 跳变(二阶差分)/ 底部面板右缘与工作区对齐 / 掉帧(>34 ms)。动画结束写一行汇总到 `[anim]`;`MINI_ANIMLOG=1` 逐帧写日志。
- **测试**:`anim_layout_test.go` 用 `kit.SetClock(tt.Now)` 把动效时钟接到 Tester 的虚拟时间,逐帧(16 ms)推进并断言探针 0 异常;`ANIM_DUMP=1|2 go test ./internal/app -run TestAnim -v` 打印汇总 / 逐帧表。动效里读时间一律用 `kit.Now()` / `kit.Since()`(不要直接 `time.Now()` / `time.Since()`)。
- 向窗口要帧用 `frameSoon()`(= `Services.Invalidate`,CVDisplayLink 驱动,跟随屏幕刷新率),不要再用 16 ms 定时器。

## 目录结构

见 `AGENTS.md` 的「结构」表;运行、打包、发布命令也在那里。旧 Rust 实现(crate 表、`cargo` 命令、各功能的 `.rs` 实现细节)已从本文件删除,需要时从 git 历史取(删除前最后一个提交的父提交是 `ddd7452`)。


## Claude 协议要点(已对真实 claude 2.1.x 验证)

- 初始化:`initialize` 带 `perTaskStopAffordance`(否则中断会杀掉后台任务)、`agentProgressSummaries`、`forwardSubagentText`;`stop_task {task_id}` 停任务,`background_tasks` 查存活任务;`system` 的 `task_started / task_progress / task_updated / task_notification / background_tasks_changed` 见 `translate.rs`(`tool_progress` 心跳刻意忽略);后台 Bash 的 `toolUseResult.backgroundTaskId` 与后台 Agent 的 `async_launched` 都只表示「已启动」,不是完成。
- 参数:`--output-format stream-json --verbose --input-format stream-json --include-partial-messages --permission-prompt-tool=stdio --allow-dangerously-skip-permissions`(后者只是放行「绕过权限」选项、不默认开启;没有它,运行中 `set_permission_mode(bypassPermissions)` 会报 `bypass_not_launched`),另按需加 `--resume`/`--session-id`/`--model`/`--permission-mode`/`--effort`。
- 握手:发 `initialize` 控制请求,响应里有模型列表与当前权限模式;`system init` 要等第一条提示之后才出现。
- **前台 Bash 的实时输出**(实测 2.1.318):stream-json 在前台命令运行期间不给任何输出(无 `tool_progress` 内容,`task_started` 的 `output_file` 为空),但 claude 会实时写 `<临时目录>/claude-<uid>/<cwd 编码>/<会话 id>/tasks/<任务 id>.output`。`mini-claude/live_output.rs` 按此规则找文件(对不上退回扫描 `claude-<uid>/*/*/tasks/`),`Translator::poll_live`(后端每 700 ms 发 `Msg::Tick`)读末尾 96 KB,发 `ToolUpdate{ patch.live_output }`:`ToolCall::apply` 把它当没有退出码的 Terminal 块**整体替换**,最终 `tool_result` 到达时取代它;工具卡对运行中且有输出的命令默认展开、贴底跟随。后台命令不走这条(走右栏「任务」标签)。集成测试:`cargo test -p mini-claude --test integration -- --ignored --nocapture 前台命令`。
- 权限:`can_use_tool` 在 `assistant` 的 `tool_use` 之后到达,按 `tool_use_id` 对应;`AskUserQuestion` 的答案写进 `updatedInput.answers`;`ExitPlanMode` 批准时回传建议的 `updatedPermissions`。
- claude 不回显用户输入,后端自己把提示追加进时间线。
- 历史在 `~/.claude/projects/<编码后的 cwd>/*.jsonl`,目录名把所有非字母数字字符替换成 `-`(有损,真实 cwd 取自会话内容)。

- **编辑器叠层必须在绘制阶段定位**(括号竖线 `ide_pairs::guides`、当前块高亮线 `ide_guides::overlay`、行内追溯 `ide_blame::inline_blame`):编辑器的布局(`range_to_bounds` / `visible_row_range`)在它**自己的绘制**里才更新,渲染阶段读到的是上一帧的,滚动停下后叠层会停在原地。所以叠层一律做成盖住编辑区的 `canvas`(`absolute().size_full()`,排在编辑器之后),在 paint 闭包里读编辑器实体、算位置、`paint_quad` / `shape_line().paint`。**`text_bounds()` 不是可视区**:它是随滚动平移的内容区域(滚下去后 y 是很大的负数),裁剪请用 `ide_guides::text_view(ed, 画布范围)`(左边 = 内容原点 − 横向滚动量,上下 = 画布自己的范围)。画布闭包要 `'static`,所以共享状态(如 `PairState`)内部用 `Rc` 且可克隆。

### 模型选择踩坑(实测 claude 2.1.325,界面显示与实际模型不一致,已修)

- **`--model <别名>` 的优先级高于 `ANTHROPIC_MODEL`**:官方偏好 `claude.model=sonnet` 会让第三方服务商会话(界面显示 deepseek)实际跑 claude-sonnet-5-5;`FillClaudeConfig` 在第三方生效时必须用 `pvActiveModel()`(映射默认模型 id)覆盖 `cfg.Model`,`applySavedPicks` 也不能再套官方偏好。
- **`set_model "default"` 会被解析成 claude 自己的默认模型(opus),不是 `ANTHROPIC_MODEL`**:切服务商 / 模型必须发确切的模型 id(`--model <id>` 与 `set_model <id>` 都实测有效)。
- **回合进行中切换服务商,环境推送不能直接丢**:`pvPushEnv` 记 `pvEnvPending`,下一条消息发出前补发(`App.send`)。
- 日志「回合完成」带 `模型=`(取自 `message_start.message.model`,是实际模型),排查「选了 A 跑了 B」先看它。
