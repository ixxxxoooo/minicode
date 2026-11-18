/* 设计稿 v3.5 · 用户消息吸顶 + 悬浮编辑卡 + 图片灯箱
   结构:会话流 = 一串「回合」.round(用户提问 + 它引出的步骤与回复)。提问卡 .msg-u 在自己的回合里 position: sticky,
   所以:① 滚到回合顶部就固定;② 回合滚完、下一回合顶上来时,上一张随回合一起被顶走;③ 任何时刻最多固定一张。
   点击提问卡(固定或没固定都一样)→ 在卡片原位置向下展开「悬浮编辑卡」(不占位、不推挤会话流);可同时编辑多条,后打开的盖在上面。
   消息里的 [Image #N] 占位在会话流与编辑卡里都画成行内「图片 N」胶囊(与输入框同款,点开灯箱);输入框发送时把图片位置胶囊展开成 [Image #N];Enter 发送 / Shift+Enter 换行 / Esc 或点外收起(未发送丢弃);编辑卡与输入框行为一致:可 ⌘V 粘贴图片(缩略图进上方一排、正文光标处插「图片 N」位置胶囊),缩略图右上角可移除(正文里它的胶囊同时去掉),删正文里的胶囊也会移除对应图片,点缩略图 / 胶囊开灯箱;重发时保留这些图片。
   图片缩略图点击 → 全屏灯箱(遮罩、缩放、平移、多图左右切换)。
   设置:对话与提示词 →「用户消息吸顶」默认开(stickyUser);关闭后 .no-sticky-user 让卡片回到普通流(编辑照常)。
   F.stickyDemo(o) 可滚动的多回合演示;F.lightboxDemo() 灯箱静态原型。
   @author ixxxxoooo */
