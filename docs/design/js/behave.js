/* 设计稿 v3 · 交互层
   app.js 只管路由与形态切换;这里负责「点了之后真的发生点什么」:
   提示条 / 模态框 / 弹层、实时偏好(密度 / 圆角 / 强调色 / 动效)、设置页(搜索、已修改与还原、需重启条、快捷键录制、服务商测试……)、
   自定义页(选中 / 启停 / 新建 / 删除 / 安装 / 授权 / 试运行)、自动化、画布、浏览器、用量。
   所有动作都走 APP.ACT 的事件委托;整页重渲染后由 APP.hooks 恢复提示条、偏好与「已修改」标记。
   @author ixxxxoooo */
(function () {
  const A = window.APP, st = A.st, ACT = A.ACT, { $, $$ } = A;
  const { ic, btn, ib, tag, kbd, esc } = U;
  const cu = W.cu, au = W.au, API = W.settingsApi;
  const later = (ms, fn) => setTimeout(fn, ms);
  const win = () => $('#vmain .win');
  const cls = (el, c, on) => el && el.classList.toggle(c, on);
  /** 重播一次 CSS 动画类 */
  const kick = (el, c) => { if (!el) return; el.classList.remove(c); void el.offsetWidth; el.classList.add(c); };
  const shake = (el) => kick(el, 'a-shake'), flash = (el) => kick(el, 'a-flash');
  const loading = (b, on) => b && b.classList.toggle('loading', on);
  const segIdx = (seg) => +(seg.style.getPropertyValue('--i') || 0);
  const idxOn = (box, sel = '.on') => [...box.children].findIndex(c => c.matches(sel));

  /* ═════════ 偏好:密度 / 圆角 / 强调色 / 动效(设计令牌的倍率,即改即生效) ═════════ */
  st.prefs = Object.assign({ density: 1, radius: 1, accent: '', motion: 1 }, st.prefs);
  const BASE = { a: { h: 24, r: .8 }, b: { h: 26, r: 1.4 }, c: { h: 22, r: 1 } };
  const applyPrefs = () => {
    const p = st.prefs;
    $$('.frame').forEach(f => { const b = BASE[f.dataset.scheme] || BASE.a;
      f.style.setProperty('--h-row', (b.h + [-2, 0, 2][p.density]) + 'px');
      f.style.setProperty('--r-k', +(b.r * [.55, 1, 1.35][p.radius]).toFixed(3));
      p.accent ? f.style.setProperty('--c-accent', `var(${p.accent})`) : f.style.removeProperty('--c-accent'); });
    document.body.classList.toggle('no-motion', !p.motion);
  };
  /** 设置页里的实时控件:分段(data-key)与动效开关(data-live="motion") */
  ACT.live = (el) => {
    if (el.dataset.live === 'motion') st.prefs.motion = el.classList.contains('on') ? 1 : 0;
    else { const s = el.closest('[data-key]'); if (s) st.prefs[s.dataset.key] = +el.dataset.i; }
    applyPrefs(); A.save();
  };
  /** 行被还原 / 恢复后,把控件状态同步回偏好 */
  const syncLive = (row) => {
    $$('[data-key]', row).forEach(s => { st.prefs[s.dataset.key] = segIdx(s); });
    $$('.tog[data-live="motion"]', row).forEach(t => { st.prefs.motion = t.classList.contains('on') ? 1 : 0; });
    $$('.swatch', row).forEach((w, k, all) => { if (w.dataset.act === 'accent' && w.classList.contains('on')) st.prefs.accent = w.dataset.v || ''; });
    applyPrefs(); A.save();
  };
  ACT.accent = (el) => { $$('.swatch', el.parentElement).forEach(s => cls(s, 'on', s === el)); st.prefs.accent = el.dataset.v || ''; applyPrefs(); A.save(); };
  ACT['accent-pick'] = (el) => $$('.swatch', el.parentElement).forEach(s => cls(s, 'on', s === el));

  /* ═════════ 提示条(右下角,可堆叠、可带动作与进度) ═════════ */
  let toastBox = null;
  const ICONS = { info: ['info', 'c-info'], ok: ['okc', 'c-ok'], warn: ['alert', 'c-warn'], err: ['errc', 'c-err'] };
  const closeToast = (t) => { if (!t || t._gone) return; t._gone = true; clearTimeout(t._t); t.classList.add('a-toast-out'); later(230, () => t.remove()); };
  /** toast('文案',{lvl,sub,act:{label,fn},progress,ms}) → { set(p, 文案), done(文案, lvl), close() } */
  const toast = (text, o = {}) => {
    const w = win(); if (!w) return {};
    if (!toastBox || !toastBox.isConnected) { toastBox = toastBox || document.createElement('div'); toastBox.className = 'toasts'; w.appendChild(toastBox); }
    const live = () => [...toastBox.children].filter(c => !c._gone);   /* 正在退场的不算,否则同步循环永远等不到它被移出 */
    while (live().length >= 3) closeToast(live()[0]);
    const [ico, c] = ICONS[o.lvl || 'info'];
    const t = document.createElement('div'); t.className = 'pop toast a-rise'; t.setAttribute('role', 'status');
    t.innerHTML = `<div class="nt"><span class="t-ic">${ic(ico, 's16 ' + c)}</span><div class="grow" style="min-width:0"><div class="m t-tx">${esc(text)}</div>${o.sub ? `<div class="t3 fs-cap ell">${esc(o.sub)}</div>` : ''}</div>${o.act ? `<button class="btn sm ghost" data-toast-act>${o.act.label}</button>` : ''}${ib('x', 'sm', '关闭', 'data-act="toast-x"')}</div>${o.progress ? '<div class="meter"><i style="width:0"></i></div>' : ''}`;
    toastBox.appendChild(t);
    const arm = (ms) => { clearTimeout(t._t); t._t = later(ms, () => closeToast(t)); };
    if (!o.progress) arm(o.ms || 3800);
    t.addEventListener('mouseenter', () => clearTimeout(t._t));
    t.addEventListener('mouseleave', () => { if (!t._busy) arm(1800); });
    if (o.act) $('[data-toast-act]', t).addEventListener('click', () => { o.act.fn(); closeToast(t); });
    t._busy = !!o.progress;
    return { el: t, close: () => closeToast(t),
      set(p, txt) { $('.meter i', t).style.width = p + '%'; if (txt) $('.t-tx', t).textContent = txt; },
      done(txt, lvl = 'ok') { t._busy = false; const m = $('.meter', t); if (m) m.remove(); $('.t-tx', t).textContent = txt; const [i2, c2] = ICONS[lvl]; $('.t-ic', t).innerHTML = ic(i2, 's16 ' + c2); kick($('.t-ic', t), 'a-ok'); arm(2800); } };
  };
  ACT.toast = (el) => toast(el.dataset.msg || '已完成');
  ACT['toast-x'] = (el) => closeToast(el.closest('.toast'));

  /* ═════════ 模态框 ═════════ */
  let M = null;
  const closeModal = (instant) => { if (!M) return; const m = M; M = null; if (m.o.onCancel && !m.done) m.o.onCancel(); m.el.classList.add('out'); later(instant ? 0 : 170, () => m.el.remove()); };
  /** modal({icon,lvl,title,body,form,ok,okCls,cancel,danger,onOk(el)→false 则不关闭}) */
  const modal = (o) => {
    closeModal(true); const w = o.win || win(); if (!w) return;
    w.insertAdjacentHTML('beforeend', `<div class="modal"><div class="scrim" data-act="modal-cancel"></div><div class="dlg ${o.wide ? 'wide' : ''}" role="dialog" aria-modal="true"><div class="f g3"><span class="dlg-i ${o.lvl || ''}">${ic(o.icon || 'info', 's16')}</span><div class="grow" style="min-width:0"><b>${o.title}</b>${o.body ? `<div class="t3" style="line-height:1.55;margin-top:3px">${o.body}</div>` : ''}</div></div>${o.form || ''}<div class="acts">${btn(o.cancel || '取消', 'ghost', '', 'data-act="modal-cancel"')}${btn(o.ok || '确定', o.danger ? 'danger' : 'pri', '', 'data-act="modal-ok"')}</div></div></div>`);
    M = { el: w.lastElementChild, o }; const f = $('input,textarea', M.el); if (f) later(60, () => f.focus());
  };
  ACT['modal-cancel'] = () => closeModal();
  ACT['modal-ok'] = () => { if (!M) return; const r = M.o.onOk ? M.o.onOk(M.el) : true; if (r === false) return shake($('.dlg', M.el)); M.done = true; closeModal(); };
  const errField = (el, msg) => { el.classList.add('err'); (el.closest('.inp') || el).classList.add('err'); let m = el.parentElement.querySelector('.err-msg'); if (!m) { m = document.createElement('div'); m.className = 'err-msg a-rise'; (el.closest('.fgrp') || el.closest('.inp') || el).appendChild(m); } m.innerHTML = ic('alert', 's12') + msg; el.focus(); };
  const clearErr = (el) => { el.classList.remove('err'); const inp = el.closest('.inp'); inp && inp.classList.remove('err'); const g = el.closest('.fgrp') || el.parentElement; const m = g && g.querySelector('.err-msg'); m && m.remove(); };

  /* ═════════ 弹层:账号菜单、状态栏弹层 ═════════ */
  const closePop = () => { const p = $('.acct, .sb-pop', win() || document); if (!p) return false; p.classList.remove('a-pop-up'); p.classList.add('a-out'); later(160, () => p.remove()); return true; };
  const openPop = (kind, html, w) => { const cur = $('.acct, .sb-pop', w); const same = cur && cur.dataset.kind === kind; closePop(); if (same) return;
    later(cur ? 170 : 0, () => { w.insertAdjacentHTML('beforeend', html); w.lastElementChild.dataset.kind = kind; }); };
  ACT.acct = (el, w) => openPop('acct', W.acctMenu(), w);
  ACT.sbpop = (el, w) => openPop(el.dataset.pop, W.sbPop(el.dataset.pop), w);
  document.addEventListener('mousedown', (e) => { if (!$('.acct, .sb-pop') || !e.target.closest || e.target.closest('.acct, .sb-pop, [data-act="acct"], [data-act="sbpop"]')) return; closePop(); });

  /** Esc:按「模态 → 录制 → 弹层 → 搜索」的顺序,关掉最上面的一层 */
  A.closeLayer = () => { if (M) { closeModal(); return true; } if (A.recording) { endRec(true); return true; } if (closePop()) return true;
    if (st.setQ) { st.setQ = ''; refreshSettings(true); return true; } return false; };

  /* ═════════ 设置页:搜索 ═════════ */
  /** 局部刷新设置页:换掉分类导航与正文,不动搜索框(保持焦点与光标) */
  const refreshSettings = (clearInput) => {
    const set = $('.set'); if (!set) return;
    const t = document.createElement('div'); t.innerHTML = W.settings(st);
    $('.set-nav > .scroll', set).replaceWith($('.set-nav > .scroll', t));
    const col = $('#set-col'); col.innerHTML = $('#set-col', t).innerHTML; col.classList.remove('a-rise'); $('#set-main').scrollTop = 0;
    if (clearInput) { const i = $('[data-in="setq"]'); if (i) i.value = ''; }
    initRows();
  };
  document.addEventListener('input', (e) => {
    const t = e.target, k = t.dataset && t.dataset.in;
    if (k === 'setq') { st.setQ = t.value; refreshSettings(); }
    else if (k === 'cuq') { st.cuQ = t.value; refreshCu({ list: 1, drawer: 1 }); }
    else if (k === 'kbq') { const q = t.value.trim().toLowerCase(); $$('#set-col .set-row').forEach(r => { r.hidden = !!q && !r.textContent.toLowerCase().includes(q); }); $$('#set-col .set-grp').forEach(g => { g.hidden = !$$('.set-row', g).some(r => !r.hidden); }); }
    else if (k === 'sys') { const n = $('[data-count="sys"]'); if (n) n.textContent = t.value.length + ' 字'; }
    if (t.classList && (t.classList.contains('err') || t.closest('.inp.err'))) clearErr(t);
    const row = t.closest('#set-col .set-row[data-k]'); if (row) markDirty(row);
  });
  ACT.setgo = (el) => { st.setTab = el.dataset.tab; st.setQ = ''; const k = el.dataset.k; refreshSettings(true); A.save();
    const row = $$('#set-col .set-row[data-k]').find(r => r.dataset.k === k); if (row) { row.scrollIntoView({ block: 'center', behavior: 'smooth' }); flash(row); } };
  ACT['setq-clear'] = () => { st.setQ = ''; refreshSettings(true); };

  /* ═════════ 设置页:已修改标记、还原、需重启条 ═════════ */
  const SAVED = new Map(), RESTART = new Set(); let LATER = 0;
  const rowSig = (r) => [$$('.tog', r).map(t => +t.classList.contains('on')), $$('.seg', r).map(segIdx), $$('input,textarea', r).map(i => i.value), $$('.stp b', r).map(b => b.textContent),
    $$('.chip', r).map(c => +c.classList.contains('on')), $$('.swatch', r).map(c => +c.classList.contains('on')), $$('.keycap', r).map(k => k.dataset.cur || '')].join('¦');
  const rowKey = (r) => W.settingsCur(st) + '|' + r.dataset.k;
  const syncAttrs = (r) => { $$('input', r).forEach(i => i.setAttribute('value', i.value)); $$('textarea', r).forEach(i => { i.textContent = i.value; }); };
  const updRbar = () => { const bar = $('.rbar:not([data-static])'); if (!bar) return; const n = RESTART.size, show = n > 0 && n !== LATER;
    if (show && bar.hidden) { bar.hidden = false; kick(bar, 'a-rise'); } else if (!show) bar.hidden = true;
    const c = $('.rcount', bar); if (c) c.textContent = n; };
  const markDirty = (row) => { if (!row._orig) return; const d = rowSig(row) !== row._sig, key = rowKey(row);
    cls(row, 'chg', d);
    if (d) { syncAttrs(row); SAVED.set(key, { html: row.innerHTML, sig0: row._sig }); row.hasAttribute('data-restart') && RESTART.add(key); } else { SAVED.delete(key); RESTART.delete(key); }
    updRbar(); };
  /** 每次(重)渲染设置页之后:记下原始状态,并恢复之前改过的行 */
  function initRows() {
    const col = $('#set-col'); if (!col) { updRbar(); return; }
    const tab = W.settingsCur(st);
    $$('.set-row[data-k]', col).forEach(r => {
      if (r.classList.contains('sr') || !$('.rst', r)) return;
      r._orig = r.innerHTML; r._sig = rowSig(r);
      const sv = SAVED.get(tab + '|' + r.dataset.k);
      if (sv) { r.innerHTML = sv.html; r._sig = sv.sig0; cls(r, 'chg', rowSig(r) !== r._sig); }
    });
    updRbar();
  }
  document.addEventListener('click', (e) => { if (e.target.closest('[data-act="runmode"]')) return; const row = e.target.closest('#set-col .set-row[data-k]'); if (row && !row.classList.contains('sr')) markDirty(row); });
  ACT['set-reset'] = (el) => { const row = el.closest('.set-row'); if (!row || !row._orig) return; row.innerHTML = row._orig; syncLive(row); markDirty(row); flash(row); };
  ACT['rbar-later'] = () => { LATER = RESTART.size; updRbar(); };
  ACT['rbar-go'] = () => { const keys = [...RESTART]; RESTART.clear(); LATER = 0; keys.forEach(k => SAVED.delete(k));
    $$('#set-col .set-row[data-restart]').forEach(r => { r._sig = rowSig(r); cls(r, 'chg', false); }); updRbar(); toast('已重启 MiniCode', { lvl: 'ok', sub: '设计稿里只模拟,不会真的退出' }); };

  /* ═════════ 设置页:基础控件 ═════════ */
  ACT.step = (el) => { const s = el.closest('.stp'), b = $('b', s), min = +s.dataset.min, max = +s.dataset.max, stp = +s.dataset.step || 1, v = +b.textContent, n = Math.min(max, Math.max(min, v + stp * +el.dataset.d));
    if (n === v) return shake(b); b.textContent = n; kick(b, 'a-ok'); };
  ACT['chip-one'] = (el) => $$('.chip', el.parentElement).forEach(c => { const on = c === el; c.classList.toggle('on', on); c.classList.toggle('line', !on); });
  ACT['chip-tog'] = (el) => { el.classList.toggle('on'); const g = el.closest('.tgrid'); if (!g) return;
    const on = $$('.tsel.on', g).map(c => c.textContent), seg = g.parentElement.querySelector('.seg');
    if (seg) { const ro = !on.some(t => ['Edit', 'Write', 'Bash'].includes(t)), i = on.length === $$('.tsel', g).length ? 2 : ro ? 0 : 1; seg.style.setProperty('--i', i); $$('button', seg).forEach((b, k) => cls(b, 'on', k === i)); } };
  const PRESET = [['Read', 'Grep', 'Glob'], ['Read', 'Grep', 'Glob', 'Edit', 'Write', 'Bash'], null];
  ACT['agent-preset'] = (el) => { const g = el.closest('.fc').querySelector('.tgrid'), set = PRESET[+el.dataset.i]; $$('.tsel', g).forEach(c => cls(c, 'on', !set || set.includes(c.textContent))); };

  /* ═════════ 设置页:服务商 ═════════ */
  const provName = (row) => $('.m', row).childNodes[0].textContent.trim();
  ACT['prov-use'] = (el) => { const row = el.closest('[data-prov]'); $$('[data-prov]').forEach(r => { $('.prov-st', r).innerHTML = r === row ? tag('当前', 'ac') : btn('切换', 'sm', '', 'data-act="prov-use"'); });
    kick($('.prov-st .tag', row), 'a-ok'); toast('已切换到 ' + provName(row), { lvl: 'ok', sub: '新会话使用该服务商,正在运行的会话不受影响' }); };
  ACT['prov-test'] = (el) => { const row = el.closest('[data-prov]'), res = $('.prov-res', row), bad = /example\.com/.test(row.textContent);
    loading(el, true); res.innerHTML = '<span class="fs-cap t3 shimmer">测试中…</span>';
    later(1000, () => { if (!row.isConnected) return; loading(el, false);
      if (bad) { res.innerHTML = tag('连接失败 · 超时', 'err'); toast(provName(row) + ' 连接失败', { lvl: 'err', sub: '5 秒内没有响应,检查地址与网络', act: { label: '重试', fn: () => el.isConnected && el.click() } }); }
      else { res.innerHTML = tag('已连通 · ' + (120 + provName(row).length * 17) + ' ms', 'ok'); kick(res, 'a-ok'); } }); };
  ACT['prov-add'] = () => { const f = $('#prov-new'); if (!f) return; const open = !f.classList.contains('open'); f.classList.toggle('open', open); if (open) later(300, () => { $('#prov-name').focus(); f.scrollIntoView({ block: 'nearest', behavior: 'smooth' }); }); };
  ACT['prov-save'] = (el) => { const n = $('#prov-name'), u = $('#prov-url'); let bad = 0;
    [n, u].forEach(clearErr);
    if (!u.value.trim()) { errField(u, '请填写 Base URL'); bad = 1; } else if (!/^https?:\/\/\S+$/.test(u.value.trim())) { errField(u, '地址需以 http:// 或 https:// 开头'); bad = 1; }
    if (!n.value.trim()) { errField(n, '请填写名称'); bad = 1; }
    if (bad) return shake($('#prov-new .card'));
    const last = $$('[data-prov]').pop(); last.insertAdjacentHTML('afterend', API.provRow(esc(n.value.trim()), esc(u.value.trim()), 0, '')); const row = last.nextElementSibling; row.classList.add('a-rise');
    ['prov-name', 'prov-url', 'prov-key'].forEach(i => { $('#' + i).value = ''; }); $('#prov-new').classList.remove('open');
    toast('已添加 ' + n.value.trim(), { lvl: 'ok', sub: '正在测试连通性…' }); later(500, () => $('[data-act="prov-test"]', row) && $('[data-act="prov-test"]', row).click()); };

  /* ═════════ 设置页:权限规则、环境变量、删除(带撤销) ═════════ */
  const KINDS = ['allow', 'ask', 'deny'], KT = { allow: '始终允许', ask: '每次询问', deny: '始终拒绝' };
  ACT['rule-add'] = (el) => { const row = el.closest('.set-row'), inp = $('input', row), v = inp.value.trim(), kind = KINDS[segIdx($('.seg', row))]; clearErr(inp);
    if (!v) { errField(inp, '请输入规则'); return shake(row); }
    if (!/^[A-Za-z_][\w]*(\(.+\))?$/.test(v)) { errField(inp, '格式如 Bash(git status)、Read(src/**)'); return shake(row); }
    const dup = KINDS.find(k => $$(`#rg-${k} .mono`).some(m => m.textContent === v));
    if (dup) { const m = $$(`#rg-${dup} .mono`).find(x => x.textContent === v); flash(m.closest('.set-row')); return toast('这条规则已在「' + KT[dup] + '」里', { lvl: 'warn' }); }
    const card = $(`#rg-${kind} [data-list]`); card.insertAdjacentHTML('beforeend', API.ruleRow(kind, v)); inp.value = ''; flash(card.lastElementChild); toast('已添加到「' + KT[kind] + '」', { lvl: 'ok', sub: v }); };
  ACT['env-add'] = (el) => { const row = el.closest('.set-row'), k = $('[data-in="env-k"]', row), v = $('[data-in="env-v"]', row), lv = ['用户', '项目', '本地'][segIdx($('.seg', row))]; clearErr(k); clearErr(v);
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(k.value.trim())) { errField(k, '变量名只能含字母、数字与下划线,且不能以数字开头'); return shake(row); }
    if (!v.value.trim()) { errField(v, '请填写值'); return shake(row); }
    const list = $('#env-grp [data-list]'), ex = $$('.mono.m', list).find(m => m.textContent === k.value.trim());
    if (ex) { const r = ex.closest('.set-row'); $('.tag', r).textContent = lv; flash(r); toast('已覆盖 ' + k.value.trim(), { lvl: 'info', sub: '范围:' + lv }); }
    else { list.insertAdjacentHTML('beforeend', API.envRow(k.value.trim(), lv)); flash(list.lastElementChild); toast('已添加 ' + k.value.trim(), { lvl: 'ok', sub: '值已遮盖' }); }
    k.value = v.value = ''; };
  const FAKE = { HTTPS_PROXY: 'http://127.0.0.1:7890', ANTHROPIC_API_KEY: 'sk-ant-api03-…a91f', API_TIMEOUT_MS: '600000' };
  ACT['env-eye'] = (el) => { const row = el.closest('.set-row'), v = $('.env-v', row), name = $('.m', row).textContent, show = v.dataset.mask === '1';
    v.dataset.mask = show ? '0' : '1'; v.textContent = show ? (FAKE[name] || '1') : '••••••••'; el.innerHTML = ic(show ? 'eyeOff' : 'eye'); clearTimeout(row._m);
    if (show) row._m = later(4000, () => { if (v.dataset.mask === '0') el.click(); }); };
  /** 删除一行:先折叠到 0 高再移除,toast 里可撤销 */
  const removeRow = (row, label, then) => { const p = row.parentElement, nx = row.nextElementSibling; row.style.maxHeight = row.offsetHeight + 'px'; void row.offsetHeight; row.classList.add('rm');
    later(330, () => { row.remove(); then && then(); });
    toast('已删除 ' + label, { lvl: 'info', act: { label: '撤销', fn: () => { row.classList.remove('rm'); row.style.maxHeight = ''; row.style.opacity = ''; (nx && nx.parentElement === p ? p.insertBefore(row, nx) : p.appendChild(row)); flash(row); then && then(); } } }); };
  ACT['del-row'] = (el) => { const row = el.closest('.set-row'); removeRow(row, ($('.mono', row) || $('.m', row)).textContent.trim()); };

  /* ═════════ 设置页:连通性、检测、诊断、更新 ═════════ */
  ACT['net-check'] = (el) => { const rows = $$('#net-grp [data-nc]'); loading(el, true);
    rows.forEach(r => { $('.nc-ic', r).innerHTML = '<span class="spin"></span>'; const v = $('.nc-v', r); v.className = 'nc-v fs-cap t3 shimmer'; v.textContent = '检测中…'; });
    rows.forEach((r, i) => later(500 + i * 520, () => { if (!r.isConnected) return; const ok = r.dataset.nc === '1', v = $('.nc-v', r);
      $('.nc-ic', r).innerHTML = ic(ok ? 'okc' : 'errc', 's12 ' + (ok ? 'c-ok' : 'c-err')); kick($('.nc-ic', r), 'a-ok'); v.className = 'nc-v fs-cap ' + (ok ? 'c-ok' : 'c-err'); v.textContent = ok ? (+r.dataset.ms + (i * 7 % 11) - 5) + ' ms' : '超时';
      if (i === rows.length - 1) { loading(el, false); const n = rows.filter(x => x.dataset.nc === '1').length; toast(`${n} / ${rows.length} 个主机可连通`, { lvl: n === rows.length ? 'ok' : 'warn', sub: n === rows.length ? '' : 'github.com 超时,检查代理设置' }); } })); };
  ACT['claude-detect'] = (el) => { loading(el, true); later(800, () => { loading(el, false); const v = $('.claude-ver'); if (v) { v.innerHTML = tag('2.1.318', 'ok'); kick(v, 'a-ok'); } toast('找到 claude 2.1.318', { lvl: 'ok', sub: '/opt/homebrew/bin/claude' }); }); };
  const INSTALL = { Pyright: 'npm i -g pyright', gopls: 'go install golang.org/x/tools/gopls@latest', 'typescript-language-server': 'npm i -g typescript-language-server typescript' };
  ACT['copy-install'] = (el) => { const name = $('.m', el.closest('.set-row')).textContent, cmd = INSTALL[name] || '';
    try { navigator.clipboard && navigator.clipboard.writeText(cmd); } catch (e) { /* 剪贴板不可用时只提示 */ }
    const old = el.innerHTML; el.innerHTML = ic('check', 's12 c-ok') + '已复制'; kick(el, 'a-ok'); later(1500, () => { if (el.isConnected) el.innerHTML = old; }); toast('已复制安装命令', { lvl: 'ok', sub: cmd }); };
  ACT['diag-export'] = (el) => { loading(el, true); later(900, () => { loading(el, false); toast('诊断包已导出', { lvl: 'ok', sub: '~/Downloads/minicode-diag-20261115.zip', act: { label: '在访达中显示', fn: () => toast('已在访达中显示') } }); }); };
  ACT.doctor = (el) => { const out = $('#doctor-out'); if (out.classList.contains('open')) return out.classList.remove('open');
    loading(el, true); later(1200, () => { loading(el, false); out.classList.add('open'); out.scrollIntoView({ block: 'nearest', behavior: 'smooth' }); }); };
  ACT['reset-all'] = () => modal({ icon: 'undo', lvl: 'warn', title: '恢复默认客户端偏好?', body: '主题、发送键、通知、代理等会恢复默认值。不会影响 Claude 配置、栏宽与最近项目。', ok: '恢复默认', danger: true,
    onOk: () => { SAVED.clear(); RESTART.clear(); LATER = 0; st.prefs = { density: 1, radius: 1, accent: '', motion: 1 }; st.theme = 'paper'; A.apply(); A.renderBar(); A.save(); A.rerender(); toast('已恢复默认偏好', { lvl: 'ok' }); } });
  ACT.runmode = (el) => { const seg = el.closest('.seg'), to = +el.dataset.i, from = 1 - to, row = el.closest('.set-row');
    const back = () => { seg.style.setProperty('--i', from); $$('button', seg).forEach((b, k) => cls(b, 'on', k === from)); };
    modal({ icon: 'refresh', title: `切换到「${to ? '演示' : '真实'}」模式并重启?`, body: to ? '演示模式使用脚本化后端与沙盒项目,不连接 claude;切换后窗口会重启一次,运行中的会话会被停止。' : '真实模式会读取 ~/.claude 并按需启动 claude。', ok: '切换并重启', onCancel: back,
      onOk: () => { row._sig = rowSig(row); toast('已切换到' + (to ? '演示' : '真实') + '模式', { lvl: 'ok', sub: '设计稿里只模拟,不会真的重启' }); } }); };
  let upd = 0;
  ACT['upd-check'] = (el) => { const b = $('#upd-btn'), s = $('#upd-st'), bar = $('#upd-bar'); if (upd) return; upd = 1; loading(b, true);
    s.innerHTML = '<span class="shimmer">正在检查更新…</span>';
    later(900, () => { if (!s.isConnected) { upd = 0; return; } s.innerHTML = '<span class="c-ac">发现新版本 0.2.1,正在下载…</span>'; bar.style.display = ''; kick(bar, 'a-rise'); let p = 0;
      const iv = setInterval(() => { if (!s.isConnected) { clearInterval(iv); upd = 0; return; } p += 6 + (p > 60 ? 3 : 0); $('i', bar).style.width = Math.min(p, 100) + '%';
        if (p < 100) s.innerHTML = `<span class="c-ac num">下载中 ${p}%</span>`;
        else { clearInterval(iv); s.innerHTML = '<span class="shimmer">校验中…</span>'; later(600, () => { s.innerHTML = '<span class="shimmer">解包中…</span>'; later(600, () => { if (!s.isConnected) { upd = 0; return; } upd = 0; bar.style.display = 'none'; s.innerHTML = `<span class="c-ok f ai g2">${ic('okc', 's12')}0.2.1 已就绪,重启后生效</span>`; kick(s, 'a-ok'); loading(b, false); b.innerHTML = ic('refresh', 's12') + '重启更新'; b.dataset.act = 'upd-restart'; b.classList.add('pri'); }); }); } }, 140); }); };
  ACT['upd-restart'] = () => toast('正在重启并应用 0.2.1…', { lvl: 'info', sub: '设计稿里只模拟' });

  /* ═════════ 设置页:快捷键录制与冲突检测 ═════════ */
  const SPECIAL = { Enter: '⏎', Tab: '⇥', Backspace: '⌫', Delete: '⌦', ArrowUp: '↑', ArrowDown: '↓', ArrowLeft: '←', ArrowRight: '→', Space: 'Space', Backquote: '`', Backslash: '\\', Slash: '/', Comma: ',', Period: '.', Semicolon: ';', Quote: "'", BracketLeft: '[', BracketRight: ']', Minus: '-', Equal: '=' };
  const combo = (e) => { const c = e.code, key = c.startsWith('Key') ? c.slice(3) : c.startsWith('Digit') ? c.slice(5) : /^F\d+$/.test(c) ? c : SPECIAL[c];
    if (!key) return null; const mods = [e.ctrlKey && '⌃', e.altKey && '⌥', e.shiftKey && '⇧', e.metaKey && '⌘'].filter(Boolean);
    return { parts: [...mods, key], mods: mods.length, fn: /^F\d+$/.test(c) }; };
  const curOf = (k) => k.dataset.cur === undefined || k.dataset.cur === '' ? k.dataset.def : k.dataset.cur;
  const paint = (cap, str) => { cap.innerHTML = str === '—' ? '<span class="t4 fs-cap">未设置</span>' : kbd(...str.split(' ')); };
  let REC = null;
  const warnBox = (html) => { const w = $('.kb-warn'); if (!w) return; w.hidden = !html; w.innerHTML = html || ''; if (html) kick(w, 'a-rise'); };
  function endRec(cancel) { if (!REC) return; const r = REC; REC = null; A.recording = false; r.cap.classList.remove('rec'); if (cancel || !r.keep) paint(r.cap, r.prev); warnBox(''); }
  ACT.keyrec = (el) => { if (REC) endRec(true); REC = { cap: el, prev: curOf(el) }; A.recording = true; el.classList.add('rec'); el.innerHTML = '<span class="fs-cap">按下新的快捷键…  Esc 取消</span>'; warnBox(''); };
  document.addEventListener('mousedown', (e) => { if (REC && !(e.target.closest && e.target.closest('.keycap.rec'))) endRec(true); });
  document.addEventListener('keydown', (e) => { if (!REC) return; e.preventDefault(); e.stopPropagation();
    if (e.key === 'Escape') return endRec(true);
    const c = combo(e); if (!c) return;
    if (!c.mods && !c.fn) { warnBox(`<span class="grow">快捷键至少要带一个修饰键(⌘ ⌃ ⌥ ⇧)</span>`); return shake($('.kb-warn')); }
    const str = c.parts.join(' '), row = REC.cap.closest('.set-row'), other = $$('.keycap').find(k => k !== REC.cap && curOf(k) === str);
    if (other) { REC.pending = str; warnBox(`${ic('alert', 's12 c-warn')}<span class="grow">「${kbd(...c.parts)}」已被「${$('.m', other.closest('.set-row')).textContent}」使用</span>${btn('替换', 'sm', '', 'data-act="kb-replace"')}${btn('取消', 'sm ghost', '', 'data-act="kb-keep"')}`); REC.other = other; REC.cap.innerHTML = kbd(...c.parts); return shake($('.kb-warn')); }
    REC.cap.dataset.cur = str; REC.keep = 1; paint(REC.cap, str); kick(REC.cap, 'a-ok'); endRec(); markDirty(row); }, true);
  ACT['kb-replace'] = () => { if (!REC) return; const { cap, other, pending } = REC, orow = other.closest('.set-row');
    other.dataset.cur = '—'; paint(other, '—'); markDirty(orow); flash(orow); cap.dataset.cur = pending; REC.keep = 1; paint(cap, pending); kick(cap, 'a-ok'); const row = cap.closest('.set-row'); endRec(); markDirty(row); };
  ACT['kb-keep'] = () => endRec(true);
  ACT['kb-reset'] = () => { $$('.keycap').forEach(k => { k.dataset.cur = ''; paint(k, k.dataset.def); }); $$('#set-col .set-row.chg').forEach(r => markDirty(r)); toast('已恢复默认快捷键', { lvl: 'ok' }); };

  /* ═════════ 自定义页 ═════════ */
  const SCK = ['all', 'u', 'p', 'l'], cc = () => cu.cur(st);
  const swapAnim = (el) => kick(el, 'a-swap');
  /** 局部刷新自定义页的列表 / 详情 / 左栏计数 */
  function refreshCu(o = { list: 1, drawer: 1 }) {
    if (o.list) { const l = $('#cu-list'); if (l) l.innerHTML = cu.list(st); }
    if (o.drawer) { const d = $('#cu-drawer'); if (d) { d.innerHTML = cu.drawer(st); swapAnim(d); } }
    $$('[data-cat-n]').forEach(n => { n.textContent = cu.DATA[n.dataset.catN].length; });
  }
  const refreshPage = () => { const p = $('.pane.page'); if (!p) return; const sc = $('.pg-main.scroll', p), top = sc ? sc.scrollTop : 0; p.innerHTML = W.pages[st.page](st); const n = $('.pg-main.scroll', p); if (n) n.scrollTop = top; };
  ACT['cu-sel'] = (el) => { st.cuSel = +el.dataset.i; $$('.cu-row, .mk-card').forEach(r => cls(r, 'on', r === el)); refreshCu({ drawer: 1 }); };
  ACT['cu-scope'] = (el) => { st.cuScope = SCK[+el.dataset.i]; st.cuSel = 0; refreshCu(); };
  ACT['cu-tab'] = (el) => { st.cuTab = +el.dataset.i ? 'market' : ''; st.cuSel = 0; A.rerender(); };
  ACT['cu-tab-market'] = () => { st.cuTab = 'market'; st.cuSel = 0; A.rerender(); };
  ACT['cu-clear'] = () => { st.cuQ = ''; st.cuScope = 'all'; A.rerender(); };
  ACT['rule-mode'] = (el) => { const pv = +el.dataset.i === 1; if (pv) { const body = $('#rule-b').value; $('#rule-pv').innerHTML = body.split('\n').map(l => l.startsWith('# ') ? `<div class="b fs-title" style="margin:2px 0 4px">${esc(l.slice(2))}</div>` : l.startsWith('- ') ? `<div class="f g3"><span class="dot ac" style="margin-top:7px"></span><span>${esc(l.slice(2))}</span></div>` : l ? `<div>${esc(l)}</div>` : '<div style="height:6px"></div>').join(''); kick($('#rule-pv'), 'a-swap'); }
    $('#rule-ed').hidden = pv; $('#rule-pv').hidden = !pv; };

  /** 开关:自定义里的整项 / MCP 工具、自动化的启停 */
  document.addEventListener('click', (e) => { const t = e.target.closest('.tog'); if (!t) return; const on = t.classList.contains('on');
    if (t.dataset.cu !== undefined) { const c = cc(), k = +t.dataset.cu, r = cu.DATA[c][k]; cu.setOn(c, k, on);
      toast((on ? '已启用 ' : '已停用 ') + r.n, { lvl: 'info', sub: c === 'mcp' ? '仅对当前项目生效,运行中的会话立即热切换' : c === 'plugins' ? '新会话生效' : '' });
      later(260, () => refreshCu(st.cuSel === k ? { list: 1, drawer: 1 } : { list: 1 })); }
    else if (t.dataset.cuTool) { const [k, j] = t.dataset.cuTool.split(':').map(Number); cu.setTool(k, j, on); }
    else if (t.dataset.au !== undefined) { const k = +t.dataset.au, a = au.AUS[k]; a.on = on ? 1 : 0; a.last = on ? a.last : '已暂停'; if (on && a.last === '已暂停') a.last = '刚刚启用';
      toast((on ? '已启用「' : '已暂停「') + a.n + '」', { lvl: 'info' }); later(260, () => { $('#au-list').innerHTML = au.auBody(st); $('#au-drawer').innerHTML = au.auDrawer(st); }); } });

  /** 新建 */
  ACT['cu-new'] = () => { const c = cc(), C = cu.CATS[c];
    modal({ icon: 'plus', title: '新建' + C.noun, wide: true, ok: '创建',
      form: `<div class="fgrp"><span class="lbl">名称</span>${U.field('', 0, { mono: 1, id: 'new-n', ph: cu.NEW_PH[c], attr: 'style="width:100%"' })}</div>${cu.newForm(c)}<div class="fgrp"><span class="lbl">保存到</span>${U.seg(['用户', '项目', '本地'], 1)}<span class="t4 fs-cap">${c === 'rules' ? '用户 = ~/.claude/CLAUDE.md;项目 = 项目根 CLAUDE.md;本地 = CLAUDE.local.md' : '用户层所有项目可用;项目层随仓库提交;本地层只在这台机器'}</span></div>`,
      onOk: (m) => { const n = $('#new-n', m); clearErr(n); let name = n.value.trim();
        if (!name) { errField(n, '请填写名称'); return false; }
        if (c === 'commands' && !name.startsWith('/')) name = '/' + name;
        if (/\s/.test(name) && ['skills', 'agents', 'commands'].includes(c)) { errField(n, '名称里不能有空格,可用 - 连接'); return false; }
        if (cu.has(c, name)) { errField(n, '已存在同名的' + C.noun); return false; }
        const x = $('#new-x', m), sc = 'upl'[segIdx($$('.seg', m).pop())], k = cu.add(c, name, sc, x ? x.value.trim() : '');
        if (c === 'hooks') cu.DATA.hooks[k].ev = Math.max(0, $$('.chip', m).findIndex(z => z.classList.contains('on')));
        st.cuSel = k; st.cuQ = ''; st.cuScope = 'all';
        if (c === 'plugins' || st.cuTab) st.cuTab = '';
        A.rerender(); later(40, () => { const row = $$('.cu-row').find(r => +r.dataset.i === k); if (row) { row.scrollIntoView({ block: 'nearest', behavior: 'smooth' }); flash(row); } });
        toast('已创建 ' + name, { lvl: 'ok', sub: c === 'mcp' ? '正在连接…' : '在右侧补全内容后保存' });
        if (c === 'mcp') later(1300, () => { const r = cu.DATA.mcp[k]; if (!r) return; r.st = 'ok'; r.d = 'stdio · 0 个工具 · 已连接'; r.tok = '连接后统计'; if (cc() === 'mcp') refreshCu(st.cuSel === k ? undefined : { list: 1 }); toast(name + ' 已连接', { lvl: 'ok' }); }); } }); };

  /** 删除 / 卸载(确认 + 撤销) */
  ACT['cu-del'] = () => { const c = cc(), k = st.cuSel, r = cu.DATA[c][k]; if (!r) return; const un = c === 'plugins';
    modal({ icon: 'trash', lvl: 'err', title: `${un ? '卸载' : '删除'}「${esc(r.n)}」?`, body: un ? '它包含的 Skills、命令、子 Agent 与钩子会一并移除。' : c === 'rules' || c === 'skills' ? '会删除磁盘上的对应文件,无法在回收站找回。' : '会从对应配置层移除,可在提示条里撤销。', ok: un ? '卸载' : '删除', danger: true,
      onOk: () => { const row = $$('.cu-row').find(x => +x.dataset.i === k); if (row) { row.style.maxHeight = row.offsetHeight + 'px'; void row.offsetHeight; row.classList.add('rm'); }
        later(row ? 320 : 0, () => { cu.remove(c, k); refreshCu(); });
        toast((un ? '已卸载 ' : '已删除 ') + r.n, { lvl: 'info', act: { label: '撤销', fn: () => { cu.DATA[c].splice(k, 0, r); st.cuSel = k; if (cc() === c) refreshCu(); } } }); } }); };

  /** 保存(校验 + 写回数据) */
  ACT['cu-save'] = (el) => {
    if (el.closest('#au-drawer')) { const a = au.AUS[st.auSel], n = $('#au-n'), p = $('#au-p'); clearErr(n); clearErr(p);
      if (!n.value.trim()) { errField(n, '请填写名称'); return shake($('#au-drawer')); } if (!p.value.trim()) { errField(p, '提示词不能为空'); return shake($('#au-drawer')); }
      a.n = n.value.trim(); a.prompt = p.value.trim(); loading(el, true); return later(450, () => { loading(el, false); $('#au-list').innerHTML = au.auBody(st); $('#au-wk').innerHTML = au.wk(st.wkOff || 0); toast('已保存「' + a.n + '」', { lvl: 'ok' }); }); }
    const c = cc(), r = cu.DATA[c][st.cuSel], d = $('#cu-drawer'), fl = $$('input.fld', d); if (!r) return;
    const bad = (f, m) => { errField(f, m); shake(d); return 1; };
    if (c === 'commands') { const n = $('#cmd-n'); clearErr(n); if (!/^\/[\w-]+$/.test(n.value.trim())) return bad(n, '以 / 开头,只含字母、数字与 -'); if (n.value.trim() !== r.n && cu.has(c, n.value.trim())) return bad(n, '已存在同名命令');
      r.n = n.value.trim(); r.desc = $('#cmd-d').value; r.hint = fl[2].value; r.body = $('#cmd-b').value; r.tools = $$('.tsel.on', d).map(x => x.textContent); r.model = segIdx($('.seg', d)); r.d = '项目 · ' + (r.desc || '无说明'); }
    else if (c === 'agents') { clearErr(fl[0]); if (!/^[\w-]+$/.test(fl[0].value.trim())) return bad(fl[0], '只能含字母、数字、_ 与 -'); r.n = fl[0].value.trim(); r.desc = fl[1].value; r.prompt = $('#ag-p').value; r.tools = $$('.tsel.on', d).map(x => x.textContent); r.model = segIdx($$('.seg', d)[0]); r.color = Math.max(0, $$('.swatch', d).findIndex(x => x.classList.contains('on'))); r.d = '项目 · ' + cu.MODELS[r.model].replace(' 5.5', '') + ' · ' + (r.tools.some(t => ['Edit', 'Write', 'Bash'].includes(t)) ? '可编辑' : '只读'); }
    else if (c === 'hooks') { const cmd = $('#hk-c'); clearErr(cmd); if (!cmd.value.trim()) return bad(cmd, '命令不能为空'); r.cmd = cmd.value.trim(); r.match = fl[0].value; r.to = +$('.stp b', d).textContent; r.ev = Math.max(0, $$('.chip', d).findIndex(x => x.classList.contains('on'))); }
    else if (c === 'rules') { r.body = $('#rule-b').value; r.tok = Math.ceil(r.body.length / 2.2); r.size = (r.body.length / 1024).toFixed(1) + ' KB'; }
    loading(el, true); later(450, () => { loading(el, false); refreshCu({ list: 1 }); toast('已保存「' + r.n + '」', { lvl: 'ok', sub: c === 'hooks' && r.sc === 'p' && !r.trust ? '项目钩子需先信任才会运行' : '新会话生效' }); }); };

  /** 插件:安装(进度)/ 更新 / 启停 */
  const progress = (label, done, then) => { const t = toast(label, { progress: true, sub: '0%' }); let p = 0; const iv = setInterval(() => { p += 9 + Math.random() * 9; if (p >= 100) { clearInterval(iv); then(); t.done(done); return; } t.set(p); $('.fs-cap', t.el).textContent = Math.round(p) + '%'; }, 160); return t; };
  ACT['plug-install'] = (el) => { const i = +el.dataset.i, m = cu.MARKET[i]; loading(el, true);
    progress('正在安装 ' + m.n + '…', m.n + ' 已安装', () => { m.inst = 1; cu.DATA.plugins.push({ n: m.n, i: 'plug', d: `官方市场 · v1.0.0 · ${m.d}`, sc: 'u', on: 1, ver: '1.0.0', src: 'claude-plugins-official', about: m.d + '。', inc: [['Skills', 6], ['斜杠命令', 2], ['子 Agent', 1]] });
      if (cc() === 'plugins') { if (st.cuTab === 'market') { $('#cu-list').innerHTML = cu.list(st); $('#cu-drawer').innerHTML = cu.drawer(st); const g = $$('.mk-card')[i]; flash(g); } $$('[data-cat-n]').forEach(n => { n.textContent = cu.DATA[n.dataset.catN].length; }); } }); };
  ACT['plug-update'] = (el) => { const r = cu.DATA.plugins[+el.dataset.i]; loading(el, true); progress('正在更新 ' + r.n + '…', r.n + ' 已更新到 ' + r.upd, () => { r.ver = r.upd; delete r.upd; r.d = r.d.replace(/v[\d.]+/, 'v' + r.ver); refreshCu(); }); };
  ACT['plug-onoff'] = (el) => { const k = +el.dataset.i, r = cu.DATA.plugins[k]; r.on = r.on ? 0 : 1; toast((r.on ? '已启用 ' : '已停用 ') + r.n, { lvl: 'info', sub: '新会话生效' }); refreshCu(); };

  /** MCP:重连 / 授权 */
  ACT['mcp-reconnect'] = (el) => { const r = cu.DATA.mcp[+el.dataset.i]; loading(el, true);
    later(1000, () => { r._try = (r._try || 0) + 1;
      if (r.st === 'err' && r._try < 2) { loading(el, false); return toast(r.n + ' 仍然连接失败', { lvl: 'err', sub: 'ECONNREFUSED 127.0.0.1:5432 · 检查命令与环境变量' }); }
      if (r.st === 'err') { r.st = 'ok'; r.d = 'stdio · 3 个工具 · 已连接'; r.tools = [['query', 1], ['list_tables', 1], ['describe_table', 1]]; r.tok = '1.1k token · 0.6%'; }
      refreshCu(); toast(r.n + ' 已连接', { lvl: 'ok', sub: '延迟 ' + (17 + r.n.length) + ' ms' }); }); };
  ACT['mcp-auth'] = (el) => { const k = el.dataset.i !== undefined ? +el.dataset.i : +el.closest('.cu-row').dataset.i, r = cu.DATA.mcp[k]; loading(el, true);
    toast('已在浏览器打开 ' + r.n + ' 的授权页', { lvl: 'info', sub: '完成授权后会自动连接' });
    later(1700, () => { r.st = 'ok'; r.d = 'http · 9 个工具 · 已连接'; r.tok = '2.4k token · 1.2%'; r.tools = [['get_design', 1], ['get_screenshot', 1], ['get_variables', 1], ['create_frame', 0]]; refreshCu(); toast(r.n + ' 授权成功', { lvl: 'ok', sub: '已连接,9 个工具可用' }); }); };
  ACT['hook-trust'] = () => { cu.DATA.hooks[st.cuSel].trust = 1; refreshCu(); toast('已信任此项目的钩子', { lvl: 'ok' }); };
  ACT['hook-test'] = (el) => { const r = cu.DATA.hooks[st.cuSel], out = $('#hk-out'); loading(el, true); out.hidden = true;
    later(700, () => { loading(el, false); const deny = /deny/.test(r.cmd); out.hidden = false; out.innerHTML = `<span class="t3">$ ${esc(r.cmd)}</span>\n${deny ? '<span class="c-err">已拦截:命令包含 rm -rf</span>\n退出码 2 · 0.04 s' : '<span class="c-ok">完成</span>\n退出码 0 · 0.31 s'}`; kick(out, 'a-rise'); }); };
  ACT['open-ide'] = (el) => { const f = el.dataset.file; if (win().dataset.mode !== 'ide' || A.st.page !== 'work') A.toWork('ide'); later(60, () => toast('已在 IDE 形态打开 ' + f, { lvl: 'info', sub: '同一个窗口,Agent 形态可随时用 ⌘E 切回' })); };

  /* ═════════ 自动化 ═════════ */
  ACT['au-sel'] = (el) => { st.auSel = +el.dataset.i; $$('.au').forEach(c => cls(c, 'on', +c.dataset.i === st.auSel)); $$('.wk-s').forEach(c => cls(c, 'sel', +c.dataset.i === st.auSel)); const d = $('#au-drawer'); d.innerHTML = au.auDrawer(st); swapAnim(d); };
  ACT['au-filter'] = (el) => { st.auF = +el.dataset.i; const l = $('#au-list'); l.innerHTML = au.auBody(st); };
  ACT['wk-nav'] = (el) => { const d = +el.dataset.d; st.wkOff = d === 0 ? 0 : (st.wkOff || 0) + d; $('#au-wkh').innerHTML = au.wkHead(st.wkOff); const w = $('#au-wk'); w.innerHTML = au.wk(st.wkOff); kick(w, 'a-swap'); A.save(); };
  ACT['au-run'] = (el) => { const k = +el.dataset.i, a = au.AUS[k], runs = $('#au-runs'); loading(el, true);
    runs.insertAdjacentHTML('afterbegin', `<div class="row a-rise" id="au-now"><span class="spin"></span><span class="grow shimmer">运行中…</span><span class="t3">刚刚</span></div>`);
    later(1700, () => { loading(el, false); a.last = '刚刚 · 成功'; a.st = 'ok'; a.runs.unshift(['刚刚', '成功', 'ok']);
      const n = $('#au-now'); if (n) { n.id = ''; n.innerHTML = `${ic('okc', 's12 c-ok')}<span class="grow">刚刚</span><span class="t3">成功</span>${ic('chevR', 's12 t4')}`; kick(n, 'a-ok'); }
      $('#au-list').innerHTML = au.auBody(st); toast('「' + a.n + '」运行完成', { lvl: 'ok', sub: '用时 1 分 12 秒', act: { label: '查看会话', fn: () => toast('已在 Agent 形态打开该运行的会话') } }); }); };
  ACT['au-del'] = (el) => { const k = +el.dataset.i, a = au.AUS[k];
    modal({ icon: 'trash', lvl: 'err', title: `删除自动化「${esc(a.n)}」?`, body: '已有的运行记录与会话会保留,但不会再按计划运行。', ok: '删除', danger: true,
      onOk: () => { const card = $$('.au').find(c => +c.dataset.i === k); if (card) { card.style.maxHeight = card.offsetHeight + 'px'; void card.offsetHeight; card.classList.add('rm'); }
        later(card ? 320 : 0, () => { au.AUS.splice(k, 1); Object.keys(au.SLOTS).forEach(d => { au.SLOTS[d] = au.SLOTS[d].filter(s => s[1] !== k).map(s => s[1] > k ? [s[0], s[1] - 1, s[2]] : s); }); st.auSel = Math.max(0, Math.min(st.auSel, au.AUS.length - 1)); A.rerender(); });
        toast('已删除「' + a.n + '」', { lvl: 'info' }); } }); };
  ACT['au-new'] = () => modal({ icon: 'timer', title: '新建自动化', wide: true, ok: '创建',
    form: `<div class="fgrp"><span class="lbl">名称</span>${U.field('', 0, { id: 'aun-n', ph: '例如:每日构建检查', attr: 'style="width:100%"' })}</div><div class="fgrp"><span class="lbl">提示词</span>${U.area('', { id: 'aun-p', rows: 3, ph: '要让 Claude 做什么?' })}</div><div class="fgrp"><span class="lbl">频率</span>${U.seg(['每天', '工作日', '每周一'], 1)}</div><div class="fgrp"><span class="lbl">时间</span>${U.field('10:00', 80, { id: 'aun-t' })}</div>`,
    onOk: (m) => { const n = $('#aun-n', m), p = $('#aun-p', m), t = $('#aun-t', m); [n, p, t].forEach(clearErr); let bad = 0;
      if (!/^([01]?\d|2[0-3]):[0-5]\d$/.test(t.value.trim())) { errField(t, '时间格式如 09:30'); bad = 1; } if (!p.value.trim()) { errField(p, '请写下要做什么'); bad = 1; } if (!n.value.trim()) { errField(n, '请填写名称'); bad = 1; } if (bad) return false;
      const f = segIdx($('.seg', m)), days = [[0, 1, 2, 3, 4, 5, 6], [0, 1, 2, 3, 4], [0]][f], lab = ['每天', '工作日', '每周一'][f], i = au.AUS.length;
      au.AUS.push({ n: n.value.trim(), sch: lab + ' · ' + t.value.trim(), proj: 'router-demo', last: '尚未运行', st: '', on: 1, prompt: p.value.trim(), kind: f === 2 ? 1 : 0, day: lab, time: t.value.trim(), perm: '只读 · 计划模式', runs: [] });
      days.forEach(d => au.SLOTS[d].push([t.value.trim(), i])); st.auSel = i; st.auF = 0; A.rerender();
      later(50, () => { const c = $$('.au').find(x => +x.dataset.i === i); if (c) { c.scrollIntoView({ block: 'nearest', behavior: 'smooth' }); flash(c); } }); toast('已创建「' + n.value.trim() + '」', { lvl: 'ok', sub: lab + ' ' + t.value.trim() + ' 运行' }); } });

  /* ═════════ 画布 ═════════ */
  ACT['cv-mode'] = (el) => { const edit = +el.dataset.i === 0, d = $('#cv-doc'); d.contentEditable = edit; d.classList.toggle('ro', !edit); toast(edit ? '编辑模式:直接在正文里改' : '预览模式:只读', { ms: 1800 }); };
  ACT['cv-outline'] = (el) => { const t = $('#' + el.dataset.id), sc = $('#cv-scroll'); if (!t) return; sc.scrollTo({ top: t.offsetTop - 16, behavior: 'smooth' }); $$('.cv-side .row').forEach(r => cls(r, 'on', r === el)); flash(t); };
  ACT['cv-focus'] = () => { const i = $('[data-in="cv-q"]'); i.focus(); flash(i.closest('.cmp')); };
  const pushVer = (who, desc) => { const V = au.VERS, n = 'v' + (V.length + 1); V.unshift([n, '刚刚', who + ' · ' + desc]); V[0][3] = 1; $('#cv-vers').innerHTML = au.versRows(); kick($('#cv-vers').firstElementChild, 'a-rise'); $('#cv-vn').innerHTML = U.cnt(V.length); $('#cv-ver').textContent = n + ' · 刚刚由' + who + '修改'; return n; };
  ACT['cv-rewind'] = (el) => { const v = el.dataset.v, n = 'v' + (au.VERS.length + 1);
    modal({ icon: 'rewind', title: `回到 ${v}?`, body: `当前内容会先保存为新版本 ${n},不会丢失;之后可以再回到任何一个版本。`, ok: '回到此版本', onOk: () => { pushVer('你', '回到 ' + v); flash($('#cv-doc')); toast('已回到 ' + v, { lvl: 'ok', sub: '生成了新版本 ' + n }); } }); };
  ACT['cv-send'] = (el) => { const i = $('[data-in="cv-q"]'), q = i.value.trim(); if (!q) return shake(i.closest('.cmp')); loading(el, true); i.value = ''; toast('已发给 Claude', { sub: q });
    later(1500, () => { loading(el, false); pushVer('Claude', q.length > 14 ? q.slice(0, 14) + '…' : q); flash($('#cv-doc')); toast('画布已更新', { lvl: 'ok' }); }); };

  /* ═════════ 浏览器 ═════════ */
  ACT['br-tab'] = (el) => { $$('.tab', el.parentElement).forEach(t => cls(t, 'on', t === el)); const p = $('#br-pane'); p.innerHTML = au.BRP[el.dataset.tab](); kick(p, 'a-swap'); };
  ACT['br-ask'] = (el) => { const i = $('[data-in="br-q"]'); i.value = el.dataset.q; i.focus(); flash(i.closest('.cmp')); };
  ACT['br-send'] = (el) => { const i = $('[data-in="br-q"]'), q = i.value.trim(); if (!q) return shake(i.closest('.cmp')); i.value = ''; $$('.drawer .tab').forEach(t => cls(t, 'on', false));
    const p = $('#br-pane'), ans = 'Router 把路径模式映射到处理器:先匹配字面量,再匹配 :id 这样的捕获参数,最后是通配。路径区分大小写,建议在中间件里统一归一化。';
    p.innerHTML = `<div class="fc g3"><div class="notice info">${ic('user', 's12 c-info')}<div class="grow">${esc(q)}</div></div><div class="md"><p class="caret"></p></div></div>`; kick(p, 'a-swap'); const el2 = $('.caret', p); let k = 0;
    const iv = setInterval(() => { if (!el2.isConnected) return clearInterval(iv); k += 3; el2.textContent = ans.slice(0, k); if (k >= ans.length) { clearInterval(iv); el2.classList.remove('caret'); } }, 24); };

  /* ═════════ 用量 ═════════ */
  ACT['use-range'] = (el) => { st.useR = +el.dataset.i; refreshPage(); A.save(); };
  ACT['lim-refresh'] = (el) => { const at = $('#lim-at'), svg = $('svg', el); svg.style.animation = 'spin .8s linear infinite'; at.textContent = '更新中…'; at.classList.add('shimmer');
    later(1000, () => { svg.style.animation = ''; at.classList.remove('shimmer'); at.textContent = '更新于 刚刚'; const m = $('#lim-box .meter i'); if (m) { m.style.width = '39%'; $('#lim-box .num').textContent = '39%'; } }); };

  /* ═════════ 回车提交、渲染后的钩子 ═════════ */
  document.addEventListener('keydown', (e) => { if (e.key !== 'Enter' || e.isComposing || e.shiftKey) return; const t = e.target; if (!t.matches || !t.matches('input[data-in]')) return;
    const box = t.closest('.dlg, .set-row, .cmp'); const b = box && box.querySelector('[data-act="modal-ok"], [data-act="rule-add"], [data-act="env-add"], [data-act="prov-save"], .send'); if (b && t.dataset.in !== 'setq' && t.dataset.in !== 'cuq' && t.dataset.in !== 'kbq') { e.preventDefault(); b.click(); } });
  /** 画布大纲:正文滚动时高亮当前所在的标题 */
  const wireCanvas = () => { const sc = $('#cv-scroll'); if (!sc) return; let raf = 0;
    sc.addEventListener('scroll', () => { cancelAnimationFrame(raf); raf = requestAnimationFrame(() => { const hs = $$('#cv-doc [id^="h-"]'); let cur = hs[0]; hs.forEach(h => { if (h.offsetTop - 40 <= sc.scrollTop) cur = h; }); $$('.cv-side .row[data-id]').forEach(r => cls(r, 'on', r.dataset.id === cur.id)); }); }); };
  A.hooks.push(() => { const w = win(); if (toastBox && w && !toastBox.isConnected) w.appendChild(toastBox); applyPrefs(); initRows(); wireCanvas(); });
  A.renderMain();
})();
