# MiniCode

macOS 原生的 Claude Code 工作台。Go 1.27 + [MyGo](https://mygo.run/zh)(Go 原生 UI,无 WebView),代码在 `mygo/`,模块名 `github.com/ixxxxoooo/minicode`。分支 `feat/mygo`。Agent / IDE 是同一个窗口的两种形态,共用一份会话 / 编辑器 / 终端。

**设计稿 `docs/design` 是界面的唯一依据**;本文件与设计稿冲突时以设计稿为准,并同步改本文件。
历史做法、踩坑记录、各功能实现要点、已知缺口都归档在 `docs/dev-notes.md`(不自动加载,要查再翻)。

## 命令

```sh
cd mygo && unset GOROOT                       # 每次执行 Go 命令前(残留的 GOROOT 会让工具链对不上)
gofmt -l . ; go build ./...                   # 日常:能编译即可
go vet ./... && go test ./...                 # 集中测试阶段才跑
scripts/dev.sh [--demo|--real]                # 编译(先 go vet)→ 杀旧实例 → 在 target/dev/MiniCode Dev.app 里启动唯一的新实例
scripts/package.sh [--dmg]                    # 打包正式版 dist/MiniCode.app(--dmg 另出 dist/MiniCode-<版本>.dmg);版本取 mygo/main.go 的 version(去 -mygo)
scripts/publish.sh                            # 发布:先改 main.go 版本号并提交推送;在干净 HEAD 检出里打包 → appcast.json(平铺 + stable 嵌套)→ tag → gh release;MINI_DRY_RUN=1 只演练
scripts/readme-shots.py [agent|ide|dark|icons]  # README 截图:演示模式真实窗口 + 渐变背景 → docs/assets/shot-*.png(全新 MINI_HOME,不带本机偏好)
scripts/gen-lockup.py                         # 锁定版式与样式条 → docs/assets/lockup*.png、icon-styles.png
scripts/with-deepseek.sh <命令…>              # 真实场景测试:从 ~/.minicode/provider-keys.json 注入 DeepSeek 密钥后执行
```

运行模式优先级:`--demo` / `--real` / `MINI_DEMO=1|0` > 开发版记住的上次选择 > 真实。常用调试变量:`MINI_MODE=ide`、`MINI_PROJECT=<目录>`、`MINI_THEME` / `MINI_SCHEME`、`MINI_SCREEN` + `MINI_TAB`(设置分类)、`MINI_SESSION`(演示用 `fixbug`)、`MINI_OPEN=review`、`MINI_ED=<编辑器动作>`(含 `refs:demo` / `refs:fold` / `refs:sel:N` 直达引用面板各态;含 `lightbox` / `lightbox:/path` 打开图片灯箱)、`MINI_HOME`、`MINI_WIN_W/H`(完整列表见 README 与 `docs/dev-notes.md`)。
日志 `~/.minicode/logs/minicode.log`,崩溃 `~/.minicode/crashes/`。**密钥只在本机文件里,不进仓库 / 日志 / 文档 / 提交。**

## 工作流(按顺序,不能跳步)

1. **先看设计稿**:动手前读 `docs/design` 对应页面(入口 `js/app.js` 的 `PAGES`)。设计稿缺状态 / 菜单 / 空态 / 错误态时,**先补设计稿**并在页面「实现状态」列如实标注,再写代码。
2. **补缺口 + 逐项审查**:对着设计稿写,每改完一处只做静态审查与 `gofmt` + `go build`,**不跑 `go vet` / `go test`、不截图**。每改完一处**立刻提交并 `git push origin HEAD`**(中文 Conventional Commits,一个功能一个提交),只按路径 `git add` 自己改的文件。设计稿「实现状态」与 `js/status.js` 随同一提交更新。
3. **修复后立即重启开发版**(用户已授权):编译通过、提交推送后直接 `scripts/dev.sh`,让用户马上看到效果。这只是重启,不等于测试。
4. **集中测试只在用户明确说「开始测」时才做**:先 `gofmt` / `go vet` / `go test` 全绿,再用 **computer use** 在真实开发版里真点击、真输入、真快捷键走一遍并对照设计稿截图。`cmd/shot` 与 `go test` 不能代替;没用 computer use 走过的功能一律写「未测试」。
5. **汇报要诚实**:区分「已实现 / 部分 / 未做 / 未测试」,不把演示数据说成真实能力,写明是否已用 computer use 实测。

## 约定

- 回答、注释、文档、提交信息一律中文。不提交 `target` / `dist` / 密钥。
- **工具使用**:读文件用 Read、改文件用 Edit / Write、搜索用 Grep / Glob;Bash 只跑 `go` / `gofmt` / `git` / `scripts/dev.sh` 这类必须走命令行的事,不要用 `sed` / `cat` / 脚本代替读写文件(脚本替换没有「先读后改」校验,出过把文件截断的事故)。
- **日志**一律 `diag.L("包名")`,不用 `fmt.Println` / `log`;用户动作写 `Info`,降级写 `Warn`,高频事件只写 `Debug`;不输出密钥与环境变量值。
- **耗时操作**(磁盘遍历 / git / LSP / 子进程 / 解析)用 `diag.SpanSlow(模块, 名称, 阈值)` 包起来(阈值不能写 0);**后台任务一律经 `a.bg(name, work)`**;新 goroutine 用 `diag.Guard` / `diag.Recover`;界面线程不做阻塞 IO。
- **外部进程 / IO**:记录启动、退出码、耗时、stderr 摘要;失败时界面给下一步按钮,日志写原因。
- **可复现**:新增界面状态要能用 `cmd/shot` 参数或 `ide_debug.go` 的 `-ed` 动作直接到达。演示模式(`internal/demo`)要有覆盖各种状态(正常 / 空 / 失败 / 进行中)的模拟数据;入口、出口、取消、撤销都要有。

## 结构

数据流:`Backend` → `core.Event` → `core.State.Apply` → 界面;界面交互转成 `core.Command` 交回后端。演示与真实后端共用同一套界面。

| 目录 | 职责 |
| --- | --- |
| `mygo/internal/core` | 与界面无关的数据模型、`Event` / `Command`、状态归约器 |
| `mygo/internal/claude` | 真实后端:stream-json 协议、翻译器、会话索引与历史回放 |
| `mygo/internal/demo` | 演示后端:脚本化场景、沙盒项目 |
| `mygo/internal/app` | 全部界面(会话流、IDE、终端面板、源代码管理、设置、命令面板……) |
| `mygo/internal/kit`、`theme` | 控件与设计令牌(6 套主题 × 3 种布局) |
| `mygo/internal/term`、`lsp`、`vcs`、`workspace`、`projects` | 终端内核 / 语言服务器客户端 / Git / 文件系统抽象 / 项目与工作区存储 |
| `mygo/internal/diag` | 日志、慢操作计时、崩溃与卡死看门狗 |
| `mygo/cmd/shot` | 无窗口截图(辅助排查,不能代替实测) |
| `docs/design` | 设计稿(HTML 原型,浏览器打开 `index.html`,左上有全文搜索) |

## 必踩的坑(最关键的几条,其余见 `docs/dev-notes.md`)

- **输入回调里不能用 `c.WriteClipboard` / `c.ReadClipboard`**:`*ui.Context` 只在构建界面期间有效,回调里调用会静默什么都不做;要用 `c.Services()`。
- `Element.Bounds()` 是**上一帧**的矩形;`c.Key(...)` 要在构造输入控件之前;`ButtonBase` / `kit.RowEl` 默认居中且固定高度。
- 异步一律 `a.bg`:后台算完返回「应用结果」闭包,经 `a.win.Update` 回主线程;`a.win == nil`(测试)时同步执行。
- 应用没有翻译目录时,`SetLocale` 会被静默忽略:`main.go` 必须先 `SetLanguages("zh-Hans")` 再 `SetLocale`。
- `EvPromoteToPermission` 会把时间线里的 `*core.ToolCall` 换成 `*core.PermissionRequest{Tool}`:遍历条目找工具要同时处理两种,统一用 `doneToolOf(it)`。
- 图标名必须存在于图标表(`scripts/gen-icons.mjs` 从设计稿 `icons.js` 生成),否则画成 ✕。
- 吸顶的用户消息卡背景必须不透明(`Fill1.Over(BG)`),否则滚动时下层内容透出来。
- `go vet` 不接受其他包结构体的无键字面量,一律写键名。
- 向开发版发键盘事件前必须确认前台就是它;Computer Use 的字母键走输入法,终端正文不能 `computer_type`。
