/* 设计稿 v3.4 · 工作区起始页(没有任何标签时)+ 多个终端的交互
   起始页不再是 8 张等权重的卡,而是三组左对齐的列表(同设置页的分组卡风格):
     本轮 —— 审查 / 计划 / 子 Agent 与任务,只列「有内容」的;都没有就一行灰字,不放置灰掉的入口
     终端 —— 每个已开的终端一行(点 = 切到它并展开底部面板),末尾「新建终端」;一个都没开时只有「打开终端」一行
     打开 —— 文件 / 源代码管理 / 搜索 / 命令面板(都只是跳转,右侧是快捷键)
   终端的「家」仍是底部面板:标签条 zsh · zsh 2 · …(+ 新建、选中的带 × 关闭);起始页的终端行与面板标签是同一份状态(APP.st.terms / termSel)。
   本文件只有构建函数(规范页加载时就要用,所以排在 ws.js 之后);交互在 wsempty-act.js。WS.empty(o) / IDE.termChips(o):o.terms 终端个数、o.sel 当前、o.files / o.plan / o.tasks 本轮内容。
   @author ixxxxoooo */
(function () {
  const { ic, kbd } = U, S = () => (window.APP && window.APP.st) || {};
  const name = (i) => i ? 'zsh ' + (i + 1) : 'zsh';
  const row = (icon, t, sub, right, attr = '', off = false) => `<div class="we-row ${off ? 'off' : ''}" ${off ? '' : `tabindex="0" ${attr}`}>${ic(icon, 's16')}<span class="m">${t}</span><span class="t3 fs-cap grow ell">${sub}</span>${right}</div>`;
  const go = ic('chevR', 's12 t4');

  /** 起始页内容。默认:2 个待审查文件、1 份计划、1 个运行中的任务、1 个终端 */
  WS.empty = (o = {}) => {
    const files = o.files == null ? 2 : o.files, plan = o.plan == null ? 1 : o.plan, tasks = o.tasks == null ? 1 : o.tasks;
    const n = o.terms == null ? (S().terms || 1) : o.terms, sel = o.sel == null ? (S().termSel || 0) : o.sel;
    const turn = [files ? row('eye', '审查本轮改动', files + ' 个文件待审查', go, 'data-act="we-open" data-tab="review"') : '',
      plan ? row('list', '计划', '最新一份计划', go, 'data-act="we-open" data-tab="plan"') : '',
      tasks ? row('activity', '子 Agent 与任务', tasks + ' 个运行中', go, 'data-act="we-open" data-tab="task:watch"') : ''].join('');
    const terms = n ? Array.from({ length: n }, (_, i) => row('terminal', name(i), i === sel ? '当前 · 项目根目录' : '后台运行', go, `data-act="we-term" data-i="${i}"`)).join('')
        + row('plus', '新建终端', '再开一个 shell', kbd('⌃', '⇧', '`'), 'data-act="we-term-new"')
      : row('terminal', '打开终端', '在项目目录里打开 shell', kbd('⌘', 'J'), 'data-act="we-term-new"');
    return `<div class="ws-empty2"><div class="we-col">
      <div><div class="fs-title m">工作区</div><div class="t3 fs-cap" style="margin-top:3px">改动、计划、任务、文件和终端都从这里进入;打开的内容会成为标签。</div></div>
      <div><div class="we-cap">本轮</div><div class="we-card">${turn || '<div class="we-none">本轮还没有改动、计划或任务。</div>'}</div></div>
      <div><div class="we-cap">终端${n > 1 ? ` · ${n} 个` : ''}</div><div class="we-card">${terms}</div></div>
      <div><div class="we-cap">打开</div><div class="we-card">${row('folder', '文件', '在导航栏浏览项目文件', go, 'data-act="nav" data-nav="files"')}${row('branch', '源代码管理', '暂存、提交、图谱', kbd('⌃', '⇧', 'G'), 'data-act="nav" data-nav="scm"')}${row('search', '在项目中搜索', '', kbd('⇧', '⌘', 'F'), 'data-act="nav" data-nav="search"')}${row('cmd', '命令面板', '快速打开文件与命令', kbd('⌘', 'K'), 'data-act="palette"')}</div></div>
    </div></div>`; };

  /** 底部面板「终端」分段里的终端标签:每个终端一个胶囊,选中的带 ×(只剩一个时不给关),末尾是只读的会话命令与「+」 */
  IDE.termChips = (o = {}) => {
    const n = o.terms == null ? (S().terms || 1) : o.terms, sel = o.sel == null ? (S().termSel || 0) : o.sel;
    return Array.from({ length: n }, (_, i) => U.chip(name(i), 'prompt', i === sel ? 'on' : '', `data-act="term-pick" data-i="${i}"`) + (n > 1 && i === sel ? U.ib('x', 'sm', '关闭这个终端', `data-act="term-close" data-i="${i}"`) : '')).join('')
      + U.chip('cargo test router', 'lock', 'ro', 'data-tip="来自会话的命令(只读)"') + U.ib('plus', 'sm', '新建终端', 'data-act="term-new"'); };
})();
