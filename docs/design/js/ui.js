/* 设计稿 v3 · 基础件模板与注册表(主题 / 方案 / 工具),全部是返回 HTML 字符串的小函数
   @author ixxxxoooo */
(function () {
  Object.assign(ICONS, {
    chat: '<path d="M5 5.5h14A1.5 1.5 0 0 1 20.5 7v8.5A1.5 1.5 0 0 1 19 17h-6.5L8 20.5V17H5a1.5 1.5 0 0 1-1.5-1.5V7A1.5 1.5 0 0 1 5 5.5z"/>',
    thumbUp: '<path d="M8 10.5 11.5 4a2 2 0 0 1 2 2.4L13 10h5a2 2 0 0 1 2 2.4l-1 5A2 2 0 0 1 17 19H8zM4 10.5h4V19H4z"/>',
    chat: '<path d="M5 5h14a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1h-8l-4.5 3.5V16H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1z"/>',
    upload: '<path d="M12 16V4M7.5 8.5 12 4l4.5 4.5M5 16v3a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-3"/>',
    user: '<circle cx="12" cy="8.5" r="3.5"/><path d="M5 19.5c.8-3.5 3.6-5.5 7-5.5s6.2 2 7 5.5"/>',
    pause: '<path d="M8 5v14M16 5v14"/>',
    power: '<path d="M12 4v8M7 7.2a7 7 0 1 0 10 0"/>',
    eyeOff: '<path d="M4 4l16 16M9.9 5.2A9 9 0 0 1 12 5c4.5 0 8 4 9 7a12 12 0 0 1-2.6 3.6M6.3 6.8A12 12 0 0 0 3 12c1 3 4.5 7 9 7 1.4 0 2.7-.4 3.8-1M10 10a3 3 0 0 0 4 4"/>',
    gift: '<path d="M4 9h16v4H4zM6 13v7h12v-7M12 9v11M12 9c-2 0-4-1-4-2.5S9.5 4 12 9zM12 9c2 0 4-1 4-2.5S14.5 4 12 9z"/>',
  });

  /** 图标。n:图标名;c:附加类(s12 / s16 / s20、颜色类) */
  const ic = (n, c = '') => `<svg class="ic ${c}" viewBox="0 0 24 24">${ICONS[n] || ''}</svg>`;
  /**
   * 文件类型图标(对标 Cursor / VS Code 图标主题):已知语言用「语言缩写 + 主题色」的小圆角徽标,
   * 其余退回线条图标。颜色取主题令牌,换主题自动跟着变;设置 → 外观「彩色文件图标」关闭时一律灰色线条图标。
   * Go 实现:mygo/internal/app/file_icon.go,两边的扩展名表保持一致。
   */
  const FILE_KINDS = {
    js: ['JS', 'warn'], mjs: ['JS', 'warn'], cjs: ['JS', 'warn'], jsx: ['JSX', 'warn'],
    ts: ['TS', 'info'], tsx: ['TSX', 'info'], go: ['GO', 'info'], rs: ['RS', 'warn'], py: ['PY', 'ok'],
    swift: ['SW', 'err'], java: ['JV', 'err'], kt: ['KT', 'vio'], c: ['C', 'info'], h: ['H', 'info'], cpp: ['C++', 'info'], hpp: ['H++', 'info'],
    rb: ['RB', 'err'], php: ['PHP', 'vio'], lua: ['LU', 'info'], sh: ['SH', 'ok'], zsh: ['SH', 'ok'], bash: ['SH', 'ok'],
    html: ['HT', 'err'], css: ['CS', 'info'], scss: ['SC', 'vio'], vue: ['VU', 'ok'],
    md: ['MD', 'tx-2'], mdx: ['MD', 'tx-2'], json: ['{}', 'warn'], toml: ['TM', 'tx-2'], yaml: ['YM', 'err'], yml: ['YM', 'err'],
    sql: ['SQ', 'vio'], csv: ['CS', 'ok'], xml: ['XM', 'warn'],
  };
  const FILE_IMG = ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'ico'];
  const fi = (name, c = '') => {
    const ext = String(name).toLowerCase().split('.').pop(), k = String(name).includes('.') ? FILE_KINDS[ext] : null;
    if (k) return `<span class="fi ${c}" style="--c:var(--${k[1]})">${k[0]}</span>`;
    if (FILE_IMG.includes(ext)) return `<span class="fi-img ${c}" style="color:var(--c-vio)">${ic('image', 's12')}</span>`;
    return ic('fileText', `s12 t3 ${c}`);
  };
  /** 按钮。t:文字;c:pri / soft / ghost / sm / danger;icon:前置图标;attr:附加属性 */
  const btn = (t, c = '', icon = '', attr = '') => `<button class="btn ${c}" ${attr}>${icon ? ic(icon, 's12') : ''}${t}</button>`;
  /** 图标按钮。tip:悬停提示 */
  const ib = (n, c = '', tip = '', attr = '') => `<button class="ib ${c}" ${tip ? `data-tip="${tip}"` : ''} ${attr}>${ic(n)}</button>`;
  /** 胶囊。c:line / on */
  const chip = (t, icon = '', c = '', attr = '') => `<span class="chip ${c}" ${attr}>${icon ? ic(icon, 's12') : ''}${t}</span>`;
  /** 徽标。c:ac / ok / err / warn / info */
  const tag = (t, c = '') => `<span class="tag ${c}">${t}</span>`;
  const cnt = (n, c = '') => `<span class="cnt ${c}">${n}</span>`;
  const kbd = (...k) => `<span class="kbds">${k.map(x => `<kbd>${x}</kbd>`).join('')}</span>`;
  const tog = (on, extra = '') => `<span class="tog ${on ? 'on' : ''}" data-act="tog" role="switch" tabindex="0" aria-checked="${on ? 'true' : 'false'}" ${extra}></span>`;
  /** 真实输入框:带图标、占位符;data-in 用来告诉交互层「这是哪个输入」 */
  const inp = (ph, o = {}) => `<label class="inp ${o.cls || ''}" ${o.style ? `style="${o.style}"` : ''}>${o.icon ? ic(o.icon, 's12') : ''}<input ${o.id ? `id="${o.id}"` : ''} data-in="${o.id || ''}" placeholder="${ph}" value="${o.v || ''}" spellcheck="false" ${o.attr || ''}></label>`;
  /** 可编辑的行内字段(设置里的路径、地址、数字等) */
  const field = (v, w = 180, o = {}) => `<input class="fld ${o.mono ? 'mono' : ''}" ${w ? `style="width:${w}px"` : ''} value="${esc(String(v))}" ${o.id ? `id="${o.id}"` : ''} data-in="${o.id || ''}" ${o.ph ? `placeholder="${o.ph}"` : ''} spellcheck="false" ${o.attr || ''}>`;
  /** 多行输入 */
  const area = (v, o = {}) => `<textarea class="ta ${o.mono ? 'mono' : ''}" ${o.id ? `id="${o.id}"` : ''} data-in="${o.id || ''}" rows="${o.rows || 4}" spellcheck="false" ${o.ph ? `placeholder="${o.ph}"` : ''}>${esc(String(v))}</textarea>`;
  /** 分段控件。items:各段 HTML;on:选中下标;act:点击后派发的动作名 */
  const seg = (items, on, cls = '', act = 'seg') => `<div class="seg ${cls}" style="--n:${items.length};--i:${on}"><i class="thumb"></i>${items.map((t, k) => `<button class="${k === on ? 'on' : ''}" data-act="${act}" data-i="${k}">${t}</button>`).join('')}</div>`;
  /** 勾选圆。state:'' 未开始 / now 进行中 / done 已完成 */
  const chk = (state = '') => state === 'now' ? '<span class="chk now"><span class="spin"></span></span>' : `<span class="chk ${state}"><svg viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg></span>`;
  /** 下拉触发器。items: {label,value,disabled?,hint?}[];onIdx:选中下标;o.mono / o.w / o.attr */
  const sel = (items, onIdx, o = {}) => {
    const k = Math.max(0, Math.min(onIdx || 0, items.length - 1));
    const it = items[k] || { label: '—', value: '' };
    const dis = (it.disabled || o.disabled) ? ' is-disabled' : '';
    const tip = it.hint ? ` data-tip="${it.hint}"` : '';
    const w = o.w ? ` style="width:${o.w}px"` : '';
    const mono = o.mono ? ' mono' : '';
    return `<button type="button" class="sel${dis}" data-act="sel" data-idx="${k}"${w}${tip} ${o.attr || ''}>${it.logo && window.LOGOS ? LOGOS.logo(it.brand, 14) : ''}<span class="grow ell${mono}">${esc(String(it.label))}</span>${ic('chevD', 's12 car')}</button>`;
  };
  const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

  const RX = /(\/\/.*$)|("(?:[^"\\]|\\.)*"|'.')|\b(use|pub|struct|impl|fn|let|mut|self|Self|return|if|else|match|for|in|assert_eq|mod)\b|\b([A-Z][A-Za-z0-9]*)\b|\b(\d+)\b|\b([a-z_]\w*)(?=\()/gm;
  /** Rust 语法着色(演示用的极简分词)。src:源码;返回带 span 的 HTML */
  const rust = (src) => esc(src).replace(RX, (m, cm, s, k, ty, nu, fn) => `<span class="${cm ? 'cm' : s ? 's' : k ? 'k' : ty ? 'ty' : nu ? 'nu' : 'fn'}">${m}</span>`);

  /* 主题注册表 */
  const THEMES = [
    { id: 'paper', name: '宣纸', sub: '暖白 · 橙', dark: false },
    { id: 'mist', name: '雾霭', sub: '冷灰 · 靛蓝', dark: false },
    { id: 'celadon', name: '青瓷', sub: '灰绿 · 松石', dark: false },
    { id: 'ink', name: '墨夜', sub: '暖黑 · 橙', dark: true },
    { id: 'graphite', name: '石墨', sub: '中性黑 · 钴蓝', dark: true },
    { id: 'aurora', name: '极光', sub: '深蓝 · 堇紫', dark: true },
  ];

  /* 方案注册表 */
  const SCHEMES = [
    { id: 'a', name: '墨线', en: 'Ink', tagline: '同一平面,细线分隔', fit: '最克制、最像原生 macOS 工具;信息密度高而不乱。',
      points: ['标题栏正中的 Agent / IDE 分段开关是唯一的模式入口', '面板之间只有 1px 细线,圆角收到 4–8px', '会话按项目分组,选中行左侧一道强调色竖线', '步骤行是纯文本行,悬停才出现底色'] },
    { id: 'b', name: '浮岛', en: 'Float', tagline: '面板浮在底色上,圆润柔和', fit: '设计感最强、最有呼吸感;紧凑度略低于另外两套(行高 26)。',
      points: ['窗口底色带一抹强调色光晕,面板是带阴影的圆角岛', '会话只看当前项目,按「今天 / 昨天 / 更早」排', '用户消息是右对齐气泡,连续步骤收进一张活动卡', '输入框是悬浮的大圆角卡,命令栏居中成胶囊'] },
    { id: 'c', name: '轨道', en: 'Rail', tagline: '图标轨常驻,步骤串成时间轴', fit: '最紧凑(行高 22)、切换最快;Agent 与 IDE 的界限最弱,几乎是一个视图。',
      points: ['左侧图标轨在两种模式下都不动,导航与页面入口都在轨上', '导航栏只放内容,比另外两套窄 16px', '会话流的步骤串在一条竖线上,像提交图谱', '编辑器与右栏标签都用下划线,用户消息是带色条的引文'] },
  ];

  /* 工具注册表:会话流里每种工具的图标、动词与所属工具族(色相令牌 --tool-<族>) */
  const TOOLS = {
    think: { icon: 'brain', verb: '思考', fam: 'think', cn: '思考' },
    read: { icon: 'fileText', verb: '读取', fam: 'explore', cn: 'Read' },
    grep: { icon: 'search', verb: '搜索', fam: 'explore', cn: 'Grep' },
    glob: { icon: 'folder', verb: '查找文件', fam: 'explore', cn: 'Glob' },
    group: { icon: 'layers', verb: '探索', fam: 'explore', cn: '工具分组' },
    edit: { icon: 'edit', verb: '编辑', fam: 'edit', cn: 'Edit' },
    write: { icon: 'file', verb: '写入', fam: 'edit', cn: 'Write' },
    bash: { icon: 'terminal', verb: '运行', fam: 'exec', cn: 'Bash' },
    bg: { icon: 'activity', verb: '后台运行', fam: 'exec', cn: '后台 Bash' },
    monitor: { icon: 'eye', verb: '监听', fam: 'exec', cn: 'Monitor' },
    fetch: { icon: 'globe', verb: '抓取', fam: 'web', cn: 'WebFetch' },
    search: { icon: 'globe', verb: '联网搜索', fam: 'web', cn: 'WebSearch' },
    mcp: { icon: 'plug', verb: '调用', fam: 'mcp', cn: 'MCP' },
    agent: { icon: 'bot', verb: '子 Agent', fam: 'agent', cn: 'Agent' },
    todo: { icon: 'todo', verb: '待办', fam: 'plan', cn: 'TodoWrite' },
    plan: { icon: 'list', verb: '计划', fam: 'plan', cn: 'ExitPlanMode' },
  };

  window.U = { ic, fi, btn, ib, chip, tag, cnt, kbd, tog, seg, chk, sel, esc, rust, inp, field, area, THEMES, SCHEMES, TOOLS };
})();
