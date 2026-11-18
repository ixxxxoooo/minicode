/* 设计稿 v3.3 · 品牌与图标生成器
   一个母题、五种样式:应用图标(1024 画布、824 连续圆角图形)与菜单栏模板图标(36 画布 = 18pt@2x)都由这里生成,
   所以设计稿里每个出现图标的地方(查看器标题栏、关于、欢迎页、设置预览、Dock / 菜单栏 / 通知仿真)都跟随同一个选择,
   主题、外观(自动 / 浅 / 深 / 透明 / 着色)与样式三个维度可以任意组合。
   默认样式是「提示符」(一根线弯成 ❯ + 光标);「经纬」保留织机母题:经线(墨色)是会话、终端、改动三条主线,纬线(主题色)是 Claude 的思路穿过它们;压穿交替 = 人审批 ⇄ Agent 执行。
   所有线宽比例来自同一组令牌(BAND 112 / GAP 26 / CUT 14,1024 画布),衍生图标按比例缩放。
   可交互的部分(点选样式、切外观)在 brand-act.js,规范页在 brand-spec.js。
   @author ixxxxoooo */
(function () {
  const { ic, tag, seg, tog, THEMES } = U;
  const S = () => (window.APP && window.APP.st) || {};

  /* ───────── 选择项与默认值 ───────── */
  const DEF = { icoStyle: 'prompt', icoMode: 'auto', icoTint: 'follow', trayAnim: 'flow', trayColor: 1, trayCount: 0, trayOn: 1, trayDyn: 1, trayClick: 'menu',
    dockBadge: 'pending', dockBounce: 'once', dockProgress: 1, fileIcon: 0 };
  /** 分段控件的取值表:下标 ↔ 取值 */
  const OPT = { icoMode: ['auto', 'light', 'dark', 'clear', 'tint'], dockBadge: ['off', 'pending', 'running'], dockBounce: ['off', 'once', 'until'], trayAnim: ['flow', 'breathe', 'none'], trayClick: ['menu', 'window'] };
  /** 当前选择。样式取值可能是旧版本存下的(如已删除的 'weave'),不在清单里就退回默认(提示符) */
  const cur = (k) => {
    const v = S()[k] ?? DEF[k];
    if (k === 'icoStyle' && !STYLES.some(s => s.id === v)) return DEF[k];
    return v;
  };
  const themeOf = (id) => THEMES.find(t => t.id === id) || THEMES[0];

  const STYLES = [
    { id: 'prompt', name: '提示符', en: 'Prompt', d: '一根线弯成 ❯,旁边一个光标。最直白的「命令行里的 Claude」。默认。' },
    { id: 'warp', name: '经纬', en: 'Warp', d: '2×2 粗织纹:经线压过纬线、纬线又压过经线;图形最大、小尺寸最清晰。' },
    { id: 'grid', name: '棋盘', en: 'Check', d: '3×3 方块,压穿的节奏变成棋盘;最几何、最安静。' },
    { id: 'lines', name: '行', en: 'Lines', d: '四行代码:缩进、被高亮的片段、长短不一。工具感最强。' },
    { id: 'braces', name: '花括号', en: 'Braces', d: '{ } 抱着一个线结,像在代码里找到了那根线。' },
  ];
  /** 菜单栏图标样式 = 应用图标样式(五种全有,同一个选择):切换时 Dock / ⌘Tab / 访达 / 菜单栏同时换成同一种 */
  const TRAYS = STYLES;
  const TRAY_STATES = [
    ['idle', '空闲', '没有运行中的回合,也没有待处理的事'],
    ['running', '运行中', '至少一个会话在跑回合 / 后台任务;图标动画,不加徽标'],
    ['waiting', '待确认', '有权限请求或提问在等你:右上角「!」徽标,优先级最高'],
    ['done', '已完成', '后台会话的回合刚结束、你还没看:实心小圆点;点开菜单或回到窗口即清除'],
    ['error', '出错', '会话失败或 claude 进程崩溃且自动重启失败:「×」徽标'],
    ['update', '有更新', '已下载新版本、等你重启:向下箭头徽标;低于上面所有状态'],
    ['paused', '已暂停', '「暂停通知」开启:整体半透明 + 暂停徽标'],
    ['demo', '演示模式', '演示后端:空心圆环徽标,避免把演示当真']];

  /* ───────── 调色板:六套主题 × 浅 / 深两套中性色 ───────── */
  const hex = (c) => [1, 3, 5].map(i => parseInt(c.slice(i, i + 2), 16));
  const mix = (a, b, t) => { const A = hex(a), B = hex(b); return '#' + A.map((v, i) => Math.round(v + (B[i] - v) * t).toString(16).padStart(2, '0')).join(''); };
  const PAPER = { ac: ['#FF8A3D', '#EE4A00'], acd: ['#FF9A5C', '#FF5A0F'], L: ['#FFFEFB', '#EAE7DE', '#26251E'], D: ['#2C2B27', '#151411', '#EDEAE2'] };
  /** ac:浅色底上的纬线渐变;acd:深色底上的(更亮);L / D:背景上下色与经线(墨)色 */
  const PAL = {
    paper: PAPER, ink: PAPER,
    mist: { ac: ['#7A92FF', '#3550E0'], acd: ['#93A6FF', '#5A74F5'], L: ['#FDFEFF', '#E3E8F1', '#1A1D24'], D: ['#2B2F3A', '#14161D', '#E6E9F2'] },
    celadon: { ac: ['#2DBBA0', '#0B7765'], acd: ['#4ED6BA', '#16A58E'], L: ['#FDFFFD', '#DFE9E2', '#18201C'], D: ['#262F2B', '#111513', '#E4EEE9'] },
    graphite: { ac: ['#8EA4FF', '#4C69F0'], acd: ['#A3B6FF', '#5F7CFF'], L: ['#FDFDFE', '#E4E5EA', '#1B1C20'], D: ['#2B2D33', '#121316', '#E8EAEE'] },
    aurora: { ac: ['#B9A2FF', '#7A58F0'], acd: ['#CBB8FF', '#8F6DFA'], L: ['#FEFDFF', '#E7E4F4', '#1E1B33'], D: ['#232749', '#0A0C1B', '#E6E8F7'] },
    glow: PAPER, 'glow-night': PAPER, // 流光用 Claude 的陶土橙,图标与宣纸 / 墨夜同一套橙色素材
  };

  /** 解析一枚图标的配色。o.t 固定主题(缺省 = 「配色」设置:跟随当前主题或固定某一套)、o.m 固定外观(缺省 = 设置里的外观) */
  const resolve = (o) => {
    const curTheme = S().theme || 'paper', tint = cur('icoTint');
    const tid = o.t || (tint === 'follow' ? curTheme : tint), p = PAL[tid] || PAPER, mode = o.m || cur('icoMode');
    const dark = mode === 'dark' || (mode === 'auto' && themeOf(o.t || curTheme).dark), th = dark ? p.acd : p.ac;
    if (mode === 'clear') return { kind: 'clear', bg: ['#ffffff', '#ffffff'], bgo: [.4, .12], ink: '#ffffff', th: [mix(th[0], '#ffffff', .55), mix(th[1], '#ffffff', .3)], rim: .85, gloss: 0, shade: 'c' };
    if (mode === 'tint') { const t = p.acd; return { kind: 'tint', bg: [mix(t[1], '#000000', .38), mix(t[1], '#000000', .74)], bgo: [1, 1], ink: mix(t[0], '#ffffff', .84), th: [mix(t[0], '#ffffff', .2), t[0]], rim: .22, gloss: .1, shade: 'd' }; }
    const n = dark ? p.D : p.L;
    return { kind: 'solid', bg: [n[0], n[1]], bgo: [1, 1], ink: n[2], th, rim: dark ? .16 : .95, gloss: dark ? .1 : .5, shade: dark ? 'd' : 'l' };
  };

  /* ───────── 全局定义:连续圆角图形与投影(只注入一次,<use> 引用) ───────── */
  const SQ = (() => { const n = 5, a = 412, pts = []; for (let i = 0; i < 240; i++) { const t = i / 240 * 2 * Math.PI, c = Math.cos(t), s = Math.sin(t);
    pts.push((512 + a * Math.sign(c) * Math.abs(c) ** (2 / n)).toFixed(1) + ' ' + (512 + a * Math.sign(s) * Math.abs(s) ** (2 / n)).toFixed(1)); } return 'M' + pts.join('L') + 'Z'; })();
  const shadow = (id, dy, sd, op) => `<filter id="mc-sh-${id}" x="-20%" y="-20%" width="140%" height="140%"><feDropShadow dx="0" dy="${dy}" stdDeviation="${sd}" flood-color="#000" flood-opacity="${op}"/></filter>`;
  document.body.insertAdjacentHTML('afterbegin', `<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs><path id="mc-sq" d="${SQ}"/>${shadow('l', 12, 16, .2)}${shadow('d', 12, 16, .5)}${shadow('c', 10, 14, .14)}</defs></svg>`);

  /* ───────── 经纬几何:按「压 / 穿」把每根带子切成若干段,不依赖背景色挖洞,所以透明 / 着色外观也成立 ───────── */
  /** 带独立圆角的矩形路径。顺序:左上、右上、右下、左下 */
  const rp = (x, y, w, h, tl, tr, br, bl) => { const k = Math.min(1, h / (tl + bl || 1), h / (tr + br || 1), w / (tl + tr || 1), w / (bl + br || 1)); tl *= k; tr *= k; br *= k; bl *= k; return `M${x + tl} ${y}H${x + w - tr}A${tr} ${tr} 0 0 1 ${x + w} ${y + tr}V${y + h - br}A${br} ${br} 0 0 1 ${x + w - br} ${y + h}H${x + bl}A${bl} ${bl} 0 0 1 ${x} ${y + h - bl}V${y + tl}A${tl} ${tl} 0 0 1 ${x + tl} ${y}Z`; };
  /** 在 [lo,hi] 里去掉 skips 之后剩下的区间 */
  const segs = (lo, hi, skips) => { const out = []; let a = lo; skips.sort((x, y) => x[0] - y[0]).forEach(([s, e]) => { if (s > a) out.push([a, s]); a = Math.max(a, e); }); if (hi > a) out.push([a, hi]); return out; };
  /** c:{n 带数, band 带宽, gap 带间缝, tail 线头, cut 压穿处的切口, ir 切口处圆角, cx 中心}。返回经线路径数组与逐行纬线路径数组 */
  const weaveParts = (c) => {
    const { n, band, gap, tail, cut, ir } = c, span = n * band + (n - 1) * gap, o = (c.cx ?? 512) - span / 2, lo = o - tail, hi = o + span + tail, R = band / 2, I = Math.min(ir, R), pos = (i) => o + i * (band + gap);
    const weft = [], warp = [];
    for (let j = 0; j < n; j++) { const sk = []; for (let i = 0; i < n; i++) if ((i + j) % 2 === 0) sk.push([pos(i) - cut, pos(i) + band + cut]);   /* 经线在上:纬线在这里断开 */
      weft.push(segs(lo, hi, sk).map(([a, b]) => rp(a, pos(j), b - a, band, a === lo ? R : I, b === hi ? R : I, b === hi ? R : I, a === lo ? R : I)).join('')); }
    for (let i = 0; i < n; i++) { const sk = []; for (let j = 0; j < n; j++) if ((i + j) % 2 === 1) sk.push([pos(j) - cut, pos(j) + band + cut]);   /* 纬线在上:经线在这里断开 */
      warp.push(segs(lo, hi, sk).map(([a, b]) => rp(pos(i), a, band, b - a, a === lo ? R : I, a === lo ? R : I, b === hi ? R : I, b === hi ? R : I)).join('')); }
    return { warp, weft };
  };
  const W2 = { n: 2, band: 170, gap: 34, tail: 106, cut: 20, ir: 16 };
  const BRACE = 'M438 322C378 322 372 352 372 398V446C372 482 352 502 312 512C352 522 372 542 372 578V626C372 672 378 702 438 702';
  const pill = (x, y, w, h) => `<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${h / 2}"/>`;

  /** 五种应用图标图形(1024 画布、图形区约 520)。ink:墨(经线 / 方块)、th:线(纬线,渐变引用) */
  const ART = {
    warp: ({ ink, th }) => { const { warp, weft } = weaveParts(W2); return `<path d="${warp.join('')}" fill="${ink}"/><path d="${weft.join('')}" fill="${th}"/>`; },
    prompt: ({ ink, th }) => `<path d="M372 368 520 512 372 656" fill="none" stroke="${th}" stroke-width="112" stroke-linecap="round" stroke-linejoin="round"/><rect x="566" y="600" width="150" height="112" rx="56" fill="${ink}"/>`,
    grid: ({ ink, th }) => { const c = 118, g = 30, o = 512 - (3 * c + 2 * g) / 2; let a = '', b = '';
      for (let i = 0; i < 3; i++) for (let j = 0; j < 3; j++) { const r = `<rect x="${o + i * (c + g)}" y="${o + j * (c + g)}" width="${c}" height="${c}" rx="34"/>`; if ((i + j) % 2 === 0) a += r; else b += r; }
      return `<g fill="${th}">${a}</g><g fill="${ink}">${b}</g>`; },
    lines: ({ ink, th }) => { const x0 = 300, h = 76, gap = 44, y0 = 512 - (4 * h + 3 * gap) / 2, rows = [[[0, 150, 1], [174, 250, 0]], [[96, 328, 0]], [[96, 190, 1], [214, 114, 0]], [[0, 230, 0], [254, 96, 1]]]; let a = '', b = '';
      rows.forEach((r, k) => r.forEach(([x, w, t]) => { const s = pill(x0 + x, y0 + k * (h + gap), w, h); if (t) a += s; else b += s; }));
      return `<g fill="${th}">${a}</g><g fill="${ink}">${b}</g>`; },
    braces: ({ ink, th }) => `<g fill="none" stroke="${ink}" stroke-width="96" stroke-linecap="round" stroke-linejoin="round"><path d="${BRACE}"/><path d="${BRACE}" transform="translate(1024 0) scale(-1 1)"/></g><circle cx="512" cy="512" r="56" fill="${th}"/>`,
  };

  /* ───────── 应用图标 ───────── */
  let uid = 0;
  /** 一枚应用图标的 SVG。o:{ s 样式, m 外观, t 主题, z 边长(px), flat 不要投影 } */
  const appSvg = (o) => {
    const r = resolve(o), id = 'bi' + (++uid), z = o.z || 64;
    const st = o.s && ART[o.s] ? o.s : cur('icoStyle');
    const small = z <= 36 || o.flat;
    return `<svg class="bi-svg" width="${z}" height="${z}" viewBox="0 0 1024 1024" aria-hidden="true"><defs>
      <linearGradient id="${id}b" x1="0" y1="100" x2="0" y2="924" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="${r.bg[0]}" stop-opacity="${r.bgo[0]}"/><stop offset="1" stop-color="${r.bg[1]}" stop-opacity="${r.bgo[1]}"/></linearGradient>
      <linearGradient id="${id}t" x1="250" y1="250" x2="774" y2="774" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="${r.th[0]}"/><stop offset="1" stop-color="${r.th[1]}"/></linearGradient>
      <linearGradient id="${id}r" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#fff" stop-opacity="${r.rim}"/><stop offset=".5" stop-color="#fff" stop-opacity="0"/></linearGradient>
      <radialGradient id="${id}g" cx="512" cy="100" r="640" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="#fff" stop-opacity="${r.gloss}"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></radialGradient></defs>
      <use href="#mc-sq" fill="url(#${id}b)" ${small ? '' : `filter="url(#mc-sh-${r.shade})"`}/>
      ${small ? '' : `<use href="#mc-sq" fill="url(#${id}g)"/><use href="#mc-sq" transform="translate(512 512) scale(.995) translate(-512 -512)" fill="none" stroke="url(#${id}r)" stroke-width="4"/>`}
      ${ART[st]({ ink: r.ink, th: `url(#${id}t)` })}</svg>`;
  };

  /* ───────── 菜单栏模板图标(36 画布,单色 currentColor) ───────── */
  const TRAY = {
    prompt: (lv) => `<path d="M10.5 9.5 18 18l-7.5 8.5" fill="none" stroke="currentColor" stroke-width="4.4" stroke-linecap="round" stroke-linejoin="round"/><rect ${lv(1)} x="20.5" y="23.6" width="8" height="4.4" rx="2.2"/>`,
    warp: (lv) => { const { warp, weft } = weaveParts(W2); return `<g transform="translate(18 18) scale(.04) translate(-512 -512)"><path d="${warp.join('')}"/>${weft.map((d, j) => `<path ${lv(j)} d="${d}" fill-opacity=".6"/>`).join('')}</g>`; },
    grid: (lv) => { const c = 7.4, g = 2.4, o = 18 - (3 * c + 2 * g) / 2; let s = '', k = 0;
      for (let i = 0; i < 3; i++) for (let j = 0; j < 3; j++) { const r = `x="${o + i * (c + g)}" y="${o + j * (c + g)}" width="${c}" height="${c}" rx="2.2"`; s += (i + j) % 2 === 0 ? `<rect ${r}/>` : `<rect ${lv(k++)} ${r} fill-opacity=".42"/>`; } return s; },
    lines: (lv) => { const sx = 27 / 424, x0 = 4.5, h = 4.4, gap = 3.6, y0 = 18 - (4 * h + 3 * gap) / 2, rows = [[[0, 150, 1], [174, 250, 0]], [[96, 328, 0]], [[96, 190, 1], [214, 114, 0]], [[0, 230, 0], [254, 96, 1]]]; let s = '', k = 0;
      rows.forEach((r, j) => r.forEach(([x, w, t]) => { s += `<rect ${t ? lv(k++) : ''} x="${(x0 + x * sx).toFixed(2)}" y="${(y0 + j * (h + gap)).toFixed(2)}" width="${(w * sx).toFixed(2)}" height="${h}" rx="${h / 2}"${t ? '' : ' fill-opacity=".55"'}/>`; })); return s; },
    braces: (lv) => `<g transform="translate(18 18) scale(.047) translate(-512 -512)"><g fill="none" stroke="currentColor" stroke-width="84" stroke-linecap="round" stroke-linejoin="round"><path d="${BRACE}"/><path d="${BRACE}" transform="translate(1024 0) scale(-1 1)"/></g><circle ${lv(1)} cx="512" cy="512" r="64"/></g>`,
  };
  /** 徽标:r 半径、c 彩色版的颜色(用 macOS 系统色,菜单栏是系统表面,不跟主题)、g 里面的符号(c 是符号色) */
  const BADGE = {
    waiting: { r: 6.4, c: '#ff9f0a', g: (x, y, c) => `<rect x="${x - .95}" y="${y - 3.5}" width="1.9" height="4.2" rx=".95" fill="${c}"/><circle cx="${x}" cy="${y + 2.6}" r="1.1" fill="${c}"/>` },
    done: { r: 4.6, c: '#30d158', g: null },
    error: { r: 6.4, c: '#ff453a', g: (x, y, c) => `<path d="M${x - 2.1} ${y - 2.1}l4.2 4.2M${x + 2.1} ${y - 2.1}l-4.2 4.2" stroke="${c}" stroke-width="1.7" stroke-linecap="round" fill="none"/>` },
    update: { r: 6.4, c: '#0a84ff', g: (x, y, c) => `<path d="M${x} ${y - 3}v5.2M${x - 2.4} ${y - .3}l2.4 2.6 2.4-2.6" stroke="${c}" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" fill="none"/>` },
    paused: { r: 6.4, c: '', g: (x, y, c) => `<rect x="${x - 2.3}" y="${y - 2.6}" width="1.7" height="5.2" rx=".6" fill="${c}"/><rect x="${x + .6}" y="${y - 2.6}" width="1.7" height="5.2" rx=".6" fill="${c}"/>` },
  };
  /** 一枚菜单栏图标。o:{ s 样式, state 状态, z 边长, col 彩色徽标, frame 动画定格帧 } */
  const traySvg = (o) => {
    const id = 'bt' + (++uid), st = o.s && TRAY[o.s] ? o.s : cur('icoStyle'), state = o.state || 'idle', z = o.z || 18, col = +(o.col ?? cur('trayColor')), fr = o.frame;
    const lv = (i) => `class="lv" style="--d:${i}${fr !== undefined ? `;opacity:${i === fr % 3 ? 1 : .26}` : ''}"`;
    let body = TRAY[st](lv), badge = '';
    const bd = BADGE[state], x = 28.2, y = 8;
    if (bd || state === 'demo') {
      const r = bd ? bd.r : 4.2;
      body = `<mask id="${id}k" maskUnits="userSpaceOnUse" x="0" y="0" width="36" height="36"><rect width="36" height="36" fill="#fff"/><circle cx="${x}" cy="${y}" r="${r + 2.3}" fill="#000"/></mask><g mask="url(#${id}k)">${body}</g>`;
      if (state === 'demo') badge = `<circle cx="${x}" cy="${y}" r="3.7" fill="none" stroke="currentColor" stroke-width="1.9"/>`;
      else if (col && bd.c) badge = `<circle cx="${x}" cy="${y}" r="${r}" fill="${bd.c}"/>${bd.g ? bd.g(x, y, '#fff') : ''}`;
      else badge = bd.g ? `<mask id="${id}b" maskUnits="userSpaceOnUse" x="0" y="0" width="36" height="36"><rect width="36" height="36" fill="#fff"/>${bd.g(x, y, '#000')}</mask><circle cx="${x}" cy="${y}" r="${r}" fill="currentColor" mask="url(#${id}b)"/>` : `<circle cx="${x}" cy="${y}" r="${r}" fill="currentColor"/>`;
    }
    const anim = state === 'running' && fr === undefined ? (o.anim || cur('trayAnim')) : 'none';
    return `<svg class="bi-tray ${anim === 'none' ? '' : 'tr-' + anim}" width="${z}" height="${z}" viewBox="0 0 36 36" fill="currentColor" aria-hidden="true"><g ${state === 'paused' ? 'opacity=".5"' : ''}>${body}</g>${badge}</svg>`;
  };

  /* ───────── 对外:带数据属性的包装(live 刷新靠它)与刷新 ───────── */
  const attrs = (o) => ['s', 'm', 't', 'z', 'state', 'col', 'flat', 'frame', 'anim'].filter(k => o[k] !== undefined).map(k => `data-${k}="${o[k]}"`).join(' ');
  const app = (o = {}) => `<span class="bi" data-bi="app" ${attrs(o)}>${appSvg(o)}</span>`;
  const tray = (o = {}) => `<span class="bi" data-bi="tray" ${attrs(o)}>${traySvg(o)}</span>`;
  /** 重新生成页面里所有 [data-bi]:读它们的数据属性,没写的字段取当前设置 */
  const refresh = (root = document) => {
    root.querySelectorAll('[data-bi]').forEach(el => { const o = {}; for (const k in el.dataset) { if (k !== 'bi') o[k] = ['z', 'col', 'flat', 'frame'].includes(k) ? +el.dataset[k] : el.dataset[k]; }
      el.innerHTML = el.dataset.bi === 'tray' ? traySvg(o) : appSvg(o); });
    root.querySelectorAll('[data-bi-name]').forEach(el => { el.textContent = styleName(el.dataset.biName); });
  };
  const styleName = () => { const k = cur('icoStyle'); return (STYLES.find(s => s.id === k) || STYLES[0]).name; };
  /** 查看器里各处「小标」:边长为 size 的图形(图标本体占 80%,所以画布放大 1.24 倍) */
  const logo = (size, extra = '') => `<span class="logo brand" style="width:${size}px;height:${size}px;${extra}">${app({ z: Math.round(size * 1.24) })}</span>`;

  /* ───────── 设置里用的构件 ───────── */
  /** 图标样式选择器:五张卡,每张同时给出应用图标与它在浅 / 深菜单栏上的样子(同一个样式,一次换两处);悬停或选中时菜单栏图标播放「运行中」动画 */
  const pickApp = () => `<div class="ico-pick" role="radiogroup" aria-label="图标样式(应用图标与菜单栏图标同步)">${STYLES.map(s => `<div class="ico-card ${s.id === cur('icoStyle') ? 'on' : ''}" data-act="ico-style" data-v="${s.id}" tabindex="0" role="radio" aria-checked="${s.id === cur('icoStyle')}" data-tip="${s.d}">${app({ s: s.id, z: 72 })}<div class="pr tr-hov"><span class="mb-chip l">${tray({ s: s.id, state: 'running', z: 18, anim: 'flow' })}</span><span class="mb-chip d">${tray({ s: s.id, state: 'running', z: 18, anim: 'flow' })}</span></div><b>${s.name}</b><span>${s.en}</span></div>`).join('')}</div>`;
  /** 配色:跟随主题 + 六套主题各一枚小图标 */
  const tintPick = () => `<span class="ico-tints" role="radiogroup" aria-label="图标配色"><span class="ico-sw fol ${cur('icoTint') === 'follow' ? 'on' : ''}" data-act="ico-tint" data-v="follow" tabindex="0" data-tip="跟随当前主题"><i class="ring"></i></span>${THEMES.map(t => `<span class="ico-sw ${cur('icoTint') === t.id ? 'on' : ''}" data-act="ico-tint" data-v="${t.id}" tabindex="0" data-tip="${t.name} · ${t.sub}">${app({ t: t.id, m: 'auto', z: 34, flat: 1 })}</span>`).join('')}</span>`;
  /** 浅 / 深两张壁纸上的 Dock 预览 + 各尺寸并排,设置里改选项时实时变 */
  const previewDocks = () => `<div class="ico-prev">${['l', 'd'].map(k => `<div class="wall wall-${k}"><div class="mk-dock ${k === 'd' ? 'd' : ''}"><span class="dk sm"><i class="gen-ic" style="--h:${k === 'l' ? 205 : 255}"></i></span><span class="dk run">${app({ z: 56 })}</span><span class="dk sm"><i class="gen-ic" style="--h:${k === 'l' ? 340 : 160}"></i></span></div><div class="sz">${app({ z: 40 })}${app({ z: 24 })}${app({ z: 16 })}</div></div>`).join('')}</div>`;
  /** 动态预览:Dock + 浅 / 深菜单栏三处用同一个样式播放某个托盘状态;遵守「Dock 角标 / 弹跳 / 运行中动画 / 彩色徽标 / 显示数字」设置。state = TRAY_STATES 的 id */
  const liveInner = (id) => {
    const bd = cur('dockBadge'), n = bd === 'pending' && id === 'waiting' ? 2 : bd === 'running' && (id === 'running' || id === 'waiting') ? 1 : 0;
    const bounce = id === 'waiting' && cur('dockBounce') !== 'off', cnt = +cur('trayCount') ? (id === 'waiting' ? 2 : id === 'running' ? 1 : '') : '';
    const bar = (k) => `<div class="blv-bar ${k}"><i></i><i></i>${tray({ state: id, z: 20 })}${cnt ? `<b>${cnt}</b>` : ''}</div>`;
    const pins = `<div class="blv-pins"><span class="chip ${S().brandPin ? '' : 'on'}" data-act="brand-pin" data-v="">自动循环</span>${TRAY_STATES.map(([k, n2]) => `<span class="chip ${S().brandPin === k ? 'on' : ''}" data-act="brand-pin" data-v="${k}">${n2}</span>`).join('')}</div>`;
    const d = TRAY_STATES.find(x => x[0] === id) || TRAY_STATES[0];
    return `<div class="blv-row"><div><div class="wall wall-l blv-wall"><div class="mk-dock"><span class="dk run ${bounce ? 'bounce' : ''}">${app({ z: 56 })}${n ? `<b class="dk-badge">${n}</b>` : ''}</span></div></div><span class="t4 fs-micro">Dock</span></div><div style="flex:1;min-width:160px"><div class="blv-bar l"><i></i><i></i>${tray({ state: id, z: 20 })}${cnt ? `<b>${cnt}</b>` : ''}</div><span class="t4 fs-micro">浅色菜单栏</span></div><div style="flex:1;min-width:160px">${bar('d')}<span class="t4 fs-micro">深色菜单栏</span></div></div>${pins}<div class="t3 fs-cap">${d[1]}:${d[2]}</div>`;
  };
  const live = () => `<div class="blv" data-blv>${liveInner(S().brandPin || 'idle')}</div>`;
  /** 锁定版式:tile = 图标本体的边长。图标本体撑满布局盒,间距 = tile / 8,字号 = tile × .58,外圈虚线 = 安全留白(tile / 4) */
  const lockup = (tile, vertical = false) => `<span class="lockup ${vertical ? 'v' : ''}" style="--tile:${tile}px">${app({ z: Math.round(tile / .8047), flat: tile < 40 ? 1 : undefined })}<span class="wm">MiniCode</span></span>`;
  /** 分段:取值写在 OPT 里,data-ikey 告诉交互层改的是哪个偏好 */
  const segOpt = (key, labels) => seg(labels, Math.max(0, OPT[key].indexOf(cur(key))), '', 'ico-opt').replace('<div class="seg ', `<div data-ikey="${key}" class="seg `);
  const togOpt = (key) => tog(+cur(key) ? 1 : 0, `data-live="ico" data-ikey="${key}"`);

  window.U.brand = { app, tray, appSvg, traySvg, refresh, logo, STYLES, TRAYS, TRAY_STATES, PAL, OPT, DEF, cur, resolve, themeOf, styleName, pickApp, live, liveInner, lockup, tintPick, previewDocks, segOpt, togOpt, ART, weaveParts, SQ, mix, uidNext: () => 'bu' + (++uid) };
})();
