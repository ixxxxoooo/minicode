/* Agent 界面的公共部件:侧栏、会话流、输入框、右栏 */
(function () {
  const { i, chip, bdg, cnt, kbd, tg, seg, dot, ib, btn, logo, meter, hl } = K;
  const C = {};

  /* ───────── 侧栏 ───────── */
  const st = {
    run: '<span class="spin"></span>',
    ask: '<span class="dot ring"></span>',
    unread: '<span class="dot info"></span>',
    idle: '<span class="dot" style="opacity:.45"></span>',
  };
  const ss = (state, title, sub, on) => `<div class="ss ${on ? 'on' : ''}"><span class="st">${st[state]}</span><div class="grow"><div class="t ell">${title}</div><div class="u">${sub}</div></div></div>`;
  const ss1 = (state, title, right, on) => `<div class="ss one ${on ? 'on' : ''}"><span class="st" style="height:auto">${st[state]}</span><div class="grow ell">${title}</div><span class="tx3 s10 nw">${right}</span></div>`;
  C.ss = ss; C.ss1 = ss1; C.st = st;

  // 侧栏方案 A:图标条 + 状态筛选 + 双行会话
  C.sbA = (on = 0, opts = {}) => `
  <div class="sb">
    <div class="tb">${K.tl()}<span class="grow"></span>${ib('panelL', 'sm')}${ib('search', 'sm')}</div>
    <div class="fc g2" style="padding:2px 8px 6px">
      <div class="sb-row" style="color:var(--tx);font-weight:500">${i('plus', 'acc')}<span class="grow">新对话</span>${kbd('⌘', 'N')}</div>
    </div>
    <div class="sb-nav" title="自动化 / 画布 / 浏览器 / 自定义">
      ${[['timer', '自动化'], ['canvas', '画布'], ['globe', '浏览器'], ['plug', '自定义']].map(([n, t], k) => `<span title="${t}" class="${opts.nav === k ? 'on' : ''}">${i(n)}</span>`).join('')}
    </div>
    <div class="sb-sec" style="padding-bottom:4px"><span class="grow nw">工作区</span>${ib('plus', 'sm')}</div>
    <div class="f g4" style="padding:0 10px 4px">
      <span class="chip on" style="height:20px;padding:0 7px;font-size:10.5px">全部</span><span class="chip" style="height:20px;padding:0 7px;font-size:10.5px">运行中 ${cnt(1, 'mute')}</span><span class="chip" style="height:20px;padding:0 7px;font-size:10.5px">待处理 ${cnt(1)}</span>
    </div>
    <div class="grow fc" style="overflow:hidden;padding-top:2px">
      <div class="proj">${i('chevD', 's12 tx3')}${i('folder', 'tx2')}<span class="grow ell">router-demo</span><span class="tx3 s10 m">4</span>${ib('plus', 'sm')}</div>
      ${ss('run', '完整修复一个 Bug:路由大小写匹配', `${i('branch', 's12')}main <span class="okc">+12</span> <span class="errc">−3</span> · 运行中`, on === 0)}
      ${ss('ask', '权限请求:修改配置与执行命令', `<span class="acc">等待你的许可</span> · 5 分钟`, on === 1)}
      ${ss('unread', '文件读写:梳理并整理路由层', `${i('branch', 's12')}feat/router <span class="okc">+48</span> <span class="errc">−12</span> · 1 小时`, on === 2)}
      ${ss('idle', 'Markdown 与代码高亮:发布说明', `${i('branch', 's12')}main · 昨天`, on === 3)}
      <div class="proj" style="margin-top:6px">${i('chevD', 's12 tx3')}${i('folder', 'tx2')}<span class="grow ell">web-dashboard</span><span class="tx3 s10 m">2</span></div>
      ${ss('idle', '内置浏览器:排查指标看板报错', `${i('branch', 's12')}main · 2 小时`)}
      ${ss('idle', '子 Agent:代码审查与调研', `${i('branch', 's12')}main · 昨天`)}
      <div class="proj" style="margin-top:6px">${i('chevR', 's12 tx3')}${i('folder', 'tx2')}<span class="grow ell">docs-site</span></div>
      <div class="proj">${i('chevR', 's12 tx3')}${i('folder', 'tx2')}<span class="grow ell">scenario-lab</span></div>
    </div>
    ${C.sbFoot()}
  </div>`;

  C.sbFoot = () => `
    <div style="padding:6px 10px 0;border-top:1px solid var(--bd)">
      <div class="f ai jb s10 tx3" style="margin-bottom:3px"><span>5 小时额度</span><span class="m">38%</span></div>${meter(38)}
    </div>
    <div class="sb-foot" style="border-top:none">${ib('gear')}<span class="bdg ac" style="margin-left:2px">演示模式</span><span class="grow"></span><span class="chip" style="color:var(--ac-tx);background:var(--ac-soft)">${i('download', 's12')}重启更新</span></div>`;

  // 侧栏方案 B:图标轨 + 会话抽屉(扁平、按时间)
  C.railB = (on = 'chat') => {
    const it = (n, t, k) => `<div class="fc ai" style="gap:1px;padding:5px 0;width:44px;border-radius:7px;color:${k === on ? 'var(--tx)' : 'var(--tx2)'};background:${k === on ? 'var(--bg-active)' : 'transparent'}">${i(n, 's16')}<span style="font-size:9.5px">${t}</span></div>`;
    return `<div class="fc ai" style="width:66px;background:var(--bg-chrome);border-right:1px solid var(--bd);flex:none;padding-bottom:8px">
      <div class="tb" style="padding:0 0 0 12px;width:100%">${K.tl()}</div>
      <div style="margin:6px 0 8px">${logo()}</div>
      <div class="fc g2">${it('bot', '会话', 'chat')}${it('timer', '自动化', 'auto')}${it('canvas', '画布', 'canvas')}${it('globe', '浏览器', 'web')}${it('plug', '自定义', 'cust')}</div>
      <span class="grow"></span>
      <div class="fc ai g4">${ib('bell')}${ib('gear')}<span class="avatar sm" style="border-radius:50%">L</span></div>
    </div>`;
  };
  C.drawerB = (on = 0) => `
  <div class="fc" style="width:236px;background:var(--bg-chrome);border-right:1px solid var(--bd);flex:none;min-height:0">
    <div class="tb" style="padding-top:2px">
      <span class="chip line grow" style="height:26px;justify-content:space-between;font-weight:600;color:var(--tx)"><span class="f ai g6">${i('folder', 'tx2')}全部项目</span>${i('chevUD', 's12 tx3')}</span>${ib('plus', 'ac')}
    </div>
    <div style="padding:0 10px 6px"><div class="inp fill">${i('search', 's12')}<span class="grow">搜索会话…</span>${kbd('⌘', 'K')}</div></div>
    <div style="padding:0 10px 4px">${seg(['全部', '运行中 1', '待处理 1', '已归档'], 0, 'sm')}</div>
    <div class="sb-sec" style="padding-top:6px">今天</div>
    ${ss('run', '完整修复一个 Bug:路由大小写匹配', `<span class="bdg" style="height:14px">router-demo</span> main · 运行中`, on === 0)}
    ${ss('ask', '权限请求:修改配置与执行命令', `<span class="bdg" style="height:14px">router-demo</span> <span class="acc">待许可</span> · 5 分钟`, on === 1)}
    ${ss('unread', '文件读写:梳理并整理路由层', `<span class="bdg" style="height:14px">router-demo</span> +48 −12 · 1 小时`, on === 2)}
    ${ss('idle', '内置浏览器:排查指标看板报错', `<span class="bdg" style="height:14px">web-dashboard</span> · 2 小时`)}
    <div class="sb-sec">昨天</div>
    ${ss('idle', 'Markdown 与代码高亮:发布说明', `<span class="bdg" style="height:14px">router-demo</span> main`)}
    ${ss('idle', '子 Agent:代码审查与调研', `<span class="bdg" style="height:14px">web-dashboard</span> main`)}
    <div class="sb-sec">更早</div>
    ${ss('idle', '2000 轮长会话:滚动与加载', `<span class="bdg" style="height:14px">scenario-lab</span> · 3 天前`)}
    <span class="grow"></span>
    <div style="padding:6px 10px 8px;border-top:1px solid var(--bd)">
      <div class="f ai jb s10 tx3" style="margin-bottom:3px"><span>5 小时额度 · 每周 61%</span><span class="m">38%</span></div>${meter(38)}
    </div>
  </div>`;

  /* ───────── 会话流 ───────── */
  const step = (ic, vb, ob, mt = '', rt = '', cls = '') => `<div class="step ${cls}"><span class="ic">${i(ic, 's12')}</span><span class="vb">${vb}</span><span class="ob ell">${ob}</span>${mt ? `<span class="mt ell">${mt}</span>` : ''}<span class="rt">${rt}</span></div>`;
  C.step = step;
  const diffCard = (file = 'router.rs', dir = 'src/', open = true) => `
    <div class="card" style="overflow:hidden">
      <div class="f ai g6" style="height:26px;padding:0 8px;border-bottom:1px solid var(--bd-soft)">${i('chevD', 's12 tx3')}${i('file', 's12 acc')}<span class="m">${file}</span><span class="tx3 s11">${dir}</span><span class="grow"></span><span class="okc mono">+4</span><span class="errc mono">−1</span>${ib('copy', 'sm')}</div>
      <div style="padding:3px 0">
        <div class="dl ctx"><span class="no">41</span><span class="tx">${hl('{k fn} {fn match_path}(&self, p: &str) -> {ty Option}<{ty Route}> {')}</span></div>
        <div class="dl del"><span class="no">42</span><span class="tx">${hl('    self.routes.get({k p})')}</span></div>
        <div class="dl add"><span class="no">42</span><span class="tx">${hl('    {k let} key = p.to_ascii_lowercase();')}</span></div>
        <div class="dl add"><span class="no">43</span><span class="tx">${hl('    {k let} key = key.trim_end_matches(\'/\');')}</span></div>
        <div class="dl add"><span class="no">44</span><span class="tx">${hl('    self.routes.get(key)')}</span></div>
        <div class="dl ctx"><span class="no">45</span><span class="tx">}</span></div>
      </div>
    </div>`;
  C.diffCard = diffCard;
  const md = () => `<div class="md"><p>已修复。根因是 <code>Router::match_path</code> 直接拿原始字符串去比,<code>/Users</code> 与 <code>/users</code> 被当成两条路由。</p>
    <ul><li>匹配前统一转小写并去掉末尾 <code>/</code></li><li>新增 <code>tests/case.rs</code>,覆盖 5 种大小写组合</li><li><code>cargo test</code> 全部通过(12 项)</li></ul></div>`;
  C.md = md;
  C.footer = (extra = '') => `<div class="foot-stat">${ib('copy', 'sm')}${ib('fork', 'sm')}${ib('rewind', 'sm')}<span>Sonnet 5.5 · 用时 1分13秒 · ↑ 44.1k ↓ 624 · 缓存 90% · $0.02 · 3 分钟前</span>${extra}</div>`;
  C.user = (t = '路由匹配对大小写敏感,/Users 打不开 /users 页面,帮我修复并加上测试') => `<div class="msg-user">${t}</div>`;
  C.think = (t = '已思考 8 秒') => `<div class="step"><span class="ic">${i('brain', 's12')}</span><span class="vb">${t}</span>${i('chevR', 's12 tx3')}</div>`;

  // 工具步骤 · 方案 A:单行流(现状的整理版:统一动词色、结果右对齐)
  C.stepsA = () => `
    ${C.think()}
    ${step('layers', '读取', '3 个文件', '搜索 1 次 · router.rs, case.rs, utils.rs', i('chevR', 's12'))}
    ${step('edit', '编辑', 'router.rs', 'src/', `<span class="okc mono">+4</span><span class="errc mono">−1</span>${i('chevD', 's12')}`)}
    <div class="nest" style="margin:2px 0 4px 7px">${diffCard()}</div>
    ${step('prompt', '运行', 'cargo test', '12 passed · 3.2s', `<span class="okc">${i('check', 's12')}</span>`)}
    ${step('todo', '待办', '3 / 3 已完成', '', i('chevR', 's12'))}`;

  // 工具步骤 · 方案 B:时间线竖轨
  C.stepsB = () => {
    const r = (ic, vb, ob, mt, t, extra = '', tone = '') => `
      <div class="f g10" style="position:relative">
        <div class="fc ai" style="width:16px;flex:none"><span style="width:16px;height:16px;border-radius:50%;background:var(--bg-card);border:1px solid var(--bd-strong);display:grid;place-items:center;color:${tone || 'var(--tx2)'};z-index:1">${i(ic, 's12')}</span><span class="grow" style="width:1px;background:var(--bd);margin:1px 0"></span></div>
        <div class="grow" style="padding-bottom:8px;min-width:0"><div class="f ai g6" style="height:16px"><span class="tx3">${vb}</span><span class="m ell">${ob}</span><span class="tx3 mono s11 ell">${mt}</span><span class="grow"></span><span class="tx3 s10">${t}</span></div>${extra}</div>
      </div>`;
    return `
      <div style="margin-top:2px">
      ${r('brain', '思考', '路由匹配与大小写', '', '8s')}
      ${r('layers', '读取 3 · 搜索 1', 'router.rs · case.rs · utils.rs', '', '1.2s')}
      ${r('edit', '编辑', 'router.rs', '+4 −1', '0.4s', `<div style="margin-top:4px">${diffCard()}</div>`, 'var(--ac)')}
      ${r('prompt', '运行', 'cargo test', '12 passed', '3.2s', '', 'var(--ok)')}
      ${r('todo', '待办', '3 / 3 完成', '', '')}
      </div>`;
  };

  // 工具步骤 · 方案 C:折叠卡(一回合一张摘要卡)
  C.stepsC = () => `
    <div class="card" style="overflow:hidden">
      <div class="f ai g8" style="height:30px;padding:0 10px;background:var(--bg-sunk);border-bottom:1px solid var(--bd-soft)">
        ${i('chevD', 's12 tx3')}<span class="m">本轮执行了 5 步</span><span class="tx3">· 用时 12 秒</span><span class="grow"></span>
        ${bdg('读 3', '')}${bdg('搜 1', '')}${bdg('改 1', 'ac')}${bdg('跑 1', 'ok')}
      </div>
      <div style="padding:4px 10px">
        ${step('layers', '读取', 'router.rs', 'src/', '<span class="tx3">0.3s</span>')}
        ${step('layers', '读取', 'case.rs', 'src/', '<span class="tx3">0.2s</span>')}
        ${step('search', '搜索', 'match_path', '2 处', '<span class="tx3">0.5s</span>')}
        ${step('edit', '编辑', 'router.rs', '', '<span class="okc mono">+4</span><span class="errc mono">−1</span>')}
        ${step('prompt', '运行', 'cargo test', '12 passed', `<span class="okc">${i('check', 's12')}</span>`)}
      </div>
      <div style="padding:0 10px 8px">${diffCard()}</div>
    </div>`;

  C.turn = (variant = 'A') => `
    <div class="flow">
      ${C.user()}
      ${variant === 'A' ? C.stepsA() : variant === 'B' ? C.stepsB() : C.stepsC()}
      <div style="margin-top:4px">${md()}</div>
      ${C.footer()}
    </div>`;

  /* ───────── 输入框 ───────── */
  const ctxRing = (p = 28) => `<span class="ctx-ring" style="--p:${p}" title="上下文 ${p}%"></span>`;
  C.ctxRing = ctxRing;
  const hint = (extra = '') => `<div class="f ai g8 s11 tx3" style="padding:4px 4px 0;height:22px">${i('branch', 's12')}<span>main</span><span>· 3 项改动</span>${extra}<span class="grow"></span><span>${kbd('⇧', 'Tab')} 切换模式</span></div>`;

  C.cmpA = (o = {}) => {
    const n = o.narrow;
    return `<div class="cmp ${o.focus ? 'focus' : ''}">
      <div class="in ${o.text ? 'v' : ''}">${o.text || '向 Claude 提问,/ 调用命令,@ 引用文件…'}</div>
      <div class="bar">
        ${chip(n ? '' : '接受编辑', 'shield', '', !n)}${chip(n ? 'Son…' : 'Sonnet 5.5', 'spark', '', !n)}${chip(n ? '' : '中', 'brain', '', !n)}${chip(o.mcp || '3', 'plug', 'warnc')}
        <span class="grow"></span>${ctxRing(o.p || 28)}${ib('wand')}${o.running ? `<span class="ib stop">${i('stop')}</span>` : `<span class="ib send ${o.text ? '' : 'off'}">${i('send')}</span>`}
      </div>
    </div>${o.hint === false ? '' : hint()}`;
  };
  C.cmpB = (o = {}) => {
    const n = o.narrow;
    return `<div class="cmp ${o.focus ? 'focus' : ''}">
      <div class="in ${o.text ? 'v' : ''}">${o.text || '向 Claude 提问…'}</div>
      <div class="bar">
        ${ib('plus')}
        <span class="chip line" style="gap:6px;color:var(--tx)">${i('shield', 's12 acc')}${n ? '' : '<span>接受编辑</span><span class="tx3">·</span>'}<span class="m">${n ? 'Sonnet' : 'Sonnet 5.5'}</span>${n ? '' : '<span class="tx3">·</span><span>中</span>'}${i('chevD', 's12 car')}</span>
        ${chip(o.mcp || '3', 'plug', 'warnc')}
        <span class="grow"></span>${ctxRing(o.p || 28)}${o.running ? `<span class="ib stop">${i('stop')}</span>` : `<span class="ib send ${o.text ? '' : 'off'}">${i('send')}</span>`}
      </div>
    </div>${o.hint === false ? '' : hint()}`;
  };
  C.cmpC = (o = {}) => {
    const n = o.narrow;
    return `<div class="cmp ${o.focus ? 'focus' : ''}" style="overflow:hidden">
      <div class="f ai g6" style="height:26px;padding:0 6px 0 8px;background:var(--ac-soft);border-bottom:1px solid var(--bd-soft)">
        ${i('shield', 's12 acc')}<span class="m" style="color:var(--ac-tx)">接受编辑</span><span class="tx3 s11">${n ? '' : '自动应用文件修改,命令仍需确认'}</span><span class="grow"></span>${kbd('⇧', 'Tab')}
      </div>
      <div class="f ai g4" style="padding:6px 8px 0;flex-wrap:wrap"><span class="ref">${i('file', 's12')}router.rs</span><span class="ref">${i('file', 's12')}case.rs</span>${n ? '' : `<span class="chip" style="height:20px;border:1px dashed var(--bd-strong)">${i('plus', 's12')}上下文</span>`}</div>
      <div class="in ${o.text ? 'v' : ''}" style="padding-top:5px">${o.text || '向 Claude 提问…'}</div>
      <div class="bar">${ib('plus')}${ib('at')}${ib('slash')}${n ? '' : ib('image')}${chip(o.mcp || '3', 'plug', 'warnc')}<span class="grow"></span>${chip(n ? 'Sonnet' : 'Sonnet 5.5 · 中', '', 'line', true)}${ctxRing(o.p || 28)}${o.running ? `<span class="ib stop">${i('stop')}</span>` : `<span class="ib send ${o.text ? '' : 'off'}">${i('send')}</span>`}</div>
    </div>${o.hint === false ? '' : hint()}`;
  };
  C.cmp = (v = 'A', o = {}) => ({ A: C.cmpA, B: C.cmpB, C: C.cmpC }[v])(o);

  // 排队卡
  C.queueCard = () => `<div class="dock" style="margin:0 10px"><div class="f ai g6" style="height:26px;padding:0 8px;border-bottom:1px solid var(--bd-soft)">${i('list', 's12 tx3')}<span class="m">排队 2 条</span><span class="tx3">本轮结束后自动发送</span><span class="grow"></span><span class="chip">暂停</span></div>
    <div class="f ai g6" style="height:26px;padding:0 8px"><span class="tx3 mono s11">1</span><span class="grow ell">再补一个 Windows 路径分隔符的测试</span><span class="chip" style="color:var(--ac-tx)">引导</span>${ib('x', 'sm')}</div>
    <div class="f ai g6" style="height:26px;padding:0 8px;border-top:1px solid var(--bd-soft)"><span class="tx3 mono s11">2</span><span class="grow ell">顺便把 CHANGELOG 更新一下</span><span class="chip" style="color:var(--ac-tx)">引导</span>${ib('x', 'sm')}</div></div>`;

  // 弹层
  C.popModel = () => `<div class="pop" style="width:232px">
    <div class="cap">Claude 官方 <span class="acc" style="float:right">使用中</span></div>
    ${['默认', 'Opus 4.5', 'Sonnet 5.5', 'Haiku 4.5'].map((t, k) => `<div class="mi ${k === 2 ? 'on' : ''}">${i('spark', 's12 tx3')}<span class="grow">${t}</span>${k === 2 ? i('check', 's12 acc') : ''}</div>`).join('')}
    <div class="sep"></div><div class="cap">DeepSeek <span class="warnc" style="float:right">未填 API Key · 去设置</span></div>
    ${['deepseek-chat', 'deepseek-reasoner'].map(t => `<div class="mi tx3">${i('spark', 's12')}<span class="grow">${t}</span></div>`).join('')}</div>`;
  C.popConfig = () => `<div class="pop" style="width:288px;padding:8px">
    <div class="f ai jb" style="margin-bottom:6px"><b>运行配置</b><span class="tx3 s11">仅对当前会话</span></div>
    <div class="cap" style="padding:0 0 3px">权限模式</div>
    <div class="fc g2" style="margin-bottom:8px">
      ${[['hand', '逐项确认', '每次修改与命令都问我', 0], ['edit', '接受编辑', '自动改文件,命令仍需确认', 1], ['bolt', '自动模式', '模型自行判断风险', 0], ['shield', '只读规划', '只读文件,不做任何修改', 0]].map(([ic, t, d, on]) => `<div class="mi ${on ? 'hl' : ''}" style="height:30px">${i(ic, 's12')}<div class="grow"><div>${t}</div></div><span class="tx3 s10">${d}</span>${on ? i('check', 's12') : ''}</div>`).join('')}
    </div>
    <div class="cap" style="padding:0 0 3px">模型</div>
    <div style="margin-bottom:8px">${seg(['Opus', 'Sonnet', 'Haiku', '更多 ▾'], 1)}</div>
    <div class="cap" style="padding:0 0 3px">思考强度</div>
    <div class="f ai g8" style="margin-bottom:2px"><span class="tx3 s10">低</span><div class="grow rel" style="height:4px;border-radius:2px;background:var(--bg-active)"><div class="abs" style="left:0;width:50%;top:0;bottom:0;background:var(--ac);border-radius:2px"></div><span class="abs" style="left:calc(50% - 6px);top:-4px;width:12px;height:12px;border-radius:50%;background:#fff;box-shadow:0 0 0 1.5px var(--ac)"></span></div><span class="tx3 s10">极高</span><span class="bdg ac">中</span></div>
  </div>`;
  C.popMode = () => `<div class="pop" style="width:270px">
    ${[['hand', '逐项确认', '每次修改与命令都问我', 0, ''], ['edit', '接受编辑', '自动改文件,命令仍需确认', 1, ''], ['bolt', '自动模式', '模型自行判断风险', 0, ''], ['shield', '计划模式', '只读规划,批准后再动手', 0, ''], ['lock', '跳过权限', '全部放行(谨慎)', 0, 'err']].map(([ic, t, d, on, c]) => `<div class="mi ${on ? 'hl' : ''}" style="height:34px">${i(ic, 's14')}<div class="grow"><div class="${c === 'err' ? 'errc' : ''}">${t}</div><div class="tx3 s10" style="line-height:1.2">${d}</div></div>${on ? i('check', 's12') : ''}</div>`).join('')}</div>`;
  C.popMcp = () => `<div class="pop" style="width:280px">
    <div class="f ai jb" style="padding:4px 8px"><b>MCP</b><span class="tx3 s11">已启用 3 / 5 · 工具定义占上下文 2.1%</span></div>
    ${[['github', 18, 1, ''], ['playwright', 21, 1, ''], ['postgres', '连接失败', 1, 'err'], ['figma', '需要授权', 0, 'warn'], ['sentry', 9, 0, '']].map(([n, c, on, tone]) => `<div class="mi">${i('chevR', 's12 tx3')}${dot(tone === 'err' ? 'err' : tone === 'warn' ? 'warn' : on ? 'ok' : '')}<span class="grow">${n}</span><span class="${tone === 'err' ? 'errc' : tone === 'warn' ? 'warnc' : 'tx3'} s11">${typeof c === 'number' ? c + ' 个工具' : c}</span>${tg(on)}</div>`).join('')}
    <div class="sep"></div><div class="mi tx2">${i('gear', 's12')}管理 MCP…</div></div>`;
  C.popContext = () => `<div class="pop" style="width:260px;padding:10px">
    <div class="f ai jb"><b>上下文</b><span class="tx3 s11 mono">56.2k / 200k</span></div>
    <div style="margin:6px 0;height:6px;border-radius:3px;background:var(--bg-active);display:flex;overflow:hidden"><i style="width:9%;background:var(--ac)"></i><i style="width:11%;background:var(--info)"></i><i style="width:6%;background:var(--ok)"></i><i style="width:2%;background:var(--vio)"></i></div>
    ${[['var(--ac)', '系统与工具', '18.0k'], ['var(--info)', '消息', '22.4k'], ['var(--ok)', '文件', '12.1k'], ['var(--vio)', 'MCP 工具', '3.7k']].map(([c, n, v]) => `<div class="f ai g6" style="height:20px"><i style="width:8px;height:8px;border-radius:2px;background:${c}"></i><span class="grow">${n}</span><span class="mono tx2 s11">${v}</span></div>`).join('')}
    <div class="sep"></div>
    <div class="f ai jb s11"><span class="tx2">5 小时额度</span><span class="m">38%</span></div>${meter(38)}
    <div class="f ai jb s11" style="margin-top:6px"><span class="tx2">每周额度</span><span class="m">61%</span></div>${meter(61, 'warn')}
    <div class="f ai jb s10 tx3" style="margin-top:6px"><span>更新于 2 分钟前</span><span class="acc">刷新</span></div></div>`;

  /* ───────── 右栏 ───────── */
  const rpTabs = (on, h = 36, compact = false) => {
    // compact:栏太窄时只有当前标签显示文字,其余降级为图标(计数变成小圆点)
    const t = (k, ic, n, c) => `<div class="tab ${on === k ? 'on' : ''}" title="${n}">${i(ic, 's12')}${compact && on !== k ? '' : n}${c ? (compact && on !== k ? '<span class="dot ac"></span>' : cnt(c, k === on ? '' : 'mute')) : ''}</div>`;
    return `<div class="tabs" style="height:${h}px">${t('changes', 'diff', '改动', 3)}${t('files', 'folder', '文件')}${t('term', 'terminal', '终端')}${t('plan', 'todo', '计划')}${t('tasks', 'activity', '任务', 2)}<span class="grow"></span>${ib('plus', 'sm')}</div>`;
  };
  C.rpTabs = rpTabs;
  const fileRow = (n, d, a, r, on) => `<div class="fileline ${on ? 'on' : ''}">${i('chevR', 's12 tx3')}${i('file', 's12 acc')}<span class="m">${n}</span><span class="tx3 s11 grow ell">${d}</span><span class="okc mono s11">+${a}</span>${r ? `<span class="errc mono s11">−${r}</span>` : ''}</div>`;
  C.panelChanges = () => `
    <div class="pad8 fc g6" style="min-height:0">
      <div class="f ai g6">${seg(['Claude 改动 3', 'Git 更改 5'], 0, 'sm')}<span class="grow"></span>${btn('全部回退', 'sm ghost')}</div>
      <div class="f ai g6 s11 tx3"><span>3 个文件</span><span class="okc mono">+18</span><span class="errc mono">−4</span><span class="grow"></span><span>左右对比 ${tg(1)}</span></div>
      <div class="card" style="overflow:hidden"><div style="padding:2px 4px 4px">${fileRow('router.rs', 'src/', 4, 1)}</div>
        <div style="padding:2px 0 4px;border-top:1px solid var(--bd-soft)">
          <div class="dl del"><span class="no">42</span><span class="tx">    self.routes.get(p)</span></div>
          <div class="dl add"><span class="no">42</span><span class="tx">    let key = p.to_ascii_lowercase();</span></div>
          <div class="dl add"><span class="no">43</span><span class="tx">    self.routes.get(key)</span></div></div></div>
      <div class="card" style="padding:2px 4px">${fileRow('case.rs', 'tests/', 12, 0)}</div>
      <div class="card" style="padding:2px 4px">${fileRow('CHANGELOG.md', '', 2, 3)}</div>
    </div>`;
  C.panelFiles = () => {
    const r = (d, ic, n, ex, g, on) => `<div class="tr ${on ? 'sel' : ''}" style="padding-left:${8 + d * 14}px">${ex ? i(ex === 1 ? 'chevD' : 'chevR', 's12 tx3') : '<span style="width:12px"></span>'}${i(ic, 's12 ' + (ic === 'folder' ? 'acc' : 'tx2'))}<span class="grow ell">${n}</span>${g ? `<span class="g ${g === 'M' ? 'warnc' : g === 'A' ? 'okc' : 'tx3'}">${g}</span>` : ''}</div>`;
    return `<div class="pad8 fc g6"><div class="inp fill">${i('search', 's12')}<span class="grow">按名称过滤…</span></div></div>
      <div>${r(0, 'folder', 'src', 1)}${r(1, 'file', 'case.rs', 0, 'A')}${r(1, 'file', 'main.rs')}${r(1, 'file', 'router.rs', 0, 'M', 1)}${r(1, 'file', 'utils.rs')}${r(0, 'folder', 'tests', 2)}${r(0, 'folder', 'examples', 2)}${r(0, 'file', 'Cargo.toml')}${r(0, 'file', 'CHANGELOG.md', 0, 'M')}${r(0, 'file', 'README.md')}</div>
      <div style="margin:8px;border-top:1px solid var(--bd);padding-top:8px" class="fc g4"><div class="f ai g6"><span class="m s11">router.rs</span><span class="bdg">预览</span><span class="grow"></span>${btn('在 IDE 打开', 'sm', 'ext')}</div>
      <pre style="margin:0;font:11px/1.6 var(--mono);background:var(--bg-sunk);border-radius:6px;padding:6px 8px;overflow:hidden">${hl('{k pub struct} {ty Router} {\n    routes: {ty HashMap}<{ty String}, {ty Route}>,\n}')}</pre></div>`;
  };
  C.panelTerm = () => `<div class="f ai g2" style="padding:4px 6px;border-bottom:1px solid var(--bd-soft)">
      <span class="chip on">${i('terminal', 's12')}zsh</span><span class="chip">${i('eye', 's12')}cargo test<span class="tx3 s10">只读</span></span><span class="grow"></span>${ib('plus', 'sm')}</div>
    <div class="term"><span class="p">➜</span> router-demo <span class="d">git:(main)</span> cargo test<br><span class="d">   Compiling router v0.3.1</span><br><span class="d">    Finished test in 1.84s</span><br>running 12 tests<br>test case::upper_lower ... <span class="p">ok</span><br>test case::trailing_slash ... <span class="p">ok</span><br>test router::basic ... <span class="p">ok</span><br><br>test result: <span class="p">ok</span>. 12 passed; 0 failed<br><span class="p">➜</span> router-demo <span class="d">git:(main)</span> <span class="pulse">▌</span></div>`;
  C.panelTasks = () => {
    const t = (ic, kind, name, meta, status, tone) => `<div class="card" style="padding:6px 8px"><div class="f ai g6">${i(ic, 's12 tx2')}<span class="m ell grow">${name}</span>${bdg(status, tone)}</div><div class="f ai g6 tx3 s11" style="margin-top:2px"><span>${kind}</span><span>·</span><span class="mono ell">${meta}</span></div></div>`;
    return `<div class="pad8 fc g6"><div class="f ai g4 wrap">${chip('子代理 1', '', 'on')}${chip('命令 2')}${chip('监听 1')}${chip('已结束 3')}</div>
      ${t('bot', '子代理', '审查 search 模块', '5 秒 · 1 次工具', '运行中', 'ac')}
      ${t('prompt', '后台命令', '启动开发服务器', 'pnpm dev · :3000', '运行中', 'ac')}
      ${t('prompt', '后台命令', '编译 release', 'cargo build --release', '12 秒前完成', 'ok')}
      ${t('eye', '监听', '监听服务日志', 'tail -f server.log', '运行中', 'ac')}</div>`;
  };
  C.panelPlan = () => `<div class="pad8 fc g8"><div class="f ai g6"><b class="grow ell">迁移旧 REST 接口到新网关</b>${bdg('待批准', 'warn')}</div>
    <div class="f ai g6">${btn('Build · 逐项确认', 'pri sm')}${btn('自动模式', 'sm')}${btn('继续规划', 'sm ghost')}</div>
    <div class="card pad"><div class="f ai jb s11 tx3" style="margin-bottom:4px"><span>执行进度</span><span>1 / 3</span></div>${meter(33)}</div>
    <div class="md"><h3>步骤</h3><ul><li>新建 <code>gateway/</code> 适配层</li><li>逐个迁移 <code>/api/v1/*</code></li><li>双写一周,对比响应</li></ul><h3>风险</h3><ul><li>老客户端依赖 v1 字段</li></ul></div></div>`;
  C.panel = (k) => ({ changes: C.panelChanges, files: C.panelFiles, term: C.panelTerm, tasks: C.panelTasks, plan: C.panelPlan }[k])();
  C.rp = (on = 'changes', w = 304) => `<div class="rp" style="width:${w}px">${rpTabs(on)}<div class="grow" style="overflow:hidden">${C.panel(on)}</div></div>`;

  // 会话顶栏
  C.head = (o = {}) => `<div class="chat-head">
    ${o.left || ''}${o.noTitle ? '' : `<span class="title ell">完整修复一个 Bug:路由大小写匹配</span>${i('edit', 's12 tx3')}
    <span class="tx3 s11 f ai g4 nw">${i('folder', 's12')}router-demo</span>`}
    <span class="grow"></span>
    ${o.mid || ''}
    ${seg(['Agent', 'IDE'], 0)}${o.right === undefined ? ib('panelR', 'on') : o.right}
  </div>`;

  window.C = C;
})();
