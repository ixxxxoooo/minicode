/* 设计稿 v3 · 工作台外壳:标题栏、图标轨、导航栏、对话栏、工作区、状态栏;三套方案共用同一份结构
   状态 st:{ scheme: a|b|c, mode: agent|ide, screen: work|welcome|palette|settings, theme }
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog, THEMES } = U;
  const W = {};
  const NAVS = [['chat', 'chat', '会话'], ['files', 'folder', '文件'], ['search', 'search', '搜索'], ['scm', 'branch', '源代码管理']];
  /** 整页功能:图标、名称、屏幕 id(与查看器页面 id 同名) */
  const PAGES = [['timer', '自动化', 'automation'], ['canvas', '画布', 'canvas'], ['globe', '浏览器', 'browser'], ['plug', '自定义', 'customize']];
  /** 整页屏幕:替换工作台的工作区 / 对话栏 */
  const PG = PAGES.map(p => p[2]).concat(['usage']);
  /** IDE 细节屏幕:固定为 IDE 模式,并带上该屏幕要展示的导航与编辑区状态 */
  /** 带固定形态 / 工作区状态的屏幕:mode 固定形态,nav 左栏分栏,ws 当前标签,其余是编辑器与面板的展示状态 */
  const IDE_SCREENS = {
    'ide-search': { nav: 'search', hl: 1, rep: 1, ws: 'router.rs' }, 'ide-scm': { nav: 'scm', graph: 1, ws: 'diff:router.rs' }, 'ide-diff': { nav: 'scm', ws: 'diff:router.rs' }, 'ide-problems': { nav: 'files', panel: 'problems', ws: 'router.rs' },
    'ide-editor': { nav: 'files', ws: 'router.rs', occ: 1 }, 'ide-review': { mode: 'agent', ws: 'review' }, 'ide-merge': { nav: 'scm', ws: 'conflict', io: 'merge' },
    'ws-plan': { mode: 'agent', ws: 'plan' }, 'ws-sub': { mode: 'ide', nav: 'files', ws: 'sub:handlers', occ: 1 }, 'ws-task': { mode: 'agent', ws: 'task:watch', panelOn: 1 },
  };
  const CRUMB = { automation: '自动化', canvas: '画布', browser: '浏览器', customize: '自定义', usage: '用量与成本' };

  /* ───────── 标题栏 ───────── */
  const modeSeg = (st) => seg([`<span data-tip="⌘E" class="f ai g2">${ic('bot', 's12')}Agent</span>`, `<span data-tip="⌘E" class="f ai g2">${ic('code', 's12')}IDE</span>`], st.mode === 'ide' ? 1 : 0, 'lg', 'mode');
  const cmdk = `<div class="cmdk" data-act="palette">${ic('search', 's12')}<span class="grow">搜索会话、文件与命令</span>${kbd('⌘', 'K')}</div>`;
  const crumb = (st) => `<div class="crumb">${ic('folder', 's12')}<span>router-demo</span>${ic('chevR', 's12')}<b class="ell">${CRUMB[st.screen] || '路由大小写匹配'}</b></div>`;
  const titlebar = (st) => {
    const lead = `<span class="lights"><i></i><i></i><i></i></span>${ib('panelL', '', '导航栏 ⌘B', 'data-act="side"')}<span class="navh">${ib('arrowL', 'dis', '后退 ⌃-')}${ib('arrow', '', '前进 ⌃⇧-')}</span>`;
    const tools = `${ib('panelB', 'on', '底部面板 ⌘J', 'data-act="panel"')}${ib('panelR', 'on', '右侧栏 ⌥⌘B(Agent = 工作区,IDE = 对话)', 'data-act="right"')}${ib('bell', '', '通知')}`;
    const parts = {
      a: [lead + crumb(st), modeSeg(st), cmdk + tools],
      b: [lead + modeSeg(st), cmdk, tools + '<button class="avatar acct-btn" style="margin-left:4px" data-act="acct" data-tip="账号与设置">L</button>'],
      c: [lead + crumb(st), modeSeg(st) + cmdk, tools],
    }[st.scheme];
    return `<div class="tbar"><div class="l">${parts[0]}</div><div class="c">${parts[1]}</div><div class="r">${parts[2]}</div></div>`;
  };

  /* ───────── 图标轨(方案 C) ───────── */
  const rail = (st) => `<div class="rail">
    ${NAVS.map(([id, n, t]) => `<button class="ri tip-r ${W.navOf(st) === id ? 'on' : ''}" data-act="nav" data-nav="${id}" data-tip="${t}">${ic(n, 's16')}${id === 'scm' ? cnt(3, 'ac') : ''}</button>`).join('')}
    <span class="sep" style="width:20px;margin:6px 0"></span>
    ${PAGES.map(([n, t, id]) => `<button class="ri tip-r ${st.screen === id ? 'on' : ''}" data-act="screen" data-screen="${id}" data-tip="${t}">${ic(n, 's16')}</button>`).join('')}
    <span class="grow"></span><button class="ri tip-r acct-btn" data-act="acct" data-tip="账号与设置">${ic('gear', 's16')}</button><button class="avatar acct-btn" style="margin-top:4px" data-act="acct">L</button></div>`;

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
  W.panel = (nav, scheme, o = {}) => `<div class="a-swap">${nav === 'chat' ? sessions(scheme) : nav === 'files' ? IDE.tree() : nav === 'search' ? IDE.search(o) : IDE.scm(o)}</div>`;
  W.navOf = (st) => st.nav || (st.mode === 'ide' ? 'files' : 'chat');
  const side = (st) => {
    const nav = W.navOf(st), k = NAVS.findIndex(n => n[0] === nav);
    return `<aside class="side"><div class="side-in">
      <div class="side-top"><div class="navseg grow">${seg(NAVS.map(n => ic(n[1])), k, 'icons', 'nav')}</div>
        <span class="side-title grow">${NAVS[k][2]}</span><span class="only-c">${ib('plus', 'sm', '新建')}</span></div>
      <div class="side-body">${W.panel(nav, st.scheme, st.io)}</div>
      <div class="side-foot">${PAGES.map(([n, t, id]) => ib(n, 'tip-u ' + (st.screen === id ? 'on' : ''), t, `data-act="screen" data-screen="${id}"`)).join('')}<span class="grow"></span>${ib('gear', 'tip-u acct-btn', '账号与设置', 'data-act="acct"')}</div>
    </div></aside>`;
  };

  /* ───────── 对话栏 ───────── */
  const starter = (n, t, d) => `<div class="starter"><span class="c-ac">${ic(n)}</span><b>${t}</b><span class="t3 fs-cap" style="line-height:1.35">${d}</span></div>`;
  const welcome = () => `<div class="welcome"><div class="fc ai g3 a-rise">${U.brand.logo(36)}<div class="fs-head b" style="margin-top:6px;text-align:center">今天想在 router-demo 里做点什么?</div><div class="t3">main · 3 项未提交改动 · 上次会话 5 分钟前</div></div>
    <div style="width:600px;max-width:92%" class="a-rise composer-slot">${F.composer({ focus: true })}</div>
    <div class="f g4 stagger starters">${starter('bug', '修复一个 Bug', '描述现象,我来定位')}${starter('book', '解释这个项目', '梳理结构与关键流程')}${starter('check', '补充测试', '为最近改动补单测')}${starter('eye', '审查改动', '3 个文件 · +18 −2')}</div></div>`;
  /** 对话栏头部。宽 / 中:标题 + 分支 + 历史 + 更多;窄(IDE 右侧停靠):标题变下拉,补上「新会话」「占满」 */
  const chatHead = (title, o = {}) => `<div class="chat-head"><div class="ttl" data-act="hist">${o.run ? '<span class="dot ac only-narrow"></span>' : ''}<b class="ell">${title}</b>${ic('chevD', 's12 t3 only-narrow')}</div>${o.branch ? chip('main', 'branch', 'not-narrow') : ''}<span class="grow"></span>
    <span class="only-narrow f ai">${ib('plus', 'sm', '新会话 ⌘N', 'data-act="newchat"')}</span>${ib('history', 'sm', '历史会话 ⌘⇧H', 'data-act="hist"')}<span class="only-narrow f ai">${ib('max', 'sm', '对话占满 ⌘⇧M', 'data-act="chatmax"')}</span>${ib('more', 'sm', '更多:重命名、分叉、归档、复制链接')}</div>`;
  /** 历史会话弹层(本项目最近的会话 + 搜索 + 全部历史)。是「跳转」不是第二份列表:家仍是左栏「会话」 */
  F.hist = () => { const r = (t, time, dot, on) => `<div class="row ${on ? 'on' : ''}">${dot ? `<span class="dot ${dot}"></span>` : '<span style="width:6px"></span>'}<span class="grow ell">${t}</span><span class="aux">${time}</span></div>`;
    return `<div class="hist-pop pop a-pop-up"><label class="inp fill">${ic('search', 's12')}<input placeholder="搜索 router-demo 的会话" spellcheck="false" style="all:unset;flex:1;min-width:0"></label>
      <div class="cap">本项目</div>${r('完整修复一个 Bug:路由大小写匹配', '刚刚', 'ac', 1)}${r('权限请求:修改配置与执行命令', '5 分钟', 'warn')}${r('文件读写:梳理并整理路由层', '1 小时')}${r('Markdown 与代码高亮:发布说明', '昨天')}
      <div class="ft">${ic('history', 's12')}<span class="grow">全部历史…</span>${kbd('⌘', '⇧', 'H')}</div><div class="ft" style="box-shadow:none;padding-top:0">${ic('plus', 's12')}<span class="grow">新会话</span>${kbd('⌘', 'N')}</div></div>`; };
  const chat = (st) => st.screen === 'welcome'
    ? `${chatHead('新对话')}${welcome()}`
    : `${chatHead('完整修复一个 Bug:路由大小写匹配', { branch: 1, run: 1 })}
      <div class="flow-wrap"><div class="col"><div class="flow">${F.turn()}</div></div></div>
      <div class="chat-foot"><div class="col">${F.tdock()}<div class="dockslot"><div class="slot-cmp">${F.composer()}</div><div class="slot-card hide"></div></div></div></div>`;
  W.chat = chat;

  /* ───────── 状态栏 ───────── */
  const sbar = () => `<div class="sbar"><span class="it">${ic('branch', 's12')}main</span><span class="it">${ic('refresh', 's12')}↑1</span><span class="it">${ic('errc', 's12')}0 ${ic('alert', 's12')}1</span>
    <span class="ide-items"><span class="it">行 17,列 58</span><span class="it">UTF-8</span><span class="it">Rust</span><span class="it">${ic('okc', 's12 c-ok')}rust-analyzer</span></span>
    <span class="grow"></span><span class="it sb-btn" data-act="sbpop" data-pop="cfg" tabindex="0">${ic('shield', 's12')}逐项确认</span><span class="it sb-btn" data-act="sbpop" data-pop="cfg" tabindex="0">Sonnet 5.5</span><span class="it num sb-btn" data-act="sbpop" data-pop="ctx" tabindex="0">上下文 38%</span><span class="it sb-btn" data-act="sbpop" data-pop="upd" tabindex="0">${ic('download', 's12 c-ac')}<span class="c-ac">更新 42%</span></span></div>`;

  /* ───────── 页面:设置 / 命令面板 ───────── */
  W.themeCards = (cur) => `<div class="f g3 wrap">${THEMES.map(t => `<div class="th-card ${t.id === cur ? 'on' : ''}" data-act="theme" data-theme-id="${t.id}"><div class="th-prev" data-theme="${t.id}"><i class="s"></i><div class="m"><i></i><i style="width:90%"></i><i style="width:70%"></i></div></div><div class="f ai jb" style="padding:5px 3px 1px"><b>${t.name}</b><span class="t4 fs-micro">${t.dark ? '深' : '浅'}</span></div></div>`).join('')}</div>`;
  const pi = (n, t, r, on) => `<div class="mi ${on ? 'on' : ''}" style="height:28px">${ic(n, 's12')}<span class="grow ell">${t}</span><span class="t3 fs-cap">${r}</span></div>`;
  const palette = () => `<div class="scrim" data-act="screen" data-screen="work"></div><div class="palette a-pop">
    <div class="pal-in">${ic('search', 's16 t3')}<span class="caret">主题</span></div>
    <div class="pal-body"><div class="pal-list stagger"><div class="cap">命令</div>${pi('palette', '主题:切换…', '⏎ 展开', 1)}${pi('swap', '切换到 IDE', '⌘E')}${pi('sliders', '设置:外观', '⌘,')}
      <div class="cap">会话</div>${pi('chat', '主题切换时的闪烁问题', 'router-demo')}<div class="cap">文件</div>${pi('code', 'theme.rs', 'crates/mini-app/src')}${pi('code', 'md_theme.rs', 'crates/mini-app/src')}</div>
      <div class="pal-prev"><b>主题:切换…</b><div class="t3 fs-cap" style="margin:4px 0 10px;line-height:1.5">展开主题列表。上下移动时即时预览,回车确认,Esc 还原。</div>${U.THEMES.map(t => `<div class="f ai g3 fs-cap" style="height:20px"><span class="dot" data-theme="${t.id}" style="background:var(--c-accent)"></span>${t.name}<span class="t4">${t.sub}</span></div>`).join('')}</div></div>
    <div class="pal-foot"><span class="f ai g2">${kbd('↑', '↓')}选择</span><span class="f ai g2">${kbd('⏎')}执行</span><span class="f ai g2">${kbd('esc')}关闭</span><span class="grow"></span><span>共 6 项</span></div></div>`;

  /** 整个窗口。st 见文件头;screen 可取 work / welcome / palette / settings / 整页 / IDE 细节与工作区标签屏幕。
      区域模型:导航栏 + 对话栏 + 工作区 + 状态栏。data-right 是「右侧那一栏」的开合(Agent = 工作区,IDE = 对话栏),data-panel 是工作区底部面板的开合。 */
  W.win = (st) => {
    const io = IDE_SCREENS[st.screen], pg = PG.includes(st.screen);
    st = { ...st, io, mode: io ? (io.mode || 'ide') : pg ? 'agent' : st.mode, nav: st.nav || (io && io.nav) };
    const mode = st.mode, wsId = (io && io.ws) || st.ws || (mode === 'ide' ? 'router.rs' : 'review');
    const tabs = WS.TAB[wsId] && !WS.DEFAULT.includes(wsId) ? WS.DEFAULT.concat([wsId]) : WS.DEFAULT;
    const panelOn = (io && io.panelOn) || st.panel === 'on' || (mode === 'ide' && st.panel !== 'off');
    const o = { tabs, active: wsId, panel: (io && io.panel) || st.panelKind, hl: io && io.hl, rep: io && io.rep, ov: st.ov, unified: st.unified, staged: st.staged, sel: st.sel, scroll: +st.scroll || 0, sticky: st.sticky, cmenu: st.cmenu };
    const stage = pg ? `<section class="pane page">${W.pages[st.screen](st)}</section>`
      : `<section class="pane chat">${chat(st)}</section><section class="pane ws">${WS.html(o)}</section>`;
    return `<div class="win" data-mode="${mode}" data-side="on" data-right="${st.right === 'off' ? 'off' : 'on'}" data-panel="${panelOn ? 'on' : 'off'}">${titlebar(st)}
    ${st.screen === 'settings' ? W.settings(st) : `<div class="body">${rail(st)}${side(st)}<div class="stage">${stage}</div></div>`}
    ${sbar()}${st.screen === 'palette' ? palette() : ''}</div>`;
  };

  /* ───────── 账号菜单(设置 / 用量 / 快捷键 / 更新 / 关于的统一入口) ───────── */
  const ami = (n, t, r, attr) => `<div class="mi" tabindex="0" ${attr}>${ic(n, 's12')}<span class="grow">${t}</span>${r ? `<span class="t3 fs-cap">${r}</span>` : ''}</div>`;
  W.acctMenu = () => `<div class="acct pop a-pop-up"><div class="who"><span class="avatar">L</span><div class="grow"><div class="m">demo</div><div class="t3 fs-cap">Claude Max · 官方账号</div></div></div>
    <div class="quota"><div class="q"><span style="width:62px" class="t3">本月费用</span><b class="num grow">$12.40</b></div><div class="q"><span style="width:62px" class="t3">5 小时</span><div class="meter"><i style="width:38%"></i></div><span class="num t3">38%</span></div></div>
    <div class="menu">${ami('sliders', '设置', kbd('⌘', ','), 'data-act="screen" data-screen="settings"')}${ami('chart', '用量与成本', '', 'data-act="screen" data-screen="usage"')}${ami('kbd', '快捷键', '', 'data-act="settab" data-tab="shortcuts" data-goto="settings"')}<div class="sep"></div>${ami('refresh', '检查更新', '', 'data-act="settab" data-tab="about" data-goto="settings"')}${ami('info', '关于 MiniCode', '0.2.0', 'data-act="settab" data-tab="about" data-goto="settings"')}</div></div>`;
  /** 状态栏弹层:ctx 上下文 / upd 更新 / cfg 运行配置 */
  W.sbPop = (kind) => ({
    ctx: `<div class="sb-pop pop a-pop-up"><div class="f ai g4"><span class="ring big" style="--p:38"></span><div class="grow"><b class="num">76k / 200k</b> <span class="t3">· 38%</span><div class="t3 fs-cap">Sonnet 5.5 · 自动压缩阈值 85%</div></div>${btn('压缩', 'sm')}</div>
      <div class="ctxbar"><i style="width:14%;background:var(--info)"></i><i style="width:11%;background:var(--ok)"></i><i style="width:9%;background:var(--warn)"></i><i style="width:4%;background:var(--c-vio)"></i></div>
      <div class="f ai fs-cap t3"><span class="grow">5 小时限额 38%</span>${btn('用量与成本', 'sm ghost', 'arrow', 'data-act="screen" data-screen="usage"')}</div></div>`,
    upd: `<div class="sb-pop pop a-pop-up"><div class="f ai g3"><b class="grow">发现新版本 0.2.1</b>${tag('42%', 'ac')}</div><div class="meter"><i style="width:42%"></i></div><div class="t3 fs-cap">正在后台下载,完成后提示重启。</div></div>`,
    cfg: `<div class="sb-pop pop a-pop-up" style="padding:4px;gap:0"><div class="cap">权限模式</div><div class="mi on">${ic('shield', 's12')}<span class="grow">逐项确认</span>${ic('check', 's12')}</div><div class="mi">${ic('edit', 's12')}<span class="grow">自动接受编辑</span></div><div class="mi">${ic('list', 's12')}<span class="grow">计划模式</span></div><div class="sep"></div><div class="cap">模型</div><div class="mi on"><span class="grow">Sonnet 5.5</span>${ic('check', 's12')}</div><div class="mi"><span class="grow">Opus 5.5</span></div><div class="mi"><span class="grow">Haiku 5.5</span></div></div>`,
  })[kind];

  window.W = W;
})();
