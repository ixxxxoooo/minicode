/* 设计稿 v4 · 设置页「预览」:外观类设置改了就在行下面看到效果
   原则:凡是改了会让界面「长得不一样」的设置(字号 / 行距 / 内容宽度 / 工具调用详细程度 / 流光 / 等待动画 / 等待指示),
   都在该行或该组下方放一块 .set-pv 预览,用真实的会话流组件(F.*)与真实令牌画,不画示意图。
   预览随控件即时变化:委托监听设置页里的点击,按「行标题」读回各控件当前值再重画。
   Go 对应:app/settings_preview.go(settingsPreview 行类型,每个预览是一个 func(a *App, c *ui.Context))。
   @author ixxxxoooo */
(function () {
  const { ic } = U;
  const $ = (s, r = document) => r.querySelector(s), $$ = (s, r = document) => [...r.querySelectorAll(s)];

  /* ───────── 控件读回:按设置行标题找行,取分段的序号 / 步进器的数值 ───────── */
  const rowOf = (title) => $$('#vmain .set-row').find(r => r.dataset.k === title);
  const segAt = (title, d = 0) => { const r = rowOf(title), s = r && $('.seg', r); return s ? +(s.style.getPropertyValue('--i') || d) : d; };
  const stpAt = (title, d) => { const r = rowOf(title), n = r && $('.stp .num', r); return n ? +n.textContent : d; };

  /* ───────── 字号预设:与 Go typography_prefs.go 的 fontPresetPairs 一致 ───────── */
  const PRESETS = [[90, 95], [100, 100], [110, 105], [125, 115]];
  const LH = [1.45, 1.6, 1.75];
  const FLOW_W = [560, 640, 712, 860, 1040, 1400], SET_W = [620, 760, 900, 1080, 1400];
  const px = (n) => Math.round(n * 2) / 2;

  /* 1) 字体与尺寸:一小段真实会话(用户消息 + 助手正文 + 两行工具 + 命令输出),所有字号走令牌变量,改变量即整体缩放 */
  const typo = () => {
    const ui = stpAt('界面大小', 100) / 100, fs = stpAt('会话正文字号', 100) / 100, lh = LH[segAt('正文行距', 1)];
    const v = { '--fs-micro': 10.5 * ui, '--fs-ctl': 11.5 * ui, '--fs-cap': Math.max(12 * ui, 11.5), '--fs-body': 13 * ui, '--fs-read': 14 * ui * fs, '--fs-title': 14 * ui * fs, '--fs-mono': 12 * ui * fs };
    const style = Object.entries(v).map(([k, n]) => `${k}:${px(n)}px`).join(';') + `;--lh-read:${lh}`;
    return `<div class="set-pv" data-pv="typo"><span class="pv-tag">预览</span>
      <div class="pv-body fc g4" style="${style}">
        ${F.user('路由匹配对大小写敏感:<code>/Users</code> 和 <code>/users</code> 应该命中同一个处理器。' + F.ref('router.rs'))}
        <div class="md"><p>问题出在路由表用原始路径作键。最小改动是抽一个 <code>normalize</code>,注册与匹配共用,顺带处理末尾斜杠。</p></div>
        <div class="acts-g">${F.step('read', 'router.rs', 'src/', '212 行')}${F.step('edit', 'router.rs', 'src/', '<span class="c-ok">+4</span><span class="c-err">−1</span>')}${F.step('bash', 'cargo test router', '', '12 passed')}</div>
      </div>
      <div class="pv-note t3">界面 ${Math.round(13 * ui * 10) / 10} · 阅读 ${Math.round(14 * ui * fs * 10) / 10} · 代码 ${Math.round(12 * ui * fs * 10) / 10} · 行距 ${lh}</div></div>`;
  };

  /* 2) 内容宽度:窗口示意 + 内容列占比(按 1280 宽窗口里主区 ≈ 740 的会话栏与整窗设置主栏画) */
  const width = () => {
    const fi = segAt('会话内容宽度', 2), si = segAt('设置页内容宽度', 1);
    const flowFull = fi === 5, setFull = si === 4;
    const pct = (w, base) => Math.min(100, Math.round(w / base * 100));
    const fp = flowFull ? 100 : pct(FLOW_W[fi], 900), sp = setFull ? 100 : pct(SET_W[si], 1000);
    const mini = (label, p, val, body) => `<div class="pv-win"><div class="pv-win-h t3">${label}<span class="grow"></span><b class="t2 num">${val}</b></div>
      <div class="pv-win-b"><div class="pv-col" style="width:${p}%">${body}</div></div></div>`;
    return `<div class="set-pv" data-pv="width"><span class="pv-tag">预览</span><div class="f g5">
      ${mini('会话', fp, flowFull ? '占满' : FLOW_W[fi] + 'px', '<i></i><i></i><i class="s"></i><i></i><b></b>')}
      ${mini('设置页', sp, setFull ? '占满' : SET_W[si] + 'px', '<i class="h"></i><i></i><i></i><i class="s"></i>')}</div>
      <div class="pv-note t3">窗口较窄时内容列自动收窄,不会出现横向滚动条。</div></div>`;
  };

  /* 3) 工具调用详细程度:三档各画一遍,当前档高亮、其余淡出 */
  const tools = () => {
    const k = segAt('工具调用详细程度', 1);
    const cap = ['相邻的工具调用合并成一行摘要', '只合并连续的读取 / 搜索;编辑与命令各占一行,点开看详情', '每个工具调用一行,编辑的差异与命令输出直接显示在行下'][k];
    const sum = `<div class="pv-sum">${ic('layers', 's12')}<span>编辑了 2 个文件、运行了 2 条命令</span>${ic('chevR', 's12 t4')}</div>`;
    const std = `<div class="acts-g">${F.step('edit', 'src/router.rs', '', '<span class="c-ok">+4</span><span class="c-err">−1</span>', { body: F.diff(false) })}${F.step('edit', 'README.md', '', '<span class="c-ok">+5</span><span class="c-err">−1</span>', { body: '<span class="t3">…</span>' })}${F.step('bash', 'cargo test router', '', '', { body: '<span class="t3">…</span>' })}${F.step('bash', 'cargo check', '', '', { body: '<span class="t3">…</span>' })}</div>`;
    const det = `<div class="acts-g">${F.step('edit', 'src/router.rs', '', '<span class="c-ok">+4</span><span class="c-err">−1</span>', { open: true, body: F.diff(false) })}${F.step('bash', 'cargo test router', '', '', { open: true, body: '<div class="tcard"><div class="term">test result: <span class="p">ok</span>. 14 passed</div></div>' })}</div>`;
    return `<div class="set-pv" data-pv="tools"><span class="pv-tag">预览</span><div class="pv-fix">${[sum, std, det][k]}</div><div class="pv-note t3">${cap}</div></div>`;
  };

  /* 4) 流光样式:三种「正在进行」的文字效果并排,当前选中的那一个高亮 */
  const shimmer = () => {
    const k = segAt('流光样式', 0), on = (i) => i === k ? '' : ' dim';
    return `<div class="set-pv" data-pv="shimmer"><span class="pv-tag">预览</span><div class="f g5 ai">
      <div class="pv-cell${on(0)}"><span class="shimmer">正在分析路由表…</span><span class="t4 fs-cap">流光</span></div>
      <div class="pv-cell${on(1)}"><span class="shimmer-legacy">正在分析路由表…</span><span class="t4 fs-cap">呼吸</span></div>
      <div class="pv-cell${on(2)}"><span class="t2">正在分析路由表…</span><span class="t4 fs-cap">无</span></div></div></div>`;
  };

  /* 5) 等待字符动画:六种风格的字符帧与动词(与 Go spinner_style.go 一致),选中的那种在跑 */
  const SPIN = [
    ['经典', ['·', '✢', '✳', '✶', '✻', '✽', '✻', '✶', '✳', '✢'], '思考中'], ['趣味', ['·', '✢', '✳', '✶', '✻', '✽', '✻', '✶', '✳', '✢'], '炼丹中'],
    ['Claude Code', ['·', '✢', '✳', '✶', '✻', '✽', '✻', '✶', '✳', '✢'], '沉思中'], ['诗意', ['◜', '◠', '◝', '◞', '◡', '◟'], '静待花开中'],
    ['极客', ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'], '遍历语法树中'], ['极简', ['●', '◉', '○', '◉'], '工作中'],
  ];
  let spinTimer = 0;
  const spinner = () => {
    const k = segAt('等待字符动画', 2), [name, , verb] = SPIN[k];
    return `<div class="set-pv" data-pv="spinner"><span class="pv-tag">预览</span><div class="f ai g4"><span class="pv-glyph" data-frames="${SPIN[k][1].join('')}">${SPIN[k][1][0]}</span>
      <span class="t2">${verb}…</span><span class="t4 fs-cap num">12 秒 · ↓ 1.2k tokens · 按 Esc 中断</span></div><div class="pv-note t3">${name}:状态行与回合进行中的字符。</div></div>`;
  };
  const tickSpin = () => { clearInterval(spinTimer); const g = $('.pv-glyph'); if (!g || document.body.classList.contains('no-motion')) return; const fr = [...g.dataset.frames]; let i = 0; spinTimer = setInterval(() => { if (!g.isConnected) return clearInterval(spinTimer); g.textContent = fr[++i % fr.length]; }, 150); };

  /* 6) 等待指示样式:四种图标各画在一行步骤里 */
  const wait = () => {
    const k = segAt('等待指示样式', 0), dim = (i) => i === k ? '' : ' dim';
    const sty = [ic('hourglass', 's12 hg-wait'), '<span class="spin"></span>', '<span class="wait-dots"><i></i><i></i><i></i></span>', '<span class="wait-pulse"></span>'];
    const nm = ['沙漏', '圆环', '点阵', '脉冲'];
    return `<div class="set-pv" data-pv="wait"><span class="pv-tag">预览</span><div class="f g5">${sty.map((h, i) => `<div class="pv-cell${dim(i)}"><div class="step run" data-tool="bash"><span class="si">${h}</span><span class="sv shimmer">运行</span><span class="so">cargo test</span></div><span class="t4 fs-cap">${nm[i]}</span></div>`).join('')}</div></div>`;
  };

  W.pv = { typo, width, tools, shimmer, spinner, wait };

  /* ───────── 联动:预设 → 步进器;步进器 → 预设;任何控件变了 → 重画所有预览 ───────── */
  const setStp = (title, v) => { const r = rowOf(title), n = r && $('.stp .num', r); if (n) n.textContent = v; };
  const setSeg = (title, i) => { const r = rowOf(title), s = r && $('.seg', r); if (!s) return; s.style.setProperty('--i', i); $$('button', s).forEach((b, k) => b.classList.toggle('on', k === i)); };
  const refresh = () => {
    $$('.set-pv[data-pv]').forEach(el => { const f = W.pv[el.dataset.pv]; if (!f) return; const t = document.createElement('div'); t.innerHTML = f(); el.replaceWith(t.firstElementChild); });
    tickSpin();
  };
  const bind = () => { const hook = () => {
    refresh();
    const main = $('#vmain');
    if (!main || main._pvBound) return;
    main._pvBound = true;
    main.addEventListener('click', (e) => {
      if (!e.target.closest('.set-grp')) return;
      /* 等控件自己先处理完(分段 / 步进器的 on 与数值在 app.js / behave.js 里更新),再读回 */
      setTimeout(() => {
        const row = e.target.closest('.set-row');
        const key = row && row.dataset.k;
        if (key === '字号预设') { const i = segAt('字号预设', 1); if (PRESETS[i]) { setStp('界面大小', PRESETS[i][0]); setStp('会话正文字号', PRESETS[i][1]); } }
        else if (key === '界面大小' || key === '会话正文字号') {
          const ui = stpAt('界面大小', 100), fs = stpAt('会话正文字号', 100), hit = PRESETS.findIndex(([u, f]) => u === ui && f === fs);
          setSeg('字号预设', hit < 0 ? 4 : hit);
        }
        refresh();
      }, 0);
    });
  };
  /* app.js 在本脚本之后才创建 APP 并首次渲染:等 load 再挂钩子,并补画一次 */
  window.addEventListener('load', () => { window.APP.hooks.push(hook); hook(); });
  };
  bind();
})();
