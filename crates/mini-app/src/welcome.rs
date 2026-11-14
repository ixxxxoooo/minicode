//! 新会话欢迎页(居中问候 + 起手式):会话还没有任何内容时代替空白时间线。
//!
//! 结构自上而下:品牌图标 → 问候语 → 项目状态一行 → 输入框 → 四张「起手式」卡片 → 最近会话。
//! 点起手式只是把提示词填进输入框(不自动发送),点最近会话直接打开。
//!
//! @author ixxxxoooo

use crate::composer::Composer;
use crate::ui::{icon, status_dot};
use crate::workbench::Workbench;
use gpui_kit::assets::IconName;
use gpui_kit::component::{ActiveTheme, h_flex, v_flex};
use gpui_kit::prelude::FluentBuilder as _;
use gpui_kit::*;
use mini_core::SessionBadge;

/// 一张起手式卡片:图标、标题、说明、填进输入框的提示词。
struct Starter {
    id: &'static str,
    icon: IconName,
    title: &'static str,
    detail: String,
    prompt: &'static str,
}

/// 四张起手式卡片;`changes` 是未提交改动的文件数,用于「审查未提交改动」的说明。
fn starters(changes: usize) -> Vec<Starter> {
    vec![
        Starter { id: "ws-fix", icon: IconName::Wrench, title: "修复一个 Bug", detail: "描述现象,我来定位并修复".into(), prompt: "帮我修复一个 Bug:" },
        Starter { id: "ws-explain", icon: IconName::FileText, title: "解释这个项目", detail: "梳理目录结构与关键流程".into(), prompt: "请梳理这个项目的目录结构与关键流程,给出简洁说明。" },
        Starter { id: "ws-test", icon: IconName::CircleCheck, title: "补充测试", detail: "为最近改动补齐单元测试".into(), prompt: "为最近改动的代码补充单元测试。" },
        Starter {
            id: "ws-review",
            icon: IconName::Eye,
            title: "审查未提交改动",
            detail: if changes > 0 { format!("{changes} 个文件待审查") } else { "检查当前工作区的改动".into() },
            prompt: "请审查当前未提交的改动,指出问题与改进建议。",
        },
    ]
}

/// 项目状态一行:分支 · 未提交改动 · 上次会话时间(没有的项省略)。
fn status_line(branch: Option<&str>, changes: usize, last: Option<&str>) -> String {
    let mut parts: Vec<String> = Vec::new();
    if let Some(b) = branch {
        parts.push(b.to_string());
    }
    if changes > 0 {
        parts.push(format!("{changes} 项未提交改动"));
    }
    if let Some(l) = last.filter(|l| !l.is_empty()) {
        parts.push(format!("上次会话 {l}"));
    }
    parts.join(" · ")
}

