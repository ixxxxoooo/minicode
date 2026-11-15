/* 设计稿 v3.4 · Markdown 所见即所得与格式化、拖拽引用与「···」菜单盘点
   依据:Typora / Obsidian 实时预览 / Notion(所见即所得里直接改,有源码开关)、VS Code(Markdown 预览与编辑分开,所以不照搬)、
   Zed / Cursor(把文件拖进对话 = 插入 @ 引用)、VS Code「更多操作…」菜单的分组。
   每页都带「实现状态」列,与 Go 代码保持一致。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog, chk } = U;
  const { head, sec } = P.h;
  const tbl = (cols, rows) => `<table class="ia-tbl"><thead><tr>${cols.map(([t, w]) => `<th ${w ? `style="width:${w}px"` : ''}>${t}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map((c, k) => `<td ${k === 0 ? '' : 'class="t2"'}>${k === 0 ? `<b>${c}</b>` : c}</td>`).join('')}</tr>`).join('')}</tbody></table>`;
  const st = (s) => ({ ok: tag('已实现', 'ok'), part: tag('部分', 'warn'), todo: tag('待做', '') }[s]);
  const grid = (...cells) => `<div class="gal">${cells.join('')}</div>`;
  const cell = (t, keys, body, why) => `<div><div class="cap2"><b>${t}</b>${keys ? kbd(...keys) : ''}</div>${body}<div class="why">${why}</div></div>`;
  const box = (body, h = 240, w = 560, extra = '') => `<div class="snip" style="height:${h}px;width:${w}px;${extra}">${body}</div>`;
  const mi = (i, t, k, cls = '') => i === '-' ? '<div class="sep"></div>' : `<div class="mi ${cls}">${i ? ic(i, 's12') : '<span style="width:12px"></span>'}<span class="grow">${t}</span>${k ? kbd(k) : ''}</div>`;
  const menu = (w, items) => `<div class="pop" style="width:${w}px;padding:4px">${items.map(([i, t, k, c]) => mi(i, t, k, c)).join('')}</div>`;
  const M = (title, items, w = 236) => `<div><div class="cap2"><b>${title}</b></div>${menu(w, items)}</div>`;

  Object.assign(WS.TAB, { 'README.md': { k: 'file', icon: 'fileText', t: 'README.md' }, 'CHANGELOG.md': { k: 'file', icon: 'fileText', t: 'CHANGELOG.md', mod: 1 } });

  /* ───────── Markdown 所见即所得 ───────── */
  /** 编辑器顶部的面包屑条:左边路径,右边是视图切换 / 换行 / 格式化 / 语言 / 编码 */
  const bar = (right, cls = '') => `<div class="${cls}" style="height:26px;display:flex;align-items:center;gap:6px;padding:0 12px 0 14px;color:var(--tx-3);font-size:11.5px;flex:none;border-bottom:1px solid var(--bd-1)">docs${ic('chevR', 's12')}<b style="color:var(--tx-1);font-weight:500">README.md</b><span style="margin-left:auto;display:flex;gap:6px;align-items:center">${right}</span></div>`;
  const tools = (o = {}) => `${seg(['所见即所得', '源码'], o.src ? 1 : 0, 'sm')}${ib('wrap', o.wrap ? 'sm on' : 'sm', o.wrap ? '自动换行(已开)  ⌥Z' : '自动换行  ⌥Z')}${ib('align', 'sm', '格式化文档  ⇧⌥F')}<span class="t4">Markdown</span><span class="t4">UTF-8</span>`;
  /** 所见即所得的格式条:B I S code link | 标题 | 列表 任务 引用 代码块 */
  const fbtn = (t, tip, on = '') => `<span class="tip" data-tip="${tip}" style="min-width:24px;height:24px;display:inline-grid;place-items:center;border-radius:var(--r-sm);font-size:12px;${on ? 'background:var(--fill-2);color:var(--tx-1)' : 'color:var(--tx-2)'}">${t}</span>`;
  const fsep = '<span style="width:1px;height:14px;background:var(--bd-2);margin:0 4px"></span>';
  const fbar = `<div style="height:32px;display:flex;align-items:center;gap:1px;padding:0 10px;flex:none;border-bottom:1px solid var(--bd-1)">${fbtn('<b>B</b>', '粗体 ⌘B')}${fbtn('<i>I</i>', '斜体 ⌘I')}${fbtn('<s>S</s>', '删除线 ⇧⌘X')}${fbtn(ic('code', 's12'), '行内代码 ⌘E')}${fbtn(ic('link', 's12'), '链接 ⌘K')}${fsep}${fbtn('H1', '一级标题', 1)}${fbtn('H2', '二级标题')}${fbtn('H3', '三级标题')}${fsep}${fbtn(ic('list', 's12'), '无序列表')}${fbtn('1.', '有序列表')}${fbtn(ic('todo', 's12'), '任务列表')}${fbtn(ic('quote', 's12'), '引用')}${fbtn('{ }', '代码块')}<span class="grow"></span><span class="t4 fs-micro" style="white-space:nowrap"># 空格 · - 空格 · \`\`\` 即时生效</span></div>`;
  const mdbody = `<div class="md scroll" style="padding:14px 34px 20px;flex:1;overflow:hidden;line-height:1.7">
    <div style="font-size:21px;font-weight:700;letter-spacing:-.01em;margin:0 0 8px">MiniCode</div>
    <p>一个只支持 <b>Claude Code</b> 的工作台,直连本机的 <code>claude</code>,不走 ACP、不依赖 Node。</p>
    <div style="font-size:15px;font-weight:650;margin:12px 0 4px">快速开始</div>
    <ul><li>运行 <code>scripts/dev.sh --demo</code> 看演示数据</li><li>把文件从文件树<b>拖进对话栏</b>,就是 <code>@</code> 引用</li></ul>
    <div style="margin:8px 0 2px"><span style="display:inline-flex;vertical-align:-2px">${chk('done')}</span> 先看设计稿 &nbsp; <span style="display:inline-flex;vertical-align:-2px">${chk('')}</span> 再做功能</div>
    <div style="border-left:3px solid var(--bd-3);padding:0 0 0 10px;color:var(--tx-2);margin:10px 0">没有设计稿的功能,先补设计稿。<span class="cursor"></span></div></div>`;
  const srcLines = ['# MiniCode', '', '一个只支持 **Claude Code** 的工作台,直连本机的 `claude`,', '不走 ACP、不依赖 Node。', '', '## 快速开始', '', '- 运行 `scripts/dev.sh --demo` 看演示数据', '- 把文件从文件树**拖进对话栏**,就是 `@` 引用', '', '- [x] 先看设计稿', '- [ ] 再做功能', '', '> 没有设计稿的功能,先补设计稿。'];
  const srcBody = `<div style="display:flex;flex:1;overflow:hidden;font:12px/20px var(--font-mono);padding-top:6px"><div style="width:46px;text-align:right;padding-right:12px;color:var(--tx-4);flex:none">${srcLines.map((_, k) => k + 1).join('<br>')}</div><div style="white-space:pre;color:var(--tx-1)">${srcLines.map(l => U.esc(l).replace(/^(#+ )/, '<span style="color:var(--syn-kw)">$1</span>').replace(/(\*\*.+?\*\*)/g, '<b>$1</b>').replace(/(`[^`]+`)/g, '<span style="color:var(--syn-str)">$1</span>')).join('\n')}</div></div>`;
  const tabsMd = (act) => WS.strip(['README.md', 'CHANGELOG.md', 'router.rs'], act);
  const notice = (kind, text, acts) => `<div style="display:flex;align-items:center;gap:8px;padding:6px 12px;font-size:11.5px;flex:none;background:${kind === 'warn' ? 'var(--warn-soft)' : 'var(--fill-1)'};border-bottom:1px solid var(--bd-1)">${ic(kind === 'warn' ? 'alert' : 'info', `s12 ${kind === 'warn' ? 'c-warn' : 'c-info'}`)}<span class="grow t2">${text}</span>${acts}</div>`;

  /** 底部提示 toast,演示「格式化」的几种结果 */
  const toast = (i, t, c = '') => `<div class="pop" style="display:inline-flex;align-items:center;gap:8px;padding:7px 12px;margin:8px 0 0">${ic(i, `s12 ${c}`)}<span>${t}</span></div>`;
  const fmtCase = (label, result, why) => `<div class="fc g2"><div class="f ai g2"><b>${label}</b></div>${result}<div class="t3 fs-cap" style="line-height:1.6">${why}</div></div>`;

  P['ide-md'] = () => `<div class="ds" style="max-width:1240px">${head('Markdown · 所见即所得与格式化', '规则:① Markdown 文件(.md / .markdown / .mdc)默认以「所见即所得」打开——直接在渲染后的文档里改,不是左源码右预览;② 标签右上角的分段「所见即所得 | 源码」随时切换,两边读写的是同一份文本,切换不丢光标之外的任何状态;③ 无法在所见即所得里无损表示的内容(表格、HTML、脚注)不冒险改写:这类文件以只读预览打开并提示「用源码编辑」;④ 自动换行与格式化是所有文本编辑器共有的能力,放在顶部条右侧,不藏进菜单。')}
    ${sec('Markdown 标签', '默认视图 = 所见即所得;设置 → 编辑器与语言服务 → 「Markdown 默认视图」可改成源码,标签上手动切换的选择优先于偏好(只在该标签开着时记住,关掉即忘)。', grid(
      cell('所见即所得(默认)', [], box(tabsMd('README.md') + bar(tools()) + fbar + mdbody, 370, 560), '格式条随光标所在块高亮当前格式;⌘B / ⌘I / ⌘K 与 Markdown 快捷输入(# 空格、- 空格、``` )都生效;选中文字时再次点格式钮取消。'),
      cell('源码', [], box(tabsMd('README.md') + bar(tools({ src: 1, wrap: 1 })) + srcBody, 370, 560), '源码视图就是普通代码编辑器:行号、查找替换、迷你地图、自动换行、格式化、Git 竖条全有。'),
      cell('含表格 / HTML:只读预览', [], box(tabsMd('README.md') + bar(tools()) + notice('warn', '这个文件含表格,所见即所得会把它压成普通段落,所以只读显示。', btn('用源码编辑', 'sm', 'code')) + `<div class="md" style="padding:12px 34px;flex:1;overflow:hidden"><div style="font-size:15px;font-weight:650;margin:0 0 6px">接口</div><table style="border-collapse:collapse;font-size:12px;width:100%"><tr style="background:var(--fill-1)"><th style="text-align:left;padding:4px 8px;border:1px solid var(--bd-1)">方法</th><th style="text-align:left;padding:4px 8px;border:1px solid var(--bd-1)">路径</th></tr><tr><td style="padding:4px 8px;border:1px solid var(--bd-1)">GET</td><td style="padding:4px 8px;border:1px solid var(--bd-1)">/users</td></tr><tr><td style="padding:4px 8px;border:1px solid var(--bd-1)">POST</td><td style="padding:4px 8px;border:1px solid var(--bd-1)">/users</td></tr></table></div>`, 300, 560), '只读预览走会话流同一套渲染(表格、代码块、链接都完整);点链接 / 路径照常跳转。想改就切到源码。'),
      cell('写法会被规范化:提示条', [], box(tabsMd('README.md') + bar(tools()) + notice('info', '有 3 行的写法(如代码块语言标注、段内换行)在所见即所得里会被规范化。只有你开始编辑后才会改写文件。', btn('用源码编辑', 'sm', 'code') + btn('知道了', 'sm ghost')) + `<div class="md" style="padding:12px 34px;flex:1;overflow:hidden;opacity:.7"><p>正文……</p></div>`, 190, 560), '浏览不改文件;第一次输入才把缓冲区换成规范化后的文本;所见即所得里的 ⌘Z 用富文本自带的历史。')))}
    ${sec('行为规则', '', tbl([['项', 200], ['规则', 520], ['状态']], [
      ['默认视图', 'Markdown 文件 → 所见即所得;偏好 editor.mdView(0 所见即所得 / 1 源码);每个标签的手动切换优先于偏好', st('ok')],
      ['切换', '分段控件 + 命令面板「Markdown:切换所见即所得 / 源码」;切换时把文本从缓冲区重新读入,所以两边永远一致', st('ok')],
      ['保存', '所见即所得里每次修改都写回同一份缓冲区,⌘S / 自动保存与源码完全一样;未保存圆点照常', st('ok')],
      ['格式条', 'B / I / 删除线 / 行内代码 / 链接 / H1–H3 / 无序 · 有序 · 任务列表 / 引用 / 代码块;只读预览时隐藏', st('ok')],
      ['从搜索 / 引用 / 转到定义打开并定位到某一行', '需要行列坐标,所以自动切到源码再定位(标签上的分段同步变化)', st('ok')],
      ['含表格 / HTML / 脚注', '只读预览(会话流同款渲染)+ 提示条「用源码编辑」;不尝试有损往返', st('ok')],
      ['写法规范化提示', '粗略比对「解析再序列化」与原文,差异行数 ≥ 1 时提示;仍可编辑', st('ok')],
      ['撤销 / 重做', '所见即所得里 ⌘Z 撤销最近的格式 / 输入(富文本自带历史);切到源码后用缓冲区的历史', st('part')],
      ['图片 / 附件', '所见即所得里图片显示为替代文字占位;粘贴图片、拖入图片成附件', st('todo')],
      ['大纲', '左栏「大纲」对 Markdown 取标题;点击跳转会先切到源码再定位(所见即所得里还不能按标题滚动)', st('part')]]))}
    ${sec('自动换行与格式化', '两个动作都在顶部条右侧(换行钮有「已开」高亮态),也在右键菜单与命令面板里;快捷键 ⌥Z / ⇧⌥F。', grid(
      cell('顶部条', [], `<div class="snip" style="height:78px;width:560px">${bar(tools({ wrap: 1 }))}${bar(`${seg(['左右', '统一'], 0, 'sm')}${ib('wrap', 'sm', '自动换行  ⌥Z')}<span class="t4">差异</span>`)}</div>`, '上:文本标签(换行已开);下:差异标签的右侧同样有换行钮。换行是全局偏好 editor.wrap,开关即时作用于所有编辑器与差异标签。'),
      cell('格式化的三种结果', ['⇧', '⌥', 'F'], `<div class="fc g3">${fmtCase('有语言服务器', toast('okc', '已格式化(rust-analyzer)', 'c-ok'), '整篇交给服务器返回编辑,期间文件被改动则取消。')}${fmtCase('没有服务器,但有内置格式化器', toast('okc', '已格式化(内置 · Go)', 'c-ok'), 'Go(gofmt)、JSON(缩进整理)、Markdown(规范化)、其余文本:去行尾空白 + 末尾恰好一个换行。')}${fmtCase('已经是整齐的', toast('info', '已经是格式化好的', 'c-info'), '不产生改动,也不弄脏文件。')}</div>`, '一律先试服务器,再退内置;都不行才提示。「保存时格式化」打开后保存前自动走同一条路径。'),
      cell('设置里的对应项', [], box(`<div style="padding:10px 14px;display:flex;flex-direction:column;gap:10px"><div class="set-row">${'<span class="m grow">自动换行</span>'}${tog(0)}</div><div class="set-row"><span class="m grow">Markdown 默认视图</span>${seg(['所见即所得', '源码'], 0)}</div><div class="set-row"><span class="m grow">保存时格式化</span>${tog(0)}</div><div class="set-row"><span class="m grow">保存时去掉行尾空白</span>${tog(0)}</div></div>`, 190, 560), '设置 → 编辑器与语言服务;「Markdown 与格式化」是新分组,自动换行仍在「显示」。')))}
    ${sec('差异标签', '差异标签与编辑器共用一套顶部条:上一处 / 下一处、左右 · 统一、自动换行、打开文件。', tbl([['能力', 200], ['说明', 520], ['状态']], [
      ['自动换行', '跟随偏好 editor.wrap;顶部条有换行钮,点它改的是全局偏好', st('ok')],
      ['Markdown 文件的差异', '仍是源码级差异(逐行),不做渲染后的差异;「打开文件」按钮按所见即所得打开', st('ok')],
      ['可编辑一侧 / 块级暂存 / 词级高亮', '见「差异标签 · 块级暂存」', st('ok')],
      ['忽略空白差异', '库没有这个选项;待做', st('todo')]]))}
    ${sec('实现状态', '', tbl([['范围', 220], ['说明', 440], ['状态']], [
      ['Markdown 所见即所得 / 源码切换', '库自带富文本编辑器(Markdown 往返),格式条、YAML 头只读属性条、表格 / HTML / 脚注只读预览、写法规范化提示条都已实现;往返会丢掉代码块语言标注、并掉段内换行(提示条如实说明);图片 / 附件未做', st('part')],
      ['自动换行 / 格式化(顶部条 · 菜单 · 快捷键)', '服务器优先,内置格式化器兜底', st('ok')],
      ['保存时格式化 / 去行尾空白', '保存前走内置格式化器;语言服务器的异步格式化暂不参与(避免保存被拖慢);所见即所得里的 Markdown 保存前不整理(以文档为准,免得冲掉正在输入的光标)', st('part')]]))}</div>`;

  /* ───────── 拖拽引用与「···」菜单 ───────── */
  const chatDrop = `<div style="position:relative;height:100%;background:var(--bg);display:flex;flex-direction:column"><div style="padding:10px 14px;font-weight:700;border-bottom:1px solid var(--bd-1)">完整修复一个 Bug:路由大小写</div><div style="flex:1;padding:12px 14px;color:var(--tx-3)">……上一条回复……</div>
    <div style="position:absolute;inset:4px;border:2px dashed var(--ac);border-radius:var(--r-lg);background:var(--ac-soft);display:grid;place-items:center;pointer-events:none"><div class="pop" style="padding:8px 14px;display:flex;align-items:center;gap:8px">${ic('link', 's12 c-ac')}<b>松手引用</b><span class="mono">@src/router.rs</span></div></div></div>`;
  const dragCopy = (t, extra = '') => `<div style="position:absolute;${extra}display:inline-flex;align-items:center;gap:6px;height:24px;padding:0 10px;border-radius:var(--r-md);background:var(--raised);box-shadow:var(--sh-2);border:1px solid var(--ac-line);font-size:12px">${ic('code', 's12 c-ac')}${t}</div>`;
  const composerRef = `<div style="padding:14px"><div class="cmp" style="padding:10px 12px;border:1px solid var(--bd-2);border-radius:var(--r-lg);background:var(--raised)"><div class="mono" style="font-size:12.5px;line-height:1.6">看一下 <span class="ref">@src/router.rs</span> <span class="ref">@tests/router_test.rs</span> <span class="ref">@docs/</span><span class="cursor"></span></div></div></div>`;

  P['drag-more'] = () => `<div class="ds" style="max-width:1240px">${head('拖拽引用与「···」菜单盘点', '规则:① 文件、文件夹、编辑器标签、源代码管理里的文件都可以拖进对话栏,落下去 = 在输入框里插入 @ 引用(多次拖入累加,不会替换已写的字);② 访达里的文件同样能拖进来;③ 每个「···」按钮都必须有菜单,菜单只放「这个对象上真的能做的事」,顺序固定为「主要动作 → 导航 → 复制 / 访达 → 危险操作」;④ 同一动作在不同菜单里叫法、图标、快捷键一致(见「右键菜单与动作矩阵」);⑤ 菜单里置灰的项悬停说明原因,不隐藏。')}
    ${sec('拖拽到对话', '拖拽开始后元素的副本跟着指针;对话栏进入「可放」态:虚线描边 + 中央提示「松手引用 @路径」。落点只认对话栏(会话流 + 输入框),别处松手什么都不发生(Esc 取消)。', grid(
      cell('拖动中', [], box(`<div style="display:flex;height:100%"><div style="width:200px;padding:8px;border-right:1px solid var(--bd-1);font-size:12px"><div class="row" style="height:22px">${ic('chevD', 's12')}${ic('folder', 's12')}<span class="ell">src</span></div><div class="row on" style="height:22px;padding-left:20px">${ic('code', 's12')}<span class="ell">router.rs</span></div><div class="row" style="height:22px;padding-left:20px">${ic('code', 's12')}<span class="ell">main.rs</span></div></div><div style="flex:1;position:relative">${chatDrop}</div></div>${dragCopy('router.rs', 'left:120px;top:38px;')}`, 230, 560), '从文件树行或标签条的标签起拖;副本显示图标 + 文件名,多选时加数量角标。'),
      cell('落下后', [], box(`<div style="height:100%;display:flex;flex-direction:column;justify-content:flex-end">${composerRef}</div>`, 120, 560), '引用插在输入框末尾,与已有文字之间补一个空格;文件夹以 / 结尾;输入框获得焦点,可以接着写。'),
      cell('Agent 形态下的落点', [], box(`<div style="padding:14px;font-size:12px;color:var(--tx-2);line-height:1.8">• 对话栏居中占满时,整栏都是落点<br>• 工作区在右侧时,从它的标签拖到左边的对话栏<br>• IDE 形态下对话栏在右侧窄栏;对话栏被收起时没有落点,需要先 ⌥⌘B 打开(拖拽中不自动展开,避免误触)</div>`, 130, 560), '拖拽中布局不能变,否则用户抓不到目标,所以落点不会自动展开面板。')))}
    ${sec('什么能拖、拖出来是什么', '', tbl([['来源', 200], ['拖出来是', 260], ['落进对话栏得到', 260], ['状态']], [
      ['文件树 · 文件', '文件路径(相对项目根;多根工作区带文件夹名前缀)', '@路径', st('ok')],
      ['文件树 · 文件夹', '目录路径', '@目录/', st('ok')],
      ['编辑器标签 · 文件', '该文件', '@路径', st('ok')],
      ['编辑器标签 · 差异', '该差异对应的文件(不是补丁文本)', '@路径', st('ok')],
      ['源代码管理 · 文件行', '该文件', '@路径', st('ok')],
      ['访达 / 其它应用的文件', '绝对路径(项目外的文件用绝对路径引用)', '@/绝对/路径', st('ok')],
      ['编辑器里选中的代码', '选区', '用「问 Claude ⌘L」或选中工具条的加号,不走拖拽(选区拖拽已是文字移动)', st('ok')],
      ['图片文件', '', '以后改为作为图片附件而不是路径', st('todo')],
      ['把对话里的文件引用拖进编辑区', '', '等价于打开该文件', st('todo')]]))}
    ${sec('「···」菜单盘点', '下面逐个列出每个「···」入口应有的菜单。已有右键菜单的对象,「···」与右键菜单内容一致(只是触发方式不同)。', `<div class="gal" style="grid-template-columns:repeat(auto-fill,minmax(250px,1fr))">
      ${M('编辑器标签条 ···', [['x', '关闭已保存'], ['x', '关闭全部'], ['-'], ['split', '向右拆分', '⌘\\'], ['arrow', '在新窗口中打开', ''], ['-'], ['code', '默认以所见即所得打开 Markdown'], ['wrap', '自动换行', '⌥Z'], ['-'], ['clock', '重新打开已关闭的标签', '⇧⌘T'], ['copy', '复制所有标签路径']])}
      ${M('标签(右键)', [['x', '关闭', '⌘W'], ['x', '关闭其他'], ['x', '关闭右侧'], ['x', '关闭已保存'], ['-'], ['pin', '固定标签'], ['split', '向右拆分', '⌘\\'], ['-'], ['spark', '引用到对话'], ['copy', '复制路径'], ['copy', '复制相对路径'], ['folder', '在访达中显示'], ['tree', '在文件树中定位']])}
      ${M('文件(文件树右键)', [['code', '打开', '⏎'], ['split', '在侧边打开', '⌘⏎'], ['-'], ['spark', '引用到对话'], ['clock', '显示文件历史'], ['-'], ['copy', '复制路径'], ['copy', '复制相对路径'], ['folder', '在访达中显示']])}
      ${M('文件夹(文件树右键)', [['search', '在文件夹中搜索…'], ['terminal', '在此处打开终端'], ['-'], ['spark', '引用到对话'], ['-'], ['copy', '复制路径'], ['copy', '复制相对路径'], ['folder', '在访达中显示']])}
      ${M('对话栏头部 ···', [['edit', '重命名…'], ['pin', '置顶'], ['archive', '归档'], ['-'], ['copy', '复制会话 ID'], ['copy', '复制标题'], ['copy', '复制为 Markdown'], ['link', '复制会话链接'], ['-'], ['fork', '分叉会话…'], ['history', '历史会话', '⇧⌘H']])}
      ${M('会话行 ···(导航栏)', [['play', '打开', '⏎'], ['edit', '重命名…'], ['pin', '置顶'], ['archive', '归档'], ['-'], ['copy', '复制会话 ID'], ['copy', '复制标题'], ['copy', '复制为 Markdown'], ['-'], ['trash', '删除会话…', '', 'danger']])}
      ${M('Markdown 标签 ···(同标签条,附加)', [['code', '切换所见即所得 / 源码'], ['eye', '只读预览'], ['-'], ['align', '格式化文档', '⇧⌥F'], ['wrap', '自动换行', '⌥Z']])}
      ${M('差异标签 ···', [['wrap', '自动换行', '⌥Z'], ['swap', '切换左右 / 统一'], ['-'], ['code', '打开文件'], ['copy', '复制路径'], ['clock', '显示文件历史']])}
      ${M('SCM 标题 ···', [['arrow', '拉取'], ['arrow', '推送'], ['sync', '同步'], ['-'], ['layers', '储藏所有更改'], ['layers', '弹出最近的储藏'], ['-'], ['list', '切换列表 / 树'], ['refresh', '刷新']])}
      ${M('文件树标题 ···', [['plus', '新建文件…'], ['folder', '新建文件夹…'], ['-'], ['refresh', '刷新'], ['tree', '全部折叠'], ['-'], ['folder', '在访达中显示项目']])}
      ${M('终端标签条 ···', [['trash', '清屏', '⌘K'], ['search', '查找…', '⌘F'], ['-'], ['plus', '新建终端'], ['x', '关闭终端'], ['-'], ['gear', '终端设置…']])}
      ${M('自定义详情 ···', [['copy', '复制名称'], ['folder', '在访达中显示文件'], ['edit', '用编辑器打开'], ['-'], ['trash', '删除…', '', 'danger']])}
    </div>`)}
    ${sec('文件操作 · 新建 / 重命名 / 删除', '会改磁盘的三个动作。入口:文件树标题「···」/ 文件夹右键(新建)、文件与文件夹右键(重命名 F2、删除 ⌘⌫)。输入走文件树里的行内输入框,不弹对话框;删除是「移到废纸篓」,走确认框。', grid(
      cell('新建 · 行内输入', '', box(`<div style="padding:8px"><div class="row" style="height:22px">${ic('chevD', 's12')}${ic('folder', 's12')}<span class="grow">src</span></div>
        <div class="row on" style="height:24px;padding-left:30px">${ic('file', 's12')}<div class="inp focus grow" style="height:20px"><span class="v caret">handlers.rs</span></div></div>
        <div class="row" style="height:22px;padding-left:30px">${ic('code', 's12')}<span class="grow">main.rs</span></div>
        <div class="row" style="height:22px;padding-left:30px">${ic('code', 's12')}<span class="grow">router.rs</span></div>
        <div class="t3 fs-cap" style="padding:8px 4px;line-height:1.7">· 在文件夹里新建:文件夹自动展开,输入行插在它的子项最前;在文件上右键「新建文件…」= 建在同级<br>· ⏎ 创建并(文件)在编辑器打开、树里选中;Esc / 点别处取消<br>· 输入 <span class="mono">a/b.rs</span> 会连同中间的文件夹一起建</div></div>`, 230, 330), '创建失败(已存在 / 没权限)不关闭输入行,行下方显示红色原因,可改名重试。'),
      cell('重命名 · 行内输入', '', box(`<div style="padding:8px"><div class="row" style="height:22px;padding-left:30px">${ic('code', 's12')}<span class="grow">main.rs</span></div>
        <div class="row on" style="height:24px;padding-left:30px">${ic('code', 's12')}<div class="inp focus grow" style="height:20px"><span class="v"><span style="background:var(--ac-soft)">router</span>.rs</span></div></div>
        <div class="err-line fs-cap" style="padding:2px 6px 0 52px;color:var(--err)">已存在同名文件「router.rs」</div>
        <div class="t3 fs-cap" style="padding:8px 4px;line-height:1.7">· F2 / 菜单进入,文件只选中不含扩展名的部分<br>· 名字不能为空、不能含 <span class="mono">/</span>、不能与同级重名<br>· 已打开的标签、最近文件、编辑器里的路径同步改名;有未保存修改的标签保留修改</div></div>`, 230, 330), '重命名不改文件内容,只改名;Git 会把它当作「删除 + 新增」,SCM 里的改名检测由 git 自己做。'),
      cell('删除 · 确认', '', box(`<div style="padding:12px 14px" class="fc g3"><div class="f ai g3"><span class="dlg-i err">${ic('trash')}</span><b class="fs-title">把 handlers.rs 移到废纸篓?</b></div><div class="t3" style="line-height:1.55">可以在访达的废纸篓里放回原处。已打开的标签会关闭;有未保存修改的标签会先询问。</div><div class="f g2" style="justify-content:flex-end">${btn('取消', 'ghost')}${btn('移到废纸篓', 'danger')}</div>
        <div class="sep"></div><div class="t3 fs-cap" style="line-height:1.7">· 文件夹:写明「连同里面的 N 个文件」<br>· 工作区顶层文件夹没有这个菜单项(用「从工作区移出」)<br>· 演示模式没有废纸篓:直接从内存项目里移除,提示「演示数据」</div></div>`, 230, 330), '真实模式经 Finder(osascript)移到废纸篓以便「放回原处」;失败给出原因,不退回永久删除。'),
      cell('菜单(补全后)', '', box(menu(220, [['plus', '新建文件…'], ['folder', '新建文件夹…'], ['-'], ['edit', '重命名…', 'F2'], ['trash', '移到废纸篓…', '⌘⌫', 'danger']]) + `<div class="t3 fs-cap" style="padding:8px 4px;line-height:1.7">文件、文件夹右键与文件树标题 ··· 都用同一组;位置按「主要动作 → 导航 → 复制 / 访达 → 危险操作」放在原有项之间(新建在最前,重命名与删除在复制路径之前)。</div>`, 230, 330), '多根时顶层文件夹行只有「新建…」。')))}
    ${sec('实现状态', '', tbl([['范围', 240], ['说明', 440], ['状态']], [
      ['文件操作 · 新建 / 重命名 / 移到废纸篓', '文件树行内输入(新建文件 / 文件夹、重命名,重名 / 非法名行下报错,Esc / 点别处取消)、移到废纸篓确认框(文件夹写明文件数、有未保存修改会提示)、重命名后标签路径同步且保留修改;真实模式经 Finder 移到废纸篓,演示是内存项目;已知缺口:F2 / ⌘⌫ 快捷键未接(F2 与编辑器「重命名符号」冲突)、重命名时全选而非只选不含扩展名的部分、跨文件夹拖动移动文件', st('part')],
      ['拖拽到对话(文件树 / 标签 / SCM / 访达)', '落点高亮 + 插入 @ 引用 + 聚焦输入框', st('ok')],
      ['编辑器标签条 ··· 与标签右键菜单', '关闭 / 其他 / 右侧 / 已保存 / 全部、引用到对话、复制路径、在访达中显示、在文件树中定位、Markdown 默认视图、自动换行、⇧⌘T 重开(最近 20 个);向右拆分、固定标签、在新窗口中打开 没有实现,菜单里暂不放', st('part')],
      ['文件 / 文件夹右键菜单、文件树标题 ···', '文件:打开 / 引用到对话 / 显示文件历史 / 复制路径 / 访达;文件夹:在文件夹中搜索 / 在此处打开终端 / 引用;标题 ···:刷新 / 全部折叠 / 在访达中显示项目。新建 / 重命名 / 移到废纸篓已实现(见上方「文件操作」一节);「在侧边打开」没有实现', st('part')],
      ['对话栏头部 ··· 与会话行 ···(右键同)', '打开 / 重命名 / 置顶 / 归档 / 复制会话 ID / 复制标题 / 复制为 Markdown / 删除会话 / 历史会话 已做(整理信息存本机 session-meta.json);「复制会话链接」「分叉会话…」后端未接,置灰并写明原因', st('part')],
      ['差异标签 ···、SCM 标题 ···、终端标签条 ···、自定义详情 ···', '本页已定稿,实现排在后面', st('todo')]]))}</div>`;
})();
