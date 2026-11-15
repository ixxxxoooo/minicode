/* 设计稿 v3.1 · 工作区交互:在会话流里点什么,就在工作区开什么标签;对话栏历史 / 新会话 / 占满;任务坞弹层;底部面板分段
   所有动作走 APP.ACT 的事件委托。Agent 形态下工作区收起时,开标签会先把它展开(并闪一下边框)。
   @author ixxxxoooo */
(function () {
  const A = window.APP, ACT = A.ACT, { $, $$ } = A;
  const win = () => $('#vmain .win');
  const cls = (el, c, on) => el && el.classList.toggle(c, on);
  const kick = (el, c) => { if (!el) return; el.classList.remove(c); void el.offsetWidth; el.classList.add(c); };

  /** 同步标题栏三个面板开关的高亮 */
  A.syncBars = (w) => { if (!w) return;
    $$('[data-act="panel"]', w).forEach(b => cls(b, 'on', w.dataset.panel !== 'off'));
    $$('.tbar [data-act="right"]', w).forEach(b => cls(b, 'on', w.dataset.right !== 'off'));
    $$('.tbar [data-act="side"]', w).forEach(b => cls(b, 'on', w.dataset.side !== 'off')); };
  A.hooks.push(() => A.syncBars(win()));

  /* ───────── 工作区标签 ───────── */
  const model = (w) => { const e = $('.ws-in', w); return e ? { tabs: e.dataset.tabs ? e.dataset.tabs.split(',') : [], active: e.dataset.active, panel: e.dataset.pk || undefined } : { tabs: [], active: '' }; };
  const render = (w, m) => { const ws = $('.ws', w), body = $('.ws-body', ws), tabs = m.tabs;
    if (!ws) return;
    ws.innerHTML = tabs.length ? WS.html({ tabs, active: m.active, panel: m.panel }) : `<div class="ws-in" data-tabs="" data-active=""><div class="etabs"></div><div class="ws-body"><div class="ws-empty"><div><div class="fs-title m t2">没有打开的标签</div><div class="t3" style="margin-top:4px">从文件树、源代码管理或会话流里打开内容</div></div></div></div>${IDE.bpanel(m.panel)}</div>`;
    const ps = $('.ws-in', ws); if (ps && m.panel) ps.dataset.pk = m.panel;
    /* 激活的标签滚进可视区(标签多、放不下时) */
    const on = $('.etabs .on, .etabs [aria-selected=true]', ws); if (on && on.scrollIntoView) on.scrollIntoView({ block: 'nearest', inline: 'nearest' }); };
  /** 打开(或激活)一个标签。Agent 形态下右侧工作区收起时先展开 */
  A.openTab = (w, id) => { if (!WS.TAB[id]) return;
    const m = model(w); if (!m.tabs.includes(id)) m.tabs.push(id); m.active = id; render(w, m);
    if (w.dataset.mode === 'agent' && w.dataset.right === 'off') { w.dataset.right = 'on'; A.syncBars(w); }
    if (w.dataset.mode === 'ide' && w.dataset.chatmax === 'on') w.dataset.chatmax = 'off';
    kick($('.ws', w), 'ws-flash'); };
  ACT.open = (el, w) => { closePops(); if (!w) return; if (!WS.TAB[el.dataset.id]) return; A.openTab(w, el.dataset.id); };
  ACT.wtab = (el, w) => { const m = model(w); m.active = el.dataset.id; render(w, m); };
  ACT.wx = (el, w, e) => { e.stopPropagation(); const m = model(w), k = m.tabs.indexOf(el.dataset.id); if (k < 0) return;
    m.tabs.splice(k, 1); if (m.active === el.dataset.id) m.active = m.tabs[Math.min(k, m.tabs.length - 1)] || '';
    render(w, m); if (!m.tabs.length && w.dataset.mode === 'agent') { w.dataset.right = 'off'; A.syncBars(w); } };

  /* ───────── 审查标签里的块级操作 ───────── */
  ACT['rv-fold'] = (el) => cls(el.closest('.rv-file'), 'closed');
  ACT['rv-viewed'] = (el) => { const c = $('.chk', el); cls(c, 'done'); };
  ACT['rv-hunk'] = (el) => { const bar = el.closest('.rv-hunk'), ok = el.dataset.do === 'ok';
    $$('.btn', bar).forEach(b => b.remove()); bar.insertAdjacentHTML('beforeend', ok ? '<span class="tag ok">已接受</span>' : '<span class="tag err">已撤销</span>');
    let n = bar.nextElementSibling; while (n && n.classList.contains('dl')) { if (!ok) n.style.opacity = .35; n = n.nextElementSibling; } };
  ACT['rv-cm'] = (el) => { const f = el.closest('.rv-file'), cm = $('.rv-cm', f); if (cm) kick(cm, 'a-flash'); };

  /* ───────── 对话栏:历史、新会话、占满、任务坞 ───────── */
  const closePops = () => { let any = false; $$('.hist-pop, .tpop, .cm-pop').forEach(p => { p.remove(); any = true; }); $$('.tdock .chip').forEach(c => c.classList.remove('pop-on')); return any; };
  const prevClose = A.closeLayer; A.closeLayer = () => closePops() || (prevClose ? prevClose() : false);
  document.addEventListener('mousedown', (e) => { if (!$('.hist-pop, .tpop') || e.target.closest('.hist-pop, .tpop, [data-act="hist"], [data-act="tpop"]')) return; closePops(); });
  ACT.hist = (el, w) => { const chat = $('.chat', w); if (!chat) return; const had = $('.hist-pop', chat); closePops(); if (had) return; chat.insertAdjacentHTML('beforeend', F.hist());
    $$('.hist-pop .row', chat).forEach(r => r.dataset.act = 'hist-pick'); const i = $('.hist-pop input', chat); if (i) setTimeout(() => i.focus(), 30); };
  ACT['hist-pick'] = (el, w) => { closePops(); const chat = $('.chat', w); chat.innerHTML = W.chat({ screen: 'work' }); };
  ACT.newchat = (el, w) => { closePops(); const chat = $('.chat', w); chat.innerHTML = W.chat({ screen: 'welcome' }); };
  ACT.chatmax = (el, w) => { w.dataset.chatmax = w.dataset.chatmax === 'on' ? 'off' : 'on'; };
  ACT.tpop = (el, w) => { const foot = $('.chat-foot .col', w), had = $('.tpop', w); const same = had && el.classList.contains('pop-on'); closePops(); if (same) return;
    el.classList.add('pop-on'); const d = $('.dockslot', foot); d.insertAdjacentHTML('afterbegin', F.tpop(el.dataset.k)); };
  ACT.pterm = (el, w) => { w.dataset.panel = 'on'; A.syncBars(w); kick($('.bpanel', w), 'a-flash');
    if (w.dataset.mode === 'agent' && w.dataset.right === 'off') { w.dataset.right = 'on'; A.syncBars(w); } };

  /* ───────── 底部面板分段 / 提交菜单 ───────── */
  const KINDS = ['term', 'problems', 'out', 'refs'];
  ACT.ptab = (el, w) => { const tabs = $$('.bpanel .tabs .tab', w), k = tabs.indexOf(el); const m = model(w), kind = KINDS[k] || 'term';
    const cur = $('.bpanel', w); const t = document.createElement('div'); t.innerHTML = IDE.bpanel(kind === 'out' ? 'term' : kind); cur.replaceWith(t.firstElementChild); const ps = $('.ws-in', w); if (ps) ps.dataset.pk = kind; };
  ACT.cmenu = (el, w) => { const body = $('.side-body', w); if (!body) return; w._cm = !w._cm; body.innerHTML = W.panel('scm', w.closest('.frame').dataset.scheme, { cmenu: w._cm }); };
})();
