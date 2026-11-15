/* 设计稿 v3.3 · 项目与工作区、多仓库 Git、右键菜单与动作矩阵、缺口总表
   依据:VS Code 多根工作区(.code-workspace:folders[] + 名称覆盖;Explorer 每个根一个顶层节点;SCM 有 Repositories 视图,
   可多选仓库,也可切到单仓库模式)、Claude Code(从主目录启动,其余目录用 --add-dir / additionalDirectories 授权;
   附加目录的 CLAUDE.md 默认不加载,需环境变量开启)、Zed / JetBrains 的项目分组。
   每页都带「实现状态」列,与 Go 代码保持一致。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog, chk } = U;
  const { head, sec } = P.h;
  const tbl = (cols, rows) => `<table class="ia-tbl"><thead><tr>${cols.map(([t, w]) => `<th ${w ? `style="width:${w}px"` : ''}>${t}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map((c, k) => `<td ${k === 0 ? '' : 'class="t2"'}>${k === 0 ? `<b>${c}</b>` : c}</td>`).join('')}</tr>`).join('')}</tbody></table>`;
  const st = (s) => ({ ok: tag('已实现', 'ok'), part: tag('部分', 'warn'), todo: tag('待做', '') }[s]);
  const grid = (...cells) => `<div class="gal">${cells.join('')}</div>`;
  const cell = (t, keys, body, why) => `<div><div class="cap2"><b>${t}</b>${keys ? kbd(...keys) : ''}</div>${body}<div class="why">${why}</div></div>`;
  const box = (body, h = 220, w = 560, extra = '') => `<div class="snip" style="height:${h}px;width:${w}px;${extra}">${body}</div>`;
  const gh = (t, n, right = '') => `<div class="grp-h">${ic('chevD', 's12')}<span class="grow">${t}</span>${right}${n === undefined ? '' : cnt(n)}</div>`;
  const mi = (i, t, k, cls = '') => i === '-' ? '<div class="sep"></div>' : `<div class="mi ${cls}">${i ? ic(i, 's12') : '<span style="width:12px"></span>'}<span class="grow">${t}</span>${k ? kbd(k) : ''}</div>`;
  const menu = (w, items) => `<div class="pop" style="width:${w}px;padding:4px">${items.map(([i, t, k, c]) => mi(i, t, k, c)).join('')}</div>`;
  const mono = (s) => `<span class="mono">${s}</span>`;

  /* ───────── 项目与工作区 ───────── */
  const side = `<div style="padding:8px">
    <div class="row on" style="height:28px">${ic('plus', 's12', 'c-ac')}<span class="grow b">新对话</span>${kbd('⌘', 'N')}</div>
    <div class="grp-h" style="margin-top:6px"><span class="grow">项目与工作区</span>${ib('plus', 'sm', '添加项目或工作区')}</div>
    <div class="row" style="height:28px">${ic('chevD', 's12')}${ic('layers', 's12', 'c-ac')}<span class="grow b">全栈工作区</span><span class="t4 fs-micro">3 个文件夹</span></div>
    <div class="row" style="height:22px;padding-left:34px;color:var(--tx-3)">${ic('folder', 's12')}<span class="grow ell">router-demo</span>${tag('主', 'ac')}</div>
    <div class="row" style="height:22px;padding-left:34px;color:var(--tx-3)">${ic('folder', 's12')}<span class="grow ell">web-dashboard</span></div>
    <div class="row" style="height:22px;padding-left:34px;color:var(--tx-3)">${ic('folder', 's12')}<span class="grow ell">docs-site</span></div>
    <div class="row" style="height:28px;padding-left:25px">完整修复一个 Bug:路由大小写<span class="aux">刚刚</span></div>
    <div class="row" style="height:28px;padding-left:25px">权限请求:修改配置…<span class="aux">5 分钟</span></div>
    <div class="row" style="height:28px">${ic('chevR', 's12')}${ic('folder', 's12')}<span class="grow b">scenario-lab</span><span class="t4 fs-micro">61</span></div>
    <div class="row" style="height:28px">${ic('chevR', 's12')}${ic('folder', 's12')}<span class="grow b">design-assets</span>${tag('非仓库', '')}</div></div>`;
  const field = (l, v, ph, extra = '') => `<div class="fc g1" style="margin-bottom:8px"><div class="t3 fs-cap">${l}</div><div class="inp ${v ? '' : 'fill'}" ${extra}>${v ? `<span class="v">${v}</span>` : `<span class="t4">${ph}</span>`}</div></div>`;
  const dlgAddProject = `<div class="pop a-pop-up" style="width:460px;padding:14px 16px;margin:14px auto;display:flex;flex-direction:column;gap:6px">
    <div class="f ai"><b class="fs-head">添加项目</b><span class="grow"></span>${ib('x', 'sm')}</div>
    <div class="t3 fs-cap">项目就是一个文件夹,Claude 会话在这个目录里运行。</div>
    <div class="f g2"><div class="inp grow focus"><span class="v mono caret">~/Code/api-gateway</span></div>${btn('浏览…', 'sm', 'folder')}</div>
    <div class="t3 fs-cap" style="margin-top:6px">最近 · 发现的项目</div>
    ${[['api-gateway', '~/Code/api-gateway', 'branch', '是 Git 仓库 · main', 1], ['mobile-app', '~/Code/mobile-app', 'branch', '是 Git 仓库 · feat/login', 0], ['notes', '~/Documents/notes', 'folder', '不是 Git 仓库', 0]].map(([n, p, i, d, on]) => `<div class="row ${on ? 'on' : ''}" style="height:36px">${ic(i, 's12')}<div class="grow" style="min-width:0"><div class="ell">${n}</div><div class="t4 fs-micro mono ell">${p}</div></div><span class="t3 fs-cap">${d}</span></div>`).join('')}
    <div class="f g2" style="margin-top:8px"><span class="t3 fs-cap grow">已在列表里的项目会置灰并显示「已添加」</span>${btn('取消', 'ghost')}${btn('添加', 'pri')}</div></div>`;
  const wrow = (n, p, primary, repo) => `<div class="row" style="height:34px">${chk('done')}${ic('folder', 's12')}<div class="grow" style="min-width:0"><div class="ell">${n} ${primary ? tag('主文件夹', 'ac') : ''}</div><div class="t4 fs-micro mono ell">${p}</div></div><span class="t3 fs-cap">${repo}</span>${ib('arrow', 'sm', '设为主文件夹')}${ib('x', 'sm', '移出')}</div>`;
  const dlgAddWs = `<div class="pop a-pop-up" style="width:500px;padding:14px 16px;margin:14px auto;display:flex;flex-direction:column;gap:6px">
    <div class="f ai"><b class="fs-head">新建工作区</b><span class="grow"></span>${ib('x', 'sm')}</div>
    <div class="t3 fs-cap">工作区 = 一组相关的文件夹。会话在「主文件夹」里运行,其余文件夹同时对 Claude 开放读写。</div>
    ${field('名称', '全栈工作区')}
    <div class="t3 fs-cap">文件夹(2–12 个)</div>
    ${wrow('router-demo', '~/Code/router-demo', 1, 'main · 3 项改动')}${wrow('web-dashboard', '~/Code/web-dashboard', 0, 'feat/charts · 2 项改动')}${wrow('docs-site', '~/Code/docs-site', 0, 'main · 干净')}
    <div class="f g2"><div class="inp grow"><span class="t4">粘贴文件夹路径后回车添加</span></div>${btn('浏览…', '', 'folder')}</div>
    <div class="f g2">${btn('从已有项目添加', 'sm', 'plus')}</div>
    <div class="f ai g3" style="margin-top:6px">${tog(1)}<span class="t2">让 Claude 同时读取附加文件夹里的 CLAUDE.md 与规则</span></div>
    <div class="f g2" style="margin-top:8px"><span class="t3 fs-cap grow">保存到 ${mono('~/.minicode/workspaces.json')},不会往项目里写文件</span>${btn('取消', 'ghost')}${btn('创建工作区', 'pri')}</div></div>`;
  const tree = `<div style="padding:6px 8px">${gh('全栈工作区(工作区)', undefined, ib('plus', 'sm', '添加文件夹'))}
    <div class="row" style="height:24px;font-weight:600">${ic('chevD', 's12')}${ic('folder', 's12')}<span class="grow">router-demo</span>${tag('主', 'ac')}<span class="g gm">3</span></div>
    <div class="row" style="height:22px;padding-left:30px">${ic('chevR', 's12')}${ic('folder', 's12')}<span>src</span></div>
    <div class="row" style="height:22px;padding-left:30px">${ic('chevR', 's12')}${ic('folder', 's12')}<span>tests</span></div>
    <div class="row" style="height:24px;font-weight:600">${ic('chevD', 's12')}${ic('folder', 's12')}<span class="grow">web-dashboard</span><span class="g gm">2</span></div>
    <div class="row" style="height:22px;padding-left:30px">${ic('chevR', 's12')}${ic('folder', 's12')}<span>src</span></div>
    <div class="row" style="height:24px;font-weight:600">${ic('chevR', 's12')}${ic('folder', 's12')}<span class="grow">docs-site</span></div>
    </div>`;
  P['proj-ws'] = () => `<div class="ds" style="max-width:1240px">${head('项目与工作区 · 添加、组织、使用', '先分清三个词:<b>项目</b> = 一个文件夹(Claude 会话的工作目录);<b>工作区</b> = 一组命名的文件夹(一个主文件夹 + 若干附加文件夹),对标 VS Code 的多根工作区;<b>仓库</b> = 文件夹里的 Git 仓库(一个文件夹至多一个,也可以没有)。会话永远属于「主文件夹」,附加文件夹通过 Claude 的 <span class="mono">--add-dir</span> 授权,不是把它们拷进来。')}
    ${sec('侧栏怎么呈现', '项目与工作区在同一列表里,标题行「+」是唯一入口(菜单:添加项目… / 新建工作区… / 打开最近)。工作区行用叠层图标与「N 个文件夹」,展开后先是成员文件夹(小号、可点、可右键),再是会话。', grid(
      cell('侧栏(会话分栏)', '', box(side, 330, 300), '项目按最近活动排序;工作区置顶。非仓库的文件夹显示灰色「非仓库」标签,不隐藏。'),
      cell('文件树(多根)', '', box(tree, 330, 300), '一个工作区有 ≥ 2 个文件夹时,文件树每个文件夹一个加粗顶层节点(默认展开,右侧显示该仓库的改动数);标题写「<工作区名>(工作区)」,单项目时与现在一致。'),
      cell('搜索(多根)', '', box(`<div style="padding:10px 12px" class="fc g3"><div class="inp"><span class="v">normalize</span></div><div class="f g2">${chip('全部文件夹', 'folder', 'on')}${chip('仅 router-demo', 'folder', 'line')}</div>${gh('router-demo/src/router.rs', 3)}${gh('web-dashboard/src/api.ts', 1)}</div>`, 220, 300), '搜索默认覆盖所有文件夹,结果路径带文件夹名前缀;chip 可限定到某个文件夹。')))}
    ${sec('添加项目', '入口:侧栏「+」、⌘O(打开文件夹)、命令面板「添加项目」、文件菜单。系统文件夹选择器是主路径;输入框也能直接粘路径(带 ~ 展开与存在性校验)。', grid(
      cell('添加项目对话框', ['⌘', 'O'], box(dlgAddProject, 400), '「最近 · 发现」来自 ~/.claude/projects 里出现过的目录;选中后在下方显示是否仓库与当前分支。已添加的置灰。'),
      cell('校验与错误', '', box(`<div style="padding:16px" class="fc g3">${[['路径不存在', '~/Code/api-gatewy', '找不到这个文件夹。要不要看看 ~/Code/api-gateway?', 'err'], ['不是文件夹', '~/Code/readme.md', '这是一个文件。要打开它所在的文件夹吗?', 'warn'], ['已在列表里', '~/Code/router-demo', '这个项目已经添加,直接切换过去?', 'info']].map(([t, p, m, k]) => `<div class="notice ${k === 'info' ? '' : k}">${ic(k === 'err' ? 'errc' : k === 'warn' ? 'alert' : 'info', k === 'err' ? 'c-err' : k === 'warn' ? 'c-warn' : '')}<div class="grow"><b>${t}</b> <span class="mono t3">${p}</span><div class="t2">${m}</div></div></div>`).join('')}</div>`, 300), '每种失败都给「下一步」:纠正路径 / 打开所在文件夹 / 直接切换;不弹模态错误框。')))}
    ${sec('新建 / 编辑工作区', '同一个对话框:新建与「编辑文件夹…」共用。第一个文件夹是主文件夹(会话 cwd、终端、CLAUDE.md 加载处),可用箭头设为主;顺序即文件树顺序;最多 12 个。', grid(
      cell('新建工作区对话框', '', box(dlgAddWs, 470, 560), '名称默认取主文件夹名 + 「工作区」;「让 Claude 同时读取附加文件夹的 CLAUDE.md」对应环境变量 CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD,默认关。'),
      cell('权限提示', '', box(`<div style="padding:16px" class="fc g3"><div class="notice">${ic('shield')}<div class="grow"><b>Claude 将获得 2 个附加文件夹的读写权限</b><div class="t2">web-dashboard、docs-site。可在 设置 → 权限与信任 → 附加目录 里随时撤销;第一次启动会话时再确认一次。</div><div class="f g2" style="margin-top:7px">${btn('我知道了', 'sm pri')}${btn('只读授权', 'sm')}</div></div></div></div>`, 160), '授权是显式的:创建 / 启动时各提示一次,之后在设置里集中管理。')))}
    ${sec('右键菜单', '', grid(
      cell('工作区行', '', menu(230, [['plus', '新对话(主文件夹)', '⌘N'], ['-'], ['edit', '重命名…'], ['folder', '编辑文件夹…'], ['plus', '添加文件夹…'], ['-'], ['folder', '在访达中显示主文件夹'], ['copy', '复制主文件夹路径'], ['-'], ['trash', '从列表移除工作区', '', 'danger']]), '移除只删列表项,不碰磁盘上的任何文件;确认框写明「不会删除文件」。'),
      cell('成员文件夹行', '', menu(230, [['folder', '在文件树中定位'], ['arrow', '设为主文件夹'], ['-'], ['branch', '在源代码管理中查看'], ['folder', '在访达中显示'], ['copy', '复制路径'], ['-'], ['x', '从工作区移出', '', 'danger']]), '主文件夹不能直接移出,先把别的设为主。'),
      cell('项目行', '', menu(230, [['plus', '在此项目新对话'], ['layers', '加入工作区…'], ['edit', '重命名显示名'], ['-'], ['folder', '在访达中显示'], ['copy', '复制路径'], ['-'], ['archive', '归档该项目的会话'], ['trash', '从列表移除项目', '', 'danger']]), '项目被某个工作区引用时,移除前提示「它属于 1 个工作区」。')))}
    ${sec('拖拽排序', '工作区与工作区里的成员文件夹可以拖着换位置;顺序写进 ' + mono('~/.minicode/workspaces.json') + '(工作区列表顺序 / 工作区内 folders 顺序,文件树顺序跟着变)。项目行按最近活动排序,不手动排。', grid(
      cell('拖动中', '', box(`<div style="padding:8px"><div class="row" style="height:28px">${ic('chevD', 's12')}${ic('layers', 's12', 'c-ac')}<span class="grow b">全栈工作区</span><span class="t4 fs-micro">3 个文件夹</span></div>
        <div class="row" style="height:22px;padding-left:34px;color:var(--tx-3)">${ic('folder', 's12')}<span class="grow ell">router-demo</span>${tag('主', 'ac')}</div>
        <div class="row" style="height:22px;padding-left:34px;color:var(--tx-3);box-shadow:inset 0 2px 0 var(--ac)">${ic('folder', 's12')}<span class="grow ell">web-dashboard</span></div>
        <div class="row" style="height:22px;padding-left:34px;color:var(--tx-3);opacity:.5">${ic('folder', 's12')}<span class="grow ell">docs-site</span></div>
        <div class="pop" style="position:relative;width:170px;padding:4px 10px;margin:-30px 0 0 120px;opacity:.92">${ic('folder', 's12')} docs-site</div></div>`, 170, 330), '拖起的行变淡、指针旁跟一份副本;经过的行顶部出现 2px 强调色插入线,松手 = 放到那一行之前。拖到主文件夹(第一行)上不可放:插入线不出现,松手提示「第一个是主文件夹,用「设为主文件夹」更换」。'),
      cell('菜单与键盘', '', box(menu(190, [['arrow', '上移'], ['arrow', '下移'], ['-'], ['arrow', '设为主文件夹']]) + `<div class="t3 fs-cap" style="padding:8px 4px;line-height:1.7">· 工作区行与成员文件夹行右键都有「上移 / 下移」,处在边界时置灰<br>· 成员文件夹不能经「上移」越过主文件夹(第二个上移 = 设为主文件夹,要用专门的菜单项)<br>· 工作区正在使用时,文件夹换序后文件树即时跟着变</div>`, 170, 330), '拖拽是快捷方式,菜单是等价的无障碍路径,两者走同一个存储函数。')))}
    ${sec('Git 与工作区的关系', '工作区本身不是仓库。每个成员文件夹各自是(或不是)一个仓库,SCM、图谱、分支、同步全部按仓库独立工作 —— 详见「多仓库 Git」页。', '')}
    ${sec('实现状态', '', tbl([['能力', 220], ['说明', 440], ['状态']], [
      ['侧栏「+」入口与添加项目对话框', '路径输入 + 最近 / 发现 + 校验;真实模式浏览按钮走系统文件夹选择器;路径不存在时给出相近路径建议', st('ok')],
      ['工作区存储', '~/.minicode/workspaces.json 一个文件同时存手动添加的项目与工作区,原子落盘;损坏时改名留底不丢数据', st('ok')],
      ['侧栏的工作区行与成员行', '工作区行 / 成员行 / 项目行右键菜单已做;成员文件夹折进工作区;拖拽排序与上移 / 下移已做', st('ok')],
      ['多根文件树 / 多根搜索', '虚拟顶层目录 = 文件夹名(workspace.Multi),路径加文件夹前缀,顶层行带「主 / 改动数 / 非仓库」;搜索按前缀路径工作,搜索条有「范围」chip 与文件夹右键「仅在此文件夹搜索」', st('ok')],
      ['Claude 附加目录', '主文件夹启动会话时追加 --add-dir(附加文件夹),开关打开时设置 CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1', st('ok')],
      ['演示数据', '演示模式内置「全栈工作区」(router-demo / web-dashboard / docs-site 各有自己的分支与改动)+ 一个手动添加的非仓库项目 design-assets', st('ok')]]))}</div>`;

  /* ───────── 多仓库 Git ───────── */
  const repoRow = (n, br, sync, changes, on, extra = '') => `<div class="row ${on ? 'on' : ''}" style="height:32px">${ic('branch', 's12')}<div class="grow" style="min-width:0"><div class="ell"><b>${n}</b> <span class="t3">${br}</span></div></div><span class="t3 fs-cap mono">${sync}</span>${changes ? cnt(changes, on ? 'ac' : '') : '<span class="t4 fs-cap">干净</span>'}${extra}</div>`;
  const scmMulti = `<div style="padding:8px">${gh('仓库', 4, ib('refresh', 'sm', '全部拉取') + ib('sync', 'sm', '全部同步'))}
    ${repoRow('router-demo', 'main', '↑1', 3, 1)}${repoRow('web-dashboard', 'feat/charts', '↑2', 2, 0)}${repoRow('docs-site', 'main', '↓1', 0, 0)}
    <div class="row" style="height:32px;color:var(--tx-3)">${ic('folder', 's12')}<div class="grow ell">design-assets</div>${btn('初始化仓库', 'sm ghost')}</div>
    <div class="sep" style="margin:6px 0"></div>
    <div class="t3 fs-cap" style="padding:2px 6px">提交到 <b>router-demo</b> · main</div>
    <div class="inp" style="margin:4px 4px 6px;height:52px"><span class="t4">消息(⌘↩ 提交到 router-demo)</span></div>
    <div class="f g2" style="padding:0 4px">${btn('提交', 'pri', 'check')}${ib('spark', '', '用 Claude 生成说明')}</div>
    ${gh('暂存的更改', 1)}<div class="row tr" style="height:22px;padding-left:20px">${ic('code', 's12')}<span class="ell">src/router.rs</span><span class="g gm">M</span></div>
    ${gh('更改', 2)}<div class="row tr" style="height:22px;padding-left:20px">${ic('code', 's12')}<span class="ell">tests/router_test.rs</span><span class="g gm">M</span></div></div>`;

  /* ── B1:多仓库的审查分组 / 逐个提交 / 图谱仓库 / 搜索范围 / 子模块 ── */
  const rvRepo = (name, br, n, a, d, files, note = '') => `<div style="border-radius:8px;box-shadow:inset 0 0 0 1px var(--bd-1);margin-bottom:8px;overflow:hidden"><div class="f ai g3" style="height:32px;padding:0 10px;background:var(--fill-1)">${ic('branch', 's12')}<b>${name}</b><span class="t3">${br}</span><span class="t4 fs-cap">${n} 个文件</span><span class="c-ok mono fs-micro">+${a}</span><span class="c-err mono fs-micro">−${d}</span><span class="grow"></span>${note}${btn('提交到 ' + name + '…', 'sm ghost', 'commit')}</div>${files.map(f => `<div class="row" style="height:26px;padding-left:22px">${ic('code', 's12', 't3')}<b class="ell">${f[0]}</b><span class="t4 fs-micro ell">${f[1]}</span><span class="grow"></span><span class="c-ok mono fs-micro">+${f[2]}</span><span class="c-err mono fs-micro">−${f[3]}</span></div>`).join('')}</div>`;
  const banner = (n, extra = '') => `<div class="f ai g3" style="padding:8px 10px;margin-bottom:8px;border-radius:8px;background:var(--ac-soft)">${ic('info', 's12', 'c-ac')}<span class="grow">本轮改动涉及 <b>${n}</b> 个仓库,提交需要分别进行。</span>${extra}</div>`;
  const wizBar = (i, n, repo, br, st) => `<div class="f ai g3" style="padding:8px 10px;border-radius:8px;background:var(--ac-soft);box-shadow:inset 0 0 0 1px var(--ac)">${ic('commit', 's12', 'c-ac')}<div class="grow"><div><b>逐个提交</b> <span class="t3">第 ${i} / ${n} 个</span></div><div class="t3 fs-cap">${repo} · ${br} ${st || ''}</div></div><span class="f g2">${Array.from({ length: n }, (_, k) => `<i style="width:7px;height:7px;border-radius:50%;background:${k < i - 1 ? 'var(--ok)' : k === i - 1 ? 'var(--ac)' : 'var(--bd-2)'}"></i>`).join('')}</span>${btn('跳过', 'sm ghost')}${ib('x', 'sm', '退出向导')}</div>`;
  const mgReview = grid(
    cell('审查标签 · 涉及两个仓库', '', box(`<div style="padding:10px 12px"><div class="f ai g3" style="margin-bottom:8px">${ic('eye', 's12', 'c-ac')}<b>本轮改动</b><span class="t3">4 个文件 · 2 个仓库</span><span class="grow"></span>${btn('全部接受', 'sm soft', 'check')}</div>${banner(2, btn('逐个提交', 'sm pri', 'commit'))}${rvRepo('router-demo', 'main', 2, 14, 2, [['router.rs', 'src/', 12, 2], ['router_test.rs', 'tests/', 2, 0]])}${rvRepo('web-dashboard', 'feat/charts', 2, 9, 1, [['api.ts', 'src/', 6, 1], ['chart.tsx', 'src/', 3, 0]])}</div>`, 400, 560), '文件按所属仓库分组;分组头可折叠,里面仍是原来的文件卡 / 块级审查。「提交到 xxx…」= 切活动仓库 + 打开 SCM + 预填说明。'),
    cell('只有一个仓库有改动 / 单仓库', '', box(`<div style="padding:10px 12px"><div class="f ai g3" style="margin-bottom:8px">${ic('eye', 's12', 'c-ac')}<b>本轮改动</b><span class="t3">2 个文件</span></div>${rvRepo('router-demo', 'main', 2, 14, 2, [['router.rs', 'src/', 12, 2], ['router_test.rs', 'tests/', 2, 0]])}<div class="t3 fs-cap" style="padding:2px 4px">只涉及一个仓库时不出现提示条;单项目(非工作区)完全不出现分组头,与现在一致。</div></div>`, 220, 560), '不在任何仓库里的文件(例如 design-assets 里的)归入「未纳入 Git」分组,没有提交按钮,只有「初始化仓库」。'));
  const mgWizard = grid(
    cell('向导条(出现在 SCM 提交框上方)', '', box(`<div style="padding:10px 12px;display:flex;flex-direction:column;gap:8px">${wizBar(1, 2, 'router-demo', 'main', '· 已预填说明,检查后 ⌘↩ 提交')}<div class="t3 fs-cap">提交成功 → 自动切到下一个有改动的仓库并预填说明。</div>${wizBar(2, 2, 'web-dashboard', 'feat/charts', '· 提交失败:pre-commit 钩子未通过,留在本步')}<div class="t3 fs-cap">失败不前进,错误写在条里;可修好再提交,或「跳过」。</div></div>`, 240, 560), '向导只是「依次切活动仓库 + 预填说明」的辅助,不会自动提交——每一次提交仍由人按下。随时「退出向导」,已提交的不受影响。'),
    cell('完成与中断', '', box(`<div style="padding:10px 12px;display:flex;flex-direction:column;gap:8px"><div class="f ai g3" style="padding:8px 10px;border-radius:8px;background:var(--ok-soft)">${ic('check', 's12', 'c-ok')}<span class="grow"><b>已提交 2 个仓库</b> <span class="t3">· 1 个跳过 · 向导结束</span></span>${btn('查看图谱', 'sm ghost', 'fork')}</div><div class="t3 fs-cap">所有仓库都处理完,向导条变成这一行,6 秒后淡出;被跳过的仓库在总览行保留「有未提交改动」小圆点。</div><div class="f ai g3" style="padding:8px 10px;border-radius:8px;background:var(--fill-1)">${ic('info', 's12', 't3')}<span class="grow t3">手动切到了别的仓库:向导暂停,点「回到向导」继续。</span>${btn('回到向导', 'sm ghost')}</div></div>`, 240, 560), '中途手动点总览行切仓库 = 暂停(不是退出),避免向导和手动操作互相打架。'));
  const mgGraph = grid(
    cell('图谱标签 · 仓库下拉', '', box(`<div style="padding:10px 12px"><div class="f ai g3" style="margin-bottom:8px">${ic('fork', 's12', 'c-ac')}<b>Git 图谱</b>${chip('router-demo', 'folder', 'on')}<span class="t3">main · 7 个提交</span><span class="grow"></span><div class="inp fill" style="width:140px;height:24px"><span class="t4">搜索提交…</span></div></div><div style="position:relative;height:150px">${menu(240, [['folder', 'router-demo · main', '', 'on'], ['folder', 'web-dashboard · feat/charts'], ['folder', 'docs-site · main']])}</div></div>`, 280, 560), '下拉只列有仓库的文件夹。切换 = 切换活动仓库(图谱、源代码管理、状态栏分支同步变),避免出现「图谱看 A、提交框写 B」的错位;标签名随之变成「图谱 · web-dashboard」。'),
    cell('切换后的状态', '', box(`<div style="padding:12px 14px" class="t2"><div>· 选中的提交与展开详情清空(不同仓库的哈希无意义)</div><div>· 搜索词保留,对新仓库重新匹配</div><div>· 新仓库的提交列表在后台读取,读取期间显示骨架行</div><div>· 非仓库文件夹不出现在下拉里;只有一个仓库时不显示下拉,只显示仓库名</div><div>· 单项目(非工作区)与现在一致,没有仓库 chip</div></div>`, 150, 560), '图谱数据按仓库各读一份,切回来不重读(8 秒内)。'));
  const mgSearch = grid(
    cell('搜索 · 范围 chip', '', box(`<div style="padding:10px 12px"><div class="inp fill" style="height:28px;margin-bottom:6px"><span class="v">match_path</span></div><div class="f ai g2" style="margin-bottom:6px">${chip('范围:全部文件夹', 'folder', 'line')}<span class="t4 fs-cap">只在多根工作区里出现</span></div>${menu(230, [['folder', '全部文件夹', '', 'on'], ['folder', 'router-demo'], ['folder', 'web-dashboard'], ['folder', 'docs-site']])}</div>`, 250, 560), '范围写进搜索条件:结果只含该文件夹;chip 在限定时变高亮并带 ✕ 一键清除。'),
    cell('限定后与入口', '', box(`<div style="padding:10px 12px"><div class="f ai g2" style="margin-bottom:8px">${chip('router-demo', 'folder', 'on')}${ib('x', 'sm', '清除范围')}<span class="t3 fs-cap">在 router-demo 内 · 4 个文件 · 7 处匹配</span></div><div class="t3 fs-cap" style="line-height:1.8">入口:<br>· 文件树里文件夹右键「在文件夹中搜索…」(多根时顶层文件夹也有「仅在此文件夹搜索」)<br>· 搜索面板里的范围 chip<br>范围只是「第一段路径 = 文件夹名」的过滤,和「要包含的文件」glob 可叠加。切换工作区时清除。</div></div>`, 190, 560), '没有命中时的空态写明范围:「router-demo 内没有结果 · 清除范围」。'));
  const mgNested = grid(
    cell('子模块 / 嵌套仓库', '', box(`<div style="padding:8px"><div class="row" style="height:24px">${ic('chevD', 's12')}${ic('folder', 's12', 'c-ac')}<span class="grow b">web-dashboard</span>${tag('主', 'ac')}</div><div class="row" style="height:22px;padding-left:34px">${ic('chevD', 's12')}${ic('folder', 's12')}<span class="grow">packages</span></div><div class="row" style="height:22px;padding-left:50px">${ic('folder', 's12', 'c-ac')}<span class="grow">ui</span>${tag('子模块')}${cnt(1)}</div><div class="sep" style="margin:6px 0"></div>${repoRow('web-dashboard', 'feat/charts', '↑2', 2, 0)}<div style="padding-left:18px">${repoRow('packages/ui', 'main', '', 1, 0, tag('子模块'))}</div></div>`, 230, 560), '嵌套仓库在总览里缩进显示在父仓库下面,带「子模块」标签(没有 .gitmodules 的嵌套仓库标「嵌套」);文件树里那个文件夹带同样的标签和改动数。'),
    cell('归属规则', '', box(`<div style="padding:12px 14px" class="t2"><div>· 一个文件属于「离它最近的 .git」所在的仓库</div><div>· 父仓库的状态把子模块当成一个条目(提交指针变更),文件级改动在子模块自己的行里处理</div><div>· 父仓库的暂存 / 提交不会带上子模块里未提交的改动,总览行提示「子模块有未提交改动」</div><div>· 发现范围:每个工作区文件夹向下 2 层找 .git,跳过 node_modules / target 等</div><div>· 子模块行同样有切换活动仓库 / 拉取 / 推送</div></div>`, 190, 560), '发现是后台、只读的 git rev-parse / ls-files;失败不影响其它仓库。'));
  P['multi-git'] = () => `<div class="ds" style="max-width:1240px">${head('多仓库 Git · 项目修改产生的 Git 文件怎么处理', '原则(对标 VS Code):每个仓库独立,<b>任何会改动仓库的操作都必须让用户一眼看清「作用在哪个仓库」</b>。默认是「活动仓库」模式 —— 顶部是仓库总览(每个仓库一行:分支 · 领先落后 · 改动数),下面是活动仓库的完整 SCM;点另一行就切换活动仓库。「全部展开」模式(设置项)把每个仓库的区块都铺开。')}
    ${sec('源代码管理(多仓库)', '总览行始终可见;提交框上方写明「提交到 router-demo · main」;非仓库文件夹给「初始化仓库」,不报错。', grid(
      cell('活动仓库模式(默认)', '', box(scmMulti, 520, 330), '总览里每行 10 秒在后台读一次状态(只读 git status,带 --no-optional-locks);活动仓库照常 4 秒 + 文件监视刷新。'),
      cell('总览行右键 / 悬停动作', '', box(`<div style="padding:10px 12px" class="fc g3"><div class="row" style="height:32px;background:var(--fill-1);border-radius:6px">${ic('branch', 's12')}<div class="grow"><b>web-dashboard</b> <span class="t3">feat/charts</span></div><span class="acts">${ib('refresh', 'sm', '拉取')}${ib('sync', 'sm', '同步')}${ib('branch', 'sm', '切换分支…')}${ib('more', 'sm')}</span></div>${menu(240, [['arrow', '设为活动仓库'], ['download', '拉取'], ['upload', '推送'], ['sync', '同步'], ['branch', '切换分支…'], ['-'], ['fork', '在图谱中查看'], ['layers', '储藏所有更改'], ['-'], ['folder', '在访达中显示'], ['terminal', '在此处打开终端']])}</div>`, 400, 330), '每个动作只作用于这一行的仓库,与是否为活动仓库无关。')))}
    ${sec('一次操作涉及多个仓库时', '', tbl([['场景', 200], ['怎么做'], ['状态']], [
      ['提交', '只提交活动仓库的暂存区;提交按钮文案写明仓库名;其它仓库的暂存区在总览行用小圆点提示「有已暂存未提交」', st('ok')],
      ['全部同步 / 全部拉取', '总览头部按钮,逐个仓库顺序执行,每行显示转圈 → 成功 / 失败;失败的一行出现「重试」,不中断其余仓库;有合并冲突的仓库跳过并在汇总里说明', st('ok')],
      ['切换分支', '状态栏分支名 = 活动仓库的分支,点它切换的就是活动仓库;总览行的「切换分支…」切换那一个', st('ok')],
      ['Claude 本轮改动(审查标签)', '按文件所属仓库分组,分组标题 = 仓库名 + 分支;「在源代码管理里暂存」只暂存该仓库内的文件', st('ok')],
      ['同一次回合改了 2 个仓库', '审查标签顶部提示「改动涉及 2 个仓库,提交需分别进行」,并给出「逐个提交」向导(依次切活动仓库、预填说明)', st('ok')],
      ['路径显示', '多根时路径前缀文件夹名(web-dashboard/src/api.ts);单根与现在一致', st('ok')]]))}
    ${sec('审查标签 · 按仓库分组', '工作区里 Claude 一轮可能改了好几个仓库的文件。审查标签只做「看」和「接受 / 撤销」,提交仍在源代码管理里按仓库分别进行。', mgReview)}
    ${sec('「逐个提交」向导', '一次回合改了 2 个及以上仓库时,审查标签顶部的提示条给出入口;向导把「切仓库 → 预填说明 → 提交 → 下一个」串起来,但不替人按提交。', mgWizard)}
    ${sec('Git 图谱 · 仓库下拉', '', mgGraph)}
    ${sec('搜索 · 仅某个文件夹', '', mgSearch)}
    ${sec('嵌套仓库与子模块', '', mgNested)}
    ${sec('图谱、冲突、储藏', '', tbl([['能力', 200], ['怎么做'], ['状态']], [
      ['Git 图谱', '整页虚拟标签 vGraph 显示活动仓库;标题行有仓库下拉可切换;标签名带仓库名「图谱 · router-demo」', st('ok')],
      ['合并冲突', 'SCM 顶部「合并冲突」分组出现在对应仓库的区块内;总览行显示红色冲突徽标(阻止「全部同步」)', st('ok')],
      ['储藏', '每个活动仓库各自的分组与菜单;总览行「储藏所有更改」只作用该仓库', st('ok')],
      ['文件树状态色', '所有仓库的改动一起上色(M 橙 / A 绿 / U 灰),按文件路径前缀归属;文件夹节点右侧显示仓库改动数', st('ok')],
      ['嵌套仓库(子模块 / 仓库里的仓库)', '按文件夹最近的 .git 归属;子模块作为独立行显示「子模块」标签;总览缩进 + 文件树目录行带标签和改动数;SCM 提示行「子模块有未提交改动」+ 父仓库指针变更(M)条目', st('ok')],
      ['非仓库文件夹', '总览里灰行 + 「初始化仓库」;初始化走 git init 并在输出面板记录', st('ok')]]))}
    ${sec('状态栏与标题栏', '', grid(
      cell('状态栏', '', box(`<div class="f ai g4" style="padding:12px 14px">${chip('router-demo · main', 'branch')}<span class="t3">↑1</span><span class="t3">⚠ 1</span><span class="grow"></span><span class="t3">全栈工作区 · 3 个文件夹</span></div><div class="t3" style="padding:0 14px;line-height:1.7">分支胶囊写「仓库名 · 分支」;点开分支选择器,顶部一行显示作用的仓库,可切换;窄窗口只留分支名。</div>`, 130, 560), '仓库名在多仓库时才出现;单仓库保持现在的「分支」。'),
      cell('分支选择器(带仓库)', '', box(`<div class="pop" style="width:380px;margin:12px auto"><div class="f ai g2" style="padding:6px 8px">${chip('router-demo', 'folder', 'on')}${chip('web-dashboard', 'folder', 'line')}${chip('docs-site', 'folder', 'line')}</div><div class="inp fill" style="margin:3px;height:28px"><span class="t4">切换或新建分支…</span></div>${mi('branch', 'main', '', 'on')}${mi('branch', 'feat/auth')}${mi('plus', '创建并切换到 “fix/case”')}</div>`, 200, 560), '顶部 chip 选仓库,下面是该仓库的分支;输入不存在的名字出现「创建并切换」。')))}
    ${sec('Claude 与多仓库', '会话 cwd 在主文件夹所在仓库;附加文件夹里的修改同样进入「本轮审查」,但提交永远由人分别确认。Claude 自己运行 git 命令时,以它的 cwd 所在仓库为准(与终端一致)。', '')}</div>`;

  /* ───────── 右键菜单与动作矩阵 ───────── */
  const M = (title, items) => `<div><div class="cap2"><b>${title}</b></div>${menu(236, items)}</div>`;
  P['menus'] = () => `<div class="ds" style="max-width:1240px">${head('右键菜单与动作矩阵', '规则:① 每个可点的对象都有右键菜单,菜单顺序固定为「主要动作 → 导航 → 复制 / 访达 → 危险操作」;② 危险项在最后、红色、带确认;③ 同一动作在不同对象里叫法与图标一致;④ 菜单项有快捷键就显示;⑤ 不可用的项置灰并在悬停提示原因,而不是隐藏。')}
    ${sec('导航栏', '', `<div class="gal" style="grid-template-columns:repeat(auto-fill,minmax(250px,1fr))">
      ${M('会话行', [['play', '打开', '⏎'], ['edit', '重命名…'], ['pin', '置顶'], ['archive', '归档'], ['-'], ['copy', '复制会话 ID'], ['copy', '复制为 Markdown'], ['-'], ['trash', '删除会话…', '', 'danger']])}
      ${M('文件(文件树)', [['code', '打开', '⏎'], ['split', '在侧边打开', '⌘⏎'], ['-'], ['eye', '打开更改', '', ''], ['clock', '显示文件历史'], ['spark', '让 Claude 解释 / 修改…'], ['-'], ['edit', '重命名…', 'F2'], ['copy', '复制'], ['trash', '删除…', '⌘⌫', 'danger'], ['-'], ['folder', '在访达中显示'], ['copy', '复制路径'], ['copy', '复制相对路径']])}
      ${M('文件夹(文件树)', [['plus', '新建文件…'], ['folder', '新建文件夹…'], ['-'], ['search', '在文件夹中搜索…'], ['terminal', '在此处打开终端'], ['-'], ['edit', '重命名…'], ['trash', '删除…', '', 'danger'], ['-'], ['folder', '在访达中显示'], ['copy', '复制路径']])}
      ${M('多根:顶层文件夹', [['branch', '在源代码管理中查看'], ['arrow', '设为主文件夹'], ['search', '仅在此文件夹搜索'], ['-'], ['folder', '在访达中显示'], ['copy', '复制路径'], ['-'], ['x', '从工作区移出', '', 'danger']])}
      ${M('搜索结果', [['code', '打开'], ['eye', '在侧边预览'], ['x', '忽略此结果'], ['-'], ['copy', '复制路径与行号'], ['copy', '复制匹配行']])}
      ${M('SCM 文件', [['diff', '打开更改'], ['code', '打开文件'], ['-'], ['plus', '暂存更改'], ['undo', '放弃更改…', '', 'danger'], ['-'], ['clock', '显示文件历史'], ['copy', '复制路径']])}
      ${M('SCM 分组 / 仓库', [['plus', '全部暂存'], ['minus', '全部取消暂存'], ['undo', '放弃全部更改…', '', 'danger'], ['-'], ['layers', '储藏所有更改'], ['refresh', '刷新']])}
      ${M('图谱提交', [['commit', '检出此提交'], ['branch', '从此提交创建分支…'], ['fork', '拣选'], ['undo', '还原'], ['-'], ['diff', '与工作区比较'], ['copy', '复制哈希'], ['-'], ['rewind', '回退到这里…', '', 'danger']])}
    </div>`)}
    ${sec('工作区(标签 · 编辑器 · 面板)', '', `<div class="gal" style="grid-template-columns:repeat(auto-fill,minmax(250px,1fr))">
      ${M('标签', [['x', '关闭', '⌘W'], ['x', '关闭其他'], ['x', '关闭右侧'], ['x', '关闭已保存'], ['-'], ['pin', '固定标签'], ['split', '向右拆分', '⌘\\'], ['-'], ['copy', '复制路径'], ['folder', '在访达中显示'], ['tree', '在文件树中定位']])}
      ${M('编辑器', [['arrow', '转到定义', 'F12'], ['eye', '预览定义', '⌥F12'], ['search', '查找所有引用', '⇧F12'], ['edit', '重命名符号', 'F2'], ['-'], ['spark', '问 Claude…', '⌘L'], ['spark', '行内编辑…', '⌘I'], ['-'], ['code', '格式化文档', '⇧⌥F'], ['clock', '显示文件历史'], ['-'], ['copy', '复制'], ['copy', '粘贴'], ['cmd', '命令面板…', '⇧⌘P']])}
      ${M('终端', [['copy', '复制', '⌘C'], ['copy', '粘贴', '⌘V'], ['-'], ['search', '查找…', '⌘F'], ['trash', '清屏', '⌘K'], ['-'], ['plus', '新建终端'], ['x', '关闭终端'], ['-'], ['folder', '在访达中打开当前目录']])}
      ${M('问题 / 输出行', [['code', '转到位置'], ['spark', '让 Claude 修复'], ['-'], ['copy', '复制消息'], ['copy', '复制全部'], ['trash', '清空输出', '', '']])}
      ${M('虚拟标签(计划 / 审查 / 任务)', [['copy', '复制内容'], ['chat', '引用到对话'], ['x', '关闭'], ['-'], ['arrow', '在新窗口中打开']])}
    </div>`)}
    ${sec('会话行 · 置顶 / 归档 / 重命名 / 删除', '会话的这四个动作只改「本机的整理信息」(置顶 / 归档 / 自定义标题),不改 Claude 的会话文件;删除才会移除磁盘上的会话记录。整理信息存 ' + mono('~/.minicode/session-meta.json') + '(原子落盘,损坏改名留底)。', grid(
      cell('侧栏:置顶与已归档', '', box(`<div style="padding:8px"><div class="row" style="height:28px">${ic('chevD', 's12')}${ic('folder', 's12')}<span class="grow b">router-demo</span><span class="t4 fs-micro">7</span></div>
        <div class="row" style="height:28px;padding-left:25px">${ic('pin', 's12', 'c-ac')}<span class="grow ell">完整修复一个 Bug:路由大小写</span><span class="aux">刚刚</span></div>
        <div class="row" style="height:28px;padding-left:25px">${ic('pin', 's12', 'c-ac')}<span class="grow ell">给 router 补一组基准测试</span><span class="aux">昨天</span></div>
        <div class="row" style="height:28px;padding-left:25px"><span class="grow ell">权限请求:修改配置…</span><span class="aux">5 分钟</span></div>
        <div class="row" style="height:28px;padding-left:25px"><span class="grow ell">梳理路由表的内存占用</span><span class="aux">周一</span></div>
        <div class="row" style="height:24px;padding-left:25px;color:var(--tx-3)">${ic('chevR', 's12')}${ic('archive', 's12')}<span class="grow">已归档</span><span class="t4 fs-micro">2</span></div></div>`, 250, 330), '置顶的会话排在项目的最前,标题前带强调色图钉(代替时间位置的留白,时间仍在行尾);归档的会话从列表里收进该项目底部的「已归档 N」折叠行,默认收起,展开后文字变淡、菜单里的「归档」变「取消归档」。当前会话被归档时仍留在对话栏,不会被关掉。'),
      cell('重命名 · 行内编辑', '', box(`<div style="padding:8px"><div class="row" style="height:28px;padding-left:25px">${ic('pin', 's12', 'c-ac')}<span class="grow ell">完整修复一个 Bug:路由大小写</span><span class="aux">刚刚</span></div>
        <div class="row on" style="height:30px;padding-left:25px"><div class="inp focus grow" style="height:24px"><span class="v caret">路由大小写匹配的修复</span></div>${ib('check', 'sm', '确认 ⏎')}${ib('x', 'sm', '取消 Esc')}</div>
        <div class="row" style="height:28px;padding-left:25px"><span class="grow ell">权限请求:修改配置…</span><span class="aux">5 分钟</span></div>
        <div class="t3 fs-cap" style="padding:8px 10px;line-height:1.7">· 菜单「重命名…」或双击标题进入,全选当前标题<br>· ⏎ 保存,Esc / 点别处取消;留空 = 恢复 Claude 生成的原标题<br>· 对话栏头部与面包屑同步显示新标题</div></div>`, 250, 330), '自定义标题只存在本机整理信息里;原标题仍在会话文件里,「恢复原标题」= 清空输入后回车。'),
      cell('删除会话 · 确认与限制', '', box(`<div style="padding:12px 14px" class="fc g3"><div class="f ai g3"><span class="dlg-i err">${ic('trash')}</span><b class="fs-title">删除这个会话?</b></div><div class="t3" style="line-height:1.55">「梳理路由表的内存占用」的会话记录会从磁盘移除,无法恢复。</div><div class="f g2" style="justify-content:flex-end">${btn('取消', 'ghost')}${btn('删除', 'danger')}</div>
        <div class="sep"></div><div class="t3 fs-cap" style="line-height:1.7">· 正在运行的会话「删除会话…」置灰,提示「会话正在运行,先停止或等它结束」<br>· 当前打开的会话可删除:删除后切到该项目的下一个会话,没有就进入新会话<br>· 置顶 / 归档 / 自定义标题的记录随会话一起清掉<br>· 演示模式只从内存列表里移除</div></div>`, 250, 330), '真实模式走 claude.DeleteSession(只认字母 / 数字 / - / _ 的 id,不会越出会话目录);失败给出原因的提示条。'),
      cell('复制为 Markdown', '', box(`<div style="padding:12px 14px" class="t2"><div class="mono fs-cap" style="line-height:1.7;white-space:pre">## 完整修复一个 Bug:路由大小写匹配\n\n**你**\n路由匹配对大小写敏感……\n\n**Claude**\n问题出在路由表直接用原始路径做键……\n\n&gt; 编辑 src/router.rs(+2 −1)\n&gt; 运行 cargo test router · 通过</div><div class="t3 fs-cap" style="margin-top:8px;line-height:1.7">· 用户消息与助手正文原样输出;思考省略;工具调用折成一行引用<br>· 只对「已打开的会话」可用(其它会话先点开);菜单项置灰时提示原因</div></div>`, 250, 330), '对话栏头部 ··· 与会话行 ··· 共用;复制后提示「已复制为 Markdown · 12 条」。')))}
    ${sec('SCM · 图谱 · 搜索 · 菜单的具体行为', '菜单项背后的对话与流程。凡是会丢东西的动作都先确认,并写明「丢什么、能不能找回」。', grid(
      cell('放弃更改 · 确认', '', box(`<div style="padding:12px 14px" class="fc g3"><div class="f ai g3"><span class="dlg-i err">${ic('undo')}</span><b class="fs-title">放弃 router.rs 的更改?</b></div><div class="t3" style="line-height:1.55">工作区里的修改会恢复成暂存区 / HEAD 的样子,无法撤销。</div><div class="f g2" style="justify-content:flex-end">${btn('取消', 'ghost')}${btn('放弃更改', 'danger')}</div>
        <div class="sep"></div><div class="t3 fs-cap" style="line-height:1.7">· 悬停按钮与右键菜单走同一个确认框<br>· 「放弃全部更改」写明文件数:放弃 3 个文件的更改?<br>· 未跟踪文件写明「会被删除」<br>· 只放弃「更改」分组,暂存区的不动</div></div>`, 250, 330), '原先悬停按钮会直接丢改动,现在统一加确认。'),
      cell('SCM 分组菜单', '', box(menu(220, [['plus', '全部暂存'], ['minus', '全部取消暂存'], ['undo', '放弃全部更改…', '', 'danger'], ['-'], ['layers', '储藏所有更改'], ['refresh', '刷新']]) + `<div class="t3 fs-cap" style="padding:8px 4px;line-height:1.7">「更改」分组与「暂存的更改」分组、仓库行共用;当前组为空的动作置灰。右键分组标题或在多仓库总览右键仓库行。</div>`, 250, 330), '仓库行菜单(拉取 / 推送 / 同步 / 切换分支 / 图谱 / 储藏 / 访达 / 终端)已有,不重复。'),
      cell('图谱 · 与工作区比较', '', box(`<div class="pop" style="width:380px;margin:12px auto"><div class="inp fill" style="margin:3px;height:28px"><span class="t4">选择要与工作区比较的文件(a3f9c21)</span></div>${mi('file', 'src/router.rs', 'M', 'on')}${mi('file', 'tests/router_test.rs', 'M')}${mi('file', 'Cargo.toml', 'A')}</div><div class="t3 fs-cap" style="padding:8px 14px;line-height:1.7">· 提交只涉及几个文件,所以先选文件;选中后在编辑区开「a3f9c21 ↔ 工作区」的差异标签<br>· 该提交里没有改动文件(合并提交)时提示「这个提交没有文件改动」</div>`, 250, 330), '差异标签只读(对比的不是暂存区,不开放块级操作)。'),
      cell('图谱 · 回退到这里', '', box(`<div style="padding:12px 14px" class="fc g3"><div class="f ai g3"><span class="dlg-i err">${ic('rewind')}</span><b class="fs-title">把 main 回退到 a3f9c21?</b></div><div class="t3" style="line-height:1.55">a3f9c21 之后的 3 个提交会从 main 上移除(仍可从 reflog 找回)。未提交的改动保留;如果与回退的内容冲突,操作会中止并告诉你原因。</div><div class="f g2" style="justify-content:flex-end">${btn('取消', 'ghost')}${btn('回退', 'danger')}</div>
        <div class="sep"></div><div class="t3 fs-cap" style="line-height:1.7">· 等价于 <span class="mono">git reset --keep</span>,不用 --hard,不丢工作区<br>· 游离 HEAD 时置灰:先切到分支<br>· 已推送的提交:提示「回退后需要强制推送」,本应用不提供强制推送</div></div>`, 250, 330), '回退是本页里唯一会改分支历史的动作,所以放在菜单最底部、红色。'),
      cell('搜索结果 · 右键', '', box(menu(230, [['code', '打开'], ['eye', '在侧边预览', '', 'dis'], ['x', '忽略此结果'], ['-'], ['copy', '复制路径与行号'], ['copy', '复制匹配行']]) + `<div class="t3 fs-cap" style="padding:8px 4px;line-height:1.7">· 匹配行与文件行各有一份:文件行 = 展开 / 折叠、在文件树中定位、复制路径<br>· 「在侧边预览」依赖编辑器拆分,暂置灰并提示原因<br>· 复制路径与行号 = <span class="mono">src/router.rs:42:9</span></div>`, 250, 330), '忽略 / 恢复与悬停按钮是同一个动作。')))}
    ${sec('会话流里的动作', '步骤行悬停时右侧出现小按钮(只列这一行真的能做的);点图标 / 文件名 / 整行(没有可展开内容时)也能跳。目标不存在时给出提示,不静默失败。', tbl([['对象', 150], ['悬停按钮', 220], ['点击行 / 对象', 180], ['右键']], [
      ['读取文件', '打开文件 · 复制路径', '在工作区打开(定位到起始行)', '复制路径 · 在访达中显示'],
      ['编辑 / 写入文件', '打开文件 · 查看差异 · 复制路径', '展开 / 收起差异预览', '同上 + 在审查标签中查看'],
      ['运行命令', '在终端面板查看 · 复制命令', '展开 / 收起输出(无输出时 = 在终端面板查看)', '复制命令 · 复制输出'],
      ['搜索 / 查找文件', '复制搜索词', '展开(列表)', '复制'],
      ['抓取网页', '在浏览器中打开 · 复制网址', '展开', '复制网址'],
      ['差异卡 / 命令卡头部', '在工作区查看差异 · 复制', '—', '复制'],
      ['助手正文里的链接', '—', '网址 → 系统浏览器;路径[:行:列] → 工作区', '在工作区打开 · 在浏览器中打开 · 复制']]))}
    ${sec('实现状态', '', tbl([['范围', 220], ['说明', 440], ['状态']], [
      ['会话流步骤行 / 命令卡 / 差异卡 / 链接动作', '本页「会话流里的动作」整张表', st('ok')],
      ['文件 / 文件夹 / 标签 / 编辑器 / 图谱 / 终端菜单', 'treeMenu / tabMenu / scm 等已对齐;「在侧边打开 / 预览」需先拆分编辑器组;编辑器右键未全接 LSP 子项', st('part')],
      ['会话行 / 项目行 / 工作区行菜单', '置顶 / 归档 / 重命名 / 删除 / 复制为 Markdown / 复制标题 / 链接已做;头部「分叉会话…」= CmdForkSession(运行中置灰);侧栏行无分叉项', st('ok')],
      ['SCM 文件行 / 分组 / 图谱提交 / 搜索结果菜单', '本页「具体行为」一节已实现:放弃更改确认(悬停按钮同走)、分组菜单、与工作区比较(先选文件,只读差异)、回退到这里(reset --keep,含已推送提示)、搜索结果菜单(匹配行 / 文件行);「在侧边预览」依赖编辑器拆分,置灰', st('ok')],
      ['问题 / 输出 / 虚拟标签菜单', '问题行:转到位置 / 让 Claude 修复 / 复制消息 / 复制全部;输出行:复制此行 / 复制全部 / 清空(应用日志通道置灰);计划 / 子 Agent / 任务标签:复制内容 / 引用到对话(标签右键里)。「在新窗口中打开」未做,不放进菜单', st('ok')]]))}</div>`;

  /* 缺口总表见 js/status.js(P['gaps'] 自动扫描各页实现状态);勿在此维护手写清单。 */
  P['gaps'] = () => `<div class="ds" style="max-width:1240px">${head('(已迁移)', '本段为历史占位;请以 status.js 的缺口总表为准。')}
    ${sec('P0 · 主流程', '', tbl([['项', 220], ['现状', 360], ['计划', 200], ['状态']], [
      ['添加项目 / 工作区', '已有:侧栏「+」→ 对话框 / 校验 / 右键菜单', '—', st('ok')],
      ['多根文件树与搜索', '文件树多根与搜索范围筛选已做', '—', st('ok')],
      ['多仓库 SCM', '总览行 + 活动仓库切换 + 全部拉取 / 同步 + 初始化仓库已做;审查按仓库分组 / 图谱仓库下拉 / 逐个提交向导 / 搜索范围 / 嵌套仓库已做;子模块脏工作区提示与指针变更已做', '—', st('ok')],
      ['窗口自适应收起', '已按宽度自动收起 / 恢复导航栏与右侧栏', '—', st('ok')],
      ['差异左右 / 统一', '差异标签与审查标签都已生效,视图选择会记住', '—', st('ok')],
      ['会话流动作', '文件图标、跳转、复制、终端面板、链接都可用', '—', st('ok')]]))}
    ${sec('P1 · 完整度', '', tbl([['项', 220], ['现状', 360], ['计划', 200], ['状态']], [
      ['语言服务:大纲 / 代码操作 / 内联提示 / 语义高亮', '仅诊断 / 悬停 / 定义 / 引用 / 补全 / 重命名 / 格式化', 'documentSymbol → codeAction → inlayHint', st('part')],
      ['终端鼠标上报 / 命令装饰', '未做', 'OSC 133 + 鼠标模式', st('todo')],
      ['合并冲突「1 / 2」逐个切换与三方结果区', '逐个切换 + base + 可编辑结果', '—', st('ok')],
      ['差异块头常驻', '注解条常驻', '—', st('ok')],
      ['审查页行内评论串', '回复续串 + 发送给 Claude', '—', st('ok')],
      ['画布 / 浏览器 / 自定义', '演示数据为主', '见「画布 / 浏览器 / 自定义 · 业界做法」', st('part')],
      ['设置页:快捷键录制 / 服务商 / 规则 / 环境变量', '演示数据', '逐项接真实配置', st('part')],
      ['用量页账户限额 / 上下文分类', '限额接 get_usage(需活动会话);分类明细仍在输入框弹层', '上下文分类归 B4', st('part')]]))}
    ${sec('P2 · 打磨', '', tbl([['项', 220], ['现状', 360], ['计划', 200], ['状态']], [
      ['新元素的入场 / 退出动效', '弹层有 popIn;终端搜索条、安装卡、对话框未统一', '全部走 popIn / dialogIn', st('part')],
      ['文字流光 / 流底部渐隐遮罩', '用呼吸代替', '库支持后换', st('todo')],
      ['完成通知(系统通知)', '未做', '回合结束 / 需要审批 / 任务完成', st('todo')],
      ['拖拽排序(工作区 / 成员文件夹)', '拖动换位 + 上移 / 下移菜单(见 proj-ws「拖拽排序」);项目行按最近活动排序,不手动排', '—', st('ok')]]))}</div>`;
})();
