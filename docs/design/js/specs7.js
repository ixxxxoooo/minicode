/* 设计稿 v3.9 · 排版与宽度 · 字号 / 行距 / 内容宽度
   定稿理由见页内说明;Go 偏好键与档位与此表一致(appearance.fontPreset / flowWidth / setWidth / lineHeight 等)。
   @author ixxxxoooo */
(function () {
  const { ic, btn, seg, tag } = U;
  const { head, sec } = P.h;
  const tbl = (cols, rows) => `<table class="ia-tbl"><thead><tr>${cols.map(([t, w]) => `<th ${w ? `style="width:${w}px"` : ''}>${t}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map((c, k) => `<td ${k === 0 ? '' : 'class="t2"'}>${k === 0 ? `<b>${c}</b>` : c}</td>`).join('')}</tr>`).join('')}</tbody></table>`;
  const st = (s) => ({ ok: tag('已实现', 'ok'), part: tag('部分', 'warn'), todo: tag('待做', '') }[s]);

  /* 三档字号令牌(100% 界面大小时的 CSS 变量值) */
  const TYPE_SCHEMES = {
    a: { name: 'A · Cursor / VS Code 式(全程 13)', cap: 11.5, body: 13, read: 13, title: 13, head: 16, display: 20, mono: 12, lh: 1.5 },
    b: { name: 'B · v3.9 旧定稿 × 默认 110%(已弃用)', cap: 13.2, body: 14.3, read: 16.5, title: 16.5, head: 21.5, display: 28.6, mono: 14.3, lh: 1.65 },
    c: { name: 'C · v4 定稿(默认,界面大小 100%)', cap: 12, body: 13, read: 14, title: 14, head: 17, display: 22, mono: 12, lh: 1.6 },
  };

  const sampleMD = (sch) => {
    const v = (n, u) => `var(--typo-${sch}-${u}, ${n}px)`;
    return `<div class="typo-sample" data-typo-sample="${sch}" style="--typo-fs-cap:${sch.cap}px;--typo-fs-body:${sch.body}px;--typo-fs-read:${sch.read}px;--typo-fs-title:${sch.title}px;--typo-fs-head:${sch.head}px;--typo-fs-mono:${sch.mono}px;--typo-lh:${sch.lh};padding:14px 16px;border-radius:var(--r-md);background:var(--bg-raised);box-shadow:var(--sh-1);border:1px solid var(--bd-1);max-width:100%">
      <div class="typo-cap" style="font-size:${sch.cap}px;line-height:1.45;color:var(--tx-3);margin-bottom:6px">步骤行附注 · 脚注说明(cap)</div>
      <div style="font-size:${sch.title}px;font-weight:500;margin:0 0 8px">审查本轮改动</div>
      <div class="typo-read" style="font-size:${sch.read}px;line-height:${sch.lh};color:var(--tx-1);margin-bottom:10px">问题出在路由表用原始路径作键,<code style="font-family:var(--font-mono);font-size:${sch.mono}px;padding:1px 5px;border-radius:var(--r-xs);background:var(--bg-sunken)">match_path</code> 对大小写敏感,导致 <code style="font-family:var(--font-mono);font-size:${sch.mono}px;padding:1px 5px;border-radius:var(--r-xs);background:var(--bg-sunken)">/API</code> 与 <code style="font-family:var(--font-mono);font-size:${sch.mono}px;padding:1px 5px;border-radius:var(--r-xs);background:var(--bg-sunken)">/api</code> 落在不同槽位。已改为统一 <span style="font-family:var(--font-mono);font-size:${sch.mono}px">to_lowercase</span> 后再查表。</div>
      <div style="font-size:${sch.body}px;line-height:var(--lh-body);color:var(--tx-2);margin-bottom:8px">UI 标签示例:模型 <span style="font-weight:500">Sonnet</span> · 分支 <span class="mono">feat/api</span></div>
      <pre class="typo-pre" style="margin:0;padding:10px 12px;border-radius:var(--r-sm);background:var(--bg-sunken);font-family:var(--font-mono);font-size:${sch.mono}px;line-height:1.6;color:var(--tx-1);overflow:auto">fn normalize(path: &amp;str) -&gt; String {
    path.to_ascii_lowercase()
}</pre>
      ${F ? `<div style="margin-top:10px">${F.step('edit', 'router.rs', 'src/', '<span class="c-ok">+4</span> <span class="c-err">−2</span>')}</div>` : ''}
    </div>`;
  };

  const ladderCompare = () => `<div class="typo-ladder" data-act="typo-scheme">
    <div class="f ai g4" style="margin-bottom:12px;flex-wrap:wrap">${Object.entries(TYPE_SCHEMES).map(([k, s]) => btn(s.name, k === 'c' ? 'pri' : 'soft', '', `data-scheme="${k}"`)).join('')}</div>
    <div class="grid3 typo-panels">${['a', 'b', 'c'].map(k => sampleMD(TYPE_SCHEMES[k])).join('')}</div>
    <div class="t3 fs-cap" style="margin-top:10px;line-height:1.55">为什么换:旧定稿 B 在默认「界面大小 110%」下实际是 <b>界面 14.3 / 阅读 16.5 / 代码 14.3</b>,比同类工具(正文 13–14)整整大一级,会话流尤其显得「撑」。v4 把默认界面大小改成 <b>100%</b>,阅读正文降到 <b>14</b>(中文需要比西文略大才视觉等重,所以不跟 Cursor 的 13 完全一致),代码与说明同为 <b>12</b>。相对界面正文的阅读比例 <b>1.08</b>,代码约为阅读的 <b>0.86</b>。</div>
  </div>`;

  /* 同类软件的公开默认值(凭记忆整理,未逐一实测;BaoCode 数值取自本机 ~/baocode 源码) */
  const refTable = () => tbl([['软件', 150], ['界面 / 步骤行', 110], ['会话正文', 130], ['次要 / 附注', 100], ['代码', 90], ['备注', 0]], [
    ['Cursor', '13', '13(Agent 面板)', '11–12', '12–13', '工具调用整行灰色小字,文件名与动词同权重,不加粗'],
    ['VS Code / Copilot Chat', '13', '13', '11–12', '12(macOS)', '全局 13,没有「阅读加大」一级'],
    ['Zed', '约 14', '约 14', '约 12', '约 13', 'UI 字体与 Agent 面板同号'],
    ['Claude 桌面 / Claude Code', '约 14', '约 15–16', '约 12–13', '约 13', '面向纯聊天,正文偏大'],
    ['BaoCode(本机源码)', '13', '13.5(用户) / 13(助手)', '11.5 / 12', '12.5', '步骤标题 13 + 次要 12;粗体只用 w500 / w600'],
    ['<b>MiniCode v4</b>', '<b>13</b>', '<b>14</b>', '<b>12</b>', '<b>12</b>', '阅读比界面大 1px 补偿中文;工具行常规字重'],
  ]);

  /* 字重规范 */
  const weightTable = () => tbl([['字重', 90], ['用在哪', 0], ['不要用在哪', 0]], [
    ['400 常规', '会话正文、工具行的动词与对象、路径 / 附注、说明文字、表单控件文字', '——'],
    ['500 中等', '当前选中项(标签、分段、列表选中)、卡片 / 面板标题(计划、子 Agent、权限)、按钮、分组小标题、待办进行中', '工具行对象、会话正文'],
    ['600 强调', '页面标题 head、数字徽标、Markdown 标题', '正文里的「重点词」(用颜色或行内代码,不用粗体)'],
    ['700', '<b>不使用</b>。浏览器默认 &lt;b&gt; 是 700,已在 base.css 统一压到 600', '全部'],
  ]);

  P['typo-width'] = () => `<div class="ds" style="max-width:1240px">${head('排版与宽度 · 字号 / 行距 / 内容宽度', '会话阅读舒适度由三层控制:① 全局字号令牌(v4 定稿 C 为默认);② 界面大小 × 会话正文字号预设组合;③ 内容列最大宽度(会话与输入框同宽,设置页单独一档)。窄屏 / 分屏时列宽随容器收窄,不强行撑出横向滚动。')}

    ${sec('字号阶梯 · v4 定稿 C(默认界面大小 100%)', '六级:micro 10.5(徽标 / 键帽)· ctl 11.5(chip / 分段 / 小按钮)· cap 12(说明 / 附注 / 右侧结果)· body 13(界面 / 工具行)· read 14(会话正文)· title 14 / head 17 / display 22。阅读正文只比界面大 1px;代码与说明同为 12。', ladderCompare())}
    ${sec('同类软件对照', '定稿前对照了常见同类产品。除 BaoCode 取自本机源码外,其余是公开默认值的大致取值,<b>未逐一实测</b>,只作量级参考。', `<div class="snip" style="height:auto;padding:10px 14px;width:100%">${refTable()}</div>`)}
    ${sec('字重规范 · 只有三档', '会话里原先「黑色加粗」突兀的根因:工具行对象用了 tx-1 + 500,浏览器 &lt;b&gt; 又是 700。v4:工具行的动词与对象都用常规字重,颜色分层(动词 tx-3 → 对象 tx-2 → 附注 tx-3 → 结果 tx-4),只有「进行中」的那一行对象提亮到 tx-1。', `<div class="snip" style="height:auto;padding:10px 14px;width:100%">${weightTable()}</div>`)}

    ${sec('平衡原则(摘要)', '', `<div class="snip" style="height:auto;padding:12px 14px;width:100%">${tbl([['项', 200], ['建议', 0]], [
      ['UI 正文 vs 阅读正文', '13 : 14(≈1.08);界面大小与会话字号可分别上调'],
      ['代码 vs 阅读正文', '等宽 ≈ 0.86×(定稿 12 / 14);混排时与中文字形视觉等重'],
      ['说明小字(cap)', '12;控件内文字 ctl 11.5;界面缩到 90% 时 cap 钳制不低于 11.5'],
      ['阅读行距(中文)', '1.6 默认;紧凑 1.45 / 宽松 1.75'],
      ['舒适行宽', '约 35–45 个汉字 → 640–760px 内容列(含内边距时视对话栏宽度而定)'],
    ])}</div>`)}

    ${sec('字号预设', '分段选择会同时写入「界面大小」与「会话正文字号」;任一滑块手动偏离后预设高亮「自定义」(Go: appearance.fontPreset, 0–3 预设, 4=自定义)。', `<div class="snip" style="height:auto;padding:10px 14px">${tbl([['预设', 72], ['界面大小', 90], ['会话正文', 100], ['说明', 0]], [
      ['紧凑', '90%', '95%', '小屏 / 信息密度高(界面 11.7 / 正文 13.3)'],
      ['标准', '100%', '100%', '默认;界面 13、阅读 14,全新安装即落在「标准」'],
      ['舒适', '110%', '105%', '长时间阅读(界面 14.3、阅读 15.5 —— 即旧默认的量级)'],
      ['大字', '125%', '115%', '视力 / 演示;仍保留层级'],
    ])}
    <div class="f g3 ai" style="margin-top:10px">${seg(['紧凑', '标准', '舒适', '大字', '自定义'], 1)} ${seg(['界面大小 100%', '会话正文 100%'], 0)}</div>
    <div class="t3 fs-cap" style="margin-top:8px">设置页目标分类 id: <code>typography</code>(与「外观 → 字体与尺寸」合并规划);落地前行列表见本页末「设置行」。</div></div>`)}

    ${sec('会话内容宽度', '六档:窄 560 / 紧凑 640 / 标准 712(默认) / 宽 860 / 特宽 1040 / 占满(仅留 flow 内边距,硬上限 1400)。时间线、排队卡、任务坞、输入框共用 <code>flowCol</code>。旧键 appearance.width 0/1/2 迁移到紧凑 / 标准 / 宽。', `<div class="snip" style="height:auto;padding:10px 14px;width:100%">${tbl([['档', 56], ['最大列宽', 80], ['场景', 0]], [
      ['窄', '560', 'IDE 侧栏对话 / 分屏'],
      ['紧凑', '640', '原「窄」'],
      ['标准', '712', '默认,≈40 汉字/行'],
      ['宽', '860', '大屏 Agent'],
      ['特宽', '1040', '超宽屏 / 演示'],
      ['占满', 'min(可用宽, 1400)', '仅内边距;避免超长行难以扫视'],
    ])}
    <div class="t3 fs-cap" style="margin-top:8px;line-height:1.55"><b>自动收窄:</b>有效列宽 = min(档位上限, 对话栏宽 − 2×flowPad)。flowPad 仍按对话栏三档(宽 20 / 中 14 / 窄 10)。欢迎页输入框与时间线同规则。形态切换吸附(<code>flowSnapWidth</code>)不变。</div></div>`)}

    ${sec('设置页内容宽度', '五档:窄 620 / 标准 760(默认) / 宽 900 / 特宽 1080 / 占满。窗口宽 &lt; 900 时现有 settingsNarrow 逻辑铺满,不分档。', `<div class="snip" style="height:auto;padding:10px 14px">${tbl([['档', 72], ['内容列', 80], ['说明', 0]], [
      ['窄', '620', '原固定值'],
      ['标准', '760', '新默认'],
      ['宽', '900', '大屏'],
      ['特宽', '1080', '超宽'],
      ['占满', '主区减导航与边距', '与窄窗自动铺满二选一取较小逻辑'],
    ])}</div>`)}

    ${sec('正文行距', '仅影响会话流 Markdown / 用户消息 / 输入框,不动编辑器与终端。', `${seg(['紧凑 1.45', '标准 1.6', '宽松 1.75'], 1)}<div class="t3 fs-cap" style="margin-top:8px">偏好键 appearance.lineHeight · 默认 1(标准)。</div>`)}

    ${sec('IDE 对话栏默认宽度(可选)', 'WDock 352 提供快捷档,仍可拖拽;偏好 appearance.dockW 四档 320 / 352 / 440 / 560。', `<div class="snip" style="height:auto;padding:10px 14px">${tbl([['档', 72], ['宽度', 60], ['实现状态', 80]], [
      ['窄', '320', st('todo')],
      ['标准', '352', '现有常量'],
      ['宽', '440', st('todo')],
      ['特宽', '560', st('todo')],
    ])}<div class="t3 fs-cap" style="margin-top:8px">拖拽微调与布局记忆优先;快捷档若实现成本高可先只存偏好默认,恢复布局时用。</div></div>`)}

    ${sec('设置行(外观 · 字体与尺寸 / typography)', '与同事合并前的目标列表;键名与 Go settings_typography_wire.go 一致。', `<div class="snip" style="height:auto;padding:10px 14px">${tbl([['行', 160], ['偏好键', 200], ['默认', 0], ['状态', 72]], [
      ['字号预设', 'appearance.fontPreset', '1 标准', st('part')],
      ['界面大小', 'appearance.uiScale', '100%(旧默认 110%)', st('part')],
      ['会话正文字号', 'appearance.fontScale', '100%', st('ok')],
      ['正文行距', 'appearance.lineHeight', '标准', st('part')],
      ['会话内容宽度', 'appearance.flowWidth', '标准 712', st('part')],
      ['设置页内容宽度', 'appearance.setWidth', '760', st('part')],
      ['IDE 对话栏默认宽', 'appearance.dockW', '352', st('todo')],
    ])}</div>`)}

    ${sec('实现状态 · 本页', '', `<div class="snip" style="height:auto;padding:10px 14px">${tbl([['范围', 200], ['状态', 80], ['说明', 0]], [
      ['css/tokens.css 字号 v4(含 --fs-ctl / --fw-strong)', st('part'), '设计稿已更新;Go theme.go 同步(未测试)'],
      ['字重三档(无 700)', st('part'), 'flow.css / base.css 已改;Go 工具行 / 卡片标题对齐(未测试)'],
      ['会话六档列宽', st('part'), 'appearance.flowWidth + 迁移'],
      ['设置五档列宽', st('part'), 'appearance.setWidth'],
      ['字号预设 / 行距', st('part'), '未集中测试'],
      ['IDE dockW 四档', st('todo'), '仅规范,未接布局'],
      ['设计稿 settings.js 分组', st('todo'), '等待 typography 分类合并'],
    ])}</div>`)}
  </div>`;

  /* 交互:字号阶梯方案切换(只改样本列,不改全局 tokens) */
  APP.hooks.push(() => {
    const root = document.querySelector('.typo-ladder');
    if (!root) return;
    const panels = () => [...root.querySelectorAll('[data-typo-sample]')];
    root.querySelectorAll('[data-scheme]').forEach(btn => {
      btn.addEventListener('click', () => {
        const k = btn.dataset.scheme;
        root.querySelectorAll('[data-scheme]').forEach(b => b.classList.toggle('pri', b === btn));
        panels().forEach(p => {
          const on = p.dataset.typoSample === k;
          p.style.opacity = on ? '1' : '.42';
          p.style.outline = on ? '2px solid var(--ac-line)' : 'none';
        });
      });
    });
    const def = root.querySelector('[data-scheme="c"]');
    if (def) def.click();
  });
})();