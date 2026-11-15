/* 设计稿 v3.4 · 用户消息吸顶 + 就地编辑(对标 Cursor)
   结构:会话流 = 一串「回合」.round(用户提问 + 它引出的步骤与回复)。提问卡 .msg-u 在自己的回合里 position: sticky,
   所以:① 滚到回合顶部就固定;② 回合滚完、下一回合顶上来时,上一张随回合一起被顶走(不是盖住);③ 任何时刻最多固定一张。
   全程是浏览器合成层的滚动,没有 JS 参与位置计算,所以丝滑;JS 只给「正固定」的卡加 .stuck(阴影 + 下缘渐隐,150ms 过渡)。
   点击提问卡(固定或没固定都一样)就地编辑:卡片以高度动画展开成编辑态,「取消 / 发送」;点外部 / Esc / 取消都关闭,⌘↩ 或 ↩ 发送。
   设置:对话与提示词 → 显示 →「用户消息吸顶」默认开,偏好键 stickyUser,关闭后 .no-sticky-user 让卡片回到普通流(编辑照常)。
   F.stickyDemo(o) 给规范页一个可滚动的多回合会话:o.init 初始滚动像素或 'push'(停在「上一张正被顶走」)、o.h 高度、o.long 第二条用长消息、o.edit 初始就在编辑固定中的那张。
   @author ixxxxoooo */
