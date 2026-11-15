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
  F.diff = (head = true) => `<div class="tcard">${head ? `<div class="tcard-h">${ic('diff', 's12')}<span class="grow ell">src/router.rs</span><span class="c-ok mono">+4</span><span class="c-err mono">−1</span>${ib('copy', 'sm')}</div>` : ''}
    <div style="padding:3px 0">${dl('', 41, '    pub fn match_path(&self, path: &str) -> Option<&Handler> {')}${dl('del', 42, '        self.routes.get(path)')}${dl('add', 42, '        let key = normalize(path);')}${dl('add', 43, '        self.routes.get(&key).or(self.fallback.as_ref())')}${dl('', 44, '    }')}</div></div>`;
  /** 终端输出卡。live:运行中(贴底跟随,尾行带光标) */
  F.term = (live) => `<div class="tcard"><div class="tcard-h">${ic('prompt', 's12')}<span class="grow mono">cargo test router</span>${live ? '' : '<span class="c-ok">通过</span>'}${ib('copy', 'sm')}</div>
    <div class="term"><span class="d">   Compiling</span> router-demo v0.3.1
<span class="d">     Running</span> tests/router_test.rs
test matches_ignoring_case ... <span class="p">ok</span>
${live ? 'test matches_trailing_slash ... <span class="caret"></span>' : 'test result: <span class="p">ok</span>. 12 passed; 0 failed <span class="d">in 0.42s</span>'}</div></div>`;
  /** 待办清单。done:已完成条数(其后一条为进行中) */
  F.todo = (done = 3) => `<div class="todo">${['定位大小写敏感的来源', '统一用 normalize 归一化路径', '补充大小写与末尾斜杠的测试'].map((t, k) => `<div class="${k < done ? 'is-done' : k === done ? 'is-now' : 't2'}">${chk(k < done ? 'done' : k === done ? 'now' : '')}<span>${t}</span></div>`).join('')}</div>`;
  /** 子 Agent 卡。run:是否运行中 */
  F.sub = (name, desc, run, stat) => `<div class="sub" data-tool="agent"><div class="sub-h"><span class="si" style="color:var(--tool-agent)">${run ? '<span class="spin"></span>' : ic('bot', 's12')}</span><b class="${run ? 'shimmer' : ''}">${name}</b><span class="t3 ell grow">${desc}</span>${run ? ib('stop', 'sm', '停止') : ic('check', 's12 c-ok')}</div>
    <div class="stat" style="padding-left:22px">${stat}</div>${run ? `<div class="ticker" style="margin-left:22px" data-ticks="grep|to_lowercase;read|handlers/user.rs;grep|unwrap()"></div>` : ''}</div>`;
  F.think = () => '<div class="think-body">路由表用原始路径做键,注册与匹配两处都没有归一化。最小改动是抽一个 normalize,两处共用,顺带处理末尾斜杠。</div>';

  /** 权限卡(就地替换输入框) */
  F.perm = () => `<div class="dockcard a-pop-up a-glow"><div class="hd">${ic('shield', 'c-ac')}允许运行这条命令吗?<span class="grow"></span>${tag('Bash', 'ac')}</div>
    <div class="cmd">cargo test router</div>
    <div class="f ai g3">${btn('允许一次', 'pri sm', '', 'data-act="perm-ok"')}${btn('本会话始终允许', 'sm')}${btn('拒绝', 'sm ghost')}<span class="grow"></span><span class="t3 fs-cap f ai g2">${kbd('⏎')}允许 ${kbd('esc')}拒绝</span></div></div>`;
  /** 提问卡 */
  F.ask = () => `<div class="dockcard"><div class="hd">${ic('info', 'c-ac')}末尾斜杠要怎么处理?<span class="grow"></span><span class="t3 fs-cap m">1 / 2</span></div>
    <div class="fc g2"><div class="opt on">${chk('done')}<span class="m">忽略末尾斜杠</span><span class="t3 fs-cap">/users/ 与 /users 等价(推荐)</span></div><div class="opt">${chk()}<span class="m">严格区分</span><span class="t3 fs-cap">保持现状</span></div></div>
    <div class="f ai g3"><span class="grow"></span>${btn('跳过', 'sm ghost')}${btn('下一题', 'pri sm')}</div></div>`;
  /** 计划卡。streaming:生成中(无按钮) */
  F.plan = (streaming) => `<div class="plan" data-tool="plan"><div class="f ai g3">${ic('list', 's12 c-warn')}<b class="${streaming ? 'shimmer' : ''}">路由大小写修复计划</b>${tag(streaming ? '生成中…' : '待批准', streaming ? '' : 'warn')}<span class="grow"></span>${btn('查看', 'sm ghost', 'ext')}</div>
    <ol><li>抽出 <span class="mono">normalize</span>,注册与匹配共用</li><li>匹配失败时回退到 fallback</li><li class="${streaming ? 'caret' : ''}">补 2 条测试并跑通</li></ol>
    ${streaming ? '' : `<div class="f ai g3">${btn('开始实施', 'pri sm', 'play')}${btn('自动模式', 'sm')}${btn('继续规划', 'sm ghost')}</div>`}</div>`;
  F.note = (icon, t) => `<div class="note-line"><span class="si" style="width:16px;display:grid;place-items:center">${ic(icon, 's12')}</span>${t}</div>`;
  F.notice = (lv, t, acts = '') => `<div class="notice ${lv}">${ic(lv === 'err' ? 'errc' : 'alert', lv === 'err' ? 'c-err' : 'c-warn')}<div class="grow">${t}</div>${acts}</div>`;

  const MD = ['<h3>已修复</h3>', '<ul><li>新增 <code>normalize()</code>:去掉末尾斜杠并转小写,<code>add</code> 与 <code>match_path</code> 共用。</li><li>匹配不到时回退到 <code>fallback</code>,行为与原先一致。</li><li>补了 2 条测试,<code>cargo test router</code> 12 项全部通过。</li></ul>', '<p>如果希望保留大小写敏感的路由,可以给 <code>Router</code> 加一个开关,我可以接着做。</p>'];
  const LEAD = '问题出在路由表直接用原始路径做键:注册和匹配都没有归一化,所以 /Users 找不到 /users。';
  F.md = () => `<div class="md"><p>${LEAD}</p>${MD.join('')}</div>`;
  F.tfoot = () => `<div class="tfoot">${ib('copy', 'sm', '复制')}${ib('refresh', 'sm', '重试')}${ib('fork', 'sm', '从这里分叉')}<span class="meta"><span>Sonnet 5.5</span>·<span>1 分 13 秒</span>·<span>↑ 44.1k ↓ 624</span>·<span>缓存 90%</span>·<span>$0.02</span></span></div>`;

  /** 输入框上方的任务坞 */
  F.tdock = () => `<div class="tdock">${chip(`子代理 <b class="num">2</b>`, 'bot', 'line')}${chip(`后台命令 <b class="num">1</b>`, 'terminal', 'line')}</div>`;
  F.queue = () => `<div class="queue"><div class="row"><span class="t4 num">1</span><span class="grow ell">顺便把 README 里的路由示例也更新一下</span><span class="acts">${ib('bolt', 'sm', '立即引导')}${ib('trash', 'sm', '删除')}</span></div></div>`;
  /** 运行配置弹层:权限模式 + 模型 + 思考强度合在一个弹层 */
  F.cfgPop = () => `<div class="pop cfg-pop a-pop-up"><div class="cap">权限模式</div>
    ${[['shield', '逐项确认', 1], ['edit', '自动接受编辑'], ['list', '计划模式'], ['bolt', '跳过权限确认']].map(([n, t, on]) => `<div class="mi ${on ? 'on' : ''}">${ic(n, 's12')}<span class="grow">${t}</span>${on ? ic('check', 's12') : ''}</div>`).join('')}
    <div class="sep"></div><div class="cap">模型</div>
    ${[['Fable 5.2', '最强'], ['Opus 5.5', ''], ['Sonnet 5.5', '', 1], ['Haiku 5.5', '最快']].map(([t, d, on]) => `<div class="mi ${on ? 'on' : ''}"><span class="grow">${t}</span><span class="t3 fs-cap">${d}</span>${on ? ic('check', 's12') : ''}</div>`).join('')}
    <div class="sep"></div><div class="f ai jb" style="padding:4px 8px 6px"><span class="t3 fs-cap m">思考强度</span>${seg(['关', '低', '中', '高'], 2)}</div></div>`;
  /** 输入框。o.focus / o.text(已输入内容)/ o.running(回合进行中,发送变停止)/ o.pop(展开运行配置) */
  F.composer = (o = {}) => `<div class="cmp ${o.focus ? 'focus' : ''}">${o.pop ? F.cfgPop() : ''}
    <div class="cmp-in ${o.text ? 'v' : ''}">${o.text || '描述任务,@ 引用文件,/ 调用命令'}</div>
    <div class="cmp-bar">${ib('plus', 'tip-u', '添加附件')}<span class="cfg ${o.pop ? 'on' : ''}" data-act="cfg">${ic('shield', 's12')}逐项确认<i></i>Sonnet 5.5<i class="dock-wide"></i><span class="dock-wide f ai g2">${ic('brain', 's12')}中</span>${ic('chevD', 's12 car')}</span>${chip(`MCP <b class="num">3</b>`, 'plug')}
      <span class="grow"></span><span class="chip tip-u" data-tip="上下文 76k / 200k"><span class="num dock-wide">38%</span><span class="ring" style="--p:38"></span></span>${o.running ? ib('stop', 'stop') : ib('send', 'send')}</div></div>`;

  /* ───────── 一个完整回合的脚本:静态渲染与重播共用 ───────── */
  const exploreBody = `<div class="nest">${F.step('read', 'router.rs', 'src/ · L1–86', '86 行')}${F.step('grep', '"match_path"', 'src/', '5 处')}${F.step('read', 'router_test.rs', 'tests/', '64 行')}</div>`;
  F.script = [
    { kind: 'block', html: () => F.user(`路由匹配对大小写敏感:<code class="mono">/Users</code> 和 <code class="mono">/users</code> 应该命中同一个处理器。看一下 ${F.ref('router.rs')},修好并补上测试。`), wait: 500 },
    { kind: 'step', run: 1100, html: (s) => F.step('think', '定位大小写敏感的来源', '', s ? '' : '4 秒', { state: s, body: F.think() }) },
    { kind: 'step', run: 2400,
      html: (s) => F.step('group', s ? '项目代码' : '2 个文件 · 1 次搜索', '', '', { state: s, verb: s ? '正在探索' : '探索了', body: exploreBody }) + (s ? '<div class="ticker" data-ticks="read|router.rs;grep|match_path;read|router_test.rs"></div>' : '') },
    { kind: 'step', run: 1200, html: (s) => F.step('edit', 'router.rs', 'src/', s ? '' : '<span class="c-ok">+4</span><span class="c-err">−1</span>', { state: s, open: !s, body: F.diff(false) }) },
    { kind: 'step', run: 800, html: (s) => F.step('edit', 'router_test.rs', 'tests/', s ? '' : '<span class="c-ok">+12</span>', { state: s }) },
    { kind: 'perm', wait: 1500 },
    { kind: 'step', run: 1700, html: (s) => F.step('bash', 'cargo test router', '', s ? '' : '2.4 秒', { state: s, open: true, body: F.term(!!s) }) },
    { kind: 'step', run: 500, html: (s) => F.step('todo', '3 / 3 完成', '', '', { state: s, body: F.todo(3) }) },
    { kind: 'stream', lead: LEAD, blocks: MD },
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
