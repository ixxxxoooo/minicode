/* 设计稿 v3 · 整页功能:自动化、画布、浏览器(自定义见 customize.js,用量见 usage.js)。它们替换工作台的「编辑区 + 对话栏 + 检查器」,
   导航栏与标题栏不变,所以在三套方案下都自然成立。交互由 behave.js 驱动:卡片选中、立即运行、新建、大纲定位、版本回退、摘要 / 大纲 / 链接切换。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog, chk, inp, field, area, esc } = U;
  const pages = {};

  /** 页头:标题 + 说明 + 右侧操作 */
  const pgh = (t, d, acts = '') => `<div class="pg-h"><div class="grow"><div class="fs-head b">${t}</div><div class="t3">${d}</div></div><div class="f g3 ai">${acts}</div></div>`;
  const miniCmp = (ph, extra = '', id = '') => `<div class="cmp"><div class="cmp-in" style="min-height:30px"><input class="cmp-input" data-in="${id}" placeholder="${ph}" spellcheck="false"></div><div class="cmp-bar">${ib('plus', 'tip-u', '添加附件')}${extra}<span class="grow"></span>${ib('send', 'send', '发送', `data-act="${id === 'br-q' ? 'br-send' : 'cv-send'}"`)}</div></div>`;

  /* ───────── 自动化:本周周历 + 卡片列表 + 编辑抽屉 ───────── */
  const DAYS = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
  const AUS = [
    { n: '日报摘要', sch: '工作日 · 09:00', proj: 'router-demo', last: '今天 09:00 · 成功', st: 'ok', on: 1, prompt: '汇总昨天的提交与未关闭的 PR,生成一份简短的日报,按「已完成 / 进行中 / 阻塞」分组。', kind: 1, day: '工作日', time: '09:00', perm: '只读 · 计划模式', runs: [['今天 09:00', '成功', 'ok'], ['昨天 09:00', '成功', 'ok'], ['周五 09:00', '成功', 'ok']] },
    { n: '依赖升级检查', sch: '每周二 · 14:00', proj: 'web-dashboard', last: '周二 14:00 · 失败:网络超时', st: 'err', on: 1, prompt: '检查 Cargo.toml 里可升级的依赖,列出破坏性变更,并给出升级顺序建议。不要修改任何文件。', kind: 1, day: '周二', time: '14:00', perm: '只读 · 计划模式', runs: [['周二 14:00', '失败', 'err'], ['上周二 14:00', '成功', 'ok'], ['11 月 2 日', '成功', 'ok']] },
    { n: '周报生成', sch: '每周五 · 17:30', proj: 'router-demo', last: '上周五 · 成功', st: 'ok', on: 1, prompt: '根据本周所有已合并的提交生成周报,突出对用户可见的变化。', kind: 1, day: '周五', time: '17:30', perm: '只读 · 计划模式', runs: [['上周五 17:30', '成功', 'ok'], ['11 月 1 日', '成功', 'ok']] },
    { n: '备份检查', sch: '每天 · 20:00', proj: 'docs-site', last: '已暂停', st: '', on: 0, prompt: '确认 docs-site 的备份任务今天已完成,没有则提醒。', kind: 0, day: '每天', time: '20:00', perm: '只读 · 计划模式', runs: [['11 月 8 日', '成功', 'ok']] },
  ];
  const SLOTS = { 0: [['09:00', 0]], 1: [['09:00', 0], ['14:00', 1, 'err']], 2: [['09:00', 0]], 3: [['09:00', 0]], 4: [['09:00', 0], ['17:30', 2]], 5: [], 6: [['20:00', 3, 'off']] };
  /** 某一周的 7 个日期(off:相对本周的周数偏移) */
  const weekDates = (off) => { const base = new Date(2026, 10, 9 + off * 7); return DAYS.map((_, k) => new Date(base.getFullYear(), base.getMonth(), base.getDate() + k)); };
  const wk = (off) => { const ds = weekDates(off); return `<div class="wk">${DAYS.map((d, k) => `<div class="wk-d ${off === 0 && k === 6 ? 'today' : ''}"><div class="wk-h"><span class="t3">${d}</span><b class="num">${ds[k].getDate()}</b></div>
    ${(SLOTS[k] || []).map(([t, i, s]) => `<div class="wk-s ${s || ''}" data-act="au-sel" data-i="${i}" tabindex="0"><span class="num">${t}</span><span class="ell">${AUS[i].n}</span></div>`).join('') || '<div class="t4 fs-cap" style="padding:2px 4px">没有计划</div>'}</div>`).join('')}</div>`; };
  const auCard = (a, k, sel) => `<div class="au ${sel ? 'on' : ''}" data-act="au-sel" data-i="${k}" tabindex="0"><div class="f ai g3"><span class="si-box">${ic('timer')}</span><div class="grow"><div class="m">${esc(a.n)}</div><div class="t3 fs-cap">${a.sch}</div></div>${tog(a.on, `data-au="${k}"`)}</div>
    <div class="f ai g3 fs-cap t3">${chip(a.proj, 'folder', 'line')}<span class="grow"></span>${a.st === 'ok' ? `${ic('okc', 's12 c-ok')}<span>${a.last}</span>` : a.st === 'err' ? `${ic('errc', 's12 c-err')}<span class="c-err">${a.last}</span>` : `<span>${a.last}</span>`}</div></div>`;
  const auFilter = (st) => AUS.map((a, k) => [a, k]).filter(([a]) => ({ 0: 1, 1: a.on, 2: !a.on }[st.auF || 0]));
  const auDrawer = (st) => { const a = AUS[st.auSel] || AUS[0], k = AUS.indexOf(a);
    return `<div class="f ai g3"><b class="grow">编辑自动化</b>${ib('x', 'sm', '关闭')}</div>
      <div class="fc g2"><span class="t3 fs-cap m">名称</span>${field(a.n, 0, { id: 'au-n', attr: 'style="width:100%"' })}</div>
      <div class="fc g2"><span class="t3 fs-cap m">提示词</span>${area(a.prompt, { id: 'au-p', rows: 4 })}</div>
      <div class="fc g2"><span class="t3 fs-cap m">计划</span>${seg(['每天', '每周', '每月', 'Cron'], a.kind)}<div class="f g2 ai">${chip(a.day, '', 'line')}${field(a.time, 70)}<span class="t3 fs-cap">时区 Asia/Shanghai</span></div></div>
      <div class="fc g2"><span class="t3 fs-cap m">项目与权限</span><div class="f g2 ai">${chip(a.proj, 'folder', 'line')}${chip(a.perm, 'shield', 'line')}</div></div>
      <div class="fc g2"><span class="t3 fs-cap m">最近运行</span><div class="card" id="au-runs" style="padding:2px 4px">${a.runs.map(([t, s, c]) => `<div class="row">${ic(c === 'ok' ? 'okc' : 'errc', 's12 ' + (c === 'ok' ? 'c-ok' : 'c-err'))}<span class="grow">${t}</span><span class="t3">${s}</span>${ic('chevR', 's12 t4')}</div>`).join('')}</div></div>
      <span class="grow"></span><div class="f g2">${btn('立即运行', 'soft', 'play', `data-act="au-run" data-i="${k}"`)}${btn('删除', 'danger ghost', 'trash', `data-act="au-del" data-i="${k}"`)}<span class="grow"></span>${btn('保存', 'pri', '', 'data-act="cu-save"')}</div>`; };
  const auBody = (st) => { const rows = auFilter(st);
    return rows.length ? `<div class="au-grid stagger">${rows.map(([a, k]) => auCard(a, k, k === (st.auSel ?? 1))).join('')}</div>` : `<div class="card"><div class="empty" style="padding:30px 12px">${ic('timer', 's20 t4')}<b>${st.auF === 1 ? '没有运行中的自动化' : '没有已暂停的自动化'}</b><div class="t3 fs-cap">切到「全部」看看。</div></div></div>`; };
  /** 周历标题行:第几周、日期范围、翻周按钮 */
  const wkHead = (off) => { const ds = weekDates(off), a0 = ds[0], a6 = ds[6];
    return `<b>${off === 0 ? '本周' : off < 0 ? `${-off} 周前` : `${off} 周后`}</b><span class="t3">${a0.getMonth() + 1} 月 ${a0.getDate()} 日 – ${a6.getMonth() !== a0.getMonth() ? (a6.getMonth() + 1) + ' 月 ' : ''}${a6.getDate()} 日</span><span class="grow"></span>${ib('chevL', 'sm', '上一周', 'data-act="wk-nav" data-d="-1"')}${btn('今天', 'sm', '', 'data-act="wk-nav" data-d="0"')}${ib('chevR', 'sm', '下一周', 'data-act="wk-nav" data-d="1"')}`; };
  pages.automation = (st) => { st.auSel = st.auSel ?? 1; const off = st.wkOff || 0;
    return `${pgh('自动化', '让 Claude 按计划自动做事:定时检查、生成报告、巡检依赖。每次运行都会留下一个可回看的会话。', btn('新建自动化', 'pri', 'plus', 'data-act="au-new"'))}
    <div class="pg-split"><div class="pg-main scroll">
      <div class="f ai g3" style="margin-bottom:8px" id="au-wkh">${wkHead(off)}</div>
      <div id="au-wk">${wk(off)}</div>
      <div class="f ai g3" style="margin:18px 0 8px"><b>全部自动化</b>${cnt(AUS.length)}<span class="grow"></span>${seg(['全部', '运行中', '已暂停'], st.auF || 0, '', 'au-filter')}</div>
      <div id="au-list">${auBody(st)}</div>
    </div>
    <aside class="drawer" id="au-drawer">${auDrawer(st)}</aside></div>`; };

  /* ───────── 画布 ───────── */
  const dgn = (x, y, w, t, hot) => `<g class="dgn ${hot ? 'hot' : ''}"><rect x="${x}" y="${y}" width="${w}" height="30" rx="7"/><text x="${x + w / 2}" y="${y + 19}" text-anchor="middle">${t}</text></g>`;
  const arrow = (d) => `<path d="${d}" class="dga" marker-end="url(#ah)"/>`;
  const diagram = () => `<svg class="dg" viewBox="0 0 520 150"><defs><marker id="ah" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="7" markerHeight="7" orient="auto"><path d="M0 0 8 4 0 8z" class="dgh"/></marker></defs>
    ${arrow('M110 45H160')}${arrow('M250 45H300')}${arrow('M390 45H430')}${arrow('M205 60V100H300')}
    ${dgn(20, 30, 90, '请求')}${dgn(160, 30, 90, '路由表', 1)}${dgn(300, 30, 90, '处理器')}${dgn(430, 30, 70, '响应')}${dgn(300, 85, 90, 'normalize')}</svg>`;
  const OUTLINE = [['Q3 路由层重构', 0, 'h-top'], ['背景', 1, 'h-bg'], ['架构', 1, 'h-arch'], ['里程碑', 1, 'h-ms'], ['待办', 1, 'h-todo']];
  const outline = () => OUTLINE.map(([t, l, id], k) => `<div class="row ${k === 0 ? 'on' : ''}" data-act="cv-outline" data-id="${id}" tabindex="0" style="padding-left:${8 + l * 14}px">${l ? '' : ic('fileText', 's12')}<span class="ell">${t}</span></div>`).join('');
  const VERS = [['v3', '刚刚', 'Claude · 补充里程碑表'], ['v2', '2 小时前', '你 · 调整架构图'], ['v1', '昨天', 'Claude · 初稿']];
  const versRows = () => VERS.map(([v, t, d], k) => `<div class="row" style="height:auto;padding:4px 8px;align-items:flex-start">${tag(v, k === 0 ? 'ac' : '')}<div class="grow"><div class="ell">${d}</div><div class="t4 fs-cap">${t}</div></div>${k === 0 ? '' : ib('rewind', 'sm', '回到此版本', `data-act="cv-rewind" data-v="${v}"`)}</div>`).join('');
  pages.canvas = () => `<div class="cv-top"><div class="grow f ai g3">${ic('canvas', 'c-ac')}<b>Q3 路由层重构</b>${tag('草稿')}<span class="t3 fs-cap" id="cv-ver">v3 · 刚刚由 Claude 修改</span></div>${seg(['编辑', '预览'], 0, '', 'cv-mode')}${btn('导出', 'sm', 'download', 'data-act="toast" data-msg="已导出为 Markdown"')}${btn('让 Claude 修改', 'sm soft', 'spark', 'data-act="cv-focus"')}</div>
    <div class="pg-split"><div class="pg-main scroll" id="cv-scroll"><div class="cv-doc" id="cv-doc" contenteditable="true" spellcheck="false">
      <h1 id="h-top">Q3 路由层重构</h1><p class="lead" id="h-bg">把路径匹配收敛到单一的归一化入口,消除大小写与末尾斜杠带来的重复路由。</p>
      <h2 id="h-arch">架构</h2><div class="cv-card" contenteditable="false"><div class="cv-ch">${ic('layers', 's12')}<span class="grow">请求经过路由表,先归一化再匹配</span>${tag('Mermaid')}${ib('max', 'sm', '全屏查看', 'data-act="toast" data-msg="图表全屏查看:滚轮缩放、拖动平移、0 适应窗口"')}${ib('code', 'sm', '查看源码')}</div>${diagram()}</div>
      <h2 id="h-ms">里程碑</h2><table class="cv-tb"><thead><tr><th>阶段</th><th>负责人</th><th>时间</th><th>状态</th></tr></thead><tbody>
        <tr><td>归一化抽取</td><td>你</td><td>10 月 18 日</td><td>${tag('已完成', 'ok')}</td></tr><tr><td>匹配层替换</td><td>Lin</td><td>10 月 25 日</td><td>${tag('进行中', 'ac')}</td></tr><tr><td>旧路由清理</td><td>Wen</td><td>11 月 3 日</td><td>${tag('未开始')}</td></tr></tbody></table>
      <h2 id="h-todo">待办</h2><div class="todo" style="gap:7px" contenteditable="false">${[['抽出 normalize', 'done'], ['补充大小写用例', 'now'], ['清理旧的 fallback 逻辑', '']].map(([t, s]) => `<div class="${s === 'done' ? 'is-done' : s === 'now' ? 'is-now' : 't2'}">${chk(s)}<span>${t}</span></div>`).join('')}</div></div></div>
    <aside class="drawer cv-side"><div class="grp-h">大纲</div><div style="margin:0 -4px">${outline()}</div><div class="sep"></div>
      <div class="grp-h">版本<span id="cv-vn">${cnt(VERS.length)}</span></div><div id="cv-vers">${versRows()}</div>
      <span class="grow"></span><div class="grp-h">让 Claude 修改</div>${miniCmp('描述要怎么改,例如「把里程碑改成表格」', chip('选中:架构图', 'at', 'line'), 'cv-q')}</aside></div>`;

  /* ───────── 浏览器(阅读模式) ───────── */
  const BRP = {
    sum: () => `<div class="fc g3 stagger">${['Router 把路径模式映射到处理器', '路径区分大小写,末尾斜杠需显式处理', '支持捕获参数、通配与嵌套路由'].map(t => `<div class="f g3"><span class="dot ac" style="margin-top:7px"></span><span style="line-height:1.55">${t}</span></div>`).join('')}</div>`,
    out: () => `<div class="fc stagger">${[['Routing', 0], ['基本用法', 1], ['路径匹配', 1], ['捕获参数', 2], ['通配与嵌套', 2]].map(([t, l]) => `<div class="row" data-act="toast" data-msg="已定位到「${t}」" style="padding-left:${8 + l * 14}px"><span class="ell">${t}</span></div>`).join('')}</div>`,
    lnk: () => `<div class="fc stagger">${[['axum::Router', 'docs.rs'], ['tower::Service', 'docs.rs'], ['路径匹配的 RFC', 'rfc-editor.org'], ['示例仓库', 'github.com']].map(([t, h]) => `<div class="row" style="height:auto;padding:5px 8px" data-act="toast" data-msg="已在阅读模式打开 ${h}"><div class="grow"><div class="ell">${t}</div><div class="t4 fs-micro">${h}</div></div>${ic('ext', 's12 t4')}</div>`).join('')}</div>`,
  };
  pages.browser = () => `<div class="br-bar">${ib('chevL', '', '后退')}${ib('chevR', '', '前进')}${ib('refresh', '', '重新载入')}
      <div class="inp grow">${ic('lock', 's12 c-ok')}<span class="v grow ell">docs.rs/axum/latest/axum/routing</span>${ib('bookmark', 'sm', '收藏')}</div>${chip('阅读模式', 'book', 'on')}${chip('与此页面对话', 'chat', 'line')}${ib('gear', '', '浏览器设置', 'data-act="settab" data-tab="browser" data-goto="settings"')}</div>
    <div class="pg-split"><div class="pg-main scroll"><article class="reader">
      <div class="t3 fs-cap">docs.rs · 阅读约 6 分钟</div><h1>Routing</h1>
      <p>Routing 决定请求交给哪个处理器。<code>Router</code> 用路径模式匹配请求,并把它转发给对应的 <code>Handler</code>。</p>
      <h2>基本用法</h2><div class="tcard"><div class="tcard-h">${ic('code', 's12')}<span class="grow">rust</span>${ib('copy', 'sm', '复制', 'data-act="toast" data-msg="已复制代码"')}</div><div class="term"><span class="k">let</span> app = Router::new()\n    .route(<span class="s">"/"</span>, get(root))\n    .route(<span class="s">"/users"</span>, post(create_user));</div></div>
      <h2>路径匹配</h2><p>路径区分大小写;末尾斜杠需要显式处理。常见做法是在中间件里统一归一化。</p><ul><li>捕获参数:<code>/users/:id</code></li><li>通配:<code>/files/*path</code></li><li>嵌套:<code>nest("/api", api)</code></li></ul></article></div>
    <aside class="drawer"><div class="f ai g3"><b class="grow">与此页面对话</b>${ib('x', 'sm', '关闭')}</div><div class="tabs" style="padding:0;margin:0 -4px">${[['sum', '摘要'], ['out', '大纲'], ['lnk', '链接']].map(([id, t], k) => `<span class="tab ${k ? '' : 'on'}" data-act="br-tab" data-tab="${id}" tabindex="0">${t}</span>`).join('')}</div>
      <div id="br-pane">${BRP.sum()}</div>
      <div class="fc g2"><span class="t3 fs-cap m">快捷提示</span><div class="f g2 wrap">${['总结要点', '提取代码示例', '翻译成中文', '对比 actix'].map(t => chip(t, '', 'line', `data-act="br-ask" data-q="${t}" tabindex="0"`)).join('')}</div></div>
      <span class="grow"></span>${miniCmp('就这个页面提问…', '', 'br-q')}</aside></div>`;

  W.pages = pages;
  W.diagram = diagram;
  W.au = { AUS, SLOTS, auDrawer, auBody, auCard, wk, wkHead, weekDates, BRP, VERS, versRows };
})();