(function () {
  const $$ = (sel, el = document) => [...el.querySelectorAll(sel)];
  const LONG = '请按下面的清单逐项处理:<br>1. 把 normalize 抽到单独的模块<br>2. 补 4 条大小写用例<br>3. 修复 CI 里失败的 clippy 警告<br>4. 更新 README 的路由示例<br>5. 把 fallback 逻辑改成显式配置<br>6. 清理未使用的依赖<br>7. 补充发布说明';
  /** 一个回合:提问 + 几行步骤 + 一段回复,足够把会话撑出滚动 */
  const round = (q, long) => `<div class="round">${long ? `<div class="msg-u clamp">${q}<div class="fade"></div></div>` : F.user(q)}
    ${F.step('read', 'router.rs', 'src/ · L1–86', '86 行')}${F.step('grep', '"match_path"', 'src/', '5 处')}${F.step('edit', 'router.rs', 'src/', '<span class="c-ok">+4</span><span class="c-err">−2</span>')}
    <div class="md"><p>已处理完这一步:大小写归一化放在 <code class="mono">normalize()</code> 里,匹配前统一调用;原来的路由表不受影响。</p><p>另外补了对应的单元测试,覆盖 <code class="mono">/Users</code> 与 <code class="mono">/users</code> 命中同一个处理器。</p></div></div>`;
  F.stickyDemo = (o = {}) => `<div class="sdemo"><div class="flow-wrap" data-sticky-init="${o.init == null ? 330 : o.init}" ${o.edit ? 'data-sticky-edit="1"' : ''} style="height:${o.h || 300}px"><div class="col"><div class="flow">
    ${round(`看一下 ${F.ref('router.rs')},路由匹配对大小写敏感,修好并补测试。`)}
    ${round(o.long ? LONG : '再把 normalize 抽到单独的模块里,并补 4 条用例。', o.long)}
    ${round('顺便把 README 里的路由示例也更新一下。')}</div></div></div></div>`;

  document.addEventListener('DOMContentLoaded', () => {   /* 运行部分依赖 APP(app.js 之后才有) */
    const A = window.APP, { ic, btn } = U, esc = (t) => t.replace(/[&<>]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' })[c]);
    const wrapOf = (m) => m.closest('.flow-wrap');

    /* ── 吸顶:标出正固定在顶部的卡 ── */
    const mark = (wrap) => {
      const top = wrap.getBoundingClientRect().top, on = A.st.prefs.stickyUser !== 0 && !wrap.closest('.no-sticky-user');
      $$('.flow .msg-u', wrap).forEach(m => { const r = (m.closest('.round') || m).getBoundingClientRect();   /* 回合还有一部分在顶边之下才算固定,滚出去的上一回合不算 */
        m.classList.toggle('stuck', on && wrap.scrollTop > 2 && m.getBoundingClientRect().top - top < 1.5 && r.bottom > top + 1); });
    };
    let raf = 0;
    document.addEventListener('scroll', (e) => { const w = e.target; if (!w || !w.classList || !w.classList.contains('flow-wrap') || raf) return;
      raf = requestAnimationFrame(() => { raf = 0; mark(w); }); }, true);

    /* ── 就地编辑 ── */
    /** 高度过渡:先记旧高度,改完 DOM 量新高度,再让浏览器补间(结束后还原成自动高度) */
    const morph = (m, change) => {
      const h0 = m.offsetHeight; change(); const h1 = m.offsetHeight;
      if (h0 === h1) return; m.classList.add('morph'); m.style.height = h0 + 'px'; void m.offsetHeight; m.style.height = h1 + 'px';
      clearTimeout(m._mt); m._mt = setTimeout(() => { m.classList.remove('morph'); m.style.height = ''; }, 230);
    };
    const editing = () => document.querySelector('.flow .msg-u.editing');
    const fit = (ta) => { ta.style.height = 'auto'; ta.style.height = Math.min(ta.scrollHeight, 220) + 'px'; };
    /** 卡片文字(换行保留,引用写成 @名字) */
    const plain = (m) => { const c = m.cloneNode(true); $$('.fade', c).forEach(x => x.remove()); $$('.ref', c).forEach(r => r.replaceWith('@' + r.textContent.trim())); c.innerHTML = c.innerHTML.replace(/<br\s*\/?>/g, '\n'); return c.textContent.trim(); };
    const startEdit = (m, instant) => {
      const cur = editing(); if (cur === m) return; if (cur) cancelEdit(true);
      m._orig = m.innerHTML; m._cls = m.className;
      const go = () => { m.classList.add('editing'); m.classList.remove('clamp');
        m.innerHTML = `<textarea class="ed" rows="1" spellcheck="false">${esc(plain(m))}</textarea><div class="ed-acts"><span class="t3 fs-cap grow">发送后会回退这一轮之后的文件改动</span>${btn('取消', 'sm ghost', '', 'data-act="su-cancel"')}${btn('发送', 'sm pri', '', 'data-act="su-send"')}</div>`;
        const ta = m.querySelector('textarea'); fit(ta); if (!instant) { ta.focus(); ta.setSelectionRange(ta.value.length, ta.value.length); } };
      instant ? go() : morph(m, go);
    };
    const cancelEdit = (instant) => { const m = editing(); if (!m) return false;
      const back = () => { m.className = m._cls.replace(/\bediting\b/g, '').trim(); m.innerHTML = m._orig; };
      instant ? back() : morph(m, back); mark(wrapOf(m)); return true; };
    const send = (m) => { const ta = m.querySelector('textarea'), v = ta.value.trim();
      if (!v) { m.classList.remove('a-shake'); void m.offsetWidth; m.classList.add('a-shake'); return; }
      morph(m, () => { m.className = m._cls.replace(/\bediting\b|\bclamp\b/g, '').trim(); m.innerHTML = esc(v).replace(/\n/g, '<br>'); });
      m.classList.add('a-flash'); const r = m.closest('.round') || m.parentElement;
      $$('.su-note', r).forEach(n => n.remove());
      m.insertAdjacentHTML('afterend', `<div class="su-note a-rise">${F.note('refresh', '已回退这一轮之后的改动,正在重新发送…')}</div>`);
      setTimeout(() => $$('.su-note', r).forEach(n => n.remove()), 2600); mark(wrapOf(m)); };
    A.ACT['su-cancel'] = () => cancelEdit();
    A.ACT['su-send'] = (el) => send(el.closest('.msg-u'));
    document.addEventListener('click', (e) => { const m = e.target.closest && e.target.closest('.flow .msg-u'); if (!m || m.classList.contains('editing') || e.target.closest('a, button, .ref')) return; startEdit(m); });
    document.addEventListener('mousedown', (e) => { const m = editing(); if (m && !m.contains(e.target)) cancelEdit(); });
    document.addEventListener('keydown', (e) => { const ta = e.target.closest && e.target.closest('.msg-u.editing textarea'); if (!ta) return;
      if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); cancelEdit(); }
      else if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) { e.preventDefault(); send(ta.closest('.msg-u')); } }, true);
    document.addEventListener('input', (e) => { if (e.target.matches && e.target.matches('.msg-u.editing textarea')) fit(e.target); });
    const prev = A.closeLayer; A.closeLayer = (...a) => cancelEdit() || (prev ? prev(...a) : false);

    /* ── 初始化:示例容器滚到指定位置;首次渲染早于本脚本,所以挂完钩子立即跑一次 ── */
    const init = () => $$('.flow-wrap').forEach(w => {
      const v = w.dataset.stickyInit;
      if (v) { w.removeAttribute('data-sticky-init');
        if (v === 'push') { const r = w.querySelectorAll('.round')[1]; w.scrollTop += r.getBoundingClientRect().bottom - w.getBoundingClientRect().top - 24; } else w.scrollTop = +v; }
      mark(w);
      if (w.dataset.stickyEdit) { w.removeAttribute('data-sticky-edit'); const m = w.querySelector('.msg-u.stuck'); if (m) startEdit(m, true); } });
    A.hooks.push(init); init();
    A.stickyChanged = () => $$('.flow-wrap').forEach(mark);   /* 设置开关变化(behave.js 的 applyPrefs 调用) */
  });
})();
