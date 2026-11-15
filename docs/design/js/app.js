/* 设计稿 v3 · 查看器:页面路由、方案 / 主题切换、原型里的交互(模式变形、导航、折叠、弹层)与回合重播
   @author ixxxxoooo */
(function () {
  const { ic, btn, seg, THEMES, SCHEMES } = U;
  const $ = (s, r = document) => r.querySelector(s), $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const NAV_IDS = ['chat', 'files', 'search', 'scm'], NAV_NAMES = ['会话', '文件', '搜索', '源代码管理'];
  const PAGES = [
    ['选型', [['overview', 'grid', '三套方案总览']]],
    ['原型', [['work', 'cols', '工作台 · Agent ⇄ IDE'], ['welcome', 'spark', '新会话欢迎页'], ['palette', 'cmd', '命令面板'], ['settings', 'sliders', '设置(22 个分类)']]],
    ['整页功能', [['automation', 'timer', '自动化'], ['canvas', 'canvas', '画布'], ['browser', 'globe', '浏览器'], ['customize', 'plug', '自定义']]],
    ['IDE 细节', [['ide-search', 'search', '搜索与替换'], ['ide-scm', 'branch', '源代码管理 · 图谱'], ['ide-diff', 'diff', '差异标签'], ['ide-problems', 'alert', '问题面板']]],
    ['规范', [['flow', 'chat', '会话流与工具'], ['overlays', 'layers', '浮层与面板'], ['comps', 'cube', '组件'], ['tokens', 'palette', '设计令牌'], ['motion', 'activity', '动效']]],
  ];
  const HINTS = {
    work: ['点标题栏 Agent / IDE:布局变形', '点步骤行:展开 / 收起', '点「逐项确认 · Sonnet」:运行配置', '点导航图标:会话 / 文件 / 搜索 / 源代码管理', '右上角「重播回合」:看工具调用的全过程'],
    welcome: ['新会话没有内容时的第一屏', '起手卡悬停上浮'], palette: ['⌘K 命令面板:左列表右预览', '点遮罩关闭'], settings: ['点左侧 22 个分类逐个看', '外观 → 界面布局:点卡片就地切换三套布局', '点主题卡片即时换肤', '分段与开关都可点'],
    automation: ['本周周历 + 卡片列表 + 右侧编辑抽屉', '开关可点'], canvas: ['文档 + 右侧大纲 / 版本 / 让 Claude 修改'], browser: ['阅读模式 + 右侧「与此页面对话」'], customize: ['点左侧分类切换:插件 / MCP / Skills / 命令 / 子 Agent / 规则 / 钩子'],
    'ide-search': ['⇧⌘F:搜索 + 替换预览(红删绿增)', '悬停命中行可忽略'], 'ide-scm': ['提交框 + 暂存 / 更改 + Git 图谱'], 'ide-diff': ['SCM 单击文件 = 左右对比的预览标签', '折叠未改动的行'], 'ide-problems': ['状态栏错误 / 警告 → 底部「问题」面板'],
  };

  /** 读取上次的选择(存不了就用默认值) */
  const load = () => {
    const d = { page: 'overview', scheme: 'a', theme: 'paper', mode: 'agent', setTab: 'appearance', cust: 'mcp' };
    try { Object.assign(d, JSON.parse(localStorage.getItem('mc-design-v3b') || '{}')); } catch (e) { /* 读不到就用默认值 */ }
    // 地址栏 #page=work&scheme=b&theme=ink&mode=ide 可直达某个状态,方便分享与截图
    new URLSearchParams(location.hash.slice(1)).forEach((v, k) => { if (k in d) d[k] = v; });
    return d;
  };
  const save = () => { try { localStorage.setItem('mc-design-v3b', JSON.stringify(st)); } catch (e) { /* 隐私模式下忽略 */ } };
  const st = load();

  /* ───────── 渲染 ───────── */
  const frame = (scheme, screen, mode) => `<div class="fit"><div class="frame" data-scheme="${scheme}">${W.win({ scheme, mode, screen, theme: st.theme, setTab: st.setTab, cust: st.cust })}</div></div>`;
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
  /** 切换 Agent / IDE:改 data-mode,面板宽度由 shell.css 的过渡完成变形 */
  const setMode = (win, mode) => {
    win.dataset.mode = mode;
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
  const ACT = {
    page: (el) => go(el.dataset.page),
    scheme: (el) => { st.scheme = SCHEMES[+el.dataset.i].id; apply(); renderMain(); save(); },
    theme: (el) => { st.theme = el.dataset.themeId; apply(); save(); $$('[data-act="theme"]').forEach(b => b.classList.toggle('on', b.dataset.themeId === st.theme)); },
    pick: (el) => { st.scheme = el.dataset.id; st.page = 'work'; renderAll(); },
    mode: (el, win) => { const mode = el.dataset.i === '1' ? 'ide' : 'agent'; setMode(win, mode); if (!el.closest('.ov-row')) { st.mode = mode; save(); } },
    'ov-mode': (el) => { const w = $('.win', el.closest('.ov-row')); setMode(w, w.dataset.mode === 'ide' ? 'agent' : 'ide'); },
    nav: (el, win) => setNav(win, el.dataset.nav || NAV_IDS[+el.dataset.i]),
    side: (el, win) => { win.dataset.side = win.dataset.side === 'off' ? 'on' : 'off'; },
    insp: (el, win) => { win.dataset.insp = win.dataset.insp === 'off' ? 'on' : 'off'; },
    fold: (el) => el.classList.toggle('open'),
    tog: (el) => el.classList.toggle('on'),
    tab: (el) => $$('.tab', el.parentElement).forEach(t => t.classList.toggle('on', t === el)),
    'insp-tab': (el, win) => { ACT.tab(el); $('.insp-body', win).innerHTML = W.inspBody(el.dataset.tab); },
    cfg: (el) => { const cmp = el.closest('.cmp'), pop = $('.cfg-pop', cmp); pop ? pop.remove() : cmp.insertAdjacentHTML('afterbegin', F.cfgPop()); el.classList.toggle('on', !pop); },
    palette: (el) => { if (!el.closest('.ov-row')) go('palette'); },
    screen: (el) => { if (el.closest('.ov-row')) return; if (el.dataset.cust) st.cust = el.dataset.cust; go(el.dataset.screen); },
    settab: (el) => { st.setTab = el.dataset.tab; renderMain(); save(); },
    cust: (el) => { st.cust = el.dataset.cust; renderMain(); save(); },
    /** 设置 → 外观 → 界面布局:就地切换三套布局(同时同步顶栏的方案分段) */
    layout: (el) => { st.scheme = el.dataset.id; apply(); renderBar(); const top = $('#vmain').scrollTop; renderMain(); $('#vmain').scrollTop = top; save(); },
    replay: () => { if (st.page !== 'work') go('work'); play($('#vmain .win')); },
    'perm-ok': () => permDone && permDone(),
    demo: (el) => { $('.demo-st', el).innerHTML = P.demos[+el.dataset.i][2](); tick(); },
    tracks: () => $$('.tracks').forEach(t => { t.classList.remove('play'); void t.offsetWidth; t.classList.add('play'); }),
    mini: (el) => { el.dataset.mode = el.dataset.mode === 'ide' ? 'agent' : 'ide'; },
  };
  document.addEventListener('click', (e) => {
    const el = e.target.closest('[data-act]'); if (!el) return;
    if (el.parentElement.classList.contains('seg')) { el.parentElement.style.setProperty('--i', el.dataset.i); $$('button', el.parentElement).forEach(b => b.classList.toggle('on', b === el)); }
    const fn = ACT[el.dataset.act]; if (fn) fn(el, el.closest('.win'), e);
  });
  window.addEventListener('resize', fit);
  setInterval(tick, 1150);
  renderAll();
})();
