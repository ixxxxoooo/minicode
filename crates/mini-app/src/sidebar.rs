//! 左侧项目栏:项目 → 会话树,置顶、状态点、搜索入口、设置入口。

use crate::project_add;
use crate::project_menu;
use crate::session_menu;
use crate::sidebar_filter::{SessFilter, visible_sessions};
use crate::ui::{icon, status_dot};
use crate::workbench::{Page, Workbench};
use gpui_kit::assets::IconName;
use gpui_kit::component::button::{Button, ButtonVariants};
use gpui_kit::component::menu::{ContextMenuExt as _, DropdownMenu as _};
use gpui_kit::component::{ActiveTheme, Selectable, Sizable, h_flex, v_flex};
use gpui_kit::prelude::FluentBuilder as _;
use gpui_kit::*;
use mini_core::*;

/// 渲染一个会话行(两行):第一行状态点 + 标题(+ 后台任务数),第二行状态说明 + 相对时间;
/// 悬停时标题让出右侧位置,置顶 / 归档按钮浮现。
fn session_row(wb: &Workbench, project_ix: usize, meta: &SessionMeta, cx: &mut Context<Workbench>) -> impl IntoElement {
    let t = cx.theme().clone();
    let active = wb.active.as_deref() == Some(meta.id.as_str());
    // 活动会话的实时状态优先于演示数据里的静态状态
    let badge = wb.live_badge(&meta.id, meta.badge, cx);
    let bg_tasks = wb.live_task_count(&meta.id, cx);
    let id = meta.id.clone();
    let group = SharedString::from(format!("grp-{}", meta.id));
    let (pin_id, arch_id) = (meta.id.clone(), meta.id.clone());
    let (pinned, archived) = (meta.pinned, meta.archived);
    // 按钮点击不应触发整行的「打开会话」
    let stop = |_: &MouseDownEvent, _: &mut Window, cx: &mut App| cx.stop_propagation();
    let cwd = wb.projects.get(project_ix).map(|p| p.path.clone()).unwrap_or_default();
    let menu = session_menu::build(cx.entity().downgrade(), meta, cwd, wb.demo);
    // 第二行:状态说明(运行中 / 等待你 / 未读用强调色)+ 相对时间
    let (state_text, state_color) = match badge {
        SessionBadge::Running => (Some("运行中"), t.primary),
        SessionBadge::Waiting => (Some("等待你的许可"), t.primary),
        SessionBadge::Unread => (Some("有新回复"), t.info),
        SessionBadge::Idle => (None, t.muted_foreground),
    };
    let time = short_time(&meta.updated);
    h_flex()
        .id(SharedString::from(format!("sess-{}", meta.id)))
        .group(group.clone())
        .relative()
        .w_full()
        .h(px(SESSION_ROW_H))
        .pl(px(10.))
        .pr_2()
        .gap_2()
        .items_start()
        .pt(px(4.))
        .rounded(px(7.))
        .cursor_pointer()
        // 选中行:中性的淡灰底(不抢项目行与状态点的颜色);悬停更淡
        .when(active, |d| d.bg(t.foreground.opacity(0.09)))
        .when(!active, |d| d.hover(|d| d.bg(t.foreground.opacity(0.05))))
        .on_click(cx.listener(move |this, _, window, cx| this.open_session(project_ix, &id, window, cx)))
        .child(div().w(px(8.)).mt(px(5.)).flex_none().child(status_dot(badge, cx).unwrap_or_else(|| div().size(px(6.)).m(px(1.)).rounded_full().flex_none().bg(t.muted_foreground.opacity(0.55)).into_any_element())))
        .child(
            v_flex().flex_1().min_w_0().gap(px(1.))
                .group_hover(group.clone(), |d| d.pr(px(44.)))
                .child(
                    h_flex().w_full().gap_1().items_center()
                        .child(div().flex_1().min_w_0().truncate().text_size(px(13.)).when(active, |d| d.font_weight(FontWeight::MEDIUM)).when(archived, |d| d.text_color(t.muted_foreground)).child(meta.title.clone()))
                        .when(bg_tasks > 0, |d| {
                            d.child(
                                h_flex().flex_none().h(px(16.)).px_1().gap_0p5().items_center().rounded(px(8.)).bg(t.foreground.opacity(0.08)).text_size(px(10.)).text_color(t.muted_foreground)
                                    .child(icon(IconName::Activity, 9., t.primary))
                                    .child(format!("{bg_tasks}")),
                            )
                        }),
                )
                .child(
                    h_flex().w_full().gap_1().items_center().text_size(px(11.)).text_color(t.muted_foreground).overflow_hidden()
                        .when(pinned, |d| d.child(icon(IconName::Pin, 10., t.muted_foreground)))
                        .when_some(state_text, |d, s| d.child(div().flex_none().text_color(state_color).child(s)))
                        .when(state_text.is_some() && !time.is_empty(), |d| d.child(div().flex_none().child("·")))
                        .child(div().truncate().child(time)),
                ),
        )
        .child(
            h_flex().absolute().top(px(5.)).right(px(6.)).items_center().gap_0p5().invisible()
                .group_hover(group, |d| d.visible())
                .on_mouse_down(MouseButton::Left, stop)
                .child(Button::new(SharedString::from(format!("pin-{}", meta.id))).xsmall().ghost()
                    .icon(IconName::Pin).tooltip(if pinned { "取消置顶" } else { "置顶" })
                    .on_click(cx.listener(move |this, _, _, cx| this.toggle_pin_session(&pin_id, cx))))
                .child(Button::new(SharedString::from(format!("arch-{}", meta.id))).xsmall().ghost()
                    .icon(IconName::Archive).tooltip(if archived { "取消归档" } else { "归档" })
                    .on_click(cx.listener(move |this, _, _, cx| this.set_session_archived(&arch_id, !archived, cx)))),
        )
        .context_menu(menu)
}

