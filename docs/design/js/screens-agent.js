/* Agent 页面的设计稿:会话主界面、欢迎页、输入框、交互卡、子 Agent 与任务、提示与错误、右栏 */
(function () {
  const { i, chip, bdg, cnt, kbd, tg, seg, dot, ib, btn, logo, meter, hl } = K;
  const { C } = window;

  S.group('agent', 'Agent 会话');

  /* ───────── 会话主界面 ───────── */
  const chatA = () => `<div class="f" style="height:100%">
    ${C.sbA(0)}
    <div class="grow fc" style="min-width:0">
      ${C.head()}
      <div class="grow" style="overflow:hidden"><div class="col">${C.turn('A')}</div></div>
      <div class="col" style="padding-bottom:10px">${C.cmpA()}</div>
    </div>
    ${C.rp('changes')}
  </div>`;

  const railIcons = `<div class="fc ai" style="width:40px;border-left:1px solid var(--bd);padding:6px 0;gap:3px;background:var(--bg-chrome);flex:none">
    <span class="ib on rel">${i('diff')}${cnt(3).replace('class="cnt', 'class="cnt abs')}</span>${ib('folder')}${ib('terminal')}${ib('todo')}<span class="ib rel">${i('activity')}</span>
    <span class="grow"></span>${ib('panelR')}</div>`;
  const chatB = () => `<div class="f" style="height:100%;position:relative">
    ${C.railB('chat')}${C.drawerB(0)}
    <div class="grow fc" style="min-width:0">
      <div class="chat-head" style="height:36px">
        <span class="title ell">完整修复一个 Bug:路由大小写匹配</span>${i('edit', 's12 tx3')}${bdg('router-demo', '')}${chip('main', 'branch', 'line')}
        <span class="grow"></span>${seg(['Agent', 'IDE'], 0)}
      </div>
      <div class="grow" style="overflow:hidden"><div class="col" style="max-width:760px">${C.turn('A')}</div></div>
      <div class="col" style="max-width:760px;padding-bottom:10px">${C.cmpB()}</div>
    </div>
    ${railIcons}
    <div class="abs card" style="right:46px;top:44px;width:316px;height:430px;box-shadow:var(--shadow-pop);overflow:hidden;z-index:5">
      <div class="f ai g6" style="height:30px;padding:0 8px;border-bottom:1px solid var(--bd)">${i('diff', 's12 acc')}<b>改动</b>${cnt(3)}<span class="grow"></span><span class="tx3 s10">悬停展开 · Esc 收起</span>${ib('pin', 'sm')}</div>
      ${C.panelChanges()}
    </div>
    <div class="abs" style="right:46px;top:482px;z-index:6;font-size:11px" ><span class="bdg ac" style="height:20px">右栏收成图标条:悬停即预览,点击钉住</span></div>
  </div>`;

  const tabC = (st, t, on) => `<div class="f ai g6" style="height:28px;padding:0 10px;border-radius:8px 8px 0 0;margin-top:6px;max-width:210px;${on ? 'background:var(--bg-main);box-shadow:0 0 0 1px var(--bd);border-bottom:none' : 'color:var(--tx2)'}"><span class="st" style="display:grid;place-items:center">${C.st[st]}</span><span class="ell">${t}</span>${on ? ib('x', 'sm') : ''}</div>`;
  const chatC = () => `<div class="fc" style="height:100%;position:relative">
    <div class="tb" style="height:40px;background:var(--bg-chrome);border-bottom:1px solid var(--bd);padding-bottom:0;align-items:flex-end">
      <div class="f ai" style="height:40px;margin-right:6px">${K.tl()}</div>
      ${ib('panelL', '', '会话列表')}
      <div class="f g2 grow" style="align-items:flex-end;min-width:0">${tabC('run', '完整修复一个 Bug:路由大小写匹配', 1)}${tabC('ask', '权限请求:修改配置…')}${tabC('unread', '文件读写:梳理并整理路由层')}${tabC('idle', 'Markdown 与代码高亮')}<span class="ib" style="margin-bottom:2px">${i('plus')}</span></div>
      <div class="f ai g6" style="height:40px">${ib('history')}${seg(['Agent', 'IDE'], 0)}</div>
    </div>
    <div class="grow rel" style="overflow:hidden">
      <div class="col" style="max-width:820px;padding-top:6px">${C.turn('A')}</div>
      <div class="abs card" style="right:14px;top:10px;width:300px;height:380px;box-shadow:var(--shadow-pop);overflow:hidden">
        <div class="f ai g6" style="height:28px;padding:0 8px;border-bottom:1px solid var(--bd)">${i('diff', 's12 acc')}<b>router.rs</b><span class="okc mono">+4</span><span class="errc mono">−1</span><span class="grow"></span>${btn('接受', 'sm ok')}${btn('回退', 'sm')}${ib('x', 'sm')}</div>
        <div style="padding:4px 0">
          <div class="dl ctx"><span class="no">41</span><span class="tx">fn match_path(&self, p: &str)</span></div>
          <div class="dl del"><span class="no">42</span><span class="tx">    self.routes.get(p)</span></div>
          <div class="dl add"><span class="no">42</span><span class="tx">    let key = p.to_ascii_lowercase();</span></div>
          <div class="dl add"><span class="no">43</span><span class="tx">    self.routes.get(key)</span></div></div>
        <div class="tx3 s10" style="padding:6px 8px;border-top:1px solid var(--bd-soft)">检查器:点击改动 / 计划 / 任务时浮出,不占据常驻空间</div>
      </div>
    </div>
    <div class="col" style="max-width:820px;padding-bottom:10px">
      <div class="f ai g6" style="margin-bottom:6px">${chip('后台命令 2', 'terminal', 'line')}${chip('子代理 1', 'bot', 'line')}${chip('改动 3', 'diff', 'line')}${chip('计划 1/3', 'todo', 'line')}<span class="grow"></span><span class="tx3 s11">${kbd('⌘', 'K')} 命令面板</span></div>
      ${C.cmpC({ hint: false })}
    </div>
  </div>`;

  S.add({
    id: 'chat', group: 'agent', title: '会话主界面',
    desc: '最核心的页面:侧栏 + 会话流 + 右栏 + 输入框。三套布局思路,信息量一致。',
    notes: [
      '侧栏会话改成双行条目(标题 + 分支/改动/状态),标题不再被截成四个字;状态点统一在左,「运行中 / 待许可 / 未读」一眼可辨。',
      '侧栏顶部的「自动化 / 自定义」两行收成一排图标条,省出两行高度;工作区标题行加入状态筛选(全部 / 运行中 / 待处理)。',
      '左下角补一条 5 小时额度细条,不用点开上下文面板也知道余量。',
      '右栏标签统一「图标 + 文字 + 计数」,不再被截成「终」;顶栏 36px 与会话顶栏同高,对齐成一条线。',
    ],
    variants: [
      { label: '经典三栏 · 优化版', tag: '最接近现状', note: '保持树形工作区,改进密度与状态表达。', render: chatA },
      { label: '图标轨 + 会话抽屉', tag: '推荐', rec: true, note: '会话按时间扁平排列、带项目徽标;右栏收成图标条悬停预览,会话区更宽。', render: chatB },
      { label: '专注单栏 + 标签', note: '像浏览器标签一样切会话,检查器浮出;适合小屏、单项目深度工作。', render: chatC },
    ],
  });

  /* ───────── 欢迎页(新会话) ───────── */
  const sug = (ic, t, d) => `<div class="card f g8" style="padding:8px 10px;align-items:flex-start;width:230px"><span class="avatar sm">${i(ic, 's12')}</span><div><div class="m">${t}</div><div class="tx3 s11" style="line-height:1.35">${d}</div></div></div>`;
  const recent = (st, t, s) => `<div class="f ai g8" style="height:28px;padding:0 8px;border-radius:6px">${C.st[st]}<span class="grow ell">${t}</span><span class="tx3 s11">${s}</span></div>`;
  const welcomeA = () => `<div class="f" style="height:100%">${C.sbA(-1)}
    <div class="grow fc" style="min-width:0">
      ${C.head({ noTitle: true })}
      <div class="grow fc ai jc" style="padding-bottom:40px">
        <div class="fc ai g8" style="margin-bottom:18px"><span class="logo" style="width:34px;height:34px;border-radius:10px">${i('mark', 's18')}</span><div class="s20 b">今天想在 router-demo 里做点什么?</div><div class="tx3">main · 3 项未提交改动 · 上次会话 5 分钟前</div></div>
        <div style="width:640px">${C.cmpA({ focus: true, hint: false })}</div>
        <div class="f g8 wrap jc" style="width:640px;margin-top:14px">${sug('bug', '修复一个 Bug', '描述现象,我来定位并修复')}${sug('book', '解释这个项目', '梳理目录结构与关键流程')}${sug('check', '补充测试', '为最近改动补齐单元测试')}${sug('eye', '审查未提交改动', '3 个文件 · +18 −4')}</div>
        <div style="width:640px;margin-top:16px"><div class="cap" style="padding-left:8px">最近会话</div>${recent('run', '完整修复一个 Bug:路由大小写匹配', '运行中')}${recent('ask', '权限请求:修改配置与执行命令', '5 分钟')}${recent('unread', '文件读写:梳理并整理路由层', '1 小时')}</div>
      </div>
    </div>
  </div>`;
  const welcomeB = () => `<div class="f" style="height:100%">${C.railB('chat')}${C.drawerB(-1)}
    <div class="grow fc" style="min-width:0">
      <div class="chat-head"><span class="title">新对话</span><span class="grow"></span>${seg(['Agent', 'IDE'], 0)}</div>
      <div class="grow f g20" style="padding:20px 28px;min-height:0">
        <div class="grow fc jc" style="max-width:560px;margin:0 auto 60px">
          <div class="s16 b" style="margin-bottom:10px">开始新的会话</div>
          ${C.cmpB({ focus: true, hint: false })}
          <div class="f g6 wrap" style="margin-top:12px">${chip('修复 Bug', 'bug', 'line')}${chip('解释项目', 'book', 'line')}${chip('补充测试', 'check', 'line')}${chip('审查改动', 'eye', 'line')}${chip('重构…', 'wrench', 'line')}</div>
        </div>
        <div class="fc g10" style="width:310px;flex:none">
          <div class="card pad"><div class="f ai g6" style="margin-bottom:6px">${i('folder', 'tx2')}<b>router-demo</b><span class="grow"></span>${bdg('Rust', '')}</div>
            <div class="fc g6 s11">
              <div class="f ai g6">${i('branch', 's12 tx3')}<span class="grow">main</span><span class="tx3">↑1 ↓0</span></div>
              <div class="f ai g6">${i('diff', 's12 tx3')}<span class="grow">未提交改动</span><span class="okc mono">+18</span><span class="errc mono">−4</span></div>
              <div class="f ai g6">${i('activity', 's12 tx3')}<span class="grow">后台任务</span><span>2 运行中</span></div>
              <div class="f ai g6">${i('plug', 's12 tx3')}<span class="grow">MCP</span><span class="warnc">3 已启用 · 1 异常</span></div></div></div>
          <div class="card pad"><div class="f ai jb" style="margin-bottom:6px"><b>本周用量</b><span class="tx3 s11">$4.82</span></div>
            <div class="f g4" style="align-items:flex-end;height:44px">${[30, 52, 22, 64, 80, 40, 58].map((h, k) => `<div class="grow" style="height:${h}%;border-radius:2px 2px 0 0;background:${k === 4 ? 'var(--ac)' : 'var(--bg-active)'}"></div>`).join('')}</div>
            <div class="f jb tx3 s10" style="margin-top:3px"><span>一</span><span>二</span><span>三</span><span>四</span><span>五</span><span>六</span><span>日</span></div></div>
          <div class="card" style="padding:4px"><div class="cap" style="padding-top:2px">继续上次</div>${recent('run', '完整修复一个 Bug', '运行中')}${recent('ask', '权限请求:修改配置', '待许可')}</div>
        </div>
      </div>
    </div></div>`;
  S.add({
    id: 'welcome', group: 'agent', title: '新会话欢迎页',
    desc: '没有内容时的第一屏。现在是一片空白,可以放更有用的东西。',
    notes: ['空状态不再只是一个输入框:给出项目上下文(分支、改动、MCP 状态)和快速起手式。', '「最近会话」让用户一键接着上次干,避免在侧栏里找。'],
    variants: [
      { label: '居中问候 + 起手式', tag: '推荐', rec: true, note: '轻量、聚焦输入,适合大多数使用场景。', render: welcomeA },
      { label: '项目仪表盘', note: '左输入右概览,信息更多但更重;适合同一项目长期使用。', render: welcomeB },
    ],
  });

  /* ───────── 输入框与弹层 ───────── */
  const boardWrap = (title, sub, a, b, c, d) => `<div class="fc" style="height:100%;padding:18px 26px;gap:12px;background:var(--bg-main)">
    <div><div class="s14 b">${title}</div><div class="tx3">${sub}</div></div>
    <div class="f g24" style="gap:28px"><div style="width:660px;flex:none"><div class="cap" style="padding-left:0">常态 · 宽栏(680)</div>${a}</div><div style="width:330px;flex:none"><div class="cap" style="padding-left:0">窄栏 · IDE 对话栏(340)</div>${b}</div></div>
    <div class="f" style="gap:28px"><div style="width:660px;flex:none"><div class="cap" style="padding-left:0">运行中 · 排队 + 引用 + 草稿</div>${c}</div><div class="grow fc g8"><div class="cap" style="padding-left:0">弹层</div><div class="f g12" style="align-items:flex-start;flex-wrap:wrap">${d}</div></div></div>
  </div>`;
  const cmpRun = (v) => `<div class="fc g6">${C.queueCard()}${C.cmp(v, { running: true, text: '把 <span class="ref">' + i('file', 's12') + 'router.rs</span> 里的 match_path 拆成两个函数,再 <span class="ref">' + i('image', 's12') + '报错截图.png</span>', focus: true })}</div>`;
  S.add({
    id: 'composer', group: 'agent', title: '输入框与运行配置',
    desc: '输入框是最常碰的控件。现在的问题:模式 / 模型 / 思考 / MCP 四个 chip 在窄栏被压成「…」,右下角三个无标签图标含义不明。',
    notes: ['三个方案都把「上下文环」放在发送按钮旁(悬停 / 点击出明细),取代原来那个无标签的圆圈。', '提示词优化(魔棒)只在有草稿时出现;发送与停止合并为一个按钮,运行中显示停止。', '分支与「3 项改动」移进输入框下方一行小字,和 ⇧Tab 提示在一起,不再孤零零一行 main。'],
    variants: [
      { label: '单行 chips(优化现状)', note: '改动最小:四个 chip 保留,窄栏自动收成纯图标 + tooltip。', render: () => boardWrap('方案 A · 单行 chips', '保留现有心智,窄栏降级为图标', C.cmpA(), C.cmpA({ narrow: true }), cmpRun('A'), `${C.popModel()}${C.popMcp()}${C.popContext()}`) },
      { label: '运行配置胶囊', tag: '推荐', rec: true, note: '模式 / 模型 / 思考合成一颗胶囊,点开一个统一面板;窄栏只剩模型名,控件从 4 个降到 2 个。', render: () => boardWrap('方案 B · 运行配置胶囊', '一个入口管全部运行配置,窄栏不再拥挤', C.cmpB(), C.cmpB({ narrow: true }), cmpRun('B'), `${C.popConfig()}${C.popContext()}`) },
      { label: '分层式(模式色带)', note: '顶部色带直接表达权限模式(像状态栏),引用 chips 单独一层;最直观,但更高。', render: () => boardWrap('方案 C · 分层 + 模式色带', '权限模式永远可见,危险模式用色带提醒', C.cmpC(), C.cmpC({ narrow: true }), cmpRun('C'), `${C.popMode()}${C.popModel()}${C.popContext()}`) },
    ],
  });

  /* ───────── 交互卡:权限 / 提问 / 计划 ───────── */
  const col = (t, tag, inner) => `<div class="fc g8" style="width:392px;flex:none"><div class="f ai g6"><b class="s13">${t}</b>${tag}</div><div class="card fc" style="height:660px;background:var(--bg-main);overflow:hidden;justify-content:flex-end">${inner}</div></div>`;
  const ghost = (w) => `<div class="fc g6" style="padding:10px 12px"><div class="msg-user" style="color:var(--tx2)">${w}</div></div>`;
  const permDock = () => `<div class="dock" style="margin:0 12px;border-color:var(--ac)">
      <div class="f ai g6" style="height:28px;padding:0 10px">${i('shield', 's12 acc')}<b>需要你的许可</b>${bdg('写入文件', 'warn')}<span class="grow"></span><span class="tx3 s10">已等待 12 秒</span></div>
      <div style="padding:0 10px 6px"><div class="m">编辑 <span class="mono">.github/workflows/ci.yml</span></div>
        <div style="margin-top:6px;border:1px solid var(--bd);border-radius:6px;overflow:hidden"><div class="dl del"><span class="no">12</span><span class="tx">    node-version: 20</span></div><div class="dl add"><span class="no">12</span><span class="tx">    node-version: 22</span></div></div></div>
      <div class="f ai g6" style="padding:6px 10px 8px;border-top:1px solid var(--bd-soft)">${btn('允许 ⏎', 'pri')}${btn('始终允许 ⌘⏎', '')}${btn('拒绝 Esc', 'ghost')}<span class="grow"></span><span class="tx3 s10">仅此会话</span></div></div>`;
  const permBar = () => `<div class="cmp" style="margin:0 12px 10px;border-color:var(--ac)">
      <div class="f ai g6" style="height:28px;padding:0 10px;border-bottom:1px solid var(--bd-soft)">${i('shield', 's12 acc')}<b>Claude 想编辑文件</b><span class="mono tx2 ell">.github/workflows/ci.yml</span>${bdg('写入', 'warn')}</div>
      <div style="padding:6px 10px"><div style="border:1px solid var(--bd);border-radius:6px;overflow:hidden"><div class="dl del"><span class="no">12</span><span class="tx">    node-version: 20</span></div><div class="dl add"><span class="no">12</span><span class="tx">    node-version: 22</span></div></div></div>
      <div class="fc" style="padding:0 6px 6px">${[['1', '允许一次', 1], ['2', '本会话始终允许 <span class="mono">.github/**</span>', 0], ['3', '拒绝,并告诉 Claude 该怎么做…', 0]].map(([n, t, on]) => `<div class="mi ${on ? 'hl' : ''}" style="height:28px"><span class="kbd" style="${on ? 'color:var(--ac-tx)' : ''}">${n}</span><span class="grow">${t}</span>${on ? kbd('⏎') : ''}</div>`).join('')}</div>
      <div class="f ai g8 s10 tx3" style="padding:4px 10px 6px;border-top:1px solid var(--bd-soft)">${kbd('↑↓')} 选择 ${kbd('⏎')} 确认 ${kbd('Esc')} 拒绝<span class="grow"></span>输入框已让位,批准后恢复</div></div>`;
  const askDock = () => `<div class="dock" style="margin:0 12px">
      <div class="f ai g6" style="height:28px;padding:0 10px;border-bottom:1px solid var(--bd-soft)">${i('hand', 's12 acc')}<b>提问</b>${bdg('匹配策略', 'ac')}<span class="grow"></span><span class="tx3 s10">1 / 2</span></div>
      <div style="padding:8px 10px 4px" class="m">路由匹配时应如何处理大小写?</div>
      <div class="fc g4" style="padding:4px 10px 6px">${[['A', '忽略大小写(推荐)', '统一转成小写后再匹配', 1], ['B', '保持敏感', '只在 404 页提示可能的拼写', 0], ['C', '其他…', '', 0]].map(([k, t, d, on]) => `<div class="f g8 ai" style="padding:5px 8px;border-radius:7px;border:1px solid ${on ? 'var(--ac)' : 'var(--bd)'};background:${on ? 'var(--ac-soft)' : 'var(--bg-card)'}"><span class="kbd">${k}</span><div class="grow"><div>${t}</div>${d ? `<div class="tx3 s11" style="line-height:1.3">${d}</div>` : ''}</div>${on ? i('check', 's12 acc') : ''}</div>`).join('')}</div>
      <div class="f ai g6" style="padding:4px 10px 8px"><span class="tx3 s11">还有 1 题未回答</span><span class="grow"></span>${btn('跳过', 'ghost')}${btn('下一题 ⏎', 'pri')}</div></div>`;
  const askBar = () => `<div class="cmp" style="margin:0 12px 10px;border-color:var(--ac)">
      <div class="f ai g6" style="height:28px;padding:0 10px">${i('hand', 's12 acc')}<b>路由匹配时应如何处理大小写?</b><span class="grow"></span><span class="tx3 s10">1 / 2</span></div>
      <div class="fc" style="padding:0 6px 4px">${[['1', '忽略大小写(推荐)', 1], ['2', '保持敏感', 0], ['3', '其他…', 0]].map(([n, t, on]) => `<div class="mi ${on ? 'hl' : ''}" style="height:28px"><span class="kbd">${n}</span><span class="grow">${t}</span>${on ? kbd('⏎') : ''}</div>`).join('')}</div>
      <div class="f ai g8 s10 tx3" style="padding:4px 10px 6px;border-top:1px solid var(--bd-soft)">${kbd('1-3')} 直接选择 ${kbd('Tab')} 补充说明 ${kbd('Esc')} 跳过</div></div>`;
  const planCard = () => `<div class="card" style="margin:0 12px 8px;border-color:color-mix(in srgb,var(--warn) 45%,var(--bd))"><div class="f ai g6" style="height:28px;padding:0 10px">${i('todo', 's12 warnc')}<b class="grow ell">迁移旧 REST 接口到新网关</b>${bdg('待批准', 'warn')}${btn('查看', 'sm ghost', 'ext')}</div>
      <div class="tx2" style="padding:0 10px 6px;line-height:1.5">1 新建 gateway/ 适配层 · 2 逐个迁移 /api/v1/* · 3 双写一周</div>
      <div class="f ai g6" style="padding:6px 10px 8px;border-top:1px solid var(--bd-soft)">${btn('Build · 逐项确认', 'pri')}${btn('自动', '')}${btn('继续规划', 'ghost')}</div></div>`;
  const planBar = () => `<div class="cmp" style="margin:0 12px 10px;border-color:var(--warn)"><div class="f ai g8" style="height:34px;padding:0 10px">${i('todo', 's14 warnc')}<div class="grow"><b>计划已就绪</b> <span class="tx3">· 3 步 · 6 个文件</span></div>${btn('查看计划', 'sm')}
      <span class="f"><span class="btn pri" style="border-radius:6px 0 0 6px">Build</span><span class="btn pri" style="border-radius:0 6px 6px 0;border-left:1px solid rgba(255,255,255,.35);padding:0 5px">${i('chevD', 's12')}</span></span></div>
      <div class="f ai g8 s10 tx3" style="padding:3px 10px 5px;border-top:1px solid var(--bd-soft)">Build 默认「逐项确认」;下拉选自动 / 跳过权限${C.ctxRing ? '' : ''}</div></div>`;
  const small = (v) => `<div style="padding:0 12px 10px">${C.cmp(v, { narrow: true, hint: false })}</div>`;
  S.add({
    id: 'interact', group: 'agent', title: '权限 · 提问 · 计划批准',
    desc: '需要用户决策的三类卡片。现在是堆在输入框上方的小卡,按钮靠鼠标;可以改得更「键盘优先」。',
    notes: ['无论哪种方案:所有选项都有数字 / 字母快捷键,⏎ 确认、Esc 拒绝;权限卡标出风险类别(写入 / 命令 / 联网 / 删除)与等待时长。', '计划卡不再占会话流里一大块:摘要 + View,批准动作放在最容易按到的位置。'],
    variants: [
      { label: '停靠在输入框上方', note: '保持现有位置,样式更紧凑;输入框仍在,可边看边打字。', render: () => `<div class="f g16" style="padding:18px 26px;height:100%">${col('权限请求', bdg('A', 'ac'), ghost('把 CI 配置里的 Node 版本升到 22 并跑一遍') + '<div class="grow"></div>' + permDock() + small('A'))}${col('提问向导', '', ghost('路由匹配对大小写敏感…') + '<div class="grow"></div>' + askDock() + small('A'))}${col('计划批准', '', ghost('把旧的 REST 接口迁移到新网关') + '<div class="grow"></div>' + planCard() + small('A'))}</div>` },
      { label: '输入框就地变形', tag: '推荐', rec: true, note: '等待决策时输入框「让位」给决策条,焦点只有一个,数字键直接选;批准后恢复输入框。更像 Claude Code 终端的手感。', render: () => `<div class="f g16" style="padding:18px 26px;height:100%">${col('权限请求', bdg('B', 'ac'), ghost('把 CI 配置里的 Node 版本升到 22 并跑一遍') + '<div class="grow"></div>' + permBar())}${col('提问向导', '', ghost('路由匹配对大小写敏感…') + '<div class="grow"></div>' + askBar())}${col('计划批准', '', ghost('把旧的 REST 接口迁移到新网关') + '<div class="grow"></div>' + planBar())}</div>` },
    ],
  });

  /* ───────── 子 Agent 与任务坞 ───────── */
  const subRow = (st, t, ty, meta, kids) => `<div class="fc"><div class="step" style="height:26px">${st}<span class="ob">${t}</span><span class="tx3">${ty}</span><span class="rt">${meta}</span></div>${kids || ''}</div>`;
  const agentsA = () => `<div class="f g20" style="padding:18px 26px;height:100%">
    <div class="fc g8" style="width:600px;flex:none"><div class="b s13">会话流里 · 并行子 Agent 组</div>
      <div class="card fc" style="padding:10px 14px;flex:1;background:var(--bg-main);overflow:hidden">
        ${C.user('同时审查 auth、billing、search 三个模块')}
        <div class="step" style="margin-top:8px"><span class="ic">${i('bot', 's12 acc')}</span><b>并行 3 个子 Agent</b><span class="acc">1 运行中</span><span class="okc">1 已完成</span><span class="errc">1 失败</span><span class="rt">${i('chevD', 's12')}</span></div>
        <div class="nest fc g2">
          ${subRow('<span class="ic okc">' + i('okc', 's12') + '</span>', '审查 auth', 'code-reviewer', '0 秒 · 1 次工具', '')}
          ${subRow('<span class="ic errc">' + i('errc', 's12') + '</span>', '审查 billing', 'code-reviewer', '0 秒 · 1 次工具', `<div class="nest" style="margin-left:7px">${C.step('file', '读取', 'billing.rs', '', '<span class="errc">失败</span>')}<div class="notice err" style="margin:2px 0 4px">${i('errc', 's12')}<span>文件不存在:src/billing.rs</span><span class="grow"></span><span class="chip">重试</span></div></div>`)}
          ${subRow('<span class="ic">' + '<span class="spin"></span>' + '</span>', '审查 search', 'code-reviewer', '<span class="bdg ac">后台</span> 5 秒 · 1 次工具', `<div class="nest" style="margin-left:7px">${C.step('search', '搜索', 'search.rs', '', '')}</div>`)}
        </div>
        <div class="md" style="margin-top:8px"><p>auth 通过,billing 失败,search 已转入后台继续运行。</p></div>
      </div></div>
    <div class="fc g8 grow"><div class="b s13">输入框上方 · 任务坞(展开「子代理」)</div>
      <div class="card fc" style="flex:1;background:var(--bg-main);justify-content:flex-end;overflow:hidden;padding-bottom:10px">
        <div class="dock" style="margin:0 12px 6px;border-bottom:1px solid var(--bd);border-radius:10px">
          <div class="f ai g6" style="height:26px;padding:0 10px;border-bottom:1px solid var(--bd-soft)">${i('bot', 's12 acc')}<b>子代理</b><span class="tx3">1 运行中 · 2 已结束</span><span class="grow"></span>${btn('全部停止', 'sm ghost')}</div>
          ${['审查 search|5 秒 · 读取 search.rs|运行中|ac', '审查 auth|0 秒|已完成|ok', '审查 billing|0 秒 · 读取 billing.rs 失败|失败|err'].map(r => { const [n, m, s, t] = r.split('|'); return `<div class="f ai g8" style="height:30px;padding:0 10px">${t === 'ac' ? '<span class="spin"></span>' : i(t === 'ok' ? 'okc' : 'errc', 's12 ' + (t === 'ok' ? 'okc' : 'errc'))}<span class="m">${n}</span><span class="tx3 ell grow">${m}</span>${bdg(s, t)}${t === 'ac' ? btn('停止', 'sm') : ''}</div>`; }).join('')}
        </div>
        <div class="f ai g6" style="padding:0 12px 6px">${chip('子代理 3', 'bot', 'line on')}${chip('后台命令 2', 'terminal', 'line')}${chip('监听 1', 'eye', 'line')}${chip('已结束 3', 'check', 'line')}</div>
        <div style="padding:0 12px">${C.cmpA({ narrow: true, hint: false })}</div>
      </div></div></div>`;
  const agentsB = () => `<div class="f g20" style="padding:18px 26px;height:100%">
    <div class="fc g8" style="width:600px;flex:none"><div class="b s13">会话流里 · 子 Agent 一行摘要(不展开细节)</div>
      <div class="card fc" style="padding:10px 14px;flex:1;background:var(--bg-main)">
        ${C.user('同时审查 auth、billing、search 三个模块')}
        <div class="card" style="margin-top:8px;overflow:hidden"><div class="f ai g8" style="height:30px;padding:0 10px">${i('bot', 's12 acc')}<b>并行 3 个子 Agent</b><span class="grow"></span><span class="tx3 s11">1 / 3 完成</span>${meter(33).replace('class="meter"', 'class="meter" style="width:60px"')}</div>
          ${[['okc', 'auth', '通过 · 无阻塞问题', 'ok'], ['errc', 'billing', '失败 · 文件不存在', 'err'], ['spin', 'search', '进行中 · 读取 search.rs', 'ac']].map(([ic, n, m, t]) => `<div class="f ai g8" style="height:26px;padding:0 10px;border-top:1px solid var(--bd-soft)">${ic === 'spin' ? '<span class="spin"></span>' : i(ic, 's12 ' + (t === 'ok' ? 'okc' : 'errc'))}<span class="m">${n}</span><span class="tx3 ell grow">${m}</span>${i('chevR', 's12 tx3')}</div>`).join('')}
        </div>
        <div class="md" style="margin-top:8px"><p>auth 通过,billing 失败,search 已转入后台继续运行。</p></div>
        <div class="tx3 s11" style="margin-top:auto">点任意一行 → 在右栏「活动」里看这个子 Agent 的完整过程</div>
      </div></div>
    <div class="fc g8 grow"><div class="b s13">统一「活动」抽屉(子代理 / 命令 / 监听合并,带实时输出)</div>
      <div class="card fc" style="flex:1;background:var(--bg-main);justify-content:flex-end;overflow:hidden;padding-bottom:10px">
        <div class="dock" style="margin:0 12px 6px;border-bottom:1px solid var(--bd);border-radius:10px;height:300px;overflow:hidden">
          <div class="panel-h">${['全部 6', '子代理 3', '命令 2', '监听 1'].map((t, k) => `<span class="pt ${k === 0 ? 'on' : ''}">${t}</span>`).join('')}<span class="grow"></span>${btn('停止全部', 'sm ghost')}</div>
          <div class="f" style="height:270px">
            <div class="fc" style="width:176px;border-right:1px solid var(--bd-soft);padding:4px;gap:1px">
              ${[['spin', '审查 search', 1], ['spin', 'pnpm dev', 0], ['spin', '监听服务日志', 0], ['okc', '编译 release', 0], ['okc', '审查 auth', 0], ['errc', '审查 billing', 0]].map(([ic, n, on]) => `<div class="mi ${on ? 'on' : ''}" style="height:24px">${ic === 'spin' ? '<span class="spin"></span>' : i(ic, 's12 ' + (ic === 'okc' ? 'okc' : 'errc'))}<span class="ell grow">${n}</span></div>`).join('')}</div>
            <div class="grow fc" style="min-width:0"><div class="f ai g6" style="height:28px;padding:0 10px;border-bottom:1px solid var(--bd-soft)"><b>审查 search</b>${bdg('后台', 'ac')}<span class="tx3 s11">5 秒 · 1 次工具 · 3.1k token</span><span class="grow"></span>${btn('停止', 'sm')}</div>
              <div style="padding:6px 10px" class="fc g2">${C.step('layers', '读取', 'search.rs', 'src/', '')}${C.step('search', '搜索', 'fn index', '4 处', '')}<div class="md" style="margin-top:6px;color:var(--tx2)">正在检查索引构建逻辑,发现 <code>rebuild()</code> 在锁内做了磁盘 I/O…<span class="pulse">▌</span></div></div></div>
          </div></div>
        <div class="f ai g6" style="padding:0 12px 6px">${chip('活动 3 运行中', 'activity', 'line on')}<span class="tx3 s11">· 全部在一处,点开即看实时输出</span></div>
        <div style="padding:0 12px">${C.cmpB({ narrow: true, hint: false })}</div>
      </div></div></div>`;
  S.add({
    id: 'agents', group: 'agent', title: '子 Agent 与后台任务',
    desc: '现在子代理、后台命令、监听、工作流分四个入口(任务坞 + 右栏「任务」+ 会话流卡片),有点分散。',
    notes: ['方案 B 把三处合一:会话流只留一行摘要,所有细节与实时输出都进同一个「活动」抽屉;失败自动展开并给「重试」。'],
    variants: [
      { label: '保留任务坞(整理版)', note: '会话流里完整展开 + 输入框上方胶囊坞;信息直观但重复。', render: agentsA },
      { label: '统一「活动」抽屉', tag: '推荐', rec: true, note: '摘要在会话流、详情在抽屉,左列表右详情,实时输出与停止按钮都在一处。', render: agentsB },
    ],
  });

  /* ───────── 提示、用量与错误 ───────── */
  const nt = (c, ic, t, d, acts = '') => `<div class="notice ${c}">${i(ic, 's14')}<div class="grow"><div class="m">${t}</div>${d ? `<div class="tx2" style="line-height:1.4">${d}</div>` : ''}</div>${acts}</div>`;
  S.add({
    id: 'notices', group: 'agent', title: '提示 · 用量限额 · 错误',
    desc: '会话流里的非内容元素:回合统计、系统提示、限额、上下文压缩与各类错误。统一成「灰字行 / 淡色条 / 带动作的卡」三级。',
    notes: ['三级:Info 是和步骤行同栅格的灰字行;Warning 是淡琥珀底;Error 是淡红底且必带动作(重试 / 登录 / 复制)。', '限额告急使用「顶部横幅」而不是会话里一条消息,可关闭。'],
    variants: [{
      label: '三级提示体系', tag: '推荐', rec: true, note: '', render: () => `<div class="f g24" style="padding:20px 28px;height:100%;gap:28px">
      <div class="fc g10" style="width:560px;flex:none">
        <div class="b s13">回合底栏(悬停出明细)</div>
        ${C.footer()}
        <div class="abs" style="position:relative;left:60px;width:260px"><div class="pop" style="padding:8px 10px;width:260px">${[['输入', '44.1k'], ['输出', '624'], ['思考', '3.2k'], ['缓存读 / 写', '39.8k / 1.1k'], ['缓存命中率', '90%'], ['费用', '$0.02'], ['首字延迟', '1.2 s'], ['调用轮数', '4']].map(([a, b]) => `<div class="f jb" style="height:19px"><span class="tx2">${a}</span><span class="mono">${b}</span></div>`).join('')}</div></div>
        <div class="b s13" style="margin-top:6px">Info:灰字行</div>
        <div class="step"><span class="ic">${i('info', 's12')}</span><span>已切换为「接受编辑」模式</span></div>
        <div class="step"><span class="ic">${i('layers', 's12')}</span><span>已自动压缩上下文 · 142k → 38k</span><span class="rt acc">查看摘要</span></div>
        <div class="b s13" style="margin-top:6px">Warning</div>
        ${nt('warn', 'alert', '上下文已用 92%', '建议压缩或开启新会话,否则回复可能被截断。', btn('压缩', 'sm') + btn('新会话', 'sm ghost'))}
        ${nt('warn', 'alert', 'Stop 钩子执行失败', 'hooks/format.sh 退出码 1,已忽略并继续。', btn('详情', 'sm ghost'))}
      </div>
      <div class="fc g10 grow">
        <div class="b s13">Error:必带动作</div>
        ${nt('err', 'errc', '上游限流:请求过于频繁', '将在 28 秒后自动重试(第 2 / 3 次)。', btn('立即重试', 'sm') + btn('复制', 'sm ghost'))}
        ${nt('err', 'errc', '登录已失效', '需要重新登录 Claude Code 才能继续。', btn('登录', 'sm pri') + btn('复制', 'sm ghost'))}
        ${nt('err', 'errc', '输出达到上限,回复被截断', '可以让 Claude 从断点继续。', btn('继续', 'sm') )}
        ${nt('info', 'info', '模型拒绝了该请求', '无法协助此请求。可以换种方式描述,或切换模型重试。', btn('编辑重发', 'sm ghost'))}
        <div class="b s13" style="margin-top:6px">限额横幅(会话顶部,可关闭)</div>
        <div class="f ai g8" style="height:30px;padding:0 10px;border-radius:8px;background:var(--warn-soft)">${i('alert', 's12 warnc')}<span>5 小时额度已用 <b>86%</b>,约 42 分钟后恢复</span><span class="grow"></span>${meter(86, 'warn').replace('class="meter', 'class="meter" style="width:90px" data-x="')}${btn('切到 Haiku', 'sm')}${ib('x', 'sm')}</div>
        <div class="b s13" style="margin-top:6px">已中断 / 处理中</div>
        <div class="step"><span class="ic">${i('stop', 's12')}</span><span>已中断 · 按 ⏎ 继续,或直接输入新指令</span></div>
        <div class="step"><span class="ic"><span class="spin"></span></span><span class="tx2">正在后台任务… <span class="tx3">5 s · Esc 中断</span></span></div>
      </div></div>`,
    }],
  });

  /* ───────── 右栏 ───────── */
  const rpBoard = (inner) => `<div class="f g16" style="padding:18px 20px;height:100%;background:var(--bg-chrome)">${inner}</div>`;
  const rpCol = (t, k, h) => `<div class="fc g6" style="width:290px;flex:none"><div class="b s13">${t}</div><div class="card fc" style="height:680px;overflow:hidden;background:var(--bg-main)"><div class="rp" style="width:100%;border:none;height:100%">${C.rpTabs(k, 36, true)}<div class="grow" style="overflow:hidden">${C.panel(k)}</div></div></div></div>`;
  const rpWide = () => `<div class="fc g6 grow"><div class="b s13">展开为宽面板(⌥⌘B 钉住)· 改动用左右对比</div><div class="card f" style="height:680px;overflow:hidden;background:var(--bg-main)">
      <div class="fc" style="width:52px;background:var(--bg-chrome);border-right:1px solid var(--bd);align-items:center;padding:6px 0;gap:3px">${['diff', 'folder', 'terminal', 'todo', 'activity'].map((n, k) => `<span class="ib ${k === 0 ? 'on' : ''}">${i(n)}</span>`).join('')}</div>
      <div class="grow fc"><div class="f ai g8" style="height:36px;padding:0 12px;border-bottom:1px solid var(--bd)"><b>改动</b>${cnt(3)}<span class="tx3">3 个文件 <span class="okc mono">+18</span> <span class="errc mono">−4</span></span><span class="grow"></span>${seg(['Claude 改动', 'Git 更改'], 0, 'sm')}${btn('全部回退', 'sm')}${btn('提交…', 'sm pri')}</div>
        <div class="f grow" style="min-height:0"><div class="fc" style="width:180px;border-right:1px solid var(--bd-soft);padding:4px">${['router.rs|src/|4|1|1', 'case.rs|tests/|12|0|0', 'CHANGELOG.md||2|3|0'].map(r => { const [n, d, a, b, on] = r.split('|'); return `<div class="fileline ${on === '1' ? 'on' : ''}">${i('file', 's12 acc')}<span class="m ell">${n}</span><span class="grow"></span><span class="okc mono s10">+${a}</span></div>`; }).join('')}</div>
          <div class="grow" style="min-width:0;padding:6px 0"><div class="f" style="border:1px solid var(--bd);margin:0 10px;border-radius:6px;overflow:hidden">
            <div class="grow" style="border-right:1px solid var(--bd)"><div class="dl ctx"><span class="no">41</span><span class="tx">fn match_path(&self, p) {</span></div><div class="dl del"><span class="no">42</span><span class="tx">  self.routes.get(p)</span></div><div class="dl ctx"><span class="no"></span><span class="tx"> </span></div><div class="dl ctx"><span class="no">43</span><span class="tx">}</span></div></div>
            <div class="grow"><div class="dl ctx"><span class="no">41</span><span class="tx">fn match_path(&self, p) {</span></div><div class="dl add"><span class="no">42</span><span class="tx">  let key = p.to_ascii_lowercase();</span></div><div class="dl add"><span class="no">43</span><span class="tx">  self.routes.get(key)</span></div><div class="dl ctx"><span class="no">44</span><span class="tx">}</span></div></div></div></div></div></div></div></div>`;
  S.add({
    id: 'rightpanel', group: 'agent', title: '右侧面板(改动 · 文件 · 终端 · 任务 · 计划)',
    desc: '右栏是动态标签。现状是标签太挤(「终端」被截成「终」),同时一直占着 300px。',
    notes: ['标签统一「图标 + 文字 + 计数」;栏窄(下图 290px)时只保留当前标签的文字,其余降级为图标 + 小圆点,不再出现截断的半个字。', '右栏支持三种尺寸:收起(图标条)/ 常规 300 / 宽面板(左右对比的大 Diff、提交)。'],
    variants: [
      { label: '顶部标签条(四个内容)', tag: '推荐', rec: true, note: '最稳妥:沿用顶部标签,图标 + 文字 + 计数。', render: () => rpBoard(rpCol('改动', 'changes') + rpCol('文件', 'files') + rpCol('终端', 'term') + rpCol('任务', 'tasks')) },
      { label: '竖向图标条 + 宽面板', note: '标签变成左侧竖向图标条;展开时可以做得很宽,适合边看大 Diff 边对话。', render: () => rpBoard(rpCol('常规宽度 · 计划', 'plan') + rpWide()) },
    ],
  });
})();
