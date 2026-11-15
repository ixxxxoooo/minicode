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
  /** 编辑区:标签 + 面包屑 + 代码 + 迷你地图 + 底部面板 */
  IDE.editor = () => `
    <div class="etabs">${etab('router.rs', { on: 1 })}${etab('router_test.rs', { mod: 1 })}${etab('Cargo.toml', { icon: 'gear', pv: 1 })}${etab('router.rs (HEAD ↔ 工作区)', { icon: 'diff' })}
      <div class="etab-tools">${ib('cols', 'sm', '向右拆分')}${ib('more', 'sm')}</div></div>
    <div class="crumbs">src${ic('chevR', 's12')}<b>router.rs</b>${ic('chevR', 's12')}${ic('cube', 's12')}impl Router${ic('chevR', 's12')}<b>match_path</b></div>
    <div class="code">
      <div class="gut">${SRC.map((_, k) => `<div class="${k + 1 === CUR ? 'cur' : ''} ${GIT[k + 1] || ''}">${k + 1}</div>`).join('')}</div>
      <div class="src">${SRC.map((l, k) => `<div class="${k + 1 === CUR ? 'cur' : ''}">${k + 1 === 13 ? rust(l).replace('insert', '<span class="sqw">insert</span>') : rust(l)}${k + 1 === CUR ? '<span class="cursor"></span><span class="blame">你 · 2 分钟前 · 未提交</span>' : ''}</div>`).join('')}</div>
      <div class="mm"><div class="vp"></div>${SRC.concat(SRC).map(l => `<i style="width:${Math.min(100, l.length * 2.1)}%;${l.trim() ? '' : 'opacity:0'}"></i>`).join('')}</div>
    </div>
    ${IDE.bpanel()}`;

  IDE.termLines = () => `<span class="p">➜</span> <b>router-demo</b> <span class="d">git:(</span><span class="e">main</span><span class="d">)</span> cargo test router
<span class="d">    Finished</span> test profile in 1.92s
test matches_ignoring_case ... <span class="p">ok</span>
test matches_trailing_slash ... <span class="p">ok</span>
test result: <span class="p">ok</span>. 12 passed; 0 failed
<span class="p">➜</span> <b>router-demo</b> <span class="d">git:(</span><span class="e">main</span><span class="d">)</span> <span class="caret"></span>`;
  /** 底部面板:终端 / 问题 / 输出 */
  IDE.bpanel = () => `<div class="bpanel">
    <div class="tabs"><span class="tab on" data-act="tab">${ic('terminal', 's12')}终端</span><span class="tab" data-act="tab">问题${cnt(1)}</span><span class="tab" data-act="tab">输出</span><span class="grow"></span>
      <span class="f ai g1">${chip('zsh', 'prompt', 'on')}${ib('plus', 'sm', '新建终端')}${ib('cols', 'sm')}${ib('x', 'sm')}</span></div>
    <div class="term grow">${IDE.termLines()}</div></div>`;

  const tr = (d, name, o = {}) => `<div class="row tr ${o.dir ? 'dir' : ''} ${o.on ? 'on' : ''} ${o.g ? 'g' + o.g.toLowerCase() : ''}" style="--d:${d}">${o.dir ? ic(o.open ? 'chevD' : 'chevR', 's12 car') : '<span style="width:12px"></span>'}${ic(o.dir ? 'folder' : o.icon || 'fileText', 's12')}<span class="ell">${name}</span>${o.g ? `<span class="g">${o.g}</span>` : ''}</div>`;
  /** 文件树。head:是否带区块标题 */
  IDE.tree = (head = true) => `${head ? `<div class="grp-h">${ic('chevD', 's12')}<span class="grow">ROUTER-DEMO</span>${ib('plus', 'sm', '新建文件')}${ib('refresh', 'sm', '刷新')}</div>` : ''}
    <div class="stagger">${tr(0, 'src', { dir: 1, open: 1 })}${tr(1, 'handlers', { dir: 1 })}${tr(1, 'main.rs', { icon: 'code' })}${tr(1, 'router.rs', { icon: 'code', on: 1, g: 'M' })}${tr(1, 'server.rs', { icon: 'code' })}
    ${tr(0, 'tests', { dir: 1, open: 1 })}${tr(1, 'router_test.rs', { icon: 'code', g: 'M' })}${tr(1, 'case_test.rs', { icon: 'code', g: 'A' })}
    ${tr(0, 'docs', { dir: 1 })}${tr(0, 'Cargo.toml', { icon: 'gear' })}${tr(0, 'CLAUDE.md')}${tr(0, 'README.md')}</div>
    ${head ? `<div class="sep" style="margin:8px 2px"></div><div class="grp-h">${ic('chevR', 's12')}大纲</div><div class="grp-h">${ic('chevR', 's12')}时间线</div>` : ''}`;

  const hit = (pre, m, post, ln) => `<div class="row" style="padding-left:26px"><span class="ell mono">${pre}<span style="background:var(--warn-soft);color:var(--tx-1);border-radius:2px">${m}</span>${post}</span><span class="aux">${ln}</span></div>`;
  /** 项目内搜索 */
  IDE.search = () => `<div class="fc g2" style="padding:0 2px 8px"><div class="inp focus">${ic('search', 's12')}<span class="v grow">match_path</span><span class="tag">Aa</span><span class="tag">.*</span></div><div class="inp">${ic('swap', 's12')}<span class="grow">替换</span></div></div>
    <div class="t3 fs-cap" style="padding:0 8px 4px">2 个文件中 5 处结果</div>
    <div class="stagger"><div class="row proj">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow ell">router.rs</span>${cnt(3)}</div>${hit('pub fn ', 'match_path', '(&self', 18)}${hit('// ', 'match_path', ' 忽略', 16)}
    <div class="row proj">${ic('chevD', 's12')}${ic('code', 's12')}<span class="grow ell">router_test.rs</span>${cnt(2)}</div>${hit('r.', 'match_path', '("/Users")', 21)}${hit('r.', 'match_path', '("/users/")', 29)}</div>`;

  const ch = (name, dir, g, plus, minus, on) => `<div class="row ${on ? 'on' : ''}">${ic('code', 's12')}<span class="ell">${name}</span><span class="t4 fs-micro ell grow">${dir}</span>${plus ? `<span class="c-ok mono fs-micro">+${plus}</span>` : ''}${minus ? `<span class="c-err mono fs-micro">−${minus}</span>` : ''}<span class="m fs-micro ${g === 'A' ? 'c-ok' : 'c-warn'}" style="width:10px;text-align:center">${g}</span></div>`;
  /** 源代码管理:提交框 + 暂存 / 更改 */
  IDE.scm = () => `<div class="commit-box">fix(router): 路径匹配忽略大小写与末尾斜杠<span class="wand">${ib('spark', 'sm tip-u', '生成提交说明')}</span></div>
    <div class="f g2" style="padding:0 2px 8px">${btn('提交', 'pri grow', 'check', 'style="justify-content:center;flex:1"')}${ib('chevD', '', '更多提交方式')}</div>
    <div class="stagger"><div class="grp-h">${ic('chevD', 's12')}<span class="grow">暂存的更改</span>${cnt(1)}</div>${ch('router.rs', 'src', 'M', 4, 1, 1)}
    <div class="grp-h">${ic('chevD', 's12')}<span class="grow">更改</span>${cnt(2)}</div>${ch('router_test.rs', 'tests', 'M', 12, 0)}${ch('case_test.rs', 'tests', 'A', 2, 0)}
    <div class="grp-h" style="margin-top:6px">${ic('chevR', 's12')}图形</div></div>`;
  IDE.changeRow = ch;

  window.IDE = IDE;
})();
