/* 设计稿 v3 · 基础件模板与注册表(主题 / 方案 / 工具),全部是返回 HTML 字符串的小函数
   @author ixxxxoooo */
(function () {
  Object.assign(ICONS, {
    chat: '<path d="M5 5.5h14A1.5 1.5 0 0 1 20.5 7v8.5A1.5 1.5 0 0 1 19 17h-6.5L8 20.5V17H5a1.5 1.5 0 0 1-1.5-1.5V7A1.5 1.5 0 0 1 5 5.5z"/>',
    thumbUp: '<path d="M8 10.5 11.5 4a2 2 0 0 1 2 2.4L13 10h5a2 2 0 0 1 2 2.4l-1 5A2 2 0 0 1 17 19H8zM4 10.5h4V19H4z"/>',
  });

  /** 图标。n:图标名;c:附加类(s12 / s16 / s20、颜色类) */
  const ic = (n, c = '') => `<svg class="ic ${c}" viewBox="0 0 24 24">${ICONS[n] || ''}</svg>`;
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
  const tog = (on) => `<span class="tog ${on ? 'on' : ''}" data-act="tog"></span>`;
  /** 分段控件。items:各段 HTML;on:选中下标;act:点击后派发的动作名 */
  const seg = (items, on, cls = '', act = 'seg') => `<div class="seg ${cls}" style="--n:${items.length};--i:${on}"><i class="thumb"></i>${items.map((t, k) => `<button class="${k === on ? 'on' : ''}" data-act="${act}" data-i="${k}">${t}</button>`).join('')}</div>`;
  /** 勾选圆。state:'' 未开始 / now 进行中 / done 已完成 */
  const chk = (state = '') => state === 'now' ? '<span class="chk now"><span class="spin"></span></span>' : `<span class="chk ${state}"><svg viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg></span>`;
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

  window.U = { ic, btn, ib, chip, tag, cnt, kbd, tog, seg, chk, esc, rust, THEMES, SCHEMES, TOOLS };
})();
