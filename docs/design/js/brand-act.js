/* 设计稿 v3.3 · 品牌与图标的交互层
   点样式卡 / 外观分段 / 配色 / 开关 → 写进查看器状态(随 localStorage 记住),并让页面里所有 [data-bi] 图标原地重绘;
   换主题时「跟随主题」的图标同步重绘;设置里「还原」一行时把该行的值同步回状态。
   @author ixxxxoooo */
(function () {
  const A = window.APP, st = A.st, ACT = A.ACT, B = U.brand, { $$ } = A;
  const ICO_KEYS = ['icoStyle', 'icoMode', 'icoTint'];

  /** 选项变了:重绘所有图标;应用图标相关的还要刷新顶栏小标 */
  const changed = (key) => { B.refresh(); liveTick(); if (ICO_KEYS.includes(key)) A.renderBar(); A.save(); };

  /* 动态预览(设置 → 图标、规范页):没固定状态时每 2.8 秒换一个托盘状态,三处(Dock / 浅 / 深菜单栏)同步播放;点状态胶囊可固定,再点「自动循环」恢复 */
  let liveIdx = 0;
  function liveTick(step) {
    const ids = B.TRAY_STATES.map(x => x[0]);
    if (step) liveIdx = (liveIdx + 1) % ids.length;
    const id = st.brandPin || ids[liveIdx];
    $$('[data-blv]').forEach(el => { el.innerHTML = B.liveInner(id); });
  }
  setInterval(() => liveTick(!st.brandPin), 2800);
  ACT['brand-pin'] = (el) => { st.brandPin = el.dataset.v || ''; liveTick(); A.save(); };
  const sel = (sel, el, attr) => $$(sel).forEach(c => { const on = c.dataset[attr] === el.dataset[attr]; c.classList.toggle('on', on); c.setAttribute('aria-checked', on); });

  ACT['ico-style'] = (el) => { st.icoStyle = el.dataset.v; sel('[data-act="ico-style"]', el, 'v'); changed('icoStyle'); };
  ACT['ico-tint'] = (el) => { st.icoTint = el.dataset.v; sel('[data-act="ico-tint"]', el, 'v'); changed('icoTint'); };
  /** 分段:data-ikey 在容器上,取值查 OPT 表 */
  ACT['ico-opt'] = (el) => { const key = el.parentElement.dataset.ikey; if (!key) return; st[key] = B.OPT[key][+el.dataset.i]; changed(key); };

  /* 开关:app.js 的 tog 会在 data-live 存在时调 ACT.live,这里接住 data-live="ico",其余交还给原处理 */
  const live = ACT.live;
  ACT.live = (el) => { if (el.dataset.live !== 'ico') return live && live(el); const key = el.dataset.ikey; st[key] = el.classList.contains('on') ? 1 : 0; changed(key); };

  /* 换主题:「配色 = 跟随主题」与「外观 = 自动」的图标要跟着变 */
  const theme = ACT.theme;
  ACT.theme = (el, ...r) => { theme(el, ...r); B.refresh(); A.renderBar(); };

  /* 设置里「还原」某一行:行内控件的值同步回状态(原 syncLive 只认 data-key) */
  const reset = ACT['set-reset'];
  ACT['set-reset'] = (el, ...r) => { reset(el, ...r); const row = el.closest('.set-row'); if (!row) return;
    $$('[data-ikey]', row).forEach(c => { const key = c.dataset.ikey;
      if (c.classList.contains('seg')) st[key] = B.OPT[key][+(c.style.getPropertyValue('--i') || 0)]; else if (c.classList.contains('tog')) st[key] = c.classList.contains('on') ? 1 : 0; });
    B.refresh(); A.renderBar(); A.save(); };
})();
