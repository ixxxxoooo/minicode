/* 设计稿 v3.5 · 各页「实现状态」与缺口总表
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
    ['B2', '项目 / 会话 / 排序 / 菜单', '会话行右键(删除 / 分叉等) · 方案 B 项目下拉接工作区 · 拖拽排序 · 各处右键与「···」菜单补全(SCM 标题 / 差异 / 终端 / 问题 / 输出 / 虚拟标签 / 自定义详情已做,缺 SCM 列表·树切换) · 文件新建 / 重命名 / 删除(已做,快捷键待补)'],
    ['B3', 'Git 与差异完整度', '冲突逐个切换与可编辑结果 · 差异块头常驻 · 行级暂存 · 提交并同步 · 推送被拒 / 切分支储藏 · SCM 列表 / 树 · 审查回退本回合'],
    ['B4', '会话流与任务', '前台命令实时输出 · 通知中心与系统通知 · 子 Agent / 任务停止 · 任务坞完整胶囊 · MCP 与上下文真数据 · 改动卡撤销 · 回合操作条 · 长输出限高 · 用户消息折叠 / 编辑'],
    ['B5', '编辑器 / 终端 / 语言服务', '问题面板重新检查 · 终端搜索全词 / 正则 · LSP 崩溃自动重启 · 代码操作 · 行号 Git 槽 · 迷你地图偏好 · 括号竖线着色 · 多光标 · 拆分编辑器组 · CodeLens'],
    ['B6', '设置与外壳', '设置搜索定位 · 需重启条真重启 · 快捷键录制 · 字号 · 品牌图标 / 托盘 / Dock · 窄窗响应式 · ⇧⌘N 新窗口 · 形态记忆'],
    ['B7', '整页功能', '浏览器真实抓取 · 自定义读写 ~/.claude · 用量限额 · 自动化 Cron / 项目 / 运行会话 · 画布落盘与写回 · 插件安装进度'],
    ['B8', '规范与打磨', '按钮 disabled / loading · 输入错误抖动 · 删除折叠与撤销 · 骨架 · 令牌补齐 · 动效退场 · 系统减弱动态联动']];

  /* ───────── 各页状态(只写没有自己状态表的页面) ───────── */
  const STATUS = {
    work: [
      R('Agent ⇄ IDE 原地切换(⌘E / 标题栏分段,440 ms 变形)', '会话 / 编辑器 / 终端全窗口各一份', 'ok'),
      R('四区域:导航栏 · 对话栏 · 工作区 · 状态栏;⌘B / ⌥⌘B / ⌘J 三个面板开关', '', 'ok'),
      R('会话流里点审查 / 计划 / 子 Agent / 文件 → 工作区开标签,自动展开并闪边', '', 'ok'),
      R('导航四分段(会话 / 文件 / 搜索 / 源代码管理)与整页入口、左下账号菜单', '', 'ok'),
      R('每个形态各自记住面板开合', '右侧栏 / 底栏 / 导航栏 / 对话占满与左栏分段按形态记忆;偏好 layout.memAgent / memIDE', 'ok', 'B6'),
      R('形态切换期间锁定指针事件(.morphing)', 'morph.go 过渡遮罩锁指针约 470ms', 'ok', 'B6')],
    welcome: [
      R('居中问候 + 状态行 + 输入框 + 四张起手卡(悬停上浮)+ 最近会话', '', 'ok'),
      R('窄档起手卡单列', '', 'ok'),
      R('空会话下的拖拽引用落点', '欢迎页同样可以拖文件进输入框', 'ok')],
    palette: [
      R('⌘K 打开 / Esc · 遮罩关闭、欢迎列表(命令 / 会话 / 最近文件)与模糊查询', '', 'ok'),
      R('主题二级列表:上下即时预览,Esc 还原', '', 'ok'),
      R('右侧预览栏(命令说明 / 文件前 16 行)', '窄于 820px 隐藏预览栏;极窄时面板宽度随窗口收缩', 'ok', 'B6')],
    settings: [
      R('4 组 14 类导航、改动圆点与悬停还原、外观(主题 / 方案 / 强调色 / 密度 / 圆角 / 动效)即时生效', '', 'ok'),
      R('设置内搜索', '命中后跳进分类并滚动高亮目标行(约 1.4s)', 'ok', 'B6'),
      R('需重启条', '「立即重启」自启动进程并退出(开发版 .app 路径)', 'ok', 'B6'),
      R('外观 → 时间线字号', '会话流读 appearance.fontScale;工具卡读 App.ToolDetailLevel()=appearance.density2', 'part', 'B6'),
      R('外观 → 应用图标 / 菜单栏图标两组', 'Dock 按偏好选 icns;系统深浅色跟随;托盘状态图 part', 'part', 'B6'),
      R('快捷键:点键录制、冲突当场提示', '全局快捷键读偏好;编辑器 / 终端快捷键未全部收录', 'part', 'B6'),
      R('其余偏好', '已补启动形态、忽略规则等廉价接线;括号 / 迷你地图细节仍待 IDE', 'part', 'B6'),
      R('服务商 / 规则 / 环境变量 / 语言服务器 / 项目信任', '规则/环境变量添加行、服务商 CRUD+密钥 0600;Skill 重命名待补', 'part', 'B6'),
      R('窄于 900 时左侧分类收成下拉', '左侧栏隐藏,顶部分类下拉', 'ok', 'B6')],
    forms: [
      R('单窗口、无融合 / 独立 IDE 窗口;两种形态共享会话 / 编辑器 / 终端', '', 'ok'),
      R('⇧⌘N 新窗口(再起一个进程)', '--new-window / 菜单 / 命令面板;从属进程不写偏好,窗口错开 32px', 'ok', 'B6'),
      R('启动形态(Agent / IDE / 上次)', 'general.start + layout.lastMode 已接 main 启动', 'ok', 'B6')],
    flows: [
      R('没有「检查器」:详情只落在工作区标签,每个内容一个家', '', 'ok'),
      R('打开规则矩阵:审查 / 计划 / 子 Agent / 差异 / 终端只读 / SCM 预览标签', '', 'ok'),
      R('预览斜体、双击固定、虚拟标签全窗口单份;导航历史 ← →', '', 'ok')],
    resp: [
      R('对话栏按自身宽度三档(≥640 / 420–639 / <420)', '', 'ok'),
      R('工作区按自身宽度三档(≥600 / 520–599 / <520:迷你地图、统一 diff、面包屑)', '', 'ok'),
      R('窗口变窄自动收起 / 恢复两侧栏', '固定断点 1100 / 900 / 720 + 24px 回差', 'ok', 'B6'),
      R('Agent 形态:工作区在窄窗变成滑入浮层;<720 导航栏变抽屉', 'overlays.go 浮层与抽屉;动效待阶段三对照', 'part', 'B6'),
      R('IDE 形态:窄窗对话栏收成图标条 + 浮层', '图标条 + 浮层已做;滑入动效待对照', 'part', 'B6')],
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
      R('过程流式滚动', '任务 / 子 Agent 输出贴底 + ↓ 回到底部', 'part', 'B4')],
    'ws-task': [
      R('任务输出标签:跟随输出、停止任务', '', 'ok'),
      R('「在终端面板中打开」', '只切到面板,没有开只读命令标签', 'part', 'B4'),
      R('重启任务 / 显示 pid', '演示 CmdRestartTask;真实置灰+原因;run id 行内显示', 'part', 'B4')],
      R('正常运行时注入任务', '只有截图 / 演示注入,真实回合里的后台任务未接入', 'part', 'B4')],
    flow: [
      R('步骤行(进行 / 错误 / 拒绝)、探索分组、工具展开 diff / 终端、待办、权限 / 提问 / 计划卡、提示三级', '', 'ok'),
      R('长输出限高折叠(12 行 → 196px,展开 480px)', '工具 / 终端 / 文本与 Markdown 代码块', 'ok', 'B4'),
      R('折叠工具分组的「最新一步」滚动行', '探索组折叠且运行中 22px 滑入行', 'ok', 'B4'),
      R('改动卡的「撤销」按钮', '逐文件 revertHunk', 'ok', 'B4'),
      R('回合操作条:复制 / 重试 / 分叉', '复制 / 重试 / 分叉(真实 fork: 会话);演示 fork 会话', 'ok', 'B4'),
      R('用户消息卡:长消息折叠 / 编辑', '折叠+Esc/点外取消;真实 CmdEditUser+resume_session_at(不支持则警告+仅本地)', 'part', 'B4')],
      R('工作区起始页(本轮 / 终端 / 打开三组列表;终端行列出已开终端 + 新建)', 'ws_empty.go 三组列表', 'ok', 'B4'),
      R('发送 / 停止按钮为圆形', '设计稿与 Go 版同步', 'ok', 'B4'),
      R('会话样式(跟随布局 / 对话气泡 / 通栏=与正文同宽的圆角卡片;设置 → 外观 → Agent 会话)', 'appearance.flowStyle 三档', 'ok', 'B4'),
      R('用户消息吸顶(回合内 sticky,下一轮顶走上一张,最多一张;点击就地编辑 / 取消 / 发送;设置 → 对话与提示词,默认开)', 'MyGo Sticky + chat.stickyUser', 'ok', 'B4'),
      R('输入框 MCP chip 与弹层', '服务器/工具开关;工具写本地 deny+RefreshMcp', 'part', 'B4')],
    overlays: [
      R('命令面板、历史会话、分支选择器、账号菜单、运行配置弹层', '', 'ok'),
      R('IDE 通知中心(铃铛 / 历史 / 右下角叠放)', 'Agent/IDE 共用铃铛与历史;系统通知 Darwin UN(点击回会话 part)', 'part', 'B4'),
      R('任务坞停滞 / 已结束 / 监听 / 工作流胶囊', '六类胶囊+演示 StopTask/RestartTask;真实任务靠 translate 事件', 'part', 'B4'),
      R('上下文弹层(输入框圆环 / 状态栏;分类明细 / 阈值刻度 / 账户限额 / 刷新 / 压缩;常态 / 接近阈值 / 压缩中 / 加载 / 无限额 / 失败)', 'CmdCompact+/compact;压缩中与 Notice 联动', 'part', 'B4'),
      R('更新进度胶囊四态', 'internal/update appcast+状态机;开发版重启走 scripts/dev.sh', 'part', 'B4'),
      R('确认框(模式切换 / 未保存 / 删除)', '通用确认框已用于会话删除 / 移除项目 / 切换模式', 'ok', 'B2')],
    'ide-editor': [
      R('迷你地图(语法色块、滑块拖动)、Git / 搜索 / 同词 / 诊断 / 光标刻度、概览标尺、同词高亮、窄档藏地图', '', 'ok'),
      R('行号槽的 Git 改动竖条', 'DiffBase 接线;无 HEAD 或非 Git 文件不画竖条', 'ok', 'B5'),
      R('迷你地图滑块 / 宽度 / Git 刻度三个设置', '三个偏好已接到地图绘制', 'ok', 'B5'),
      R('CodeLens(引用数 / 测试 / 问 Claude)', '库无独立 CodeLens 行,用符号行上方叠层;测试数由本地符号索引推断(Go/Rust/Py/TS);窄窗与粘性滚动错位时 CodeLens 下移', 'ok', 'B5'),
      R('迷你地图悬停显示源码', '悬停该行左侧浮层显示源码摘要', 'ok', 'B5')],
    'ide-nav': [
      R('粘性滚动、查找替换、悬停、预览定义、跳转定义 / 引用 / 类型 / 实现、补全、重命名、格式化、跳转历史', '', 'ok'),
      R('代码操作 ⌘.', 'LSP codeAction + 本地兜底;无服务器时仍为本地菜单', 'ok', 'B5'),
      R('选中工具条 / ⌘I 行内编辑', '真实模式 claude -p Haiku 生成替换预览,Esc/取消可中断;演示仍走占位', 'ok', 'B5'),
      R('行内审查(悬停接受 / 撤销)', '改动行悬停接受写入审查状态 ok、撤销走 revertHunk', 'ok', 'B5'),
      R('括号配对竖线 / 着色', '竖线接 NoIndentGuides;着色用括号深度染色(非 tree-sitter)', 'part', 'B5'),
      R('多光标', '库内置 ⌥点击加/减光标、⌘D / ⇧⌘L / ⌥⌘↑↓', 'ok', 'B5'),
      R('编辑器组向右 / 向下拆分', '两组标签 + 共享缓冲区;标签可跨组拖动', 'ok', 'B5'),
      R('Claude 行内补全(✦)', '真实 Haiku claude -p + SetSuggestion;演示占位,Tab 接受', 'ok', 'B5')],
    'ide-search': [
      R('⇧⌘F、选区预填、大小写 / 全词 / 正则、包含排除、文件折叠、虚拟列表、点击预览、替换预览、悬停忽略、全部替换', '', 'ok'),
      R('单条替换(⏎)', '替换行 ⏎ 替换当前选中匹配', 'ok', 'B5'),
      R('多根:「仅某个文件夹」范围筛选', '范围 chip + 下拉 + ✕ 清除,文件夹右键「仅在此文件夹搜索」', 'ok', 'B1'),
      R('搜索结果右键菜单', '匹配行 / 文件行;「在侧边预览」置灰(依赖编辑器拆分)', 'ok', 'B2'),
      R('空 / 错误 / 搜索中 / 结果截断四种状态的画面', '空查询 / 错误 / 搜索中 / 截断摘要已补', 'ok', 'B5')],
    'ide-problems': [
      R('状态栏错误 / 警告计数 → 问题面板、按文件分组、点击跳转、未安装语言服务器的安装卡', '', 'ok'),
      R('「重新检查」按钮与「全部 / 仅错误」筛选', '刷新按钮触发工作区重检;演示模式同步筛选', 'ok', 'B5'),
      R('工作区级诊断', '设置开关下对项目源文件 didOpen 轻量检查(有上限);仍依赖各服务器能力', 'part', 'B5')],
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
      R('图谱提交菜单:与工作区比较 / 回退…', '与工作区比较先选文件(只读差异);回退 = reset --keep', 'ok', 'B2')],
    automation: [
      R('周历(翻周 / 今天)、卡片列表筛选与启停、右侧编辑抽屉、新建 / 删除、真实落盘', '', 'ok'),
      R('计划 Cron', '五段 cron 解析、FiresOn/Due 到点触发、下次运行时间', 'ok', 'B7'),
      R('项目选择 / 权限选择', '新建模态与编辑抽屉:项目分段 + 权限分段', 'ok', 'B7'),
      R('立即运行', 'claude -p 流式输出;取消杀进程组;到点触发同样落记录', 'part', 'B7'),
      R('最近运行可点进会话', '点运行行打开只读输出浮层', 'ok', 'B7'),
      R('新建模态与失败 / 运行中状态', '设计稿 behave.js au-new/au-run;Go 模态+运行浮层', 'ok', 'B7')],
    canvas: [
      R('块编辑 / 预览、大纲定位、版本提交与回退、导出 Markdown', '', 'ok'),
      R('落盘', '~/.minicode/canvases/*.json 原子写;损坏改名留底;多画布列表', 'ok', 'B7'),
      R('让 Claude 修改', '块引用;```canvas 围栏/JSON 写回;在画布页回合结束自动解析', 'part', 'B7'),
      R('图表全屏查看 / 源码', '全屏拖动平移/适应/Esc/方向键;复制源码', 'part', 'B7'),
      R('窄窗右栏收成底栏', '宽<900 时大纲/版本落底部', 'ok', 'B7'),
      R('左侧画布列表(多画布)', '新建/切换/重命名/删除+撤销', 'part', 'B7')],
    browser: [
      R('后退 / 前进 / 刷新 / 收藏、摘要 · 大纲 · 链接、齿轮直达设置', '', 'ok'),
      R('真实网页抓取与阅读模式正文', 'net/http 抽取正文;超时/大小上限;演示源并存', 'ok', 'B7'),
      R('「与此页面对话」的页面引用', '框选段落发给 Claude;日志 RedactURL 脱敏', 'part', 'B7'),
      R('系统浏览器打开 / 隐私设置', '地址栏外链;设置 browser.fetch 可关抓取', 'ok', 'B7')],
    customize: [
      R('七类左栏、筛选与作用域、开关 / 新建 / 删除、详情表单', '数据在内存', 'ok'),
      R('读写真实的 ~/.claude 与项目 .claude', 'MCP 新建写 JSON;插件卸载 CLI;市场 list 失败兜底演示数据', 'part', 'B7'),
      R('插件市场安装进度', '四态进度条;install/uninstall CLI', 'part', 'B7'),
      R('MCP 授权 / 重连 / 工具开关接真实状态', 'claude mcp list/login + disabledMcpServers/工具 deny;无会话时状态可能不完整', 'part', 'B7'),
      R('命令 / 子 Agent 试运行、钩子试运行', 'claude -p 与 shell 试运行已接;钩子带确认与超时', 'part', 'B7'),
      R('Skills / 规则「在编辑器里打开」', '已接 openFile;用户层路径依赖绝对路径', 'part', 'B7')],
    usage: [
      R('7 天 / 30 天 / 本月、汇总卡、柱图悬停、按模型 / 项目 / 贵会话、导出 CSV、空数据加载态(真实扫描)', '', 'ok'),
      R('账户限额与刷新', '接会话 get_usage + 5 分钟缓存;无会话时如实说明', 'part', 'B7'),
      R('贵会话点击跳转到会话', 'TopSessions 带 SessionID 时 OpenSession', 'part', 'B7')],
    comps: [
      R('按钮 / 图标按钮 / 胶囊 / 标签 / 计数 / 分段 / 开关 / 标签条 / 列表行 / Spin / Meter / Chk / Dot / Kbd', '', 'ok'),
      R('按钮 disabled / loading 态', 'kit.Btn Disabled/Loading;提交/安装/重试等接线', 'ok', 'B8'),
      R('输入框统一封装与错误描边', 'kit.Field + 设置/对话框典型路径', 'part', 'B8'),
      R('菜单 / 弹层壳', 'kit.PopIn/PopOut/DialogIn;Popover 退场;旧菜单不强改', 'part', 'B8'),
      R('进度环、骨架占位', 'kit.Ring/SkelRows;用量加载/侧栏重扫', 'ok', 'B8')],
    states: [
      R('控件悬停 / 按下 / 选中、需重启条、提示条、账号菜单、设置搜索', '部分,详见下行', 'part', 'B8'),
      R('输入错误 + 抖动', 'kit.FieldErr/ShakeInput;添加项目路径等', 'part', 'B8'),
      R('删除折叠 + 6 秒撤销', 'UndoBar+折叠行;会话/自定义/设置规则', 'part', 'B8'),
      R('快捷键录制与冲突、连通性测试四态', '设置页已做;其它页未统一', 'part', 'B6'),
      R('减弱动态联动系统设置', 'applyBehavior:偏好动效 ∧ ¬SystemReduceMotion', 'ok', 'B8')],
    tokens: [
      R('六套色板、三种布局的圆角 / 间距 / 行高倍率、8 档间距、7 档字号、布局宽高', 'CSS 与 Go 一致', 'ok'),
      R('--dur-loop(循环档 1600 ms)', 'kit.DurLoop;Pulse/Spin 已读', 'ok', 'B8'),
      R('行高令牌 --lh-*', 'theme.LhTight/Body/Read;典型处已替换', 'part', 'B8')],
    motion: [
      R('时长 5 档 + 5 条曲线、形态切换 440 ms、流畅 / 简化 / 无', '', 'ok'),
      R('弹层退场(pop-out)', 'kit.PopOut+popClosing;居中 DialogOut 部分', 'part', 'B8'),
      R('文字流光 / 流底部渐隐遮罩', 'MyGo 无渐变文字,Pulse 呼吸;底部渐隐未做', 'part', 'B8'),
      R('列表入场 stagger、折叠的高度插值', '设置分组 StaggerRise;删除 CollapseHide', 'part', 'B8'),
      R('states 增补的 11 种动效(抖动 / 折叠 / 抽屉 / 骨架 / 浮起 …)', '抖动/折叠/骨架/模态入场部分有;抽屉/拖起/ok-pop 等未全接', 'part', 'B8')]
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
    R('嵌套仓库 / 子模块', '已补总览缩进、文件树标签、归属规则;「子模块有未提交改动」提示行的画面仍缺', 'part', 'B1'),
    R('搜索「仅某个文件夹」', '已补范围下拉、限定态与空态', 'ok', 'B1'),
    R('审查 / 搜索 / 问题 / 自动化的空、错误、加载状态', '这几页只有「有结果」的画面', 'part', 'B0'),
    R('layout-dock 页把「编辑器组拆分」标成已实现', '代码里拆分按钮没有接线,应改成待做', 'ok'),
    R('term-spec 页把「⌘F 搜索」标成已实现', '全词 / 正则 / 命中刻度 / 选区预填已接', 'ok'),
    R('artifact-spec 页把浏览器能力标成待做', '实际有演示源的界面,应改成部分', 'ok'),
    R('brand-spec 的「实现状态」引用 Rust 的 app_icon.rs', 'Go 分支应改写成 mygo 的现状', 'ok'),
    R('tokens 页「每套 14 色」、motion 页「时长 6 档」与实际不符', '以 tokens.css 为准:5 档时长 + 1 个循环档', 'ok'),
    R('drag-more 页的会话菜单项(复制标题 / 复制为 Markdown)与代码不一致', '以代码为准,复制为 Markdown 标待做', 'ok'),
    R('ide-scm / ide-diff / ide-merge / ide-git / ide-review 等页没有实现状态表', '用本文件补齐', 'ok')];

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
