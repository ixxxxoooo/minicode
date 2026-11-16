/* 设计稿 v3.2 · 补充规格页:布局与拖拽、终端能力、语言服务、画布 / 浏览器 / 自定义、会话流任务追踪
   每页都带「实现状态」列,如实标出 Go 版已实现 / 部分 / 待做,避免设计稿与代码对不上。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog } = U;
  const { head, sec, spec } = P.h;
  const tbl = (cols, rows) => `<table class="ia-tbl"><thead><tr>${cols.map(([t, w]) => `<th ${w ? `style="width:${w}px"` : ''}>${t}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map((c, k) => `<td ${k === 0 ? '' : 'class="t2"'}>${k === 0 ? `<b>${c}</b>` : c}</td>`).join('')}</tr>`).join('')}</tbody></table>`;
  const st = (s) => ({ ok: tag('已实现', 'ok'), part: tag('部分', 'warn'), todo: tag('待做', '') }[s]);
  const grid = (...cells) => `<div class="gal">${cells.join('')}</div>`;
  const cell = (t, keys, body, why) => `<div><div class="cap2"><b>${t}</b>${keys ? kbd(...keys) : ''}</div>${body}<div class="why">${why}</div></div>`;
  const box = (body, h = 200, w = 560, extra = '') => `<div class="snip" style="height:${h}px;width:${w}px;${extra}">${body}</div>`;

  /* ───────── 布局:拖拽与面板停靠 ───────── */
  const handle = (dir, t) => `<div class="f ai g3" style="padding:6px 0"><span class="mono t3" style="width:30px;text-align:center">${dir}</span><span>${t}</span></div>`;
  P['layout-dock'] = () => `<div class="ds" style="max-width:1240px">${head('布局 · 拖拽与面板停靠', '所有分隔线都能拖;双击还原默认;宽度与开合按形态(Agent / IDE)分别记住。终端与问题 / 输出面板可以在「工作区底部」与「占满工作区整栏」之间切换,这是 VS Code「面板位置 / 最大化面板」、Zed「停靠位置」、JetBrains「视图模式」的共同做法。')}
    ${sec('可拖的分隔线', '指针悬停分隔线 4px 内变成 resize 光标并高亮 2px 强调色;拖动中全局锁定选中文字;越过限制值就停住,不会把内容挤没。', tbl([['分隔线', 150], ['范围', 120], ['双击', 100], ['拖到极限时', 200], ['状态']], [
      ['导航栏 | 对话栏 / 工作区', '200 – 420', '还原 260', '低于 200 → 吸附成「收起」', st('ok')],
      ['对话栏 | 工作区(Agent)', '工作区 360 – 窗口 −420', '还原 560', '对话栏最小 420,优先保对话', st('ok')],
      ['工作区 | 对话栏(IDE)', '对话栏 300 – 600', '还原 352', '低于 300 → 收成图标条', st('ok')],
      ['编辑器 | 底部面板', '面板 120 – 工作区 −120', '还原 240', '拖过最大值 −12px 吸附成「占满」;拖离再还原', st('ok')],
      ['编辑器组之间(拆分)', '各 ≥ 200', '均分', '标签条拆分钮 / 菜单 / ⌘\\ 已接;拖动分隔条调比', st('ok')]]))}
    ${sec('底部面板:位置与大小', '三种停靠状态,标题栏右侧的按钮就是切换器。任何状态都有「出口」:占满时按钮变成「还原面板」,Esc 在面板获得焦点且没有搜索条时还原。', grid(
      cell('底部(默认)', ['⌘', 'J'], box(`<div class="reg-win" style="height:170px;margin:14px"><div class="rw" style="flex:1"><b>编辑器</b></div></div><div style="position:absolute;left:14px;right:14px;bottom:14px;height:60px;border-radius:8px;background:var(--bg-sunken);box-shadow:inset 0 0 0 1px var(--bd-2);padding:8px 10px" class="mono t3">终端 / 问题 / 输出</div>`, 200), '默认:编辑器在上,面板在下,高度记住。'),
      cell('占满工作区整栏', ['⇧', '⌘', 'J'], box(`<div style="position:absolute;inset:14px;border-radius:8px;background:var(--bg-sunken);box-shadow:inset 0 0 0 1px var(--bd-2);padding:10px 12px"><div class="f ai g3"><b>终端</b><span class="grow"></span>${ib('max', 'sm on', '还原面板')}</div><div class="mono t3" style="margin-top:8px">$ cargo test<br>…</div></div>`, 200), '标签条保留(可切回编辑器标签);编辑器标签挤到面板上方一条细横条里,点任何编辑器标签 = 自动还原。适合长时间看日志、跑 TUI。'),
      cell('拖动吸附', '', box(`<div class="f ai g3" style="padding:14px 18px"><span class="mono">▲</span><div class="t2" style="line-height:1.7">手柄拖到顶部边缘 12px 内 → 出现整栏蓝色预览框 → 松手占满。<br>占满状态把标题栏往下拖超过 40px → 还原到上次高度。</div></div>`, 120), '拖动手势与按钮手势完全等价;任一状态都能退出,没有「只进不出」。')))}
    ${sec('终端「搬家」的取舍', '问题:要不要像 VS Code 那样把终端整个挪到右侧栏、或像 Zed 那样停靠到左右。结论:本产品右侧要留给对话(IDE 形态)/ 工作区(Agent 形态),所以只做「底部 ↔ 占满」两态,不做左右停靠;长时间的命令输出走会话流里的「任务」标签(工作区虚拟标签),那是另一个更适合并排的位置。', tbl([['场景', 200], ['怎么做'], ['状态']], [
      ['盯着长任务输出,同时继续聊天', 'IDE 形态:底部面板拉高;Agent 形态:任务标签在右侧工作区打开,对话照常居中', st('ok')],
      ['需要整屏终端', '占满工作区整栏(⇧⌘J),对话栏不动', st('ok')],
      ['终端要停靠到左 / 右侧', '不做:右侧是对话。极端需求用 ⇧⌘N 新开窗口', tag('不做')]]))}
    ${sec('工作区起始页与多个终端', '工作区没有任何标签时显示起始页:三组左对齐列表(本轮 / 终端 / 打开),只是「跳转」,内容的家不变。「本轮」只列有内容的项,都没有就一行灰字,不放灰掉的入口。「终端」每个已开的终端一行,末尾「新建终端」;起始页的终端行和底部面板的标签是同一份状态。', grid(
      cell('有内容 · 一个终端', '', box(`<div style="height:100%;display:flex;flex-direction:column">${WS.empty({ files: 2, plan: 1, tasks: 1, terms: 1, sel: 0 })}</div>`, 520, 520, 'display:flex;flex-direction:column'), '「本轮」三行带计数;终端组只有 zsh 一行 + 新建终端;点终端行 = 展开底部面板并聚焦它。'),
      cell('什么都没有 · 还没开终端', '', box(`<div style="height:100%;display:flex;flex-direction:column">${WS.empty({ files: 0, plan: 0, tasks: 0, terms: 0 })}</div>`, 520, 520, 'display:flex;flex-direction:column'), '「本轮」一行灰字;终端组只有「打开终端 ⌘J」。'),
      cell('开了第二个终端', '', box(`<div style="height:100%;display:flex;flex-direction:column">${WS.empty({ files: 0, plan: 1, tasks: 0, terms: 2, sel: 1 })}<div style="height:150px;flex:none;display:flex;flex-direction:column;box-shadow:inset 0 1px var(--bd-2)">${IDE.bpanel('term', { terms: 2, sel: 1 })}</div></div>`, 700, 520, 'display:flex;flex-direction:column'), '终端组出现「· 2 个」与两行;底部面板标签条 zsh · zsh 2,选中的带 ×,第二个终端是新的空白 shell。')))}
    ${sec('多个终端怎么交互', '终端的家是底部面板(位置只有底部 / 占满两态);起始页与「+」只是入口。多个终端并存,但一次只显示一个(不并排拆分)。', tbl([['动作', 220], ['结果'], ['状态', 80]], [
      ['起始页「打开终端」/ ⌘J', '展开底部面板,在项目目录启动第一个 shell,焦点进终端', st('ok')],
      ['面板「+」/ 起始页「新建终端」', '新增 zsh 2(再多依次 zsh 3…)并切过去;面板展开并闪一下边框;起始页终端组多一行、标题带「· N 个」;新 shell 在项目根目录(文件树「在此处打开终端」则在该目录)', st('ok')],
      ['点面板标签 / 起始页某个终端行', '切到那个终端,shell 不重启、滚动与输入保留;起始页行点击同时展开面板并聚焦', st('ok')],
      ['选中标签上的 ×', '结束该 shell(有前台命令时先确认);只剩一个时不给 ×;关掉最后一个后留空白标签,下次显示再开', st('ok')],
      ['Claude 跑的命令', '只读标签(锁形),排在终端标签之后;「在终端面板查看」跳到它', st('ok')],
      ['拆分并排 / 终端分组', '不做:多个终端一次只显示一个;要并排看输出用工作区的「任务」标签(重命名见上方「终端 · 能力与主题」)', tag('不做')]]))}
    ${sec('Agent / IDE 切换的布局记忆', '每个形态各自记住「导航栏 / 工作区 / 底部面板」的开合与宽度,切形态时用 440ms 的宽度过渡;窗口里同一份会话 / 编辑器 / 终端不会被重建。', tbl([['记忆项', 160], ['Agent', 120], ['IDE'], ['状态']], [['导航栏', '开,会话', '开,文件', st('ok')], ['工作区', '默认收起,开标签自动展开并闪边框', '占满主区', st('ok')], ['底部面板', '关', '开', st('ok')], ['对话栏占满', '—', '右侧栏「占满」可退出(头部按钮变「还原」,Esc 同效)', st('ok')]]))}</div>`;

  /* ───────── 终端 ───────── */
  P['term-spec'] = () => `<div class="ds" style="max-width:1240px">${head('终端 · 能力与主题', '对标 VS Code / iTerm2 / Warp 的共同底线。终端颜色不单独切换:前景 / 背景 / 光标 / 选区 / 16 色 ANSI 全部来自当前主题令牌,换主题终端立即跟随。')}
    ${sec('能力清单', '', tbl([['能力', 160], ['交互', 300], ['实现状态']], [
      ['鼠标选择与复制', '拖选字符、双击选词、三击选行、⇧点击扩展;⌘C 复制(选区存在时),无选区时 ⌘C 不拦截;选中即高亮,点任意处取消;右键菜单:复制 / 粘贴 / 全选 / 清屏 / 在访达中打开', st('ok')],
      ['在输出里搜索', '⌘F 打开右上角搜索条:区分大小写 / 全词 / 正则、计数「3 / 12」、⏎ 下一个 ⇧⏎ 上一个、Esc 关闭;命中在滚动条上出刻度;有选区时预填', st('ok')],
      ['超链接', '识别 URL 与 <span class="mono">path:line:col</span>;⌘ 悬停出下划线与手型,⌘点击:URL 用系统浏览器、路径在编辑器打开并定位', st('ok')],
      ['字体与光标设置', '面板右上齿轮弹出:字体大小(11–20)、字体族、行高、光标样式(块 / 竖线 / 下划线)与闪烁;完整项在 设置 → 终端', st('ok')],
      ['主题', '跟随全局主题;不提供单独的终端配色开关;「调整对比度」保证 ANSI 文字在底色上 ≥ 4.5:1', st('ok')],
      ['多标签', '新建 / 关闭 / 重命名;来自会话的命令是只读标签(带锁);shell 退出后显示「按回车重开」。重命名:双击标签或右键「重命名…」,⏎ 提交 / Esc 取消,留空或与默认名相同 = 恢复默认;起始页终端列表同名。<b>已在开发版实测</b>(此前第 0 个标签因 ix 零值一直显示成空输入框,已修)', st('ok')],
      ['滚动与历史', '触控板平滑滚动;滚离底部时右下角出现「↓ 回到底部」;历史 10000 行(设置里可调)', st('ok')],
      ['鼠标上报(vim / htop / tmux 里用鼠标)', '应用请求鼠标模式时把点击 / 滚轮编码发给程序;按住 ⌥ 临时回到选择模式。<b>已在开发版实测</b>(滚轮与点击的 SGR 编码;此前移动事件会漏报给只要点击的程序,已按模式过滤)', st('ok')],
      ['Nerd Font 图标', '画面里出现私有区图标码点时,顶部提示条:本机没装 → 给安装命令;已装 Nerd Font → 引导到「终端设置」选用;「不再提示」写偏好 terminal.nerdHint。字体按「系统已装字体」扫描(后台),只有选了未安装的 Nerd 字体才回退系统等宽。<b>已在开发版实测</b>(提示条按钮可点,此前叠在正文上的按钮会被选择处理吃掉,已修)。未验证:装好字体后的图标渲染', st('ok')],
      ['命令装饰(Shell 集成)', 'OSC 133(zsh/bash)、命令行左侧装饰点(绿 = 成功 / 红 = 失败,悬停提示退出码)、点左边距复制该命令的输出、清屏重置、偏好 term.shellIntegration。<b>已在开发版实测</b>(真实 zsh + oh-my-zsh:false 红点 / ls 绿点、剪贴板核对;期间修了 zsh 缺 autoload、OSC 载荷带编号前缀、读光标时的死锁、空输出命令抢点击)。fish 未做', st('ok')]]))}
    ${sec('终端面板的头部', '一行放下:标签 · 新建 · 更多 · 搜索 · 齿轮 · 占满 · 收起。标签放不下时横向滚动,不折行。', box(IDE.bpanel('term'), 190))}
    ${sec('选择 / 搜索 / 链接的视觉', '', grid(
      cell('选区', '', box(`<div class="mono" style="padding:14px 16px;line-height:1.7"><span class="t3">~/router-demo</span> $ cargo test<br><span style="background:var(--ac-soft);box-shadow:0 0 0 1px var(--ac-line)">test matches_ignoring_case ... ok</span><br>test result: ok. 12 passed</div>`, 120), '选区色 = 强调色 28% 不透明;失焦后变灰。'),
      cell('搜索条', ['⌘', 'F'], box(`<div class="mono" style="padding:14px 16px;line-height:1.7">Compiling router-demo<br>test <mark style="background:var(--warn-soft);color:inherit;outline:1px solid var(--warn)">matches</mark>_ignoring_case ... ok</div><div class="pop" style="position:absolute;right:10px;top:10px;padding:4px 6px;display:flex;gap:4px;align-items:center">${ic('search', 's12')}<span class="mono">matches</span>${chip('Aa')}${chip('ab')}${chip('.*')}<span class="t3 fs-cap">1 / 3</span>${ib('arrow', 'sm')}${ib('x', 'sm')}</div>`, 130), '当前命中橙色实线框,其余淡黄;计数实时。'),
      cell('超链接', ['⌘', '点击'], box(`<div class="mono" style="padding:14px 16px;line-height:1.7">error[E0425]: cannot find value<br>&nbsp;&nbsp;--&gt; <u style="color:var(--ac);text-decoration-thickness:1px">src/router.rs:23:9</u><span class="t3"> ← ⌘点击在编辑器打开</span></div>`, 110), '悬停才出现下划线;不按 ⌘ 时点击仍是普通光标定位。')))}</div>`;

  /* ───────── 语言服务 ───────── */
  const langRow = (n, srv, state, act) => `<div class="set-row"><span class="si-box">${ic('code')}</span><div class="grow"><div class="m">${n}</div><div class="d mono">${srv}</div></div>${state}${act}</div>`;
  P['lsp-spec'] = () => `<div class="ds" style="max-width:1240px">${head('语言服务(LSP)· 识别、安装、状态', '原则:打开文件才启动(不预热);找不到就说清楚并给一条能复制 / 一键运行的安装命令;失败不阻塞编辑,自动退回「本地符号索引」(正则级,不懂语义)。')}
    ${sec('从打开文件到可用:状态机', '', tbl([['阶段', 120], ['界面', 300], ['说明']], [
      ['识别', '无', '按扩展名选服务器:rs → rust-analyzer、go → gopls、py → pyright / pylsp、lua → lua-language-server、ts/js → typescript-language-server、c/cpp → clangd、swift → sourcekit-lsp'],
      ['查找', '无', 'PATH + 常见目录(~/go/bin、~/.cargo/bin、Homebrew、npm 全局);rust-analyzer 跳过 rustup 代理(工具链没装该组件时它一启动就退出)'],
      ['未安装', '状态栏「⚠ rust-analyzer 未安装」;底部「输出 → 语言服务」写明命令;点状态栏弹出安装卡', '安装卡:命令文本 + 「在终端里运行」(开一个可见终端标签并预填命令,让用户看见过程)+「复制」;每种服务器每次运行只主动提示一次'],
      ['启动中', '状态栏转圈「rust-analyzer 索引中 42%」(来自 $/progress)', '期间悬停 / 跳转可用但可能不全,用本地索引兜底'],
      ['就绪', '状态栏「rust-analyzer ✓」;诊断波浪线、悬停、补全、跳转、引用、重命名、格式化可用', '编辑后 250ms 去抖发 didChange(整篇文本);请求前先补同步'],
      ['崩溃', '状态栏「… 已停止 · 正在重启」;退避自动重启最多 3 次,仍失败则停并提示看日志', '日志在「输出 → 语言服务」通道']]))}
    ${sec('设置 → 编辑器与语言服务', '每种语言一行:服务器名、路径(可覆盖)、状态、操作。自动安装不静默执行,所有安装都经由可见终端。Go: settings_lsp.go 后台扫 lsp.Specs + 偏好 lsp.server.* / editor.lsp,未测试。', `<div class="spec" style="padding:4px 10px">${langRow('Rust', 'rust-analyzer · ~/.rustup/…/bin/rust-analyzer', tag('就绪', 'ok'), btn('重启', 'sm ghost'))}${langRow('Go', 'gopls · ~/go/bin/gopls', tag('就绪', 'ok'), btn('重启', 'sm ghost'))}${langRow('Python', 'pyright-langserver', tag('未安装', 'warn'), btn('安装…', 'sm soft'))}${langRow('Lua', 'lua-language-server', tag('未安装', 'warn'), btn('安装…', 'sm soft'))}${langRow('TypeScript', 'typescript-language-server', tag('就绪', 'ok'), btn('重启', 'sm ghost'))}</div>`)}
    ${sec('安装命令', '只给官方推荐的一条,不替用户选包管理器之外的方案。', tbl([['语言', 100], ['命令', 380], ['备注']], [
      ['Rust', '<span class="mono">rustup component add rust-analyzer</span>', '随工具链,版本与 rustc 匹配'],
      ['Go', '<span class="mono">go install golang.org/x/tools/gopls@latest</span>', '需要 GOBIN 在 PATH'],
      ['Python', '<span class="mono">npm i -g pyright</span> 或 <span class="mono">pipx install pyright</span>', '二选一,检测到谁用谁'],
      ['Lua', '<span class="mono">brew install lua-language-server</span>', 'macOS;其它平台给发行页链接'],
      ['TypeScript', '<span class="mono">npm i -g typescript typescript-language-server</span>', ''],
      ['C / C++', '随 Xcode 命令行工具(<span class="mono">xcode-select --install</span>)', 'clangd']]))}
    ${sec('编辑器里能用到什么', '', tbl([['能力', 160], ['键位', 120], ['来源', 160], ['状态']], [
      ['诊断(波浪线 / 问题面板 / 状态栏计数)', '', 'publishDiagnostics', st('ok')], ['悬停', '', 'textDocument/hover', st('ok')], ['转到定义 / 类型定义 / 实现', 'F12 / 右键', 'definition / typeDefinition / implementation', st('ok')],
      ['查找引用 → 底部「引用」面板', '⇧F12', 'references', st('ok')], ['补全', '⌃Space', 'completion', st('ok')], ['重命名(跨文件)', 'F2', 'rename + 工作区编辑', st('ok')], ['格式化', '⇧⌥F', 'formatting', st('ok')],
      ['大纲 / 转到符号', '⇧⌘O', 'documentSymbol(失败退回本地索引)', st('ok')], ['代码操作(快速修复)', '⌘.', 'codeAction', st('ok')], ['内联提示(inlayHint)', '', 'inlayHint', st('ok')], ['语义高亮', '', 'semanticTokens:MyGo 没有前景着色 API,明确不做(见 lsp_semantic.go)', st('todo')],
      ['索引进度「42%」', '', '$/progress→lsp.FormatProgressLine→s.work;启动中状态栏显示百分比(lsp/progress_test.go、lsp_status_progress_test.go);未 Computer Use 实测', st('part')]]))}</div>`;

  /* ───────── 画布 / 浏览器 / 自定义:业界做法 ───────── */
  P['artifact-spec'] = () => `<div class="ds" style="max-width:1240px">${head('画布 · 浏览器 · 自定义:业界做法与取舍', '依据:Claude Artifacts、ChatGPT Canvas、Cursor Canvases(React 画布,作为持久产物与终端 / 浏览器 / 源代码管理并列)、AX Patterns「侧栏与产物」、AI Patterns「Artifact Panel」。共同点比差异多,下面只落共同点。')}
    ${sec('共同的七条做法', '', tbl([['做法', 220], ['含义', 360], ['MiniCode 的落点', 280], ['状态']], [
      ['对话是方向盘,产物有自己的家', '计划/Mermaid/审查等=虚拟标签(ws_tabs);画布整页 canvas_page;缺口:会话自动拉起画布整页', st('part')],
      ['自动打开,按需打开', 'EvPlanUpdated 流式自动开计划标签;Mermaid 卡「查看」;缺口:旧消息卡片按版本回放', st('part')],
      ['版本不覆盖', 'canvas_page verIx 版本步进+ApplyClaudeReply;缺口:生成中旧版变暗', st('part')],
      ['预览 / 代码双视图', 'Preview 与 Code 是 tablist,可复制当前版本源码', '编辑/预览/源码三分段+复制 Markdown', st('ok')],
      ['按容器宽度决定并排还是盖住', '约 672px 以上并排;更窄覆盖对话,Close 返回并还原滚动与焦点', '工作区三档 + 窄档强制统一布局', st('ok')],
      ['可关闭、可返回、焦点归位', 'mermaid_viewer Esc/遮罩关闭;虚拟标签 Esc;缺口:关闭后焦点未统一回到触发卡片', st('part')],
      ['人与 AI 共享同一份状态', '用户改的和 AI 改的在同一对象上,有撤销;AI 改动逐块接受 / 撤销', '审查标签 + 编辑器行内审查', st('ok')]]))}
    ${sec('画布页(整页)的信息架构', '画布 = 「Claude 为你生成的、可交互的持久页面」:仪表盘、PR 审查板、学习笔记。不是白板。', grid(
      cell('布局', '', box(`<div class="reg-win" style="height:150px;margin:12px"><div class="rn" style="width:90px">画布列表<br>最近 / 项目</div><div class="rc" style="flex:1">画布正文<br>(Preview / Source)</div><div class="rw" style="width:130px"><b>大纲</b>版本<br>让 Claude 修改</div></div>`, 180), '左:列表;中:正文;右:大纲、版本、修改提示。窄窗口右栏收成底部分段抽屉。'),
      cell('块类型', '', box(`<div class="f g2 wrap" style="padding:16px">${chip('文本', 'fileText')}${chip('表格', 'grid')}${chip('图表', 'chart')}${chip('Mermaid', 'fork')}${chip('代码', 'code')}${chip('待办', 'list')}${chip('差异', 'diff')}${chip('指标卡', 'activity')}</div><div class="t3" style="padding:0 16px">复用会话流里已有的组件(差异、待办),不另造一套;图表遵循数据可视化规范:单位、图例、无障碍对比度。</div>`, 130), '块是有类型的、可校验的数据,不是任意 HTML;不执行用户脚本,安全边界清晰。'),
      cell('让 Claude 修改', '', box(`<div style="padding:14px 16px" class="fc g3"><div class="inp"><span class="grow t3">把图表改成按周聚合…</span>${ib('send', 'send')}</div><div class="f g2">${chip('v3', 'clock', 'on')}${tag('写入中…', 'ac')}</div><div class="t3 fs-cap">旧版本保持可见并变暗;完成后自动切到 v4,「1 处改动 · 查看改动」。</div></div>`, 140), '修改 = 在同一画布上产生新版本;生成期间禁用编辑,避免被覆盖。')))}
    ${sec('浏览器页(阅读模式 + 与页面对话)', '参考 Cursor 内置浏览器与阅读器:这个页面不是通用浏览器,是「让 Claude 读网页」的工作面。', tbl([['能力', 160], ['说明', 420], ['状态']], [
      ['地址栏 + 阅读模式正文', '抓取后抽取正文(标题、段落、代码、图片说明),离线可读;「在系统浏览器打开」始终在', st('ok')],
      ['摘要 / 大纲 / 链接', 'browser_page 右栏分段+internal/browser 抓取。未测试', st('ok')],
      ['与此页面对话', '右栏输入框 + 快捷提示;页面正文作为引用附进对话,不另开会话', st('ok')],
      ['选取元素发给 Claude', 'browser_select 框选+「发给 Claude」;browser_page 接线。未测试', st('ok')],
      ['隐私', '抓取只发生在本机,不经第三方;设置 → 浏览器可关', st('ok')]]))}
    ${sec('自定义页(7 类)', '参考 VS Code 扩展视图 + Claude Code 的 /plugin /mcp /skills:左侧分类,中间列表,右侧详情;「来源作用域」(用户 / 项目 / 本地)是每一行的一等属性。', tbl([['分类', 110], ['要点', 520], ['状态']], [
      ['插件', 'custom_wire/custom_detail 市场 JSON+安装四态+CLI;演示内存/真实 OpenDisk。未测试', st('ok')], ['MCP', 'custom_mcp_edit+SaveMCP;composer_mcp 工具 deny。未测试', st('ok')], ['Skills', 'custom_skill_save 重命名;详情「文件」纵向列表(路径+大小、可滚动选中),下方预览选中文件;复制路径/在编辑器打开针对选中项。未测试', st('ok')],
      ['命令 / 子 Agent / 钩子', 'custom_wire 读写盘+试运行。未测试', st('ok')], ['规则与记忆', 'custom_wire CLAUDE.md/rules 原子写+编辑器。未测试', st('ok')]]))}
    <div class="t3" style="margin-top:14px;line-height:1.7">说明:浏览器 HTTP/框选已接线;画布版本步进在 canvas_page;会话流 Mermaid(flow_mermaid)全屏查看;均未做开发版实测。</div></div>`;

  /* ───────── 会话流任务追踪 ───────── */
  P['task-spec'] = () => `<div class="ds" style="max-width:1240px">${head('会话流 · 待办、审批、后台任务', '三类「需要你注意的东西」各有一个固定的家:正在做的事在会话流里(步骤行),需要你拍板的事就地替换输入框(交互卡),脱离回合继续跑的事在输入框上方的任务坞。互不抢位置,不弹模态框。')}
    ${sec('三个落点', '', tbl([['类别', 130], ['出现在哪', 260], ['谁先谁后', 300], ['点击去哪']], [
      ['待办(TodoWrite)', '会话流里一行「待办 2 / 3 完成」,展开是清单;最新一份才有进度环', '每次 TodoWrite 更新同一行,不新增行', '展开 / 收起;审查标签的「执行进度」读同一份'],
      ['审批(权限 / 提问 / 计划)', '输入框原位替换成交互卡(同一位置、同一宽度);计划卡留在正文', '同时多个时按到达顺序排队,卡头显示「1 / 3」;处理完输入框原位回来', '权限:允许一次 / 总是允许 / 拒绝(带理由);计划:Build 三种模式 / 继续规划'],
      ['后台任务(子 Agent / 后台命令 / 监听 / 工作流)', '输入框上方「任务坞」胶囊一排;点哪个胶囊在上方展开那一类列表', '运行中在前,已结束折在「已结束」胶囊', '列表项 = 工作区开对应虚拟标签(子 Agent 记录 / 任务输出);「在终端面板中打开」才进底部面板']]))}
    ${sec('就地替换的交互卡', '卡片出现时输入框收起、焦点进卡片;处理后 120ms 内淡出,输入框原位回来并恢复焦点和草稿。Esc = 拒绝(权限)/ 跳过(提问)。', grid(
      cell('权限请求', '', `<div class="snip" style="height:auto;padding:12px 14px">${F.perm()}</div>`, '显示工具名、将执行的命令 / 路径、影响范围;「总是允许」写入项目本地规则并可在 设置 → 权限与信任 撤销。'),
      cell('提问', '', `<div class="snip" style="height:auto;padding:12px 14px">${F.ask()}</div>`, '单选 / 多选 / 自由输入;多题时顶部步骤条可来回切换(详见「会话流 · 排队引导 / 问答卡 / 回到最新」);提交前可改;跳过也是一个明确的选项。')))}
    ${sec('待办与任务坞', '', grid(
      cell('待办清单', '', `<div class="snip" style="height:auto;padding:12px 14px">${F.step('todo', '2 / 3 完成', '', '', { open: true, body: F.todo(2) })}</div>`, '三态:未开始 / 进行中(旋转)/ 已完成(划掉并变淡);只有一条进行中。'),
      cell('任务坞 + 排队', '', `<div class="snip" style="height:auto;padding:12px 14px">${F.tdock()}${F.queue()}</div>`, '待发送区:排队与引导共用(详见「会话流 · 排队引导 / 问答卡 / 回到最新」);点文字退回输入框编辑 / 改为引导 / 停止并发送 / 删除。回合被中断时队列暂停,出「继续 / 撤回」。')))}
    ${sec('后台任务的状态与保活', '', tbl([['情形', 220], ['行为'], ['状态']], [
      ['会话切走时回合还在跑', '后端进程保活,回合结束后才关闭;侧栏会话行显示运行状态点;切回来接上,不重放', st('ok')],
      ['子 Agent 停滞', '超过阈值显示「疑似停滞 · 停止」;前台子 Agent 随取消变「已停止」,后台的继续', st('ok')],
      ['重开历史会话里遗留的后台任务', '统一收尾成「已停止」,不再显示运行中', st('ok')],
      ['前台命令的实时输出', 'flow_tail+pollTaskOutputFiles 尾读 output_file;演示 tools-bash-fg-out。未测试', st('ok')],
      ['完成通知', 'notice_sys.afterEventNotify:后台时 maybeSystemNotify(回合/审批/任务);偏好 general.notify*。未测试', st('ok')]]))}
    ${sec('实现状态', '', tbl([['范围', 220], ['说明', 440], ['状态']], [
      ['待办 / 权限·提问·计划卡 / 任务坞', 'dock_pending+dockBadge 审批 1/N;flow_note+flow_user TaskNote;dock 排队含停止并发送;演示 scn_flow_gaps。未测试', st('ok')],
      ['前台命令 output_file 实时尾读', '同上一节 flow_tail.go。未测试', st('ok')],
      ['系统通知与点击回会话', 'mac_notify_darwin→OpenSessionFromNotify;非 .app 包走 osascript 无点击回调。未测试', st('ok')],
      ['回合操作条分叉', '演示 + 真实 CmdForkSession;对话栏「分叉会话…」与之同路径', st('ok')]]))}
    ${sec('不同形态下「点哪里打开」', '同一个入口在 Agent 与 IDE 形态里打开的位置不同,但永远是「工作区的标签」,不是新窗口也不是模态框。', tbl([['入口', 180], ['Agent 形态', 240], ['IDE 形态']], [
      ['计划卡 [查看]', '右侧工作区展开并开「计划」标签', '编辑区开「计划」标签,对话栏仍在右侧窄栏'],
      ['子 Agent 卡 / 任务坞项', '右侧工作区「子 Agent 记录」标签', '编辑区开同一标签'],
      ['后台命令 [查看输出]', '右侧工作区「任务输出」标签', '底部面板「终端」里的只读标签(IDE 里有现成终端区)'],
      ['改动卡 [审查]', '右侧工作区「审查」标签', '编辑区「审查」标签;左栏切到源代码管理'],
      ['文件链接', '右侧工作区编辑器标签', '编辑区标签']]))}</div>`;

  /* ───────── 问题面板 ───────── */
  const probRow = (lv, msg, src, ln) => `<div class="row">${ic(lv === 'err' ? 'errc' : 'alert', 's12 ' + (lv === 'err' ? 'c-err' : 'c-warn'))}<span class="ell">${msg}</span><span class="t4 fs-cap nw">${src}</span><span class="aux">${ln}</span></div>`;
  const probHead = (scope, o = {}) => `<div class="tabs"><span class="tab ${scope === 'open' ? 'on' : ''}">问题 · 已打开 ${cnt(o.open || 3, scope === 'open' ? 'ac' : '')}</span><span class="tab ${scope === 'ws' ? 'on' : ''}">工作区 ${cnt(o.ws || 12, scope === 'ws' ? 'ac' : '')}</span><span class="tab">输出</span><span class="grow"></span><span class="f ai g1">${scope === 'ws' ? `${chip('全部', 'filter', 'on')}${chip('仅错误', 'filter')}${ib('refresh', 'sm', '重新检查')}${o.scan ? btn('取消', 'sm ghost') : ''}` : `${chip('全部', 'filter', 'on')}${ib('refresh', 'sm', '重新检查')}`}${ib('max', 'sm')}${ib('x', 'sm', '收起 ⌘J')}</span></div>`;
  const probBody = (rows) => `<div class="grow scroll" style="padding:2px 6px;min-height:120px">${rows}</div>`;
  P['ide-problems'] = () => `<div class="ds" style="max-width:1240px">${head('问题面板', '底部面板「问题」= 语言服务器诊断的家;状态栏错误 / 警告计数点击同一面板。默认只看「已打开」文件的诊断;开启工作区级扫描后扩大范围。')}
    ${sec('已打开 · 工作区分段', '分段切换不清空筛选;点击行跳转并定位(保持面板开)。分组折叠状态记忆到 state.json。', grid(
      cell('已打开(默认)', '', box(`<div class="bpanel tall" style="height:auto;box-shadow:none">${probHead('open')}<div class="grow scroll" style="padding:2px 6px"><div class="grp-h">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow">router.rs</span>${cnt(2)}</div>${probRow('warn', '`fallback` 字段从未被读取', 'rust-analyzer', '行 5')}${probRow('warn', '未使用的变量:`h`', 'rust-analyzer', '行 11')}
        <div class="grp-h">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow">router_test.rs</span>${cnt(1)}</div>${probRow('err', '找不到函数 `normalize`', 'rust-analyzer', '行 8')}</div></div>`, 220), '仅已打开且已同步的语言服务会话;未打开文件不出现在此分段。'),
      cell('工作区 · 扫描中', '', box(`<div class="bpanel tall" style="height:auto;box-shadow:none">${probHead('ws', { ws: 0, scan: 1 })}<div class="t3 fs-cap" style="padding:4px 12px;display:flex;align-items:center;gap:8px"><span class="grow">已检查 <b>128</b> / 640 个文件</span><span>23%</span></div><div class="prog-scan"><i></i></div>${probBody('<div class="t3" style="padding:20px;text-align:center">扫描中… 已发现的诊断会边扫边出现</div>')}</div>`, 180), '取消 = 停止当前代次,保留已收集结果;新一次「重新检查」递增代次号。'),
      cell('工作区 · 完成', '', box(`<div class="bpanel tall" style="height:auto;box-shadow:none">${probHead('ws', { ws: 12 })}<div class="t3 fs-cap" style="padding:2px 12px 6px">共检查 640 个文件 · 来源:服务器 pull</div>${probBody(`<div class="grp-h">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow">handlers/mod.rs</span>${cnt(1)}</div>${probRow('warn', '未使用的 import', 'rust-analyzer', '行 3')}`)}</div>`, 200), '服务器支持 <span class="mono">workspace/diagnostic</span> 时标注「来源:服务器 pull」;否则「来源:逐个打开扫描」,且只含实际扫过的文件。'),
      cell('截断', '', box(`<div class="bpanel tall" style="height:auto;box-shadow:none">${probHead('ws', { ws: 2000 })}<div class="notice warn" style="margin:0;border-radius:0;padding:6px 12px">${ic('alert', 'c-warn')}<span class="grow t2">已达上限 2000 个文件,只显示前 2000 个</span>${btn('调高上限…', 'sm ghost', 'sliders', 'data-act="settab" data-tab="editor" data-goto="settings"')}</div></div>`, 150), '链到 设置 → 编辑器与语言服务 的扫描上限;调高后需手动「重新检查」。'),
      cell('空态', '', box(`<div class="bpanel tall" style="height:auto;box-shadow:none">${probHead('ws', { ws: 0 })}${probBody('<div class="t3" style="padding:28px;text-align:center">没有发现问题</div>')}</div>`, 130), '工作区与已打开两种分段各自独立空态。'),
      cell('错误态', '', box(`<div class="bpanel tall" style="height:auto;box-shadow:none">${probHead('ws', { ws: 0 })}<div class="notice err" style="margin:0;border-radius:0;padding:8px 12px">${ic('errc', 'c-err')}<span class="grow"><b>rust-analyzer 没有响应</b><span class="t3"> · 超时 30 s</span></span>${btn('重试', 'sm pri')}</div></div>`, 120), '崩溃 / 超时 / 用户取消都走通知条 + 可重试;重试清空错误条并重新排队扫描。')))}
    ${sec('设置与规则', '', `<div class="spec" style="padding:8px 10px;margin-bottom:12px"><div class="set-row"><span class="m grow">工作区级诊断</span>${tog(1)}</div><div class="set-row"><span class="m grow">工作区扫描文件上限</span><span class="sel"><span class="grow">2000</span>${ic('chevD', 's12 car')}</span></div><div class="set-row"><span class="m grow">扫描时包含被 gitignore 的文件</span>${tog(0)}</div></div>
      ${tbl([['项', 200], ['规则', 520]], [
        ['重新检查', '工作区分段下刷新 = 新扫描;已打开分段 = 对已打开文档 <span class="mono">didOpen</span> 同步 + 请求诊断'],
        ['筛选', '「全部 / 仅错误」只影响列表,不改状态栏计数(计数始终错误 + 警告)'],
        ['跳转', '点击行 → 编辑器开/切换文件并定位;若文件未打开则先打开再跳'],
        ['未安装服务器', '该语言的行显示安装卡(与状态栏同源),不占工作区扫描配额']])}`)}</div>`;

  /* ───────── 滚动与滚动条 ───────── */
  P['scroll-spec'] = () => `<div class="ds" style="max-width:960px">${head('滚动 · Shift 滚轮与滚动条', '对标 macOS overlay / VS Code / Zed:鼠标纵向滚轮 + Shift 横向滚;滚动条滚动时出现,静止约 3 秒隐藏,仅指针停在滚动条上或拖动时保持。')}
    ${sec('Shift + 滚轮', '触控板横向手势走系统;鼠标滚轮在任意可横向滚动区域按住 Shift 后纵向刻度转为横向。双向容器上 Shift 时只横向不纵向。', tbl([['区域', 180], ['做法', 520], ['状态']], [
      ['kit.VScroll / HScroll / XYScroll', '会话流、侧栏、设置等经 kit 封装;长行代码块用 XYScroll', st('part')],
      ['编辑器 / Diff', '库内 editor.State 自管滚动,Shift 转横向需框架支持(当前未接)', st('todo')],
      ['终端', '滚轮交给 vt 或外层列表;Shift 不特殊处理', st('part')]]))}
    ${sec('滚动条', '设置 → 外观「滚动条」:跟随系统 = MyGo 默认(悬停内容区显示);自动隐藏 = 滚动 / 拖动 / 靠近滚动条时常驻,静止满 3 秒整条隐藏(用框架的 KeepScrollbars / Scrollbars(false) 切换,不另画、不套容器,没有淡出动画);始终显示 = KeepScrollbars。动效关闭时显隐无过渡。', tbl([['行为', 200], ['说明']], [
      ['出现', '滚轮 / 触控板滚动、把 ScrollState 滚离起点'],
      ['保持', '指针在滑块轨道或拖动拇指'],
      ['隐藏', '停止滚动约 3 秒;内容区悬停不动也隐藏'],
      ['减弱动态', 'appearance.motion 或系统减弱动态 → 直接显隐']]))}</div>`;
})();