/// 会话行高度(两行:标题 + 状态 / 时间)。
const SESSION_ROW_H: f32 = 38.;

/// 把「5 分钟前 / 1 小时前 / 昨天」压成侧栏用的短时间(「5分钟」「1小时」「昨天」)。
///
/// 参数:`s` 原始相对时间文案。返回:去掉「前」和空格后的短文案。
fn short_time(s: &str) -> String {
    s.trim_end_matches('前').replace(' ', "")
}

/// 侧栏顶部的一行入口(图标 + 文字 + 可选快捷键)。
fn nav_row(
    id: &'static str,
    ic: IconName,
    label: &'static str,
    shortcut: Option<&'static str>,
    active: bool,
    cx: &mut Context<Workbench>,
    on_click: impl Fn(&mut Workbench, &mut Window, &mut Context<Workbench>) + 'static,
) -> impl IntoElement {
    let t = cx.theme().clone();
    h_flex().id(id).h(px(32.)).px_2().gap_2().items_center().rounded(px(6.)).cursor_pointer()
        .when(active, |d| d.bg(t.sidebar_accent))
        .hover(|d| d.bg(t.sidebar_accent))
        .on_click(cx.listener(move |this, _, window, cx| on_click(this, window, cx)))
        .child(icon(ic, 15., if id == "nav-new" { t.primary } else { t.muted_foreground }))
        .child(div().flex_1().text_size(px(13.)).font_weight(FontWeight::MEDIUM).child(label))
        .when_some(shortcut, |d, k| d.child(div().text_size(px(11.)).text_color(t.muted_foreground).child(k)))
}

/// 图标条:自动化 / 画布 / 浏览器 / 自定义四个页面入口(只有图标,悬停显示名称),当前页高亮。
fn nav_strip(wb: &Workbench, cx: &mut Context<Workbench>) -> impl IntoElement {
    let t = cx.theme().clone();
    let items: [(&'static str, IconName, &'static str, Page); 4] = [
        ("nav-auto", IconName::Timer, "自动化", Page::Automation),
        ("nav-canvas", IconName::LayoutDashboard, "画布", Page::Canvas),
        ("nav-browser", IconName::Globe, "浏览器", Page::Browser),
        ("nav-customize", IconName::Plug, "自定义", Page::Customize),
    ];
    h_flex().w_full().gap_1().children(items.into_iter().map(|(id, ic, label, page)| {
        let on = wb.page == page;
        h_flex().id(id).flex_1().h(px(28.)).items_center().justify_center().rounded(px(6.)).cursor_pointer()
            .bg(if on { t.sidebar_accent } else { t.transparent })
            .hover(|d| d.bg(t.sidebar_accent))
            .tooltip(move |window, cx| gpui_kit::component::tooltip::Tooltip::new(label).build(window, cx))
            .on_click(cx.listener(move |this, _, _, cx| this.set_page(page, cx)))
            .child(icon(ic, 15., if on { t.primary } else { t.muted_foreground }))
    }))
}

