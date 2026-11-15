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
      R('窄档起手卡单列', '', 'ok'),
      R('空会话下的拖拽引用落点', '欢迎页同样可以拖文件进输入框', 'ok')],
    palette: [
      R('⌘K 打开 / Esc · 遮罩关闭、欢迎列表(命令 / 会话 / 最近文件)与模糊查询', '', 'ok'),
      R('主题二级列表:上下即时预览,Esc 还原', '', 'ok'),
      R('右侧预览栏(命令说明 / 文件前 16 行)', '窄于 820px 隐藏预览栏;极窄时面板宽度随窗口收缩', 'ok', 'B6'),
      R('欢迎列表里的高频命令', 'palette.go:权限循环(⇧Tab 同源)、压缩 / 分叉 / 新会话、计划 / 审查 / 任务虚拟标签、14 类设置直达、真实演示重启;无会话或无偿计划时置灰并 toast;未测试', 'part', 'B6'),
      R('⌘P 里输入 :行号 转到行', '文件项只会 openFile,不解析 :42;与编辑器「转到行」是同一缺口(见编辑器能力) ◆ 第一波:palette.go 解析 :行号 → gotoLineInCurrent(超界夹到末行);未测试', 'part', 'B5')],
    settings: [
      R('4 组 14 类导航、改动圆点与悬停还原、外观(主题 / 方案 / 强调色 / 密度 / 圆角 / 动效)即时生效', '', 'ok'),
      R('设置内搜索', '命中后跳进分类并滚动高亮目标行(约 1.4s)', 'ok', 'B6'),
      R('需重启条', '「立即重启」自启动进程并退出(开发版 .app 路径)', 'ok', 'B6'),
      R('外观 → 时间线字号', 'FlowFontSize / FlowCapFont / FlowMonoSize 统一缩放;助手 Markdown 走 mdOptions;工具卡 ToolDetailLevel 控制默认展开;任务通知 / 子 Agent / 用户卡 / 输入框已接;未测试', 'part', 'B6'),
      R('外观 → 应用图标 / 菜单栏图标两组', '六样式×四外观 assets、Dock 弹跳/进度/角标、托盘动画/菜单/数字/VoiceOver;访达换装与 .icon 打包未接', 'part', 'B6'),
      R('快捷键:点键录制、冲突当场提示', '全局 / 编辑器 / 终端快捷键读偏好', 'ok', 'B6'),
      R('其余偏好', '启动形态、忽略规则、编辑器 / 终端 / 浏览器等大半已接线;还没接线的逐类列在下面四行(设置行存在,但改了不影响任何行为)', 'part', 'B6'),
      R('偏好未接线 · 模型 / 对话', '模型:默认模型 / 默认思考强度(键 models.* 与 claude.model / claude.effort 不是同一个键)。对话:回合结束后折叠过程(chat.fold)、默认权限模式、追加系统提示词(chat.sys 只写偏好,没有下发给 claude)。◆ 已接未测:general.*、general.paste(@ 引用+发送展开)、models.thinking/compact/compactAt、perm.sandbox 仍只存偏好(无 Claude 外部键)', 'part', 'B6'),
      R('偏好未接线 · 权限 / 运行环境', '权限:默认模式、禁止绕过、Bash 沙盒、额外可访问目录,只存偏好,没写回 claude 设置。运行环境:可执行文件(runtime.bin 没传给 claude.Config.ClaudePath,仍走 PATH / MINI_CLAUDE_PATH)、代理模式 / 地址 / 绕过、超时 ◆ 第一波:settings_wire.go 接了 runtime.bin / 代理 / 超时,权限写回 claude 设置;未测试,Bash 沙盒与额外目录是否真被 claude 采用待核', 'part', 'B6'),
      R('偏好未接线 · 编辑器 / 终端 / SCM / 浏览器', '编辑器:语法高亮开关、追溯栏(editor.blameGutter,代码只实现行内追溯)。终端:登录 shell、在项目目录启动(term.Start 没读)。SCM:Diff 大小上限、自动 fetch、Worktree 存放位置。浏览器:默认阅读模式 ◆ 第一波:term.Start 经 TermStartOpts 读登录 shell / 目录 / shell / 回看,Diff 大小上限在 openDiffKind 弹确认(仍然显示),自动 fetch 节拍常开按偏好执行;追溯栏见编辑器行;未测试', 'part', 'B6'),
      R('偏好未接线 · 隐私 / 高级 / 关于', '隐私:崩溃报告、使用统计、日志级别(diag.SetLevel 存在,设置页没调)。高级:前台命令实时输出开关、主线程看门狗开关。关于:自动下载更新、更新通道。外观:「跟随系统深浅色」只驱动 Dock 图标,不切主题 ◆ 第一波:日志级别经 ApplyDiagPrefs、看门狗开关、前台实时输出开关已接;「会话流一次测完行高」只存偏好并注明 Go 版不生效;未测试', 'part', 'B6'),
      R('设计稿有、设置页没有的行', '外观:等待动画、滚动条。模型:输出风格。运行环境:claude「检测」按钮与版本、环境变量显示 / 遮盖、连通性第三主机。隐私:诊断包「导出」、claude doctor 与结果折叠、「崩溃与卡顿」近 30 天块。高级:settings.json 导出 / 导入、会话流一次测完行高。关于:开源致谢、在 GitHub 上查看。SCM:Worktree 列表行 ◆ 第一波:settings_runtime_ui / settings_privacy_extra / settings_advanced_data 补了检测按钮、环境变量遮盖、诊断包导出、settings.json 导出导入等;外观「等待动画 / 滚动条」是否齐全待阶段三对照;未测试', 'part', 'B6'),
      R('设置里「检查器」措辞与设计稿不一致', '设计稿 v3.1 没有「检查器」,对应项是「开标签时自动展开右侧工作区」;Go 的 general.insp、快捷键列表「显示 / 隐藏检查器」、方案卡示意图仍沿用旧名(以设计稿为准,改 Go 文案) ◆ general.insp 的文案已改为「开标签时自动展开右侧工作区」;快捷键列表与方案卡示意图是否还有旧名待阶段三核对', 'part', 'B6'),
      R('规则 / 环境变量 / 语言服务器 / 项目信任', '规则 allow/deny/ask 与环境变量 claudecfg 真实读写(空态 / 读失败);语言服务器 settings_lsp 后台检测 lsp.Specs + 单服务器偏好 lsp.server.*、复制/终端安装;项目信任合并 ~/.claude.json 与注册表 hasTrustDialogAccepted 原子写、MINI_SET_TRUST=empty|fail|writefile;均未测试', 'part', 'B6'),
      R('服务商自定义与预设', 'prov_settings.go + claudecfg.Presets:6 预设 + 自定义、Bearer / x-api-key、模型列表地址、连通性测试、providers.json + 0600 provider-keys.json 已接线,单测 + 界面测试通过;真实 DeepSeek 密钥联调因密钥无效(401)未能验证', 'part', 'B6'),
      R('获取模型列表与勾选', 'claudecfg.FetchModels + provInventory:骨架 / 空 / 失败 / 不支持 / 缺密钥、手动添加、过滤 / 全选 / 清空已接线,未测试;演示只覆盖部分场景(设计稿原型规则 example.com / legacy 未全部对应)', 'part', 'B6'),
      R('模型映射下拉', 'claudecfg.Slots / MappingStatus / Env:五个槽位与生效状态已接线;回合运行中 pvPushEnv 跳过注入(与设计「运行中不受影响」一致)。未测试,需真 claude 验证环境变量确实生效', 'part', 'B6'),
      R('会话内按服务商选模型', 'prov_picker.go cfgModels + popups.go cfgPopup + 状态栏分组、「管理服务商…」入口已接线;切换 toast 缺「新会话 / 下一条消息起生效」副句;官方模型子选型仍走 a.st.Models。用真实 claude + 本机桩服务验证过「映射 → 环境变量 → 请求」与中途换服务商(MINI_REAL_CLAUDE=1)', 'part', 'B6'),
      R('模型品牌 logo', 'scripts/gen-logos.py 从 Lobe Icons(MIT)单色版生成 kit/logos_gen.go 与 js/logos.js(20 个品牌:DeepSeek / 智谱 / Kimi / Qwen / MiniMax / Claude / OpenAI / Gemini 等;匹配规则与品牌色同源):服务商行、模型清单、映射下拉、会话里的选择器、输入框胶囊都带 logo,认不出的模型用统一占位芯片,黑白品牌跟随文字色。缺口:状态栏模型名与上下文面板无 logo;品牌图标是单色,没有渐变版', 'part', 'B6'),
      R('窄于 900 时左侧分类收成下拉', '左侧栏隐藏,顶部分类下拉', 'ok', 'B6')],
    forms: [
      R('单窗口、无融合 / 独立 IDE 窗口;两种形态共享会话 / 编辑器 / 终端', '', 'ok'),
      R('⇧⌘N 新窗口(再起一个进程)', '--new-window / 菜单 / 命令面板;从属进程不写偏好,窗口错开 32px', 'ok', 'B6'),
      R('启动形态(Agent / IDE / 上次)', 'general.start + layout.lastMode 已接 main 启动', 'ok', 'B6'),
      R('形态切换的内容编排(甘特图)', 'morph_content.go:工作区 / 对话流 / 导航分轨淡入淡出(90ms 延迟 / 120ms 淡出 / 60ms 侧栏延迟);标题栏滑块未单独分轨;motion 关闭时直达;未测试', 'part', 'B6')],
    ia: [
      R('信息架构四条原则、4 组 14 类设置树、迁移对照表', '以 settings 页的实现为准,设置搜索兜底(settingsHits / jumpSettingRow)已接', 'ok', 'B0'),
      R('「服务商状态与交互清单」(8 行)', '对应 settings 页的服务商四行,代码已接线、未提交、未测试', 'part', 'B6'),
      R('入口矩阵 ENTRY(同一内容别处只是跳转)', '工作区虚拟标签入口都走 openVirtual', 'ok', 'B0')],
    flows: [
      R('没有「检查器」:详情只落在工作区标签,每个内容一个家', '', 'ok'),
      R('打开规则矩阵:审查 / 计划 / 子 Agent / 差异 / 终端只读 / SCM 预览标签', '', 'ok'),
      R('预览斜体、双击固定、虚拟标签全窗口单份;导航历史 ← →', '', 'ok')],
    resp: [
      R('对话栏按自身宽度三档(≥640 / 420–639 / <420)', '', 'ok'),
      R('工作区按自身宽度三档(≥600 / 520–599 / <520:迷你地图、统一 diff、面包屑)', '', 'ok'),
      R('窗口变窄自动收起 / 恢复两侧栏', '固定断点 1100 / 900 / 720 + 24px 回差', 'ok', 'B6'),
      R('Agent 形态:工作区在窄窗变成滑入浮层;<720 导航栏变抽屉', 'overlays.go 浮层与抽屉;动效待阶段三对照', 'part', 'B6'),
      R('IDE 形态:窄窗对话栏收成图标条 + 浮层', '图标条 + 浮层已做;滑入动效待对照', 'part', 'B6'),
      R('整页功能的窄窗规则(自动化 / 画布 / 浏览器 / 自定义 / 用量,<1000 / <900)', '五页已按 resp 表核对:自动化周历纵向 + 编辑全屏弹层、画布 / 浏览器右栏落底、自定义顶部分段 + 列表详情两级、用量单列堆叠;Go 用 winW 断点(非容器宽),未测试', 'part', 'B6')],
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
      R('正常运行时注入任务', 'translate task_* + 任务坞;未测试', 'part', 'B4')],
    flow: [
      R('步骤行(进行 / 错误 / 拒绝)、探索分组、工具展开 diff / 终端、待办、权限 / 提问 / 计划卡、提示三级', '', 'ok'),
      R('长输出限高折叠(12 行 → 196px,展开 480px)', '工具 / 终端 / 文本与 Markdown 代码块', 'ok', 'B4'),
      R('折叠工具分组的「最新一步」滚动行', '探索组折叠且运行中 22px 滑入行', 'ok', 'B4'),
      R('改动卡的「撤销」按钮', '逐文件 revertHunk', 'ok', 'B4'),
      R('回合操作条:复制 / 重试 / 分叉', '复制 / 重试 / 分叉(真实 fork: 会话);演示 fork 会话', 'ok', 'B4'),
      R('用户消息卡:长消息折叠 / 编辑', '折叠+Esc/点外取消;真实 resume_session_at 控制请求,无 uuid 置灰+分叉提示', 'part', 'B4'),
      R('工作区起始页(本轮 / 终端 / 打开三组列表;终端行列出已开终端 + 新建)', 'ws_empty.go 三组列表', 'ok', 'B4'),
      R('发送 / 停止按钮为圆形', '设计稿与 Go 版同步', 'ok', 'B4'),
      R('会话样式(跟随布局 / 对话气泡 / 通栏=与正文同宽的圆角卡片;设置 → 外观 → Agent 会话)', 'appearance.flowStyle 三档', 'ok', 'B4'),
      R('用户消息吸顶(回合内 sticky,下一轮顶走上一张,最多一张;点击就地编辑 / 取消 / 发送;设置 → 对话与提示词,默认开)', 'MyGo Sticky + chat.stickyUser;吸顶卡背景必须不透明(Fill1 之类半透明填充先叠在 BG 上,否则下面的计划卡文字透出来叠在一起),固定时抬一层阴影 + 1px 描边 + 下沿 14px 渐隐(按上一帧提问卡顶边是否低于回合顶边判断);未测试', 'part', 'B4'),
      R('侧栏工作区 / 项目右键与「编辑项目」', '工作区展开只列会话;右键 = 编辑工作区… / 编辑项目…(新对话 · 编辑 · 访达 / 复制 · 移除);全应用右键菜单改为窗口内自绘(SetMenuPresentation);其余菜单的图标与快捷键提示已经 menu_item.go 的 menuItem 统一补齐;未测试', 'ok', 'B2'),
      R('右键菜单图标与快捷键提示统一(menuItem 单点维护)', 'menu_item.go:文字 → 设计稿图标表,「文字  快捷键」拆成右侧提示;图标名取自 kit.iconPaths;未测试', 'ok', 'B2'),
      R('会话流步骤行 / 差异卡头 / 命令卡头右键菜单', 'menu_extra.go stepMenu + flow.go cardM:复制路径 / 命令 / 输出 / 网址、访达、查看差异 / 审查标签、终端面板;未测试', 'ok', 'B2'),
      R('多根顶层文件夹右键:在源代码管理中查看 / 设为主文件夹 / 从工作区移出…', 'menu_extra.go;设为主 / 移出走 wsMutate(保存后重新进入工作区),主文件夹与只剩 2 个文件夹时「移出」置灰写明原因;未测试', 'ok', 'B2'),
      R('Markdown 标签 ··· 附加项(切换预览 / 源码、只读预览、格式化文档)', '「只读预览」只反映含表格 / HTML / 脚注时的自动只读状态(置灰带勾),没有手动开关;未测试', 'part', 'B2'),
      R('输入框 MCP chip 与弹层', 'mcp_status+无会话 mcp list;工具 deny+RefreshMcp', 'part', 'B4'),
      R('会话流 · 后台任务完成通知行(TaskNote)', 'core.TaskNote 与归约(session.go)已有,但会话流 flowTail 没有渲染分支:后台命令 / 监听结束后,流里看不到「已完成 · 退出码 0」那一行(设计稿 overlays「已结束 · 通知行」) ◆ 第一波:flow_note.go 渲染 TaskNote;未测试', 'part', 'B4'),
      R('会话流 · 并行子 Agent 汇总行「并行 N 个」', '设计稿 F.step(agent, 并行 2 个);Go 只在演示场景里有,flow*.go 没有汇总步骤行,连续子 Agent 逐个排开 ◆ 第一波:flow_parallel.go 汇总行;未测试', 'part', 'B4'),
      R('会话流 · 回合折叠「已处理 1 分 13 秒」', '设计稿与设置项 chat.fold 都有;应用侧没读这个偏好,回合结束后过程不折叠 ◆ 第一波:flow_turn_fold.go 读 chat.fold,失败 / 运行中 / 待处理的回合保持展开;未测试', 'part', 'B4'),
      R('用户消息 · 悬停动作条(回退文件 / 重试 / 复制)', '设计稿 sticky / overlays 有;flow_user.go 只有折叠 / 吸顶 / 就地编辑,没有悬停动作条 ◆ 第一波:flow_user_actions.go;未测试', 'part', 'B4'),
      R('「回退文件」预览对话框(用户消息入口)', '设计稿 overlays「回退到这条消息之前」;core/rewind.go 与 CmdRewindFiles 内核已有,只接在审查页页脚,用户消息上没有入口 ◆ 第一波:flow_rewind.go 用户消息入口,接 CmdRewindFiles 预览;未测试', 'part', 'B4'),
      R('用户消息里的图片引用 / 缩略图 / 全屏', '设计稿 F.ref(image);core.User.Images 有数据,界面不画 ◆ 第一波:flow_user_img.go(缩略图 + 展开,图片经 img_cache.go 后台解码);未测试', 'part', 'B4'),
      R('输入框「+」添加附件与斜杠命令', '「+」按钮没有点击处理(composer.go),输入框没有 / 命令补全与发送展开;拖入图片与 @ 引用已有 ◆ 第一波:composer_attach.go(+ 菜单 / 附件 chip)与 composer_slash.go(/ 命令补全);未测试,发送展开 / 命令来源待核', 'part', 'B4'),
      R('审批多张排队「1 / 3」与处理后输入框淡入、恢复草稿', 'pendingDock 只取第一项,没有队列指示;处理完没有 120 ms 输入框淡入与草稿恢复(设计稿 task-spec) ◆ 第一波:dock_pending.go(1 / N 指示 + 处理后淡入与草稿恢复);未测试', 'part', 'B4'),
      R('排队卡「停止并发送」', '设计稿 F.queue 三个动作;Go queueCard 只有引导 / 删除 ◆ 第一波:composer_queue.go;未测试', 'part', 'B4'),
      R('排队消息带图片', '排队项保留草稿图片语法;卡上缩略图 chip;退回编辑还原;发送经 submitPrompt 带 Images;演示 b4-queue-img。未测试', 'part', 'B4'),
      R('提示卡的动作(重试 / 压缩)与权限「你拒绝了…」专用行', '错误/警告卡 noticeActions 有重试·压缩;已拒绝权限:工具行 deny+lock 与 flow_note permissionDenyNote(shield 灰字)。不可重试。未测试', 'part', 'B4'),
      R('会话流内 Mermaid 图表卡与全屏查看', '◆ flow_mermaid.go 图表卡(复制/全屏/查看源码)+internal/mermaid 排版+后台缓存;mermaid_viewer.go 全屏适应/±/平移/Esc。画布页仍为简版方框。未测;滚轮缩放未接;复杂 flow 样式类/er/gantt/class 等未做', 'part', 'B7'),
      R('task-spec 页实现状态表', 'specs2 待办 / 尾读 / 通知 / 分叉四行', 'ok', 'B0')],
    overlays: [
      R('命令面板、历史会话、分支选择器、账号菜单、运行配置弹层', '', 'ok'),
      R('IDE 通知中心(铃铛 / 历史 / 右下角叠放)', '铃铛已接;.app 内 UN 点击 sess 回会话;go run 降级', 'part', 'B4'),
      R('任务坞停滞 / 已结束 / 监听 / 工作流胶囊', '六类胶囊+translate 事件;未测试', 'part', 'B4'),
      R('上下文弹层(输入框圆环 / 状态栏;分类明细 / 阈值刻度 / 账户限额 / 刷新 / 压缩;常态 / 接近阈值 / 压缩中 / 加载 / 无限额 / 失败)', 'get_context_usage+失败重试;get_usage 限额', 'part', 'B4'),
      R('更新进度胶囊四态', '下载/校验/解包/替换脚本;开发版 dev.sh;正式包退出后 ditto', 'part', 'B4'),
      R('确认框(模式切换 / 未保存 / 删除)', '通用确认框已用于会话删除 / 移除项目 / 切换模式', 'ok', 'B2'),
      R('账号菜单的真实身份与用量', '真实:claudecfg.ReadOAuthProfile + UsageScanner 本月费用 + get_usage 五小时%;演示 demo.AcctMenuDemo;无数据/加载中状态。未测试', 'part', 'B6'),
      R('上下文弹层的「查询失败」独立状态', '失败时有重试条,但 ctxState() 没有独立的 err 状态,与设计稿六态标签没完全对齐', 'part', 'B4')],
    'ide-editor': [
      R('迷你地图(语法色块、滑块拖动)、Git / 搜索 / 同词 / 诊断 / 光标刻度、概览标尺、同词高亮、窄档藏地图', '', 'ok'),
      R('行号槽的 Git 改动竖条', 'DiffBase 接线;无 HEAD 或非 Git 文件不画竖条', 'ok', 'B5'),
      R('迷你地图滑块 / 宽度 / Git 刻度三个设置', '三个偏好已接到地图绘制', 'ok', 'B5'),
      R('CodeLens(引用数 / 测试 / 问 Claude)', '库无独立 CodeLens 行,用符号行上方叠层;测试数由本地符号索引推断(Go/Rust/Py/TS);窄窗与粘性滚动错位时 CodeLens 下移', 'ok', 'B5'),
      R('迷你地图悬停显示源码', '悬停该行左侧浮层显示源码摘要', 'ok', 'B5'),
      R('追溯栏(行号左侧逐行显示提交,同提交只在首行显示作者)', '设计稿 flows / ide-git 有,设置项 editor.blameGutter 也有;代码只实现了光标行末尾的行内追溯,没有追溯栏渲染 ◆ 第一波:ide_blame_ui.go 追溯栏;未测试', 'part', 'B5')],
    'ide-nav': [
      R('粘性滚动、查找替换、悬停、预览定义、跳转定义 / 引用 / 类型 / 实现、补全、重命名、格式化、跳转历史', '', 'ok'),
      R('代码操作 ⌘.', 'LSP codeAction + 本地兜底;无服务器时仍为本地菜单', 'ok', 'B5'),
      R('选中工具条 / ⌘I 行内编辑', '真实模式 claude -p Haiku 生成替换预览,Esc/取消可中断;演示仍走占位', 'ok', 'B5'),
      R('行内审查(悬停接受 / 撤销)', '改动行悬停接受写入审查状态 ok、撤销走 revertHunk', 'ok', 'B5'),
      R('括号配对竖线 / 着色', '竖线接 NoIndentGuides;着色用括号深度染色(MyGo 无 tree-sitter 应用层 API,已取舍)', 'ok', 'B5'),
      R('多光标', '依赖 MyGo 编辑器库自带的键位(⌥点击 / ⌘D / ⇧⌘L);应用层没有接线也没有测试,AGENTS.md 曾把「⌥点击多光标」记为未做,阶段三实测后再标已实现', 'part', 'B5'),
      R('转到行(⌃G)', '设计稿清单有;Go 里没有 GotoLine / ⌃G 绑定,⌘P 也不解析 :行号 ◆ 第一波:ide_goto.go + 快捷键 gotoLine;未测试', 'part', 'B5'),
      R('块注释(⌥⇧A)', '设计稿清单有;只实现了行注释 ⌘/ ◆ 第一波:ide_overlay.go 绑定 blockComment,语言无块注释时提示;未测试', 'part', 'B5'),
      R('自动保存之外的「外部修改」处理', '设计稿:磁盘变了——无本地修改则自动重载,有修改则提示「重新载入 / 保留」。Go 里没有文件监视(没有 fsnotify,也没有修改时间轮询),编辑器不知道磁盘上的文件被改过;自动保存本身已有 ◆ 第一波:ide_extmod.go 修改时间轮询,无本地修改自动重载,有修改提示「重新载入 / 保留」;未测试', 'part', 'B5'),
      R('面包屑每一级的同级下拉', '设计稿面包屑点击 = 同级目录 / 符号下拉;Go 只有路径分段,符号段点开是快速选择', 'part', 'B5'),
      R('编辑器组向右 / 向下拆分', '两组标签 + 共享缓冲区;标签可跨组拖动', 'ok', 'B5'),
      R('Claude 行内补全(✦)', '真实 Haiku claude -p + SetSuggestion;演示占位,Tab 接受', 'ok', 'B5')],
    'ide-search': [
      R('⇧⌘F、选区预填、大小写 / 全词 / 正则、包含排除、文件折叠、虚拟列表、点击预览、替换预览、悬停忽略、全部替换', '', 'ok'),
      R('单条替换(⏎)', '替换行 ⏎ 替换当前选中匹配', 'ok', 'B5'),
      R('多根:「仅某个文件夹」范围筛选', '范围 chip + 下拉 + ✕ 清除,文件夹右键「仅在此文件夹搜索」', 'ok', 'B1'),
      R('搜索结果右键菜单', '匹配行 / 文件行已做;「在侧边预览」= searchPreviewSide(侧边组开预览并定位命中处,没有拆分时先向右拆分);未测试', 'ok', 'B2'),
      R('空 / 错误 / 搜索中 / 结果截断四种状态的画面', '空查询 / 错误 / 搜索中 / 截断摘要已补', 'ok', 'B5')],
    'ide-problems': [
      R('状态栏错误 / 警告计数 → 问题面板、按文件分组、点击跳转、未安装语言服务器的安装卡', '', 'ok'),
      R('「重新检查」按钮与「全部 / 仅错误」筛选', '刷新按钮触发工作区重检;演示模式同步筛选', 'ok', 'B5'),
      R('工作区级诊断', '开关 + 上限 + 取消代次;用 didOpen 批量扫盘(gopls/pyright/tsserver/rust-analyzer)代替,没有 LSP workspace/diagnostic pull,诊断是「扫过的文件」而不是服务器的全量结果', 'part', 'B5')],
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
      R('图谱菜单「与 HEAD 比较」与「选中两个提交比较」', '设计稿 ide-git 图谱右键菜单有这两项;commitMenu 只有「与工作区比较」 ◆ 第一波:scm_compare.go / scm_graph_picks.go;未测试', 'part', 'B3'),
      R('推送被拒:「拉取并变基 / 拉取并合并」分按钮', '设计稿给两个下一步按钮;Go 只有「先拉取」(内部 pull --ff-only)与强制推送,遇到需要合并 / 变基的分叉走不下去 ◆ 第一波:git_push_actions.go 分「拉取并变基 / 拉取并合并」两个按钮(vcs/sync.go);未测试', 'part', 'B3'),
      R('审查 → 一键暂存已接受的改动', '设计稿审查页可把已接受的块直接写入暂存区;Go 只有「在源代码管理里暂存」跳转到 SCM ◆ 第一波:review_stage.go + 撤销条;未测试', 'part', 'B3'),
      R('行内追溯的悬停提交卡', '设计稿悬停出提交卡(哈希 / 作者 / 说明 / 操作);Go 只有行尾淡色追溯文字 ◆ 第一波:ide_blame_ui.go 悬停卡;未测试', 'part', 'B3'),
      R('储藏条目展开看各文件差异', '设计稿储藏可点开看文件;Go 的储藏区只有应用 / 弹出 / 删除 ◆ 第一波:scm_stash_expand.go + vcs/stash_diff.go;未测试', 'part', 'B3')],
    automation: [
      R('周历(翻周 / 今天)、卡片列表筛选与启停、右侧编辑抽屉、新建 / 删除、真实落盘', '', 'ok'),
      R('计划 Cron', '五段 cron 解析、FiresOn/Due 到点触发、下次运行时间', 'ok', 'B7'),
      R('项目选择 / 权限选择', '新建模态与编辑抽屉:项目分段 + 权限分段', 'ok', 'B7'),
      R('立即运行', '真实 claude -p 流式+取消;演示 RunAutomationDemo 流式+取消+失败样例', 'ok', 'B7'),
      R('最近运行可点进会话', '定时与立即运行共用 runAutomationCore,成功/失败/取消回写 session_id 并落盘;无 id 时置灰并说明原因,真实模式打开前可触发索引重扫(未测试)', 'part', 'B7'),
      R('新建模态与失败 / 运行中状态', '设计稿 behave.js au-new/au-run;Go 模态+运行浮层', 'ok', 'B7')],
    canvas: [
      R('块编辑 / 预览、大纲定位、版本提交与回退、导出 Markdown', '', 'ok'),
      R('落盘', '~/.minicode/canvases/*.json 原子写;损坏改名留底;多画布列表', 'ok', 'B7'),
      R('让 Claude 修改', '块引用+ApplyClaudeReply;在画布页回合结束自动写回并落盘', 'ok', 'B7'),
      R('图表全屏查看 / 源码', '全屏:适应/100%/±/平移/Esc/坐标;复杂图仍简版方框布局', 'part', 'B7'),
      R('窄窗右栏收成底栏', '宽<900 时大纲/版本落底部', 'ok', 'B7'),
      R('左侧画布列表(多画布)', 'Library 多画布:切换/新建/重命名/删除+撤销;演示多套 demoPgCanvas', 'ok', 'B7')],
    browser: [
      R('后退 / 前进 / 刷新 / 收藏、摘要 · 大纲 · 链接、齿轮直达设置', '', 'ok'),
      R('真实网页抓取与阅读模式正文', 'net/http 抽取正文;超时/大小上限;演示源并存', 'ok', 'B7'),
      R('「与此页面对话」的页面引用', '正文框选→@浏览器:url#段+填输入框;日志 RedactURL 脱敏', 'ok', 'B7'),
      R('系统浏览器打开 / 隐私设置', '地址栏外链;设置 browser.fetch 可关抓取', 'ok', 'B7')],
    customize: [
      R('七类左栏、筛选与作用域、开关 / 新建 / 删除、详情表单', '界面与校验完整;演示内存+演示多套', 'ok'),
      R('读写真实的 ~/.claude 与项目 .claude', 'MCP 编辑配置 SaveMCP+env;命令/Agent/规则/钩子写盘;Skill 重命名目录;市场 list JSON/文本', 'part', 'B7'),
      R('插件市场安装进度', '四态进度条;install/uninstall CLI', 'part', 'B7'),
      R('MCP 授权 / 重连 / 工具开关接真实状态', 'mcp_status+mcp list 兜底+init 初值;login 仍手工', 'part', 'B7'),
      R('命令 / 子 Agent 试运行、钩子试运行', 'claude -p 与 shell 试运行已接;钩子带确认与超时', 'part', 'B7'),
      R('Skills / 规则「在编辑器里打开」', '已接 openFile;用户层路径依赖绝对路径', 'part', 'B7')],
    usage: [
      R('7 天 / 30 天 / 本月、汇总卡、柱图悬停、按模型 / 项目 / 贵会话、导出 CSV、空数据加载态(真实扫描)', '', 'ok'),
      R('账户限额与刷新', 'get_usage+缓存;无会话/失败文案;限额失败「重试」', 'part', 'B7'),
      R('贵会话点击跳转到会话', 'TopSessions 带 SessionID 时 OpenSession', 'ok', 'B7')],
    comps: [
      R('按钮 / 图标按钮 / 胶囊 / 标签 / 计数 / 分段 / 开关 / 标签条 / 列表行 / Spin / Meter / Chk / Dot / Kbd', '', 'ok'),
      R('按钮 disabled / loading 态', 'kit.Btn Disabled/Loading;提交/安装/重试等接线(仅按钮;图标按钮 / 开关 / 胶囊的禁用与加载态见 states 页)', 'ok', 'B8'),
      R('图标按钮 / 开关的 disabled 与 error 态', '设计稿 states 页矩阵有;kit.Ib 没有 disabled / loading / error 选项,kit.Tog 没有 disabled ◆ 第一波:kit.Btn / Ib / Tog 加 Disabled / Loading / Error(kit_interact.go);未测试', 'part', 'B8'),
      R('输入框统一封装与错误描边', 'kit.Field:设置单行、项目/工作区/自动化/自定义新建对话框(提交框除外)', 'ok', 'B8'),
      R('菜单 / 弹层壳', 'kit.PopIn/PopOut/DialogIn/DialogOut+dialogOpen;终端搜索/安装卡/通知 toast', 'part', 'B8'),
      R('进度环、骨架占位', 'kit.Ring/SkelRows;用量加载/侧栏重扫', 'ok', 'B8')],
    states: [
      R('控件悬停 / 按下 / 选中、需重启条、提示条、账号菜单、设置搜索', 'kit 控件+重启条+ToastSlide+设置搜索/stagger;账号菜单 pop', 'part', 'B8'),
      R('输入错误 + 抖动', 'kit.Field+ShakeInput;添加项目/设置/自定义新建', 'ok', 'B8'),
      R('删除折叠 + 6 秒撤销', 'UndoBar+折叠行;会话/自定义/设置规则', 'part', 'B8'),
      R('快捷键录制与冲突、连通性测试四态', '快捷键设置页;连通性 kit.ConnTestRow(服务商+诊断)', 'part', 'B8'),
      R('减弱动态联动系统设置', 'applyBehavior:偏好动效 ∧ ¬SystemReduceMotion', 'ok', 'B8'),
      R('统一按下态(缩到 96% + 弹簧)', '设计稿 states 规则;kit 里没有全局按下缩放 ◆ 第一波:kit.clickPressScale 已接 Btn / Ib / Tog;未测试', 'part', 'B8'),
      R('键盘聚焦环(鼠标点击不出环)', '设计稿 :focus-visible 3px 环;MyGo 控件层区分不了鼠标 / 键盘聚焦,只有输入框 FocusWithin 描边 ◆ 第一波:kit.clickFocusRing(MyGo 区分不了鼠标 / 键盘聚焦,按点击后不出环近似);未测试', 'part', 'B8'),
      R('加载态超过 400 ms 才显示', '设计稿 states 规则;kit.Btn.Loading 一设置就立即转圈 ◆ 第一波:kit.btnLoadingSpin 延迟 400 ms 再转圈;未测试', 'part', 'B8'),
      R('卡片悬停上浮 1px + 阴影', '设计稿规则;没有统一的 kit 卡片组件 ◆ 第一波:kit.Card(kit_card.go),业务页尚未全部改用;未测试', 'part', 'B8')],
    tokens: [
      R('六套色板、三种布局的圆角 / 间距 / 行高倍率、8 档间距、7 档字号、布局宽高', 'CSS 与 Go 一致', 'ok'),
      R('--dur-loop(循环档 1600 ms)', 'kit.DurLoop;Pulse/Spin 已读', 'ok', 'B8'),
      R('行高令牌 --lh-*', 'theme.LhTight~LhRead 七档;设置/通知等已替换,长文仍有字面量', 'part', 'B8')],
    motion: [
      R('时长 5 档 + 5 条曲线、形态切换 440 ms、流畅 / 简化 / 无', '', 'ok'),
      R('弹层退场(pop-out)', 'PopOut+popClosing;dialogOpen+DialogOut 居中框', 'ok', 'B8'),
      R('文字流光 / 流底部渐隐遮罩', 'TextShimmer 透明度扫光+Pulse;流底 MaskLinearGradient', 'part', 'B8'),
      R('列表入场 stagger、折叠的高度插值', '设置/命令面板 StaggerRise;删除 CollapseHide;FoldHeight 预留', 'part', 'B8'),
      R('states 增补的 11 种动效(抖动 / 折叠 / 抽屉 / 骨架 / 浮起 …)', '抽屉/拖起/toast/终端搜索/LSP 卡/模态退场已接;ok-pop / 柱状生长 / 骨架→内容交叉淡入 / 不确定进度条仍缺', 'part', 'B8'),
      R('已定义但没有调用点的动效', 'GrowY 已接用量页日/周柱、Crossfade 接用量页统计骨架→内容;OkPop / IndetMeter / FoldHeight 已在自动化页接线;未测试', 'part', 'B8')]
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
    R('审查 / 搜索 / 问题 / 自动化的空、错误、加载状态', 'Go 已补;flows 专屏+pages/usage 加载失败态', 'part', 'B0'),
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
