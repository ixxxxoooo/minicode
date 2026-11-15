/* 设计稿 v3.4 · 工作区起始页与多个终端的交互(构建函数见 wsempty.js)
   起始页终端行 / 底部面板标签共用 APP.st.terms(个数,默认 1)与 APP.st.termSel(当前);任一处改动,两处一起重画。
   @author ixxxxoooo */
(function () {
  const A = window.APP, ACT = A.ACT, { $ } = A;
  /** 状态变了:起始页(若正显示)与底部面板一起重画;focus = true 时展开面板并闪一下 */
  const sync = (w, focus) => { if (!w) return; const e = $('.ws-empty2', w); if (e) e.outerHTML = WS.empty();
    const cur = $('.bpanel', w), pk = ($('.ws-in', w) || { dataset: {} }).dataset.pk || 'term';
    if (cur && pk === 'term') { const t = document.createElement('div'); t.innerHTML = IDE.bpanel('term'); cur.replaceWith(t.firstElementChild); }
    if (focus) { w.dataset.panel = 'on'; A.syncBars(w); const p = $('.bpanel', w); if (p) { p.classList.remove('a-flash'); void p.offsetWidth; p.classList.add('a-flash'); }
      if (w.dataset.mode === 'agent' && w.dataset.right === 'off') { w.dataset.right = 'on'; A.syncBars(w); } } };
  const W0 = () => $('#vmain .win');
  const pick = (i, focus) => { A.st.termSel = i; sync(W0(), focus); };
  ACT['we-term'] = (el) => pick(+el.dataset.i, true);
  ACT['term-pick'] = (el) => pick(+el.dataset.i, false);
  ACT['we-term-new'] = ACT['term-new'] = (el) => { const n = A.st.terms == null ? 1 : A.st.terms; A.st.terms = n + 1; A.st.termSel = n; sync(W0(), el.dataset.act === 'we-term-new'); };
  ACT['term-close'] = (el) => { const n = A.st.terms || 1; if (n <= 1) return; A.st.terms = n - 1; A.st.termSel = Math.min(+el.dataset.i, A.st.terms - 1); sync(W0(), false); };
  ACT['we-open'] = (el, w) => A.openTab(w, el.dataset.tab);
})();