/// 状态筛选胶囊:全部 / 运行中 N / 待处理 N(计数按全部项目里未归档的会话算)。
fn filter_chips(wb: &Workbench, cx: &mut Context<Workbench>) -> impl IntoElement {
    let t = cx.theme().clone();
    let (mut running, mut waiting) = (0usize, 0usize);
    for p in wb.projects.iter().filter(|p| !wb.ui_state.hidden_projects.contains(&p.path)) {
        for s in p.sessions.iter().filter(|s| !s.archived) {
            match wb.live_badge(&s.id, s.badge, cx) {
                SessionBadge::Running => running += 1,
                SessionBadge::Waiting => waiting += 1,
                _ => {}
            }
        }
    }
    let chips: [(&'static str, &'static str, SessFilter, usize); 3] = [
        ("flt-all", "全部", SessFilter::All, 0),
        ("flt-run", "运行中", SessFilter::Running, running),
        ("flt-wait", "待处理", SessFilter::Waiting, waiting),
    ];
    h_flex().px_2p5().pb_1().gap_1().items_center().children(chips.into_iter().map(|(id, label, f, n)| {
        let on = wb.sess_filter == f;
        h_flex().id(id).h(px(20.)).px_2().gap_1().items_center().rounded(px(10.)).cursor_pointer().text_size(px(11.))
            .bg(if on { t.foreground.opacity(0.09) } else { t.transparent })
            .text_color(if on { t.foreground } else { t.muted_foreground })
            .hover(|d| d.bg(t.foreground.opacity(0.06)))
            .on_click(cx.listener(move |this, _, _, cx| { this.sess_filter = if on && f != SessFilter::All { SessFilter::All } else { f }; cx.notify(); }))
            .child(label)
            .when(n > 0, |d| d.child(div().min_w(px(14.)).h(px(14.)).px_1().flex().items_center().justify_center().rounded(px(7.)).text_size(px(9.5))
                .bg(if f == SessFilter::Waiting { t.primary } else { t.foreground.opacity(0.12) })
                .text_color(if f == SessFilter::Waiting { t.primary_foreground } else { t.muted_foreground })
                .child(n.to_string())))
    }))
}

/// 「显示 / 隐藏已归档会话」按钮。
fn archive_btn(on: bool, cx: &mut Context<Workbench>) -> impl IntoElement {
    Button::new("show-archived").small().ghost().icon(IconName::Archive)
        .tooltip(if on { "隐藏已归档会话" } else { "显示已归档会话" })
        .selected(on)
        .on_click(cx.listener(|this, _, _, cx| this.toggle_show_archived(cx)))
}

/// 一个项目折叠时要显示的状态:等待你处理 > 运行中 > 未读 > 无(已归档的会话不算)。
pub fn summary_badge(sessions: &[SessionMeta]) -> SessionBadge {
    let live = || sessions.iter().filter(|s| !s.archived).map(|s| s.badge);
    [SessionBadge::Waiting, SessionBadge::Running, SessionBadge::Unread].into_iter().find(|b| live().any(|x| x == *b)).unwrap_or(SessionBadge::Idle)
}

