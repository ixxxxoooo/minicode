/* 设计稿 v3 · IDE 部件:编辑器、底部面板、文件树、搜索、源代码管理(导航栏与检查器也复用这里的树与终端)
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
  const CUR = 17, GIT = { 11: 'gm', 17: 'ga', 18: 'ga', 22: 'ga', 23: 'ga', 24: 'ga' };

  const etab = (name, o = {}) => `<div class="etab ${o.on ? 'on' : ''} ${o.pv ? 'pv' : ''}">${ic(o.icon || 'code', 's12')}<span class="nm">${name}</span>${o.mod ? '<span class="mod"></span>' : `<span class="x">${ic('x', 's12')}</span>`}</div>`;
  /** 搜索命中高亮(只替换文字,不碰标签属性) */
  const hl = (s) => s.replace(/match_path/g, '<mark class="hit">match_path</mark>');
  /** 差异正文的行:o = 旧侧 [行号, 文本] / n = 新侧;k:add / del / chg(同一行改动)/ 空 = 未改 */
  const DV = [
    { fold: 34 },
    { k: '', o: [35, '    }'], n: [35, '    }'] }, { k: '', o: [36, ''], n: [36, ''] },
    { k: '', o: [37, '    /// 匹配路径:忽略大小写与末尾斜杠'], n: [37, '    /// 匹配路径:忽略大小写与末尾斜杠'] },
    { k: '', o: [38, '    pub fn match_path(&self, path: &str) -> Option<&Handler> {'], n: [38, '    pub fn match_path(&self, path: &str) -> Option<&Handler> {'] },
    { k: 'chg', o: [39, '        self.routes.get(path)'], n: [39, '        let key = normalize(path);'] },
    { k: 'add', n: [40, '        self.routes.get(&key).or(self.fallback.as_ref())'] },
    { k: '', o: [40, '    }'], n: [41, '    }'] }, { k: '', o: [41, '}'], n: [42, '}'] },
    { fold: 27 },
  ];
  const dvCell = (side, r) => {
    const v = r[side], bad = r.k === 'chg' ? (side === 'o' ? 'del' : 'add') : r.k === 'add' && side === 'n' ? 'add' : r.k === 'del' && side === 'o' ? 'del' : '';
    return v ? `<div class="dvc ${bad}"><span class="no">${v[0]}</span><span class="tx">${rust(v[1])}</span></div>` : '<div class="dvc e"></div>';
  };
  /** 差异标签:左右对比(HEAD ↔ 工作区),折叠未改动的行 */
  IDE.diffView = () => `<div class="dv-bar">${ic('diff', 's12')}<b>router.rs</b><span class="t3">HEAD ↔ 工作区</span>${tag('+4 −1', 'ok')}<span class="grow"></span>
      ${ib('chevU', 'sm', '上一处差异')}${ib('chevD', 'sm', '下一处差异')}${U.seg(['左右', '统一'], 0)}${btn('打开文件', 'sm ghost', 'ext')}</div>
    <div class="dv-wrap"><div class="dv"><div class="dv-head"><span>HEAD</span><span>工作区</span></div>${DV.map(r => r.fold ? `<div class="dv-fold">${ic('more', 's12')}已折叠 ${r.fold} 行未更改 · 点击展开</div>` : `<div class="dvr">${dvCell('o', r)}${dvCell('n', r)}</div>`).join('')}</div>
      <div class="mm dvm"><div class="vp" style="height:90px"></div>${DV.concat(DV, DV).map(r => `<i style="width:${r.fold ? 100 : 55}%;${r.k === 'add' || r.k === 'chg' ? 'background:var(--ok);opacity:.6' : r.fold ? 'opacity:.12' : ''}"></i>`).join('')}</div></div>`;

  /** 编辑区:标签 + 面包屑 + 代码 + 迷你地图 + 底部面板。o.diff:显示差异标签;o.hl:高亮搜索命中;o.panel:底部面板分段 */
  IDE.editor = (o = {}) => `
    <div class="etabs">${etab('router.rs', { on: !o.diff })}${etab('router_test.rs', { mod: 1 })}${etab('Cargo.toml', { icon: 'gear', pv: 1 })}${etab('router.rs (HEAD ↔ 工作区)', { icon: 'diff', on: o.diff })}
      <div class="etab-tools">${ib('cols', 'sm', '向右拆分')}${ib('more', 'sm')}</div></div>
    ${o.diff ? IDE.diffView() : `<div class="crumbs">src${ic('chevR', 's12')}<b>router.rs</b>${ic('chevR', 's12')}${ic('cube', 's12')}impl Router${ic('chevR', 's12')}<b>match_path</b></div>
    <div class="code">
      <div class="gut">${SRC.map((_, k) => `<div class="${k + 1 === CUR ? 'cur' : ''} ${GIT[k + 1] || ''}">${k + 1}</div>`).join('')}</div>
      <div class="src">${SRC.map((l, k) => `<div class="${k + 1 === CUR ? 'cur' : ''}">${o.hl ? hl(rust(l)) : k + 1 === 13 ? rust(l).replace('insert', '<span class="sqw">insert</span>') : rust(l)}${k + 1 === CUR && !o.hl ? '<span class="cursor"></span><span class="blame">你 · 2 分钟前 · 未提交</span>' : ''}</div>`).join('')}</div>
      <div class="mm"><div class="vp"></div>${SRC.concat(SRC).map(l => `<i style="width:${Math.min(100, l.length * 2.1)}%;${l.trim() ? '' : 'opacity:0'}"></i>`).join('')}</div>
    </div>`}
    ${IDE.bpanel(o.panel)}`;

  IDE.termLines = () => `<span class="p">➜</span> <b>router-demo</b> <span class="d">git:(</span><span class="e">main</span><span class="d">)</span> cargo test router
<span class="d">    Finished</span> test profile in 1.92s
test matches_ignoring_case ... <span class="p">ok</span>
test matches_trailing_slash ... <span class="p">ok</span>
test result: <span class="p">ok</span>. 12 passed; 0 failed
<span class="p">➜</span> <b>router-demo</b> <span class="d">git:(</span><span class="e">main</span><span class="d">)</span> <span class="caret"></span>`;
  const prob = (lv, msg, src, ln) => `<div class="row">${ic(lv === 'err' ? 'errc' : 'alert', 's12 ' + (lv === 'err' ? 'c-err' : 'c-warn'))}<span class="ell">${msg}</span><span class="t4 fs-cap nw">${src}</span><span class="aux">${ln}</span></div>`;
  /** 底部面板:终端 / 问题 / 输出。which:term(默认)/ problems */
  IDE.bpanel = (which = 'term') => `<div class="bpanel ${which === 'problems' ? 'tall' : ''}">
    <div class="tabs"><span class="tab ${which === 'term' ? 'on' : ''}" data-act="tab">${ic('terminal', 's12')}终端</span><span class="tab ${which === 'problems' ? 'on' : ''}" data-act="tab">问题${cnt(3, which === 'problems' ? 'ac' : '')}</span><span class="tab" data-act="tab">输出</span><span class="grow"></span>
      <span class="f ai g1">${which === 'problems' ? `${chip('全部', 'filter', 'on')}${ib('refresh', 'sm', '重新检查')}` : chip('zsh', 'prompt', 'on') + ib('plus', 'sm', '新建终端') + ib('cols', 'sm')}${ib('x', 'sm')}</span></div>
    ${which === 'problems' ? `<div class="grow scroll" style="padding:2px 6px"><div class="grp-h">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow">router.rs</span>${cnt(2)}</div>${prob('warn', '`fallback` 字段从未被读取', 'rust-analyzer', '行 5')}${prob('warn', '未使用的变量:`h`', 'rust-analyzer', '行 11')}
      <div class="grp-h">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow">router_test.rs</span>${cnt(1)}</div>${prob('err', '找不到函数 `normalize`,需要 `use crate::router::normalize`', 'rust-analyzer', '行 8')}</div>`
      : `<div class="term grow">${IDE.termLines()}</div>`}</div>`;

  const tr = (d, name, o = {}) => `<div class="row tr ${o.dir ? 'dir' : ''} ${o.on ? 'on' : ''} ${o.g ? 'g' + o.g.toLowerCase() : ''}" style="--d:${d}">${o.dir ? ic(o.open ? 'chevD' : 'chevR', 's12 car') : '<span style="width:12px"></span>'}${ic(o.dir ? 'folder' : o.icon || 'fileText', 's12')}<span class="ell">${name}</span>${o.g ? `<span class="g">${o.g}</span>` : ''}</div>`;
  /** 文件树。head:是否带区块标题 */
  IDE.tree = (head = true) => `${head ? `<div class="grp-h">${ic('chevD', 's12')}<span class="grow">ROUTER-DEMO</span>${ib('plus', 'sm', '新建文件')}${ib('refresh', 'sm', '刷新')}</div>` : ''}
    <div class="stagger">${tr(0, 'src', { dir: 1, open: 1 })}${tr(1, 'handlers', { dir: 1 })}${tr(1, 'main.rs', { icon: 'code' })}${tr(1, 'router.rs', { icon: 'code', on: 1, g: 'M' })}${tr(1, 'server.rs', { icon: 'code' })}
    ${tr(0, 'tests', { dir: 1, open: 1 })}${tr(1, 'router_test.rs', { icon: 'code', g: 'M' })}${tr(1, 'case_test.rs', { icon: 'code', g: 'A' })}
    ${tr(0, 'docs', { dir: 1 })}${tr(0, 'Cargo.toml', { icon: 'gear' })}${tr(0, 'CLAUDE.md')}${tr(0, 'README.md')}</div>
    ${head ? `<div class="sep" style="margin:8px 2px"></div><div class="grp-h">${ic('chevR', 's12')}大纲</div><div class="grp-h">${ic('chevR', 's12')}时间线</div>` : ''}`;

  /** 搜索命中行。rep:替换预览(红色删除线旧文 + 绿色新文) */
  const hit = (pre, m, post, ln, rep) => `<div class="row hitrow" style="padding-left:26px"><span class="ell mono">${pre}${rep ? `<del>${m}</del><ins>${rep}</ins>` : `<mark class="hit">${m}</mark>`}${post}</span><span class="acts">${ib('x', 'sm', '忽略此处')}</span><span class="aux swap">${ln}</span></div>`;
  const tgl = (t, on, tip) => `<span class="tgl ${on ? 'on' : ''}" ${tip ? `data-tip="${tip}"` : ''}>${t}</span>`;
  /** 项目内搜索。o.rep:展开替换行(并在命中行显示替换预览) */
  IDE.search = (o = {}) => { const r = o.rep ? 'find_route' : '';
    return `<div class="fc g2" style="padding:0 2px 6px"><div class="f g1 ai"><span class="t3" style="width:14px">${ic(o.rep ? 'chevD' : 'chevR', 's12')}</span><div class="inp focus grow">${ic('search', 's12')}<span class="v grow">match_path</span>${tgl('Aa', 0, '区分大小写 ⌥⌘C')}${tgl('ab', 1, '全词匹配 ⌥⌘W')}${tgl('.*', 0, '正则 ⌥⌘R')}</div></div>
      ${o.rep ? `<div class="f g1 ai"><span style="width:14px"></span><div class="inp grow">${ic('swap', 's12')}<span class="v grow">find_route</span>${tgl('AB', 0, '保留大小写 ⌥⌘P')}${ib('check', 'sm', '全部替换 ⌥⌘↩')}</div></div>` : ''}
      <div class="f ai g2 t3 fs-cap" style="padding-left:16px">${tgl('···', 0, '要包含 / 排除的文件')}<span class="grow"></span>${o.rep ? '<span>仅替换列出的匹配</span>' : ''}</div></div>
    <div class="t3 fs-cap f ai" style="padding:0 8px 4px"><span class="grow">2 个文件中 4 处结果</span>${ib('refresh', 'sm', '重新搜索')}${ib('x', 'sm', '清除结果')}</div>
    <div class="stagger"><div class="row proj">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow ell">router.rs</span>${cnt(2)}</div>${hit('pub fn ', 'match_path', '(&self', 18, r)}${hit('// ', 'match_path', ' 忽略', 16, r)}
    <div class="row proj">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow ell">router_test.rs</span>${cnt(2)}</div>${hit('r.', 'match_path', '("/Users")', 21, r)}${hit('r.', 'match_path', '("/users/")', 29, r)}</div>`; };

  const ch = (name, dir, g, plus, minus, on) => `<div class="row ${on ? 'on' : ''}">${ic('code', 's12')}<span class="ell">${name}</span><span class="t4 fs-micro ell grow">${dir}</span>${plus ? `<span class="c-ok mono fs-micro">+${plus}</span>` : ''}${minus ? `<span class="c-err mono fs-micro">−${minus}</span>` : ''}<span class="m fs-micro ${g === 'A' ? 'c-ok' : 'c-warn'}" style="width:10px;text-align:center">${g}</span></div>`;
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
  IDE.scm = (o = {}) => `<div class="f ai g2 t3 fs-cap" style="padding:0 6px 6px">${chip('main', 'branch', 'line', 'data-act="noop"')}<span class="grow"></span>${ib('refresh', 'sm', '同步更改 ↑1 ↓0')}${ib('download', 'sm', '拉取')}${ib('more', 'sm')}</div>
    <div class="commit-box">fix(router): 路径匹配忽略大小写与末尾斜杠<span class="wand">${ib('spark', 'sm tip-u', '生成提交说明')}</span></div>
    <div class="f g2" style="padding:0 2px 8px">${btn('提交', 'pri grow', 'check', 'style="justify-content:center;flex:1"')}${ib('chevD', '', '更多提交方式')}</div>
    <div class="stagger"><div class="grp-h">${ic('chevD', 's12')}<span class="grow">暂存的更改</span>${cnt(1)}</div>${ch('router.rs', 'src', 'M', 4, 1, 1)}
    <div class="grp-h">${ic('chevD', 's12')}<span class="grow">更改</span>${cnt(2)}</div>${ch('router_test.rs', 'tests', 'M', 12, 0)}${ch('case_test.rs', 'tests', 'A', 2, 0)}
    ${o.graph ? '' : `<div class="grp-h" style="margin-top:6px">${ic('chevR', 's12')}图形</div>`}</div>${o.graph ? IDE.graph() : ''}`;
  IDE.changeRow = ch;

  window.IDE = IDE;
})();
