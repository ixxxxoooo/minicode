/* 设计稿 v3 · 工作台外壳:标题栏、图标轨、导航栏、对话栏、检查器、状态栏;三套方案共用同一份结构
   状态 st:{ scheme: a|b|c, mode: agent|ide, screen: work|welcome|palette|settings, theme }
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog, THEMES } = U;
  const W = {};
  const NAVS = [['chat', 'chat', '会话'], ['files', 'folder', '文件'], ['search', 'search', '搜索'], ['scm', 'branch', '源代码管理']];
  const PAGES = [['timer', '自动化'], ['canvas', '画布'], ['globe', '浏览器'], ['plug', '自定义']];

  /* ───────── 标题栏 ───────── */
  const modeSeg = (st) => seg([`${ic('bot', 's12')}Agent`, `${ic('code', 's12')}IDE`], st.mode === 'ide' ? 1 : 0, 'lg', 'mode');
  const cmdk = `<div class="cmdk" data-act="palette">${ic('search', 's12')}<span class="grow">搜索会话、文件与命令</span>${kbd('⌘', 'K')}</div>`;
  const crumb = `<div class="crumb">${ic('folder', 's12')}<span>router-demo</span>${ic('chevR', 's12')}<b class="ell">路由大小写匹配</b></div>`;
  const titlebar = (st) => {
    const lead = `<span class="lights"><i></i><i></i><i></i></span>${ib('panelL', '', '导航栏 ⌘B', 'data-act="side"')}`;
    const tools = `${ib('panelB', 'only-ide', '终端 ⌃`')}${ib('panelR', 'only-agent', '检查器 ⌥⌘B', 'data-act="insp"')}${ib('bell', '', '通知')}`;
    const parts = {
      a: [lead + crumb, modeSeg(st), cmdk + tools],
      b: [lead + modeSeg(st), cmdk, tools + '<span class="avatar" style="margin-left:4px">L</span>'],
      c: [lead + crumb, modeSeg(st) + cmdk, tools],
    }[st.scheme];
    return `<div class="tbar"><div class="l">${parts[0]}</div><div class="c">${parts[1]}</div><div class="r">${parts[2]}</div></div>`;
  };

  /* ───────── 图标轨(方案 C) ───────── */
  const rail = (st) => `<div class="rail">
    ${NAVS.map(([id, n, t]) => `<button class="ri tip-r ${W.navOf(st) === id ? 'on' : ''}" data-act="nav" data-nav="${id}" data-tip="${t}">${ic(n, 's16')}${id === 'scm' ? cnt(3, 'ac') : ''}</button>`).join('')}
    <span class="sep" style="width:20px;margin:6px 0"></span>
    ${PAGES.map(([n, t]) => `<button class="ri tip-r" data-tip="${t}">${ic(n, 's16')}</button>`).join('')}
    <span class="grow"></span><button class="ri tip-r" data-act="screen" data-screen="settings" data-tip="设置 ⌘,">${ic('gear', 's16')}</button><span class="avatar" style="margin-top:4px">L</span></div>`;

  /* ───────── 导航栏 ───────── */
  const sess = (t, time, on, flat) => `<div class="row ${flat ? '' : 'sess'} ${on ? 'on' : ''}"><span class="grow ell">${t}</span><span class="aux swap">${time}</span><span class="acts">${ib('more', 'sm')}</span></div>`;
  const proj = (name, n, open) => `<div class="row proj ${open ? 'open' : ''}">${ic('chevR', 's12 car')}${ic('folder', 's12 t3')}<span class="grow ell">${name}</span><span class="aux swap">${n}</span><span class="acts">${ib('plus', 'sm', '在此项目新建会话')}</span></div>`;
  const newChat = `<div class="row new-chat" style="margin:0 0 4px">${ic('plus', 'c-ac')}<span class="grow">新对话</span>${kbd('⌘', 'N')}</div>`;
  /** 会话列表:只有标题与时间,不带任何运行状态。A / C 按项目分组,B 只看当前项目并按时间分段 */
  const sessions = (scheme) => scheme === 'b'
    ? `<div class="f g2" style="padding:0 0 6px">${chip(`${ic('folder', 's12')}<b class="grow">router-demo</b>${ic('chevUD', 's12 car')}`, '', 'line grow', 'style="height:26px;flex:1"')}${ib('edit', '', '新对话 ⌘N', 'style="width:26px;height:26px"')}</div>
      <div class="stagger"><div class="grp-h">今天</div>${sess('完整修复一个 Bug:路由大小写匹配', '刚刚', 1, 1)}${sess('权限请求:修改配置与执行命令', '5 分钟', 0, 1)}${sess('文件读写:梳理并整理路由层', '1 小时', 0, 1)}
      <div class="grp-h">昨天</div>${sess('Markdown 与代码高亮:发布说明', '18:20', 0, 1)}${sess('补充集成测试并修复 CI', '11:02', 0, 1)}
      <div class="grp-h">更早</div>${sess('重构 handlers 模块的错误处理', '周一', 0, 1)}${sess('2000 轮长会话:滚动与加载', '11月2日', 0, 1)}${sess('调研:axum 与 actix 的路由差异', '10月28日', 0, 1)}</div>`
    : `${newChat}<div class="stagger">${proj('router-demo', 4, 1)}${sess('完整修复一个 Bug:路由大小写匹配', '刚刚', 1)}${sess('权限请求:修改配置与执行命令', '5 分钟')}${sess('文件读写:梳理并整理路由层', '1 小时')}${sess('Markdown 与代码高亮:发布说明', '昨天')}
      ${proj('web-dashboard', 2, 1)}${sess('内置浏览器:排查指标看板报错', '2 小时')}${sess('子 Agent:代码审查与调研', '昨天')}
      ${proj('docs-site', 6)}${proj('scenario-lab', 61)}</div>`;
  W.panel = (nav, scheme) => `<div class="a-swap">${nav === 'chat' ? sessions(scheme) : nav === 'files' ? IDE.tree() : nav === 'search' ? IDE.search() : IDE.scm()}</div>`;
  W.navOf = (st) => st.nav || (st.mode === 'ide' ? 'files' : 'chat');
  const side = (st) => {
    const nav = W.navOf(st), k = NAVS.findIndex(n => n[0] === nav);
    return `<aside class="side"><div class="side-in">
      <div class="side-top"><div class="navseg grow">${seg(NAVS.map(n => ic(n[1])), k, 'icons', 'nav')}</div>
        <span class="side-title grow">${NAVS[k][2]}</span><span class="only-c">${ib('plus', 'sm', '新建')}</span></div>
      <div class="side-body">${W.panel(nav, st.scheme)}</div>
      <div class="side-foot">${PAGES.map(([n, t]) => ib(n, 'tip-u', t)).join('')}<span class="grow"></span>${ib('gear', 'tip-u', '设置 ⌘,', 'data-act="screen" data-screen="settings"')}</div>
    </div></aside>`;
  };

  /* ───────── 对话栏 ───────── */
  const starter = (n, t, d) => `<div class="starter"><span class="c-ac">${ic(n)}</span><b>${t}</b><span class="t3 fs-cap" style="line-height:1.35">${d}</span></div>`;
  const welcome = () => `<div class="welcome"><div class="fc ai g3 a-rise"><span class="logo" style="width:36px;height:36px;border-radius:var(--r-lg)">${ic('mark', 's20')}</span><div class="fs-head b" style="margin-top:6px">今天想在 router-demo 里做点什么?</div><div class="t3">main · 3 项未提交改动 · 上次会话 5 分钟前</div></div>
    <div style="width:600px;max-width:92%" class="a-rise">${F.composer({ focus: true })}</div>
    <div class="f g4 stagger">${starter('bug', '修复一个 Bug', '描述现象,我来定位')}${starter('book', '解释这个项目', '梳理结构与关键流程')}${starter('check', '补充测试', '为最近改动补单测')}${starter('eye', '审查改动', '3 个文件 · +18 −4')}</div></div>`;
  const chat = (st) => st.screen === 'welcome'
    ? `<div class="chat-head"><b>新对话</b><span class="grow"></span>${ib('history', 'sm', '历史会话')}</div>${welcome()}`
    : `<div class="chat-head"><b class="ell">完整修复一个 Bug:路由大小写匹配</b>${chip('main', 'branch')}<span class="grow"></span>${ib('history', 'sm', '历史会话')}${ib('more', 'sm')}</div>
      <div class="flow-wrap"><div class="col"><div class="flow">${F.turn()}</div></div></div>
      <div class="chat-foot"><div class="col">${F.tdock()}<div class="dockslot"><div class="slot-cmp">${F.composer()}</div><div class="slot-card hide"></div></div></div></div>`;

  /* ───────── 检查器(Agent 右栏) ───────── */
  const TABS = [['changes', '改动', 3], ['files', '文件'], ['term', '终端'], ['plan', '计划'], ['tasks', '任务', 2]];
  const task = (n, t, d, time) => `<div class="row" style="height:auto;padding:5px 8px;align-items:flex-start"><span style="margin-top:3px"><span class="spin"></span></span><div class="grow"><div class="f ai g3"><span class="m ell" style="color:var(--tx-1)">${t}</span>${tag(n)}</div><div class="t3 fs-cap ell shimmer">${d}</div></div><span class="aux">${time}</span></div>`;
  W.inspBody = (tab) => `<div class="a-swap">${{
    changes: () => `<div class="f ai g3" style="padding:0 12px 6px"><span class="t3">3 个文件</span><span class="c-ok mono">+18</span><span class="c-err mono">−1</span><span class="grow"></span>${btn('审查', 'sm ghost', 'eye')}${btn('提交', 'sm soft', 'check')}</div>
      <div class="ch-file">${IDE.changeRow('router.rs', 'src', 'M', 4, 1, 1)}<div style="margin:3px 0 6px">${F.diff(false)}</div>${IDE.changeRow('router_test.rs', 'tests', 'M', 12, 0)}${IDE.changeRow('case_test.rs', 'tests', 'A', 2, 0)}</div>`,
    files: () => `<div style="padding:0 6px">${IDE.tree(false)}</div>`,
    term: () => `<div class="term">${IDE.termLines()}</div>`,
    plan: () => `<div style="padding:2px 12px" class="fc g4"><div class="f ai g3"><b class="grow">执行进度</b><span class="t3 num">2 / 3</span></div><div class="meter"><i style="width:66%"></i></div>${F.todo(2)}<div class="sep"></div>${F.plan(false)}</div>`,
    tasks: () => `<div style="padding:0 6px">${task('子代理', '审查 handlers 的错误处理', '搜索 "unwrap()"', '48 秒')}${task('后台命令', 'cargo watch -x test', '等待文件变化…', '6 分')}<div class="grp-h" style="margin-top:6px">已结束</div><div class="row">${ic('check', 's12 c-ok')}<span class="grow ell">调研 axum 的路由匹配</span><span class="aux">1 分 20 秒</span></div></div>`,
  }[tab]()}</div>`;
  const insp = () => `<aside class="pane insp"><div class="insp-in">
    <div class="tabs">${TABS.map(([id, t, n], k) => `<span class="tab ${k ? '' : 'on'}" data-act="insp-tab" data-tab="${id}">${t}${n ? cnt(n) : ''}</span>`).join('')}<span class="grow"></span><span class="f ai">${ib('plus', 'sm')}</span></div>
    <div class="insp-body">${W.inspBody('changes')}</div></div></aside>`;

  /* ───────── 状态栏 ───────── */
  const sbar = () => `<div class="sbar"><span class="it">${ic('branch', 's12')}main</span><span class="it">${ic('refresh', 's12')}↑1</span><span class="it">${ic('errc', 's12')}0 ${ic('alert', 's12')}1</span>
    <span class="ide-items"><span class="it">行 17,列 58</span><span class="it">UTF-8</span><span class="it">Rust</span><span class="it">${ic('okc', 's12 c-ok')}rust-analyzer</span></span>
    <span class="grow"></span><span class="it">${ic('shield', 's12')}逐项确认</span><span class="it">Sonnet 5.5</span><span class="it num">上下文 38%</span><span class="it">${ic('download', 's12 c-ac')}<span class="c-ac">更新 42%</span></span></div>`;

  /* ───────── 页面:设置 / 命令面板 ───────── */
  const navi = (n, t, on) => `<div class="row ${on ? 'on' : ''}">${ic(n, 's12')}<span>${t}</span></div>`;
  const setRow = (t, d, ctl) => `<div class="set-row"><div class="grow"><div class="m">${t}</div><div class="d">${d}</div></div>${ctl}</div>`;
  W.themeCards = (cur) => `<div class="f g3 wrap">${THEMES.map(t => `<div class="th-card ${t.id === cur ? 'on' : ''}" data-act="theme" data-theme-id="${t.id}"><div class="th-prev" data-theme="${t.id}"><i class="s"></i><div class="m"><i></i><i style="width:90%"></i><i style="width:70%"></i></div></div><div class="f ai jb" style="padding:5px 3px 1px"><b>${t.name}</b><span class="t4 fs-micro">${t.dark ? '深' : '浅'}</span></div></div>`).join('')}</div>`;
  const settings = (st) => `<div class="set"><div class="set-nav fc g1"><div class="grp-h">客户端</div>${navi('sliders', '通用')}${navi('palette', '外观', 1)}${navi('code', '编辑器')}${navi('terminal', '终端')}
      <div class="grp-h" style="margin-top:6px">CLAUDE CODE</div>${navi('cpu', '模型与服务商')}${navi('shield', '权限')}${navi('plug', 'MCP')}${navi('bolt', '钩子')}
      <div class="grp-h" style="margin-top:6px">其他</div>${navi('activity', '诊断')}${navi('info', '关于')}<span class="grow"></span>${btn('返回', 'ghost', 'chevL', 'data-act="screen" data-screen="work"')}</div>
    <div class="set-main"><div class="set-col a-rise"><div><div class="fs-head b">外观</div><div class="t3">主题、密度与动效。所有选项都落在设计令牌上,即改即生效。</div></div>
      <div class="card" style="padding:10px 12px"><div class="m" style="margin-bottom:8px">主题</div>${W.themeCards(st.theme)}</div>
      <div class="card">${setRow('跟随系统深浅色', '浅色用「宣纸」,深色用「墨夜」', tog(0))}${setRow('界面密度', '行高令牌 --h-row:紧凑 22 / 标准 24 / 宽松 26', seg(['紧凑', '标准', '宽松'], 1))}${setRow('圆角', '圆角倍率 --r-k', seg(['利落', '适中', '圆润'], 1))}</div>
      <div class="card">${setRow('动效', '流光、滑入与布局变形;关闭后所有时长令牌归零', tog(1))}${setRow('界面字号', '正文 12 · 阅读 13 · 标题 14', seg(['小', '默认', '大'], 1))}${setRow('代码字体', 'SF Mono 12 / 行高 20', chip('SF Mono', '', 'line'))}</div></div></div></div>`;
  const pi = (n, t, r, on) => `<div class="mi ${on ? 'on' : ''}" style="height:28px">${ic(n, 's12')}<span class="grow ell">${t}</span><span class="t3 fs-cap">${r}</span></div>`;
  const palette = () => `<div class="scrim" data-act="screen" data-screen="work"></div><div class="palette a-pop">
    <div class="pal-in">${ic('search', 's16 t3')}<span class="caret">主题</span></div>
    <div class="pal-body"><div class="pal-list stagger"><div class="cap">命令</div>${pi('palette', '主题:切换…', '⏎ 展开', 1)}${pi('swap', '切换到 IDE', '⌘E')}${pi('sliders', '设置:外观', '⌘,')}
      <div class="cap">会话</div>${pi('chat', '主题切换时的闪烁问题', 'router-demo')}<div class="cap">文件</div>${pi('code', 'theme.rs', 'crates/mini-app/src')}${pi('code', 'md_theme.rs', 'crates/mini-app/src')}</div>
      <div class="pal-prev"><b>主题:切换…</b><div class="t3 fs-cap" style="margin:4px 0 10px;line-height:1.5">展开主题列表。上下移动时即时预览,回车确认,Esc 还原。</div>${U.THEMES.map(t => `<div class="f ai g3 fs-cap" style="height:20px"><span class="dot" data-theme="${t.id}" style="background:var(--c-accent)"></span>${t.name}<span class="t4">${t.sub}</span></div>`).join('')}</div></div>
    <div class="pal-foot"><span class="f ai g2">${kbd('↑', '↓')}选择</span><span class="f ai g2">${kbd('⏎')}执行</span><span class="f ai g2">${kbd('esc')}关闭</span><span class="grow"></span><span>共 6 项</span></div></div>`;

  /** 整个窗口。st 见文件头 */
  W.win = (st) => `<div class="win" data-mode="${st.mode}" data-side="on" data-insp="on">${titlebar(st)}
    ${st.screen === 'settings' ? settings(st) : `<div class="body">${rail(st)}${side(st)}<div class="stage"><section class="pane ed">${IDE.editor()}</section><section class="pane chat">${chat(st)}</section>${insp()}</div></div>`}
    ${sbar()}${st.screen === 'palette' ? palette() : ''}</div>`;

  window.W = W;
})();
