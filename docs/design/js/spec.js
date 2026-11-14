/* 设计规范页 + 现状问题清单(数据)*/
(function () {
  const { i, chip, bdg, cnt, kbd, tg, seg, dot, ib, btn, logo, meter } = K;
  S.group('overview', '总览与规范');

  const sw = (name, v, note = '') => `<div class="f ai g8" style="height:24px"><i style="width:34px;height:18px;border-radius:5px;background:var(${v});box-shadow:inset 0 0 0 1px var(--bd)"></i><span class="mono s11" style="width:92px">${v}</span><span class="tx2">${name}</span><span class="tx3 s10 grow" style="text-align:right">${note}</span></div>`;
  const sec = (t, inner, w) => `<div class="fc g6" ${w ? `style="width:${w}px;flex:none"` : 'class="grow"'}><div class="grp-t" style="padding-bottom:0">${t}</div>${inner}</div>`;
  const ty = (s, w, t, n) => `<div class="f ai g10" style="height:${Math.max(22, s + 8)}px"><span class="tx3 mono s10" style="width:68px">${s}px / ${w}</span><span style="font-size:${s}px;font-weight:${w}">${t}</span><span class="tx3 s10 grow" style="text-align:right">${n}</span></div>`;
  const rh = (h, n) => `<div class="f ai g8"><span style="width:${h * 2.2}px;height:${h}px;border-radius:5px;background:var(--bg-active);flex:none;max-width:140px"></span><span class="mono s11" style="width:34px">${h}px</span><span class="tx2">${n}</span></div>`;

  S.add({
    id: 'spec', group: 'overview', title: '设计规范(令牌与组件)',
    desc: '这一套设计的「地基」:色彩、字号、间距、行高与基础组件。所有页面都由它们拼出来,切换右上角的浅 / 深色与强调色即可全局预览。',
    notes: [
      '紧凑法则:界面字号 12px 为主、11px 做次要、10px 只用于徽标与快捷键;可点击行高 22 / 24 / 26 三档,工具栏按钮 24,标题栏 36,状态栏 22。',
      '平面优先:用 1px 细线和极淡底色分层,阴影只给浮层(弹出菜单、悬浮卡);强调色只出现在「当前选中 / 主要动作 / 进度」三处。',
      '暖灰中性色(带一点黄)替代原来偏冷的灰,与橙色强调色同一色温;深色模式同样是暖黑,不用纯黑。',
    ],
    variants: [{
      label: '设计令牌', tag: '基础', note: '', render: () => `<div class="f g24" style="padding:20px 26px;height:100%;gap:28px">
      <div class="fc g14" style="width:360px;flex:none;gap:14px">
        ${sec('背景与表面', sw('侧栏 / 工具条', '--bg-chrome') + sw('主内容', '--bg-main') + sw('卡片 / 输入框', '--bg-card') + sw('凹陷(代码块)', '--bg-sunk') + sw('悬停', '--bg-hover') + sw('选中', '--bg-active'))}
        ${sec('边框与文字', sw('分隔线', '--bd-soft', '列表行间') + sw('描边', '--bd', '卡片 / 输入') + sw('强描边', '--bd-strong', '控件') + sw('正文', '--tx') + sw('次要', '--tx2') + sw('弱化', '--tx3'))}
        ${sec('强调与语义', sw('强调色', '--ac', '主动作 / 选中') + sw('成功', '--ok', '增加 / 通过') + sw('危险', '--err', '删除 / 失败') + sw('警告', '--warn', '待确认') + sw('信息', '--info', '链接 / 引用') + sw('扩展', '--vio', '符号 / MCP'))}
        <div class="f ai g6"><span class="tx2">强调色可选:</span><i style="width:16px;height:16px;border-radius:50%;background:#E8541A"></i><i style="width:16px;height:16px;border-radius:50%;background:#3B6FD8"></i><i style="width:16px;height:16px;border-radius:50%;background:#2F9E5B"></i><i style="width:16px;height:16px;border-radius:50%;background:#7A5AE0"></i></div>
      </div>
      <div class="fc g14" style="width:330px;flex:none;gap:14px">
        ${sec('字号层级', ty(20, 650, '页面标题', '设置 / 欢迎页') + ty(16, 650, '区块标题', '设置分类头') + ty(14, 650, '对话框标题', '') + ty(13, 600, '卡片小标题', '') + ty(12, 500, '正文 / 控件文字', '默认') + ty(11, 400, '次要说明', '') + `<div class="f ai g10" style="height:22px"><span class="tx3 mono s10" style="width:68px">10px / 500</span><span class="s10 tx2">徽标 · 快捷键 · 时间戳</span></div>`)}
        ${sec('行高与控件尺寸(紧凑档)', rh(36, '标题栏 / 页头') + rh(32, '右栏标签条') + rh(28, '输入框工具栏 · 面板标题') + rh(26, '侧栏行 · 菜单项 · 输入') + rh(24, '按钮 · 树节点 · 小标题') + rh(22, '工具步骤行 · 状态栏 · chip'))}
        ${sec('圆角', `<div class="f g10 ai">${[4, 6, 8, 11, 99].map(r => `<div class="fc ai g4"><i style="width:40px;height:28px;border:1.5px solid var(--bd-strong);border-radius:${r}px;background:var(--bg-card)"></i><span class="tx3 s10">${r === 99 ? '胶囊' : r}</span></div>`).join('')}</div><div class="tx3 s10">徽标 4 · 按钮 / 输入 6 · 卡片 8 · 输入框 / 弹层 10–11</div>`)}
      </div>
      <div class="fc grow" style="gap:14px">
        ${sec('按钮与控件', `<div class="f g6 ai wrap"><span class="btn pri">主要动作</span><span class="btn soft">次要强调</span><span class="btn">默认</span><span class="btn ghost">幽灵</span><span class="btn danger">危险</span><span class="btn ok">${i('check', 's12')}已通过</span></div>
          <div class="f g6 ai wrap">${ib('send', 'send')}${ib('stop', 'stop')}${ib('gear')}${ib('panelR', 'on')}${chip('Sonnet 5.5', 'spark', '', true)}${chip('3', 'plug', 'warnc')}${chip('全部', '', 'on')}${chip('筛选', 'filter', 'line')}</div>
          <div class="f g6 ai wrap">${bdg('默认')}${bdg('运行中', 'ac')}${bdg('成功', 'ok')}${bdg('失败', 'err')}${bdg('待确认', 'warn')}${bdg('信息', 'info')}${cnt(3)}${cnt(12, 'mute')}${kbd('⌘', 'K')}</div>
          <div class="f g10 ai">${tg(1)}${tg(0)}${seg(['左右对比', '统一'], 0)}${seg(['Agent', 'IDE'], 1, 'sm')}<div class="inp" style="width:150px">${i('search', 's12')}<span>搜索…</span></div></div>`)}
        ${sec('列表行与状态点', `<div class="card" style="padding:3px"><div class="sb-row on">${C_dot('run')}<span class="grow">选中行 · 运行中</span><span class="tx3 s10">5 分钟</span></div><div class="sb-row">${C_dot('ask')}<span class="grow">待许可</span></div><div class="sb-row">${C_dot('unread')}<span class="grow">未读</span></div><div class="sb-row">${C_dot('idle')}<span class="grow">空闲</span></div></div>`)}
        ${sec('卡片与弹层', `<div class="f g10"><div class="card pad grow"><b>卡片</b><div class="tx3 s11">1px 描边 + 白底,无阴影</div></div><div class="pop grow" style="padding:8px 10px"><b>弹层</b><div class="tx3 s11">唯一带阴影的层级</div></div></div>`)}
        ${sec('Diff 与代码', `<div class="card" style="overflow:hidden;padding:3px 0"><div class="dl del"><span class="no">42</span><span class="tx">    self.routes.get(p)</span></div><div class="dl add"><span class="no">42</span><span class="tx">    self.routes.get(key)</span></div></div>`)}
      </div></div>`,
    }],
  });
  function C_dot(k) { return { run: '<span class="spin"></span>', ask: '<span class="dot ring"></span>', unread: '<span class="dot info"></span>', idle: '<span class="dot" style="opacity:.45"></span>' }[k]; }

  // 现状问题清单(决策页使用)
  window.PROBLEMS = [
    ['右栏标签被截断', '「终端」只剩一个「终」字,标签条图标与文字挤在一起。', '统一「图标 + 文字 + 计数」,放不下时降级为纯图标 + 溢出菜单;顶栏与会话顶栏同高对齐。', 'chat / rightpanel'],
    ['输入框工具栏在窄栏被压成「…」', '模式 / 模型 / 思考 / MCP 四个 chip 在 IDE 对话栏里只剩省略号。', '合并为「运行配置」胶囊(模式 · 模型 · 思考),窄栏只显示模型;MCP 单独一个计数图标。', 'composer / ide'],
    ['右下角三个无标签圆图标', '圆圈、魔棒、停止互相难以区分,发送态不清楚。', '上下文环(带百分比,点开明细)+ 魔棒仅有草稿时出现 + 发送 / 停止合一按钮。', 'composer'],
    ['侧栏会话标题被截成四个字', '单行条目里标题 + 时间 + 状态抢位置。', '双行条目:第一行标题,第二行分支 / 改动 / 状态;状态点统一在左。', 'chat'],
    ['侧栏占三行的次级入口', '「新对话 / 自动化 / 自定义」占了三行高度。', '「新对话」保留整行,其余入口收成一排图标条(或进入图标轨)。', 'chat'],
    ['决策卡片靠鼠标', '权限 / 提问 / 计划的按钮小,没有统一快捷键。', '所有选项加数字 / 字母键位,⏎ 确认、Esc 拒绝;权限标出风险类别与等待时长。', 'interact'],
    ['子 Agent / 任务入口分散', '任务坞、右栏任务标签、会话流卡片三处重复。', '会话流只留一行摘要,详情统一进「活动」抽屉(带实时输出与停止)。', 'agents'],
    ['设置页行高偏大、留白多', '22 个分类各自满宽,看不出哪些被改过。', '行高 40→34、内容限宽 660、已修改小点 + 只看已修改、搜索 ⌘F 过滤。', 'settings-general'],
    ['自定义页首屏被安装表单占据', '药丸标签 + 安装输入框 + 满宽表单,没有详情视图。', 'master-detail 三栏:类别 / 列表 / 详情;安装改为按钮。', 'customize'],
    ['画布 / 自动化 / 浏览器页头各不相同', '三个页面三种头部,返回与主操作位置不一致。', '统一页头:返回 / 图标 / 标题 / 视图切换 / 全局动作 / Agent|IDE。', 'canvas / automation / browser'],
    ['IDE 左栏大纲 / 时间线占大块空白', '没打开文件时大纲区只有「没有打开的文件」。', '大纲与时间线默认折成标题行,文件树独占高度。', 'ide'],
    ['IDE 欢迎页只有一列快捷键', '没有最近文件与待办入口。', '两列快捷键或「工作台概览」(最近文件 / 待审查改动 / 让 Claude 来)。', 'ide-welcome'],
    ['命令面板盲选', '只有列表,选会话 / 文件看不到内容。', '右侧预览:会话最近回复、文件片段、命令说明。', 'palette'],
    ['额度只能点开才能看', '5 小时 / 每周额度藏在上下文面板里。', '侧栏底部放常驻细条;告急时会话顶部横幅提醒。', 'chat / notices'],
  ];
})();
