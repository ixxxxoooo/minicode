# MiniCode

一个 **macOS 原生**的 Claude Code 工作台:Agent(会话为主)与 IDE(编辑器为主)是同一个窗口的两种形态,共用一份会话、编辑器和终端。
只支持 Claude Code,直连本机 `claude` 的 stream-json 协议(不走 ACP、不依赖 Node)。

当前分支 `feat/mygo`:界面用 [MyGo](https://mygo.run/zh)(Go 原生 UI,无 WebView)重写,代码在 [`mygo/`](mygo)。
设计稿(HTML 原型)在 [`docs/design`](docs/design),**Go 实现以设计稿为准**。

> 状态说明:大量功能是「代码已写、编译通过、有单测」,但**没有在真实环境逐项验证**。
> 缺口以设计稿「缺口总表」为准(见下文「缺口与进度」),不要把演示数据当成真实能力。

---

## 环境要求

| 依赖 | 说明 |
| --- | --- |
| macOS 13+ | 目前只在 macOS 上开发和验证(cgo 调 AppKit / UserNotifications) |
| Go 1.27.4 | `mygo/go.mod` 指定;用 Homebrew 装的 Go 即可 |
| Xcode 命令行工具 | cgo 需要 `clang`、`codesign` |
| `claude` CLI | **真实模式**需要(`PATH` 里能找到,或设 `MINI_CLAUDE_PATH`);演示模式不需要 |
| Google Chrome | 只用于给设计稿出截图(`scripts/proto-shot.sh`) |

**每次执行 Go 命令前先 `unset GOROOT`**(环境里残留的 `GOROOT` 会让工具链对不上):

```sh
cd mygo && unset GOROOT
```

---

## 启动开发版

```sh
scripts/dev.sh --demo     # 演示模式:脚本化后端 + 沙盒项目,不连 claude,不会动你的真实数据
scripts/dev.sh --real     # 真实模式:读 ~/.claude/projects,按需惰性启动 claude
scripts/dev.sh            # 不带开关:沿用上次选择(记在 ~/.minicode/dev-mode),从没选过则是真实模式
```

`dev.sh` 做的事:`go vet` → 编译 → **杀掉所有旧实例** → 在最小 `.app` 包 `target/dev/MiniCode Dev.app` 里启动**唯一**的新实例(经 `open` 启动,关终端不会连带被杀)。
编译或 vet 失败时旧实例保持运行,先修好再重试。**不要用 `go run` 另起实例,也不要碰打包版。**

- 开发版左下角有徽章:演示模式显示「演示模式」,真实模式显示「开发版」。
- 运行模式优先级:`--demo` / `--real` / `MINI_DEMO=1|0` > 上次记住的选择 > 真实。
- 演示与真实**各存一份**运行状态(`~/.minicode/last-session-demo.json` / `last-session.json`),互不串。
- 想在启动时带环境变量,直接写在前面,`dev.sh` 会把所有 `MINI_*` 转给应用:

```sh
MINI_MODE=ide MINI_THEME=ink MINI_SCHEME=b scripts/dev.sh --demo
MINI_PROJECT=$HOME/code/foo scripts/dev.sh --real
```

### 调试环境变量

| 变量 | 作用 |
| --- | --- |
| `MINI_DEMO=1\|0` | 强制演示 / 真实 |
| `MINI_MODE=ide` | 启动进 IDE 形态 |
| `MINI_PROJECT=<目录>` | 真实模式的初始项目 |
| `MINI_THEME` / `MINI_SCHEME` | 主题 id / 布局方案 a、b、c |
| `MINI_SCREEN` | 启动直接进某个屏幕(settings / usage / automation …) |
| `MINI_WIN_W` / `MINI_WIN_H` | 窗口尺寸(最小宽 900) |
| `MINI_WS=1` | 启动即进入演示工作区「全栈工作区」 |
| `MINI_SESSION=<id>` | 启动打开指定会话 |
| `MINI_DEMO_SLOW` | 演示实时播放的慢放倍数 |
| `MINI_CLAUDE_PATH` | 指定 `claude` 可执行文件 |
| `MINI_LOG=off\|error\|warn\|info\|debug` | 临时覆盖日志级别 |

### 日志与排查

| 位置 | 内容 |
| --- | --- |
| `~/.minicode/logs/minicode.log` | 应用日志(`diag.L("包名")`) |
| `target/dev.log`(上一个在 `dev.log.prev`) | 标准错误、帧统计 |
| `~/.minicode/crashes/` | panic、崩溃、卡死记录 |
| 设置 → 隐私与诊断 | 资源曲线、慢操作统计、崩溃列表、导出诊断包、`claude doctor` |

### 数据在哪

全部在 `~/.minicode/`:`settings.json`(偏好)、`workspaces.json`(手动添加的项目 / 工作区)、`session-meta.json`(置顶 / 归档 / 重命名)、`automations.json`、`providers.json` + `provider-keys.json`(**0600,密钥只在这里**)、`canvases/`、`paste/`。
应用**不读取、不输出**任何密钥;环境变量只展示名称。

---

## 用真实 claude 跑(用 DeepSeek 代替官方模型)

`scripts/with-deepseek.sh` 运行时从 `~/.minicode/provider-keys.json` 读取 DeepSeek 密钥并注入环境变量,**密钥不进仓库 / 日志 / 提交**:

```sh
scripts/with-deepseek.sh scripts/dev.sh --real
cd mygo && MINI_REAL_CLAUDE=1 ../scripts/with-deepseek.sh go test ./internal/claude -run Real
```

先在应用「设置 → 模型与服务商」给 DeepSeek 填好密钥。`DEEPSEEK_MODEL=deepseek-flash|deepseek-v4-pro` 可切带思考的模型。

---

## 测试与截图

```sh
cd mygo && unset GOROOT
gofmt -l . ; go vet ./... && go test ./...
```

- 界面用 `ui.Tester` 写测试(`internal/app/*_test.go`,可参考 `flow_test.go` 的 `sessionApp`、`settings_test.go` 的 `settingsApp`)。
- 真实 claude 的集成测试默认跳过,要 `MINI_REAL_CLAUDE=1`。
- **磁盘空间**:Go 构建缓存会长到几个 GB;编译报 `no space left on device` 时先 `go clean -cache`。

### 无窗口截图 `cmd/shot`

```sh
cd mygo && go build -o /tmp/shotbin ./cmd/shot
/tmp/shotbin -scheme a -theme paper -mode agent -screen work -o /tmp/x.png
/tmp/shotbin -screen settings -tab privacy -o /tmp/set.png
/tmp/shotbin -state scn:mermaid-samples -w 1280 -h 1900 -o /tmp/mm.png   # 回放任一演示场景
```

常用参数:`-scheme a|b|c`、`-theme`、`-mode agent|ide`、`-screen`、`-state`(welcome / perm / ask / plan / queue / steer …,或 `scn:<场景 id>`)、`-popup`、`-click`、`-palette`、`-tab`、`-ed`(编辑器动作)、`-ws`、`-dlg`,`-h` 看全部。
**静态截图只用来出对照图;判断软件真实表现请直接看运行中的开发版**(见下节)。

### 设计稿截图(对照用)

```sh
scripts/proto-shot.sh <页面> <方案> <主题> <形态> <输出.png>
scripts/proto-shot.sh work a paper agent /tmp/proto-work.png
scripts/compare.sh work agent a:paper b:mist c:ink      # Go 版 / 设计稿并排,输出在 /tmp/minicode-cmp/
```

Read 类工具会缓存图片,**每次换新文件名**。

---

## 设计稿

直接用浏览器打开 `docs/design/index.html`(页面入口见 `js/app.js` 的 `PAGES`)。
「缺口总表」页**不是手写的**:它扫描各页面里的「实现状态」表,把「部分 / 待做」自动汇总;改了哪页的状态它立刻同步。状态数据主要在 `js/status.js`,新增页在 `js/specs*.js`。

**流程(强制):先改设计稿、再写代码。** 设计稿没有或缺状态 / 菜单 / 空态 / 错误态时,先补设计稿并在「实现状态」列如实标注。

---

## 目录结构

| 路径 | 职责 |
| --- | --- |
| `mygo/internal/core` | 与界面无关的数据模型:时间线条目、`Event` / `Command`、`State.Apply` 归约器 |
| `mygo/internal/backend` | `Backend` 抽象(`Start(sink)` / `Send(cmd)` / `Close()`) |
| `mygo/internal/claude` | 真实后端:stream-json 协议、翻译器、惰性启动与崩溃自动 `--resume`、历史回放、会话索引 |
| `mygo/internal/demo` | 演示后端:脚本化场景、沙盒项目、演示仓库 |
| `mygo/internal/app` | **全部界面**:会话流 / 输入框 / IDE / 终端面板 / 源代码管理 / 设置 / 命令面板等 |
| `mygo/internal/lsp` | 语言服务器客户端内核 |
| `mygo/internal/term` | 终端内核(pty + 仿真器) |
| `mygo/internal/vcs` | Git 访问(`CLI` 走 git 命令行,`Mem` 供测试 / 演示) |
| `mygo/internal/workspace` / `projects` | 文件系统抽象 / 项目与工作区存储 |
| `mygo/internal/mermaid` | Mermaid 解析与排版(流程 / 时序 / 饼 / 状态图) |
| `mygo/internal/kit` / `theme` | 控件与设计令牌(6 套主题 × 3 种布局) |
| `mygo/internal/diag` | 日志、慢操作计时、崩溃与卡死看门狗 |
| `mygo/cmd/shot` | 无窗口截图工具 |
| `docs/design` | 设计稿(HTML 原型) |
| `scripts/` | `dev.sh`、`proto-shot.sh`、`compare.sh`、`with-deepseek.sh` |
| `assets/icon/` | 图标与托盘资源 |

数据流:`Backend` → `core.Event` → `core.State.Apply` → 界面;界面交互转成 `core.Command` 交回后端。演示与真实后端共用同一套界面。
更详细的约定与「踩过的坑」见 [`AGENTS.md`](AGENTS.md)。

---

## 开发约定

- 回答、注释、文档一律中文;提交信息用中文 Conventional Commits(`feat:` / `fix:` …),**一个功能一个提交**,改完立刻 `git push origin HEAD`(推到 `feat/mygo`)。
- 并行 / 多会话开发时**只按路径 `git add` 自己改的文件**;提交后用干净检出验证 HEAD 能编译:
  `git worktree add --detach /tmp/wt HEAD && cd /tmp/wt/mygo && go build ./...`
- 日志一律 `diag.L("包名")`,不用 `fmt.Println` / `log`;耗时操作用 `diag.SpanSlow`;后台任务经 `a.bg(name, work)`。
- 不提交 `target` / `dist`;不读取、不输出密钥。
- 三阶段工作流(补缺口 → 逐项审查 → 集中测试)见 `AGENTS.md`「开发工作流」。

---

## 用 Computer Use 检查运行中的开发版

检查界面时直接操作运行中的 `MiniCode Dev`,不要只看静态截图。注意:

1. **先用 `scripts/dev.sh --demo` 启动**,再在 Computer Use 里 `computer_resolve_app` 取 `target_id`;**每次重启后 pid 和 `target_id` 都会变**。
2. 动作要带最近一次截图返回的 `coordinate_token`;截图坐标系是 1280×800。
3. 弹层(模型菜单、`+` 菜单、账号菜单)对窗口焦点敏感,点不开时先 `open -b dev.minicode.app.dev` 把窗口切到前台。
4. 合成的「鼠标移动」不一定触发悬停态,悬停类效果可能验证不了,改用 `ui.Tester` 测。
5. 键入文字用 `computer_type`;`Esc` 关弹层。

---

## 缺口与进度

- 缺口总表:设计稿「缺口总表」页(`docs/design`),数据来自各页「实现状态」。
- 已知缺口与各批次实现说明:[`AGENTS.md`](AGENTS.md) 的「已知缺口」。
- 汇报口径:区分 **已实现 / 部分 / 未做 / 未测试**;演示数据不等于真实能力。
- 当前明确**未做**:语义高亮(MyGo 无前景着色 API)、Mermaid 的 gantt / 类图 / ER 图、`perm.sandbox` 与 `adv.flowMeasure` 只存偏好不生效。

---

## 常见问题

| 现象 | 处理 |
| --- | --- |
| `dev.sh` 在 vet 阶段失败 | 先修 vet 报错;旧实例仍在运行 |
| 报 `no space left on device` | 磁盘满了,`go clean -cache` 释放构建缓存 |
| 工具链版本对不上 | `unset GOROOT` |
| 开发版被别人 / 别的会话重启成了另一种模式 | 约定同一时间只有一个会话在重启开发版;重启前先 `pgrep -fl "MiniCode Dev"` 看当前是谁 |
| 真实模式启动在一个打不开的目录 | 已修:初始目录和「最近项目」会跳过不存在的目录 |
| 设置页「崩溃与卡顿」里出现 `unclean-exit` | `dev.sh` 重启杀旧实例留下的记录,不是真崩溃 |
