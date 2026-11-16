/* 设计稿 v3.4 · Markdown 预览与格式化、拖拽引用与「···」菜单盘点
   依据:Typora / Obsidian 实时预览 / Notion(预览里直接改,有源码开关)、VS Code(Markdown 预览与编辑分开,所以不照搬)、
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

  /* ───────── Markdown 预览 ───────── */
  /** 编辑器顶部的面包屑条:左边路径,右边是视图切换 / 换行 / 格式化 / 语言 / 编码 */
  const bar = (right, cls = '') => `<div class="${cls}" style="height:26px;display:flex;align-items:center;gap:6px;padding:0 12px 0 14px;color:var(--tx-3);font-size:11.5px;flex:none;border-bottom:1px solid var(--bd-1)">docs${ic('chevR', 's12')}<b style="color:var(--tx-1);font-weight:500">README.md</b><span style="margin-left:auto;display:flex;gap:6px;align-items:center">${right}</span></div>`;
  const tools = (o = {}) => `${seg(['预览', '源码'], o.src ? 1 : 0, 'sm')}${ib('wrap', o.wrap ? 'sm on' : 'sm', o.wrap ? '自动换行(已开)  ⌥Z' : '自动换行  ⌥Z')}${ib('align', 'sm', '格式化文档  ⇧⌥F')}<span class="t4">Markdown</span><span class="t4">UTF-8</span>`;
  /** 预览的格式条:B I S code link | 标题 | 列表 任务 引用 代码块 */
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

  P['ide-md'] = () => `<div class="ds" style="max-width:1240px">${head('Markdown · 预览与格式化', '规则:① Markdown 文件(.md / .markdown / .mdc)默认以「预览」打开——直接在渲染后的文档里改,不是左源码右预览;② 标签右上角的分段「预览 | 源码」随时切换,两边读写的是同一份文本,切换不丢光标之外的任何状态;③ 表格在预览里就地编辑(文档按块拆成「正文 / 表格」若干段,表格段是可编辑网格,单元格里的内容按 Markdown 源码原样保留);HTML、脚注、嵌套在列表里的表格仍无法无损表示,不冒险改写:这类文件以只读预览打开并提示「用源码编辑」;④ 自动换行与格式化是所有文本编辑器共有的能力,放在顶部条右侧,不藏进菜单。')}
    ${sec('Markdown 标签', '默认视图 = 预览;设置 → 编辑器与语言服务 → 「Markdown 默认视图」可改成源码,标签上手动切换的选择优先于偏好(只在该标签开着时记住,关掉即忘)。', grid(
      cell('预览(默认)', [], box(tabsMd('README.md') + bar(tools()) + mdbody, 370, 560), '预览不再显示格式条(B I S / 标题 / 列表那一排),界面只留渲染后的文档;⌘B / ⌘I / ⌘K 与 Markdown 快捷输入(# 空格、- 空格、``` )照常生效。'),
      cell('源码', [], box(tabsMd('README.md') + bar(tools({ src: 1, wrap: 1 })) + srcBody, 370, 560), '源码视图就是普通代码编辑器:行号、查找替换、迷你地图、自动换行、格式化、Git 竖条全有。'),
      cell('表格:就地编辑', [], box(tabsMd('README.md') + bar(tools()) + `<div class="md" style="padding:12px 34px;flex:1;overflow:hidden"><div style="font-size:15px;font-weight:650;margin:0 0 6px">接口</div><div style="border:1px solid var(--bd-2);border-radius:6px;overflow:hidden;font-size:12px"><div style="display:flex;background:var(--fill-1);font-weight:650"><div style="flex:1;padding:5px 8px;border-right:1px solid var(--bd-1)">方法</div><div style="flex:1;padding:5px 8px;border-right:1px solid var(--bd-1)">路径</div><div style="width:26px"></div></div><div style="display:flex;border-top:1px solid var(--bd-1)"><div style="flex:1;padding:5px 8px;border-right:1px solid var(--bd-1);outline:2px solid var(--ac);outline-offset:-2px">GET</div><div style="flex:1;padding:5px 8px;border-right:1px solid var(--bd-1)">/users</div><div style="width:26px;display:flex;align-items:center;justify-content:center">${ic('x', 's12 c-t4')}</div></div><div style="display:flex;border-top:1px solid var(--bd-1)"><div style="flex:1;padding:5px 8px;border-right:1px solid var(--bd-1)">POST</div><div style="flex:1;padding:5px 8px;border-right:1px solid var(--bd-1)">/users</div><div style="width:26px"></div></div></div><div style="display:flex;gap:6px;margin-top:6px">${btn('添加行', 'sm ghost', 'plus')}${btn('添加列', 'sm ghost', 'plus')}</div></div>`, 300, 560), '表格段是单独的可编辑网格:单元格直接输入(按 Markdown 源码原样,**粗体**、`代码`、链接不被改写),Tab / ⇧Tab 在格子间走,⏎ 跳到下一行同列(最后一行再按 ⏎ 新增一行),悬停行出现「×」删行,网格下方「添加行 / 添加列」,右键单元格有 上方 / 下方插入行、左 / 右插入列、删除行 / 列、列对齐。写回只重写这一张表(第一次编辑才把列宽对齐规范化),表格前后的正文原样不动。'),
      cell('含 HTML / 脚注:只读预览', [], box(tabsMd('README.md') + bar(tools()) + notice('warn', '这个文件含 HTML,预览里编辑会把它压成普通段落,所以只读显示。', btn('用源码编辑', 'sm', 'code')) + `<div class="md" style="padding:12px 34px;flex:1;overflow:hidden"><div style="font-size:15px;font-weight:650;margin:0 0 6px">接口</div><table style="border-collapse:collapse;font-size:12px;width:100%"><tr style="background:var(--fill-1)"><th style="text-align:left;padding:4px 8px;border:1px solid var(--bd-1)">方法</th><th style="text-align:left;padding:4px 8px;border:1px solid var(--bd-1)">路径</th></tr><tr><td style="padding:4px 8px;border:1px solid var(--bd-1)">GET</td><td style="padding:4px 8px;border:1px solid var(--bd-1)">/users</td></tr><tr><td style="padding:4px 8px;border:1px solid var(--bd-1)">POST</td><td style="padding:4px 8px;border:1px solid var(--bd-1)">/users</td></tr></table></div>`, 300, 560), '只读预览走会话流同一套渲染(表格、代码块、链接都完整);点链接 / 路径照常跳转。想改就切到源码。只剩 HTML、脚注与嵌套在列表 / 引用里的表格会走这一步。'),
      cell('写法会被规范化:提示条', [], box(tabsMd('README.md') + bar(tools()) + notice('info', '有 3 行的写法(如代码块语言标注、段内换行)在预览里会被规范化。只有你开始编辑后才会改写文件。', btn('用源码编辑', 'sm', 'code') + btn('知道了', 'sm ghost')) + `<div class="md" style="padding:12px 34px;flex:1;overflow:hidden;opacity:.7"><p>正文……</p></div>`, 190, 560), '浏览不改文件;第一次输入才把缓冲区换成规范化后的文本;预览里的 ⌘Z 用富文本自带的历史。')))}
    ${sec('行为规则', '', tbl([['项', 200], ['规则', 520], ['状态']], [
      ['默认视图', 'Markdown 文件 → 预览;偏好 editor.mdView(0 预览 / 1 源码);每个标签的手动切换优先于偏好', st('ok')],
      ['切换', '分段控件 + 命令面板「Markdown:切换预览 / 源码」;切换时把文本从缓冲区重新读入,所以两边永远一致', st('ok')],
      ['保存', '预览里每次修改都写回同一份缓冲区,⌘S / 自动保存与源码完全一样;未保存圆点照常', st('ok')],
      ['格式条', 'B / I / 删除线 / 行内代码 / 链接 / H1–H3 / 无序 · 有序 · 任务列表 / 引用 / 代码块;只读预览时隐藏', st('ok')],
      ['从搜索 / 引用 / 转到定义打开并定位到某一行', '需要行列坐标,所以自动切到源码再定位(标签上的分段同步变化)', st('ok')],
      ['表格(顶层)', '预览里是可编辑网格:按块拆成「正文 / 表格」段,每段独立写回;格内按 Markdown 源码原样;Tab / ⇧Tab / ⏎ 移动,增删行列、列对齐;未编辑的表保持原字节。已知限制:光标不能用方向键从正文穿进表格、选区不能跨段、撤销按「写回缓冲区」整体走(未实机验证)', st('part')],
      ['HTML / 脚注 / 嵌套表格', '只读预览(会话流同款渲染)+ 提示条「用源码编辑」;不尝试有损往返', st('ok')],
      ['写法规范化提示', '粗略比对「解析再序列化」与原文,差异行数 ≥ 1 时提示;仍可编辑', st('ok')],
      ['撤销 / 重做', '预览 ⌘Z 走富文本历史;切源码走缓冲区;切换视图 BreakUndoCoalescing', st('ok')],
      ['图片 / 附件', '相对路径 LoadImage 预览;拖入保存到 editor.mdAssets(默认 docs/assets)并插入语法;对话拖入为 ![]() 附件', st('ok')],
      ['大纲', 'Markdown 标题点击在预览里滚动定位,不强制切源码', st('ok')]]))}
    ${sec('自动换行与格式化', '两个动作都在顶部条右侧(换行钮有「已开」高亮态),也在右键菜单与命令面板里;快捷键 ⌥Z / ⇧⌥F。', grid(
      cell('顶部条', [], `<div class="snip" style="height:78px;width:560px">${bar(tools({ wrap: 1 }))}${bar(`${seg(['左右', '统一'], 0, 'sm')}${ib('wrap', 'sm', '自动换行  ⌥Z')}<span class="t4">差异</span>`)}</div>`, '上:文本标签(换行已开);下:差异标签的右侧同样有换行钮。换行是全局偏好 editor.wrap,开关即时作用于所有编辑器与差异标签。'),
      cell('格式化的三种结果', ['⇧', '⌥', 'F'], `<div class="fc g3">${fmtCase('有语言服务器', toast('okc', '已格式化(rust-analyzer)', 'c-ok'), '整篇交给服务器返回编辑,期间文件被改动则取消。')}${fmtCase('没有服务器,但有内置格式化器', toast('okc', '已格式化(内置 · Go)', 'c-ok'), 'Go(gofmt)、JSON(缩进整理)、Markdown(规范化)、其余文本:去行尾空白 + 末尾恰好一个换行。')}${fmtCase('已经是整齐的', toast('info', '已经是格式化好的', 'c-info'), '不产生改动,也不弄脏文件。')}</div>`, '一律先试服务器,再退内置;都不行才提示。「保存时格式化」打开后保存前自动走同一条路径。'),
      cell('设置里的对应项', [], box(`<div style="padding:10px 14px;display:flex;flex-direction:column;gap:10px"><div class="set-row">${'<span class="m grow">自动换行</span>'}${tog(0)}</div><div class="set-row"><span class="m grow">Markdown 默认视图</span>${seg(['预览', '源码'], 0)}</div><div class="set-row"><span class="m grow">保存时格式化</span>${tog(0)}</div><div class="set-row"><span class="m grow">保存时去掉行尾空白</span>${tog(0)}</div></div>`, 190, 560), '设置 → 编辑器与语言服务;「Markdown 与格式化」是新分组,自动换行仍在「显示」。')))}
    ${sec('写回策略', '预览里编辑触发写回时,**只替换被改动的块**(段落 / 标题 / 列表项 / 代码块围栏);缓冲区里其余字节原样保留。整篇规范化只发生在用户点「格式化文档」或显式切源码后走格式化时,不由预览随便触发。', grid(
      cell('并排对比', '', `<div class="grid2" style="gap:10px"><div class="spec" style="padding:10px 12px"><div class="cap2"><b>编辑前(磁盘)</b></div><pre class="mono t3" style="margin:0;white-space:pre-wrap;line-height:1.55"># 快速开始\n\n第一段:设计稿先行。\n\n第二段:先改<strong>这一</strong>段。\n\n第三段:段内硬换行\n仍在本段。\n\n\`\`\`rust\nfn main() {}\n\`\`\`\n\n- 用 * 列表</pre></div><div class="spec" style="padding:10px 12px"><div class="cap2"><b>只改第二段后写回</b>${tag('块 2', 'ac')}</div><pre class="mono t3" style="margin:0;white-space:pre-wrap;line-height:1.55"># 快速开始\n\n第一段:设计稿先行。 <span class="tag ok" style="font-size:10px">原样</span>\n\n第二段:先改<b>预览里</b>这一段。\n\n第三段:段内硬换行\n仍在本段。 <span class="tag ok" style="font-size:10px">原样</span>\n\n\`\`\`rust\nfn main() {}\n\`\`\` <span class="tag ok" style="font-size:10px">原样</span>\n\n- 用 * 列表 <span class="tag ok" style="font-size:10px">原样</span></pre></div></div>`, '未改段落:空行数量、段内硬换行、列表符(* vs -)、代码块语言行均不动;只替换光标所在块的 Markdown 源码区间。'),
      cell('什么算「被编辑过」', '', tbl([['块类型', 160], ['写回范围', 520]], [
        ['段落 / 标题 / 引用', '该块对应的 Markdown 行区间(到下一个空行或同级标题为止)'],
        ['列表项', '单条 <span class="mono">-</span> / <span class="mono">*</span> / <span class="mono">1.</span> 行及其续行缩进'],
        ['代码块', '整段围栏 <span class="mono">```</span>…<span class="mono">```</span>(含语言标注行)'],
        ['任务列表', '单条 <span class="mono">- [ ]</span> 行'],
        ['跨块选区', '从块 A 首到块 B 尾合并写回(仍不碰中间未选中的块)'],
        ['整篇规范化', '仅 ⇧⌥F / 保存时格式化(若开)或用户确认「规范化预览」后的一次性全量替换']]), '浏览与只读预览不写盘;「写法规范化提示条」逻辑保持不变。')))}
    ${sec('差异标签', '差异标签与编辑器共用一套顶部条:上一处 / 下一处、左右 · 统一、自动换行、打开文件。', tbl([['能力', 200], ['说明', 520], ['状态']], [
      ['自动换行', '跟随偏好 editor.wrap;顶部条有换行钮,点它改的是全局偏好', st('ok')],
      ['Markdown 文件的差异', '仍是源码级差异(逐行),不做渲染后的差异;「打开文件」按钮按预览打开', st('ok')],
      ['可编辑一侧 / 块级暂存 / 词级高亮', '见「差异标签 · 块级暂存」', st('ok')],
      ['忽略空白差异', '差异顶部条「忽略空白」开关;比较前归一化空白,显示仍用原文', st('ok')]]))}
    ${sec('非文本文件 · 图片与表格', '规则:常规编辑器能打开的文件这里都能打开,不再一律提示「二进制文件」。按扩展名分流:图片 → 图片查看器;csv / tsv / xlsx → 只读表格查看器;其余二进制 → 提示条 +「在访达中显示」「用默认应用打开」两个出口。这些标签与文本标签同样可固定 / 预览 / 拆分,不进「未保存」,不接语言服务与 Git 行竖条。', grid(
      cell('图片查看器', [], box(tabsMd('logo.png') + bar(`<span class="t3 fs-cap">PNG · 512 × 512 · 38 KB</span><span class="grow"></span>${ib('minus', 'sm', '缩小')}<span class="t3 mono fs-cap">适应窗口</span>${ib('plus', 'sm', '放大')}${ib('folder', 'sm', '在访达中显示')}`) + `<div style="flex:1;display:grid;place-items:center;background:var(--fill-1)"><div style="width:120px;height:120px;border-radius:24px;background:var(--ac-soft);border:1px solid var(--bd-2)"></div></div>`, 300, 560), '默认「适应窗口」居中,小图不放大;浅灰底色衬出透明通道(Go 实现未做棋盘格);顶部条显示格式 / 像素 / 大小,工具:缩小、放大、在访达中显示;解码在后台,完成前显示「正在载入…」。支持 PNG / JPEG / GIF(首帧)/ WebP / BMP / TIFF / SVG;超过 64 MB 或解码失败给提示条。'),
      cell('表格查看器(csv · tsv · xlsx)', [], box(tabsMd('sales.xlsx') + bar(`${seg(['Q1', 'Q2', 'Q3'], 0, 'sm')}<span class="grow"></span><span class="t3 fs-cap">120 行 × 6 列</span>`) + `<div class="mono fs-cap" style="flex:1;overflow:hidden"><div class="row" style="height:24px;background:var(--fill-1);border-bottom:1px solid var(--bd-1);font-weight:650"><span style="width:40px;text-align:right;padding-right:8px" class="t4">#</span><span style="width:120px">地区</span><span style="width:120px">产品</span><span style="width:100px">数量</span></div><div class="row" style="height:24px"><span style="width:40px;text-align:right;padding-right:8px" class="t4">1</span><span style="width:120px">华东</span><span style="width:120px">Mini</span><span style="width:100px">42</span></div></div>`, 300, 560), '只读:首行默认当表头(顶部条「首行作表头」开关可关),行号列在左,表头与表体用虚拟列表,列宽可拖;列宽按前 200 行内容估算(60–320px),单元格单行截断;xlsx 多工作表在顶部条用分段切换;最多显示 5000 行 × 100 列,超出时顶部条写明「仅显示前 N 行」。CSV 自动识别逗号 / 分号 / 制表符、去 BOM、引号与转义;GBK 不识别时提示「编码无法识别」。'),
      cell('其他二进制文件', [], box(tabsMd('app.bin') + `<div style="flex:1;display:grid;place-items:center"><div style="text-align:center;display:flex;flex-direction:column;gap:10px;align-items:center">${ic('file', 's20 c-t3')}<div class="t3">这是二进制文件,不能用文本方式打开</div><div class="f g2">${btn('在访达中显示', 'sm', 'folder')}${btn('用默认应用打开', 'sm', 'ext')}</div></div></div>`, 220, 560), '原来只有一句灰字没有出口;现在补两个按钮。文件太大(> 8 MB 的文本)同理,文案改「文件太大,不适合在编辑器里打开」。')))}
    ${sec('实现状态', '', tbl([['范围', 220], ['说明', 440], ['状态']], [
      ['Markdown 预览 / 源码切换', '预览排版对齐设计稿 .md(正文 13px / 行高 1.7、H1 21 · H2 15 · H3 14、行内代码取强调色、列表符 tx-4、任务框成功色且完成项不加删除线、代码块恢复正文色、左右留白 34px;`rtTheme` / `rtSpans`);格式条(默认隐藏)/ YAML 头 / 只读预览 / 拖入与 ⌘V 粘贴 PNG·JPEG 到 editor.mdAssets;预览写回只重写被编辑的块(未改段落的段内换行、列表符、围栏语言原样保留,增删块 LCS 对齐,对不齐的局部退回整段序列化);规范化提示条只统计会被改写的块。缺口:跨多块选区的合并写回、匹配率 <50% 时退回整篇往返;未测试', st('part')],
      ['自动换行 / 格式化(顶部条 · 菜单 · 快捷键)', '服务器优先,内置格式化器兜底', st('ok')],
      ['保存时格式化 / 去行尾空白', '源码保存可走内置格式化;WYSIWYG 保存不整理(以免冲光标);LSP 异步不参与保存', st('ok')],
      ['非文本文件:图片查看器 / csv · tsv · xlsx 只读表格 / 二进制出口', 'ide_viewer.go + table 解析(internal/tabular,xlsx 用标准库 zip + xml);未测试', st('part')],
      ['⌘V 粘贴 JPEG 图片', 'edit_md_paste.go:先读 PNG、读不到再读 image/jpeg 存成 .jpg;去重文件名、超过 10 MB 提示、预览与源码视图都可粘贴、可撤销;未测试', st('ok')]]))}</div>`;

  /* ───────── 拖拽引用与「···」菜单 ───────── */
  const chatDrop = `<div style="position:relative;height:100%;background:var(--bg);display:flex;flex-direction:column"><div style="padding:10px 14px;font-weight:700;border-bottom:1px solid var(--bd-1)">完整修复一个 Bug:路由大小写</div><div style="flex:1;padding:12px 14px;color:var(--tx-3)">……上一条回复……</div>
    <div style="position:absolute;inset:4px;border:2px dashed var(--ac);border-radius:var(--r-lg);background:var(--ac-soft);display:grid;place-items:center;pointer-events:none"><div class="pop" style="padding:8px 14px;display:flex;align-items:center;gap:8px">${ic('link', 's12 c-ac')}<b>松手引用</b><span class="mono">@src/router.rs</span></div></div></div>`;
  const dragCopy = (t, extra = '') => `<div style="position:absolute;${extra}display:inline-flex;align-items:center;gap:6px;height:24px;padding:0 10px;border-radius:var(--r-md);background:var(--raised);box-shadow:var(--sh-2);border:1px solid var(--ac-line);font-size:12px">${ic('code', 's12 c-ac')}${t}</div>`;
  const composerRef = `<div style="padding:14px"><div class="cmp" style="padding:8px 12px 10px;border:1px solid var(--bd-2);border-radius:var(--r-lg);background:var(--raised)"><div class="f g2 wrap" style="margin-bottom:6px"><span class="ref" style="height:22px;padding:0 3px 0 7px">${ic('code', 's12')}router.rs${ib('x', 'sm', '移除引用')}</span><span class="ref" style="height:22px;padding:0 3px 0 7px">${ic('code', 's12')}router_test.rs${ib('x', 'sm', '移除引用')}</span><span class="ref" style="height:22px;padding:0 3px 0 7px">${ic('folder', 's12')}docs${ib('x', 'sm', '移除引用')}</span></div><div style="font-size:13px;line-height:1.6">看一下这几个文件<span class="cursor"></span></div></div></div>`;

  P['drag-more'] = () => `<div class="ds" style="max-width:1240px">${head('拖拽引用与「···」菜单盘点', '规则:① 文件、文件夹、编辑器标签、源代码管理里的文件都可以拖进对话栏,落下去 = 输入框顶部多一枚引用胶囊(.ref 样式:类型图标 + 文件名 + ×;多次拖入累加,不会动已写的字;正文里不再重复写一段 @路径,发送时才按「@路径」放在消息最前面);② 访达里的文件同样能拖进来;③ 每个「···」按钮都必须有菜单,菜单只放「这个对象上真的能做的事」,顺序固定为「主要动作 → 导航 → 复制 / 访达 → 危险操作」;④ 同一动作在不同菜单里叫法、图标、快捷键一致(见「右键菜单与动作矩阵」);⑤ 菜单里置灰的项悬停说明原因,不隐藏。')}
    ${sec('拖拽到对话', '拖拽开始后元素的副本跟着指针;对话栏进入「可放」态:虚线描边 + 中央提示「松手引用 @路径」。落点只认对话栏(会话流 + 输入框),别处松手什么都不发生(Esc 取消)。', grid(
      cell('拖动中', [], box(`<div style="display:flex;height:100%"><div style="width:200px;padding:8px;border-right:1px solid var(--bd-1);font-size:12px"><div class="row" style="height:22px">${ic('chevD', 's12')}${ic('folder', 's12')}<span class="ell">src</span></div><div class="row on" style="height:22px;padding-left:20px">${ic('code', 's12')}<span class="ell">router.rs</span></div><div class="row" style="height:22px;padding-left:20px">${ic('code', 's12')}<span class="ell">main.rs</span></div></div><div style="flex:1;position:relative">${chatDrop}</div></div>${dragCopy('router.rs', 'left:120px;top:38px;')}`, 230, 560), '从文件树行或标签条的标签起拖;副本显示图标 + 文件名,多选时加数量角标。'),
      cell('落下后', [], box(`<div style="height:100%;display:flex;flex-direction:column;justify-content:flex-end">${composerRef}</div>`, 120, 560), '引用以胶囊排在输入框顶部(同一文件不重复;悬停显示完整路径,点胶囊打开文件,点 × 移除),正文原样不动、输入框获得焦点可以接着写;文件夹胶囊用文件夹图标,发送时路径以 / 结尾。发出去后消息里的引用也是同样的 .ref 胶囊。'),
      cell('Agent 形态下的落点', [], box(`<div style="padding:14px;font-size:12px;color:var(--tx-2);line-height:1.8">• 对话栏居中占满时,整栏都是落点<br>• 工作区在右侧时,从它的标签拖到左边的对话栏<br>• IDE 形态下对话栏在右侧窄栏;对话栏被收起时没有落点,需要先 ⌥⌘B 打开(拖拽中不自动展开,避免误触)</div>`, 130, 560), '拖拽中布局不能变,否则用户抓不到目标,所以落点不会自动展开面板。')))}
    ${sec('什么能拖、拖出来是什么', '', tbl([['来源', 200], ['拖出来是', 260], ['落进对话栏得到', 260], ['状态']], [
      ['文件树 · 文件', '文件路径(相对项目根;多根工作区带文件夹名前缀)', '@路径', st('ok')],
      ['文件树 · 文件夹', '目录路径', '@目录/', st('ok')],
      ['编辑器标签 · 文件', '该文件', '@路径', st('ok')],
      ['编辑器标签 · 差异', '该差异对应的文件(不是补丁文本)', '@路径', st('ok')],
      ['源代码管理 · 文件行', '该文件', '@路径', st('ok')],
      ['访达 / 其它应用的文件', '绝对路径(项目外的文件用绝对路径引用)', '@/绝对/路径', st('ok')],
      ['编辑器里选中的代码', '选区', '用「问 Claude ⌘L」或选中工具条的加号,不走拖拽(选区拖拽已是文字移动)', st('ok')],
      ['图片文件', '', '拖到对话栏或 Markdown:复制进工作区 assets 并插入 ![]() 语法', st('ok')],
      ['把对话里的文件引用拖进编辑区', '', '拖到代码区 = 打开该文件', st('ok')]]))}
    ${sec('「···」菜单盘点', '下面逐个列出每个「···」入口应有的菜单。已有右键菜单的对象,「···」与右键菜单内容一致(只是触发方式不同)。', `<div class="gal" style="grid-template-columns:repeat(auto-fill,minmax(250px,1fr))">
      ${M('编辑器标签条 ···', [['x', '关闭已保存'], ['x', '关闭全部'], ['-'], ['split', '向右拆分', '⌘\\'], ['split', '向下拆分'], ['-'], ['code', '默认以预览打开 Markdown'], ['wrap', '自动换行', '⌥Z'], ['-'], ['clock', '重新打开已关闭的标签', '⇧⌘T'], ['copy', '复制所有标签路径']])}
      ${M('标签(右键)', [['x', '关闭', '⌘W'], ['x', '关闭其他'], ['x', '关闭右侧'], ['x', '关闭已保存'], ['-'], ['pin', '固定标签'], ['split', '向右拆分', '⌘\\'], ['split', '向下拆分'], ['split', '在侧边打开', '⌘⏎'], ['eye', '在侧边预览'], ['-'], ['spark', '引用到对话'], ['copy', '复制路径'], ['copy', '复制相对路径'], ['folder', '在访达中显示'], ['tree', '在文件树中定位'], ['arrow', '在新窗口中打开']])}
      ${M('文件(文件树右键)', [['code', '打开', '⏎'], ['split', '在侧边打开', '⌘⏎'], ['-'], ['spark', '引用到对话'], ['clock', '显示文件历史'], ['-'], ['copy', '复制路径'], ['copy', '复制相对路径'], ['folder', '在访达中显示']])}
      ${M('文件夹(文件树右键)', [['search', '在文件夹中搜索…'], ['terminal', '在此处打开终端'], ['-'], ['spark', '引用到对话'], ['-'], ['copy', '复制路径'], ['copy', '复制相对路径'], ['folder', '在访达中显示']])}
      ${M('对话栏头部 ···', [['edit', '重命名…'], ['pin', '置顶'], ['archive', '归档'], ['-'], ['copy', '复制会话 ID'], ['copy', '复制标题'], ['copy', '复制为 Markdown'], ['link', '复制会话链接'], ['-'], ['fork', '分叉会话…'], ['history', '历史会话', '⇧⌘H']])}
      ${M('会话行 ···(导航栏)', [['play', '打开', '⏎'], ['edit', '重命名…'], ['pin', '置顶'], ['archive', '归档'], ['-'], ['copy', '复制会话 ID'], ['copy', '复制标题'], ['copy', '复制为 Markdown'], ['-'], ['trash', '删除会话…', '', 'danger']])}
      ${M('Markdown 标签 ···(同标签条,附加)', [['code', '切换预览 / 源码'], ['eye', '只读预览'], ['-'], ['align', '格式化文档', '⇧⌥F'], ['wrap', '自动换行', '⌥Z']])}
      ${M('差异标签 ···', [['wrap', '自动换行', '⌥Z'], ['swap', '切换左右 / 统一'], ['-'], ['code', '打开文件'], ['copy', '复制路径'], ['clock', '显示文件历史']])}
      ${M('SCM 标题 ···', [['arrow', '拉取'], ['arrow', '推送'], ['sync', '同步'], ['-'], ['layers', '储藏所有更改'], ['layers', '弹出最近的储藏'], ['-'], ['list', '切换列表 / 树'], ['refresh', '刷新']])}
      ${M('文件树标题 ···', [['filePlus', '新建文件…'], ['folderPlus', '新建文件夹…'], ['-'], ['refresh', '刷新'], ['collapseAll', '全部折叠'], ['-'], ['folder', '在访达中显示项目']])}
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
      cell('菜单(补全后)', '', box(menu(220, [['filePlus', '新建文件…'], ['folderPlus', '新建文件夹…'], ['-'], ['edit', '重命名…', 'F2'], ['trash', '移到废纸篓…', '⌘⌫', 'danger']]) + `<div class="t3 fs-cap" style="padding:8px 4px;line-height:1.7">文件、文件夹右键与文件树标题 ··· 都用同一组;位置按「主要动作 → 导航 → 复制 / 访达 → 危险操作」放在原有项之间(新建在最前,重命名与删除在复制路径之前)。</div>`, 230, 330), '多根时顶层文件夹行只有「新建…」。')))}
    ${sec('实现状态', '', tbl([['范围', 240], ['说明', 440], ['状态']], [
      ['文件操作 · 新建 / 重命名 / 移到废纸篓', 'file_ops+ide_tree:F2(ide_overlay)重命名、⌘⌫ 删除、拖到文件夹 moveFile;输入框显示完整文件名且默认只选中主名(不含扩展名),提交按输入原样改名(不自动补回原扩展名);文件树有选中项且聚焦时 F2 直接重命名(不依赖编辑器标签)。有 go test,未用 Computer Use 实测;缺口:Computer Use 点不进文件树输入框的选区,选区只靠单测', st('part')],
      ['拖拽到对话(文件树 / 标签 / SCM / 访达)', '落点高亮 + 输入框顶部加引用胶囊(App.refs,去重 / 可移除 / 点开文件,发送时 foldRefs 并进正文最前)+ 聚焦输入框;`-ed ref:路径;路径` 可截图;未用 Computer Use 实测', st('part')],
      ['编辑器标签条 ··· 与标签右键菜单', '向右 / 向下拆分、在侧边打开 ⌘⏎(已接线)、在新窗口中打开、虚拟标签新窗口(计划 / 审查等,真实会话有限制);Markdown 标签追加「切换预览 / 源码 · 只读预览(状态)· 格式化文档」。未测试', st('ok')],
      ['文件 / 文件夹右键菜单、文件树标题 ···', '文件:打开 / 在侧边打开 ⌘⏎ / 打开更改(无改动置灰)/ 显示文件历史 / 让 Claude… / 引用到对话 / 复制·剪切·粘贴 / 重命名 F2 / 移到废纸篓 ⌘⌫ / 复制路径 / 访达;文件夹:新建 / 在文件夹中搜索 / 在此处打开终端 / 引用;多根顶层文件夹另有「在源代码管理中查看 / 从工作区移出…」;标题 ···:新建 / 刷新 / 全部折叠 / 在访达中显示项目;标题行图标:新建文件(filePlus)/ 新建文件夹(folderPlus)/ 全部收起(collapseAll,一键折叠整棵树)/ 刷新 / ···,多根时首枚换成「添加文件夹」。未测试', st('ok')],
      ['对话栏头部 ··· 与会话行 ···(右键同)', '复制标题 / 复制为 Markdown(仅当前已打开会话)/ 复制会话链接 / 分叉会话(运行中置灰)· CmdForkSession;侧栏会话行与头部共用 sessionMenu,已一致。未测试', st('ok')],
      ['差异标签 ···、SCM 标题 ···、终端标签条 ···、自定义详情 ···', 'more_menus.go:SCM 含列表↔树切换;差异 / 终端 / 自定义详情同上;终端标签右键与终端内右键均有「关闭终端」(至少保留一个,置灰写明原因)。未测试', st('ok')]]))}</div>`;
})();