/// 渲染整个左侧栏。
pub fn render(wb: &Workbench, cx: &mut Context<Workbench>) -> impl IntoElement {
    let t = cx.theme().clone();
    let projects = wb.projects.clone();
    // 先算出每个项目要显示的会话(已归档的按开关取舍);搜索统一走 ⌘K 命令面板,侧栏不再放过滤框
    let shown: Vec<(usize, Vec<SessionMeta>)> = projects
        .iter()
        .enumerate()
        .filter(|(_, p)| !wb.ui_state.hidden_projects.contains(&p.path))
        .filter_map(|(pi, p)| visible_sessions("", p, wb.show_archived, wb.active.as_deref()).map(|v| (pi, v.into_iter().cloned().collect::<Vec<SessionMeta>>())))
        // 状态筛选(全部 / 运行中 / 待处理):有筛选时只留命中的会话,没有命中的项目整体隐藏
        .map(|(pi, v)| (pi, v.into_iter().filter(|s| wb.sess_filter.keep(wb.live_badge(&s.id, s.badge, cx))).collect::<Vec<_>>()))
        .filter(|(_, v)| wb.sess_filter == SessFilter::All || !v.is_empty())
        .collect();
    v_flex()
        .size_full()
        .bg(t.sidebar)
        .text_color(t.sidebar_foreground)
        .child(wb.left_header(cx))
        .child(
            v_flex().px_2().pt_0p5().pb_1().gap_1()
                .child(nav_row("nav-new", IconName::Plus, "新对话", Some("⌘N"), false, cx,
                    |this, window, cx| this.new_session(this.target_project(), window, cx)))
                .child(nav_strip(wb, cx)),
        )
        .child(
            h_flex().id("ws-head").group("ws-head").px_3().pt_1p5().pb_1().gap_1().items_center().h(px(28.))
                .child(div().flex_1().text_size(px(11.)).text_color(t.muted_foreground).child("工作区"))
                // 操作按钮只在鼠标移到这一行时出现(对标 VS Code 的区块标题);打开了「显示已归档」时该按钮常驻,提示当前处于筛选状态
                .child(
                    h_flex().gap_1().items_center().invisible().group_hover("ws-head", |d| d.visible())
                        .child(Button::new("collapse-projects").xsmall().ghost().icon(IconName::ChevronsDownUp).tooltip("全部折叠")
                            .on_click(cx.listener(|this, _, _, cx| this.collapse_all_projects(cx))))
                        .when(!wb.demo, |d| d.child(Button::new("add-project").xsmall().ghost().icon(IconName::FolderPlus).tooltip("添加项目")
                            .dropdown_menu(project_add::build_menu(cx.entity().downgrade(), project_add::existing_recent(&wb.settings.recent_projects)))))
                        .when(!wb.show_archived, |d| d.child(archive_btn(wb.show_archived, cx))),
                )
                .when(wb.show_archived, |d| d.child(archive_btn(true, cx))),
        )
        .child(filter_chips(wb, cx))
        .child(
            v_flex().id("projects").flex_1().min_h_0().overflow_y_scroll().px_2().py_1().gap_1()
                .on_drop(cx.listener(|this, paths: &ExternalPaths, window, cx| this.drop_projects(paths.paths(), window, cx)))
                .children(shown.into_iter().map(|(pi, sessions)| {
                    let p = &projects[pi];
                    let expanded = p.expanded;
                    // 当前会话所在的项目:文件夹图标用主色,其余灰色
                    let holds_active = p.sessions.iter().any(|s| wb.active.as_deref() == Some(s.id.as_str()));
                    let pinned = wb.ui_state.pinned_projects.contains(&p.path);
                    let workspace = wb.ui_state.workspace_by_dir(&p.path).cloned();
                    let worktree = wb.is_worktree_project(&p.path);
                    let menu = project_menu::build(cx.entity().downgrade(), p.path.clone(), p.name.clone(), pinned, workspace.is_some(), worktree);
                    let more_menu = project_menu::build(cx.entity().downgrade(), p.path.clone(), p.name.clone(), pinned, workspace.is_some(), worktree);
                    let group = SharedString::from(format!("proj-group-{pi}"));
                    v_flex()
                        .child(
                            h_flex().id(SharedString::from(format!("proj-{pi}"))).group(group.clone()).h(px(34.)).px_2().gap_2p5().items_center().rounded(px(8.)).cursor_pointer()
                                // 项目行不带常驻底色,只在悬停时有一层淡灰(层次交给选中的会话行)
                                .hover(|d| d.bg(t.foreground.opacity(0.05)))
                                .on_click(cx.listener(move |this, _, _, cx| this.toggle_project(pi, cx)))
                                // 展开 = 蓝色打开的文件夹,折叠 = 灰色合上的文件夹
                                .child(icon(
                                    if workspace.is_some() { IconName::Folders } else if worktree { IconName::GitFork } else if expanded { IconName::FolderOpen } else { IconName::Folder },
                                    15.,
                                    if holds_active { t.primary } else { t.muted_foreground },
                                ))
                                .child(div().flex_1().truncate().text_size(px(13.)).font_weight(FontWeight::MEDIUM).child(p.name.clone()))
                                .when(pinned, |d| d.child(icon(IconName::Pin, 11., t.muted_foreground)))
                                .when_some(workspace.as_ref(), |d, w| d.child(div().text_size(px(11.)).text_color(t.muted_foreground).child(format!("{} 个文件夹", w.folders.len()))))
                                // 折叠时把项目里最需要关注的会话状态带到这一行:等待你 > 运行中 > 未读
                                .when(!expanded, |d| d.children(status_dot(summary_badge(&p.sessions), cx)))
                                // 悬停时才出现的操作:在这个项目里新建会话、更多菜单(同右键菜单)
                                .child(
                                    h_flex().gap_0p5().items_center().when(!expanded, |d| d.invisible()).group_hover(group.clone(), |d| d.visible())
                                        .on_mouse_down(MouseButton::Left, |_, _, cx| cx.stop_propagation())
                                        .child(Button::new(SharedString::from(format!("proj-new-{pi}"))).xsmall().ghost().icon(IconName::Plus).tooltip("在此项目新建会话")
                                            .on_click(cx.listener(move |this, _, window, cx| this.new_session(pi, window, cx))))
                                        .child(Button::new(SharedString::from(format!("proj-more-{pi}"))).xsmall().ghost().icon(IconName::Ellipsis).tooltip("更多")
                                            .dropdown_menu(more_menu)),
                                )
                                .context_menu(menu),
                        )
                        .when(expanded, |d| d.child(v_flex().gap(px(2.)).pt(px(2.)).pb(px(6.)).children(sessions.iter().map(|s| session_row(wb, pi, s, cx).into_any_element()))))
                })),
        )
        .child(
            h_flex().h(px(40.)).px_2().gap_1().items_center()
                .child(Button::new("nav-settings").small().ghost().icon(IconName::Settings).tooltip("设置")
                    .on_click(cx.listener(|this, _, _, cx| this.set_page(Page::Settings, cx))))
                .when_some(crate::ui::env_badges(wb.demo, cx), |d, b| d.child(b))
                // 更新胶囊固定在最右侧:下载中显示百分比,下载完成显示「更新」
                .child(div().flex_1().min_w_0())
                .when_some(update_capsule(wb, cx), |d, b| d.child(div().flex_none().pr_1().child(b)))
        )
}

