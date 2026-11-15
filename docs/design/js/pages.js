/* 设计稿 v3 · 整页功能:自动化、画布、浏览器、自定义。它们替换工作台的「编辑区 + 对话栏 + 检查器」,
   导航栏与标题栏不变,所以在三套方案下都自然成立。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog, chk } = U;
  const pages = {};

  /** 页头:标题 + 说明 + 右侧操作 */
  const pgh = (t, d, acts = '') => `<div class="pg-h"><div class="grow"><div class="fs-head b">${t}</div><div class="t3">${d}</div></div><div class="f g3 ai">${acts}</div></div>`;
  const fld = (v, w = 0, c = '') => `<div class="fld ${c}" ${w ? `style="width:${w}px"` : 'style="width:100%"'}>${v}</div>`;
  const miniCmp = (ph, extra = '') => `<div class="cmp"><div class="cmp-in" style="min-height:30px">${ph}</div><div class="cmp-bar">${ib('plus', 'tip-u', '添加附件')}${extra}<span class="grow"></span>${ib('send', 'send')}</div></div>`;

  /* ───────── 自动化:本周周历 + 卡片列表 + 编辑抽屉 ───────── */
  const DAYS = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
  const SLOTS = { 0: [['09:00', '日报摘要']], 1: [['09:00', '日报摘要'], ['14:00', '依赖升级检查', 'err']], 2: [['09:00', '日报摘要']], 3: [['09:00', '日报摘要']], 4: [['09:00', '日报摘要'], ['17:30', '周报生成']], 5: [], 6: [['20:00', '备份检查', 'off']] };
  const wk = () => `<div class="wk">${DAYS.map((d, k) => `<div class="wk-d ${k === 6 ? 'today' : ''}"><div class="wk-h"><span class="t3">${d}</span><b class="num">${9 + k}</b></div>
    ${(SLOTS[k] || []).map(([t, n, s]) => `<div class="wk-s ${s || ''}"><span class="num">${t}</span><span class="ell">${n}</span></div>`).join('') || '<div class="t4 fs-cap" style="padding:2px 4px">没有计划</div>'}</div>`).join('')}</div>`;
  const auto = (n, sch, proj, last, st, on, sel) => `<div class="au ${sel ? 'on' : ''}"><div class="f ai g3"><span class="si-box">${ic('timer')}</span><div class="grow"><div class="m">${n}</div><div class="t3 fs-cap">${sch}</div></div>${tog(on)}</div>
    <div class="f ai g3 fs-cap t3">${chip(proj, 'folder', 'line')}<span class="grow"></span>${st === 'ok' ? `${ic('okc', 's12 c-ok')}<span>${last}</span>` : st === 'err' ? `${ic('errc', 's12 c-err')}<span class="c-err">${last}</span>` : `<span>${last}</span>`}</div></div>`;
  pages.automation = () => `${pgh('自动化', '让 Claude 按计划自动做事:定时检查、生成报告、巡检依赖。每次运行都会留下一个可回看的会话。', btn('新建自动化', 'pri', 'plus'))}
    <div class="pg-split"><div class="pg-main scroll">
      <div class="f ai g3" style="margin-bottom:8px"><b>本周</b><span class="t3">11 月 9 日 – 15 日</span><span class="grow"></span>${ib('chevL', 'sm')}${btn('今天', 'sm')}${ib('chevR', 'sm')}</div>
      ${wk()}
      <div class="f ai g3" style="margin:18px 0 8px"><b>全部自动化</b>${cnt(4)}<span class="grow"></span>${seg(['全部', '运行中', '已暂停'], 0)}</div>
      <div class="au-grid stagger">${auto('日报摘要', '工作日 · 09:00', 'router-demo', '今天 09:00 · 成功', 'ok', 1)}${auto('依赖升级检查', '每周二 · 14:00', 'web-dashboard', '周二 14:00 · 失败:网络超时', 'err', 1, 1)}${auto('周报生成', '每周五 · 17:30', 'router-demo', '上周五 · 成功', 'ok', 1)}${auto('备份检查', '每天 · 20:00', 'docs-site', '已暂停', '', 0)}</div>
    </div>
    <aside class="drawer a-rise"><div class="f ai g3"><b class="grow">编辑自动化</b>${ib('x', 'sm')}</div>
      <div class="fc g2"><span class="t3 fs-cap m">名称</span>${fld('依赖升级检查')}</div>
      <div class="fc g2"><span class="t3 fs-cap m">提示词</span><div class="ta">检查 Cargo.toml 里可升级的依赖,列出破坏性变更,并给出升级顺序建议。不要修改任何文件。</div></div>
      <div class="fc g2"><span class="t3 fs-cap m">计划</span>${seg(['每天', '每周', '每月', 'Cron'], 1)}<div class="f g2 ai">${chip('周二', '', 'line')}${fld('14:00', 70)}<span class="t3 fs-cap">时区 Asia/Shanghai</span></div></div>
      <div class="fc g2"><span class="t3 fs-cap m">项目与权限</span><div class="f g2 ai">${chip('web-dashboard', 'folder', 'line')}${chip('只读 · 计划模式', 'shield', 'line')}</div></div>
      <div class="fc g2"><span class="t3 fs-cap m">最近运行</span><div class="card" style="padding:2px 4px">${[['周二 14:00', '失败', 'err'], ['上周二 14:00', '成功', 'ok'], ['11 月 2 日', '成功', 'ok']].map(([t, s, c]) => `<div class="row">${ic(c === 'ok' ? 'okc' : 'errc', 's12 ' + (c === 'ok' ? 'c-ok' : 'c-err'))}<span class="grow">${t}</span><span class="t3">${s}</span>${ic('chevR', 's12 t4')}</div>`).join('')}</div></div>
      <span class="grow"></span><div class="f g2">${btn('立即运行', 'soft', 'play')}${btn('删除', 'danger ghost', 'trash')}<span class="grow"></span>${btn('保存', 'pri')}</div></aside></div>`;

  /* ───────── 画布 ───────── */
  const dgn = (x, y, w, t, hot) => `<g class="dgn ${hot ? 'hot' : ''}"><rect x="${x}" y="${y}" width="${w}" height="30" rx="7"/><text x="${x + w / 2}" y="${y + 19}" text-anchor="middle">${t}</text></g>`;
  const arrow = (d) => `<path d="${d}" class="dga" marker-end="url(#ah)"/>`;
  const diagram = () => `<svg class="dg" viewBox="0 0 520 150"><defs><marker id="ah" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="7" markerHeight="7" orient="auto"><path d="M0 0 8 4 0 8z" class="dgh"/></marker></defs>
    ${arrow('M110 45H160')}${arrow('M250 45H300')}${arrow('M390 45H430')}${arrow('M205 60V100H300')}
    ${dgn(20, 30, 90, '请求')}${dgn(160, 30, 90, '路由表', 1)}${dgn(300, 30, 90, '处理器')}${dgn(430, 30, 70, '响应')}${dgn(300, 85, 90, 'normalize')}</svg>`;
  const outline = () => [['Q3 路由层重构', 0, 1], ['背景', 1], ['架构', 1], ['里程碑', 1], ['待办', 1]].map(([t, l, on]) => `<div class="row ${on ? 'on' : ''}" style="padding-left:${8 + l * 14}px">${l ? '' : ic('fileText', 's12')}<span class="ell">${t}</span></div>`).join('');
  pages.canvas = () => `<div class="cv-top"><div class="grow f ai g3">${ic('canvas', 'c-ac')}<b>Q3 路由层重构</b>${tag('草稿')}<span class="t3 fs-cap">v3 · 刚刚由 Claude 修改</span></div>${seg(['编辑', '预览'], 0)}${btn('导出', 'sm', 'download')}${btn('让 Claude 修改', 'sm soft', 'spark')}</div>
    <div class="pg-split"><div class="pg-main scroll"><div class="cv-doc">
      <h1>Q3 路由层重构</h1><p class="lead">把路径匹配收敛到单一的归一化入口,消除大小写与末尾斜杠带来的重复路由。</p>
      <h2>架构</h2><div class="cv-card"><div class="cv-ch">${ic('layers', 's12')}<span class="grow">请求经过路由表,先归一化再匹配</span>${tag('Mermaid')}${ib('max', 'sm', '全屏查看')}${ib('code', 'sm', '查看源码')}</div>${diagram()}</div>
      <h2>里程碑</h2><table class="cv-tb"><thead><tr><th>阶段</th><th>负责人</th><th>时间</th><th>状态</th></tr></thead><tbody>
        <tr><td>归一化抽取</td><td>你</td><td>10 月 18 日</td><td>${tag('已完成', 'ok')}</td></tr><tr><td>匹配层替换</td><td>Lin</td><td>10 月 25 日</td><td>${tag('进行中', 'ac')}</td></tr><tr><td>旧路由清理</td><td>Wen</td><td>11 月 3 日</td><td>${tag('未开始')}</td></tr></tbody></table>
      <h2>待办</h2><div class="todo" style="gap:7px">${[['抽出 normalize', 'done'], ['补充大小写用例', 'now'], ['清理旧的 fallback 逻辑', '']].map(([t, s]) => `<div class="${s === 'done' ? 'is-done' : s === 'now' ? 'is-now' : 't2'}">${chk(s)}<span>${t}</span></div>`).join('')}</div></div></div>
    <aside class="drawer cv-side"><div class="grp-h">大纲</div><div style="margin:0 -4px">${outline()}</div><div class="sep"></div>
      <div class="grp-h">版本${cnt(3)}</div>${[['v3', '刚刚', 'Claude · 补充里程碑表', 1], ['v2', '2 小时前', '你 · 调整架构图'], ['v1', '昨天', 'Claude · 初稿']].map(([v, t, d, on]) => `<div class="row" style="height:auto;padding:4px 8px;align-items:flex-start">${tag(v, on ? 'ac' : '')}<div class="grow"><div class="ell">${d}</div><div class="t4 fs-cap">${t}</div></div>${on ? '' : ib('rewind', 'sm', '回到此版本')}</div>`).join('')}
      <span class="grow"></span><div class="grp-h">让 Claude 修改</div>${miniCmp('描述要怎么改,例如「把里程碑改成表格」', chip('选中:架构图', 'at', 'line'))}</aside></div>`;

  /* ───────── 浏览器(阅读模式) ───────── */
  pages.browser = () => `<div class="br-bar">${ib('chevL', '', '后退')}${ib('chevR', '', '前进')}${ib('refresh', '', '重新载入')}
      <div class="inp grow">${ic('lock', 's12 c-ok')}<span class="v grow ell">docs.rs/axum/latest/axum/routing</span>${ib('bookmark', 'sm', '收藏')}</div>${chip('阅读模式', 'book', 'on')}${chip('与此页面对话', 'chat', 'line')}</div>
    <div class="pg-split"><div class="pg-main scroll"><article class="reader">
      <div class="t3 fs-cap">docs.rs · 阅读约 6 分钟</div><h1>Routing</h1>
      <p>Routing 决定请求交给哪个处理器。<code>Router</code> 用路径模式匹配请求,并把它转发给对应的 <code>Handler</code>。</p>
      <h2>基本用法</h2><div class="tcard"><div class="tcard-h">${ic('code', 's12')}<span class="grow">rust</span>${ib('copy', 'sm')}</div><div class="term"><span class="k">let</span> app = Router::new()\n    .route(<span class="s">"/"</span>, get(root))\n    .route(<span class="s">"/users"</span>, post(create_user));</div></div>
      <h2>路径匹配</h2><p>路径区分大小写;末尾斜杠需要显式处理。常见做法是在中间件里统一归一化。</p><ul><li>捕获参数:<code>/users/:id</code></li><li>通配:<code>/files/*path</code></li><li>嵌套:<code>nest("/api", api)</code></li></ul></article></div>
    <aside class="drawer"><div class="f ai g3"><b class="grow">与此页面对话</b>${ib('x', 'sm')}</div><div class="tabs" style="padding:0;margin:0 -4px">${['摘要', '大纲', '链接'].map((t, k) => `<span class="tab ${k ? '' : 'on'}" data-act="tab">${t}</span>`).join('')}</div>
      <div class="fc g3 stagger">${['Router 把路径模式映射到处理器', '路径区分大小写,末尾斜杠需显式处理', '支持捕获参数、通配与嵌套路由'].map(t => `<div class="f g3"><span class="dot ac" style="margin-top:7px"></span><span style="line-height:1.55">${t}</span></div>`).join('')}</div>
      <div class="fc g2"><span class="t3 fs-cap m">快捷提示</span><div class="f g2 wrap">${['总结要点', '提取代码示例', '翻译成中文', '对比 actix'].map(t => chip(t, '', 'line')).join('')}</div></div>
      <span class="grow"></span>${miniCmp('就这个页面提问…')}</aside></div>`;

  /* ───────── 自定义:左分类栏 + 列表 + 详情 ───────── */
  const CU = {
    plugins: { i: 'plug', t: '插件', d: '插件把 Skills、命令、子 Agent 与钩子打包在一起,可在市场里浏览与启停。', add: '浏览市场',
      rows: [['plug', 'superpowers', '官方市场 · v4.2.0 · 12 个 Skills', 1], ['plug', 'code-review', '官方市场 · v1.3.0 · 2 个命令', 1], ['plug', 'commit-helper', '本地 · v0.2.0', 1], ['plug', 'design-kit', '官方市场 · v0.9.1', 0]],
      det: { name: 'superpowers', kv: [['来源', 'claude-plugins-official'], ['版本', '4.2.0'], ['安装位置', '用户']], lt: '包含内容', list: [['12 个 Skills', 1], ['3 个命令', 1], ['1 个钩子', 1]] } },
    mcp: { i: 'server', t: 'MCP 服务器', d: '给 Claude 接入外部工具与数据。状态来自 claude 的 mcp_status;开关只对当前项目生效。', add: '添加服务器',
      rows: [['server', 'github', 'stdio · 14 个工具 · 已连接', 1, 'ok'], ['server', 'filesystem', 'stdio · 8 个工具 · 已连接', 1, 'ok'], ['server', 'figma', 'http · 需要授权', 1, 'warn'], ['server', 'postgres', 'stdio · 连接失败:ECONNREFUSED', 0, 'err']],
      det: { name: 'github', kv: [['传输', 'stdio'], ['命令', 'npx -y @modelcontextprotocol/server-github'], ['配置层', '用户'], ['工具定义占用', '3.2k token · 1.6%']], lt: '工具(14)', list: [['create_issue', 1], ['list_pull_requests', 1], ['get_file_contents', 1], ['search_code', 1], ['create_branch', 0]] } },
    skills: { i: 'spark', t: 'Skills', d: 'Claude 按场景自动调用的能力包,一个目录 + SKILL.md。', add: '新建 Skill',
      rows: [['spark', 'brainstorming', '插件 · 动手前先梳理需求', 1], ['spark', 'test-driven-development', '插件 · 先写失败的测试', 1], ['spark', 'release-notes', '项目 · 生成发布说明', 1], ['spark', 'sql-review', '用户 · 审查 SQL', 0]],
      det: { name: 'release-notes', kv: [['范围', '项目'], ['路径', '.claude/skills/release-notes/SKILL.md'], ['触发', '提到「发布说明」「changelog」时']], lt: '文件', list: [['SKILL.md', 1], ['template.md', 1]] } },
    commands: { i: 'slash', t: '斜杠命令', d: '在输入框里用 / 调用的提示词模板。', add: '新建命令',
      rows: [['slash', '/review', '项目 · 审查当前改动', 1], ['slash', '/test', '项目 · 跑测试并总结失败', 1], ['slash', '/release', '用户 · 生成并发布版本', 1], ['slash', '/clean', '用户 · 清理构建产物', 1]],
      det: { name: '/review', kv: [['范围', '项目'], ['路径', '.claude/commands/review.md'], ['参数', '$ARGUMENTS']], lt: '提示词', list: [['审查当前 git diff,按严重程度列出问题', 1]] } },
    agents: { i: 'bot', t: '子 Agent', d: '带独立上下文与工具权限的专用助手,由主 Agent 委派任务。', add: '新建子 Agent',
      rows: [['bot', 'code-reviewer', '项目 · Sonnet · 只读', 1], ['bot', 'researcher', '用户 · Haiku · 联网', 1], ['bot', 'test-writer', '项目 · Sonnet · 可编辑', 1]],
      det: { name: 'code-reviewer', kv: [['模型', 'Sonnet 5.5'], ['范围', '项目'], ['说明', '审查代码的正确性与风格,不修改文件']], lt: '工具', list: [['Read', 1], ['Grep', 1], ['Glob', 1], ['Edit', 0], ['Bash', 0]] } },
    rules: { i: 'book', t: '规则与记忆', d: 'CLAUDE.md 与 .claude/rules 里的长期约定,新会话自动加载。', add: '新建规则',
      rows: [['fileText', '~/.claude/CLAUDE.md', '用户 · 1.2 KB', 1], ['fileText', 'router-demo/CLAUDE.md', '项目 · 3.4 KB', 1]],
      det: { name: 'router-demo/CLAUDE.md', kv: [['范围', '项目'], ['大小', '3.4 KB'], ['修改', '昨天']], lt: '摘要', list: [['回答、注释一律中文', 1], ['提交信息用 Conventional Commits', 1], ['不提交构建产物', 1]] } },
    hooks: { i: 'bolt', t: '钩子', d: '在工具调用前后自动运行的命令。陌生项目的钩子需要先在「项目信任」里允许。', add: '添加钩子',
      rows: [['bolt', 'PreToolUse · Bash', '拦截含 rm -rf 的命令', 1], ['bolt', 'PostToolUse · Edit', '保存后运行 cargo fmt', 1], ['bolt', 'Stop', '回合结束时发系统通知', 1]],
      det: { name: 'PostToolUse · Edit', kv: [['匹配', 'Edit|Write'], ['配置层', '项目'], ['超时', '10 秒']], lt: '命令', list: [['cargo fmt --all', 1]] } },
  };
  pages.customize = (st) => {
    const cur = CU[st.cust] ? st.cust : 'mcp', C = CU[cur], D = C.det;
    return `<div class="cu"><nav class="cu-rail"><div class="grp-h">自定义</div>${Object.entries(CU).map(([id, c]) => `<div class="row ${id === cur ? 'on' : ''}" data-act="cust" data-cust="${id}">${ic(c.i, 's12')}<span class="grow">${c.t}</span><span class="aux">${c.rows.length}</span></div>`).join('')}</nav>
      <div class="cu-main"><div class="pg-h"><div class="grow"><div class="fs-head b">${C.t}</div><div class="t3" style="max-width:560px;line-height:1.5">${C.d}</div></div>${btn(C.add, 'pri', 'plus')}</div>
        <div class="pg-split"><div class="pg-main scroll"><div class="inp fill" style="margin-bottom:8px">${ic('search', 's12')}<span class="grow">筛选${C.t}</span></div><div class="card cu-list stagger">${C.rows.map(([n, name, d, on, s], k) => `<div class="cu-row ${k === 0 ? 'on' : ''}"><span class="si-box">${ic(n)}</span><div class="grow"><div class="m f ai g3">${name}${s ? `<span class="dot ${s}"></span>` : ''}</div><div class="t3 fs-cap ell">${d}</div></div>${s === 'warn' ? btn('授权', 'sm soft') : ''}${tog(on)}</div>`).join('')}</div></div>
        <aside class="drawer"><div class="f ai g3">${ic(C.i, 'c-ac')}<b class="grow ell">${D.name}</b>${ib('more', 'sm')}</div>
          <div class="kv">${D.kv.map(([k, v]) => `<span class="t3">${k}</span><span class="${/\/|npx|--/.test(v) ? 'mono' : ''}" style="word-break:break-all">${v}</span>`).join('')}</div>
          <div class="sep" style="margin:0"></div><div class="f ai"><b class="grow">${D.lt}</b><span class="t3 fs-cap">逐项开关</span></div>
          <div class="fc">${D.list.map(([n, on]) => `<div class="row"><span class="grow ell mono">${n}</span>${tog(on)}</div>`).join('')}</div></aside></div></div></div>`;
  };

  W.pages = pages;
  W.diagram = diagram;
})();
