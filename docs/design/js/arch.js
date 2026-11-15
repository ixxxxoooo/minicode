/* 设计稿 v3 · 架构页:信息架构(功能归位)与窗口形态(单窗口 Agent ⇄ IDE)
   回答两个问题:① 每一项设置、每一个功能现在住在哪、为什么;② 不再有「融合模式 / 独立 IDE 窗口」之后,窗口怎么在两种形态间切换。
   @author ixxxxoooo */
(function () {
  const { ic, btn, chip, tag, kbd, seg } = U;
  const { head, sec, spec, tok } = P.h;

  /* ───────── 信息架构 ───────── */
  const jump = (id, t, i, cls = '') => `<div class="it row ${cls}" data-act="settab" data-tab="${id}" data-goto="settings" tabindex="0">${ic(i, 's12')}<span class="grow">${t}</span></div>`;
  const page = (id, t, i, cls = '') => `<div class="it row ${cls}" data-act="screen" data-screen="${id}" tabindex="0">${ic(i, 's12')}<span class="grow">${t}</span></div>`;
  const tree = `<div class="ia-tree">
    <div class="ia-col"><b>客户端</b>${jump('general', '通用', 'sliders')}${jump('appearance', '外观', 'palette')}${jump('shortcuts', '快捷键', 'kbd', 'new')}</div>
    <div class="ia-col"><b>CLAUDE CODE</b>${jump('models', '模型与服务商', 'cpu', 'new')}${jump('chat', '对话与提示词', 'chat', 'new')}${jump('permissions', '权限与信任', 'shield', 'new')}${jump('runtime', '运行环境', 'cube', 'new')}</div>
    <div class="ia-col"><b>工作台</b>${jump('editor', '编辑器与语言服务', 'code', 'new')}${jump('terminal', '终端', 'terminal')}${jump('scm', '源代码管理', 'branch')}${jump('browser', '浏览器', 'globe')}</div>
    <div class="ia-col"><b>其他</b>${jump('privacy', '隐私与诊断', 'lock', 'new')}${jump('advanced', '高级', 'wrench')}${jump('about', '关于', 'info')}</div></div>
    <div class="ia-tree" style="margin-top:10px;grid-template-columns:2fr 1fr"><div class="ia-col"><b>整页功能(管理内容,不是偏好)</b><div class="f g2 wrap">${page('automation', '自动化', 'timer')}${page('canvas', '画布', 'canvas')}${page('browser', '浏览器', 'globe')}${page('customize', '自定义(7 类)', 'plug')}${page('usage', '用量与成本', 'chart', 'new')}</div></div>
    <div class="ia-col"><b>账号菜单(左下角)</b><div class="t3 fs-cap" style="line-height:1.7">设置 ⌘, · 用量与成本 · 快捷键<br>检查更新 · 关于</div></div></div>`;

  const MOVES = [
    ['融合模式(开关)', '客户端 → 窗口', '<span class="del">删除</span>', 'Agent / IDE 变成同一窗口的两种形态,开关没有意义了'],
    ['独立 IDE 窗口', '工作台', '<span class="del">删除</span>', '多开靠 ⇧⌘N 新开完整窗口(同样能切形态)'],
    ['记忆文件(CLAUDE.md)', '设置 → 记忆文件', '自定义 → 规则与记忆', '它是「内容」:要编辑、要看加载顺序,属于功能页;设置里只留一行跳转'],
    ['追加系统提示词', '设置 → 提示词', '设置 → 对话与提示词', '和回合行为、会话流显示同属「对话怎么进行」'],
    ['MCP 运行参数 / 按需加载', '设置 → MCP', 'MCP 详情面板(自定义)与输入框 MCP 弹层', '只对当前项目生效,离开 MCP 就没有上下文;全局偏好只剩工具搜索开关'],
    ['项目信任(钩子 / MCP)', '设置 → 项目信任', '设置 → 权限与信任', '信任就是一种权限,和允许 / 询问 / 拒绝规则放在一起'],
    ['服务商与模型映射', '设置 → 服务商', '设置 → 模型与服务商', '默认模型、思考强度、服务商是一件事:「用哪个模型」'],
    ['环境变量 / 网络 / claude 路径', '设置里三处分散', '设置 → 运行环境', '全是「MiniCode 怎么启动 claude」,带连通性检测与遮盖的变量表'],
    ['用量统计', '设置 → 用量', '整页「用量与成本」', '是数据看板不是偏好;入口在账号菜单与状态栏上下文弹层'],
    ['诊断 + 隐私', '两个分类', '设置 → 隐私与诊断', '都回答「我的数据去哪了 / 出问题怎么查」'],
    ['语言服务(LSP)与忽略规则', '设置 → 代码智能', '设置 → 编辑器与语言服务', '服务器装没装、忽略哪些目录,都是编辑器行为'],
    ['运行模式(真实 / 演示)', '关于页里的按钮', '设置 → 高级', '低频且需重启,不该占「关于」的位置'],
    ['设置入口', '左下齿轮直达设置', '账号菜单(齿轮 / 头像)', '菜单里还有用量、快捷键、更新;设置不再是唯一的家'],
    ['浏览器 / 终端设置', '设置里各一类', '保留,并加直达入口', '浏览器页与终端面板头部各有一个齿轮,点了直接落到那一类'],
  ];
  const moves = `<table class="ia-tbl"><thead><tr><th style="width:190px">项</th><th style="width:150px">原来</th><th style="width:210px">现在</th><th>为什么</th></tr></thead><tbody>${MOVES.map(([a, b, c, d]) => `<tr><td><b>${a}</b></td><td class="from">${b}</td><td>${c}</td><td class="t2">${d}</td></tr>`).join('')}</tbody></table>`;

  const ENTRY = [
    ['设置', '账号菜单', '⌘,', '命令面板', '侧栏齿轮'],
    ['用量与成本', '账号菜单', '状态栏「上下文」弹层', '命令面板', ''],
    ['快捷键', '账号菜单', '设置 → 快捷键', '命令面板', ''],
    ['MCP 管理', '自定义 → MCP', '输入框 MCP 弹层「管理 MCP」', '命令面板', ''],
    ['权限规则', '设置 → 权限与信任', '权限卡「始终允许」', '', ''],
    ['模型 / 思考强度', '输入框运行配置', '状态栏「Sonnet 5.5」', '设置 → 模型', ''],
    ['终端 / 浏览器偏好', '设置对应分类', '终端面板 / 浏览器页的齿轮', '', ''],
  ];
  const entry = `<table class="ia-tbl"><thead><tr><th style="width:150px">功能</th><th colspan="4">入口(第一个是「家」,其余都是跳转,不复制一份)</th></tr></thead><tbody>${ENTRY.map(r => `<tr><td><b>${r[0]}</b></td>${r.slice(1).map((c, k) => `<td>${c ? (k === 0 ? `<b>${c}</b>` : c) : ''}</td>`).join('')}</tr>`).join('')}</tbody></table>`;

  P.ia = () => `<div class="ds">${head('信息架构 · 功能归位', '旧版设置有 22 项,偏好、功能内容与数据看板混在一起,同一件事散在三处。重新归位只靠四条原则;设置页的搜索兜底:找不到时,搜一下就能直达。')}
    ${sec('四条原则', '每一次「放哪里」的争论,都用它们裁决。', `<div class="grid2">
      ${spec('一项配置只有一个家', ['设置 / 自定义'], '<div class="t2" style="line-height:1.6">别处出现的同一项一律是「跳转按钮」,不是第二份控件。记忆文件在「自定义 → 规则与记忆」,设置里只有一行「前往」。</div>')}
      ${spec('功能页管内容,设置管偏好', ['customize ≠ settings'], '<div class="t2" style="line-height:1.6">要增删改的东西(插件、MCP、Skills、命令、子 Agent、规则、钩子)在「自定义」;开关与参数(字号、代理、快捷键)才进设置。</div>')}
      ${spec('看板不是设置', ['usage'], '<div class="t2" style="line-height:1.6">用量与成本是随时间变化的数据,需要图表与范围切换,所以升为整页;设置里不放任何「只读的数」,除了隐私与诊断的本机性能。</div>')}
      ${spec('入口随处可达,位置只有一个', ['账号菜单 · 面板齿轮'], '<div class="t2" style="line-height:1.6">在哪个上下文里用到,就在哪里放一个直达入口(终端齿轮、浏览器齿轮、MCP「管理」),落点精确到分类。</div>')}</div>`)}
    ${sec('新结构:4 组 14 类', '点任意一类可直接跳到设置页对应分类。蓝底是本次内容有变动的分类。', tree)}
    ${sec('迁移对照', '每一行都对应一次「位置不对」的修正。', moves)}
    ${sec('入口矩阵', '同一个功能可以有多个入口,但「家」只有一个。', entry)}
    ${sec('服务商 · 状态与交互清单', '设置 → 模型与服务商 与运行配置弹层共用 <span class="mono">APP.st.prov</span> 原型状态。', (() => {
      const tbl = (cols, rows) => `<table class="ia-tbl"><thead><tr>${cols.map(([t, w]) => `<th ${w ? `style="width:${w}px"` : ''}>${t}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map((c, k) => `<td ${k === 0 ? '' : 'class="t2"'}">${k === 0 ? `<b>${c}</b>` : c}</td>`).join('')}</tr>`).join('')}</tbody></table>`;
      return tbl([['状态', 110], ['入口', 160], ['出口', 140], ['文案 / 触发']], [
        ['未获取且清单空', '编辑区 · 模型清单', '获取 / 手动添加', '空态「还没有模型」'],
        ['加载中', '获取模型列表', '—', '按钮转圈 + 4 行骨架'],
        ['成功', '获取模型列表', '勾选 / 映射', 'toast「获取到 N 个模型」'],
        ['失败', 'example.com 地址', '重试 / 改为手动添加', '红色 banner「HTTP 401: invalid api key」'],
        ['不支持', 'URL 含 legacy', '手动添加 / 填列表地址', '黄色 banner 无列表接口'],
        ['缺密钥', '获取模型列表', '填写 API Key', '按钮置灰,悬停「先填写 API Key」'],
        ['切换服务商', '列表「切换」/ 映射下拉 / 选模型', 'toast', '新会话生效,运行中会话不变'],
        ['删除', '行内删除 / 编辑区删除', '撤销 toast', '官方行不可删']]);
    })())}</div>`;

  /* ───────── 窗口形态 ───────── */
  const bar = (label, from, to, cls = '') => `<div class="bar"><i class="${cls}" style="left:${from / 5}%;width:${(to - from) / 5}%">${label}</i></div>`;
  const gantt = (title, rows) => `<div class="fc g3"><b>${title}</b><div class="gantt">${rows.map(([n, a, b, c, l]) => `<span class="t2">${n}</span>${bar(l || (b - a) + 'ms', a, b, c)}`).join('')}<span></span><div class="axis"><span>0</span><span>100</span><span>200</span><span>300</span><span>400</span><span>500 ms</span></div></div></div>`;
  const rows2 = (r) => `<table class="ia-tbl"><thead><tr><th style="width:130px">区域</th><th>Agent 形态</th><th>IDE 形态</th></tr></thead><tbody>${r.map(([a, b, c]) => `<tr><td><b>${a}</b></td><td>${b}</td><td>${c}</td></tr>`).join('')}</tbody></table>`;
  const mini = `<div class="forms-demo"><div class="mini-win" data-act="mini" data-mode="agent" tabindex="0"><div class="mw-t"><i></i><i></i><i></i><span class="grow"></span><span>点一下窗口:Agent ⇄ IDE</span><span class="grow"></span></div>
    <div class="mw-b"><div class="mw-s">导航栏</div><div class="mw-e">工作区 · 文件</div><div class="mw-c">对话</div><div class="mw-i">工作区</div></div></div></div>`;

  P.forms = () => `<div class="ds">${head('窗口形态 · 一个窗口两种形态', '不再有「融合模式」,也不再有独立的 IDE 窗口。一个窗口默认打开 Agent 形态,按 ⌘E(或点标题栏的分段)原地变形成 IDE,再按一次变回来。对话、终端、打开的文件都不会重载。')}
    ${sec('切换', `演示:点窗口切换。真实窗口里每一帧都是同一个布局在改宽度。${kbd('⌘', 'E')}`, mini)}
    ${sec('两种形态里各区域是什么', '形态只改变「谁占主要空间」。同一份会话、终端、编辑器标签在两个形态里都在。', rows2([
      ['导航栏', '会话列表(按项目分组)', '文件树(同一栏顶部分段可切回会话)'],
      ['工作区', '右侧 560 宽的面板(默认收起,开标签时自动展开):文件、差异、审查、计划、子 Agent、任务输出都是它的标签,底部可开终端 / 问题 / 输出', '占据主区域(同一份编辑器组与底部面板),多标签、拆分、迷你地图'],
      ['对话栏', '居中的会话流,内容宽度上限 712;有工作区时让出右侧', '挪到右侧 352 的窄栏(三档响应式:头部带会话下拉 / 新会话 / 历史 / 占满)'],
      ['标题栏', '分段高亮 Agent', '分段高亮 IDE;项目名固定在进入时的项目'],
      ['状态栏', '分支、上下文、更新', '另有光标位置、编码、语言服务状态']]))}
    ${sec('过渡编排', '同一个窗口里面板改宽度,所以变形是「连续的」。进入与离开不是简单倒放:内容在大小稳定时才出现,避免文字被挤压成一团。', `<div class="grid2">
      ${spec('进入 IDE', ['--dur-morph', '--ease-morph', '--dur-slow'], gantt('Agent → IDE · 约 440ms', [['工作区展开', 0, 440, ''], ['对话栏收窄', 0, 440, ''], ['编辑区内容淡入', 90, 350, 'b3', '延迟 90ms'], ['导航栏内容换成文件树', 0, 180, 'b2'], ['标题栏滑块', 0, 180, 'b3']]))}
      ${spec('回到 Agent', ['--dur-morph', '--dur-fast', '--ease-io'], gantt('IDE → Agent · 约 440ms', [['编辑区内容淡出', 0, 120, 'b3'], ['编辑区收回', 0, 440, ''], ['对话栏拉宽', 0, 440, ''], ["工作区回到右侧", 60, 320, 'b2', '延迟 60ms'], ['导航栏内容换回会话', 0, 180, 'b2'], ['标题栏滑块', 0, 180, 'b3']]))}</div>
      <div class="t3 fs-cap" style="margin-top:8px;line-height:1.6">变形期间窗口加 <span class="mono">.morphing</span>(约 470ms)锁定指针事件,防止半途点击;「动效」关闭或系统「减弱动态效果」时,所有时长归零,直接跳到终态。</div>`)}
    ${sec('每个形态各自记住什么', '记「开合」,不记内容。内容(文件、终端、会话)全窗口只有一份。', `<div class="grid2">
      ${spec('各形态独立记忆', ['win._mem'], '<div class="t2" style="line-height:1.8">· 导航栏开 / 合<br>· 工作区开 / 合(Agent 默认收起,开标签时自动展开)<br>· 对话栏占满(IDE)<br>· 底部面板开合与当前分段<br>· 导航栏当前分栏(IDE 默认文件,Agent 默认会话)</div>')}
      ${spec('全窗口共用', ['SessionModel', 'Workbench.editor'], '<div class="t2" style="line-height:1.8">· 会话与正在进行的回合、输入框草稿与排队消息<br>· 编辑器的全部标签、选区、滚动位置、未保存修改<br>· 终端会话(按项目根分组)<br>· 源代码管理状态与 Git 图谱<br>· 后台任务与子 Agent</div>')}</div>`)}
    ${sec('怎么触发', '所有入口走同一个函数,所以结果一致。', `<table class="ia-tbl"><tbody>${[
      ['快捷键', `${kbd('⌘', 'E')} 在 Agent 与 IDE 之间切换;设置页、整页功能里按它会先回到工作台再切形态`],
      ['标题栏分段', '点「Agent」或「IDE」;在整页里点同样先回工作台'],
      ['命令面板', '「切换到 IDE」/「切换到 Agent」(⌘K)'],
      ['会话流里的文件链接', '在工作区开标签(Agent 形态下自动展开右侧工作区;IDE = 编辑区);按住 ⌥ 点击强制在 IDE 打开'],
      ['「在 IDE 中打开」按钮', '自定义页的 Skills / 规则、改动列表、差异;点了切到 IDE 形态并打开该文件,右下角提示一句'],
      ['启动时', '设置 → 通用 →「启动时的形态」:Agent / IDE / 上次']].map(([a, b]) => `<tr><td style="width:170px"><b>${a}</b></td><td class="t2">${b}</td></tr>`).join('')}</tbody></table>`)}
    ${sec('与整页功能、设置的关系', '它们仍是替换工作区的页面,标题栏保持不变,形态分段在其中仍可点。', `<div class="grid2">
      ${spec('整页功能 / 设置', ['Page', 'Esc'], '<div class="t2" style="line-height:1.7">自动化 / 画布 / 浏览器 / 自定义 / 用量 / 设置都是「页面」,不属于任一形态。点标题栏的 Agent / IDE = 回到工作台并切到那个形态;设置按 Esc 或「返回」回到进入前的位置。</div>')}
      ${spec('再开一个窗口', ['⇧⌘N'], '<div class="t2" style="line-height:1.7">想同时看另一个项目:⇧⌘N 新开一个完整窗口(自己的 Agent / IDE 形态与记忆)。<b>不再有「独立 IDE 窗口」</b>:IDE 永远是某个窗口的一种形态。</div>')}</div>`)}
    ${sec('窄窗口', '窗口变窄时先让出次要面板,变宽后恢复(自动收起不覆盖你手动设置的开合)。', `<table class="ia-tbl"><tbody>${[['< 1100px', 'Agent 形态的工作区自动收起(开标签时仍会展开并覆盖对话);标题栏搜索框收窄'], ['< 900px', 'IDE 形态的对话栏收成图标,点开变浮层'], ['< 720px', '导航栏变抽屉,点标题栏最左的按钮滑出']].map(([a, b]) => `<tr><td style="width:120px"><span class="mono">${a}</span></td><td class="t2">${b}</td></tr>`).join('')}</tbody></table>`)}</div>`;
})();
