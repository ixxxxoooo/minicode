/* 设计稿 v3.1 · 交互串联与细节页
   flows:区域模型 / 每个内容唯一的家 / 打开规则矩阵;resp:对话栏与工作区的容器响应式;
   ide-nav:编辑器能力(查找、悬停、跳转、补全、重命名、代码操作、问 Claude、行内审查…);ide-git:Git 与差异工作流清单。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, chk } = U;
  const { head, sec, spec } = P.h;
  const tbl = (cols, rows) => `<table class="ia-tbl"><thead><tr>${cols.map(([t, w]) => `<th ${w ? `style="width:${w}px"` : ''}>${t}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map((c, k) => `<td ${k === 0 ? '' : 'class="t2"'}>${k === 0 ? `<b>${c}</b>` : c}</td>`).join('')}</tr>`).join('')}</tbody></table>`;

  /* ───────── 交互串联 ───────── */
  const regAgent = `<div class="reg-win"><div class="rn">导航栏<br>会话</div><div class="rc" style="flex:1">对话栏<br>(主位,居中)</div><div class="rw" style="width:38%"><b>工作区</b>标签 + 底部面板<br>按需展开</div></div>`;
  const regIde = `<div class="reg-win"><div class="rn">导航栏<br>文件</div><div class="rw" style="flex:1"><b>工作区(主位)</b>编辑器标签 + 底部面板</div><div class="rc" style="width:26%">对话栏<br>(窄栏)</div></div>`;
  const REGION = [
    ['导航栏(左)', '会话 / 文件 / 搜索 / 源代码管理 四个分栏,同一个容器', '默认「会话」', '默认「文件」,可随时切回「会话」'],
    ['对话栏', '会话流 + 任务坞 + 输入框(同一份会话)', '主位:居中,内容宽度上限 712', '右侧窄栏 352:按自己的宽度三档自适应(见「响应式」)'],
    ['工作区', '编辑器标签组 + 底部面板(终端 / 问题 / 输出 / 引用),全窗口只有一份', '右侧停靠,默认收起;会话里点任何「详情」时自动展开', '主位:占满剩余宽度'],
    ['状态栏', '分支、同步、诊断、模型、上下文、更新', '同一条', '另有光标位置、编码、语言、语言服务状态'],
  ];
  const HOME = [
    ['文件树', '导航栏「文件」+ Agent 检查器「文件」标签', '导航栏「文件」', '会话里的文件链接:直接在工作区开标签;检查器「文件」标签<span class="del"> 删除</span>'],
    ['文件预览 / 编辑', 'Agent 检查器预览 + IDE 编辑区(两份)', '工作区编辑器组(唯一)', '导航栏点文件 / 会话里点路径 / ⌘P'],
    ['Claude 本轮改动', 'Agent 检查器「改动」+ 左栏 SCM(两处都像「改动」)', '工作区「审查」标签(Claude 改了什么:接受 / 撤销 / 评论)', '回合末的「改动卡」[审查] 与 SCM 顶部「审查本轮」只是跳转;检查器「改动」<span class="del"> 删除</span>'],
    ['仓库状态(暂存 / 提交 / 图谱)', '左栏 SCM', '导航栏「源代码管理」', '审查标签里的「在源代码管理里暂存」是跳转'],
    ['终端', 'Agent 检查器「终端」+ IDE 底部终端', '工作区底部面板(唯一)', 'Agent 形态 ⌘J 展开;会话里命令卡的「在终端面板查看」开一个只读标签;检查器「终端」<span class="del"> 删除</span>'],
    ['计划', '会话流卡 + Agent 检查器「计划」', '工作区「计划」标签(全文 + Build)', '会话流里只留一张紧凑计划卡,[查看] 是跳转;检查器「计划」<span class="del"> 删除</span>'],
    ['子 Agent 记录', '会话流卡 + Agent 检查器「任务」', '工作区「子 Agent」标签(完整过程与结果)', '子 Agent 卡 / 任务坞列表项点击进入'],
    ['后台命令 / 监听', 'Agent 检查器「任务」', '输入框上方「任务坞」(列表与状态)+ 工作区「任务输出」标签(实时输出)', '「在终端面板中打开」才进底部面板;检查器「任务」<span class="del"> 删除</span>'],
    ['会话列表', '导航栏 + 对话栏头部历史', '导航栏「会话」(完整、按项目分组、可管理)', '对话栏头部「历史」是快速跳转弹层(本项目最近 + 搜索 + 全部历史),不是第二份列表'],
    ['新会话', '导航栏「新对话」+ 对话栏', '一个命令 ⌘N', '导航栏收起 / 对话栏很窄时,头部才出现「+」'],
    ['分支', '状态栏 / SCM / 对话栏头部', '状态栏(切换分支选择器)', 'SCM 顶部、宽对话栏头部只显示;窄栏头部不显示'],
  ];
  const OPEN = [
    ['会话流 · 文件路径 / 引用', '点击', '文件标签(预览,斜体)', '右侧工作区展开并开标签', '主位编辑器开标签'],
    ['会话流 · 编辑步骤 / 差异卡头部', '点击', '差异标签(暂存区 ↔ 工作区)', '同上', '同上'],
    ['回合末改动卡 · 审查', '点击', '审查标签(本轮全部改动)', '工作区展开', '编辑器组开标签'],
    ['计划卡 · 查看 / 标题', '点击', '计划标签', '工作区展开', '编辑器组开标签;对话栏仍是窄栏'],
    ['子 Agent 卡 / 任务坞列表项', '点击', '子 Agent 标签 / 任务输出标签', '工作区展开', '编辑器组开标签'],
    ['命令卡 · 在终端面板查看', '点击图标', '底部面板里的只读终端标签', '面板展开(工作区一并展开)', '面板展开'],
    ['SCM · 文件行', '单击 / 双击', '差异标签(预览 / 固定)', '(SCM 在导航栏:切到 SCM 分栏后同左)', '编辑器组开标签'],
    ['SCM · 合并冲突文件', '点击', '三方合并标签', '同上', '同上'],
    ['状态栏 · 问题计数', '点击', '底部面板「问题」', '面板展开', '面板展开'],
    ['编辑器 · 查看引用', '⇧F12', '底部面板「引用」', '—', '面板展开'],
    ['命令面板 · 文件 / 符号', '⌘P / ⇧⌘O', '文件标签并定位', '工作区展开', '编辑器组'],
  ];
  const LIFE = [
    ['预览标签', '单击打开,标签名斜体;再点别的文件会被替换;双击标签或开始编辑即「固定」'],
    ['虚拟标签', '审查 / 计划 / 子 Agent / 任务输出 / 合并冲突 不是磁盘文件:不能直接编辑文本,内容随会话更新;关闭不丢数据,再点入口会重新打开'],
    ['同一内容只开一份', '再次点同一个入口 = 激活已有标签并闪一下工作区边框,不会重复开'],
    ['展开提示', 'Agent 形态下工作区收起时开标签,会先展开(约 440ms)再显示内容,并闪一下边框'],
    ['关掉最后一个标签', 'Agent 形态:工作区自动收起;IDE 形态:显示空编辑器概览(最近文件 / 待审查 / 快捷键)'],
    ['导航历史', '标题栏左侧 ← → 在「跳转过的位置」间前进后退(⌃- / ⌃⇧-),包括文件、行、差异块;与标签切换无关'],
  ];
  P.flows = () => `<div class="ds">${head('交互串联 · 入口矩阵', '一个窗口只有四块区域,每个内容只有一个「家」。别处出现的同一个东西都是跳转,不是第二份。Agent 与 IDE 的区别只是「对话栏」与「工作区」谁在主位;所有详情(文件、差异、审查、计划、子 Agent、任务输出、终端)都在工作区里,检查器这个概念不存在了。')}
    ${sec('区域模型', '两种形态是同一个布局的两个预设:对话栏与工作区左右互换主位,导航栏与状态栏不变。内容全窗口只有一份(会话、标签、终端、SCM)。', `<div class="grid2"><div class="reg"><b>Agent 形态</b>${regAgent}</div><div class="reg"><b>IDE 形态</b>${regIde}</div></div><div style="height:12px"></div>${tbl([['区域', 130], ['里面是什么'], ['Agent 形态'], ['IDE 形态']], REGION)}`)}
    ${sec('每个内容的家(消除重复)', '同一个东西不再在两个地方各有一份。删除了的只有一样:Agent 的「检查器」五个标签(改动 / 文件 / 终端 / 计划 / 任务)。', tbl([['内容', 120], ['原来', 230], ['现在的家', 230], ['其它入口(只是跳转)']], HOME))}
    ${sec('打开规则:从哪里点,在哪里开', '「详情」一律开在工作区。Agent 形态工作区收起时先展开;IDE 形态本来就在主位。', tbl([['来源', 170], ['动作', 70], ['打开成', 190], ['Agent 形态', 180], ['IDE 形态']], OPEN))}
    ${sec('三个面板开关', '与 VS Code 同一套心智:开关按位置命名,在两种形态里含义随布局而变,各自记忆。', `<div class="grid2">${spec('开关', ['⌘B', '⌘J', '⌥⌘B'], `<table class="ia-tbl"><tbody>${[['⌘B', '导航栏', '两种形态各自记忆'], ['⌘J', '底部面板(终端 / 问题 / 输出 / 引用)', 'Agent 默认收起、IDE 默认展开'], ['⌥⌘B', '右侧栏:Agent = 工作区,IDE = 对话栏', 'IDE 收起对话栏后工作区占满;⌘⇧M 让对话占满主区']].map(r => `<tr><td style="width:60px">${kbd(...r[0].split(''))}</td><td><b>${r[1]}</b></td><td class="t2">${r[2]}</td></tr>`).join('')}</tbody></table>`)}
      ${spec('标题栏', ['panelL', 'panelB', 'panelR'], '<div class="t2" style="line-height:1.7">左:导航栏开关 + 前进 / 后退。中:Agent / IDE 分段。右:搜索框(⌘K)、底部面板开关、右侧栏开关、通知。三个开关亮着 = 对应区域已展开。</div>')}</div>`)}
    ${sec('标签的生命周期', '工作区里所有内容都是标签,规则只有一套。', tbl([['概念', 130], ['规则']], LIFE))}
    ${sec('空态 / 加载 / 错误(要点)', '原型截图页多数只画了「有内容」;Go 版与调试入口如下,集中测试时再补设计稿画面。', tbl([['画面', 150], ['Go 行为', 520], ['调试']], [
      ['审查 · 本轮无改动', '审查标签首屏文案「本轮还没有改动」', 'MINI_ED=review:empty 或演示 -ed review:empty'],
      ['搜索 · 空 / 进行中 / 失败 / 截断', '侧栏搜索:空查询提示、转圈「搜索中…」、错误行、超出上限摘要', 'MINI_ED=search:<查询>'],
      ['问题 · 无诊断', '「没有检测到问题」或未打开文件时的引导', '打开无错误的 .md 或演示 demo.Problems'],
      ['工作区起始页 · 本轮全空', '三组列表「本轮」一行灰字(不放置灰链接)', '关掉全部虚拟标签后看起始页'],
      ['自动化 · 运行中 / 失败 / 新建', '演示 RunAutomationDemo;运行浮层流式输出;新建模态校验', '命令面板「演示:自动化」;见 pages auto-states'],
      ['用量 · 加载 / 限额失败 / 无会话', 'Ring+SkelRows;LimitsChecking/Unavailable+重试', 'MINI_PAGE=usage;演示 pages usage-states']]))}</div>`;

  /* ───────── 响应式 ───────── */
  const chatBox = (w, screen = 'work') => `<div class="rz-cell"><div class="f ai g3"><b>${w < 420 ? '窄' : w < 640 ? '中' : '宽'}</b><span class="mono t3">${w}px</span>${tag(w < 420 ? 'IDE 右侧停靠' : w < 640 ? 'Agent 默认(带工作区)' : 'Agent 默认(无工作区)', w < 420 ? 'ac' : '')}</div><div class="rz-box chat" style="width:${w}px">${W.chat({ screen })}</div></div>`;
  const wsBox = (w, label, o, crop = 0) => `<div class="rz-cell"><div class="f ai g3"><b>${label}</b><span class="mono t3">${w}px</span></div><div class="snip" style="width:${w}px;height:260px;--crop:${crop}">${WS.strip(['router.rs', 'router_test.rs', 'review'], 'router.rs')}${IDE.fileView(o)}</div></div>`;
  const TIER = [
    ['宽', '≥ 640', 'Agent 形态,工作区收起时', '内容宽度上限 712 居中;输入框完整工具栏(权限 + 模型 + 强度 + MCP + 上下文);步骤行带路径附注;任务坞按类别分胶囊;助手脚注完整指标'],
    ['中', '420 – 639', 'Agent 形态,工作区展开时(默认 560)', '内边距收到 14;欢迎页起手卡换行;其余同宽版'],
    ['窄', '< 420', 'IDE 形态右侧停靠(352)/ 窗口很窄', '头部:标题变下拉(带运行点),补「新会话」「历史」「占满」,分支隐藏;步骤行去掉附注;代码与终端输出折行;输入框:配置胶囊压缩、MCP 只留图标与个数、上下文只留环;任务坞并成「任务 N」;助手脚注只留模型 + 用时;欢迎页起手卡变成单列列表;交互卡按钮折成两行,次要按钮收进更多'],
  ];
  const WSTIER = [
    ['宽', '≥ 600', '迷你地图(语法色块 + 视口 + Git / 搜索 / 同词刻度)+ 概览标尺 + 面包屑右侧的语言 / 编码'],
    ['中', '520 – 599', '隐藏迷你地图,保留概览标尺;面包屑右侧信息收起(Agent 默认的 560 就是这一档)'],
    ['窄', '< 520', '面包屑隐藏;非当前标签只留 70px 标题;差异自动变统一视图;审查头部只留「提交」;三方合并变上下堆叠;底部面板只留终端 / 问题'],
  ];
  const WIN = [
    ['≥ 1100', '默认:Agent 工作区 560、IDE 对话栏 352'],
    ['900 – 1099', 'Agent 工作区不再并排挤对话栏:改成从右侧滑入的浮层(宽 min(560, 90%)),点外部收起;IDE 导航栏自动收起(⌘B 手动展开为浮层)'],
    ['720 – 899', 'IDE 对话栏收成右侧图标条,点开变浮层;标题栏搜索框收成图标'],
    ['< 720', '导航栏变抽屉(标题栏最左按钮滑出);Agent / IDE 分段收成单个图标按钮'],
  ];
  const PAGE = [
    ['设置', '左侧分类导航在 < 900 收成顶部下拉;正文列宽 620 → 自适应'],
    ['自动化', '< 1000:周历变成纵向日程列表;编辑抽屉改全屏弹层'],
    ['画布', '< 1000:右侧「大纲 / 版本 / 让 Claude 修改」收成底部分段抽屉'],
    ['浏览器', '< 1000:「与此页面对话」栏变底部抽屉,阅读区占满'],
    ['自定义', '< 900:左侧分类栏变顶部横向分段;列表 + 详情改成「点列表进详情,有返回」两级'],
    ['用量与成本', '< 900:图表与汇总单列堆叠;7 / 30 天分段保持在顶部'],
    ['命令面板', '< 820:预览栏隐藏;< 520:宽度改为窗口 − 24'],
  ];
  P.resp = () => `<div class="ds">${head('响应式 · 按容器而不是按窗口', '对话栏、工作区各是一个 CSS 容器,按「自己的宽度」自适应:同一段会话在 Agent 的居中主位、带工作区的中档、IDE 右侧 352 的窄栏里,是同一份 HTML、三套表现。窗口断点只负责让出次要区域。')}
    ${sec('对话栏三档', '三个框里是完全一样的内容,只有容器宽度不同(窄栏的头部多出「新会话」与「占满」,历史变成标题下拉)。', `<div class="rz-grid">${chatBox(352)}${chatBox(520)}${chatBox(712)}</div>`)}
    ${sec('档位说明', '按钮与文字让位的顺序:先附注 → 再次要指标 → 最后才折叠控件;主要操作(发送、停止、批准、审查)任何档位都在。', tbl([['档', 50], ['宽度(px)', 90], ['什么时候出现', 220], ['变化']], TIER))}
    ${sec('窄栏的新会话与欢迎页', '窄栏里的欢迎页:问候 + 输入框 + 起手卡单列。头部「+」就是 ⌘N,「⏱」是历史弹层。', `<div class="rz-grid">${chatBox(352, 'welcome')}<div class="rz-cell"><div class="f ai g3"><b>历史弹层</b><span class="t3">点标题或「⏱」</span></div><div class="rz-box chat" style="width:352px;height:300px">${W.chat({ screen: 'work' }).replace('</div>\n      <div class="flow-wrap">', '</div>\n      <div class="flow-wrap">')}${F.hist()}</div></div></div>`)}
    ${sec('工作区三档', '同样按容器宽度:Agent 默认的工作区是 560(中档),IDE 主位通常 ≥ 700(宽档)。', `<div class="rz-grid">${wsBox(720, '宽', { occ: 1, hl: 0, word: 'Handler' })}${wsBox(560, '中', { word: 'Handler' })}${wsBox(440, '窄', { word: 'Handler' })}</div>${tbl([['档', 50], ['宽度(px)', 90], ['变化']], WSTIER)}`)}
    ${sec('窗口宽度断点', '只处理「放不下」的区域让位,不改内容。变宽后恢复,自动收起不覆盖你手动设置的开合。', tbl([['窗口宽度', 110], ['行为']], WIN))}
    ${sec('整页功能的窄窗口行为', '整页屏幕替换工作台,自己决定怎么在窄窗口里折叠;规则都是「先叠、再藏、最后变抽屉」。', tbl([['页面', 110], ['行为']], PAGE))}</div>`;

  /* ───────── 编辑器能力 ───────── */
  const snip = (o, crop = 0, w = 560, h = 240) => `<div class="snip" style="width:${w}px;height:${h}px;--crop:${crop}">${IDE.fileView(o)}</div>`;
  const cell = (t, keys, body, why) => `<div><div class="cap2"><b>${t}</b>${keys ? kbd(...keys) : ''}</div>${body}<div class="why">${why}</div></div>`;
  const quick = (title, ph, rows, hint) => `<div class="snip" style="height:240px;display:grid;place-items:start center;padding-top:16px"><div class="pop" style="width:440px"><div class="inp fill" style="margin:3px;height:30px">${ic(title === '转到符号' ? 'cube' : 'search', 's12')}<span class="v">${ph}</span><span class="cursor"></span></div>
    ${rows.map(([i, a, b, on]) => `<div class="mi ${on ? 'on' : ''}" style="height:26px">${ic(i, 's12')}<span class="grow">${a}</span><span class="t3 fs-cap">${b}</span></div>`).join('')}<div class="f ai g3 t3 fs-micro" style="padding:5px 8px">${hint}</div></div></div>`;
  const crumbPick = `<div class="snip" style="height:200px"><div class="crumbs" style="margin-top:6px">src${ic('chevR', 's12')}<b>router.rs</b>${ic('chevR', 's12')}<span class="sym">${ic('cube', 's12')}impl Router</span>${ic('chevR', 's12')}<b>match_path</b></div>
    <div class="pop" style="position:absolute;left:150px;top:34px;width:200px"><div class="cap">impl Router 中的符号</div>${[['add', 0], ['match_path', 1]].map(([n, on]) => `<div class="mi ${on ? 'on' : ''}">${ic('cube', 's12')}<span class="grow mono">${n}</span><span class="t3 fs-micro">${n === 'add' ? 11 : 17}</span></div>`).join('')}</div></div>`;
  P['ide-nav'] = () => `<div class="ds" style="max-width:1240px">${head('编辑器能力 · 跳转与浮层', '对标 VS Code / Zed / Cursor 的常用能力。每一项都由语言服务器(LSP)或编辑器本身提供,弹层位置全部相对代码区;Claude 相关的动作(问、改、审)是编辑器的一等公民,不用切到对话栏。')}
    ${sec('看得见的辅助(不用触发)', '迷你地图是缩略的「语法色块」:当前视口(滑块,可拖)、Git 改动竖条(左缘)、搜索命中(黄)、同词出现(蓝框)、Claude 改过的行(紫)。右边另有概览标尺(整篇缩放):错误 / 警告 / 搜索 / Git / 光标。点任意位置跳转。', `<div class="gal">
      ${cell('迷你地图 + 概览标尺 + 同词高亮', '', snip({ mmTip: 1 }, 0, 560, 300), '光标在 <span class="mono">Handler</span> 上:所有同词浅蓝高亮,地图与标尺同步出刻度;行号旁折叠箭头悬停才出现;函数上方是 CodeLens(引用数 / 测试数 / 问 Claude),可点。')}
      ${cell('粘性滚动', '', snip({ sticky: 1, scroll: 14 }, 0), '滚进长函数后,所属的 <span class="mono">impl</span> 与 <span class="mono">fn</span> 头固定在顶部,点击跳回该行(对标 VS Code sticky scroll)。')}</div>`)}
    ${sec('查找与替换', '⌘F 是当前文件,⌘⇧F 是项目(见「搜索与替换」)。命中会同时在迷你地图与标尺上出刻度。', `<div class="gal">
      ${cell('当前文件查找', ['⌘', 'F'], snip({ ov: 'find' }, 280, 560, 220), '计数、上一个 / 下一个、区分大小写 / 全词 / 正则、在选区中查找;Esc 关闭并把光标留在当前命中。')}
      ${cell('查找替换', ['⌥', '⌘', 'F'], snip({ ov: 'find', rep: 1 }, 280, 560, 220), '替换行默认折起;「保留大小写」;全部替换前先预览数量。')}</div>`)}
    ${sec('理解代码:悬停、预览定义、引用', '', `<div class="gal">
      ${cell('悬停', '', snip({ ov: 'hover', word: 'normalize' }, 300, 560, 300), '300ms 延迟、鼠标移到卡片上不收起、光标移动 / 滚动收起;签名 + 文档 + 动作行。')}
      ${cell('预览定义', ['⌥', 'F12'], snip({ ov: 'peek', word: 'normalize' }, 300, 560, 320), '在原地内嵌一个小编辑器看定义 / 引用,不离开当前位置;⌥⏎ 在编辑器中打开。⌘点击 / F12 直接跳转。')}
      ${cell('查看引用', ['⇧', 'F12'], `<div class="snip" style="height:200px;display:flex;flex-direction:column;justify-content:flex-end">${IDE.bpanel('refs')}</div>`, '结果按文件分组,在底部面板「引用」标签;点击跳转并保持面板,方便逐个看。')}
      ${cell('面包屑 / 大纲跳转', '', crumbPick, '面包屑每一级都是下拉:同级文件 / 同级符号;左栏「大纲」是同一份符号数据的树。')}</div>`)}
    ${sec('写代码:补全、代码操作、重命名', '', `<div class="gal">
      ${cell('补全', ['⌃', 'Space'], snip({ ov: 'complete', word: '' }, 330, 560, 300), '类型图标 + 详情 + 键位提示;「✦ Claude 建议」是整行 / 多行预测(Tab 接受),与语言服务器补全并存。')}
      ${cell('代码操作(灯泡)', ['⌘', '.'], snip({ ov: 'action', word: '' }, 180, 560, 300), '行号旁灯泡;快速修复在前,「让 Claude 修复」永远在第一位;下面是重构与源代码操作。')}
      ${cell('重命名符号', ['F2'], snip({ ov: 'rename', word: 'normalize' }, 400, 560, 200), '原地输入;显示将更新的引用数;⌃⏎ 先预览再应用,跨文件一次性写入。')}
      ${cell('转到文件 / 行 / 符号', ['⌘', 'P'], quick('转到文件', 'route', [['code', 'router.rs', 'src/', 1], ['code', 'router_test.rs', 'tests/'], ['code', 'handlers/mod.rs', 'src/']], `${kbd('⏎')}打开 · 输入 <span class="mono">:42</span> 转到行 · <span class="mono">@</span> 转到符号(⇧⌘O)`), '一个入口三种用法:文件名、<span class="mono">:行号</span>、<span class="mono">@符号</span>;右侧可预览。')}</div>`)}
    ${sec('Claude 在编辑器里', '问、改、审三件事都不必切回对话栏:选中即出工具条,⌘I 行内改,改完在原地逐块接受或撤销。', `<div class="gal">
      ${cell('选中 → 问 Claude', ['⌘', 'L'], snip({ ov: 'ask', word: '' }, 280, 560, 240), '选中代码出现悬浮工具条:问(把选区作为引用加进对话)/ 编辑 / 解释 / 加到对话。')}
      ${cell('行内编辑', ['⌘', 'I'], snip({ ov: 'inline', word: '' }, 300, 560, 260), '在选区上方弹小输入框,用一句话描述修改;不进对话历史,只产出一个改动块。')}
      ${cell('行内审查', ['⌘', '⏎'], snip({ ov: 'claude', word: '' }, 290, 560, 220), 'Claude 改过的行底色偏紫、地图与标尺出紫色刻度;块上方「撤销 / 接受」;全部改动见工作区「审查」标签。')}
      ${cell('导航历史', ['⌃', '-'], `<div class="snip" style="height:150px;padding:14px 18px"><div class="f ai g3">${ib('arrowL', '', '后退')}${ib('arrow', '', '前进')}<span class="t3">跳转历史(标题栏左侧同一对按钮)</span></div><div class="fc g1" style="margin-top:10px">${[['router.rs · 17', 1], ['router.rs · 23(转到定义)', 0], ['router_test.rs · 8', 0]].map(([t, on]) => `<div class="mi ${on ? 'on' : ''}">${ic('code', 's12')}<span class="grow">${t}</span></div>`).join('')}</div></div>`, '记录的是「位置」不是标签:转到定义、搜索命中、差异块、会话里的文件链接都会入栈;⌃- / ⌃⇧- 前进后退。')}</div>`)}
    ${sec('其它常见能力(清单)', '同样要有,形态沿用 VS Code,不再单独画图。', tbl([['能力', 150], ['键位', 120], ['说明']], [['转到行 / 符号', '⌃G / ⇧⌘O', '行内输入框;符号来自语言服务器,Markdown 取标题'], ['折叠 / 展开', '⌥⌘[ / ⌥⌘]', '行号旁箭头悬停出现;折叠后行尾显示「…」徽标,点击展开'], ['括号配对 / 缩进线', '', '当前配对高亮;光标所在缩进块的引导线更亮'], ['多光标', '⌥点击 / ⌘D / ⇧⌘L', '添加下一个匹配 / 选中所有匹配'], ['移动 / 复制行', '⌥↑↓ / ⇧⌥↑↓', ''], ['行注释 / 块注释', '⌘/ / ⌥⇧A', ''], ['格式化', '⇧⌥F', '保存时格式化是设置项'], ['拆分编辑器', '⌘\\', '标签可在两组间拖动'], ['内联 Git 追溯', '', '光标行末尾淡色「你 · 2 分钟前」,悬停出提交卡(见 Git 页)'], ['自动保存 / 外部修改', '', '磁盘变了:无本地修改自动重载,有修改提示「重新载入 / 保留」']]))}</div>`;

  /* ───────── Git 与差异 ───────── */
  const menu = (items) => `<div class="pop" style="width:250px;padding:4px">${items.map(([i, t, k, c]) => i === '-' ? '<div class="sep"></div>' : `<div class="mi ${c || ''}">${i ? ic(i, 's12') : '<span style="width:12px"></span>'}<span class="grow">${t}</span>${k ? kbd(k) : ''}</div>`).join('')}</div>`;
  const box = (body, h = 240, w = 560, extra = '') => `<div class="snip" style="height:${h}px;width:${w}px;${extra}">${body}</div>`;
  const dvSnip = (o, h = 250) => box(IDE.diffView(o), h);
  const commitCard = `<div class="pop a-pop-up" style="width:300px;padding:10px 12px;display:flex;flex-direction:column;gap:6px"><div class="f ai g3"><span class="avatar">L</span><div class="grow"><b>你</b> <span class="t3 fs-cap">· 2 分钟前(未提交)</span></div></div>
    <div class="m">fix(router): 路径匹配忽略大小写与末尾斜杠</div><div class="t3 fs-cap">a3f9c21 · 本次修改影响 2 个文件</div><div class="f g2">${btn('查看提交', 'sm', 'commit')}${btn('差异', 'sm ghost', 'diff')}${btn('复制哈希', 'sm ghost', 'copy')}</div></div>`;
  const hist = `<div style="padding:8px">${[['fix(router): 路径匹配忽略大小写…', '你', '刚刚', 1], ['test: 补充 router 的大小写用例', '你', '2 小时'], ['refactor: 抽出 handlers 模块', 'Lin', '昨天'], ['chore: 升级 axum 到 0.8', '你', '周一']].map(([m, a, t, on]) => `<div class="row ${on ? 'on' : ''}" style="height:30px">${ic('commit', 's12')}<span class="grow ell">${m}</span><span class="t4 fs-micro">${a}</span><span class="aux">${t}</span></div>`).join('')}</div>`;
  P['ide-git'] = () => `<div class="ds" style="max-width:1240px">${head('Git 工作流清单', '对标 VS Code SCM + GitLens 的最佳实践。仓库状态的家是左栏「源代码管理」;Claude 改了什么的家是工作区「审查」标签;差异、冲突、历史都是工作区标签。下面每一项都是要实现的行为,不是装饰。')}
    ${sec('差异 = 暂存的最小单位', '改动块、行、词三层:块级暂存 / 还原、行级暂存(选中几行再暂存)、词级高亮(同一行里到底改了哪几个字)。', `<div class="gal">
      ${cell('块级暂存 / 还原', '', dvSnip({}, 300), '每个改动块头部有「暂存此块 · 还原此块 · 复制补丁」;右侧迷你地图标出改动位置;折叠未改动行,点击展开。')}
      ${cell('统一视图 + 行级暂存', '', dvSnip({ unified: 1, sel: 1 }, 300), '统一视图适合窄栏(工作区 &lt; 520 自动切换);选中若干行后出现「暂存所选行 / 还原所选行」,与 <span class="mono">git add -p</span> 等价。')}
      ${cell('暂存区视图', '', dvSnip({ staged: 1 }, 260), 'SCM 里点「暂存的更改」里的文件:对比的是 HEAD ↔ 暂存区,块头变成「取消暂存此块」;未暂存的是 暂存区 ↔ 工作区。')}
      ${cell('右侧可编辑', '', box(`<div class="dv-bar">${ic('diff', 's12')}<b>router.rs</b><span class="t3">暂存区 ↔ 工作区</span><span class="grow"></span>${ib('edit', 'sm on', '右侧可编辑')}${tag('可编辑', 'ac')}</div><div style="padding:14px 18px;line-height:1.8" class="t2">工作区一侧就是真文件:直接改,差异实时重算;暂存区一侧只读。这样「看差异」与「改代码」是同一个界面,不用来回开。</div>`, 150), '对标 VS Code diff editor:修改侧可编辑,只读侧加锁。')}</div>`)}
    ${sec('提交与同步', '', `<div class="gal">
      ${cell('提交菜单', '', box(`<div style="padding:12px;position:relative;width:260px">${IDE.scm({ cmenu: 1 }).split('<div class="stagger">')[0]}</div>`, 270, 300), '主按钮 = 提交;分裂箭头里:提交并推送 / 同步、修改上次提交、撤销上次提交(保留改动)。提交说明可由 Claude 生成(星标),生成期间输入框呼吸高亮。')}
      ${cell('推送被拒', '', box(`<div style="padding:18px"><div class="notice warn">${ic('alert', 'c-warn')}<div class="grow"><b>推送被拒绝</b><div class="t2">远端 <span class="mono">origin/main</span> 比本地多 2 个提交。先拉取并变基,再推送。</div><div class="f g2" style="margin-top:7px">${btn('拉取并变基', 'sm pri', 'download')}${btn('拉取并合并', 'sm')}${btn('强制推送(带租约)', 'sm ghost danger')}</div></div></div></div>`, 150), '失败的 Git 操作一律给「下一步」按钮,不只是报错;强制推送默认 <span class="mono">--force-with-lease</span> 且二次确认。')}
      ${cell('切换分支时有未提交改动', '', box(`<div style="padding:18px"><div class="pop" style="width:380px;padding:12px 14px;display:flex;flex-direction:column;gap:8px"><b>切换到 feat/auth?</b><div class="t2" style="line-height:1.55">有 3 项未提交的改动,其中 router.rs 与目标分支有冲突风险。</div><div class="f g2 wrap">${btn('储藏后切换', 'sm pri', 'layers')}${btn('带着改动切换', 'sm')}${btn('取消', 'sm ghost')}</div></div></div>`, 160), '默认推荐「储藏后切换」,切回来时提示「恢复储藏」;Claude 正在回合中时,切分支前提示它会影响正在写的文件。')}
      ${cell('储藏', '', box(`<div style="padding:8px 10px">${gh('储藏', 1)}<div class="row" style="height:36px">${ic('layers', 's12')}<div class="grow" style="min-width:0"><div class="ell">WIP on main · 路由实验</div><div class="t4 fs-micro">2 小时前 · 3 个文件</div></div><span class="acts">${ib('play', 'sm', '应用')}${ib('download', 'sm', '弹出')}${ib('trash', 'sm', '删除')}</span></div></div>`, 120, 560, 'padding:6px'), 'SCM 面板的「储藏」分组:应用(保留)/ 弹出(应用并删除)/ 删除;点开可看储藏里每个文件的差异。')}</div>`)}
    ${sec('历史、追溯与图谱', '', `<div class="gal">
      ${cell('行内追溯 + 提交卡', '', box(`<div style="padding:14px 18px"><div class="mono t3" style="line-height:20px;white-space:pre">${U.rust('    pub fn match_path(&self, path: &str) -> Option<&Handler> {')}  <i class="blame">你 · 2 分钟前 · 未提交</i></div><div style="margin:6px 0 0 130px">${commitCard}</div></div>`, 220), '光标行末尾淡色追溯,悬停出提交卡;行号左侧可开追溯栏(同提交的连续行只在首行显示作者)。')}
      ${cell('文件历史(时间线)', '', box(hist, 150), '左栏「时间线」区块:当前文件的提交,点提交 = 工作区开「该提交里这个文件」的左右对比;支持「跟随重命名」。')}
      ${cell('图谱右键菜单', '', box(`<div style="padding:10px 14px">${menu([['commit', '检出此提交'], ['branch', '从此提交创建分支…'], ['rewind', '回退到这里(重置)…'], ['-'], ['fork', '拣选(cherry-pick)'], ['undo', '还原(revert)'], ['-'], ['diff', '与工作区比较'], ['diff', '与 HEAD 比较'], ['copy', '复制哈希']])}</div>`, 270), '破坏性操作(重置、变基)都要二次确认并显示将影响的提交数;选中两个提交可「比较」。')}
      ${cell('合并冲突', '', box(`<div style="padding:16px 18px" class="fc g3"><div class="notice err">${ic('errc', 'c-err')}<div class="grow"><b>2 个文件有冲突</b><div class="t2">SCM 顶部「合并冲突」分组,点击进入三方合并标签。</div></div></div><div class="f g2">${btn('打开三方合并', 'sm pri', 'merge', 'data-act="screen" data-screen="ide-merge"')}${btn('让 Claude 解决', 'sm soft', 'spark')}</div></div>`, 150), '当前 / 传入并排、结果在下;逐处「接受当前 / 传入 / 双方」;全部解决后「标记为已解决」才取消冲突状态。')}</div>`)}
    ${sec('审查 Claude 的改动(与 Git 的分工)', '', tbl([['', 110], ['审查标签(工作区)', 250], ['源代码管理(导航栏)']], [['回答什么', 'Claude 本轮改了什么,我要不要接受', '仓库现在什么状态,我要提交什么'], ['单位', '回合 → 文件 → 改动块', '暂存区 / 工作区 → 文件 → 改动块'], ['动作', '接受 / 撤销 / 评论(发给 Claude)/ 标记已查看', '暂存 / 放弃 / 提交 / 推送'], ['关系', '「接受」不写入 Git;全部接受后一键「在源代码管理里暂存」', '提交前可再回审查标签看一眼本轮']]))}</div>`;
  const gh = (t, n) => `<div class="grp-h">${ic('chevD', 's12')}<span class="grow">${t}</span>${cnt(n)}</div>`;
})();
