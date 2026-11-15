/* 设计稿 v3 · 查看器:页面路由、方案 / 主题切换、原型里的交互(模式变形、导航、折叠、弹层)与回合重播
   @author ixxxxoooo */
(function () {
  const { ic, btn, seg, THEMES, SCHEMES } = U;
  const $ = (s, r = document) => r.querySelector(s), $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const NAV_IDS = ['chat', 'files', 'search', 'scm'], NAV_NAMES = ['会话', '文件', '搜索', '源代码管理'];
  const PAGES = [
    ['选型', [['overview', 'grid', '三套方案总览']]],
    ['原型', [['work', 'cols', '工作台 · Agent ⇄ IDE'], ['welcome', 'spark', '新会话欢迎页'], ['palette', 'cmd', '命令面板'], ['settings', 'sliders', '设置(4 组 14 类)']]],
    ['整页功能', [['automation', 'timer', '自动化'], ['canvas', 'canvas', '画布'], ['browser', 'globe', '浏览器'], ['customize', 'plug', '自定义(7 类)'], ['usage', 'chart', '用量与成本']]],
    ['IDE 细节', [['ide-search', 'search', '搜索与替换'], ['ide-scm', 'branch', '源代码管理 · 图谱'], ['ide-diff', 'diff', '差异标签'], ['ide-problems', 'alert', '问题面板']]],
    ['架构', [['forms', 'swap', '窗口形态(单窗口)'], ['ia', 'list', '信息架构 · 功能归位']]],
    ['规范', [['flow', 'chat', '会话流与工具'], ['overlays', 'layers', '浮层与面板'], ['comps', 'cube', '组件'], ['states', 'hand', '交互状态'], ['tokens', 'palette', '设计令牌'], ['motion', 'activity', '动效']]],
  ];
  const HINTS = {
    work: ['⌘E 或点标题栏 Agent / IDE:同一窗口原地变形', '点步骤行:展开 / 收起', '点「逐项确认 · Sonnet」:运行配置', '点导航图标:会话 / 文件 / 搜索 / 源代码管理', '点左下齿轮:账号菜单', '右上角「重播回合」:看工具调用的全过程'],
    welcome: ['新会话没有内容时的第一屏', '起手卡悬停上浮'], palette: ['⌘K 命令面板:左列表右预览', '点遮罩或 Esc 关闭'],
    settings: ['左上搜索:输入即过滤,点结果直达并高亮', '外观 → 界面布局:点卡片就地切换三套布局', '密度 / 圆角 / 强调色 / 动效:改了立刻看到', '快捷键:点一个键开始录制,冲突当场提示', '改动过的行出现圆点,悬停可还原', '带「需重启」的项会弹出底部重启条'],
    automation: ['点卡片或周历里的计划:右侧抽屉跟着换', '立即运行:运行中 → 成功,并多一条记录', '周历可翻周'], canvas: ['正文可直接编辑,预览模式只读', '点大纲定位', '版本旁的回退按钮:确认后回到该版本'],
    browser: ['摘要 / 大纲 / 链接可切换', '点快捷提示:填进下方输入框', '右上齿轮直达 设置 → 浏览器'],
    customize: ['七类各有专属详情:插件有市场与安装进度、MCP 有授权与重连、Skills 有文件树、命令 / Agent / 钩子是表单', '筛选框实时过滤,范围可选 用户 / 项目 / 本地', '开关、新建、删除都会真的改数据'],
    usage: ['切换 7 天 / 30 天 / 本月:图表与汇总重算', '悬停柱子看当天数值', '右侧「刷新」才会查询限额'],
    'ide-search': ['⇧⌘F:搜索 + 替换预览(红删绿增)', '悬停命中行可忽略'], 'ide-scm': ['提交框 + 暂存 / 更改 + Git 图谱'], 'ide-diff': ['SCM 单击文件 = 左右对比的预览标签', '折叠未改动的行'], 'ide-problems': ['状态栏错误 / 警告 → 底部「问题」面板'],
  };

  /** 读取上次的选择(存不了就用默认值) */
  const load = () => {
    const d = { page: 'overview', scheme: 'a', theme: 'paper', mode: 'agent', setTab: 'appearance', cust: 'mcp', cuSel: 0, cuScope: 'all', cuTab: '', auSel: 1, auF: 0, wkOff: 0, useR: 1 };
    try { Object.assign(d, JSON.parse(localStorage.getItem('mc-design-v3c') || '{}')); } catch (e) { /* 读不到就用默认值 */ }
    // 地址栏 #page=work&scheme=b&theme=ink&mode=ide 可直达某个状态,方便分享与截图
    new URLSearchParams(location.hash.slice(1)).forEach((v, k) => { if (k in d) d[k] = v; });
    return d;
  };
  const save = () => { try { const { setQ, cuQ, ...keep } = st; localStorage.setItem('mc-design-v3c', JSON.stringify(keep)); } catch (e) { /* 隐私模式下忽略 */ } };
  const st = load();
  /** 交互层(behave.js)通过它拿到状态与重渲染入口;hooks 在每次整页渲染后运行 */
  const APP = window.APP = { st, hooks: [] };

  /* ───────── 渲染 ───────── */
  const frame = (scheme, screen, mode) => `<div class="fit"><div class="frame" data-scheme="${scheme}">${W.win({ scheme, mode, screen, theme: st.theme, setTab: st.setTab, setQ: st.setQ, cust: st.cust, cuSel: st.cuSel, cuScope: st.cuScope, cuTab: st.cuTab, cuQ: st.cuQ, auSel: st.auSel, auF: st.auF, wkOff: st.wkOff, useR: st.useR })}</div></div>`;
  const overview = () => `<div class="ov"><div class="ds" style="padding:0;margin:0"><h1>三套方案</h1><div class="lede">三套方案共用同一份令牌、同一批组件、同一套动效,区别只在「面板怎么分隔、导航放在哪、步骤行怎么串」。每个画框都是活的:可以直接点里面的 Agent / IDE 开关。选定后其余页面都按这一套展开。</div></div>
    ${SCHEMES.map(s => `<div class="ov-row ${s.id === st.scheme ? 'cur' : ''}" data-scheme="${s.id}">${frame(s.id, 'work', 'agent')}<div class="ov-txt">
      <div class="f ai g4"><span class="ov-k">${s.id.toUpperCase()}</span><div><div class="fs-head b">${s.name} <span class="t3" style="font-weight:400">${s.en}</span></div><div class="t3">${s.tagline}</div></div></div>
      <ul>${s.points.map(p => `<li>${p}</li>`).join('')}</ul><div class="t2" style="line-height:1.6">${s.fit}</div>
      <div class="f g3">${btn('Agent ⇄ IDE', '', 'swap', 'data-act="ov-mode"')}${btn('用这套看全部页面', 'pri', 'arrow', `data-act="pick" data-id="${s.id}"`)}</div></div></div>`).join('')}</div>`;
  const renderBar = () => { $('#vbar').innerHTML = `<span class="logo">${ic('mark')}</span><span class="ttl">MiniCode 设计稿 v3</span><span class="vsep"></span>
    <span class="lbl">方案</span>${seg(SCHEMES.map(s => `${s.id.toUpperCase()} ${s.name}`), SCHEMES.findIndex(s => s.id === st.scheme), 'lg', 'scheme')}
    <span class="lbl">主题</span><span class="f g3 ai">${THEMES.map(t => `<button class="swt tip ${t.id === st.theme ? 'on' : ''}" data-theme="${t.id}" data-act="theme" data-theme-id="${t.id}" data-tip="${t.name} · ${t.sub}"></button>`).join('')}</span>
    <span class="grow"></span>${btn('重播回合', 'soft', 'play', 'data-act="replay"')}<a href="v2/index.html">旧稿 v2 →</a>`; };
  const renderNav = () => { $('#vnav').innerHTML = PAGES.map(([g, items]) => `<div class="grp-h">${g}</div>${items.map(([id, n, t]) => `<div class="row ${id === st.page ? 'on' : ''}" data-act="page" data-page="${id}">${ic(n, 's12')}<span class="ell">${t}</span></div>`).join('')}`).join('')
    + '<div class="foot">画框 1280 × 800,按窗口等比缩放。<br>所有颜色、尺寸、动效都来自 css/tokens.css。</div>'; };
  const renderMain = () => {
    const m = $('#vmain'), p = st.page;
    m.innerHTML = p === 'overview' ? overview() : P[p] ? P[p](st) : `<div class="stage-wrap">${frame(st.scheme, p, st.mode)}<div class="hint">${(HINTS[p] || []).map(h => `<span>${h}</span>`).join('')}</div></div>`;
    m.scrollTop = 0; fit(); tick(); $$('.tracks', m).forEach(t => t.classList.add('play'));
    APP.hooks.forEach(f => f());
  };
  /** 按主区可用空间给每个画框算缩放比 */
  const fit = () => { const m = $('#vmain'), w = m.clientWidth, h = m.clientHeight;
    $$('.fit', m).forEach(f => f.style.setProperty('--s', Math.max(.3, f.closest('.ov-row') ? Math.min(.68, (w - 64 - 328) / 1280) : Math.min(1, (w - 48) / 1280, (h - 96) / 800)))); };
  const apply = () => { document.body.dataset.theme = st.theme; document.body.dataset.scheme = st.scheme; };
  const renderAll = () => { apply(); renderBar(); renderNav(); renderMain(); save(); };
  const go = (page) => { st.page = page; renderNav(); renderMain(); save(); };

  /* ───────── 原型内的状态切换(只改属性,让 CSS 过渡自己跑) ───────── */
  /** 切换导航栏内容。win:窗口元素;nav:chat / files / search / scm */
  const setNav = (win, nav) => {
    const body = $('.side-body', win), k = NAV_IDS.indexOf(nav); if (!body) return;
    body.innerHTML = W.panel(nav, win.closest('.frame').dataset.scheme);
    $$('.ri[data-nav]', win).forEach(r => r.classList.toggle('on', r.dataset.nav === nav));
    const sg = $('.navseg .seg', win); sg.style.setProperty('--i', k); $$('button', sg).forEach((b, j) => b.classList.toggle('on', j === k));
    $('.side-title', win).textContent = NAV_NAMES[k]; win.dataset.side = 'on';
  };
  /** 切换 Agent / IDE:改 data-mode,面板宽度由 shell.css / interact.css 的过渡完成变形。
      每种形态各自记住导航栏与检查器的开合(进入 IDE 时检查器收起、回到 Agent 时恢复),切换期间锁定指针事件。 */
  const setMode = (win, mode) => {
    if (win.dataset.mode === mode) return;
    win._mem = win._mem || { agent: { side: 'on', insp: 'on' }, ide: { side: 'on', insp: 'off' } };
    win._mem[win.dataset.mode] = { side: win.dataset.side, insp: win.dataset.insp };
    win.dataset.mode = mode;
    const m = win._mem[mode]; win.dataset.side = m.side; win.dataset.insp = m.insp;
    win.classList.add('morphing'); clearTimeout(win._mt); win._mt = setTimeout(() => win.classList.remove('morphing'), 470);
    $$('button[data-act="mode"]', win).forEach(b => { const on = (b.dataset.i === '1') === (mode === 'ide'); b.classList.toggle('on', on); if (on) b.parentElement.style.setProperty('--i', b.dataset.i); });
    setNav(win, mode === 'ide' ? 'files' : 'chat');
  };
  /** 推进所有「最新一步」:旧行上移淡出,新行自下滑入 */
  const tick = () => $$('.ticker[data-ticks]').forEach(t => {
    const list = t.dataset.ticks.split(';').map(x => x.split('|')), k = ((+t.dataset.k || 0) + 1) % list.length, old = t.lastElementChild;
    t.dataset.k = k; t.innerHTML = (old ? old.outerHTML.replace(/class="[^"]*"/, 'class="out"') : '') + F.tick(list[k][0], list[k][1], 'in');
  });

  /* ───────── 回合重播 ───────── */
  let playId = 0, permDone = null;
  const wait = (ms) => new Promise(r => setTimeout(r, ms));
  /** 按脚本把一个回合重新演一遍。win:工作台窗口元素 */
  async function play(win) {
    const id = ++playId, flow = $('.flow', win), wrap = $('.flow-wrap', win), slotC = $('.slot-cmp', win), slotK = $('.slot-card', win);
    const alive = () => id === playId && flow.isConnected;
    const down = () => wrap.scrollTo({ top: wrap.scrollHeight, behavior: 'smooth' });
    const add = (parent, html) => { const d = document.createElement('div'); d.className = 'it a-rise'; d.innerHTML = html; parent.appendChild(d); down(); return d; };
    flow.innerHTML = ''; let grp = null;
    for (const s of F.script) {
      if (!alive()) return;
      if (s.kind === 'step') {
        if (!grp) { grp = document.createElement('div'); grp.className = 'acts-g'; flow.appendChild(grp); }
        const el = add(grp, s.html('run')); tick(); await wait(s.run); if (!alive()) return;
        el.innerHTML = s.html(''); const op = $('.step.open', el);
        if (op) { op.classList.remove('open'); void op.offsetWidth; op.classList.add('open'); }
        setTimeout(down, 280);
      } else if (s.kind === 'perm') {
        slotK.innerHTML = F.perm(); slotK.classList.remove('hide'); slotC.classList.add('hide');
        await Promise.race([wait(s.wait + 1600), new Promise(r => { permDone = r; })]);
        slotK.classList.add('hide'); slotC.classList.remove('hide'); slotC.firstElementChild.classList.add('a-pop-up');
      } else if (s.kind === 'stream') {
        grp = null; const md = add(flow, '<div class="md"><p class="caret"></p></div>').firstElementChild, p = md.firstElementChild;
        for (let k = 0; k <= s.lead.length; k += 2) { if (!alive()) return; p.textContent = s.lead.slice(0, k); await wait(18); }
        p.textContent = s.lead; p.classList.remove('caret');
        for (const b of s.blocks) { if (!alive()) return; md.insertAdjacentHTML('beforeend', b); md.lastElementChild.classList.add('a-rise'); down(); await wait(280); }
      } else { grp = null; add(flow, s.html('')); await wait(s.wait); }
    }
  }

  /* ───────── 事件 ───────── */
  /** 在不是「工作台」的页面里点 Agent / IDE:回到工作台并切到那个形态 */
  const toWork = (mode) => { st.mode = mode; st.page = 'work'; renderNav(); renderMain(); save(); };
  const FORM_PAGES = ['work', 'welcome'];
  const ACT = {
    page: (el) => go(el.dataset.page),
    scheme: (el) => { st.scheme = SCHEMES[+el.dataset.i].id; apply(); renderMain(); save(); },
    theme: (el) => { st.theme = el.dataset.themeId; apply(); save(); $$('[data-act="theme"]').forEach(b => b.classList.toggle('on', b.dataset.themeId === st.theme)); },
    pick: (el) => { st.scheme = el.dataset.id; st.page = 'work'; renderAll(); },
    mode: (el, win) => { const mode = el.dataset.i === '1' ? 'ide' : 'agent';
      if (!el.closest('.ov-row') && !FORM_PAGES.includes(st.page) && !el.closest('.mini')) return toWork(mode);
      setMode(win, mode); if (!el.closest('.ov-row')) { st.mode = mode; save(); } },
    'ov-mode': (el) => { const w = $('.win', el.closest('.ov-row')); setMode(w, w.dataset.mode === 'ide' ? 'agent' : 'ide'); },
    nav: (el, win) => setNav(win, el.dataset.nav || NAV_IDS[+el.dataset.i]),
    side: (el, win) => { win.dataset.side = win.dataset.side === 'off' ? 'on' : 'off'; },
    insp: (el, win) => { win.dataset.insp = win.dataset.insp === 'off' ? 'on' : 'off'; },
    fold: (el) => el.classList.toggle('open'),
    tog: (el) => { el.classList.toggle('on'); el.setAttribute('aria-checked', el.classList.contains('on')); if (el.dataset.live && ACT.live) ACT.live(el); },
    tab: (el) => $$('.tab', el.parentElement).forEach(t => t.classList.toggle('on', t === el)),
    'insp-tab': (el, win) => { ACT.tab(el); $('.insp-body', win).innerHTML = W.inspBody(el.dataset.tab); },
    cfg: (el) => { const cmp = el.closest('.cmp'), pop = $('.cfg-pop', cmp); pop ? pop.remove() : cmp.insertAdjacentHTML('afterbegin', F.cfgPop()); el.classList.toggle('on', !pop); },
    palette: (el) => { if (!el.closest('.ov-row')) go('palette'); },
    screen: (el) => { if (el.closest('.ov-row')) return; if (el.dataset.cust) { st.cust = el.dataset.cust; st.cuSel = 0; st.cuQ = ''; st.cuScope = 'all'; st.cuTab = ''; } go(el.dataset.screen); },
    /** 设置分类:data-goto="settings" 时从别处跳进设置并落到该分类 */
    settab: (el) => { st.setTab = el.dataset.tab; st.setQ = ''; if (el.dataset.goto) return go(el.dataset.goto); renderMain(); save(); },
    cust: (el) => { st.cust = el.dataset.cust; st.cuSel = 0; st.cuQ = ''; st.cuScope = 'all'; st.cuTab = ''; renderMain(); save(); },
    /** 设置 → 外观 → 界面布局:就地切换三套布局(同时同步顶栏的方案分段) */
    layout: (el) => { st.scheme = el.dataset.id; apply(); renderBar(); const top = ($('.set-main') || {}).scrollTop || 0; renderMain(); const sm = $('.set-main'); if (sm) sm.scrollTop = top; save(); },
    replay: () => { if (st.page !== 'work') go('work'); play($('#vmain .win')); },
    'perm-ok': () => permDone && permDone(),
    demo: (el) => { $('.demo-st', el).innerHTML = P.demos[+el.dataset.i][2](); tick(); },
    tracks: () => $$('.tracks').forEach(t => { t.classList.remove('play'); void t.offsetWidth; t.classList.add('play'); }),
    mini: (el) => { el.dataset.mode = el.dataset.mode === 'ide' ? 'agent' : 'ide'; },
    noop: () => {},
  };
  Object.assign(APP, { ACT, go, $, $$, save, apply, toWork, setMode, setNav, rerender: () => { const m = $('#vmain'), top = m.scrollTop, sm = $('.set-main'), st0 = sm ? sm.scrollTop : 0; renderMain(); m.scrollTop = top; const s2 = $('.set-main'); if (s2) s2.scrollTop = st0; } });

  /** 全局快捷键:⌘E 切换形态、⌘, 设置、⌘K 命令面板、⌘B 导航栏、⌥⌘B 检查器、Esc 关闭浮层 / 离开设置 */
  document.addEventListener('keydown', (e) => {
    const t = e.target, typing = t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable), mod = e.metaKey || e.ctrlKey;
    if (APP.recording) return;
    const win = $('#vmain .win');
    if (mod && !e.shiftKey && !e.altKey && e.key.toLowerCase() === 'e') { e.preventDefault(); const next = (win && win.dataset.mode === 'ide') ? 'agent' : 'ide'; if (FORM_PAGES.includes(st.page) && win) { setMode(win, next); st.mode = next; save(); } else toWork(next); return; }
    if (mod && e.key === ',') { e.preventDefault(); go('settings'); return; }
    if (mod && !e.shiftKey && e.key.toLowerCase() === 'k') { e.preventDefault(); go(st.page === 'palette' ? 'work' : 'palette'); return; }
    if (mod && !e.shiftKey && !e.altKey && e.key.toLowerCase() === 'b' && win) { e.preventDefault(); ACT.side(null, win); return; }
    if (mod && e.altKey && e.key.toLowerCase() === 'b' && win) { e.preventDefault(); ACT.insp(null, win); return; }
    if (e.key === 'Escape') { if (APP.closeLayer && APP.closeLayer()) return; if (typing) { t.blur(); return; } if (st.page === 'palette' || st.page === 'settings') go('work'); }
  });
  document.addEventListener('click', (e) => {
    const el = e.target.closest('[data-act]'); if (!el) return;
    if (el.parentElement.classList.contains('seg')) { el.parentElement.style.setProperty('--i', el.dataset.i); $$('button', el.parentElement).forEach(b => b.classList.toggle('on', b === el)); }
    const fn = ACT[el.dataset.act]; if (fn) fn(el, el.closest('.win'), e);
  });
  window.addEventListener('resize', fit);
  setInterval(tick, 1150);
  renderAll();
  APP.renderBar = renderBar; APP.renderNav = renderNav; APP.renderMain = renderMain;
})();
