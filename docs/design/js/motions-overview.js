/* 设计稿 v3 · 动效总览(集中目录 + 可重放演示)
   应用内:命令面板「演示:动效总览…」、设置 → 动效与反馈;MINI_ED=motion 直达。
   @author ixxxxoooo */
(function () {
  const { btn, sec, tbl } = U;
  const head = (t, d) => `<h1>${t}</h1><div class="lede">${d}</div>`;
  const replay = (id) => `data-act="motion-replay" data-id="${id}"`;
  const ROWS = [
    ['主界面切换', 'Agent ⇄ IDE 变形', '点标题栏 / ⌘E', '440ms', 'ease-morph', 'morph_belt.go', '是', 'ok'],
    ['主界面切换', '导航分段下划线', '点分段', '180ms', 'ease-spring', 'side.go', '是', 'ok'],
    ['会话与列表', '会话流 a-rise', '切会话', '30ms/行', 'ease-out', 'flow_rise.go', '是', 'ok'],
    ['会话与列表', '导航列表落下', '换项目 / 分段', '8ms/行', 'ease-out', 'side_drop.go', '是', 'part'],
    ['弹层与菜单', 'pop 升起', '菜单 / 圆环', '120–180ms', 'ease-out', 'kit/popshell.go', '是', 'ok'],
    ['弹层与菜单', '居中对话框', '⌘K / 新建', '180ms', 'ease-out', 'popups.go', '是', 'ok'],
    ['反馈状态', '提示条', 'toast', '260ms', 'ease-out', 'kit.ToastSlide', '是', 'ok'],
    ['反馈状态', '骨架 → 内容', '用量页', '700ms', 'ease-io', 'usage_page.go', '是', 'part'],
    ['反馈状态', '校验抖动', '表单错误', '320ms', 'ease-io', 'kit.ShakeInput', '是', 'ok'],
    ['窗口布局', '柱状生长', '用量图', '260ms+', 'ease-out', 'kit.GrowY', '是', 'part'],
  ];
  const demo = (title, inner) => `<div class="card pad" style="margin-top:8px"><div class="f ai g4" style="margin-bottom:6px"><b class="grow">${title}</b>${btn('▶ 重放', 'sm soft', 'play', replay(title))}</div>${inner}</div>`;
  W.pages['motions-overview'] = () => `<div class="ds">${head('动效总览', '把主界面切换、会话列表、弹层与反馈类动效集中在一页,便于对照实现与重放。关闭动效或系统减弱动态时,时长令牌归零。')}
    ${sec('登记表', '与 Go motion_catalog.go 同步;「实现状态」见 status.js。', tbl(
      [['分组', 100], ['名称', 140], ['触发', 160], ['时长', 72], ['缓动', 80], ['涉及文件', 120], ['减少动效', 64], ['状态']],
      ROWS.map(r => r.map((c, i) => i === 7 ? `<span class="stt stt-${c}">${c === 'ok' ? '已实现' : c === 'part' ? '部分' : '未做'}</span>` : c))
    ))}
    ${sec('分组演示', '点 ▶ 重放触发 CSS 关键帧(与 motion.css / states.js 同源)。', `
      ${demo('pop 升起', '<div class="pop a-pop-up" style="width:160px;padding:8px">示例菜单项</div>')}
      ${demo('柱状生长', '<div class="bars" style="height:48px;--n:6">' + [30, 55, 40, 70, 50, 65].map((h, i) => `<i style="height:${h}%;--k:${i};animation:grow-y var(--dur-slow) var(--ease-out) ${i * 24}ms both"></i>`).join('') + '</div>')}
      ${demo('校验抖动', '<span class="inp err a-shake" style="width:200px">错误示例</span>')}
    `)}
    <p class="t3 fs-cap" style="margin-top:12px">上下文圆环「官方 / 第三方」判定见 ctxpop.js 说明 · 用量图悬停见 usage.js。</p></div>`;
})();