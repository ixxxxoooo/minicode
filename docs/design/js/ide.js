/* 设计稿 v3 · IDE 部件:编辑器、底部面板、文件树、搜索、源代码管理(导航栏与工作区底部面板复用这里的树与终端)
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, rust } = U;
  const IDE = {};

  const SRC = `use std::collections::HashMap;

/// 路由表:路径 → 处理器
pub struct Router {
    routes: HashMap<String, Handler>,
    fallback: Option<Handler>,
}

impl Router {
    /// 注册一条路由,路径先归一化
    pub fn add(&mut self, path: &str, h: Handler) {
        let key = normalize(path);
        self.routes.insert(key, h);
    }

    /// 匹配路径:忽略大小写与末尾斜杠
    pub fn match_path(&self, path: &str) -> Option<&Handler> {
        let key = normalize(path);
        self.routes.get(&key).or(self.fallback.as_ref())
    }
}

fn normalize(path: &str) -> String {
    path.trim_end_matches('/').to_ascii_lowercase()
}`.split('\n');
  const GIT = { 12: 'gm', 13: 'ga', 18: 'gm', 19: 'ga', 23: 'ga', 24: 'ga', 25: 'ga' };
  const CUR = 17, LENS = { 11: '4 个引用 · 问 Claude', 17: '3 个引用 · 2 个测试 · 问 Claude' };
  /** 只在文本节点里高亮某个词(不碰标签属性) */
  const mark = (html, word, cls) => html.split(/(<[^>]+>)/).map(s => s[0] === '<' ? s : s.replace(new RegExp('\\b' + word + '\\b', 'g'), `<span class="${cls}">${word}</span>`)).join('');
  const hl = (s) => s.replace(/match_path/g, '<mark class="hit">match_path</mark>');
  /** 第 n 行(1 起)顶边的像素:CodeLens 占 16px,出现在它所属行的上方 */
  const ytop = (n) => (n - 1) * 20 + Object.keys(LENS).filter(k => +k <= n).length * 16;
  const tgl = (t, on, tip) => `<span class="tgl ${on ? 'on' : ''}" ${tip ? `data-tip="${tip}"` : ''}>${t}</span>`;

  /* ───────── 差异数据:两个改动块,行内词级高亮 ───────── */
  /** 带词级高亮的一行:sub 是行内真正变化的那一段 */
  const rustWd = (src, sub) => { const i = sub ? src.indexOf(sub) : -1; return i < 0 ? rust(src) : `${rust(src.slice(0, i))}<mark class="wd">${rust(sub)}</mark>${rust(src.slice(i + sub.length))}`; };
  /** 行:k = '' / add / del / chg(同一行改了几个词);o / n = [行号, 文本, 变化词] */
  const DV = [
    { fold: 10 },
    { hunk: '@@ 第 11 行 · 注册前先归一化路径', id: 1 },
    { k: '', o: [11, '    pub fn add(&mut self, path: &str, h: Handler) {'], n: [11, '    pub fn add(&mut self, path: &str, h: Handler) {'] },
    { k: 'add', n: [12, '        let key = normalize(path);'] },
    { k: 'chg', o: [12, '        self.routes.insert(path.to_string(), h);', 'path.to_string()'], n: [13, '        self.routes.insert(key, h);', 'key'] },
    { k: '', o: [13, '    }'], n: [14, '    }'] },
    { fold: 2 },
    { hunk: '@@ 第 17 行 · 匹配前先归一化路径', id: 2 },
    { k: '', o: [17, '    pub fn match_path(&self, path: &str) -> Option<&Handler> {'], n: [17, '    pub fn match_path(&self, path: &str) -> Option<&Handler> {'] },
    { k: 'add', n: [18, '        let key = normalize(path);'] },
    { k: 'chg', o: [18, '        self.routes.get(path)', 'path'], n: [19, '        self.routes.get(&key).or(self.fallback.as_ref())', '&key).or(self.fallback.as_ref()'] },
    { k: '', o: [19, '    }'], n: [20, '    }'] },
    { fold: 6 },
  ];
  const dvCell = (side, r) => {
    const v = r[side], bad = r.k === 'chg' ? (side === 'o' ? 'del' : 'add') : r.k === 'add' && side === 'n' ? 'add' : r.k === 'del' && side === 'o' ? 'del' : '';
    return v ? `<div class="dvc ${bad}"><span class="no">${v[0]}</span><span class="tx">${bad ? rustWd(v[1], v[2]) : rust(v[1])}</span></div>` : '<div class="dvc e"></div>';
  };
  /** 块头:暂存区视图里把「暂存」换成「取消暂存」 */
  const dvHunk = (r, o) => `<div class="dv-hunk"><span class="mono grow ell">${r.hunk}</span>${o.staged ? btn('取消暂存此块', 'ghost', 'minus') : `${btn('暂存此块', 'ghost', 'plus')}${btn('还原此块', 'ghost', 'undo')}`}${ib('copy', 'sm', '复制补丁')}</div>`;
  /** 统一视图的一行 */
  const uniRows = () => DV.map(r => {
    if (r.fold) return `<div class="dv-fold">${ic('more', 's12')}已折叠 ${r.fold} 行未更改 · 点击展开</div>`;
    if (r.hunk) return '';
    if (r.k === 'chg') return `<div class="ur del"><span class="no">${r.o[0]}</span><span class="no no2"></span><span class="sg">−</span><span class="tx">${rustWd(r.o[1], r.o[2])}</span></div><div class="ur add"><span class="no"></span><span class="no no2">${r.n[0]}</span><span class="sg">+</span><span class="tx">${rustWd(r.n[1], r.n[2])}</span></div>`;
    const a = r.k === 'add';
    return `<div class="ur ${a ? 'add' : ''}"><span class="no">${a ? '' : r.o[0]}</span><span class="no no2">${r.n[0]}</span><span class="sg">${a ? '+' : ''}</span><span class="tx">${rust((a ? r.n : r.o)[1])}</span></div>`;
  }).join('');
  /** 差异标签。o.unified 统一视图 / o.staged 暂存区 ↔ HEAD / o.sel 选中行后的行级操作条 */
  IDE.diffView = (o = {}) => `<div class="dv-bar">${ic('diff', 's12')}<b>router.rs</b><span class="t3">${o.staged ? 'HEAD ↔ 暂存区' : '暂存区 ↔ 工作区'}</span>${tag('+4 −2', 'ok')}<span class="grow"></span><span class="t3 fs-cap num">1 / 2</span>
      ${ib('chevU', 'sm', '上一处差异 ⌥F5')}${ib('chevD', 'sm', '下一处差异 F5')}${U.seg(['左右', '统一'], o.unified ? 1 : 0)}${ib('wand', 'sm', '忽略空白')}${ib('edit', 'sm', '在差异里直接编辑(右侧)')}${btn('打开文件', 'sm ghost', 'ext')}</div>
    ${o.sel ? `<div class="dv-sel card">${ic('check', 's12 c-ac')}<span class="grow">已选中 2 行</span>${btn('暂存所选行', 'sm soft', 'plus')}${btn('还原所选行', 'sm ghost', 'undo')}</div>` : ''}
    <div class="dv-wrap"><div class="dv ${o.unified ? 'uni' : ''}">${o.unified ? uniRows() : `<div class="dv-head"><span>${o.staged ? 'HEAD' : '暂存区'}</span><span>${o.staged ? '暂存区' : '工作区'}</span></div>${DV.map(r => r.fold ? `<div class="dv-fold">${ic('more', 's12')}已折叠 ${r.fold} 行未更改 · 点击展开</div>` : r.hunk ? dvHunk(r, o) : `<div class="dvr">${dvCell('o', r)}${dvCell('n', r)}</div>`).join('')}`}</div>
      <div class="mm dvm"><div class="vp" style="height:90px"></div>${DV.concat(DV, DV).filter(r => !r.hunk).map(r => `<i style="width:${r.fold ? 100 : 55}%;${r.k === 'add' || r.k === 'chg' ? 'background:var(--ok);opacity:.7' : r.fold ? 'opacity:.12' : ''}"></i>`).join('')}</div></div>`;

  /* ───────── 文件内容:代码区 + 迷你地图 + 概览标尺 + 浮层 ───────── */
  const hitLines = () => SRC.map((l, k) => /match_path/.test(l) ? k + 1 : 0).filter(Boolean);
  const mmRow = (l, n, o) => { const c = /^\s*\/\//.test(l) ? 'c' : /\b(pub|fn|impl|struct|use)\b/.test(l) ? 'm' : l.trim() ? '' : 'z';
    const cls = [c, o.hits && o.hits.includes(n) ? 'hl' : '', o.occ && o.occ.includes(n) ? 'oc' : '', o.edit && o.edit.includes(n) ? 'ed' : ''].filter(Boolean).join(' ');
    return `<i class="${cls}" style="width:${Math.min(100, l.length * 2.4)}%;${l.trim() ? '' : 'opacity:0'}"></i>`; };
  /** 浮层:位置全部相对代码区,字符宽 7.2px、行高 20px */
  const OVS = {
    find: (o) => `<div class="ov find pop a-pop-up" style="top:calc(var(--crop, 0) * 1px + 4px)"><div class="rowi">${ic(o.rep ? 'chevD' : 'chevR', 's12 t3')}<div class="inp focus"><span class="v grow">match_path</span>${tgl('Aa', 0, '区分大小写 ⌥⌘C')}${tgl('ab', 1, '全词匹配 ⌥⌘W')}${tgl('.*', 0, '正则 ⌥⌘R')}</div><span class="cntl num">2 / 4</span>${ib('chevU', 'sm', '上一个 ⇧⌘G')}${ib('chevD', 'sm', '下一个 ⌘G')}${ib('list', 'sm', '在选区中查找')}${ib('x', 'sm', '关闭 Esc')}</div>
      ${o.rep ? `<div class="rowi"><span style="width:12px"></span><div class="inp"><span class="v grow">find_route</span>${tgl('AB', 0, '保留大小写')}</div>${ib('check', 'sm', '替换 ⏎')}${ib('layers', 'sm', '全部替换 ⌥⌘⏎')}</div>` : ''}</div>`,
    hover: () => `<div class="ov hover-card pop a-pop-up" style="top:${ytop(18) + 22}px;left:190px"><div class="sig">${rust('fn normalize(path: &str) -> String')}</div><div class="doc">去掉末尾斜杠并转成小写。注册与匹配共用,保证 <code>/Users/</code> 与 <code>/users</code> 命中同一条路由。</div>
      <div class="t3 fs-cap">router.rs · 第 23 行 · 4 个引用</div><div class="act"><span>转到定义 ⌘点击</span><span>查看引用 ⇧F12</span><span>问 Claude ⌘L</span></div></div>`,
    complete: () => `<div class="ov comp pop a-pop-up" style="top:${ytop(19) + 22}px;left:194px"><div class="it on"><span class="ik f">f</span><span class="grow">get</span><span class="k">Option&lt;&amp;V&gt;</span></div><div class="it"><span class="ik f">f</span><span class="grow">get_mut</span><span class="k">Option&lt;&amp;mut V&gt;</span></div><div class="it"><span class="ik f">f</span><span class="grow">get_key_value</span></div><div class="it"><span class="ik f">f</span><span class="grow">contains_key</span><span class="k">bool</span></div><div class="it"><span class="ik v">v</span><span class="grow">routes</span><span class="k">HashMap</span></div>
      <div class="det"><span class="mono">pub fn get&lt;Q&gt;(&amp;self, k: &amp;Q) -&gt; Option&lt;&amp;V&gt;</span><br><span class="t3">返回键对应值的引用。</span></div><div class="f ai g3 t3 fs-micro" style="padding:3px 7px 2px">${kbd('⏎')}接受 ${kbd('⇥')}替换 ${kbd('esc')}关闭 <span class="grow"></span><span class="c-ac">✦ Claude 建议</span></div></div>`,
    peek: () => `<div class="ov peek a-pop-up" style="top:${ytop(18) + 22}px;height:176px"><div class="hd">${ic('code', 's12')}<b class="t2">normalize</b><span>router.rs · 第 23 行</span><span class="grow"></span><span>2 个定义 · 4 个引用</span>${ib('ext', 'sm', '在编辑器中打开 ⌥⏎')}${ib('x', 'sm')}</div>
      <div class="bd"><div class="lst"><div class="row on">${ic('code', 's12')}<span class="ell">router.rs</span><span class="aux">23</span></div><div class="grp-h" style="height:20px;padding:0 6px">引用 4</div><div class="row">${ic('code', 's12')}<span class="ell">router.rs</span><span class="aux">12</span></div><div class="row">${ic('code', 's12')}<span class="ell">router_test.rs</span><span class="aux">8</span></div></div>
        <div class="cd"><div>${rust('fn normalize(path: &str) -> String {')}</div><div class="hl">${rust("    path.trim_end_matches('/').to_ascii_lowercase()")}</div><div>${rust('}')}</div></div></div></div>`,
    action: () => `<div class="ov menu-pop pop a-pop-up" style="top:${ytop(13) + 22}px;left:60px"><div class="cap">快速修复</div><div class="mi on">${ic('spark', 's12 c-ac')}<span class="grow">让 Claude 修复这个警告</span>${kbd('⌘', '.')}</div><div class="mi">${ic('wand', 's12')}<span class="grow">给返回值加 <span class="mono">let _ =</span></span></div><div class="mi">${ic('eyeOff', 's12')}<span class="grow">为这一行忽略此警告</span></div>
      <div class="sep"></div><div class="cap">重构</div><div class="mi">${ic('layers', 's12')}<span class="grow">提取为函数</span></div><div class="mi">${ic('edit', 's12')}<span class="grow">重命名符号</span>${kbd('F2')}</div><div class="mi">${ic('more', 's12')}<span class="grow">源代码操作…</span></div></div>`,
    rename: () => `<div class="ov rename pop a-pop-up" style="top:${ytop(23) + 22}px;left:60px"><div class="inp"><span class="v">normalize_path</span><span class="cursor"></span></div><div class="t3 fs-cap">将更新 <b class="t2">4 处引用 · 2 个文件</b></div><div class="f ai g3 t3 fs-micro">${kbd('⏎')}重命名 ${kbd('⌃', '⏎')}预览改动 ${kbd('esc')}取消</div></div>`,
    ask: () => `<div class="ov seltool pop a-pop-up" style="top:${ytop(18) - 28}px;left:96px">${btn('问 Claude ⌘L', 'sm', 'chat')}${btn('编辑 ⌘I', 'sm ghost', 'edit')}${btn('解释', 'sm ghost', 'info')}${ib('plus', 'sm', '添加到对话')}</div>`,
    inline: () => `<div class="ov inl pop a-pop-up" style="top:${ytop(18) + 44}px;left:70px"><div class="in">${ic('spark', 's12 c-ac')}<span class="grow">匹配失败时回退到原始路径</span><span class="cursor"></span></div><div class="f ai g3 t3 fs-micro">${chip('Sonnet 5.5', '', 'line')}<span class="grow"></span>${kbd('⏎')}生成 ${kbd('esc')}取消</div></div>`,
    claude: () => `<div class="ov hunkbar pop a-pop-up" style="top:${ytop(18) - 22}px;right:24px;border-radius:var(--r-sm)">${ic('spark', 's12 c-vio')}<span style="margin:0 6px 0 2px">Claude 的修改 · 2 行</span>${btn('撤销 esc', 'ghost no')}${btn('接受 ⌘⏎', 'ghost ok')}${ib('chevD', 'sm')}</div>`,
  };
  /** 文件内容(router.rs)。o.ov 选一个浮层;o.scroll 已滚走的行数(配合 o.sticky 粘性滚动);o.word 同词高亮 */
  IDE.fileView = (o = {}) => {
    if (o.file && o.file !== 'router.rs') return IDE.otherFile(o.file);
    const word = o.word === undefined ? 'Handler' : o.word, ov = o.ov, hits = (o.hl || ov === 'find') ? hitLines() : [];
    const sel = ov === 'ask' || ov === 'inline' ? [18, 19] : [], edit = ov === 'claude' ? [18, 19] : [], occ = word ? SRC.map((l, k) => new RegExp('\\b' + word + '\\b').test(l) ? k + 1 : 0).filter(Boolean) : [];
    const scrollPx = o.scroll ? ytop(o.scroll + 1) : 0;
    const gut = SRC.map((_, k) => { const n = k + 1; return `${LENS[n] ? '<div class="lens-sp"></div>' : ''}<div class="ln ${n === CUR ? 'cur' : ''} ${GIT[n] || ''}">${[4, 9, 11, 17].includes(n) ? ic('chevD', 'fold') : ''}${ov === 'action' && n === 13 ? ic('bulb', 'bulb') : ''}${n}</div>`; }).join('');
    const src = SRC.map((l, k) => { const n = k + 1; let h = rust(l);
      if (hits.length) h = hl(h.split(/(<[^>]+>)/).map(s => s[0] === '<' ? s : s).join(''));
      else if (word) h = mark(h, word, 'occ');
      if (!hits.length && n === 13 && ov !== 'action') h = h.replace('insert', '<span class="sqw">insert</span>'); else if (n === 13) h = h.replace('insert', '<span class="sqw">insert</span>');
      if (hits.length && n === 17) h = h.replace('<mark class="hit">', '<mark class="hit cur-hit">');
      const caret = n === CUR && !ov && !o.hl ? '<span class="cursor"></span><span class="blame">你 · 2 分钟前 · 未提交</span>' : '';
      return `${LENS[n] ? `<div class="lens"><span>${LENS[n].split(' · ')[0]}</span>${LENS[n].includes('测试') ? '<span>2 个测试</span>' : ''}<span class="c-ac">问 Claude</span></div>` : ''}<div class="ln ${n === CUR && !ov ? 'cur' : ''} ${sel.includes(n) ? 'sel' : ''} ${edit.includes(n) ? 'edit' : ''}">${h}${caret}</div>`; }).join('');
    const mm = SRC.map((l, k) => mmRow(l, k + 1, { hits, occ: o.hl || ov === 'find' ? [] : occ, edit })).join('');
    const gm = Object.keys(GIT).map(n => `<b class="gm ${GIT[n] === 'ga' ? 'a' : 'm'}" style="top:${4 + (n - 1) * 4}px;height:4px"></b>`).join('');
    const T = SRC.length, rp = (n) => ((n - 0.5) / T * 100).toFixed(1) + '%';
    const ruler = [...hits.map(n => `<b class="h" style="top:${rp(n)}"></b>`), ...(hits.length ? [] : occ.map(n => `<b class="o" style="top:${rp(n)}"></b>`)), `<b class="w" style="top:${rp(13)}"></b>`, ...Object.keys(GIT).map(n => `<b class="${GIT[n] === 'ga' ? 'g' : 'gmod'}" style="top:${rp(+n)}"></b>`), ...edit.map(n => `<b class="ce" style="top:${rp(n)}"></b>`), `<b class="c" style="top:${rp(CUR)}"></b>`].join('');
    const sticky = o.sticky ? `<div class="sticky"><div>${rust('impl Router {')}</div><div>${rust('    pub fn match_path(&self, path: &str) -> Option<&Handler> {')}</div></div>` : '';
    return `<div class="crumbs">src${ic('chevR', 's12')}<b>router.rs</b>${ic('chevR', 's12')}<span class="sym">${ic('cube', 's12')}impl Router</span>${ic('chevR', 's12')}<b>match_path</b><span class="r"><span>Rust</span><span>UTF-8</span></span></div>
    <div class="code"><div class="gut"><div style="margin-top:-${scrollPx}px">${gut}</div></div>
      <div class="src-wrap">${sticky}<div class="src" style="margin-top:-${scrollPx}px">${src}</div>${ov && OVS[ov] ? OVS[ov](o) : ''}</div>
      <div class="mm"><div class="vp" style="top:${4 + (o.scroll || 0) * 4}px;height:${o.scroll ? 74 : 92}px"></div>${gm}${mm}${o.mmTip ? `<div class="tip pop mono">${SRC.slice(15, 21).map(l => esc(l)).join('\n')}</div>` : ''}</div>
      <div class="ruler">${ruler}</div></div>`;
  };
  const esc = U.esc;
  /** 其它文件:同一套外壳,内容简单一点 */
  const OTHER = {
    'router_test.rs': ['use super::*;', '', '#[test]', 'fn matches_ignoring_case() {', '    let mut r = Router::default();', '    r.add("/users", handler());', '    assert!(r.match_path("/Users").is_some());', '}', '', '#[test]', 'fn matches_trailing_slash() {', '    let mut r = Router::default();', '    r.add("/users", handler());', '    assert!(r.match_path("/users/").is_some());', '}'],
    'Cargo.toml': ['[package]', 'name = "router-demo"', 'version = "0.3.1"', 'edition = "2021"', '', '[dependencies]', 'axum = "0.8"', 'tokio = { version = "1", features = ["full"] }'],
  };
  IDE.otherFile = (f) => { const L = OTHER[f] || [''];
    return `<div class="crumbs">${f.includes('_test') ? 'tests' : '.'}${ic('chevR', 's12')}<b>${f}</b></div><div class="code"><div class="gut"><div>${L.map((_, k) => `<div class="ln">${k + 1}</div>`).join('')}</div></div><div class="src-wrap"><div class="src">${L.map(l => `<div class="ln">${rust(l)}</div>`).join('')}</div></div><div class="mm"><div class="vp" style="height:60px"></div>${L.map(l => `<i style="width:${Math.min(100, l.length * 2.4)}%;${l.trim() ? '' : 'opacity:0'}"></i>`).join('')}</div><div class="ruler"></div></div>`; };
  /** 兼容旧调用:整个编辑区 = 工作区 */
  IDE.editor = (o = {}) => WS.html(o);

  IDE.termLines = () => `<span class="p">➜</span> <b>router-demo</b> <span class="d">git:(</span><span class="e">main</span><span class="d">)</span> cargo test router
<span class="d">    Finished</span> test profile in 1.92s
test matches_ignoring_case ... <span class="p">ok</span>
test matches_trailing_slash ... <span class="p">ok</span>
test result: <span class="p">ok</span>. 12 passed; 0 failed
<span class="p">➜</span> <b>router-demo</b> <span class="d">git:(</span><span class="e">main</span><span class="d">)</span> <span class="caret"></span>`;
  const prob = (lv, msg, src, ln) => `<div class="row">${ic(lv === 'err' ? 'errc' : 'alert', 's12 ' + (lv === 'err' ? 'c-err' : 'c-warn'))}<span class="ell">${msg}</span><span class="t4 fs-cap nw">${src}</span><span class="aux">${ln}</span></div>`;
  const refr = (n, tx, ln, on) => `<div class="row ${on ? 'on' : ''}"><span class="ell mono">${tx}</span><span class="aux">${n}:${ln}</span></div>`;
  /** 底部面板:终端 / 问题 / 输出 / 引用。which:term(默认)/ problems / refs;终端分段里「只读」标签是会话里命令的输出 */
  IDE.bpanel = (which = 'term', o = {}) => `<div class="bpanel ${which === 'problems' || which === 'refs' ? 'tall' : ''}">
    <div class="tabs"><span class="tab ${which === 'term' ? 'on' : ''}" data-act="ptab">${ic('terminal', 's12')}终端</span><span class="tab ${which === 'problems' ? 'on' : ''}" data-act="ptab">问题${cnt(3, which === 'problems' ? 'ac' : '')}</span><span class="tab" data-act="ptab">输出</span>${which === 'refs' ? `<span class="tab on" data-act="ptab">引用${cnt(4, 'ac')}</span>` : ''}<span class="grow"></span>
      <span class="f ai g1">${which === 'problems' ? `${chip('全部', 'filter', 'on')}${ib('refresh', 'sm', '重新检查')}` : which === 'refs' ? '' : IDE.termChips(o) + ib('gear', 'sm', '终端设置', 'data-act="settab" data-tab="terminal" data-goto="settings"')}${ib('max', 'sm', '最大化面板')}${ib('x', 'sm', '收起 ⌘J', 'data-act="panel"')}</span></div>
    ${which === 'problems' ? `<div class="grow scroll" style="padding:2px 6px"><div class="grp-h">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow">router.rs</span>${cnt(2)}</div>${prob('warn', '`fallback` 字段从未被读取', 'rust-analyzer', '行 5')}${prob('warn', '未使用的变量:`h`', 'rust-analyzer', '行 11')}
      <div class="grp-h">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow">router_test.rs</span>${cnt(1)}</div>${prob('err', '找不到函数 `normalize`,需要 `use crate::router::normalize`', 'rust-analyzer', '行 8')}</div>`
      : which === 'refs' ? `<div class="grow scroll" style="padding:2px 6px"><div class="grp-h">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow">router.rs</span>${cnt(3)}</div>${refr(0, rust('let key = normalize(path);'), 12, 1)}${refr(0, rust('let key = normalize(path);'), 18)}${refr(0, rust('fn normalize(path: &str) -> String {'), 23)}
      <div class="grp-h">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow">router_test.rs</span>${cnt(1)}</div>${refr(0, rust('assert_eq!(normalize("/A/"), "/a");'), 8)}</div>`
      : `<div class="term grow">${(o.sel == null ? ((window.APP && APP.st.termSel) || 0) : o.sel) ? '<span class="p">➜</span> <b>router-demo</b> <span class="caret"></span>' : IDE.termLines()}</div>`}</div>`;

  const tr = (d, name, o = {}) => `<div class="row tr ${o.dir ? 'dir' : ''} ${o.on ? 'on' : ''} ${o.g ? 'g' + o.g.toLowerCase() : ''}" style="--d:${d}">${o.dir ? ic(o.open ? 'chevD' : 'chevR', 's12 car') : '<span style="width:12px"></span>'}${ic(o.dir ? 'folder' : o.icon || 'fileText', 's12')}<span class="ell">${name}</span>${o.g ? `<span class="g">${o.g}</span>` : ''}</div>`;
  /** 文件树。head:是否带区块标题 */
  IDE.tree = (head = true) => `${head ? `<div class="grp-h">${ic('chevD', 's12')}<span class="grow">ROUTER-DEMO</span>${ib('plus', 'sm', '新建文件')}${ib('refresh', 'sm', '刷新')}</div>` : ''}
    <div class="stagger">${tr(0, 'src', { dir: 1, open: 1 })}${tr(1, 'handlers', { dir: 1 })}${tr(1, 'main.rs', { icon: 'code' })}${tr(1, 'router.rs', { icon: 'code', on: 1, g: 'M' })}${tr(1, 'server.rs', { icon: 'code' })}
    ${tr(0, 'tests', { dir: 1, open: 1 })}${tr(1, 'router_test.rs', { icon: 'code', g: 'M' })}${tr(1, 'case_test.rs', { icon: 'code', g: 'A' })}
    ${tr(0, 'docs', { dir: 1 })}${tr(0, 'Cargo.toml', { icon: 'gear' })}${tr(0, 'CLAUDE.md')}${tr(0, 'README.md')}</div>
    ${head ? `<div class="sep" style="margin:8px 2px"></div><div class="grp-h">${ic('chevR', 's12')}大纲</div><div class="grp-h">${ic('chevR', 's12')}时间线</div>` : ''}`;

  /** 搜索命中行。rep:替换预览(红色删除线旧文 + 绿色新文) */
  const hit = (pre, m, post, ln, rep) => `<div class="row hitrow" style="padding-left:26px"><span class="ell mono">${pre}${rep ? `<del>${m}</del><ins>${rep}</ins>` : `<mark class="hit">${m}</mark>`}${post}</span><span class="acts">${ib('x', 'sm', '忽略此处')}</span><span class="aux swap">${ln}</span></div>`;
  /** 项目内搜索。o.rep:展开替换行(并在命中行显示替换预览) */
  IDE.search = (o = {}) => { const r = o.rep ? 'find_route' : '';
    return `<div class="fc g2" style="padding:0 2px 6px"><div class="f g1 ai"><span class="t3" style="width:14px">${ic(o.rep ? 'chevD' : 'chevR', 's12')}</span><div class="inp focus grow">${ic('search', 's12')}<span class="v grow">match_path</span>${tgl('Aa', 0, '区分大小写 ⌥⌘C')}${tgl('ab', 1, '全词匹配 ⌥⌘W')}${tgl('.*', 0, '正则 ⌥⌘R')}</div></div>
      ${o.rep ? `<div class="f g1 ai"><span style="width:14px"></span><div class="inp grow">${ic('swap', 's12')}<span class="v grow">find_route</span>${tgl('AB', 0, '保留大小写 ⌥⌘P')}${ib('check', 'sm', '全部替换 ⌥⌘↩')}</div></div>` : ''}
      <div class="f ai g2 t3 fs-cap" style="padding-left:16px">${tgl('···', 0, '要包含 / 排除的文件')}<span class="grow"></span>${o.rep ? '<span>仅替换列出的匹配</span>' : ''}</div></div>
    <div class="t3 fs-cap f ai" style="padding:0 8px 4px"><span class="grow">2 个文件中 4 处结果</span>${ib('refresh', 'sm', '重新搜索')}${ib('x', 'sm', '清除结果')}</div>
    <div class="stagger"><div class="row proj">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow ell">router.rs</span>${cnt(2)}</div>${hit('pub fn ', 'match_path', '(&self', 18, r)}${hit('// ', 'match_path', ' 忽略', 16, r)}
    <div class="row proj">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow ell">router_test.rs</span>${cnt(2)}</div>${hit('r.', 'match_path', '("/Users")', 21, r)}${hit('r.', 'match_path', '("/users/")', 29, r)}</div>`; };

  /** 变更行。acts:悬停时出现的行内操作;g:状态字母(M 修改 / A 新增 / D 删除 / C 冲突) */
  const GC = { A: 'c-ok', M: 'c-warn', D: 'c-err', C: 'c-err' };
  const ch = (name, dir, g, plus, minus, on, acts = '', open = '') => `<div class="row ${on ? 'on' : ''}" ${open ? `data-act="open" data-id="${open}"` : ''}>${ic('code', 's12')}<span class="ell">${name}</span><span class="t4 fs-micro ell grow">${dir}</span><span class="acts">${acts}</span>${plus ? `<span class="c-ok mono fs-micro swap">+${plus}</span>` : ''}${minus ? `<span class="c-err mono fs-micro swap">−${minus}</span>` : ''}<span class="m fs-micro ${GC[g] || 'c-warn'}" style="width:10px;text-align:center">${g}</span></div>`;
  /** Git 图谱:每行 28px 的泳道画布 + 提交说明 + 引用标签。lanes:该行经过的泳道 / 节点所在泳道 / 分叉与合并 */
  const COMMITS = [
    { m: 'fix(router): 路径匹配忽略大小写与末尾斜杠', a: '你', t: '刚刚', refs: ['HEAD', 'main'], n: 0, ln: [0] },
    { m: 'test: 补充 router 的大小写用例', a: '你', t: '2 小时', n: 0, ln: [0, 1], fork: 1 },
    { m: 'feat(auth): 支持 Bearer 令牌', a: 'Lin', t: '昨天', refs: ['feat/auth'], n: 1, ln: [0, 1] },
    { m: 'refactor: 抽出 handlers 模块', a: 'Lin', t: '昨天', n: 1, ln: [0, 1], join: 1 },
    { m: 'chore: 升级 axum 到 0.8', a: '你', t: '周一', refs: ['origin/main'], n: 0, ln: [0] },
    { m: 'docs: 补充路由匹配说明', a: 'Wen', t: '11月8日', n: 0, ln: [0] },
  ];
  const lane = (c) => { const x = (i) => 9 + i * 12; return `<svg class="gl" width="30" height="28" viewBox="0 0 30 28">${c.ln.map(i => `<path d="M${x(i)} 0V28" class="l${i}"/>`).join('')}${c.fork ? `<path d="M${x(0)} 14C${x(0)} 22 ${x(1)} 18 ${x(1)} 28" class="l1"/>` : ''}${c.join ? `<path d="M${x(1)} 14C${x(1)} 22 ${x(0)} 18 ${x(0)} 28" class="l1"/>` : ''}<circle cx="${x(c.n)}" cy="14" r="4" class="n l${c.n}"/></svg>`; };
  IDE.graph = () => `<div class="grp-h">${ic('chevD', 's12')}<span class="grow">图形</span>${ib('filter', 'sm', '仅当前分支')}${ib('refresh', 'sm')}</div>
    <div class="graph">${COMMITS.map((c, k) => `<div class="row gr ${k === 0 ? 'on' : ''}">${lane(c)}<span class="ell grow">${c.m}</span>${(c.refs || []).map(r => `<span class="tag ${r === 'HEAD' ? 'ac' : r.startsWith('origin') ? 'info' : ''}">${r}</span>`).join('')}<span class="aux">${c.t}</span></div>`).join('')}
      <div class="gr-detail"><div class="t3 fs-cap">a3f9c21 · 你 · 刚刚</div>${ch('router.rs', 'src', 'M', 4, 1)}${ch('router_test.rs', 'tests', 'A', 12, 0)}</div></div>`;
  /** 源代码管理:提交框 + 暂存 / 更改。o.graph:展开图形分区 */
  const gh = (open, t, n, acts = '', cls = '') => `<div class="grp-h ${cls}">${ic(open ? 'chevD' : 'chevR', 's12')}<span class="grow">${t}</span><span class="acts f g1">${acts}</span>${cnt(n)}</div>`;
  /** 源代码管理(对标 VS Code):合并冲突 / 暂存 / 更改 / 储藏 / 图形;文件行悬停才出现暂存、放弃、打开文件 */
  IDE.scm = (o = {}) => `<div class="f ai g2 t3 fs-cap" style="padding:0 6px 6px">${chip('main', 'branch', 'line', 'data-act="noop"')}<span class="grow"></span>${ib('list', 'sm', '切换列表 / 树')}<span class="rel">${ib('refresh', 'sm', '同步更改 ↑1 ↓0')}</span>${ib('download', 'sm', '拉取')}${ib('more', 'sm', '更多:推送、储藏、拉取(变基)…')}</div>
    <div class="commit-box">fix(router): 路径匹配忽略大小写与末尾斜杠<span class="wand">${ib('spark', 'sm tip-u', '生成提交说明')}</span></div>
    <div class="f g1" style="padding:0 2px 8px;position:relative">${btn('提交', 'pri grow', 'check', 'style="justify-content:center;flex:1;border-radius:var(--r-sm) 0 0 var(--r-sm)"')}${ib('chevD', '', '提交并推送 / 同步 · 修改上次提交', 'data-act="cmenu" style="background:var(--ac);color:var(--on-ac);border-radius:0 var(--r-sm) var(--r-sm) 0;width:24px"')}
      ${o.cmenu ? `<div class="pop a-pop-up" style="position:absolute;right:2px;top:28px;z-index:10;width:210px"><div class="mi on">${ic('check', 's12')}<span class="grow">提交</span></div><div class="mi">${ic('upload', 's12')}<span class="grow">提交并推送</span></div><div class="mi">${ic('refresh', 's12')}<span class="grow">提交并同步</span></div><div class="sep"></div><div class="mi">${ic('edit', 's12')}<span class="grow">修改上次提交</span></div><div class="mi">${ic('undo', 's12')}<span class="grow">撤销上次提交</span></div></div>` : ''}</div>
    <div class="stagger">${gh(1, '合并冲突', 2, ib('check', 'sm', '全部标记为已解决'), 'c-err')}${ch('router.rs', 'src', 'C', 0, 0, 0, ib('merge', 'sm', '解决冲突'), 'conflict')}${ch('mod.rs', 'src/handlers', 'C')}
      ${gh(1, '暂存的更改', 1, ib('minus', 'sm', '全部取消暂存'))}${ch('router.rs', 'src', 'M', 4, 2, 1, ib('ext', 'sm', '打开文件') + ib('minus', 'sm', '取消暂存'), 'diff:router.rs')}
      ${gh(1, '更改', 2, ib('undo', 'sm', '全部放弃') + ib('plus', 'sm', '全部暂存'))}${ch('router_test.rs', 'tests', 'M', 12, 0, 0, ib('ext', 'sm', '打开文件') + ib('undo', 'sm', '放弃更改') + ib('plus', 'sm', '暂存'))}${ch('case_test.rs', 'tests', 'A', 2, 0, 0, ib('ext', 'sm', '打开文件') + ib('undo', 'sm', '放弃更改') + ib('plus', 'sm', '暂存'))}
      ${gh(0, '储藏', 1, ib('plus', 'sm', '储藏当前更改'))}
      ${o.graph ? '' : `<div class="grp-h" style="margin-top:6px">${ic('chevR', 's12')}图形</div>`}</div>${o.graph ? IDE.graph() : ''}`;
  IDE.changeRow = ch;

  window.IDE = IDE;
})();
