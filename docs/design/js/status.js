/* 设计稿 v3.6 · 各页「实现状态」与缺口总表(v3.6:逐页对照代码重新盘点,补 ia 页与会话流 / 设置 / IDE / kit 的遗漏行,降级偏乐观的状态)
   规则(开发工作流):任何改动先改本稿再改代码;做完一项,就把对应行的状态改成「已实现」(并把「批」列写成 —)。
   · 自己带「实现状态」表的页面(specs2 / specs3 / specs4 / brand-spec)继续用自己的表;
   · 没有的页面(原型、整页功能、工作区标签、IDE 细节、规范)在这里用 STATUS 补一张,渲染在页面最下方;
   · 「缺口总表」页不再手写:它扫描全部页面里的状态表,把「部分 / 待做」自动汇总,所以改了哪页的状态,这里立刻同步。
   批次(B1…B8)是补全顺序,见缺口总表页的「补全批次」。
   @author ixxxxoooo */
(function () {
  const { tag, ic } = U;
  const { head, sec } = P.h;
  const S = { ok: tag('已实现', 'ok'), part: tag('部分', 'warn'), todo: tag('待做', '') };
  /** R(范围, 说明, 状态, 批次):已实现的批次写 — */
  const R = (name, desc, s, b) => [name, desc, s === 'ok' ? '—' : (b || ''), s];

  const BATCHES = [
    ['B0', '设计稿体检', '补齐各页「实现状态」、修正自相矛盾的标注、缺画面的先补稿、缺口页改为自动汇总'],
    ['B1', '多仓 Git 与搜索范围', '审查按仓库分组 + 逐个提交向导 · 图谱仓库下拉 · 搜索「仅某文件夹」 · 分支选择器仓库 chip · 多仓总览行动作 · 嵌套仓库与子模块'],
    ['B2', '项目 / 会话 / 排序 / 菜单', '会话行右键(删除 / 分叉等) · 方案 B 项目下拉接工作区 · 拖拽排序 · 各处右键与「···」菜单补全(SCM 标题 / 差异 / 终端 / 问题 / 输出 / 虚拟标签 / 自定义详情 / 会话流步骤行 / 多根顶层文件夹 / Markdown 标签均已做,图标与快捷键提示统一) · 文件新建 / 重命名 / 删除(F2 / ⌘⌫ / ⌘⏎ 已接线)'],
    ['B3', 'Git 与差异完整度', '冲突逐个切换与可编辑结果 · 差异块头常驻 · 行级暂存 · 提交并同步 · 推送被拒 / 切分支储藏 · SCM 列表 / 树 · 审查回退本回合'],
    ['B4', '会话流与任务', '前台命令实时输出 · 通知中心与系统通知 · 子 Agent / 任务停止 · 任务坞完整胶囊 · MCP 与上下文真数据 · 改动卡撤销 · 回合操作条 · 长输出限高 · 用户消息折叠 / 编辑'],
    ['B5', '编辑器 / 终端 / 语言服务', '问题面板重新检查 · 终端搜索全词 / 正则 · LSP 崩溃自动重启 · 代码操作 · 行号 Git 槽 · 迷你地图偏好 · 括号竖线着色 · 多光标 · 拆分编辑器组 · CodeLens'],
    ['B6', '设置与外壳', '设置搜索定位 · 需重启条真重启 · 快捷键录制 · 字号 · 品牌图标 / 托盘 / Dock · 窄窗响应式 · ⇧⌘N 新窗口 · 形态记忆'],
    ['B7', '整页功能', '浏览器真实抓取 · 自定义读写 ~/.claude · 用量限额 · 自动化 Cron / 项目 / 运行会话 · 画布落盘与写回 · 插件安装进度'],
    ['B8', '规范与打磨', '按钮 disabled / loading · 输入错误抖动 · 删除折叠与撤销 · 骨架 · 令牌补齐 · 动效退场 · 系统减弱动态联动']];

  /* ───────── 各页状态(只写没有自己状态表的页面) ───────── */
  const STATUS = {
    overview: [
      R('三套方案活画框、主题点选、工作台内 Agent ⇄ IDE', 'Go 版固定 scheme A + appearance.scheme 偏好;本页仅选型', 'ok'),
      R('「用这套看全部页面」', 'localStorage mc-design-v3c;Go 实现读 appearance.scheme', 'ok')],
    work: [
      R('Agent ⇄ IDE 原地切换(⌘E / 标题栏分段,440 ms 变形)', '会话 / 编辑器 / 终端全窗口各一份', 'ok'),
      R('四区域:导航栏 · 对话栏 · 工作区 · 状态栏;⌘B / ⌥⌘B / ⌘J 三个面板开关', '', 'ok'),
      R('会话流里点审查 / 计划 / 子 Agent / 文件 → 工作区开标签,自动展开并闪边', '', 'ok'),
      R('导航四分段(会话 / 文件 / 搜索 / 源代码管理)与整页入口、左下账号菜单', '', 'ok'),
      R('每个形态各自记住面板开合', '右侧栏 / 底栏 / 导航栏 / 对话占满与左栏分段按形态记忆;偏好 layout.memAgent / memIDE', 'ok', 'B6'),
      R('形态切换期间锁定指针事件(.morphing)', 'morph.go 过渡遮罩锁指针约 470ms', 'ok', 'B6')],
    welcome: [
      R('居中问候 + 状态行 + 输入框 + 四张起手卡(悬停上浮)', '欢迎页 64px 品牌光晕徽标(brand_logo.go 嵌入 PNG)', 'ok', 'B6'),
      R('最近会话列表', '演示 WelcomeRecent;真实跨项目索引最近 5 条,点击 OpenSession', 'ok', 'B2'),
      R('起手卡与最近会话的排版', '顺序:输入框 → 起手卡 → 最近会话(原先最近会话插在中间,5 行把起手卡挤到底);起手卡等宽:宽档一排 4 张、中档 2×2(原先固定 150 宽,放不下 3+1 折行,「审查改动」孤零零掉到第二行);最近会话 3 条、优先当前项目(标题写「在 <项目> 里」),当前项目没有会话才跨项目;布局回归 welcome_layout_test.go;已在开发版看过', 'ok', 'B6'),
      R('窄档起手卡单列', '', 'ok'),
      R('空会话下的拖拽引用落点', '欢迎页同样可以拖文件进输入框', 'ok'),
      R('侧栏切换已有会话时对话区过渡', 'session_load.go:回放完成前不显示欢迎页;无上一会话内容时「正在载入会话…」+骨架,否则暂留上一会话流;历史失败可行动错误;新会话仍欢迎页;未测试', 'part', 'B2')],
    palette: [
      R('⌘K 打开 / Esc · 遮罩关闭、欢迎列表(命令 / 会话 / 最近文件)与模糊查询', '', 'ok'),
      R('主题二级列表:上下即时预览,Esc 还原', '', 'ok'),
      R('右侧预览栏(命令说明 / 文件前 16 行)', '窄于 820px 隐藏预览栏;极窄时面板宽度随窗口收缩', 'ok', 'B6'),
      R('欢迎列表里的高频命令', 'palette.go palCommands:权限循环(⇧Tab 同源)、压缩 / 分叉 / 新会话、计划 / 审查 / 任务虚拟标签、除外观/通用外 12 类设置直达、演示/真实切换重启;addPal 无前置条件时置灰并 toast;未测试', 'ok'),
      R('⌘P 里输入 :行号 转到行', '与转到行共用 parseLineCol / parsePathLine(支持 :42、:42:8、路径:行:列),超界夹到末行;未测试', 'ok', 'B5')],
    settings: [
      R('4 组 14 类导航、改动圆点与悬停还原、外观(主题 / 方案 / 强调色 / 密度 / 圆角 / 动效)即时生效', '', 'ok'),
      R('设置内搜索', '命中后跳进分类并滚动高亮目标行(约 1.4s)', 'ok', 'B6'),
      R('需重启条', '「立即重启」自启动进程并退出(开发版 .app 路径)', 'ok', 'B6'),
      R('外观 → 时间线字号', 'appearance.fontScale → prefs_apply FlowFontSize / FlowCapFont / FlowMonoSize;助手 Markdown 走 mdOptions;present.ToolDetailLevel;flow / dock / composer 已读;未测试', 'ok'),
      R('外观 → 应用图标 / 菜单栏图标两组', '六样式×四外观 assets(内嵌)、Dock 弹跳/进度/角标、托盘五样式×八态/动画/菜单/数字/VoiceOver、应用菜单栏;访达换装已接(NSWorkspace.setIcon,签名缓存,关闭即还原),.icon 打包未接', 'part', 'B6'),
      R('快捷键:点键录制、冲突当场提示', '全局 / 编辑器 / 终端快捷键读偏好', 'ok', 'B6'),
      R('其余偏好', '启动形态、忽略规则、动效、多数编辑器/终端/SCM 键已接线;仍只存偏好或仅占位的项集中在下面「未接线」各行(含 browser.reader、about.autoUpdate、appearance.scrollbar、输出风格占位、privacy.crash/usage)', 'part', 'B6'),
      R('偏好未接线 · 模型 / 对话', 'claude.model/effort 为源(prefs_model_sync 迁移旧 models.*);chat.fold→flow_turn_fold;perm.mode→DefaultPermMode;chat.sys→ChatSystemPrompt/FillClaudeConfig;models.thinking/compact/compactAt、general.paste 已接线;perm.sandbox 行固定关(无外部键);models 页「输出风格」仅占位;未测试', 'part', 'B6'),
      R('偏好未接线 · 权限 / 运行环境', 'FillClaudeConfig:runtime.bin→ClaudePath、perm.mode、ExtraPermDirs、RuntimeExtraEnv(代理/超时);SyncClaudeSettingsFromPrefs 写权限/思考到 claude 设置;Bash 沙盒开关无 Claude 键(置灰)。未测试;额外目录与代理是否被 claude 子进程采纳需真后端核', 'part', 'B6'),
      R('偏好未接线 · 编辑器 / 终端 / SCM / 浏览器', 'editor.syntax→applyEditorPrefs;editor.blameGutter→ide_blame_ui;TermStartOpts 读 term.login/cwd/shell/scrollback;scm.diffMax/fetch 与 ScmFetchTick;worktree 列表+新建已接(settings_scm_wire,Computer Use 实测过新建/删除)。缺口:browser.reader 仅设置行未读。未测试', 'part', 'B6'),
      R('偏好未接线 · 隐私 / 高级 / 关于', 'privacy.level→ApplyDiagPrefs;adv.watchdog/LiveTaskOutput 已接;appearance.system→ApplySystemAppearance 切换浅深主题对。仍只存:privacy.crash/usage(行内写明无上报)、about.autoUpdate/about.channel(更新器未读);adv.flowMeasure 只存且注明不生效。未测试', 'part', 'B6'),
      R('设计稿有、设置页没有的行', '已补:appearance.waitAnim→SpinStyled;滚动条分段(仅 UI,运行时未读 pScrollbar)、claude 检测/版本、环境变量遮盖、运行环境三主机 netCheckBody、诊断包导出、doctor 折叠、崩溃列表、settings 导入导出、Worktree 列表、关于页 GitHub 链。仍弱:输出风格无键;开源致谢非完整许可列表;服务商行内「测试」已走真实 testProvider + ProbeProvider。未测试', 'part', 'B6'),
      R('设置里「检查器」措辞与设计稿不一致', 'Go:general.insp 与快捷键分组已用「右侧工作区/右侧栏」(settings_shortcuts 无「检查器」);layout_mem 读 general.insp。缺口:Go 源码注释(pages.go)仍写检查器;设计稿 overview 方案卡示意图是否仍用旧名未在本任务核对', 'part', 'B6'),
      R('规则 / 环境变量 / 语言服务器 / 项目信任', 'settings_rules_env + claudecfg 三层规则/环境变量;settings_lsp 后台检测与 lsp.server.*;settings_trust 读写 hasTrustDialogAccepted。单测覆盖;界面未测试', 'ok'),
      R('服务商自定义与预设', 'prov_settings.go + claudecfg:预设/自定义 CRUD、密钥 0600、ProbeProvider/FetchModels(含 Real DeepSeek 联调);官方行 meta(OAuth/演示账号)、清单空态「获取+手动添加」双按钮、演示 Mapping.Enabled 对齐 providers.js defaultState。包内单测通过;设置页全链路未测试。缺口:连通成功只在行内标签(无 toast)。未测试', 'part', 'B6'),
      R('获取模型列表与勾选', 'prov_settings provInventory:FetchModels 骨架/空/失败/不支持/缺密钥、手动添加、过滤/全选/清空;演示 demo.DemoFetchModels 覆盖 example.com/legacy。未测试', 'part', 'B6'),
      R('模型映射下拉', 'claudecfg.Slots/MappingStatus + ProviderEnv;prov_settings 五槽 UI;运行中 pvPushEnv 跳过(与设计一致)。未测试;环境变量是否进真实 claude 请求需 MINI_REAL_CLAUDE 核', 'part', 'B6'),
      R('会话内按服务商选模型', 'prov_picker cfgModels、popups cfgPopup、composer/statusbar 分组与管理入口已接;官方槽仍读 a.st.Models;切换 toast 的「下一条消息起生效」副句已有(modelSwitchToastNote);弹层 picks 已写入 providers.json 并按服务商恢复;胶囊缩写 + tooltip 已接(cfgChip/statusbar)。映射→环境变量有 MINI_REAL_CLAUDE 集成测试;界面未测试', 'part', 'B6'),
      R('模型品牌 logo', 'gen-logos.py→kit/logos_gen;prov/composer/statusbar/ctx_panel 等已 kit.Logo。缺口:单色无渐变(稿面可选);与设计彩色渐变版未对齐。未测试', 'part', 'B6'),
      R('窄于 900 时左侧分类收成下拉', '左侧栏隐藏,顶部分类下拉', 'ok', 'B6')],
    forms: [
      R('单窗口、无融合 / 独立 IDE 窗口;两种形态共享会话 / 编辑器 / 终端', '', 'ok'),
      R('⇧⌘N 新窗口(再起一个进程)', '--new-window / 菜单 / 命令面板;从属进程不写偏好,窗口错开 32px', 'ok', 'B6'),
      R('启动形态(Agent / IDE / 上次)', 'general.start + layout.lastMode 已接 main 启动', 'ok', 'B6'),
      R('形态切换的内容编排(甘特图)', 'morph_content.go morphWrap 已挂 chat/side/workspace 三轨;标题栏形态滑块未分轨(仍跟 morph.go 整体变形);motion 关时直达。未测试', 'part', 'B6')],
    ia: [
      R('信息架构四条原则、4 组 14 类设置树、迁移对照表', '以 settings 页的实现为准,设置搜索兜底(settingsHits / jumpSettingRow)已接', 'ok', 'B0'),
      R('「服务商状态与交互清单」(8 行)', 'ia 页 8 行映射 settings 服务商 CRUD/模型列表/映射/会话选择四块;prov_settings + prov_picker 等已接线。弹层 picks 已持久化;运行配置胶囊第三方 logo + 缩写/短模型名 + tooltip 全名已接。未测试', 'part', 'B6'),
      R('入口矩阵 ENTRY(同一内容别处只是跳转)', '工作区虚拟标签入口都走 openVirtual', 'ok', 'B0')],
    flows: [
      R('没有「检查器」:详情只落在工作区标签,每个内容一个家', '', 'ok'),
      R('打开规则矩阵:审查 / 计划 / 子 Agent / 差异 / 终端只读 / SCM 预览标签', '', 'ok'),
      R('预览斜体、双击固定、虚拟标签全窗口单份;导航历史 ← →', '', 'ok'),
      R('整页功能的出口:再点已选中的整页图标 / 点导航分段 = 回到工作台', 'side.go togglePage / leavePage(侧栏图标、IDE 图标条、导航分段共用);此前整页打开后这两处点了没反应,只能靠别的入口;page_exit_test.go;已在开发版实测', 'ok')],
    resp: [
      R('对话栏按自身宽度三档(≥640 / 420–639 / <420)', '', 'ok'),
      R('工作区按自身宽度三档(≥600 / 520–599 / <520:迷你地图、统一 diff、面包屑)', '', 'ok'),
      R('窗口变窄自动收起 / 恢复两侧栏', '固定断点 1100 / 900 / 720 + 24px 回差', 'ok', 'B6'),
      R('Agent 形态:工作区在窄窗变成滑入浮层;<720 导航栏变抽屉', 'overlays.go + kit.DrawerSlide + overlay_motion;responsive 断点 720。静态核对未发现矛盾,滑入曲线/遮罩需实机验证', 'part', 'B6'),
      R('IDE 形态:窄窗对话栏收成图标条 + 浮层', 'dockRail + dockOverlay + DrawerSlide 已接。静态核对未发现矛盾,需实机验证', 'part', 'B6'),
      R('整页功能的窄窗规则(自动化 / 画布 / 浏览器 / 自定义 / 用量,<1000 / <900)', 'pageNarrowMid/winW<900:custom/usage/automation/canvas 窄布局已接(窗口宽非容器)。未测试', 'part', 'B6')],
    'ide-review': [
      R('审查虚拟标签:统计头、行内 / 左右、逐文件折叠、已查看勾选', '', 'ok'),
      R('块级接受 / 撤销(撤销写盘)与块评论;评论只在「发送给 Claude」后进对话', '', 'ok'),
      R('空态「本轮还没有改动」', '演示 -ed review:empty 直达', 'ok', 'B3'),
      R('页脚「回退到这一回合之前」', '接 CmdRewindFiles(最近用户消息)', 'ok', 'B3'),
      R('页脚回合时间 / 文案', '取 TurnStats / Durations', 'ok', 'B3'),
      R('行内评论串回复', '评论上「回复」续串;发送给 Claude 带行引用', 'ok', 'B3'),
      R('按仓库分组 + 多仓「逐个提交」向导', '分组头 / 横幅 / 向导步骤已做(设计稿 multi-git 页)', 'ok', 'B1')],
    'ws-plan': [
      R('计划虚拟标签:头部 Build 三键、执行进度、Markdown 全文;计划卡「查看」开标签', '', 'ok'),
      R('流式生成时自动打开计划标签', 'EvPlanUpdated 流式时自动 OpenWorkspaceTab', 'ok', 'B4'),
      R('选中文字批注 → 继续规划', '计划标签按段落「批注」按钮填入输入框+继续规划(无富文本选区)', 'ok', 'B4')],
    'ws-sub': [
      R('子 Agent 标签:任务 / 过程 / 结果、引用到对话、复制全部', '', 'ok'),
      R('会话流子 Agent 卡与标签里的「停止」', '会话流卡片与标签页 CmdStopTask', 'ok', 'B4'),
      R('过程流式滚动', '任务标签有跟随输出与 ↓ 回到底部;子 Agent 标签运行中过程区自动贴底(无单独开关)', 'ok', 'B4')],
    'ws-task': [
      R('任务输出标签:跟随输出、停止任务', '', 'ok'),
      R('「在终端面板中打开」', '展开底部面板并选中只读 cmd: 标签(与命令卡「在终端面板查看」同路径)', 'ok', 'B4'),
      R('重启任务 / 显示 pid', '演示 CmdRestartTask;真实 claude 置灰+原因;run id / pid 行内显示', 'ok', 'B4'),
      R('正常运行时注入任务', 'translate task_*+任务坞+会话流 TaskNote(flow_note.go);后端 TestRealBackgroundTaskAndSubagent;界面仅 ui.Tester/无真实端到端;未测试', 'part')],
    flow: [
      R('步骤行(进行 / 错误 / 拒绝)、探索分组、工具展开 diff / 终端、待办、权限 / 提问 / 计划卡、提示三级', '', 'ok'),
      R('长输出限高折叠(12 行 → 196px,展开 480px)', '工具 / 终端 / 文本与 Markdown 代码块', 'ok', 'B4'),
      R('折叠工具分组的「最新一步」滚动行', '探索组折叠且运行中 22px 滑入行', 'ok', 'B4'),
      R('改动卡的「撤销」按钮', '逐文件 revertHunk', 'ok', 'B4'),
      R('回合操作条:复制 / 重试 / 分叉', '复制 / 重试 / 分叉(真实 fork: 会话);演示 fork 会话', 'ok', 'B4'),
      R('用户消息卡:长消息折叠 / 编辑', 'flow_user_edit.go 就地编辑;真实 CmdEditUser/resume_session_at,无 uuid 置灰+分叉提示;局部截断无后端分叉核验;未测试', 'part'),
      R('工作区起始页(本轮 / 终端 / 打开三组列表;终端行列出已开终端 + 新建)', 'ws_empty.go 三组列表', 'ok', 'B4'),
      R('发送 / 停止按钮为圆形', '设计稿与 Go 版同步', 'ok', 'B4'),
      R('会话样式(跟随布局 / 对话气泡 / 通栏=与正文同宽的圆角卡片;设置 → 外观 → Agent 会话)', 'appearance.flowStyle 三档', 'ok', 'B4'),
      R('用户消息吸顶(回合内 sticky,下一轮顶走上一张,最多一张;点击就地编辑 / 取消 / 发送;设置 → 对话与提示词,默认开)', 'flow_user.go:Sticky+chat.stickyUser、不透明 Raised.Over(BG)、吸顶时短阴影 ShadowStuck+细描边(去掉大偏移 Shadow2 与下沿渐隐,它们叠起来会拖出长暗带);编辑态卡片外圈一道描边、内部输入区无框无聚焦环;未测试', 'ok'),
      R('行内代码跟随主题色(助手正文 + 用户消息)', 'mdtheme.go inlineCodeColors:字色 AcTx、底色强调色 11%,左右 5px;mdOptions 缓存键含强调色;设计稿 flow.css `.md code, .msg-u code`;未测试', 'ok'),
      R('命令卡 / 代码块输出:内部滚动 + 拖选复制', 'flow_fold.go foldScroll 按 foldKey 跨帧保留滚动状态(原先每帧新建 ScrollState,滚轮被拉回顶部且不向外层传递);命令 / 输出卡的正文 Selectable(拖选、双击选词、⌘C),Markdown 围栏代码块沿用 Markdown 视图的统一选区;未测试', 'ok'),
      R('「回到最新」悬停底色不透明', 'flow_scroll.go:悬停 Fill2.Over(Raised),不再透出后面的文字;设计稿 .to-latest:hover 同步;未测试', 'ok'),
      R('输出面板:鼠标拖选 / 双击选词 / ⌘C 复制', 'ide_output.go 整列日志 Selectable 共用一个选区;行右键补「复制所选」(无选区置灰并提示);未测试', 'ok'),
      R('命令卡语法高亮(命令行 + 输出)', 'flow_hl.go:shellSpans(命令名 / 子命令 / 标志 / 引号 / 变量 / 管道·重定向 / 注释)与 termLineSz(错误·警告·通过、URL、路径:行:列、提交哈希、git 状态码与 ## 分支行),颜色取主题语法色与状态色,按「主题+字号+文本」缓存;命令头与输出共用;设计稿 flow.js F.term + flow.css .sh/.term;未测试', 'ok'),
      R('侧栏工作区 / 项目右键与「编辑项目」', '工作区展开只列会话;右键 = 编辑工作区… / 编辑项目…(新对话 · 编辑 · 访达 / 复制 · 移除);全应用右键菜单改为窗口内自绘(SetMenuPresentation);其余菜单的图标与快捷键提示已经 menu_item.go 的 menuItem 统一补齐;未测试', 'ok', 'B2'),
      R('右键菜单图标与快捷键提示统一(menuItem 单点维护)', 'menu_item.go:文字 → 设计稿图标表,「文字  快捷键」拆成右侧提示;图标名取自 kit.iconPaths;未测试', 'ok', 'B2'),
      R('会话流步骤行 / 差异卡头 / 命令卡头右键菜单', 'menu_extra.go stepMenu + flow.go cardM:复制路径 / 命令 / 输出 / 网址、访达、查看差异 / 审查标签、终端面板;未测试', 'ok', 'B2'),
      R('多根顶层文件夹右键:在源代码管理中查看 / 设为主文件夹 / 从工作区移出…', 'menu_extra.go;设为主 / 移出走 wsMutate(保存后重新进入工作区),主文件夹与只剩 2 个文件夹时「移出」置灰写明原因;未测试', 'ok', 'B2'),
      R('Markdown 标签 ··· 附加项(切换预览 / 源码、只读预览、格式化文档)', 'menu_extra.go mdMenuItems:切换预览/源码、只读预览置灰勾选自 mdUnsafe、格式化文档;未测试', 'ok'),
      R('输入框 MCP chip 与弹层', 'composer_mcp.go mcpChip/弹层+CmdRefreshMcp;composer_mcp_tools.go 工具级开关写 permissions.deny;未测试', 'ok'),
      R('会话流 · 后台任务完成通知行(TaskNote)', 'flow_user.go flowTail→taskNoteLine(flow_note.go);core.TaskNote 归约;flow_test TestFlowGapsTaskNoteAndFold;未测试', 'ok'),
      R('会话流 · 并行子 Agent 汇总行「并行 N 个」', 'flow.go stepGroup→parallelSubGroup(flow_parallel.go)连续≥2 子 Agent 卡;未测试', 'ok'),
      R('会话流 · 回合折叠「已处理 1 分 13 秒」', 'flow_turn_fold.go 读 chat.fold;flow_user.go flowRound 折叠步骤组;失败/运行中/待处理保持展开;未测试', 'ok'),
      R('用户消息 · 悬停动作条(回退文件 / 重试 / 复制)', 'flow_user.go userMsg 悬停→userMsgHoverBar(flow_user_actions.go)回退/重试/复制;未测试', 'ok'),
      R('「回退文件」预览对话框(用户消息入口)', 'flow_rewind.go userRewindBegin→CmdRewindFiles 预览;flow.go flowRewindCard 确认;审查页 review_turn.go 亦有入口;未测试', 'ok'),
      R('用户消息里的图片引用 / 缩略图 / 全屏', 'flow_user_img.go userMsgImages+img_cache 后台解码;文字里的「[Image #N]」占位标记不再显示,缩略图固定高 64、宽按比例夹在 64–112 并整排排在文字下方,点哪张在下方展开哪张(按比例缩到 360×280 内);未测试', 'ok'),
      R('输入框「+」添加附件与斜杠命令', 'composer.go plusAttachMenu(composer_attach.go)+slashPopup(composer_slash.go);超长粘贴发送前 expandPasteRefs;自定义斜杠选中后仅填入草稿;未测试', 'ok'),
      R('审批多张排队「1 / 3」与处理后输入框淡入、恢复草稿', 'dock_pending.go dockBadge 1/N+dock_ask 接线;syncDockComposer 处理后 composerInAt 淡入并恢复草稿;未测试', 'ok'),
      R('排队卡「停止并发送」', 'dock.go queueCard 悬停停止图标→composer_queue.go qStopAndSend;未测试', 'ok'),
      R('排队消息带图片', 'queueCard splitDraftImages+queueImgBadge;退回编辑还原草稿;submitPrompt 带 Images;未测试', 'ok'),
      R('提示卡的动作(重试 / 压缩)与权限「你拒绝了…」专用行', 'flow.go noticeActions;flow_note.go permissionDenyNote;未测试', 'ok'),
      R('会话流内 Mermaid 图表卡与全屏查看', 'flow_mermaid.go 图表卡+internal/mermaid;mermaid_viewer.go 全屏适应/±/拖拽/Esc;缺口:全屏未接滚轮(mmWheelZoom 未调用)、gantt/class/er 等 ErrUnsupported;画布页仍简版;未测试', 'part'),
      R('task-spec 页实现状态表', 'specs2 待办 / 尾读 / 通知 / 分叉四行', 'ok', 'B0')],
    overlays: [
      R('命令面板、历史会话、分支选择器、账号菜单、运行配置弹层', '', 'ok'),
      R('IDE 通知中心(铃铛 / 历史 / 右下角叠放)', 'notice_ui.go 铃铛/面板/noticeToasts(shell.go);.app 内 mac_notify OpenSessionFromNotify;测试/裸二进制无 UN 点击回会话;未测试', 'part'),
      R('任务坞停滞 / 已结束 / 监听 / 工作流胶囊', 'chat.go taskDock(task_dock.go)六类胶囊+停滞/停止;flow_test TestTaskDockCapsules;未测试', 'ok'),
      R('上下文弹层(输入框圆环 / 状态栏;分类明细 / 阈值刻度 / 账户限额 / 刷新 / 压缩;常态 / 接近阈值 / 压缩中 / 加载 / 无限额 / 失败)', 'ctx_panel.go 六态+get_context_usage;get_usage 无数据时 none 说明;未测试', 'ok'),
      R('更新进度胶囊四态', 'update_progress.go+update_engine.go;statusbar 胶囊只在检查中 / 有新版本 / 下载 / 校验 / 解包 / 待重启 / 失败时出现,「已是最新」不占状态栏(只在设置 → 关于里显示);开发版 dev.sh 重启;未测试', 'ok'),
      R('确认框(模式切换 / 未保存 / 删除)', '通用确认框已用于会话删除 / 移除项目 / 切换模式', 'ok', 'B2'),
      R('账号菜单的真实身份与用量', 'popups.go acctPopup+acct_popup.go ReadOAuthProfile/用量;演示 demo.AcctMenuDemo;未测试', 'ok'),
      R('上下文弹层的「查询失败」独立状态', 'ctx_panel.go ctxState 返回 err+红色重试条(CmdRefreshUsage);与主上下文弹层同一面板;未测试', 'ok')],
    'ide-editor': [
      R('迷你地图(语法色块、滑块拖动)、Git / 搜索 / 同词 / 诊断 / 光标刻度、概览标尺、同词高亮、窄档藏地图', '', 'ok'),
      R('行号槽的 Git 改动竖条', 'DiffBase 接线;无 HEAD 或非 Git 文件不画竖条', 'ok', 'B5'),
      R('迷你地图滑块 / 宽度 / Git 刻度三个设置', '三个偏好已接到地图绘制', 'ok', 'B5'),
      R('CodeLens(引用数 / 测试 / 问 Claude)', '库无独立 CodeLens 行,用符号行上方叠层;测试数由本地符号索引推断(Go/Rust/Py/TS);窄窗与粘性滚动错位时 CodeLens 下移', 'ok', 'B5'),
      R('迷你地图悬停显示源码', '悬停该行左侧浮层显示源码摘要', 'ok', 'B5'),
      R('追溯栏(行号左侧逐行显示提交,同提交只在首行显示作者)', 'ide_blame_ui.go:同提交只在首行显示作者、色条按哈希取色、加载中 / 暂无历史 / 非仓库状态、自动换行时隐藏并在顶栏说明;偏好 editor.blameGutter;未测试', 'ok', 'B5')],
    'ide-nav': [
      R('粘性滚动、查找替换、悬停、预览定义、跳转定义 / 引用 / 类型 / 实现、补全、重命名、格式化、跳转历史', '', 'ok'),
      R('代码操作 ⌘.', 'LSP codeAction + 本地兜底;无服务器时仍为本地菜单', 'ok', 'B5'),
      R('选中工具条 / ⌘I 行内编辑', '真实模式 claude -p Haiku 生成替换预览,Esc/取消可中断;演示仍走占位', 'ok', 'B5'),
      R('行内审查(悬停接受 / 撤销)', '改动行悬停接受写入审查状态 ok、撤销走 revertHunk', 'ok', 'B5'),
      R('括号配对竖线 / 着色', '竖线:prefs_apply 接 editor.bracketGuide→st.NoIndentGuides;括号匹配高亮:editor.bracketMatch;着色:ide_brackets.go 包装 wrapBracketLang/bracketHL 按深度换 Token 种类,syncBracketLang 在 ide_overlay 接线;字符串/注释内括号跳过;未测试', 'ok'),
      R('多光标', 'ide_multicursor.go:状态栏芯片+菜单(mcStatusChip/mcPop)、⌘D/⇧⌘L/行尾添加光标/回到单光标、1000 上限(mcClamp)、tbCodeEditable 对只读/差异整标签/MD 预览/虚拟标签置灰并 toast、命令面板与 -ed multicursor。差异标签整侧不可编辑(非左右分侧)。缺口:库自带 Esc/⌥点击/一次撤销整批多光标未在应用层实测;未测试', 'part', 'B5'),
      R('转到行(⌃G)', 'ide_goto.go 自建浮条:正常 / 超界夹末行并提示 / 非数字红描边 / 行:列 / 空文件五态,与 ⌘P :行号 共用解析;-ed gotoline:<文本>;状态栏「行,列」可点击打开(与 ⌃G 同入口);输入框打开时留空、占位符是当前行号(库无「聚焦即全选」,预填会让新输入被追加);已在开发版实测:⌘K 输入 :10 → 回车跳到第 10 行、状态栏点击打开浮条;键盘 ⌃G 在编辑器获得焦点后才触发(终端焦点时 ⌃G 归 shell)', 'ok', 'B5'),
      R('块注释(⌥⇧A)', '⌥⇧A 切换块注释,多光标逐选区处理;语言无块注释时退回行注释并在顶栏说明,完全无法注释时 toast;命令面板「块注释」、-ed blockcomment;未测试', 'ok', 'B5'),
      R('自动保存之外的「外部修改」处理', 'ide_extmod.go:聚焦约 1 s / 失焦 5 s 轮询修改时间 + 大小;无本地修改自动重载 + toast;有修改顶栏「重新载入 / 保留我的 / 查看差异」(磁盘 ↔ 编辑器只读差异);文件被删除顶栏 + 标签删除线;自写盘与自动保存不自触发、同一次修改不重复提示;-ed extmod:auto|clean|dirty|deleted。缺口:「保留在编辑器」用确认框原路重建而非另存为,演示 Mem 文件系统不走轮询;未测试', 'part', 'B5'),
      R('面包屑每一级的同级下拉', 'ide_crumb.go:路径段点开列同级目录 / 文件(文件夹在前、当前项置灰、多根顶层段),符号段列同级符号并跳转,空态「没有同级项」/ 失败态,工作区 <520 不渲染;-ed crumb:file|sym。缺口:下拉标题行、键盘导航未验证、符号取本地索引而非强制 LSP documentSymbol;未测试', 'part', 'B5'),
      R('编辑器组向右 / 向下拆分', '两组标签 + 共享缓冲区;标签可跨组拖动', 'ok', 'B5'),
      R('Claude 行内补全(✦)', '真实 Haiku claude -p + SetSuggestion;演示占位,Tab 接受', 'ok', 'B5'),
      R('工作区外文件打开', 'workspace.WithExternal + openAnyFile:自定义页/⌘P 绝对路径/会话流引用/访达双击;标签与面包屑「工作区外」chip、~ 路径;Git 降级;LSP 以文件目录为根;真实模式包装 ide.fs,演示 Mem 不包装;未测试', 'part', 'B5')],
    'ide-search': [
      R('⇧⌘F、选区预填、大小写 / 全词 / 正则、包含排除、文件折叠、虚拟列表、点击预览、替换预览、悬停忽略、全部替换', '', 'ok'),
      R('单条替换(⏎)', '替换行 ⏎ 替换当前选中匹配', 'ok', 'B5'),
      R('多根:「仅某个文件夹」范围筛选', '范围 chip + 下拉 + ✕ 清除,文件夹右键「仅在此文件夹搜索」', 'ok', 'B1'),
      R('搜索结果右键菜单', '匹配行 / 文件行已做;「在侧边预览」= searchPreviewSide(侧边组开预览并定位命中处,没有拆分时先向右拆分);未测试', 'ok', 'B2'),
      R('空 / 错误 / 搜索中 / 结果截断四种状态的画面', '空查询 / 错误 / 搜索中 / 截断摘要已补', 'ok', 'B5')],
    'ide-problems': [
      R('状态栏错误 / 警告计数 → 问题面板、按文件分组、点击跳转、未安装语言服务器的安装卡', '', 'ok'),
      R('「重新检查」按钮与「全部 / 仅错误」筛选', '刷新按钮触发工作区重检;演示模式同步筛选', 'ok', 'B5'),
      R('工作区级诊断', 'internal/lsp 声明 diagnostic 能力并实现 workspace/diagnostic / textDocument/diagnostic(previousResultId + 流式片段),按能力选 服务器 pull / 逐文件拉取 / didOpen 扫盘 三条路径并显示来源;问题面板分段「已打开 | 工作区」、进度 + 取消($/cancelRequest)、截断(lsp.workspaceDiagLimit 默认 2000)、空 / 错误态 + 重试;-ed wsdiag:scan|done|trunc|empty|err。阶段三实测:协议单测通过;本机 gopls 0.22 / pyright 都不声明 diagnosticProvider,rust-analyzer 在本机 initialize 即退出(rustup 代理),所以真实环境目前一律走 didOpen 扫盘兜底,pull 路径没有任何真实服务器验证。触发:lspWorkspaceScan 仅在「重新检查」或切工作区分段时调用,进项目不自动扫。跳过:skipWorkspacePath 硬编码目录;偏好 lsp.workspaceDiagGitignore 打开会包含这些路径,不解析 .gitignore;未测试', 'part', 'B5')],
    'ide-scm': [
      R('分支行、拉取 / 推送 / 同步、储藏、提交框(⌘↩ + AI 说明)、暂存 / 更改 / 冲突分组、悬停操作、内嵌图谱、审查入口', '', 'ok'),
      R('提交分裂按钮里的「提交并同步」', '提交后 pull --ff-only + push', 'ok', 'B3'),
      R('列表 / 树切换', 'scm.viewMode 列表/树,目录折叠记忆', 'ok', 'B3'),
      R('多仓库总览行的悬停动作、逐仓转圈与失败重试、冲突徽标', '含右键菜单;全部同步跳过冲突仓库', 'ok', 'B1'),
      R('文件行 / 分组 / 仓库行右键菜单', '放弃更改统一加确认框', 'ok', 'B2')],
    'ide-diff': [
      R('SCM 单击预览 / 双击固定、左右 / 统一(记住偏好)、词级高亮、块级暂存与取消暂存', '', 'ok'),
      R('块头常驻显示动作', '差异注解条常驻块头动作(HunkActions 关)', 'ok', 'B3'),
      R('行级暂存(选中行)', '行号拖选 + 浮条暂存/还原', 'ok', 'B3'),
      R('忽略空白差异', '顶部条开关 scm.ignoreWs', 'ok', 'B3')],
    'ide-merge': [
      R('三方冲突标签、逐处接受当前 / 传入 / 双方、标记为已解决(写盘 + 暂存)', '', 'ok'),
      R('「冲突 1 / 2」逐个切换', '上一处/下一处,一次只显示一处', 'ok', 'B3'),
      R('三方「结果」可编辑区', '保留 diff3 base;结果 TextArea 可编辑', 'ok', 'B3')],
    'ide-git': [
      R('审查 vs 源代码管理分工、块 / 词级差异、追溯、文件时间线、图谱', '', 'ok'),
      R('推送被拒的引导、切分支时的储藏对话框', 'non-fast-forward 引导;脏工作区储藏确认', 'ok', 'B3'),
      R('图谱提交菜单:与工作区比较 / 回退…', '与工作区比较先选文件(只读差异);回退 = reset --keep', 'ok', 'B2'),
      R('图谱菜单「与 HEAD 比较」与「选中两个提交比较」', 'scm_compare.go / scm_graph_picks.go:多选提示条 + 取消、双选后「比较」、相同提交 toast、取活动仓库;-ed cmp2 / cmphead。空文件列表仍只有 toast;未测试', 'ok', 'B3'),
      R('推送被拒:「拉取并变基 / 拉取并合并」分按钮', 'git_dialogs.go scmPushGuide + git_push_actions.go:拉取并变基/合并分按钮(进行中 Loading);冲突时同条引导打开冲突、中止变基(RebaseAbort)/中止合并(MergeAbort);演示 Mem 可造冲突。缺口:失败 stderr 主要靠日志与 friendlyGitErr,未用 Computer Use 实测整条引导', 'ok', 'B3'),
      R('审查 → 一键暂存已接受的改动', 'review_stage.go:多仓库按 repoAt 暂存,部分失败提示 + 撤销条只覆盖成功数,无已接受改动时置灰并说明;缺口:分组审查页还没有按仓库各放一颗暂存钮;未测试', 'ok', 'B3'),
      R('行内追溯的悬停提交卡', 'ide_blame_ui.go blameCardOverlay:悬停 300ms 出卡,移出追溯/卡片后 blameCloseDelay 250ms 收起(可移入卡片,换行即关;blame_card_delay_test.go)。未提交行「查看提交/差异」置灰(Tip 说明)。缺口:未提交行无快捷开 HEAD↔工作区差异;静态核对未发现其它矛盾,悬停交互需实机验证', 'part', 'B3'),
      R('储藏条目展开看各文件差异', 'scm_stash_expand.go + vcs/stash_diff.go:展开读取中 / 空储藏 / >8 个文件折叠「还有 N 个…」、文件右键「打开差异 / 复制路径」、-ed stashfiles;缺口:储藏父行本身没有右键;未测试', 'ok', 'B3')],
    automation: [
      R('周历(翻周 / 今天)、卡片列表筛选与启停、右侧编辑抽屉、新建 / 删除、真实落盘', '', 'ok'),
      R('计划 Cron', '五段 cron 解析、FiresOn/Due 到点触发、下次运行时间', 'ok', 'B7'),
      R('项目选择 / 权限选择', '新建模态与编辑抽屉:项目分段 + 权限分段', 'ok', 'B7'),
      R('立即运行', '真实 claude -p 流式+取消;演示 RunAutomationDemo 流式+取消+失败样例', 'ok', 'B7'),
      R('最近运行可点进会话', 'auto_runner.go:runAutomationCore 回写 SessionID;openAutomationSession 索引未命中触发 RefreshProjects;无 id 置灰 autoNoSessionReason;auto_runview_test;未测试', 'ok'),
      R('新建模态与失败 / 运行中状态', '设计稿 behave.js au-new/au-run;Go 模态+运行浮层', 'ok', 'B7')],
    canvas: [
      R('块编辑 / 预览、大纲定位、版本提交与回退、导出 Markdown', '', 'ok'),
      R('落盘', '~/.minicode/canvases/*.json 原子写;损坏改名留底;多画布列表', 'ok', 'B7'),
      R('让 Claude 修改', '块引用+ApplyClaudeReply;在画布页回合结束自动写回并落盘', 'ok', 'B7'),
      R('图表全屏查看 / 源码', 'canvas_viewer.go:适应/100%/±/平移/Esc/坐标;绘制 canvas.Diagram 方框(未接 internal/mermaid);未测试', 'part', 'B7'),
      R('窄窗右栏收成底栏', '宽<900 时大纲/版本落底部', 'ok', 'B7'),
      R('左侧画布列表(多画布)', 'Library 多画布:切换/新建/重命名/删除+撤销;演示多套 demoPgCanvas', 'ok', 'B7')],
    browser: [
      R('后退 / 前进 / 刷新 / 收藏、摘要 · 大纲 · 链接、齿轮直达设置', '', 'ok'),
      R('真实网页抓取与阅读模式正文', 'net/http 抽取正文;超时/大小上限;演示源并存', 'ok', 'B7'),
      R('「与此页面对话」的页面引用', '正文框选→@浏览器:url#段+填输入框;日志 RedactURL 脱敏', 'ok', 'B7'),
      R('系统浏览器打开 / 隐私设置', '地址栏外链;设置 browser.fetch 可关抓取', 'ok', 'B7')],
    customize: [
      R('七类左栏、筛选与作用域、开关 / 新建 / 删除、详情表单', '界面与校验完整;演示内存+演示多套', 'ok'),
      R('读写真实的 ~/.claude 与项目 .claude', 'OpenDisk+Persist/SaveMCP 七类写盘;cuMCPEditDlg;Skill 重命名。缺口:已安装插件详情无编辑表单回写 JSON;演示不落盘;未测试', 'part', 'B7'),
      R('插件市场安装进度', 'cuInstallPlugin+RunPluginInstall;演示四相位,真实安装中多仅 phase0 至结束;未测试', 'part', 'B7'),
      R('MCP 授权 / 重连 / 工具开关接真实状态', 'mcp_status+mcp list 兜底+init 初值;login 仍手工', 'part', 'B7'),
      R('命令 / 子 Agent 试运行、钩子试运行', 'custom_wire.go:cuTryCommand/cuTryAgent/cuTryHook;未测试', 'ok'),
      R('Skills / 规则「在编辑器里打开」', 'cuOpenPath 解析 ~/ 与相对路径,Skill 补 SKILL.md;未测试', 'ok')],
    usage: [
      R('7 天 / 30 天 / 本月、汇总卡、柱图悬停、按模型 / 项目 / 贵会话、导出 CSV、空数据加载态(真实扫描)', '', 'ok'),
      R('账户限额与刷新', 'CmdRefreshUsage+usage_page 抽屉;Checking/Unavailable 文案与重试;第三方网关常无限额数据;未测试', 'part', 'B7'),
      R('贵会话点击跳转到会话', 'TopSessions 带 SessionID 时 OpenSession', 'ok', 'B7')],
    comps: [
      R('按钮 / 图标按钮 / 胶囊 / 标签 / 计数 / 分段 / 开关 / 标签条 / 列表行 / Spin / Meter / Chk / Dot / Kbd', '', 'ok'),
      R('按钮 disabled / loading 态', 'kit.Btn Disabled/Loading;提交/安装/重试等接线(仅按钮;图标按钮 / 开关 / 胶囊的禁用与加载态见 states 页)', 'ok', 'B8'),
      R('图标按钮 / 开关的 disabled 与 error 态', 'kit.IbOpt/TogOpt 已有 Disabled/Loading/Error;业务侧很少传 Disabled;未测试', 'ok'),
      R('输入框统一封装与错误描边', 'kit.Field:设置单行、项目/工作区/自动化/自定义新建对话框(提交框除外)', 'ok', 'B8'),
      R('菜单 / 弹层壳', 'PopIn/PopOut/DialogIn/DialogOut;dialogOpen 接 palette/自动化/自定义/项目框;画布/Mermaid 仍 DialogBase;未测试', 'ok'),
      R('进度环、骨架占位', 'kit.Ring/SkelRows;用量加载/侧栏重扫', 'ok', 'B8')],
    states: [
      R('控件悬停 / 按下 / 选中、需重启条、提示条、账号菜单、设置搜索', 'restartForSettings;ToastSlide;settingsHits/StaggerRise;acct_popup。缺口:与 states 页矩阵未逐项对照;未测试', 'part', 'B8'),
      R('输入错误 + 抖动', 'kit.Field+ShakeInput;添加项目/设置/自定义新建', 'ok', 'B8'),
      R('删除折叠 + 6 秒撤销', 'UndoSlot+UndoBar+CollapseHide:会话/自定义/画布/设置规则/审查暂存;未测试', 'part', 'B8'),
      R('快捷键录制与冲突、连通性测试四态', 'settings_shortcuts+ConnTestRow;冲突与四态画面未全核对;未测试', 'part', 'B8'),
      R('减弱动态联动系统设置', 'applyBehavior:偏好动效 ∧ ¬SystemReduceMotion', 'ok', 'B8'),
      R('统一按下态(缩到 96% + 弹簧)', 'kit.clickPressScale 接 Btn/Ib/Tog;未测试', 'ok'),
      R('键盘聚焦环(鼠标点击不出环)', 'kit.clickFocusRing 用 FocusVisible(Btn/Ib/Tog);与设计 3px 外环仍有差;未测试', 'part', 'B8'),
      R('加载态超过 400 ms 才显示', 'kit.btnLoadingSpin(Btn/ConnTestRow 等);有单测;未测试', 'ok'),
      R('卡片悬停上浮 1px + 阴影', 'kit.Card 已定义,业务用 cuCard 等,零 kit.Card 调用;未测试', 'part', 'B8')],
    tokens: [
      R('六套色板、三种布局的圆角 / 间距 / 行高倍率、8 档间距、7 档字号、布局宽高', 'CSS 与 Go 一致', 'ok'),
      R('--dur-loop(循环档 1600 ms)', 'kit.DurLoop;Pulse/Spin 已读', 'ok', 'B8'),
      R('行高令牌 --lh-*', 'theme.LhTight~LhRead;设置/通知/对话框已用,flow/composer/浏览器等仍字面量;未测试', 'part', 'B8')],
    motion: [
      R('时长 5 档 + 5 条曲线、形态切换 440 ms、流畅 / 简化 / 无', '', 'ok'),
      R('弹层退场(pop-out)', 'PopOut+popClosing;dialogOpen+DialogOut 居中框', 'ok', 'B8'),
      R('文字流光 / 流底部渐隐遮罩', 'kit.RunText+chat_fade.go;appearance.shimmer;未测试', 'ok'),
      R('列表入场 stagger、折叠的高度插值', 'StaggerRise/CollapseHide 已接;FoldHeight 仅自动化 Cron 段;通用列表仍缺高度插值;未测试', 'part', 'B8'),
      R('states 增补的 11 种动效(抖动 / 折叠 / 抽屉 / 骨架 / 浮起 …)', 'ShakeInput~Crossfade 等均有调用点;未测试', 'ok'),
      R('已定义但没有调用点的动效', 'GrowY/Crossfade/OkPop/IndetMeter/FoldHeight 均已接线;登记行无新增缺口;未测试', 'ok')]
  };

  /** 在页面底部追加状态表(自己已有状态表的页面不会出现在 STATUS 里) */
  const row = ([name, desc, b, s]) => `<tr><td><b>${name}</b></td><td class="t2">${desc}</td><td class="t3">${b}</td><td>${S[s]}</td></tr>`;
  const table = (rows) => `<table class="ia-tbl"><thead><tr><th>范围</th><th>说明</th><th style="width:44px">批</th><th style="width:76px">状态</th></tr></thead><tbody>${rows.map(row).join('')}</tbody></table>`;
  window.STATUS_BLOCK = (id) => STATUS[id] ? `<div class="ds" style="max-width:1240px;margin:0 auto;padding:8px 24px 48px"><section class="sec"><h2>实现状态</h2><p>如实标出 Go 版现在做到哪一步;「批」是补全批次(见「缺口总表」)。做完一项就把状态改成已实现。</p>${table(STATUS[id])}</section></div>` : '';

  /* ───────── 缺口总表:扫描全部页面里的状态表自动汇总 ───────── */
  /** 自带状态表的老页面没有「批」列:按页面给默认批次 */
  const DEF_BATCH = { 'proj-ws': 'B2', 'multi-git': 'B1', menus: 'B2', 'drag-more': 'B2', 'term-spec': 'B5', 'lsp-spec': 'B5', 'layout-dock': 'B5', 'ide-md': 'B5', 'task-spec': 'B4', 'artifact-spec': 'B7', 'brand-spec': 'B6' };
  const KIND = { 已实现: 'ok', 部分: 'part', 待做: 'todo' };
  /** 取一页里所有带状态标签的表格行 */
  const collect = (id) => {
    let html = '';
    try { html = P[id] ? P[id](APP.st) : ''; } catch (e) { html = ''; }
    html += STATUS[id] ? STATUS_BLOCK(id) : '';
    if (!html) return [];
    const doc = new DOMParser().parseFromString(html, 'text/html'), out = [];
    doc.querySelectorAll('table.ia-tbl tbody tr').forEach(tr => {
      const tds = [...tr.children], last = tds[tds.length - 1], t = last && last.querySelector('.tag');
      const k = t && KIND[t.textContent.trim()];
      if (!k || tds.length < 2) return;
      let batch = tds.length >= 4 && /^B\d$/.test(tds[tds.length - 2].textContent.trim()) ? tds[tds.length - 2].textContent.trim() : '';
      if (!batch && k !== 'ok') batch = DEF_BATCH[id] || '';
      out.push({ page: id, name: tds[0].textContent.trim(), desc: tds[1].textContent.trim(), s: k, b: batch });
    });
    return out;
  };

  /** 设计稿本身待补的东西:先补稿再写代码(状态在这里维护) */
  const DESIGN_TODO = [
    R('多仓:审查按仓库分组、「逐个提交」向导', '已补画面:分组头 / 横幅 / 向导步骤 / 全部完成 / 中途失败', 'ok', 'B1'),
    R('多仓:图谱仓库下拉、分支选择器仓库 chip', '已补「切换后标签名、选中提交清空」等规则', 'ok', 'B1'),
    R('嵌套仓库 / 子模块', '总览缩进/标签/归属;scm 提示行画面见 multi-git 页', 'ok', 'B1'),
    R('搜索「仅某个文件夹」', '已补范围下拉、限定态与空态', 'ok', 'B1'),
    R('审查 / 搜索 / 问题 / 自动化的空、错误、加载状态', 'Go 空错加载与 -ed;flows 要点表;pages 有 usage-states/auto-states。缺口:审查/搜索/问题专屏画面;未测试', 'part', 'B0'),
    R('layout-dock 页把「编辑器组拆分」标成已实现', 'specs2 已改回已实现(标签条 / 菜单 / ⌘\\ 已接线)', 'ok'),
    R('term-spec 页把「⌘F 搜索」标成已实现', '全词 / 正则 / 命中刻度 / 选区预填已接', 'ok'),
    R('artifact-spec 页把浏览器能力标成待做', '实际有演示源的界面,应改成部分', 'ok'),
    R('brand-spec 的「实现状态」引用 Rust 的 app_icon.rs', 'Go 分支应改写成 mygo 的现状', 'ok'),
    R('tokens 页「每套 14 色」、motion 页「时长 6 档」与实际不符', '以 tokens.css 为准:5 档时长 + 1 个循环档', 'ok'),
    R('drag-more 页的会话菜单项(复制标题 / 复制为 Markdown)', '与 sessionMenu 一致;仅当前已打开会话可复制 Markdown', 'ok'),
    R('ide-scm / ide-diff / ide-merge / ide-git / ide-review 等页没有实现状态表', '用本文件补齐', 'ok'),
    R('逐页盘点(v3.6):ia 页没有状态表;flows「其它常见能力」清单、会话流多处画面没有状态行', '已补进本文件(ia / flow / overlays / ide-nav / ide-git / settings 等);子代理盘点的结论里,TaskNote、chat.sys、转到行、外部修改、OkPop 等已用 grep 逐个核实', 'ok', 'B0'),
    R('status.js 里服务商 4 行「Go 未实现」已过时', '代码已有 prov_*.go 与 claudecfg/providers_*.go,改为「部分:已接线,未提交,未测试」', 'ok', 'B0'),
    R('specs2 / specs3 / specs4 里几处「已实现」偏乐观', '终端标签重命名、LSP 语义高亮、task-spec 首行、侧栏会话行菜单已降级;搜索「在侧边预览」、工作区级诊断、多光标、自动化「查看会话」同步降级', 'ok', 'B0')];

  P['gaps'] = () => {
    const ids = APP.pages.flatMap(([, items]) => items.map(i => i[0])).filter(i => i !== 'gaps');
    const title = Object.fromEntries(APP.pages.flatMap(([, items]) => items.map(i => [i[0], i[2]])));
    const all = ids.flatMap(collect);
    const open = all.filter(x => x.s !== 'ok'), cnt = (s, l = all) => l.filter(x => x.s === s).length;
    const pct = all.length ? Math.round(cnt('ok') * 100 / all.length) : 0;
    const byPage = ids.map(i => [i, all.filter(x => x.page === i)]).filter(([, l]) => l.length);
    const bar = (l) => { const t = l.length || 1; return `<div style="display:flex;height:6px;border-radius:3px;overflow:hidden;background:var(--fill-2);width:160px"><i style="width:${cnt('ok', l) * 100 / t}%;background:var(--ok)"></i><i style="width:${cnt('part', l) * 100 / t}%;background:var(--warn)"></i></div>`; };
    const prog = byPage.map(([i, l]) => `<tr><td><b>${title[i]}</b></td><td>${bar(l)}</td><td class="t2">已实现 ${cnt('ok', l)} · 部分 ${cnt('part', l)} · 待做 ${cnt('todo', l)}</td></tr>`).join('');
    const batchRows = BATCHES.map(([b, name, desc]) => { const l = open.filter(x => x.b === b); const n = l.length + (b === 'B0' ? DESIGN_TODO.filter(r => r[3] !== 'ok' && r[2] === 'B0').length : 0); return [`${b} · ${name}`, desc, n ? `${n} 项` : '—', n ? 'todo' : 'ok']; });
    const listRows = open.sort((a, b) => (a.b || 'Z').localeCompare(b.b || 'Z')).map(x => `<tr><td class="t3">${x.b || '—'}</td><td class="t3">${title[x.page] || x.page}</td><td><b>${x.name}</b></td><td class="t2">${x.desc}</td><td>${S[x.s]}</td></tr>`).join('');
    return `<div class="ds" style="max-width:1240px">${head('缺口总表 · 设计稿 ↔ Go 实现', `这一页由脚本扫描每个页面里的「实现状态」表自动汇总,不手写:哪页改了状态,这里立刻同步。<br>当前 ${all.length} 个功能点,已实现 ${cnt('ok')}(${pct}%),部分 ${cnt('part')},待做 ${cnt('todo')}。工作流:先改设计稿(画面 + 状态)→ 再改代码 → 做完把那一行改成「已实现」。`)}
    ${sec('补全批次', '按这个顺序一批一批做;一批的所有行都变成「已实现」,这一批就关闭。', `<table class="ia-tbl"><thead><tr><th style="width:220px">批</th><th>内容</th><th style="width:70px">未完成</th><th style="width:76px">状态</th></tr></thead><tbody>${batchRows.map(([a, b, c, s]) => `<tr><td><b>${a}</b></td><td class="t2">${b}</td><td class="t3">${c}</td><td>${S[s]}</td></tr>`).join('')}</tbody></table>`)}
    ${sec('设计稿自己要先补的', '先补稿再动代码;补完把状态改成已实现。', table(DESIGN_TODO))}
    ${sec('全部未完成项(' + open.length + ')', '按批次排序;「—」表示还没有分批。', `<table class="ia-tbl"><thead><tr><th style="width:44px">批</th><th style="width:170px">页面</th><th style="width:300px">范围</th><th>说明</th><th style="width:76px">状态</th></tr></thead><tbody>${listRows}</tbody></table>`)}
    ${sec('各页进度', '', `<table class="ia-tbl"><tbody>${prog}</tbody></table>`)}</div>`;
  };
})();
