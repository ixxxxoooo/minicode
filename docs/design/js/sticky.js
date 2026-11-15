/* 设计稿 v3.4 · 用户消息吸顶
   会话流里滚到「这一轮用户提问」的上沿时,这张卡固定在会话流顶部;下一条提问滚到顶部时从下方盖住并接替它。
   实现:用户消息 .msg-u 本身 position: sticky(见 css/flow.css),零 JS 也能固定;JS 只负责给「正固定在顶部」的卡加 .stuck(阴影 + 下缘渐隐带)。
   设置:对话与提示词 → 显示 → 「用户消息吸顶」,默认开,偏好键 stickyUser,关闭后 body.no-sticky-user 让卡片回到普通流。
   F.stickyDemo(o) 给规范页一个可滚动的多轮会话:o.init 初始滚动的行数、o.h 高度、o.long 第二条提问用长消息。
   @author ixxxxoooo */
(function () {
  const $$ = (sel, el = document) => [...el.querySelectorAll(sel)];
  const LONG = '请按下面的清单逐项处理:<br>1. 把 normalize 抽到单独的模块<br>2. 补 4 条大小写用例<br>3. 修复 CI 里失败的 clippy 警告<br>4. 更新 README 的路由示例<br>5. 把 fallback 逻辑改成显式配置<br>6. 清理未使用的依赖<br>7. 补充发布说明';
  /** 一轮:用户提问 + 一段助手回复(几行步骤 + 正文),足够把会话撑出滚动 */
  const round = (q, long) => `${long ? `<div class="msg-u clamp">${q}<div class="fade"></div></div>` : F.user(q)}
    ${F.step('read', 'router.rs', 'src/ · L1–86', '86 行')}${F.step('grep', '"match_path"', 'src/', '5 处')}${F.step('edit', 'router.rs', 'src/', '<span class="c-ok">+4</span><span class="c-err">−2</span>')}
    <div class="md"><p>已处理完这一步:大小写归一化放在 <code class="mono">normalize()</code> 里,匹配前统一调用;原来的路由表不受影响。</p><p>另外补了对应的单元测试,覆盖 <code class="mono">/Users</code> 与 <code class="mono">/users</code> 命中同一个处理器。</p></div>`;
  F.stickyDemo = (o = {}) => `<div class="sdemo"><div class="flow-wrap" data-sticky-init="${o.init == null ? 330 : o.init}" style="height:${o.h || 300}px"><div class="col"><div class="flow">
    ${round(`看一下 ${F.ref('router.rs')},路由匹配对大小写敏感,修好并补测试。`)}
    ${round(o.long ? LONG : '再把 normalize 抽到单独的模块里,并补 4 条用例。', o.long)}
    ${round('顺便把 README 里的路由示例也更新一下。')}</div></div></div></div>`;

  /* 运行部分依赖 APP(app.js 之后才有),等所有脚本加载完再挂 */
  document.addEventListener('DOMContentLoaded', () => {
    const A = window.APP;
    /** 给一个会话流容器标出正固定在顶部的用户消息 */
    const mark = (wrap) => {
      const top = wrap.getBoundingClientRect().top, on = A.st.prefs.stickyUser !== 0 && !wrap.closest('.no-sticky-user');
      $$('.flow .msg-u', wrap).forEach(m => m.classList.toggle('stuck', on && wrap.scrollTop > 2 && m.getBoundingClientRect().top - top < 1.5));
    };
    let raf = 0;
    document.addEventListener('scroll', (e) => { const w = e.target; if (!w || !w.classList || !w.classList.contains('flow-wrap') || raf) return;
      raf = requestAnimationFrame(() => { raf = 0; mark(w); }); }, true);
    /** 整页渲染后:示例容器滚到初始位置,所有容器重新标记 */
    const init = () => $$('.flow-wrap').forEach(w => { if (w.dataset.stickyInit) { w.scrollTop = +w.dataset.stickyInit; w.removeAttribute('data-sticky-init'); } mark(w); });
    A.hooks.push(init); init();   /* 首次渲染早于本脚本挂钩,所以挂完立即跑一次 */
    /** 设置开关变化时(behave.js 的 applyPrefs 调用) */
    A.stickyChanged = () => $$('.flow-wrap').forEach(mark);
  });
})();