/// 有新版本时,设置按钮右侧的更新胶囊:发现新版本 → 下载更新;下载完成 → 重启更新;下载中显示进度文字。
/// 没有更新(或不是可更新的状态)时返回 `None`。
fn update_capsule(wb: &Workbench, cx: &mut Context<Workbench>) -> Option<Stateful<Div>> {
    use crate::updater::UpdateState as U;
    let t = cx.theme().clone();
    let (text, tip): (String, String) = match &wb.update {
        U::Available(i) => ("下载中".into(), format!("v{} · 正在准备下载", i.version)),
        U::Downloading(i, p) => (
            // 下载阶段带上百分比(或已下载量),校验 / 解包阶段直接写阶段名
            match p { mini_config::update::Progress::Downloading { .. } => format!("下载中 {}", p.short()), _ => p.short() },
            format!("v{} · {}", i.version, p.detail()),
        ),
        U::Ready(_) => ("更新".into(), "新版本已下载并校验,点击重启完成更新".into()),
        _ => return None,
    };
    let clickable = !matches!(wb.update, U::Downloading(..));
    Some(
        div().id("update-capsule").when(clickable, |d| d.cursor_pointer()).child(crate::ui::pill(text, t.primary_foreground, t.primary))
            .tooltip(move |window, cx| gpui_kit::component::tooltip::Tooltip::new(tip.clone()).build(window, cx))
            .on_click(cx.listener(|this, _, _, cx| match this.update {
                U::Available(_) => this.download_update(cx),
                U::Ready(_) => this.install_update(cx),
                _ => {}
            })),
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    fn meta(badge: SessionBadge, archived: bool) -> SessionMeta {
        SessionMeta { id: "x".into(), title: "t".into(), updated: String::new(), pinned: false, archived, badge }
    }

    #[::core::prelude::v1::test]
    fn 折叠状态按优先级汇总() {
        assert_eq!(summary_badge(&[]), SessionBadge::Idle);
        assert_eq!(summary_badge(&[meta(SessionBadge::Unread, false), meta(SessionBadge::Running, false)]), SessionBadge::Running);
        assert_eq!(summary_badge(&[meta(SessionBadge::Running, false), meta(SessionBadge::Waiting, false)]), SessionBadge::Waiting);
        assert_eq!(summary_badge(&[meta(SessionBadge::Waiting, true)]), SessionBadge::Idle);
    }
}
