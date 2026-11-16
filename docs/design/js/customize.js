/* 设计稿 v3 · 自定义页:左分类栏 + 列表 + 专属详情
   插件(已安装 / 市场 / 安装进度)、MCP(状态与动作 / 工具开关 / 运行参数)、Skills(文件树 + 预览)、斜杠命令(编辑器)、
   子 Agent(工具权限表单)、规则与记忆(编辑 / 预览 + 加载顺序)、钩子(事件 / 匹配 / 命令 / 试运行)。
   数据放在本文件里并随交互真的改动(开关、新建、删除、安装),由 behave.js 驱动局部刷新。
   删除:详情标题行垃圾桶 → 居中确认框「删除 xxx? / 此操作不可撤销 / 取消·删除」→ 选中项落到相邻行,列表空时左侧空态。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, seg, tog, inp, field, area, esc } = U;
  const SC = { u: '用户', p: '项目', l: '本地' }, SCK = ['all', 'u', 'p', 'l'];

  /* ───────── 数据 ───────── */
  const DATA = {
    plugins: [
      { n: 'superpowers', i: 'plug', d: '官方市场 · v4.2.0 · 12 个 Skills', sc: 'u', on: 1, ver: '4.2.0', upd: '4.3.0', src: 'claude-plugins-official', about: '一套工程方法论 Skills:先梳理需求、先写失败的测试、系统化调试。', inc: [['Skills', 12], ['斜杠命令', 3], ['钩子', 1]] },
      { n: 'code-review', i: 'plug', d: '官方市场 · v1.3.0 · 2 个命令', sc: 'u', on: 1, ver: '1.3.0', src: 'claude-plugins-official', about: '按严重程度审查当前改动,并给出可直接应用的修改建议。', inc: [['斜杠命令', 2], ['子 Agent', 1]] },
      { n: 'commit-helper', i: 'plug', d: '本地 · v0.2.0', sc: 'p', on: 1, ver: '0.2.0', src: '本地目录 ~/plugins/commit-helper', about: '根据暂存区改动生成符合 Conventional Commits 的提交说明。', inc: [['斜杠命令', 1], ['钩子', 1]] },
      { n: 'design-kit', i: 'plug', d: '官方市场 · v0.9.1', sc: 'u', on: 0, ver: '0.9.1', src: 'claude-plugins-official', about: '设计令牌检查与组件审查。', inc: [['Skills', 4]] },
    ],
    mcp: [
      { n: 'github', i: 'server', d: 'stdio · 14 个工具 · 已连接', sc: 'u', on: 1, st: 'ok', tr: 'stdio', cmd: 'npx -y @modelcontextprotocol/server-github', env: ['GITHUB_TOKEN'], tok: '3.2k token · 1.6%', tools: [['create_issue', 1], ['list_pull_requests', 1], ['get_file_contents', 1], ['search_code', 1], ['create_branch', 0], ['merge_pull_request', 0]] },
      { n: 'filesystem', i: 'server', d: 'stdio · 8 个工具 · 已连接', sc: 'p', on: 1, st: 'ok', tr: 'stdio', cmd: 'npx -y @modelcontextprotocol/server-filesystem ~/Projects', env: [], tok: '按需加载,不占上下文', tools: [['read_file', 1], ['write_file', 1], ['list_directory', 1], ['move_file', 0]] },
      { n: 'figma', i: 'server', d: 'http · 需要授权', sc: 'u', on: 1, st: 'warn', tr: 'http', cmd: 'https://mcp.figma.com/mcp', env: [], tok: '连接后统计', tools: [] },
      { n: 'postgres', i: 'server', d: 'stdio · 连接失败:ECONNREFUSED', sc: 'l', on: 0, st: 'err', tr: 'stdio', cmd: 'npx -y @modelcontextprotocol/server-postgres', env: ['DATABASE_URL'], tok: '—', tools: [] },
    ],
    skills: [
      { n: 'brainstorming', i: 'spark', d: '插件 · 动手前先梳理需求', sc: 'u', on: 1, path: '插件 superpowers / brainstorming', trig: ['想做一个新功能', '帮我设计'], files: [['SKILL.md', 0, 1], ['examples.md', 0]] },
      { n: 'test-driven-development', i: 'spark', d: '插件 · 先写失败的测试', sc: 'u', on: 1, path: '插件 superpowers / test-driven-development', trig: ['补测试', '修这个 bug'], files: [['SKILL.md', 0, 1]] },
      { n: 'release-notes', i: 'spark', d: '项目 · 生成发布说明', sc: 'p', on: 1, path: '.claude/skills/release-notes', trig: ['发布说明', 'changelog', '这个版本改了什么'], files: [['SKILL.md', 0, 1], ['template.md', 0], ['scripts', 0, 0, 1], ['collect.sh', 1]] },
      { n: 'sql-review', i: 'spark', d: '用户 · 审查 SQL', sc: 'u', on: 0, path: '~/.claude/skills/sql-review', trig: ['审查 SQL', '这条查询慢'], files: [['SKILL.md', 0, 1]] },
    ],
    commands: [
      { n: '/review', i: 'slash', d: '项目 · 审查当前改动', sc: 'p', on: 1, desc: '审查当前 git diff', hint: '[PR 编号]', tools: ['Bash(git diff:*)', 'Read', 'Grep'], model: 0, body: '审查当前 git diff,按严重程度列出问题。\n如果提供了 $ARGUMENTS,只审查该 PR 的改动。\n每个问题给出文件、行号与建议的修改。' },
      { n: '/test', i: 'slash', d: '项目 · 跑测试并总结失败', sc: 'p', on: 1, desc: '运行测试并总结失败', hint: '[过滤条件]', tools: ['Bash(cargo test:*)'], model: 1, body: '运行 cargo test $ARGUMENTS,只总结失败的用例与最可能的原因。' },
      { n: '/release', i: 'slash', d: '用户 · 生成并发布版本', sc: 'u', on: 1, desc: '生成发布说明并打 tag', hint: '[版本号]', tools: ['Bash(git:*)', 'Bash(gh:*)'], model: 0, body: '读取自上个 tag 以来的提交,生成发布说明,然后创建 $ARGUMENTS 的 tag 与 GitHub Release。' },
      { n: '/clean', i: 'slash', d: '用户 · 清理构建产物', sc: 'u', on: 1, desc: '清理 target 与 dist', hint: '', tools: ['Bash(rm:*)'], model: 3, body: '清理 target 与 dist 目录,汇报释放的空间。' },
    ],
    agents: [
      { n: 'code-reviewer', i: 'bot', d: '项目 · Sonnet · 只读', sc: 'p', on: 1, desc: '审查代码的正确性与风格,不修改文件', model: 1, color: 1, tools: ['Read', 'Grep', 'Glob'], prompt: '你是严格的代码审查员。只读代码,不修改任何文件。\n按「正确性 → 安全 → 可读性」的顺序给出问题,每条带文件与行号。' },
      { n: 'researcher', i: 'bot', d: '用户 · Haiku · 联网', sc: 'u', on: 1, desc: '联网调研并整理出处', model: 3, color: 3, tools: ['Read', 'WebFetch', 'WebSearch'], prompt: '你负责联网调研。每个结论都要附来源链接,区分事实与推测。' },
      { n: 'test-writer', i: 'bot', d: '项目 · Sonnet · 可编辑', sc: 'p', on: 1, desc: '为改动补充单元测试', model: 1, color: 2, tools: ['Read', 'Grep', 'Glob', 'Edit', 'Write', 'Bash'], prompt: '为最近的改动补充单元测试,覆盖边界与失败路径。先写失败的测试,再确认通过。' },
    ],
    rules: [
      { n: '~/.claude/CLAUDE.md', i: 'fileText', d: '用户 · 1.2 KB', sc: 'u', on: 1, size: '1.2 KB', tok: 320, body: '# 通用约定\n\n- 回答、注释一律中文\n- 先说明理解的需求,再动手\n- 不读取、不输出任何密钥' },
      { n: 'router-demo/CLAUDE.md', i: 'fileText', d: '项目 · 3.4 KB', sc: 'p', on: 1, size: '3.4 KB', tok: 900, body: '# router-demo\n\n- 提交信息用 Conventional Commits\n- 不提交构建产物(target / dist)\n- 改完先 `cargo test`,再提交推送' },
      { n: 'CLAUDE.local.md', i: 'fileText', d: '本地 · 0.2 KB', sc: 'l', on: 1, size: '0.2 KB', tok: 60, body: '# 本地笔记\n\n- 我的测试库端口是 5433' },
    ],
    hooks: [
      { n: 'PreToolUse · Bash', i: 'bolt', d: '项目 · 拦截含 rm -rf 的命令', sc: 'p', on: 1, ev: 0, match: 'Bash', cmd: 'scripts/deny-rm-rf.sh', to: 10, trust: 0 },
      { n: 'PostToolUse · Edit', i: 'bolt', d: '项目 · 保存后运行 cargo fmt', sc: 'p', on: 1, ev: 1, match: 'Edit|Write', cmd: 'cargo fmt --all', to: 10, trust: 1 },
      { n: 'Stop', i: 'bolt', d: '用户 · 回合结束时发系统通知', sc: 'u', on: 1, ev: 2, match: '', cmd: 'osascript -e \'display notification "回合结束"\'', to: 5, trust: 1 },
    ],
  };
  const MARKET = [
    { n: 'frontend-design', by: '官方', d: '让界面设计有风格,而不是千篇一律', stars: '4.2k', inst: 0 },
    { n: 'security-guidance', by: '官方', d: '写代码时提示常见的安全隐患', stars: '2.8k', inst: 0 },
    { n: 'pr-review-toolkit', by: '官方', d: '一组专门审查 PR 的子 Agent', stars: '3.1k', inst: 0 },
    { n: 'code-review', by: '官方', d: '按严重程度审查当前改动', stars: '5.6k', inst: 1 },
    { n: 'superpowers', by: '官方', d: '工程方法论 Skills 合集', stars: '9.4k', inst: 1 },
    { n: 'explanatory-output-style', by: '官方', d: '边做边解释关键决策', stars: '1.1k', inst: 0 },
  ];
  const EVENTS = ['PreToolUse', 'PostToolUse', 'Stop', 'UserPromptSubmit', 'Notification'];
  const MODELS = ['继承会话', 'Sonnet 5.5', 'Opus 5.5', 'Haiku 5.5'];
  const TOOLS = ['Read', 'Grep', 'Glob', 'Edit', 'Write', 'Bash', 'WebFetch', 'WebSearch'];
  const COLORS = ['--c-info', '--c-ok', '--c-vio', '--c-cyan', '--c-pink', '--c-warn'];
  const CATS = {
    plugins: { i: 'plug', t: '插件', d: '插件把 Skills、命令、子 Agent 与钩子打包在一起,可在市场里浏览、安装与启停。', add: '从市场安装', noun: '插件' },
    mcp: { i: 'server', t: 'MCP 服务器', d: '给 Claude 接入外部工具与数据。列表顶「刷新状态」= 重读配置 + 已启动会话时 mcp_status;未启动时显示配置层并提示。与输入框 MCP 弹层同源视图。', add: '添加服务器', noun: '服务器' },
    skills: { i: 'spark', t: 'Skills', d: 'Claude 按场景自动调用的能力包:一个目录加一份 SKILL.md。', add: '新建 Skill', noun: 'Skill' },
    commands: { i: 'slash', t: '斜杠命令', d: '在输入框里用 / 调用的提示词模板,支持 $ARGUMENTS 参数。', add: '新建命令', noun: '命令' },
    agents: { i: 'bot', t: '子 Agent', d: '带独立上下文与工具权限的专用助手,由主 Agent 委派任务。', add: '新建子 Agent', noun: '子 Agent' },
    rules: { i: 'book', t: '规则与记忆', d: 'CLAUDE.md 与 .claude/rules 里的长期约定,新会话自动加载;层级越靠后优先级越高。', add: '新建规则', noun: '规则文件' },
    hooks: { i: 'bolt', t: '钩子', d: '在工具调用前后自动运行的命令。陌生项目的钩子需要先在「权限与信任」里允许。', add: '添加钩子', noun: '钩子' },
  };

  /* ───────── 小部件 ───────── */
  const cur = (st) => CATS[st.cust] ? st.cust : 'mcp';
  const empty = (i, t, d, a = '') => `<div class="empty" style="padding:36px 12px">${ic(i, 's20 t4')}<b>${t}</b><div class="t3 fs-cap" style="line-height:1.5">${d}</div>${a}</div>`;
  const lbl = (t, c) => `<div class="fc g2"><span class="t3 fs-cap m">${t}</span>${c}</div>`;
  const kv = (rows) => `<div class="kv">${rows.map(([k, v]) => `<span class="t3">${k}</span><span class="${/\/|npx|--|\.md|\.sh/.test(String(v)) ? 'mono' : ''}" style="word-break:break-all">${v}</span>`).join('')}</div>`;
  const dhead = (i, name, extra = '', del = true) => `<div class="f ai g3">${ic(i, 'c-ac')}<b class="grow ell">${name}</b>${extra}${del ? ib('trash', 'sm', '删除', 'data-act="cu-del"') : ''}${ib('more', 'sm', '更多')}</div>`;
  const banner = (lv, icon, text, acts = '') => `<div class="notice ${lv}" style="gap:8px">${ic(icon, { err: 'c-err', warn: 'c-warn', info: 'c-info', ok: 'c-ok' }[lv])}<div class="grow" style="line-height:1.5">${text}</div>${acts}</div>`;
  const selI = (st, c, vis) => { const k = vis.includes(st.cuSel) ? st.cuSel : vis[0]; st.cuSel = k === undefined ? 0 : k; return k; };
  const saveBtn = (extra = '') => `<span class="grow"></span>${extra}${btn('保存', 'pri', '', 'data-act="cu-save"')}`;
  const toolChip = (t, on) => `<span class="chip line tsel ${on ? 'on' : ''}" data-act="chip-tog" tabindex="0">${t}</span>`;

  /* ───────── 列表 ───────── */
  const visible = (st, c) => DATA[c].map((r, k) => [r, k]).filter(([r, k]) => (st.cuScope || 'all') === 'all' || r.sc === st.cuScope).filter(([r]) => !st.cuQ || (r.n + ' ' + r.d).toLowerCase().includes(st.cuQ.toLowerCase())).map(([, k]) => k);
  const rowHtml = (c, r, k, on) => `<div class="cu-row ${on ? 'on' : ''}" data-act="cu-sel" data-i="${k}" tabindex="0"><span class="si-box">${ic(r.i)}</span><div class="grow"><div class="m f ai g3">${esc(r.n)}${r.st ? `<span class="dot ${r.st}"></span>` : ''}${r.upd ? tag('可更新', 'info') : ''}${r.on ? '' : tag('已停用')}</div><div class="t3 fs-cap ell">${esc(r.d)}</div></div>${tag(SC[r.sc])}${r.st === 'warn' ? btn('授权', 'sm soft', '', 'data-act="mcp-auth"') : ''}${tog(r.on, `data-cu="${k}"`)}</div>`;
  const marketGrid = (st) => `<div class="mk-grid stagger">${MARKET.map((m, k) => `<div class="mk-card ${k === st.cuSel ? 'on' : ''}" data-act="cu-sel" data-i="${k}" tabindex="0"><div class="f ai g3"><span class="si-box">${ic('plug')}</span><div class="grow"><div class="m">${m.n}</div><div class="t3 fs-cap">${m.by} · ★ ${m.stars}</div></div></div><div class="t2 fs-cap" style="line-height:1.5;min-height:34px">${m.d}</div><div class="f ai">${m.inst ? tag('已安装', 'ok') : '<span></span>'}<span class="grow"></span>${m.inst ? '' : btn('安装', 'sm soft', 'download', `data-act="plug-install" data-i="${k}"`)}</div></div>`).join('')}</div>`;
  const list = (st) => {
    const c = cur(st);
    if (c === 'plugins' && st.cuTab === 'market') return marketGrid(st);
    const vis = visible(st, c), sel = selI(st, c, vis), filtering = st.cuQ || (st.cuScope && st.cuScope !== 'all');
    if (!vis.length) return `<div class="card">${filtering ? empty('search', '没有匹配的' + CATS[c].noun, '换个关键字,或放宽范围筛选。', btn('清除筛选', 'sm', 'x', 'data-act="cu-clear"')) : empty(CATS[c].i, '还没有' + CATS[c].noun, CATS[c].d, btn(CATS[c].add, 'pri sm', 'plus', 'data-act="cu-new"'))}</div>`;
    return `<div class="card cu-list">${vis.map(k => rowHtml(c, DATA[c][k], k, k === sel)).join('')}</div><div class="t4 fs-cap" style="padding:6px 4px">${vis.length} / ${DATA[c].length} 个 · 点一行在右侧查看与编辑</div>`;
  };

  /* ───────── 详情 ───────── */
  const DETAIL = {
    plugins: (st) => {
      if (st.cuTab === 'market') { const m = MARKET[st.cuSel] || MARKET[0];
        return `${dhead('plug', m.n, tag(m.by, 'ac'), false)}<div class="t2" style="line-height:1.6">${m.d}。安装后会把它包含的 Skills、命令、子 Agent 与钩子一并加入。</div>${kv([['来源', 'claude-plugins-official'], ['热度', '★ ' + m.stars], ['安装到', '用户(所有项目)']])}${lbl('将包含', `<div class="card" style="padding:2px 4px">${[['spark', 'Skills', 6], ['slash', '斜杠命令', 2], ['bot', '子 Agent', 1]].map(([i, n, v]) => `<div class="row">${ic(i, 's12')}<span class="grow">${n}</span><span class="t3 num">${v}</span></div>`).join('')}</div>`)}<div class="cu-foot f g2">${m.inst ? tag('已安装', 'ok') : btn('安装', 'pri', 'download', `data-act="plug-install" data-i="${st.cuSel}"`)}<span class="grow"></span>${btn('查看源码', 'ghost', 'ext')}</div>`; }
      const r = DATA.plugins[st.cuSel] || DATA.plugins[0], k = DATA.plugins.indexOf(r);
      return `${dhead('plug', esc(r.n), tag('v' + r.ver))}
        ${r.upd ? banner('info', 'download', `有新版本 <b>${r.upd}</b>`, btn('更新', 'sm soft', '', `data-act="plug-update" data-i="${k}"`)) : ''}
        <div class="t2" style="line-height:1.6">${r.about}</div>
        ${kv([['来源', r.src], ['版本', r.ver], ['安装到', seg(['用户', '项目'], r.sc === 'p' ? 1 : 0)]])}
        ${lbl('包含内容', `<div class="card" style="padding:2px 4px">${r.inc.map(([n, v]) => `<div class="row"><span class="grow">${n}</span><span class="t3 num">${v}</span>${tog(1)}</div>`).join('')}</div>`)}
        <div class="cu-foot f g2"><span class="grow"></span>${btn(r.on ? '停用' : '启用', '', '', `data-act="plug-onoff" data-i="${k}"`)}</div>`;
    },
    mcp: (st) => { const r = DATA.mcp[st.cuSel] || DATA.mcp[0], k = DATA.mcp.indexOf(r);
      const bn = r.st === 'ok' ? banner('ok', 'okc', `已连接 · ${r.tools.length} 个工具 · 延迟 23 ms`, btn('重新连接', 'sm', 'refresh', `data-act="mcp-reconnect" data-i="${k}"`))
        : r.st === 'warn' ? banner('warn', 'alert', '需要授权后才能使用', btn('授权', 'sm soft', '', `data-act="mcp-auth" data-i="${k}"`))
        : banner('err', 'errc', `连接失败:${esc(r.d.split(':')[1] || '')}。检查命令与环境变量后重试。`, btn('重试', 'sm', 'refresh', `data-act="mcp-reconnect" data-i="${k}"`));
      return `${dhead('server', esc(r.n))}${bn}
        ${kv([['传输', r.tr], [r.tr === 'http' ? '地址' : '命令', esc(r.cmd)], ['配置层', seg(['用户', '项目', '本地'], 'upl'.indexOf(r.sc))], ['工具定义占用', r.tok]])}
        ${r.env.length ? lbl('环境变量(值已遮盖)', `<div class="card" style="padding:2px 4px">${r.env.map(n => `<div class="row"><span class="mono grow">${n}</span><span class="mono t3">••••••••</span></div>`).join('')}</div>`) : ''}
        <div class="f ai"><b class="grow">工具(${r.tools.length})</b><span class="t3 fs-cap">逐项开关 · 写入本地层 deny 规则</span></div>
        ${r.tools.length ? `<div class="fc" style="max-height:132px;overflow:auto">${r.tools.map(([n, on], j) => `<div class="row"><span class="grow ell mono">${n}</span>${tog(on, `data-cu-tool="${k}:${j}"`)}</div>`).join('')}</div>` : `<div class="t4 fs-cap" style="padding:4px 0">${r.st === 'ok' ? '没有工具' : '连接成功后才会列出工具'}</div>`}
        <div class="card" style="padding:2px 4px"><div class="row" style="height:auto;padding:5px 8px"><div class="grow"><div class="m">按需加载工具(全部 MCP)</div><div class="t3 fs-micro">ENABLE_TOOL_SEARCH · 工具很多时只在需要时加载描述,节省上下文;对所有 MCP 服务器生效,写入 Claude 设置(写在它当前生效的那一层:本地 &gt; 项目 &gt; 用户),新会话起效;使用第三方服务商时会被强制关闭</div></div>${tog(1)}</div></div>
        <div class="cu-foot f g2"><span class="grow"></span>${btn('编辑配置', '', 'edit')}</div>`; },
    skills: (st) => { const r = DATA.skills[st.cuSel] || DATA.skills[0];
      return `${dhead('spark', esc(r.n))}${kv([['范围', SC[r.sc]], ['位置', r.path]])}
        ${lbl('文件', `<div class="card" style="padding:3px 4px">${r.files.map(([n, ind, on, dir]) => `<div class="row ${on ? 'on' : ''}" style="padding-left:${8 + ind * 14}px">${ic(dir ? 'folder' : 'fileText', 's12')}<span class="grow ell mono">${n}</span></div>`).join('')}</div>`)}
        ${lbl('SKILL.md 预览', `<div class="mdprev mono"><span class="t4">---</span>\n<span class="c-vio">name</span>: ${esc(r.n)}\n<span class="c-vio">description</span>: ${esc(r.d.split(' · ')[1] || r.d)}\n<span class="t4">---</span>\n\n# 何时使用\n当用户提到下面这些场景时,先读取本文件再动手。</div>`)}
        ${lbl('会被这些说法触发', `<div class="f g2 wrap">${r.trig.map(t => chip('“' + t + '”', '', 'line')).join('')}</div>`)}
        <span class="grow"></span><div class="f g2">${btn('复制路径', 'ghost', 'copy', 'data-act="toast" data-msg="已复制路径"')}<span class="grow"></span>${btn('在 IDE 中打开', 'soft', 'code', `data-act="open-ide" data-file="${esc(r.n)}/SKILL.md"`)}</div>`; },
    commands: (st) => { const r = DATA.commands[st.cuSel] || DATA.commands[0];
      return `${dhead('slash', esc(r.n))}
        <div class="fc g3">${lbl('名称', field(r.n, 0, { mono: 1, id: 'cmd-n', attr: 'style="width:100%"' }))}${lbl('说明', field(r.desc, 0, { id: 'cmd-d', attr: 'style="width:100%"' }))}${lbl('参数提示', field(r.hint, 0, { mono: 1, ph: '无参数', attr: 'style="width:100%"' }))}</div>
        ${lbl('允许的工具', `<div class="f g2 wrap">${r.tools.map(t => toolChip(t, 1)).join('')}${chip('+ 添加', '', 'line', 'data-act="toast" data-msg="点一下弹出工具选择器(见下方示意)"')}</div>
          <div class="pop" style="width:320px;margin-top:6px" data-spec="tool-picker"><div style="padding:4px 4px 6px"><div class="f ai g2" style="height:28px;padding:0 8px;background:var(--fill-1);border:1px solid var(--bd-2);border-radius:var(--r-sm)"><span class="grow t4 fs-cap">搜索或输入规则,如 Bash(git diff:*)</span>${btn('添加', 'sm soft')}</div></div>
            <div class="cap">内置工具</div>${[['Read', 1], ['Grep', 1], ['Glob', 0], ['Edit', 0], ['Write', 0], ['Bash', 0], ['WebFetch', 0], ['WebSearch', 0]].map(([n, on]) => `<div class="mi ${on ? 'on' : ''}"><span class="grow ell">${n}</span>${on ? ic('check', 's12') : ''}</div>`).join('')}
            <div class="sep"></div><div class="cap">MCP 工具(mcp__服务器 = 整台,再加 __工具名 = 单个)</div>${[['mcp__github', 0], ['mcp__github__create_issue', 0], ['mcp__filesystem__read_file', 1]].map(([n, on]) => `<div class="mi ${on ? 'on' : ''}"><span class="grow ell">${n}</span>${on ? ic('check', 's12') : ''}</div>`).join('')}
            <div class="sep"></div><div class="cap">常用规则</div>${['Bash(git status:*)', 'Bash(git diff:*)', 'Bash(npm run:*)'].map(n => `<div class="mi"><span class="grow ell">${n}</span></div>`).join('')}
            <div class="t3 fs-micro" style="padding:6px 8px 4px">写法错误(括号不配对、空括号、含逗号)时输入框描边变红并抖动,下方写明原因;已存在的规则提示「已经在允许的工具里了」。点已选项 = 移除;改的是草稿,点「保存」才写进 allowed-tools。</div></div>`)}
        ${lbl('模型', seg(MODELS.map(m => m.replace('继承会话', '继承')), r.model))}
        ${lbl('提示词 <span class="t4" style="font-weight:400">$ARGUMENTS 会被命令后面的文字替换</span>', area(r.body, { mono: 1, id: 'cmd-b', rows: 5 }))}
        <div class="t3 fs-cap" style="line-height:1.5">输入 <span class="mono c-ac">${esc(r.n)}${r.hint ? ' 123' : ''}</span> 时,<span class="mono">$ARGUMENTS</span> = <span class="mono">${r.hint ? '123' : '(空)'}</span></div>
        <div class="cu-foot f g2"><span class="grow"></span>${btn('在输入框试用', '', 'send', 'data-act="toast" data-msg="已把命令填进输入框"')}${btn('保存', 'pri', '', 'data-act="cu-save"')}</div>`; },
    agents: (st) => { const r = DATA.agents[st.cuSel] || DATA.agents[0], ro = !r.tools.some(t => ['Edit', 'Write', 'Bash'].includes(t)), all = r.tools.length === TOOLS.length;
      return `${dhead('bot', esc(r.n), `<span class="dot" style="background:var(${COLORS[r.color]})"></span>`)}
        <div class="fc g3">${lbl('名称', field(r.n, 0, { mono: 1, attr: 'style="width:100%"' }))}${lbl('什么时候委派给它', field(r.desc, 0, { attr: 'style="width:100%"' }))}</div>
        <div class="f ai g4">${lbl('模型', seg(MODELS.map(m => m.replace('继承会话', '继承').replace(' 5.5', '')), r.model))}${lbl('颜色', `<span class="f g2">${COLORS.map((c, j) => `<span class="swatch ${j === r.color ? 'on' : ''}" data-act="accent-pick" tabindex="0" style="background:var(${c})"></span>`).join('')}</span>`)}</div>
        ${lbl('工具权限', `${seg(['只读', '可编辑', '全部'], all ? 2 : ro ? 0 : 1, '', 'agent-preset')}<div class="f g2 wrap tgrid" style="margin-top:4px">${TOOLS.map(t => toolChip(t, r.tools.includes(t))).join('')}</div>`)}
        ${lbl('系统提示词', area(r.prompt, { id: 'ag-p', rows: 5 }))}
        <div class="cu-foot f g2"><span class="grow"></span>${btn('试运行', '', 'play', 'data-act="toast" data-msg="已用一个示例任务试运行,结果见新会话"')}${btn('保存', 'pri', '', 'data-act="cu-save"')}</div>`; },
    rules: (st) => { const r = DATA.rules[st.cuSel] || DATA.rules[0];
      const md = r.body.split('\n').map(l => l.startsWith('# ') ? `<div class="b fs-title" style="margin:2px 0 4px">${esc(l.slice(2))}</div>` : l.startsWith('- ') ? `<div class="f g3"><span class="dot ac" style="margin-top:7px"></span><span>${esc(l.slice(2))}</span></div>` : l ? `<div>${esc(l)}</div>` : '<div style="height:6px"></div>').join('');
      return `${dhead('fileText', esc(r.n))}
        <div class="f ai">${seg(['编辑', '预览'], 0, '', 'rule-mode')}<span class="grow"></span><span class="t3 fs-cap num">${r.size} · 约 ${r.tok} token · 占上下文 ${(r.tok / 2000).toFixed(1)}%</span></div>
        <div id="rule-ed">${area(r.body, { mono: 1, id: 'rule-b', rows: 8 })}</div><div id="rule-pv" class="mdprev" hidden>${md}</div>
        ${lbl('加载顺序(后加载的优先)', `<div class="f ai g2 fs-cap">${['用户', '项目', '本地'].map((n, j) => `<span class="chip ${'upl'[j] === r.sc ? 'on' : 'line'}">${n}</span>${j < 2 ? ic('chevR', 's12 t4') : ''}`).join('')}</div>`)}
        <div class="card" style="padding:2px 4px"><div class="row" style="height:auto;padding:5px 8px"><div class="grow"><div class="m">加载 CLAUDE.md 记忆文件</div><div class="t3 fs-micro">关闭后新会话不读任何记忆文件(CLAUDE_CODE_DISABLE_CLAUDE_MDS=1)</div></div>${tog(1)}</div></div>
        <div class="cu-foot f g2"><span class="grow"></span>${btn('在 IDE 中打开', '', 'code', `data-act="open-ide" data-file="${esc(r.n)}"`)}${btn('保存', 'pri', '', 'data-act="cu-save"')}</div>`; },
    hooks: (st) => { const r = DATA.hooks[st.cuSel] || DATA.hooks[0];
      return `${dhead('bolt', esc(r.n))}
        ${r.sc === 'p' && !r.trust ? banner('warn', 'shield', '这个项目的钩子还没有被信任,不会运行。', btn('信任', 'sm soft', '', 'data-act="hook-trust"')) : ''}
        ${lbl('事件', `<div class="f g2 wrap">${EVENTS.map((e, j) => chip(e, '', j === r.ev ? 'on' : 'line', 'data-act="chip-one" tabindex="0"')).join('')}</div>`)}
        <div class="fc g3">${lbl('匹配(工具名,可用 | 分隔)', field(r.match, 0, { mono: 1, ph: '留空 = 全部', attr: 'style="width:100%"' }))}${lbl('命令', area(r.cmd, { mono: 1, id: 'hk-c', rows: 2 }))}</div>
        <div class="f ai g4">${lbl('超时', `<span class="stp" data-min="1" data-max="120" data-step="1"><button data-act="step" data-d="-1">−</button><b class="num">${r.to}</b><span class="t3">秒</span><button data-act="step" data-d="1">+</button></span>`)}${lbl('范围', seg(['用户', '项目', '本地'], 'upl'.indexOf(r.sc)))}</div>
        <div id="hk-out" class="mdprev mono" hidden></div>
        <div class="cu-foot f g2"><span class="grow"></span>${btn('试运行', '', 'play', 'data-act="hook-test"')}${btn('保存', 'pri', '', 'data-act="cu-save"')}</div>`; },
  };
  const drawer = (st) => { const c = cur(st);
    if (c !== 'plugins' || st.cuTab !== 'market') { const vis = visible(st, c); if (!vis.length) return `<div class="t4 fs-cap" style="padding:20px 0;text-align:center">没有可显示的${CATS[c].noun}</div>`; selI(st, c, vis); }
    return DETAIL[c](st); };

  /* ───────── 整页 ───────── */
  const toolbar = (st, c) => `<div class="f ai g3" style="margin-bottom:8px">${inp('筛选' + CATS[c].t, { id: 'cuq', icon: 'search', cls: 'fill', style: 'flex:1', v: esc(st.cuQ || '') })}
    ${c === 'plugins' ? seg(['已安装', '市场'], st.cuTab === 'market' ? 1 : 0, '', 'cu-tab') : ''}${st.cuTab === 'market' ? '' : seg(['全部', '用户', '项目', '本地'], Math.max(0, SCK.indexOf(st.cuScope || 'all')), '', 'cu-scope')}</div>`;
  W.pages.customize = (st) => { const c = cur(st), C = CATS[c];
    return `<div class="cu"><nav class="cu-rail"><div class="grp-h">自定义</div>${Object.entries(CATS).map(([id, x]) => `<div class="row ${id === c ? 'on' : ''}" data-act="cust" data-cust="${id}" tabindex="0">${ic(x.i, 's12')}<span class="grow">${x.t}</span><span class="aux" data-cat-n="${id}">${DATA[id].length}</span></div>`).join('')}</nav>
      <div class="cu-main"><div class="pg-h"><div class="grow"><div class="fs-head b">${C.t}</div><div class="t3" style="max-width:560px;line-height:1.5">${C.d}</div></div>${c === 'plugins' ? btn('浏览市场', '', 'search', 'data-act="cu-tab-market"') : ''}${btn(c === 'plugins' ? '从本地目录安装' : C.add, 'pri', 'plus', c === 'plugins' ? 'data-act="toast" data-msg="已打开目录选择器"' : 'data-act="cu-new"')}</div>
        <div class="pg-split"><div class="pg-main scroll">${toolbar(st, c)}<div id="cu-list">${list(st)}</div></div>
        <aside class="drawer ${['commands', 'agents', 'rules', 'hooks'].includes(c) ? 'wide' : ''}" id="cu-drawer">${drawer(st)}</aside></div></div></div>`;
  };

  /** 新建对话框的表单(每类不同) */
  const newForm = (c) => ({
    mcp: `${lblF('传输', seg(['stdio', 'http'], 0))}${lblF('命令或地址', field('', 0, { mono: 1, id: 'new-x', ph: 'npx -y @modelcontextprotocol/server-…', attr: 'style="width:100%"' }))}`,
    commands: lblF('说明', field('', 0, { id: 'new-x', ph: '这个命令做什么', attr: 'style="width:100%"' })),
    agents: lblF('什么时候委派给它', field('', 0, { id: 'new-x', ph: '例如:审查安全问题', attr: 'style="width:100%"' })),
    hooks: lblF('事件', `<div class="f g2 wrap">${EVENTS.map((e, j) => chip(e, '', j === 0 ? 'on' : 'line', 'data-act="chip-one" tabindex="0"')).join('')}</div>`),
  }[c] || '');
  const lblF = (t, c) => `<div class="fgrp"><span class="lbl">${t}</span>${c}</div>`;
  const NEW_PH = { mcp: 'github', skills: 'my-skill', commands: '/my-command', agents: 'my-agent', rules: 'docs/CLAUDE.md', hooks: 'PostToolUse · Edit' };
  W.cu = {
    DATA, MARKET, CATS, cur, list, drawer, newForm, NEW_PH, SC,
    /** 新建一项并返回它的下标 */
    add(c, name, scope, extra) { const base = { n: name, i: CATS[c].i, sc: scope, on: 1, d: SC[scope] + ' · 刚刚创建' };
      const more = { mcp: { st: 'err', tr: 'stdio', cmd: extra || '', env: [], tok: '—', tools: [], d: SC[scope] + ' · 正在连接…' }, skills: { path: '.claude/skills/' + name, trig: [], files: [['SKILL.md', 0, 1]] }, commands: { desc: extra || '', hint: '', tools: [], model: 0, body: '' }, agents: { desc: extra || '', model: 0, color: 0, tools: ['Read', 'Grep', 'Glob'], prompt: '' }, rules: { size: '0 KB', tok: 0, body: '# ' + name }, hooks: { ev: 0, match: '', cmd: '', to: 10, trust: 0 } }[c] || {};
      DATA[c].push({ ...base, ...more }); return DATA[c].length - 1; },
    remove(c, k) { DATA[c].splice(k, 1); },
    has: (c, n) => DATA[c].some(r => r.n === n),
    setOn(c, k, on) { if (DATA[c][k]) DATA[c][k].on = on ? 1 : 0; },
    setTool(k, j, on) { const t = DATA.mcp[k] && DATA.mcp[k].tools[j]; if (t) t[1] = on ? 1 : 0; },
    visible, MODELS,
  };
})();
