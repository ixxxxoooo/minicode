/* 设计稿查看器:导航、方案切换、主题 / 强调色、并排对比、全屏、决策清单 */
(function () {
  const ORDER = ['overview', 'agent', 'ide', 'settings', 'pages', 'global'];
  const GROUP_TITLE = { overview: '总览与规范', agent: 'Agent 会话', ide: 'IDE 模式', settings: '设置', pages: '其他页面', global: '全局浮层' };
  const load = (k, d) => { try { return JSON.parse(localStorage.getItem(k)) || d; } catch (e) { return d; } };
  const st = Object.assign({ screen: 'home', variant: 'A', theme: 'light', accent: 'orange', compare: false, zoom: 'fit' }, load('mc-design-state', {}));
  const choices = load('mc-design-choices', {});
  const save = () => { localStorage.setItem('mc-design-state', JSON.stringify(st)); localStorage.setItem('mc-design-choices', JSON.stringify(choices)); };
  const $ = (s, r = document) => r.querySelector(s);
  const app = $('#app');
  const ACCENTS = { orange: '#E8541A', blue: '#3B6FD8', green: '#2F9E5B', violet: '#7A5AE0' };

  const screensIn = (g) => S.list.filter(s => s.group === g);
  const totalVariants = S.list.reduce((a, s) => a + s.variants.length, 0);

  /* ───── 渲染一个 frame ───── */
  const frame = (v, extra = '') => {
    let html;
    try { html = v.render(); } catch (e) { html = `<div style="padding:30px;color:#c00;font:13px monospace">渲染出错:${K.esc(String(e && e.stack || e))}</div>`; console.error(e); }
    return `<div class="frame" data-theme="${st.theme}" data-accent="${st.accent}" ${extra}>${html}</div>`;
  };
  const stageWidth = () => Math.max(600, (document.querySelector('main') || document.body).clientWidth - 56);
  const scaleNow = () => st.zoom === 'fit' ? Math.min(1, stageWidth() / 1280) : 1;
  const fbox = (v) => { const s = scaleNow(); return `<div class="fbox" style="width:${1280 * s}px;height:${800 * s}px"><div class="scaler" style="transform:scale(${s})">${frame(v)}</div></div>`; };

  /* ───── 页头与导航 ───── */
  const renderChrome = () => {
    const nav = ORDER.map(g => `<div class="g">${GROUP_TITLE[g]}</div>${g === 'overview' ? `<a data-go="home" class="${st.screen === 'home' ? 'on' : ''}"><span class="t">总览墙</span></a><a data-go="decide" class="${st.screen === 'decide' ? 'on' : ''}"><span class="t">问题清单与方案选择</span><span class="n">${Object.keys(choices).length}</span></a>` : ''}${screensIn(g).map(s => `<a data-go="${s.id}" class="${st.screen === s.id ? 'on' : ''}"><span class="t">${s.title.replace(/\s*\(.*\)$/, '')}</span>${s.variants.length > 1 ? `<span class="n">${s.variants.length}</span>` : ''}</a>`).join('')}`).join('');
    app.innerHTML = `
      <header>
        <div class="brand"><i>&lt;/&gt;</i>MiniCode <small>UI 设计稿 v2 · 紧凑版</small></div>
        <span class="sp"></span>
        <span class="vseg" data-k="theme">${[['light', '浅色'], ['dark', '深色']].map(([k, t]) => `<button data-v="${k}" class="${st.theme === k ? 'on' : ''}">${t}</button>`).join('')}</span>
        <span class="swatches" data-k="accent">${Object.entries(ACCENTS).map(([k, c]) => `<button data-v="${k}" title="${k}" style="background:${c};color:${c}" class="${st.accent === k ? 'on' : ''}"></button>`).join('')}</span>
        <span class="vseg" data-k="zoom">${[['fit', '适应'], ['100', '100%']].map(([k, t]) => `<button data-v="${k}" class="${st.zoom === k ? 'on' : ''}">${t}</button>`).join('')}</span>
        <button class="vbtn ${st.compare ? 'on' : ''}" id="b-cmp">并排对比</button>
        <button class="vbtn" id="b-fs">全屏 <kbd>F</kbd></button>
      </header>
      <nav>${nav}</nav>
      <main id="main"></main>`;
    document.body.dataset.theme = st.theme;
  };

  /* ───── 总览墙 ───── */
  const pickV = (s) => s.variants.find(v => v.rec) || s.variants[0];
  const renderHome = () => {
    const wall = ORDER.map(g => { const ss = screensIn(g); if (!ss.length) return ''; return `<div class="wall-g">${GROUP_TITLE[g]}</div><div class="wall">${ss.map(s => { const v = pickV(s); return `<div class="thumb" data-go="${s.id}"><div class="tb2"><div class="sc">${frame(v)}</div></div><div class="cap2">${s.title.replace(/\s*\(.*\)$/, '')}<small>${s.variants.length > 1 ? s.variants.length + ' 套方案' : '1 套'}</small></div></div>`; }).join('')}</div>`; }).join('');
    return `<div class="hero"><h1>MiniCode 整体 UI 重设计</h1>
      <div class="desc" style="margin:4px 0 0">基于现有界面逐页复盘后重做:保留「Claude Code 工作台」的紧凑气质与全部功能,统一视觉语言(暖灰中性 + 橙色强调、平面分层、12px 为主)、整理信息架构,并对有争议的交互给出多套方案供你选择。</div>
      <div class="stats"><div><b>${S.list.length}</b><span>个页面 / 板块</span></div><div><b>${totalVariants}</b><span>套设计方案</span></div><div><b>2</b><span>种主题 × 4 种强调色</span></div><div><b>${window.PROBLEMS.length}</b><span>处现状问题已梳理</span></div></div>
      <div style="margin-top:14px;color:var(--v-tx2)">怎么看:左侧选页面 → 顶部切换方案 A / B / C(带「推荐」标记的是我的建议) → 右上角切浅 / 深色与强调色;<kbd>←</kbd><kbd>→</kbd> 切方案、<kbd>↑</kbd><kbd>↓</kbd> 切页面、<kbd>T</kbd> 主题、<kbd>C</kbd> 并排对比、<kbd>F</kbd> 全屏。看完去「问题清单与方案选择」把你的选择点一下,我再按它落地。</div></div>${wall}`;
  };

  /* ───── 单个页面 ───── */
  const renderScreen = (s) => {
    if (!s.variants.some(v => v.key === st.variant)) st.variant = s.variants[0].key;
    const cur = s.variants.find(v => v.key === st.variant);
    const tabs = s.variants.length > 1 || s.variants[0].tag ? `<div class="vtabs">${s.variants.map(v => `<div class="vtab ${v.key === st.variant && !st.compare ? 'on' : ''}" data-variant="${v.key}"><b>${v.key}</b>${v.label}${v.rec ? '<span class="tag rec">推荐</span>' : v.tag ? `<span class="tag">${v.tag}</span>` : ''}</div>`).join('')}</div>` : '';
    const head = `<h1>${s.title}</h1><p class="desc">${s.desc}</p>${tabs}`;
    if (st.compare) {
      return head + `<div class="cmpwrap">${s.variants.map(v => `<div><div class="cmphead"><span class="vtab on" style="cursor:default"><b>${v.key}</b>${v.label}</span>${v.rec ? '<span class="tag rec">推荐</span>' : ''}<small>${v.note || ''}</small></div>${fbox(v)}</div>`).join('')}</div>${notes(s, null)}`;
    }
    return head + fbox(cur) + notes(s, cur);
  };
  const notes = (s, cur) => {
    const chosen = choices[s.id];
    return `<div class="below"><div class="vcard"><h3>设计说明</h3><ul>${s.notes.map(n => `<li>${n}</li>`).join('')}</ul></div>
      <div class="vcard">${cur ? `<h3>方案 ${cur.key} · ${cur.label}</h3><p style="margin:0 0 10px;color:var(--v-tx2)">${cur.note || '该页只有一套设计。'}</p>` : '<h3>选择</h3>'}
        ${s.variants.length > 1 ? `<div style="display:flex;gap:6px;flex-wrap:wrap">${s.variants.map(v => `<button class="vbtn ${chosen === v.key ? 'pri' : ''}" data-choose="${s.id}:${v.key}">${chosen === v.key ? '✓ ' : ''}选方案 ${v.key}</button>`).join('')}</div><div style="margin-top:8px;color:var(--v-tx3);font-size:12px">${chosen ? `你当前选的是方案 ${chosen}。` : '还没选择,可以在这里或「问题清单」页里选。'}</div>` : ''}</div></div>`;
  };

  /* ───── 问题清单与方案选择 ───── */
  const renderDecide = () => {
    const probs = `<table class="ptable"><tr><th>现状问题</th><th>具体表现</th><th>改进方向</th><th>相关页面</th></tr>${window.PROBLEMS.map(([a, b, c, d]) => `<tr><td>${a}</td><td>${b}</td><td>${c}</td><td>${d.split(' / ').map(x => `<a data-go="${x}">${x}</a>`).join(' · ')}</td></tr>`).join('')}</table>`;
    const decs = S.list.filter(s => s.variants.length > 1).map(s => `<div class="dec"><h3>${s.title.replace(/\s*\(.*\)$/, '')}${choices[s.id] ? '<span class="tag rec">已选 ' + choices[s.id] + '</span>' : ''}</h3><div class="dd">${s.desc}</div><div class="opts">${s.variants.map(v => `<div class="opt ${choices[s.id] === v.key ? 'on' : ''}" data-choose="${s.id}:${v.key}"><div class="oh"><i>${v.key}</i>${v.label}${v.rec ? '<span class="tag rec">推荐</span>' : ''}</div><p>${v.note || ''}</p><a data-open="${s.id}:${v.key}">查看设计 →</a></div>`).join('')}</div></div>`).join('');
    return `<h1>问题清单与方案选择</h1><p class="desc">先看我从现有界面里梳理出的问题和改进方向;再在下面逐项挑选你喜欢的方案(点卡片即选中,可随时改)。选完点「复制我的选择」发给我即可。</p>
      <div style="display:flex;gap:8px;margin-bottom:16px"><button class="vbtn pri" id="b-rec">全部按推荐选择</button><button class="vbtn" id="b-copy">复制我的选择</button><button class="vbtn" id="b-clear">清空选择</button></div>
      <div class="h2" style="margin-top:0">一、现状问题与改进方向</div>${probs}<div class="h2">二、需要你拍板的方案(${S.list.filter(s => s.variants.length > 1).length} 项)</div>${decs}`;
  };

  /* ───── 总渲染 ───── */
  const render = () => {
    renderChrome();
    const main = $('#main');
    if (st.screen === 'home') main.innerHTML = renderHome();
    else if (st.screen === 'decide') main.innerHTML = renderDecide();
    else { const s = S.find(st.screen); if (!s) { st.screen = 'home'; return render(); } main.innerHTML = renderScreen(s); }
    history.replaceState(null, '', `#${st.screen}${S.find(st.screen) ? '/' + st.variant : ''}`);
    save();
  };
  const go = (id, variant) => { st.screen = id; if (variant) st.variant = variant; render(); $('#main').scrollTop = 0; };
  const toast = (t) => { const d = document.createElement('div'); d.className = 'toast'; d.textContent = t; document.body.appendChild(d); setTimeout(() => d.remove(), 1800); };

  /* ───── 全屏 ───── */
  let fsEl = null;
  const closeFs = () => { if (fsEl) { fsEl.remove(); fsEl = null; } };
  const openFs = () => {
    const s = S.find(st.screen); if (!s) return;
    const draw = () => {
      const cur = s.variants.find(v => v.key === st.variant);
      const sc = Math.min(innerWidth / 1280, (innerHeight - 52) / 800);
      fsEl.innerHTML = `<div class="bar"><b>${s.title}</b><div class="vtabs" style="margin:0">${s.variants.map(v => `<div class="vtab ${v.key === st.variant ? 'on' : ''}" data-fsv="${v.key}"><b>${v.key}</b>${v.label}</div>`).join('')}</div><span class="sp"></span><button class="vbtn" data-fsc>关闭 Esc</button></div><div class="stage"><div style="width:${1280 * sc}px;height:${800 * sc}px"><div class="scaler" style="transform:scale(${sc})">${frame(cur)}</div></div></div>`;
    };
    fsEl = document.createElement('div'); fsEl.className = 'fs'; document.body.appendChild(fsEl); draw();
    fsEl.addEventListener('click', (e) => { const v = e.target.closest('[data-fsv]'); if (v) { st.variant = v.dataset.fsv; draw(); } if (e.target.closest('[data-fsc]')) closeFs(); });
    fsEl.draw = draw;
  };

  /* ───── 事件 ───── */
  document.addEventListener('click', (e) => {
    const t = e.target;
    const go_ = t.closest('[data-go]'); if (go_) { e.preventDefault(); const id = go_.dataset.go; if (id === 'home' || id === 'decide' || S.find(id)) { go(id); return; } }
    const op = t.closest('[data-open]'); if (op) { const [id, v] = op.dataset.open.split(':'); e.stopPropagation(); go(id, v); return; }
    const ch = t.closest('[data-choose]'); if (ch) { const [id, v] = ch.dataset.choose.split(':'); choices[id] = v; save(); const sc = $('#main').scrollTop; render(); $('#main').scrollTop = sc; return; }
    const vt = t.closest('[data-variant]'); if (vt) { st.variant = vt.dataset.variant; st.compare = false; render(); return; }
    const sg = t.closest('.vseg button, .swatches button'); if (sg) { const k = sg.parentElement.dataset.k; st[k] = sg.dataset.v; const sc = $('#main').scrollTop; render(); $('#main').scrollTop = sc; return; }
    if (t.closest('#b-cmp')) { st.compare = !st.compare; render(); return; }
    if (t.closest('#b-fs')) { openFs(); return; }
    if (t.closest('#b-rec')) { S.list.filter(s => s.variants.length > 1).forEach(s => { const v = s.variants.find(v => v.rec); if (v) choices[s.id] = v.key; }); render(); toast('已按推荐方案选择'); return; }
    if (t.closest('#b-clear')) { Object.keys(choices).forEach(k => delete choices[k]); render(); return; }
    if (t.closest('#b-copy')) {
      const lines = S.list.filter(s => s.variants.length > 1).map(s => { const k = choices[s.id]; const v = s.variants.find(v => v.key === k); return `- ${s.title.replace(/\s*\(.*\)$/, '')}:${v ? `方案 ${v.key}「${v.label}」` : '(未选,按你的推荐)'}`; });
      const text = `我对 MiniCode UI 设计稿的选择:\n${lines.join('\n')}`;
      (navigator.clipboard ? navigator.clipboard.writeText(text) : Promise.reject()).then(() => toast('已复制到剪贴板'), () => { prompt('复制下面的文字:', text); });
      return;
    }
  });
  document.addEventListener('keydown', (e) => {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    const s = S.find(st.screen);
    if (e.key === 'Escape') return closeFs();
    if (e.key === 't' || e.key === 'T') { st.theme = st.theme === 'light' ? 'dark' : 'light'; render(); if (fsEl) fsEl.draw(); return; }
    if (e.key === 'f' || e.key === 'F') return fsEl ? closeFs() : openFs();
    if (e.key === 'c' || e.key === 'C') { st.compare = !st.compare; return render(); }
    if (s && (e.key === 'ArrowRight' || e.key === 'ArrowLeft')) {
      const ix = s.variants.findIndex(v => v.key === st.variant); const n = (ix + (e.key === 'ArrowRight' ? 1 : -1) + s.variants.length) % s.variants.length;
      st.variant = s.variants[n].key; st.compare = false; render(); if (fsEl) fsEl.draw(); return;
    }
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      const ids = ['home', 'decide', ...ORDER.flatMap(g => screensIn(g).map(x => x.id))]; const ix = ids.indexOf(st.screen);
      const n = Math.min(ids.length - 1, Math.max(0, ix + (e.key === 'ArrowDown' ? 1 : -1))); e.preventDefault(); go(ids[n]);
    }
  });
  addEventListener('resize', () => { if (new URLSearchParams(location.search).get('shot')) return; if (st.zoom === 'fit' && !fsEl) { const sc = $('#main') && $('#main').scrollTop; render(); if (sc) $('#main').scrollTop = sc; } if (fsEl) fsEl.draw(); });

  // 截图模式:?shot=页面:方案[:dark][:accent] 只渲染一个 1280×800 的画面(用于校对与出图)
  const q = new URLSearchParams(location.search).get('shot');
  if (q) {
    const [id, key, th, ac] = q.split(':');
    const s = S.find(id); const v = s && (s.variants.find(v => v.key === (key || 'A')) || s.variants[0]);
    if (th) st.theme = th; if (ac) st.accent = ac;
    document.body.dataset.theme = st.theme; document.body.style.overflow = 'hidden'; document.body.style.background = '#888';
    app.style.display = 'block'; app.innerHTML = v ? frame(v) : '未找到页面 ' + id;
    return;
  }
  // 初始路由:#id/变体
  const h = location.hash.replace('#', '').split('/');
  if (h[0] && (h[0] === 'home' || h[0] === 'decide' || S.find(h[0]))) { st.screen = h[0]; if (h[1]) st.variant = h[1]; }
  render();
})();
