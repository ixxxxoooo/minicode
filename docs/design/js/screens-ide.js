/* IDE 模式设计稿:主界面三种布局、欢迎页、源代码管理、搜索与问题 */
(function () {
  const { i, chip, bdg, cnt, kbd, tg, seg, dot, ib, btn, logo, meter, hl } = K;
  const { C } = window;
  S.group('ide', 'IDE 模式');

  /* ───────── 公共:标题栏、活动栏、文件树、编辑器、状态栏 ───────── */
  const titleBar = (o = {}) => `<div class="tb bd" style="background:var(--bg-chrome)">${K.tl()}${ib('panelL', 'sm')}${ib('chevL', 'sm')}${ib('chevR', 'sm')}
    <div class="grow f jc" style="min-width:0"><div class="inp fill grow" style="max-width:420px;justify-content:center;height:24px">${i('search', 's12')}<span>router-demo</span><span class="tx3 s10">${o.branch || ''}</span></div></div>
    ${ib('panelB', 'sm')}${ib('panelR', 'sm on')}${seg(['Agent', 'IDE'], 1)}</div>`;
  const actBar = (on = 0) => `<div class="act">${['folder', 'search', 'branch'].map((n, k) => `<div class="a ${k === on ? 'on' : ''}">${i(n, 's16')}${n === 'branch' ? cnt(3) : ''}</div>`).join('')}<div class="a">${i('bug', 's16')}</div><div class="a">${i('plug', 's16')}</div><span class="grow"></span><div class="a">${i('gear', 's16')}</div></div>`;
  const tr = (d, ic, n, ex, g, o = '') => `<div class="tr ${o}" style="padding-left:${6 + d * 13}px">${ex ? i(ex === 1 ? 'chevD' : 'chevR', 's12 tx3') : '<span style="width:12px;flex:none"></span>'}${i(ic, 's12 ' + (ic === 'folder' ? 'acc' : 'tx2'))}<span class="grow ell">${n}</span>${g ? `<span class="g ${g === 'M' ? 'warnc' : g === 'A' || g === 'U' ? 'okc' : 'tx3'}">${g}</span>` : ''}</div>`;
  const tree = () => `<div style="padding-bottom:4px">${tr(0, 'folder', 'examples', 2)}${tr(0, 'folder', 'src', 1)}${tr(1, 'file', 'case.rs', 0, 'A')}${tr(1, 'file', 'main.rs')}${tr(1, 'file', 'middleware.rs')}${tr(1, 'file', 'router.rs', 0, 'M', 'sel')}${tr(1, 'file', 'routes.rs')}${tr(1, 'file', 'utils.rs')}${tr(0, 'folder', 'tests', 2)}${tr(0, 'file', '.gitignore')}${tr(0, 'file', 'Cargo.toml')}${tr(0, 'file', 'CHANGELOG.md', 0, 'M')}${tr(0, 'file', 'README.md')}</div>`;
  const exp = (o = {}) => `<div class="exp" style="width:${o.w || 228}px">
    <div class="sec-h" style="margin-top:4px"><span class="grow">ROUTER-DEMO</span>${ib('plus', 'sm')}${ib('refresh', 'sm')}${ib('more', 'sm')}</div>
    <div class="grow" style="overflow:hidden">${tree()}</div>
    <div style="border-top:1px solid var(--bd)"><div class="sec-h">${i('chevR', 's12')}大纲<span class="grow"></span><span class="tx3" style="font-weight:400;letter-spacing:0">router.rs · 4 个符号</span></div>
      <div class="sec-h" style="border-top:1px solid var(--bd-soft)">${i('chevR', 's12')}时间线</div></div>
  </div>`;

  const RS = [
    '{k use} std::collections::{ty HashMap};', '', '{c /// 路由表:路径 → 处理函数}', '{k pub struct} {ty Router} {', '    routes: {ty HashMap}<{ty String}, {ty Route}>,', '    fallback: {ty Option}<{ty Route}>,', '}', '',
    '{k impl} {ty Router} {', '    {k pub fn} {fn new}() -> {ty Self} {', '        {ty Self} { routes: {ty HashMap}::new(), fallback: {ty None} }', '    }', '',
    '    {c /// 按路径查找路由,大小写不敏感}', '    {k pub fn} {fn match_path}(&{k self}, p: &{ty str}) -> {ty Option}<&{ty Route}> {', '        {k let} key = p.to_ascii_lowercase();', '        {k let} key = key.trim_end_matches(\'/\');', '        {k self}.routes.get(key).or({k self}.fallback.as_ref())', '    }', '',
    '    {k pub fn} {fn add}(&{k mut self}, path: &{ty str}, route: {ty Route}) {', '        {k self}.routes.insert(path.to_ascii_lowercase(), route);', '    }', '}',
  ];
  const gitMark = { 16: 'var(--warn)', 17: 'var(--ok)', 22: 'var(--warn)' };
  const editorBody = (o = {}) => {
    const n = RS.length + (o.pad || 0);
    const lines = RS.map((l, k) => {
      let h = hl(l);
      if (k === 15) h = h.replace('key', '<span class="sqw" title="变量被遮蔽">key</span>');
      const cur = k === 16;
      return `<div class="${cur ? 'cur' : ''}">${h}${cur ? '<span class="blame">李文娇 · 3 天前 · fix: 路径末尾斜杠</span>' : ''}</div>`;
    }).join('');
    const gut = Array.from({ length: n }, (_, k) => `<div style="${k === 16 ? 'color:var(--tx)' : ''};position:relative">${k + 1}${gitMark[k] ? `<i style="position:absolute;right:2px;top:1px;bottom:1px;width:2px;border-radius:1px;background:${gitMark[k]}"></i>` : ''}</div>`).join('');
    const mm = Array.from({ length: 34 }, (_, k) => `<i style="width:${[30, 55, 20, 70, 60, 40, 10, 0, 45, 62, 40, 15, 0, 50, 75, 60, 66, 68, 12, 0, 70, 66, 12, 8][k % 24] || 30}%;${k === 16 ? 'background:var(--ac);opacity:.9' : ''}"></i>`).join('');
    return `<div class="f grow" style="min-height:0;overflow:hidden">
      <div class="code grow" style="padding-top:6px;min-width:0"><div class="gut">${gut}</div><div class="src">${lines}</div></div>
      ${o.minimap === false ? '' : `<div class="mm"><div class="abs" style="left:0;right:0;top:4px;height:60px;background:var(--bg-hover);border-top:1px solid var(--bd);border-bottom:1px solid var(--bd)"></div>${mm}</div>`}
    </div>`;
  };
  const etabs = () => `<div class="etabs">
    <div class="etab on">${i('file', 's12 acc')}router.rs<span class="mod"></span></div>
    <div class="etab">${i('file', 's12 tx3')}case.rs<span class="x">${i('x', 's12')}</span></div>
    <div class="etab pv"><span class="nm">${i('file', 's12 tx3')} Cargo.toml</span><span class="x">${i('x', 's12')}</span></div>
    <span class="grow"></span><div class="f ai g2" style="padding-right:6px">${ib('cols', 'sm')}${ib('more', 'sm')}</div></div>`;
  const crumbs = () => `<div class="crumbs">router-demo ${i('chevR', 's12')} src ${i('chevR', 's12')} <b>router.rs</b> ${i('chevR', 's12')} <span class="f ai g4">${i('cube', 's12 vioc')}<b>Router</b></span> ${i('chevR', 's12')} <span class="f ai g4">${i('cube', 's12 infoc')}<b>match_path</b></span><span class="grow"></span><span class="f ai g4">${ib('branch', 'sm')}${ib('eye', 'sm')}</span></div>`;
  const editor = (o = {}) => `<div class="grow fc" style="min-width:0;min-height:0">${etabs()}${crumbs()}${editorBody(o)}${o.panel || ''}</div>`;
  const termPanel = (h = 150) => `<div class="fc" style="height:${h}px;border-top:1px solid var(--bd);flex:none"><div class="panel-h">${['终端', '问题 1', '输出', '任务 2'].map((t, k) => `<span class="pt ${k === 0 ? 'on' : ''}">${t}</span>`).join('')}<span class="grow"></span>${ib('plus', 'sm')}${ib('max', 'sm')}${ib('x', 'sm')}</div>
    <div class="term grow" style="overflow:hidden"><span class="p">➜</span> router-demo <span class="d">git:(main)</span> cargo test<br>test case::upper_lower ... <span class="p">ok</span><br>test case::trailing_slash ... <span class="p">ok</span><br>test result: <span class="p">ok</span>. 12 passed; 0 failed<br><span class="p">➜</span> router-demo <span class="d">git:(main)</span> <span class="pulse">▌</span></div></div>`;
  const status = (o = {}) => `<div class="stat"><span class="bdg ac">演示模式</span><span class="it">${i('branch', 's12')}main${i('refresh', 's12 tx3')}</span><span class="it errc">${i('errc', 's12')}0</span><span class="it warnc">${i('alert', 's12')}1</span><span class="it">${i('cube', 's12')}rust-analyzer 已就绪</span><span class="grow"></span><span class="it">行 17,列 24</span><span class="it">空格:4</span><span class="it">UTF-8</span><span class="it">Rust</span><span class="it">权限:接受编辑</span><span class="it">${i('bell', 's12')}</span></div>`;

  // 右侧对话栏
  const ideChat = (w = 340, o = {}) => `<div class="fc" style="width:${w}px;border-left:1px solid var(--bd);background:var(--bg-main);flex:none;min-height:0">
    <div class="tabs" style="height:32px;padding:0 4px;gap:0"><div class="tab on" style="max-width:190px"><span class="spin"></span><span class="ell">完整修复一个 Bug…</span></div><div class="tab" style="max-width:100px"><span class="dot ring"></span><span class="ell">权限请求</span></div><span class="grow"></span>${ib('plus', 'sm')}${ib('history', 'sm')}</div>
    <div class="grow" style="overflow:hidden;padding:8px 10px" class="fc g6">
      ${C.user('路由匹配对大小写敏感,/Users 打不开 /users,帮我修复并加测试')}
      <div class="fc g2" style="margin-top:6px">${C.think()}${C.step('layers', '读取', '3 个文件', '', '')}${C.step('edit', '编辑', 'router.rs', '', '<span class="okc mono">+4</span><span class="errc mono">−1</span>')}${C.step('prompt', '运行', 'cargo test', '12 passed', '<span class="okc">' + i('check', 's12') + '</span>')}</div>
      <div style="margin-top:6px" class="md"><p>已修复:匹配前统一转小写并去掉末尾 <code>/</code>,新增 5 个测试,全部通过。</p></div>
      <div class="foot-stat" style="margin-top:2px">${ib('copy', 'sm')}${ib('rewind', 'sm')}<span>Sonnet 5.5 · 1分13秒 · $0.02</span></div>
    </div>
    <div style="padding:0 10px 8px">${o.cmp || C.cmpB({ narrow: true })}</div>
  </div>`;

  /* ───────── IDE 主界面 ───────── */
  const ideA = () => `<div class="fc" style="height:100%">${titleBar({ branch: 'main' })}
    <div class="f grow" style="min-height:0">${actBar(0)}${exp()}${editor({ panel: termPanel() })}${ideChat()}</div>${status()}</div>`;

  const ideB = () => `<div class="fc" style="height:100%">${titleBar({ branch: 'main' })}
    <div class="f grow" style="min-height:0">
      <div class="fc" style="width:372px;border-right:1px solid var(--bd);flex:none;background:var(--bg-main)">
        <div class="tabs" style="height:32px;padding:0 4px;gap:0"><div class="tab on"><span class="spin"></span>完整修复一个 Bug…</div><div class="tab"><span class="dot ring"></span>权限请求</div><span class="grow"></span>${ib('plus', 'sm')}${ib('history', 'sm')}</div>
        <div class="grow" style="overflow:hidden;padding:10px 12px">${C.user('路由匹配对大小写敏感,/Users 打不开 /users,帮我修复并加测试')}
          <div class="fc g2" style="margin-top:8px">${C.think()}${C.step('layers', '读取', '3 个文件', '', '')}${C.step('edit', '编辑', 'router.rs', '', '<span class="okc mono">+4</span><span class="errc mono">−1</span>')}
            <div class="nest" style="margin-left:7px">${C.diffCard()}</div>${C.step('prompt', '运行', 'cargo test', '12 passed', '<span class="okc">' + i('check', 's12') + '</span>')}</div>
          <div style="margin-top:8px" class="md"><p>已修复:匹配前统一转小写并去掉末尾 <code>/</code>,新增 5 个测试,全部通过。</p></div>${C.footer().replace(/<span>Sonnet[^<]*<\/span>/, '<span>Sonnet 5.5 · $0.02</span>')}</div>
        <div style="padding:0 12px 10px">${C.cmpB({ narrow: true })}</div>
      </div>
      ${actBar(0)}${exp({ w: 200 })}${editor({ panel: termPanel(130), minimap: false })}
    </div>${status()}</div>`;

  const ideC = () => `<div class="fc" style="height:100%">${titleBar({ branch: 'main' })}
    <div class="f grow rel" style="min-height:0">${actBar(0)}${exp({ w: 220 })}
      <div class="grow fc rel" style="min-width:0">${etabs()}${crumbs()}
        <div class="grow rel" style="min-height:0;overflow:hidden">${editorBody({ pad: 14 })}
          <div class="abs card" style="left:60px;right:96px;top:276px;box-shadow:var(--shadow-pop);border-color:var(--ac);overflow:hidden;z-index:3">
            <div class="f ai g6" style="height:26px;padding:0 10px;background:var(--ac-soft);color:var(--ac-tx)">${i('wand', 's12')}<b>Claude 建议</b><span class="tx2">把大小写处理抽成 normalize()</span><span class="grow"></span>${btn('接受 ⌘⏎', 'sm pri')}${btn('拒绝 Esc', 'sm')}</div>
            <div class="dl del"><span class="no">16</span><span class="tx">        let key = p.to_ascii_lowercase();</span></div><div class="dl del"><span class="no">17</span><span class="tx">        let key = key.trim_end_matches('/');</span></div>
            <div class="dl add"><span class="no">16</span><span class="tx">        let key = normalize(p);</span></div></div>
        </div>
        <div class="abs fc ai" style="left:0;right:72px;bottom:12px;z-index:4;pointer-events:none"><div style="width:620px;pointer-events:auto" class="fc g6">
          <div class="card pad" style="box-shadow:var(--shadow-pop)"><div class="f ai g6" style="margin-bottom:4px">${i('spark', 's12 acc')}<b>已修复</b><span class="tx3 s11">· 编辑 router.rs · 12 项测试通过</span><span class="grow"></span>${ib('max', 'sm')}${ib('x', 'sm')}</div><div class="md" style="color:var(--tx2)">匹配前统一转小写并去掉末尾 <code>/</code>。要不要把它抽成 <code>normalize()</code>?</div></div>
          <div class="cmp focus" style="box-shadow:var(--shadow-pop)"><div class="in">在编辑器旁追问,或 ⌘K 就地改写选中代码…</div><div class="bar">${chip('编辑', 'edit', 'on')}${chip('提问', 'bot')}${chip('接受编辑', 'shield')}${chip('Sonnet 5.5', 'spark')}<span class="grow"></span>${C.ctxRing(28)}<span class="ib send off">${i('send')}</span></div></div></div></div>
      </div>
    </div>${status()}</div>`;

  S.add({
    id: 'ide', group: 'ide', title: 'IDE 主界面',
    desc: '对话与编辑器并排的 IDE 模式。三种布局取向:编辑优先、对话优先、浮动对话。',
    notes: ['大纲 / 时间线默认折成标题行,文件树多出约 150px 高度;展开仍可用。', '标题栏只保留:侧栏开关、前进后退、项目搜索、面板开关、Agent|IDE 分段。', '面包屑显示符号层级(Router › match_path),状态栏补上语言服务器状态、缩进与当前权限模式。'],
    variants: [
      { label: '编辑优先 · 对话在右', tag: '推荐', rec: true, note: '最接近 Cursor / VS Code 的肌肉记忆;对话栏 340px,可拖宽,可与终端并存。', render: ideA },
      { label: '对话优先 · 对话在左', note: '把对话放在主视线(左),编辑器退到辅助;适合「让 Claude 干活、我只是审查」的工作方式。', render: ideB },
      { label: '浮动对话 + 就地改写', note: '编辑器占满;对话是底部悬浮的小面板,选中代码 ⌘K 就地出 diff 并接受;适合以手写为主、偶尔求助。', render: ideC },
    ],
  });

  /* ───────── IDE 欢迎(空编辑器) ───────── */
  const sc = [['快速打开文件', '⌘P'], ['命令面板', '⇧⌘P'], ['在项目中搜索', '⇧⌘F'], ['转到符号', '⇧⌘O'], ['显示 / 隐藏终端', '⌃`'], ['引用选中内容到对话', '⌘L'], ['切换侧栏 / 对话栏', '⌘B'], ['切换到 Agent', '⇧⌘E']];
  const welcomeIdeA = () => `<div class="fc" style="height:100%">${titleBar({ branch: 'main' })}<div class="f grow" style="min-height:0">${actBar(0)}${exp()}
    <div class="grow fc ai jc" style="min-width:0"><div class="fc ai g8" style="margin-bottom:18px"><span class="logo" style="width:30px;height:30px;border-radius:9px">${i('mark', 's16')}</span><b class="s14">router-demo</b><span class="tx3">打开一个文件,或者从下面开始</span></div>
      <div style="display:grid;grid-template-columns:repeat(2,230px);gap:2px 40px">${sc.map(([t, k]) => `<div class="f ai jb" style="height:26px"><span class="tx2">${t}</span>${kbd(...k.split(''))}</div>`).join('')}</div></div>
    ${ideChat()}</div>${status()}</div>`;
  const welcomeIdeB = () => `<div class="fc" style="height:100%">${titleBar({ branch: 'main' })}<div class="f grow" style="min-height:0">${actBar(0)}${exp()}
    <div class="grow fc" style="min-width:0;padding:30px 44px;overflow:hidden"><div class="s20 b">router-demo</div><div class="tx3" style="margin:2px 0 16px">main · 3 项未提交改动 · 上次打开 router.rs</div>
      <div class="f g12"><div class="card pad grow"><div class="f ai g6 b" style="margin-bottom:6px">${i('clock', 's12')}最近文件</div>${['router.rs|src/', 'case.rs|src/', 'Cargo.toml|', 'CHANGELOG.md|'].map(r => { const [n, d] = r.split('|'); return `<div class="f ai g6" style="height:24px">${i('file', 's12 tx3')}<span class="m">${n}</span><span class="tx3">${d}</span></div>`; }).join('')}</div>
        <div class="card pad grow"><div class="f ai g6 b" style="margin-bottom:6px">${i('diff', 's12')}待审查改动</div>${['router.rs|+4 −1', 'case.rs|+12', 'CHANGELOG.md|+2 −3'].map(r => { const [n, d] = r.split('|'); return `<div class="f ai g6" style="height:24px">${i('file', 's12 acc')}<span class="m grow">${n}</span><span class="tx3 mono s11">${d}</span></div>`; }).join('')}</div>
        <div class="card pad grow"><div class="f ai g6 b" style="margin-bottom:6px">${i('bot', 's12')}让 Claude 来</div>${['解释项目结构', '补充缺失的测试', '审查未提交改动'].map(t => `<div class="f ai g6" style="height:24px">${i('arrow', 's12 acc')}<span>${t}</span></div>`).join('')}</div></div>
      <div class="b s13" style="margin:18px 0 6px">常用快捷键</div>
      <div class="f g20 wrap">${sc.slice(0, 5).map(([t, k]) => `<span class="f ai g6">${kbd(...k.split(''))}<span class="tx2">${t}</span></span>`).join('')}</div></div>
    ${ideChat()}</div>${status()}</div>`;
  S.add({
    id: 'ide-welcome', group: 'ide', title: 'IDE 空编辑器 / 欢迎页',
    desc: '没打开任何文件时的编辑区。现在只有一列快捷键,可以更有用。',
    notes: ['快捷键改成两列网格,每项带按键;方案 B 额外给出最近文件、待审查改动与「让 Claude 来」的入口。'],
    variants: [
      { label: '两列快捷键', note: '安静、不抢视线,只是更整齐。', render: welcomeIdeA },
      { label: '工作台概览', tag: '推荐', rec: true, note: '最近文件 + 待审查改动 + 快速起手,把「接下来做什么」放在眼前。', render: welcomeIdeB },
    ],
  });

  /* ───────── 源代码管理 + Git 图谱 + 差异标签 ───────── */
  const chg = (n, d, g, tone) => `<div class="tr" style="padding-left:12px">${i('file', 's12 tx2')}<span class="m">${n}</span><span class="tx3 s11 grow ell">${d}</span><span class="g ${tone}">${g}</span></div>`;
  const graphRow = (h, m, a, t, refs, on) => `<div class="f ai g8" style="height:24px;padding:0 8px;${on ? 'background:var(--bg-active)' : ''}"><span style="width:28px;flex:none;position:relative;height:24px"><i style="position:absolute;left:13px;top:0;bottom:0;width:1.5px;background:var(--ac);opacity:.55"></i><i style="position:absolute;left:9px;top:8px;width:9px;height:9px;border-radius:50%;background:${on ? 'var(--ac)' : 'var(--bg-card)'};border:1.5px solid var(--ac)"></i></span><span class="grow ell">${m}</span>${refs}<span class="tx3 s10 nw">${a} · ${t}</span></div>`;
  const diffTab = () => `<div class="grow fc" style="min-width:0">
    <div class="etabs"><div class="etab on">${i('diff', 's12 acc')}<span style="font-style:italic">router.rs (工作区)</span><span class="x">${i('x', 's12')}</span></div><div class="etab">${i('file', 's12 tx3')}case.rs</div></div>
    <div class="f ai g8" style="height:28px;padding:0 10px;border-bottom:1px solid var(--bd-soft)"><b>router.rs</b><span class="tx3">src/ · 更改</span><span class="okc mono">+4</span><span class="errc mono">−1</span><span class="grow"></span>${seg(['左右对比', '统一'], 0, 'sm')}${ib('chevU', 'sm')}${ib('chevD', 'sm')}${btn('暂存块', 'sm')}${btn('放弃', 'sm danger')}</div>
    <div class="f grow" style="min-height:0;overflow:hidden">
      <div class="grow" style="border-right:1px solid var(--bd);padding-top:4px;min-width:0">${['41|fn match_path(&self, p: &str) {|ctx', '42|    self.routes.get(p)|del', '43|}|ctx', '', '', ''].map(r => { const [n, t, c] = r.split('|'); return `<div class="dl ${c || 'ctx'}" style="${c ? '' : 'height:19px'}"><span class="no">${n || ''}</span><span class="tx" style="${c === 'del' ? '' : ''}">${t ? K.esc(t).replace(/^(\s*)/, '$1') : ''}</span></div>`; }).join('').replace(/dl (add|del|ctx)([^>]*)><span class="no">(\d*)<\/span><span class="tx">/g, (m, c, r, n) => `dl ${c}${r}><span class="no">${n}</span><span class="tx" style="padding-left:4px">`)}</div>
      <div class="grow" style="padding-top:4px;min-width:0"><div class="dl ctx"><span class="no">41</span><span class="tx">fn match_path(&amp;self, p: &amp;str) {</span></div><div class="dl add"><span class="no">42</span><span class="tx">    let key = p.to_ascii_lowercase();</span></div><div class="dl add"><span class="no">43</span><span class="tx">    let key = key.trim_end_matches('/');</span></div><div class="dl add"><span class="no">44</span><span class="tx">    self.routes.get(key)</span></div><div class="dl ctx"><span class="no">45</span><span class="tx">}</span></div></div>
      <div class="mm" style="width:56px"><i style="width:60%"></i><i style="width:80%;background:var(--err);opacity:.6"></i><i style="width:70%;background:var(--ok);opacity:.7"></i><i style="width:75%;background:var(--ok);opacity:.7"></i><i style="width:20%"></i></div></div></div>`;
  const scmPanel = () => `<div class="exp" style="width:272px">
    <div class="sec-h" style="margin-top:4px"><span class="grow">源代码管理</span>${ib('branch', 'sm')}${ib('refresh', 'sm')}${ib('more', 'sm')}</div>
    <div style="padding:0 8px 6px" class="fc g6"><div class="cmp" style="border-radius:7px"><div class="in v" style="min-height:44px;padding:6px 8px">fix: 路由匹配忽略大小写并去除末尾斜杠</div><div class="bar" style="padding:2px 4px 4px">${ib('star', 'sm ac')}<span class="grow"></span><span class="btn pri sm" style="border-radius:6px 0 0 6px">${i('check', 's12')}提交</span><span class="btn pri sm" style="border-radius:0 6px 6px 0;border-left:1px solid rgba(255,255,255,.35);padding:0 4px">${i('chevD', 's12')}</span></div></div>
      <div class="f ai g6 s11 tx3">${i('star', 's12 acc')}<span>AI 已根据 3 个文件生成说明</span></div></div>
    <div class="sec-h">${i('chevD', 's12')}暂存的更改 ${cnt(1, 'mute')}<span class="grow"></span>${ib('minus', 'sm')}</div>${chg('CHANGELOG.md', '', 'M', 'warnc')}
    <div class="sec-h">${i('chevD', 's12')}更改 ${cnt(2, 'mute')}<span class="grow"></span>${ib('undo', 'sm')}${ib('plus', 'sm')}</div>${chg('router.rs', 'src/', 'M', 'warnc').replace('class="tr"', 'class="tr sel"')}${chg('case.rs', 'src/', 'U', 'okc')}
    <div class="sec-h" style="margin-top:6px;border-top:1px solid var(--bd)">${i('chevD', 's12')}图形 ${chip('当前分支', '', 'on')}<span class="grow"></span>${ib('refresh', 'sm')}</div>
    <div class="grow" style="overflow:hidden">${graphRow('a1', 'fix: 路径末尾斜杠', '李文娇', '3 天前', '<span class="bdg ac">main</span><span class="bdg">origin</span>', 1)}${graphRow('a2', 'feat: 添加 case.rs 测试', '李文娇', '5 天前', '')}${graphRow('a3', 'refactor: 拆分 middleware', '王凯', '1 周前', '')}${graphRow('a4', 'chore: 版本升到 0.3.1', '李文娇', '2 周前', '<span class="bdg info">v0.3.1</span>')}${graphRow('a5', 'docs: 更新 README', '王凯', '2 周前', '')}</div>
  </div>`;
  S.add({
    id: 'ide-scm', group: 'ide', title: '源代码管理 · 图谱 · 差异',
    desc: '对标 VS Code SCM,保持功能:提交框、暂存 / 更改分组、AI 提交说明、Git 图谱、左右对比差异标签。',
    notes: ['提交按钮做成分体按钮(提交 / 修改上次 / 提交并推送);AI 说明状态用一行小字,不再占提交框空间。', '图谱与更改在同一栏上下排布,标题行固定,列表区独立滚动;差异标签顶栏提供上 / 下一处、暂存块、放弃。'],
    variants: [{ label: 'SCM + 差异标签', tag: '推荐', rec: true, note: '', render: () => `<div class="fc" style="height:100%">${titleBar({ branch: 'main' })}<div class="f grow" style="min-height:0">${actBar(2)}${scmPanel()}${diffTab()}${ideChat(320)}</div>${status()}</div>` }],
  });

  /* ───────── 搜索与问题 ───────── */
  const hit = (n, t, h) => `<div class="tr" style="padding-left:30px;height:20px"><span class="tx3 mono s10" style="width:22px;text-align:right">${n}</span><span class="ell grow">${t.replace(h, `<mark style="background:color-mix(in srgb,var(--ac) 30%,transparent);color:inherit;border-radius:2px">${h}</mark>`)}</span></div>`;
  const searchPanel = () => `<div class="exp" style="width:272px"><div class="sec-h" style="margin-top:4px"><span class="grow">搜索</span>${ib('refresh', 'sm')}${ib('x', 'sm')}</div>
    <div style="padding:0 8px" class="fc g4"><div class="f g4 ai">${i('chevR', 's12 tx3')}<div class="inp grow" style="border-color:var(--ac)"><span class="v grow">to_ascii_lowercase</span><span class="chip on" style="height:18px;padding:0 4px">Aa</span><span class="chip" style="height:18px;padding:0 4px">ab</span><span class="chip" style="height:18px;padding:0 4px">.*</span></div></div>
      <div class="f ai g6 s11 tx3" style="padding-left:16px"><span>4 个结果,3 个文件</span><span class="grow"></span><span class="acc">展开筛选</span></div></div>
    <div class="grow" style="overflow:hidden;margin-top:4px">
      <div class="tr" style="padding-left:6px">${i('chevD', 's12 tx3')}${i('file', 's12 acc')}<b class="grow">router.rs</b><span class="tx3 s10">src/</span>${cnt(2, 'mute')}</div>${hit(16, 'let key = p.to_ascii_lowercase();', 'to_ascii_lowercase')}${hit(22, 'path.to_ascii_lowercase(), route', 'to_ascii_lowercase')}
      <div class="tr" style="padding-left:6px">${i('chevD', 's12 tx3')}${i('file', 's12 tx2')}<b class="grow">case.rs</b><span class="tx3 s10">tests/</span>${cnt(1, 'mute')}</div>${hit(8, 'assert_eq!(r.to_ascii_lowercase(), …)', 'to_ascii_lowercase')}
      <div class="tr" style="padding-left:6px">${i('chevR', 's12 tx3')}${i('file', 's12 tx2')}<b class="grow">utils.rs</b><span class="tx3 s10">src/</span>${cnt(1, 'mute')}</div></div></div>`;
  const problems = () => `<div class="fc" style="height:160px;border-top:1px solid var(--bd);flex:none"><div class="panel-h">${['终端', '问题 1', '输出', '任务'].map((t, k) => `<span class="pt ${k === 1 ? 'on' : ''}">${t}</span>`).join('')}<span class="grow"></span>${seg(['全部', '错误', '警告 1'], 0, 'sm')}${ib('x', 'sm')}</div>
    <div style="padding:4px 0"><div class="tr" style="padding-left:8px">${i('chevD', 's12 tx3')}${i('file', 's12 acc')}<b>router.rs</b><span class="tx3 s11">src/</span>${cnt(1, 'mute')}</div><div class="tr" style="padding-left:30px">${i('alert', 's12 warnc')}<span class="grow ell">变量 <span class="mono">key</span> 遮蔽了先前的绑定</span><span class="tx3 s10">rust-analyzer · 行 16</span></div></div></div>`;
  S.add({
    id: 'ide-search', group: 'ide', title: '项目搜索 · 问题面板',
    desc: '⌘⇧F 搜索与底部「问题」面板。保持 VS Code 的使用习惯。',
    notes: ['开关(Aa / ab / .*)放进输入框内,替换行用左侧箭头展开;结果行的命中词高亮,折叠箭头与计数徽标在行内右对齐。', '问题面板按文件分组,行尾标出来源与行号,点击跳转。'],
    variants: [{ label: '搜索 + 问题', tag: '推荐', rec: true, note: '', render: () => `<div class="fc" style="height:100%">${titleBar({ branch: 'main' })}<div class="f grow" style="min-height:0">${actBar(1)}${searchPanel()}${editor({ panel: problems() })}${ideChat(320)}</div>${status()}</div>` }],
  });
})();