/// 渲染欢迎页。
///
/// 参数:`wb` 工作台;`composer` 当前会话的输入框;`footer` 输入框下方的页脚。
pub fn render(wb: &Workbench, composer: Entity<Composer>, footer: AnyElement, cx: &mut Context<Workbench>) -> AnyElement {
    let t = cx.theme().clone();
    let w = crate::timeline::content_max_w(cx).min(640.);
    let project = wb.current_project_ix().and_then(|ix| wb.projects.get(ix)).cloned();
    let changes = wb.scm.read(cx).count(cx);
    let name = project.as_ref().map(|p| p.name.clone());
    let last = project.as_ref().and_then(|p| p.sessions.iter().find(|s| !s.archived && wb.active.as_deref() != Some(s.id.as_str()))).map(|s| s.updated.clone());
    let branch = wb.active_branch();
    let status = status_line(branch.as_deref(), changes, last.as_deref());
    // 最近会话(不含当前这个空会话与已归档的),最多三条
    let recent: Vec<_> = project
        .as_ref()
        .map(|p| p.sessions.iter().filter(|s| !s.archived && wb.active.as_deref() != Some(s.id.as_str())).take(3).cloned().collect())
        .unwrap_or_default();
    let project_ix = wb.current_project_ix().unwrap_or(0);

    let cards = starters(changes).into_iter().map(|s| {
        let c = composer.clone();
        let prompt = s.prompt;
        h_flex().id(s.id).w_full().min_w_0().gap_2().p_2().items_start().rounded(px(8.)).border_1().border_color(t.border).bg(t.popover).cursor_pointer()
            .hover(|d| d.border_color(t.primary.opacity(0.5)).bg(t.list_hover))
            .on_click(move |_, window, cx| {
                c.update(cx, |c, cx| { c.set_text(prompt, window, cx); c.focus(window, cx); });
            })
            .child(
                div().flex_none().size(px(24.)).flex().items_center().justify_center().rounded(px(6.)).bg(t.primary.opacity(0.12))
                    .child(icon(s.icon, 13., t.primary)),
            )
            .child(
                v_flex().flex_1().min_w_0()
                    .child(div().text_size(px(13.)).font_weight(FontWeight::MEDIUM).truncate().child(s.title))
                    .child(div().text_size(px(11.)).text_color(t.muted_foreground).truncate().child(s.detail)),
            )
    }).collect::<Vec<_>>();
    let mut rows = cards.into_iter();
    let mut grid = v_flex().w_full().gap_2();
    while let Some(a) = rows.next() {
        let b = rows.next();
        grid = grid.child(h_flex().w_full().gap_2().child(div().flex_1().min_w_0().child(a)).child(div().flex_1().min_w_0().when_some(b, |d, b| d.child(b))));
    }

    v_flex().size_full().bg(t.background).items_center().overflow_hidden()
        .child(div().flex_1())
        .child(
            v_flex().w_full().max_w(px(w)).items_center().gap_1().px_4().pb_3()
                .child(
                    div().size(px(30.)).flex().items_center().justify_center().rounded(px(8.)).bg(t.primary)
                        .child(icon(IconName::Code, 16., t.primary_foreground)),
                )
                .child(div().pt_1().text_size(px(20.)).font_weight(FontWeight::SEMIBOLD).child(match name {
                    Some(n) => format!("今天想在 {n} 里做点什么?"),
                    None => "今天想做点什么?".to_string(),
                }))
                .when(!status.is_empty(), |d| d.child(div().text_size(px(12.)).text_color(t.muted_foreground).child(status))),
        )
        .child(div().w_full().max_w(px(w)).child(composer))
        .child(div().w_full().max_w(px(w)).px_4().child(footer))
        .child(
            v_flex().w_full().max_w(px(w)).px_4().pt_1().gap_3().child(grid)
                .when(!recent.is_empty(), |d| d.child(
                    v_flex().w_full().gap_0p5()
                        .child(div().px_1().pb_0p5().text_size(px(11.)).text_color(t.muted_foreground).child("最近会话"))
                        .children(recent.into_iter().map(|s| {
                            let id = s.id.clone();
                            let badge = wb.live_badge(&s.id, s.badge, cx);
                            let tail = match badge {
                                SessionBadge::Running => "运行中".to_string(),
                                SessionBadge::Waiting => "等待你".to_string(),
                                _ => s.updated.clone(),
                            };
                            let hot = matches!(badge, SessionBadge::Running | SessionBadge::Waiting);
                            h_flex().id(SharedString::from(format!("ws-recent-{}", s.id))).h(px(28.)).px_1().gap_2().items_center().rounded(px(6.)).cursor_pointer()
                                .hover(|d| d.bg(t.list_hover))
                                .on_click(cx.listener(move |this, _, window, cx| this.open_session(project_ix, &id, window, cx)))
                                .child(div().w(px(8.)).flex_none().child(status_dot(badge, cx).unwrap_or_else(|| div().size(px(6.)).m(px(1.)).rounded_full().bg(t.muted_foreground.opacity(0.55)).into_any_element())))
                                .child(div().flex_1().min_w_0().truncate().text_size(px(12.5)).child(s.title.clone()))
                                .child(div().flex_none().text_size(px(11.)).text_color(if hot { t.primary } else { t.muted_foreground }).child(tail))
                        })),
                )),
        )
        .child(div().flex_1())
        .into_any_element()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[::core::prelude::v1::test]
    fn 状态行按有无拼接() {
        assert_eq!(status_line(Some("main"), 3, Some("5 分钟前")), "main · 3 项未提交改动 · 上次会话 5 分钟前");
        assert_eq!(status_line(Some("main"), 0, None), "main");
        assert_eq!(status_line(None, 0, Some("")), "");
    }

    #[::core::prelude::v1::test]
    fn 起手式有四张且审查卡带改动数() {
        let s = starters(4);
        assert_eq!(s.len(), 4);
        assert_eq!(s[3].detail, "4 个文件待审查");
        assert_eq!(starters(0)[3].detail, "检查当前工作区的改动");
    }
}
