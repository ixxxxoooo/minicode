/* 设计稿 v3 · 会话流部件:步骤行、工具卡、交互卡、输入框,以及可重播的一个完整回合
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, kbd, seg, chk, rust, TOOLS } = U;
  const F = {};

  /**
   * 步骤行。tool:工具名(见 U.TOOLS);obj:对象;meta:等宽附注;right:右侧结果
   * o.state:'' 完成 / run 运行中 / err 失败 / deny 已拒绝;o.body:展开内容;o.open:默认展开;o.verb:覆盖动词
   */
  F.step = (tool, obj, meta = '', right = '', o = {}) => {
    const t = TOOLS[tool], st = o.state || '', sh = st === 'run' ? 'shimmer' : '';
    const icon = st === 'run' ? '<span class="spin"></span>' : ic(st === 'err' ? 'errc' : st === 'deny' ? 'lock' : t.icon, 's12');
    return `<div class="step ${st} ${o.open ? 'open' : ''}" data-tool="${tool}" ${o.body ? 'data-act="fold"' : ''}>
      <span class="si">${icon}</span><span class="sv ${sh}">${o.verb || t.verb}</span><span class="so ell ${sh}">${obj}</span>${meta ? `<span class="sm ell">${meta}</span>` : ''}
      <span class="sr">${right}${o.body ? ic('chevR', 's12 chev') : ''}</span></div>${o.body ? `<div class="fold"><div class="fold-in"><div>${o.body}</div></div></div>` : ''}`;
  };
  /** 「最新一步」行。tool:工具名;obj:对象 */
  F.tick = (tool, obj, cls = '') => `<div class="${cls}">${ic(TOOLS[tool].icon, 's12')}<span class="shimmer">${TOOLS[tool].verb} ${obj}</span></div>`;

  F.user = (html) => `<div class="msg-u">${html}</div>`;
  F.ref = (name, icon = 'file') => `<span class="ref">${ic(icon, 's12')}${name}</span>`;

  const dl = (k, no, tx) => `<div class="dl ${k}"><span class="no">${no}</span><span class="tx">${rust(tx)}</span></div>`;
  /** 差异卡。head:是否带文件头 */
  F.diff = (head = true) => `<div class="tcard">${head ? `<div class="tcard-h" data-act="open" data-id="diff:router.rs">${ic('code', 's12')}<span class="grow ell">src/router.rs</span><span class="c-ok mono">+4</span><span class="c-err mono">−2</span>${ib('ext', 'sm', '在工作区打开差异')}${ib('copy', 'sm')}</div>` : ''}
    <div style="padding:3px 0">${dl('', 41, '    pub fn match_path(&self, path: &str) -> Option<&Handler> {')}${dl('del', 42, '        self.routes.get(path)')}${dl('add', 42, '        let key = normalize(path);')}${dl('add', 43, '        self.routes.get(&key).or(self.fallback.as_ref())')}${dl('', 44, '    }')}</div></div>`;
  /** 终端输出卡。live:运行中(贴底跟随,尾行带光标) */
  F.term = (live) => `<div class="tcard"><div class="tcard-h">${ic('prompt', 's12')}<span class="grow mono sh"><b class="fn">cargo</b> <b class="k">test</b> router <span class="sf">--</span> <span class="sf">--nocapture</span> <span class="so">2&gt;&amp;1</span> <span class="so">|</span> <b class="fn">tail</b> <span class="sf">-20</span></span>${live ? '' : '<span class="c-ok">通过</span>'}${ib('terminal', 'sm', '在终端面板查看', 'data-act="pterm"')}${ib('copy', 'sm')}</div>
    <div class="term"><span class="d">   Compiling</span> router-demo v0.3.1
<span class="d">     Running</span> <span class="pa">tests/router_test.rs</span>
<span class="e">warning:</span> unused variable: <span class="s">\`key\`</span> <span class="d">-&gt;</span> <span class="pa">src/router.rs:42:13</span>
test matches_ignoring_case ... <span class="p">ok</span>
${live ? 'test matches_trailing_slash ... <span class="caret"></span>' : 'test result: <span class="p">ok</span>. 12 passed; 0 failed <span class="d">in 0.42s</span>'}</div></div>`;
  /** 待办清单。done:已完成条数(其后一条为进行中) */
  F.todo = (done = 3) => `<div class="todo">${['定位大小写敏感的来源', '统一用 normalize 归一化路径', '补充大小写与末尾斜杠的测试'].map((t, k) => `<div class="${k < done ? 'is-done' : k === done ? 'is-now' : 't2'}">${chk(k < done ? 'done' : k === done ? 'now' : '')}<span>${t}</span></div>`).join('')}</div>`;
  /** 子 Agent 卡。run:是否运行中 */
  F.sub = (name, desc, run, stat) => `<div class="sub" data-tool="agent" data-act="open" data-id="sub:handlers"><div class="sub-h"><span class="si" style="color:var(--tool-agent)">${run ? '<span class="spin"></span>' : ic('bot', 's12')}</span><b class="${run ? 'shimmer' : ''}">${name}</b><span class="t3 ell grow">${desc}</span>${run ? ib('stop', 'sm', '停止') : ic('check', 's12 c-ok')}</div>
    <div class="stat" style="padding-left:22px">${stat}</div>${run ? `<div class="ticker" style="margin-left:22px" data-ticks="grep|to_lowercase;read|handlers/user.rs;grep|unwrap()"></div>` : ''}</div>`;
  F.think = () => '<div class="think-body">路由表用原始路径做键,注册与匹配两处都没有归一化。最小改动是抽一个 normalize,两处共用,顺带处理末尾斜杠。</div>';

  /** 权限卡(就地替换输入框) */
  F.perm = () => `<div class="dockcard a-pop-up a-glow"><div class="hd">${ic('shield', 'c-ac')}允许运行这条命令吗?<span class="grow"></span>${tag('Bash', 'ac')}</div>
    <div class="cmd">cargo test router</div>
    <div class="f ai g3">${btn('允许一次', 'pri sm', '', 'data-act="perm-ok"')}${btn('本会话始终允许', 'sm')}${btn('拒绝', 'sm ghost')}<span class="grow"></span><span class="t3 fs-cap f ai g2">${kbd('⏎')}允许 ${kbd('esc')}拒绝</span></div></div>`;
  /** 提问卡(可多题):顶部步骤条可直接跳到任意一题,底部「上一题 / 下一题(最后一题是提交)」。o.cur 默认第几题(0 起) */
  const QS = [
    { q: '末尾斜杠要怎么处理?', short: '末尾斜杠', multi: false, opts: [['忽略末尾斜杠', '/users/ 与 /users 等价(推荐)', 1], ['严格区分', '保持现状', 0]] },
    { q: '要给哪些路由补测试?', short: '补测试', multi: true, opts: [['/users', '', 1], ['/users/:id', '', 1], ['/health', '', 0], ['其它…', '输入下面的说明', 0]] }];
  F.QS = QS;
  F.ask = (o = {}) => { const cur = o.cur || 0, n = o.single ? 1 : QS.length;
    return `<div class="dockcard qa" data-cur="${cur}"><div class="hd">${ic('info', 'c-ac')}<span class="grow ell">${n > 1 ? '需要你确认 ' + n + ' 个问题' : QS[0].q}</span>${n > 1 ? `<span class="t3 fs-cap m num">${cur + 1} / ${n}</span>` : ''}</div>
    ${n > 1 ? `<div class="qa-steps">${QS.slice(0, n).map((x, k) => `<span class="qa-step ${k === cur ? 'cur' : ''} ${k === 0 ? 'done' : ''}" data-act="qa-step" data-i="${k}"><span class="n">${k === 0 ? ic('check', 's12') : k + 1}</span><span class="ell">${x.short}</span></span>`).join('')}</div>` : ''}
    ${QS.slice(0, n).map((x, k) => `<div class="qp fc g2" data-i="${k}">${n > 1 ? `<div class="m" style="font-weight:600">${x.q}</div>` : ''}${x.multi ? '<div class="t3 fs-cap">可多选</div>' : ''}${x.opts.map(([t, d, on]) => `<div class="opt ${on ? 'on' : ''}" data-act="qa-opt">${chk(on ? 'done' : '')}<span class="m">${t}</span><span class="t3 fs-cap">${d}</span></div>`).join('')}</div>`).join('')}
    <div class="f ai g3">${n > 1 ? btn('上一题', 'sm ghost', 'arrowL', 'data-act="qa-go" data-d="-1"') : ''}<span class="grow"></span><span class="t3 fs-cap f ai g2">${n > 1 ? kbd('←') + kbd('→') + '切换 ' : ''}${kbd('⏎')}${n > 1 ? '下一题' : '提交'}</span>${btn('跳过', 'sm ghost')}${n > 1 ? `${btn('下一题', 'pri sm qa-next', '', 'data-act="qa-go" data-d="1"')}${btn('提交', 'pri sm qa-last')}` : btn('提交', 'pri sm')}</div></div>`; };
  /** 已回答的提问:会话流里的卡片(问题 + 回答)。rows:[[问题, [回答…], 'skip'|'free'|'']] */
  F.askDone = (rows) => { rows = rows || [['末尾斜杠要怎么处理?', ['忽略末尾斜杠'], ''], ['要给哪些路由补测试?', ['/users', '/users/:id'], '']];
    return `<div class="qa-done" data-tool="ask"><div class="hd">${ic('check', 'c-ok')}<span class="grow">已回答 ${rows.length} 个问题</span>${rows.some(r => r[2] === 'skip') ? tag(rows.filter(r => r[2] === 'skip').length + ' 个已跳过') : ''}</div>
    ${rows.map(([q, a, k], i) => `<div class="qr"><div class="qq"><span class="n">${i + 1}</span>${q}</div><div class="qa-a">${(k === 'skip' ? ['已跳过'] : a).map(x => `<span class="ans ${k === 'skip' ? 'skip' : k === 'free' ? 'free' : ''}">${x}</span>`).join('')}</div></div>`).join('')}</div>`; };
  /** 计划卡。streaming:生成中(无按钮) */
  F.plan = (streaming, state) => { const ok = state === 'approved';
    return `<div class="plan" data-tool="plan"><div class="f ai g3">${ic('list', 's12 c-warn')}<b class="${streaming ? 'shimmer' : ''}">路由大小写修复计划</b>${tag(streaming ? '生成中…' : ok ? '已批准' : '待批准', streaming ? '' : ok ? 'ok' : 'warn')}<span class="grow"></span>${btn('查看', 'sm ghost', 'ext', 'data-act="open" data-id="plan"')}</div>
    <ol><li>抽出 <span class="mono">normalize</span>,注册与匹配共用</li><li>匹配失败时回退到 <span class="mono">fallback</span></li><li class="${streaming ? 'caret' : ''}">补 2 条测试并跑通</li></ol>
    ${streaming || ok ? '' : `<div class="f ai g3">${btn('开始实施', 'pri sm', 'play')}${btn('自动模式', 'sm opt-sec')}${btn('继续规划', 'sm ghost')}</div>`}</div>`; };
  F.note = (icon, t) => `<div class="note-line"><span class="si" style="width:16px;display:grid;place-items:center">${ic(icon, 's12')}</span>${t}</div>`;
  F.notice = (lv, t, acts = '') => `<div class="notice ${lv}">${ic(lv === 'err' ? 'errc' : 'alert', lv === 'err' ? 'c-err' : 'c-warn')}<div class="grow">${t}</div>${acts}</div>`;

  const MD = ['<h3>已修复</h3>', '<ul><li>新增 <code>normalize()</code>:去掉末尾斜杠并转小写,<code>add</code> 与 <code>match_path</code> 共用。</li><li>匹配不到时回退到 <code>fallback</code>,行为与原先一致。</li><li>补了 2 条测试,<code>cargo test router</code> 12 项全部通过。</li></ul>', '<p>如果希望保留大小写敏感的路由,可以给 <code>Router</code> 加一个开关,我可以接着做。</p>'];
  const LEAD = '问题出在路由表直接用原始路径做键:注册和匹配都没有归一化,所以 /Users 找不到 /users。';
  F.md = () => `<div class="md"><p>${LEAD}</p>${MD.join('')}</div>`;
  F.tfoot = () => `<div class="tfoot">${ib('copy', 'sm', '复制')}${ib('refresh', 'sm', '重试')}${ib('fork', 'sm', '从这里分叉')}<span class="meta"><span>Sonnet 5.5</span>·<span>1 分 13 秒</span>·<span>↑ 44.1k ↓ 624</span>·<span>缓存 90%</span>·<span>$0.02</span></span></div>`;

  /** 输入框上方的任务坞:宽时按类别分胶囊,窄时并成一个「任务 N」;点胶囊弹出列表,点列表项在工作区打开对应标签 */
  F.tdock = () => `<div class="tdock">${chip(`子代理 <b class="num">2</b>`, 'bot', 'line split', 'data-act="tpop" data-k="agent"')}${chip(`后台命令 <b class="num">1</b>`, 'terminal', 'line split', 'data-act="tpop" data-k="cmd"')}${chip(`任务 <b class="num">3</b>`, 'activity', 'line merged', 'data-act="tpop" data-k="all"')}</div>`;
  F.tpop = (k) => { const spin = '<span class="spin" style="margin-top:3px"></span>', row = (ico, t, d, time, id) => `<div class="row" data-act="open" data-id="${id}"><span style="margin-top:2px;flex:none">${ico}</span><div class="grow" style="min-width:0"><div class="m ell" style="color:var(--tx-1)">${t}</div><div class="t3 fs-cap ell">${d}</div></div><span class="aux">${time}</span></div>`;
    const agents = row(spin, '审查 handlers 的错误处理', '搜索 "expect("', '48 秒', 'sub:handlers') + row(ic('check', 's12 c-ok'), '调研 axum 的路由匹配', '已完成 · 1 分 20 秒', '昨天', 'sub:handlers');
    const cmds = row(spin, 'cargo watch -x test', '等待文件变化…', '6 分', 'task:watch');
    return `<div class="tpop pop a-pop-up">${k !== 'cmd' ? `<div class="cap">子代理</div>${agents}` : ''}${k !== 'agent' ? `<div class="cap">后台命令</div>${cmds}` : ''}</div>`; };
  /** 回合结束的改动卡:代替旧的「检查器 · 改动」标签,入口是「审查」 */
  F.chg = () => `<div class="chg-card">${ic('diff', 's12 c-ac')}<b>3 个文件已修改</b><span class="c-ok mono fs-cap">+18</span><span class="c-err mono fs-cap">−2</span><div class="files">${['router.rs', 'router_test.rs', 'case_test.rs'].map(n => `<span class="ref" data-act="open" data-id="${n === 'router.rs' ? 'diff:router.rs' : n}">${ic('code', 's12')}${n}</span>`).join('')}</div>${btn('撤销', 'sm ghost', 'undo')}${btn('审查', 'sm soft', 'eye', 'data-act="open" data-id="review"')}</div>`;
  /** 待发送区(输入框正上方):排队与引导共用。items:[{k:'queue'|'steer', t}];o.paused 暂停态 */
  F.queue = (items, o = {}) => { items = items || [{ k: 'steer', t: '先别动 handlers,只改 router.rs' }, { k: 'queue', t: '对照截图修 UI', img: 2 }, { k: 'queue', t: '顺便把 README 里的路由示例也更新一下' }, { k: 'queue', t: '', img: 1 }];
    return `<div class="queue"><div class="qh">${ic('clock', 's12')}<b>待发送 ${items.length}</b><span class="grow"></span><span class="f ai g2">${kbd('⏎')}排队<span style="width:6px"></span>${kbd('⌘⏎')}引导</span></div>
    ${o.paused ? `<div class="qwarn">${ic('alert', 's12 c-warn')}<span class="grow t2 fs-cap">回合被中断,排队已暂停</span>${btn('继续', 'sm soft')}${btn('撤回', 'sm ghost')}</div>` : ''}
    ${items.map((it, n) => it.k === 'steer'
      ? `<div class="row steer"><span class="qk">${ic('bolt', 's12')}引导</span><span class="grow ell">${it.t || '(图片)'}</span>${it.img ? `<span class="t3 fs-cap f ai g2">${ic('image', 's12')}${it.img} 张图</span>` : ''}<span class="qs"><span class="spin"></span>等 Claude 接收</span></div>`
      : `<div class="row queue-i"><span class="qk"><span class="num">${n + 1}</span>排队</span><span class="grow ell" data-tip="点击退回输入框编辑">${it.t || '(图片)'}</span>${it.img ? `<span class="t3 fs-cap f ai g2">${ic('image', 's12')}${it.img} 张图</span>` : ''}<span class="acts">${ib('bolt', 'sm', '改为引导:现在就插进当前回合')}${ib('stop', 'sm', '停止并发送')}${ib('trash', 'sm', '删除')}</span></div>`).join('')}</div>`; };
  /** 回到最新:输入框正上方居中的圆形向下箭头。fresh:离开底部后又有新内容 */
  F.toLatest = (fresh) => `<div class="to-latest ${fresh ? 'fresh' : ''}" data-act="to-latest" data-tip="回到最新" role="button" aria-label="回到最新">${ic('arrowDown', 's12')}</div>`;
  /** 运行配置弹层:权限模式 + 按服务商分组的模型 + 思考强度(详情见 providers.js) */
  F.cfgPop = () => (typeof PV !== 'undefined' ? PV.cfgPopHtml() : `<div class="pop cfg-pop a-pop-up"></div>`);
  /** 输入框。o.focus / o.text(已输入内容)/ o.running(回合进行中,发送变停止)/ o.pop(展开运行配置) */
  F.composer = (o = {}) => `<div class="cmp ${o.focus ? 'focus' : ''}">${o.pop ? F.cfgPop() : ''}${o.ctx ? F.ctxPop({ state: o.ctx === true ? 'ok' : o.ctx }) : ''}
    <div class="cmp-in ${o.text ? 'v' : ''}">${o.text || (o.running ? 'Claude 正在工作 · ⏎ 排队,⌘⏎ 引导' : '描述任务,@ 引用文件,/ 调用命令')}</div>
    <div class="cmp-bar">${ib('plus', 'tip-u', '添加附件')}<span class="cfg ${o.pop ? 'on' : ''} f ai g2" data-act="cfg">${ic('shield', 's12')}逐项确认<i></i><span class="cfg-mod ell">${typeof PV !== 'undefined' ? PV.cfgChipHtml() : 'Sonnet 5.5'}</span><i class="dock-wide"></i><span class="dock-wide f ai g2">${ic('brain', 's12')}中</span>${ic('chevD', 's12 car')}</span>${chip(`<span class="lbl">MCP</span> <b class="num">3</b>`, 'plug', 'mcp-chip')}
      <span class="grow"></span><span class="chip tip-u ctx-chip ${o.ctx ? 'on' : ''}" data-tip="上下文 76k / 200k" data-act="ctx" tabindex="0" role="button" aria-label="上下文用量">${o.ctx ? '' : ''}<span class="num dock-wide">38%</span><span class="ring" style="--p:38"></span></span>${o.running ? ib('stop', 'stop') : ib('send', 'send')}</div></div>`;

  /* ───────── 一个完整回合的脚本:静态渲染与重播共用 ───────── */
  const exploreBody = `<div class="nest">${F.step('read', 'router.rs', 'src/ · L1–86', '86 行')}${F.step('grep', '"match_path"', 'src/', '5 处')}${F.step('read', 'router_test.rs', 'tests/', '64 行')}</div>`;
  F.script = [
    { kind: 'block', html: () => F.user(`路由匹配对大小写敏感:<code class="mono">/Users</code> 和 <code class="mono">/users</code> 应该命中同一个处理器。看一下 ${F.ref('router.rs')},修好并补上测试。`), wait: 500 },
    { kind: 'step', run: 1100, html: (s) => F.step('think', '定位大小写敏感的来源', '', s ? '' : '4 秒', { state: s, body: F.think() }) },
    { kind: 'block', html: () => F.plan(false, 'approved'), wait: 300 },
    { kind: 'block', html: () => F.sub('审查 handlers 的错误处理', 'src/handlers', true, '<span>48 秒</span><span>12 个工具</span><span>6.2k</span>'), wait: 300 },
    { kind: 'step', run: 2400,
      html: (s) => F.step('group', s ? '项目代码' : '2 个文件 · 1 次搜索', '', '', { state: s, verb: s ? '正在探索' : '探索了', body: exploreBody }) + (s ? '<div class="ticker" data-ticks="read|router.rs;grep|match_path;read|router_test.rs"></div>' : '') },
    { kind: 'step', run: 1200, html: (s) => F.step('edit', 'router.rs', 'src/', s ? '' : '<span class="c-ok">+4</span><span class="c-err">−2</span>', { state: s, open: !s, body: F.diff(false) }) },
    { kind: 'step', run: 800, html: (s) => F.step('edit', 'router_test.rs', 'tests/', s ? '' : '<span class="c-ok">+12</span>', { state: s }) },
    { kind: 'perm', wait: 1500 },
    { kind: 'step', run: 1700, html: (s) => F.step('bash', 'cargo test router', '', s ? '' : '2.4 秒', { state: s, open: true, body: F.term(!!s) }) },
    { kind: 'step', run: 500, html: (s) => F.step('todo', '3 / 3 完成', '', '', { state: s, body: F.todo(3) }) },
    { kind: 'stream', lead: LEAD, blocks: MD },
    { kind: 'block', html: F.chg, wait: 0 },
    { kind: 'block', html: F.tfoot, wait: 0 },
  ];
  /** 整个回合的静态 HTML(每个条目包一层 .it,连续步骤收进 .acts-g) */
  F.turn = () => {
    let out = '', open = false;
    for (const s of F.script) {
      if (s.kind === 'perm') continue;
      const step = s.kind === 'step';
      if (step && !open) { out += '<div class="acts-g">'; open = true; }
      if (!step && open) { out += '</div>'; open = false; }
      out += `<div class="it">${s.kind === 'stream' ? F.md() : s.html('')}</div>`;
    }
    return out + (open ? '</div>' : '');
  };

  window.F = F;
})();
