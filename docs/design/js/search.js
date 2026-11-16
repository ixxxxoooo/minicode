/* 设计稿 v3 · 全文搜索:左上搜索框(按 / 聚焦)搜遍所有页面的正文,点结果跳到那一页并高亮命中处,
   右上角小条可在本页的多处命中之间前后跳。索引直接取 APP.pageHtml(与主区渲染同一份 HTML),
   所以搜到的就是页面上看到的;只收各页默认状态下的内容(设置 / 自定义里切换后才出现的分类不在其中)。
   @author ixxxxoooo */
(function () {
  const { ic } = U, APP = window.APP, $ = (s, r = document) => r.querySelector(s);
  const BLOCK = new Set(['DIV', 'P', 'LI', 'UL', 'OL', 'TR', 'TD', 'TH', 'TABLE', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'SECTION', 'BR', 'BUTTON', 'PRE', 'HEADER', 'ASIDE', 'NAV', 'MAIN', 'ARTICLE', 'DT', 'DD', 'SUMMARY', 'LABEL', 'FIGURE']);
  const esc = (s) => s.replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
  const reEsc = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

  /** 把一棵 DOM 切成「文字块」:行内元素的文字连在一起,块级元素处断开;每块记下它的文本节点,用来回到页面上定位 */
  const chunks = (root) => {
    const out = []; let cur = null;
    const flush = () => { if (cur) { const t = cur.parts.join('').replace(/\s+/g, ' ').trim(); if (t) out.push({ text: t, nodes: cur.nodes, el: cur.el }); cur = null; } };
    const walk = (n) => {
      if (n.nodeType === 3) { if (!cur) cur = { parts: [], nodes: [], el: n.parentElement }; cur.parts.push(n.nodeValue); cur.nodes.push(n); return; }
      if (n.nodeType !== 1) return;
      const tag = n.tagName.toUpperCase();
      if (tag === 'SCRIPT' || tag === 'STYLE' || tag === 'SVG') return;
      const blk = BLOCK.has(tag); if (blk) flush();
      for (const c of n.childNodes) walk(c);
      if (blk) flush();
    };
    for (const c of root.childNodes) walk(c); flush();
    return out;
  };

  /* ───────── 索引(开页后分页逐个建,不卡首屏) ───────── */
  const index = []; let total = 0, done = false;
  const flat = () => APP.pages.flatMap(([g, items]) => items.map(([id, icon, title]) => ({ g, id, icon, title })));
  const buildOne = (pg) => {
    try {
      const tpl = document.createElement('template'); tpl.innerHTML = APP.pageHtml(pg.id);
      const ch = chunks(tpl.content).map((c) => ({ t: c.text, l: c.text.toLowerCase(), h: !!(c.el && c.el.closest('h1,h2,h3,h4,.grp-h,.sec-h')) }));
      index.push({ ...pg, l: pg.title.toLowerCase(), ch });
    } catch (e) { index.push({ ...pg, l: pg.title.toLowerCase(), ch: [] }); }
  };
  const startIndex = () => {
    const list = flat(); total = list.length; let i = 0;
    const step = () => { if (i >= list.length) { done = true; render(); return; } buildOne(list[i++]); if (i % 6 === 0) { status(); if (q()) render(); } setTimeout(step, 0); };
    setTimeout(step, 400);
  };

  /* ───────── 搜索框与结果 ───────── */
  const box = $('#vsearch'), list = $('#vlist'), res = $('#vres'), pill = $('#vfind');
  box.innerHTML = `<label class="vs-box">${ic('search', 's12')}<input id="vq" type="search" placeholder="搜索设计稿" autocomplete="off" spellcheck="false" aria-label="搜索设计稿"><kbd class="vs-k">/</kbd><button class="vs-x" id="vx" type="button" aria-label="清除" hidden>${ic('x', 's12')}</button></label><div class="vs-st" id="vst"></div>`;
  const inp = $('#vq'), clr = $('#vx'), st = $('#vst');
  const q = () => inp.value.trim();
  const toks = () => q().toLowerCase().split(/\s+/).filter(Boolean);
  const status = () => { st.textContent = done ? '' : `正在建立索引 ${index.length} / ${total}`; };
  let flatRows = [], act = -1;

  /** 在每个页面里找「同时含全部关键词」的文字块;页面标题命中加权,小标题次之 */
  const search = () => {
    const ts = toks(); if (!ts.length) return [];
    const out = [];
    for (const pg of index) {
      let score = ts.every((t) => pg.l.includes(t)) ? 100 : 0; const hits = [];
      pg.ch.forEach((c, k) => { if (ts.every((t) => c.l.includes(t))) { hits.push(k); score += c.h ? 6 : 1; } });
      if (score) out.push({ pg, score, hits, title: score >= 100 });
    }
    return out.sort((a, b) => b.score - a.score);
  };
  const snippet = (text, ts) => {
    const l = text.toLowerCase(); let at = Infinity;
    ts.forEach((t) => { const i = l.indexOf(t); if (i >= 0 && i < at) at = i; });
    const from = Math.max(0, (at === Infinity ? 0 : at) - 22), part = (from ? '…' : '') + text.slice(from, from + 96) + (text.length > from + 96 ? '…' : '');
    const re = new RegExp('(' + ts.map(reEsc).join('|') + ')', 'ig');
    return part.split(re).map((s, i) => (i % 2 ? `<mark>${esc(s)}</mark>` : esc(s))).join('');
  };
  const titleHtml = (title, ts) => title.split(new RegExp('(' + ts.map(reEsc).join('|') + ')', 'ig')).map((s, i) => (i % 2 ? `<mark>${esc(s)}</mark>` : esc(s))).join('');
  function render() {
    const has = !!q(); clr.hidden = !has; list.hidden = has; res.hidden = !has; status();
    if (!has) { flatRows = []; act = -1; return; }
    const ts = toks(), rs = search(), n = rs.reduce((a, r) => a + r.hits.length, 0);
    let html = `<div class="vr-sum">${rs.length ? `${rs.length} 页 · ${n} 处命中` : (done ? '没有找到' : '还在建立索引…')}</div>`;
    rs.slice(0, 40).forEach((r) => {
      html += `<div class="vr-pg" data-r="page" data-page="${r.pg.id}">${ic(r.pg.icon, 's12')}<span class="ell">${titleHtml(r.pg.title, ts)}</span><span class="vr-n">${r.hits.length || ''}</span></div>`;
      r.hits.slice(0, 3).forEach((k) => { html += `<div class="vr-hit" data-r="hit" data-page="${r.pg.id}" data-k="${k}">${snippet(r.pg.ch[k].t, ts)}</div>`; });
      if (r.hits.length > 3) html += `<div class="vr-more" data-r="page" data-page="${r.pg.id}" data-k="${r.hits[3]}">还有 ${r.hits.length - 3} 处…</div>`;
    });
    res.innerHTML = html; flatRows = [...res.querySelectorAll('[data-r]')]; act = -1;
  }
  const mark = (i) => { flatRows.forEach((r, j) => r.classList.toggle('act', j === i)); act = i; if (flatRows[i]) flatRows[i].scrollIntoView({ block: 'nearest' }); };

  /* ───────── 回到页面上定位并高亮 ───────── */
  let find = null; // { page, q, ranges, cur }
  const hl = !!(window.CSS && CSS.highlights && window.Highlight);
  const clearFind = () => { find = null; pill.hidden = true; if (hl) { CSS.highlights.delete('vhit'); CSS.highlights.delete('vcur'); } document.querySelectorAll('.vhit-el').forEach((e) => e.classList.remove('vhit-el')); };
  /** 主区里所有同时含全部关键词的文字块里,每个关键词的每一处,换成跨文本节点的 Range */
  const collect = (ts) => {
    const out = [];
    for (const c of chunks($('#vmain'))) {
      if (!ts.every((t) => c.text.toLowerCase().includes(t))) continue;
      const raw = c.nodes.map((n) => n.nodeValue).join(''), low = raw.toLowerCase(), found = [];
      ts.forEach((t) => { for (let i = low.indexOf(t); i >= 0 && found.length < 200; i = low.indexOf(t, i + t.length)) found.push([i, i + t.length]); });
      found.sort((a, b) => a[0] - b[0]);
      for (const [a, b] of found) {
        const r = document.createRange(); let pos = 0, s = false;
        for (const n of c.nodes) {
          const len = n.nodeValue.length;
          if (!s && a < pos + len) { r.setStart(n, a - pos); s = true; }
          if (s && b <= pos + len) { r.setEnd(n, b - pos); break; }
          pos += len;
        }
        if (s) out.push({ r, el: c.el, chunk: c });
      }
      if (out.length > 600) break;
    }
    return out;
  };
  const paint = () => {
    if (!find) return;
    const cur = find.ranges[find.cur];
    if (hl) { CSS.highlights.set('vhit', new Highlight(...find.ranges.map((x) => x.r))); if (cur) CSS.highlights.set('vcur', new Highlight(cur.r)); }
    pill.hidden = false;
    pill.innerHTML = `<span class="vf-q">「${esc(find.q)}」</span><span class="vf-n">${find.ranges.length ? `${find.cur + 1} / ${find.ranges.length}` : '本页无命中'}</span>
      <button type="button" data-f="prev" aria-label="上一处" ${find.ranges.length ? '' : 'disabled'}>${ic('chevU', 's12')}</button><button type="button" data-f="next" aria-label="下一处" ${find.ranges.length ? '' : 'disabled'}>${ic('chevD', 's12')}</button><button type="button" data-f="close" aria-label="关闭">${ic('x', 's12')}</button>`;
  };
  const reveal = (x) => {
    if (!x) return;
    const el = x.el || x.chunk.el; if (!el) return;
    el.scrollIntoView({ block: 'center', behavior: 'smooth' });
    document.querySelectorAll('.vhit-el').forEach((e) => e.classList.remove('vhit-el'));
    void el.offsetWidth; el.classList.add('vhit-el'); setTimeout(() => el.classList.remove('vhit-el'), 1800);
  };
  const step = (d) => { if (!find || !find.ranges.length) return; find.cur = (find.cur + d + find.ranges.length) % find.ranges.length; paint(); reveal(find.ranges[find.cur]); };
  /** 打开某页并定位到第 k 个文字块(k 缺省 = 本页第一处命中) */
  const open = (id, k) => {
    const ts = toks(), query = q();
    if (APP.st.page !== id) APP.go(id);
    clearFind();
    const ranges = collect(ts); let cur = 0;
    if (k != null) { // 索引里的第 k 块与页面上的同序;对不上(页面有动态内容)就退回第一处
      const want = index.find((p) => p.id === id), txt = want && want.ch[k] ? want.ch[k].t : '';
      const i = ranges.findIndex((x) => x.chunk.text === txt); if (i >= 0) cur = i;
    }
    find = { page: id, q: query, ranges, cur }; paint();
    if (ranges.length) reveal(ranges[cur]); else $('#vmain').scrollTop = 0;
  };

  /* ───────── 事件 ───────── */
  inp.addEventListener('input', render);
  clr.addEventListener('click', () => { inp.value = ''; render(); inp.focus(); });
  inp.addEventListener('keydown', (e) => {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); if (!flatRows.length) return; mark(e.key === 'ArrowDown' ? Math.min(flatRows.length - 1, act + 1) : Math.max(0, act - 1)); return; }
    if (e.key === 'Enter') { e.preventDefault(); const r = flatRows[act < 0 ? 0 : act]; if (r) open(r.dataset.page, r.dataset.k != null ? +r.dataset.k : null); return; }
    if (e.key === 'Escape') { e.stopPropagation(); if (inp.value) { inp.value = ''; render(); } else { clearFind(); inp.blur(); } }
  });
  res.addEventListener('click', (e) => { const r = e.target.closest('[data-r]'); if (!r) return; mark(flatRows.indexOf(r)); open(r.dataset.page, r.dataset.k != null ? +r.dataset.k : null); });
  pill.addEventListener('click', (e) => { const b = e.target.closest('[data-f]'); if (!b) return; const f = b.dataset.f; if (f === 'close') clearFind(); else step(f === 'next' ? 1 : -1); });
  document.addEventListener('keydown', (e) => {
    const t = e.target, typing = t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable), mod = e.metaKey || e.ctrlKey;
    if ((e.key === '/' && !typing && !mod) || (mod && e.shiftKey && e.key.toLowerCase() === 'f')) { e.preventDefault(); inp.focus(); inp.select(); return; }
    if (find && !typing && e.key === 'F3') { e.preventDefault(); step(e.shiftKey ? -1 : 1); }
  });
  // 重新渲染本页(换主题 / 方案、交互刷新)后高亮会丢,按原关键词补回;换了页就清掉
  APP.hooks.push(() => { if (!find) return; if (find.page !== APP.st.page) return clearFind(); const cur = find.cur; find.ranges = collect(find.q.toLowerCase().split(/\s+/).filter(Boolean)); find.cur = Math.min(cur, Math.max(0, find.ranges.length - 1)); paint(); });
  APP.search = { open, index: () => index };
  startIndex();
})();