(function () {
  const $$ = (sel, el = document) => [...el.querySelectorAll(sel)];
  const LONG = '请按下面的清单逐项处理:<br>1. 把 normalize 抽到单独的模块<br>2. 补 4 条大小写用例<br>3. 修复 CI 里失败的 clippy 警告<br>4. 更新 README 的路由示例<br>5. 把 fallback 逻辑改成显式配置<br>6. 清理未使用的依赖<br>7. 补充发布说明';
  const round = (q, long) => `<div class="round">${long ? `<div class="msg-u clamp">${q}<div class="fade"></div></div>` : F.user(q)}
    ${F.step('read', 'router.rs', 'src/ · L1–86', '86 行')}${F.step('grep', '"match_path"', 'src/', '5 处')}${F.step('edit', 'router.rs', 'src/', '<span class="c-ok">+4</span><span class="c-err">−2</span>')}
    <div class="md"><p>已处理完这一步:大小写归一化放在 <code class="mono">normalize()</code> 里,匹配前统一调用;原来的路由表不受影响。</p><p>另外补了对应的单元测试,覆盖 <code class="mono">/Users</code> 与 <code class="mono">/users</code> 命中同一个处理器。</p></div></div>`;
  F.stickyDemo = (o = {}) => `<div class="sdemo"><div class="flow-wrap" data-sticky-init="${o.init == null ? 330 : o.init}" ${o.pinedit ? 'data-sticky-pinedit="1"' : ''} style="height:${o.h || 300}px"><div class="col"><div class="flow">
    ${round(`看一下 ${F.ref('router.rs')},路由匹配对大小写敏感,修好并补测试。`)}
    ${round(o.long ? LONG : '再把 normalize 抽到单独的模块里,并补 4 条用例。', o.long)}
    ${round('顺便把 README 里的路由示例也更新一下。')}</div></div></div></div>`;

  F.lightboxDemo = () => `<div class="lb" style="position:relative;height:420px;border-radius:var(--r-lg);overflow:hidden;background:rgba(0,0,0,.88)">
    <div class="f ai" style="height:48px;padding:0 16px"><span class="grow mono fs-body" style="color:rgba(255,255,255,.85)">screenshot-error.png · 2 / 3</span>${U.ib('x', 'sm', '关闭')}</div>
    <div class="f jc ai grow" style="min-height:280px"><div class="card" style="width:72%;max-height:72%;background:var(--bg-sunken);aspect-ratio:16/10;border-radius:var(--r-md)"></div></div>
    <div class="f jc" style="padding-bottom:24px"><div class="f ai g2" style="padding:8px 12px;border-radius:var(--r-lg);background:rgba(0,0,0,.6);border:1px solid rgba(255,255,255,.12)">
      ${U.ib('minus', 'sm', '缩小')}${U.btn('78%', 'sm ghost', '')}${U.ib('plus', 'sm', '放大')}${U.ib('max', 'sm', '适应')}${U.ib('ext', 'sm', '打开')}${U.ib('copy', 'sm', '复制路径')}
    </div></div>
    <div class="t4 fs-micro" style="position:absolute;left:16px;bottom:8px;color:rgba(255,255,255,.45)">Esc 关闭 · 滚轮缩放 · 拖动平移 · ← → 切换</motion-div>`;

  document.addEventListener('DOMContentLoaded', () => {
    const A = window.APP, { ic, btn } = U, esc = (t) => t.replace(/[&<>]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' })[c]);
    const wrapOf = (m) => m.closest('.flow-wrap');
    const mark = (wrap) => {
      const top = wrap.getBoundingClientRect().top, on = A.st.prefs.stickyUser !== 0 && !wrap.closest('.no-sticky-user');
      $$('.flow .msg-u', wrap).forEach(m => { const r = (m.closest('.round') || m).getBoundingClientRect();
        m.classList.toggle('stuck', on && wrap.scrollTop > 2 && m.getBoundingClientRect().top - top < 1.5 && r.bottom > top + 1); });
    };
    let raf = 0;
    document.addEventListener('scroll', (e) => { const w = e.target; if (!w || !w.classList || !w.classList.contains('flow-wrap') || raf) return;
      raf = requestAnimationFrame(() => { raf = 0; mark(w); }); }, true);

    const pinned = new Map();
    const plain = (m) => { const c = m.cloneNode(true); $$('.fade,.pin-edit', c).forEach(x => x.remove()); $$('.ref', c).forEach(r => r.replaceWith('@' + r.textContent.trim())); c.innerHTML = c.innerHTML.replace(/<br\s*\/?>/g, '\n'); return c.textContent.trim(); };
    const fit = (ta) => { ta.style.height = 'auto'; ta.style.height = Math.min(ta.scrollHeight, 220) + 'px'; };
    const closePin = (id) => { const el = pinned.get(id); if (el) { el.remove(); pinned.delete(id); } };
    const closeAllPin = () => { pinned.forEach((el) => el.remove()); pinned.clear(); };
    const openPin = (m) => {
      const id = m.dataset.uid || (m.dataset.uid = 'u' + Math.random().toString(36).slice(2, 7));
      if (pinned.has(id)) { closePin(id); return; }
      const r = m.getBoundingClientRect(), w = m.closest('.flow-wrap'), wt = w.getBoundingClientRect();
      const top = m.classList.contains('stuck') ? wt.top + 8 : r.top;
      const layer = document.createElement('div');
      layer.className = 'pin-edit a-pop';
      layer.style.cssText = `position:fixed;left:${r.left}px;top:${top}px;width:${r.width}px;z-index:30`;
      layer.innerHTML = `<textarea class="ed" spellcheck="false">${esc(plain(m))}</textarea><div class="thumbs f g2" style="margin-bottom:6px">${ic('image', 's12')}<span class="t3 fs-cap">附件只读 · 点击预览</span></div><div class="ed-acts"><span class="t3 fs-cap grow">Enter 发送 · Shift+Enter 换行</span>${btn('复制', 'sm ghost')}${btn('取消', 'sm ghost')}${btn('发送', 'sm pri')}</div>`;
      document.body.appendChild(layer); pinned.set(id, layer);
      const ta = layer.querySelector('textarea'); fit(ta); ta.focus(); ta.setSelectionRange(ta.value.length, ta.value.length);
    };
    document.addEventListener('click', (e) => { const m = e.target.closest && e.target.closest('.flow .msg-u'); if (!m || e.target.closest('a, button, .ref, .pin-edit')) return;
      if (!e.target.closest('.thumbs')) openPin(m); });
    document.addEventListener('mousedown', (e) => { if (!e.target.closest('.pin-edit')) closeAllPin(); });
    document.addEventListener('keydown', (e) => { if (e.key === 'Escape') closeAllPin(); }, true);
    document.addEventListener('keydown', (e) => { const ta = e.target.closest && e.target.closest('.pin-edit textarea'); if (!ta) return;
      if (e.key === 'Escape') { e.preventDefault(); closeAllPin(); }
      else if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) { e.preventDefault(); const pin = ta.closest('.pin-edit'); closeAllPin(); if (pin) pin.dataset.sent = '1'; } }, true);
    document.addEventListener('input', (e) => { if (e.target.matches && e.target.matches('.pin-edit textarea')) fit(e.target); });

    const init = () => $$('.flow-wrap').forEach(w => {
      const v = w.dataset.stickyInit;
      if (v) { w.removeAttribute('data-sticky-init');
        if (v === 'push') { const r = w.querySelectorAll('.round')[1]; w.scrollTop += r.getBoundingClientRect().bottom - w.getBoundingClientRect().top - 24; } else w.scrollTop = +v; }
      mark(w);
      if (w.dataset.stickyPinedit) { w.removeAttribute('data-sticky-pinedit'); const m = w.querySelector('.msg-u.stuck') || w.querySelector('.msg-u'); if (m) openPin(m); } });
    A.hooks.push(init); init();
    A.stickyChanged = () => $$('.flow-wrap').forEach(mark);
  });
})();