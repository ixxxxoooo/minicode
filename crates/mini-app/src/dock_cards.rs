//! 输入框上方的「待你处理」卡片(对标 OhMyAgent):权限请求、提问向导。
//! 这些卡片不进时间线;处理完后时间线上只留一行结果(见 `ask_cards`)。计划待批准不在这里:
//! 它只在时间线里留一张卡片(`plan_card`),Build 按钮在卡片和右栏「计划」标签上。
//!
//! @author ixxxxoooo

use crate::model::SessionModel;
use crate::tool_card::tool_icon;
use crate::tool_content::render_content;
use crate::ui::{icon, mono_box};
use gpui_kit::assets::IconName;
use gpui_kit::component::{ActiveTheme, h_flex, v_flex};
use gpui_kit::prelude::FluentBuilder as _;
use gpui_kit::*;
use mini_core::*;

/// 待处理的请求(从时间线里挑出还没有结果的)。
pub enum Pending<'a> {
    Permission(&'a PermissionRequest),
    Question(&'a QuestionRequest),
}

/// 找出所有等待用户处理的请求:权限全部列出,提问只呈现最早的一个(其余计入「还有 N 个」)。
///
/// 参数:`items` 时间线条目。返回:按时间顺序的待处理请求。
pub fn pending_of(items: &[TimelineItem]) -> Vec<Pending<'_>> {
    let mut out = Vec::new();
    let mut asked = false;
    for it in items {
        match it {
            TimelineItem::Permission(p) if p.resolved.is_none() => out.push(Pending::Permission(p)),
            TimelineItem::Question(q) if q.answers.is_none() && !asked => {
                asked = true;
                out.push(Pending::Question(q));
            }
            _ => {}
        }
    }
    out
}

/// 某个条目是否是还在等待处理的权限 / 提问(它们只在输入框上方显示,时间线里隐藏)。
pub fn hidden_in_timeline(item: &TimelineItem) -> bool {
    matches!(item, TimelineItem::Permission(p) if p.resolved.is_none()) || matches!(item, TimelineItem::Question(q) if q.answers.is_none())
}

/// 输入框上方的卡片区;没有待处理请求时返回 `None`。
///
/// 参数:`session` 会话模型。返回:居中、与输入框同宽的卡片栈。
pub fn render(session: &Entity<SessionModel>, cx: &App) -> Option<AnyElement> {
    let m = session.read(cx);
    let list = pending_of(&m.state.items);
    if list.is_empty() {
        return None;
    }
    let queued = m.state.items.iter().filter(|i| matches!(i, TimelineItem::Question(q) if q.answers.is_none())).count().saturating_sub(1);
    let cards: Vec<AnyElement> = list
        .iter()
        .map(|p| match p {
            Pending::Permission(r) => permission(session, r, cx),
            Pending::Question(q) => crate::wizard_card::render(session, q, queued, cx),
        })
        .collect();
    Some(
        div().w_full().flex().justify_center().px_4().pt_1().pb_3()
            .child(v_flex().w_full().max_w(px(crate::timeline::content_max_w(cx))).gap_2().children(cards))
            .into_any_element(),
    )
}

/// 权限卡右上角的类别小标签(写入 / 命令 / 读取…),一眼看出这次要做什么类型的事。
pub fn kind_badge(kind: ToolKind) -> &'static str {
    match kind {
        ToolKind::Edit | ToolKind::Write => "写入",
        ToolKind::Delete => "删除",
        ToolKind::Move => "移动",
        ToolKind::Execute => "命令",
        ToolKind::Read | ToolKind::Search => "读取",
        ToolKind::Fetch => "网络",
        ToolKind::Browser => "浏览器",
        ToolKind::Mcp => "MCP",
        _ => "工具",
    }
}

/// 权限卡(就地替换输入框):标题行(图标 + 标题 + 类别标签)+ 命令 / Diff 预览 + 编号选项行 + 快捷键提示。
/// 允许一次是第一行(强调色底);高风险操作整张卡改用危险色描边。
fn permission(session: &Entity<SessionModel>, req: &PermissionRequest, cx: &App) -> AnyElement {
    let t = cx.theme();
    let cmd = req.tool.input.get("command").and_then(|v| v.as_str()).map(str::to_string).filter(|c| *c != req.tool.title);
    let preview: Vec<AnyElement> = req.tool.content.iter().map(|c| render_content(c, None, "text", None, cx)).collect();
    let danger = crate::perm_keys::is_dangerous(&format!("{} {}", req.tool.title, cmd.as_deref().unwrap_or("")));
    let accent = if danger { t.danger } else { t.primary };
    let focus = session.read(cx).perm_focus.clone().filter(|(id, _)| *id == req.id).map(|(_, f)| f);
    let key_session = session.clone();
    let key = |k: &'static str| div().h(px(16.)).min_w(px(16.)).px_1().flex().items_center().justify_center().rounded(px(4.)).border_1().border_color(t.border).text_size(px(10.)).child(k);
    v_flex().w_full().rounded(px(14.)).border_1().border_color(accent.opacity(0.75)).bg(t.popover).shadow_sm().overflow_hidden()
        .key_context("PermissionCard")
        .when_some(focus, |d, f| d.track_focus(&f))
        // 键盘:⏎ 允许一次、数字键选对应选项、Esc 拒绝
        .on_key_down(move |ev, _, cx| {
            let m = ev.keystroke.modifiers;
            if crate::perm_keys::on_key(&key_session, &ev.keystroke.key, !m.control && !m.platform && !m.alt, cx) {
                cx.stop_propagation();
            }
        })
        .child(
            h_flex().h(px(34.)).px_3().gap_2().items_center()
                .child(icon(if danger { IconName::TriangleAlert } else { tool_icon(req.tool.kind) }, 14., if danger { t.danger } else { t.warning }))
                .child(div().flex_1().min_w_0().truncate().text_size(px(13.)).font_weight(FontWeight::MEDIUM).child(req.tool.title.clone()))
                .when(danger, |d| d.child(div().flex_none().text_size(px(11.)).text_color(t.danger).child("高风险")))
                .child(
                    div().flex_none().h(px(18.)).px_1p5().flex().items_center().rounded(px(4.)).text_size(px(11.)).bg(t.warning.opacity(0.14)).text_color(t.warning)
                        .child(kind_badge(req.tool.kind)),
                ),
        )
        .when_some(cmd, |d, c| d.child(div().px_3().pb_2().child(mono_box(cx).max_h(px(120.)).overflow_hidden().child(crate::code_hl::shell_text(&c, "$ ", cx)))))
        .when(!preview.is_empty(), |d| d.child(
            v_flex().id(SharedString::from(format!("perm-prev-{}", req.id))).w_full().max_h(px(240.)).overflow_y_scroll().gap_2().px_3().pb_2().children(preview),
        ))
        .child(v_flex().px_2().pb_1().gap(px(2.)).children(req.options.iter().enumerate().map(|(i, opt)| {
            let (s, rid, oid) = (session.clone(), req.id.clone(), opt.id.clone());
            let first = opt.kind == PermissionOptionKind::AllowOnce;
            // 行尾快捷键提示:允许一次 ⏎、拒绝 Esc;其余用数字键(行首已有编号)
            let tail = match opt.kind { PermissionOptionKind::AllowOnce => Some("⏎"), PermissionOptionKind::RejectOnce => Some("Esc"), _ => None };
            h_flex().id(SharedString::from(format!("{}-{}", req.id, opt.id))).h(px(30.)).px_2().gap_2().items_center().rounded(px(6.)).cursor_pointer()
                .when(first, |d| d.bg(accent.opacity(0.12)))
                .hover(|d| d.bg(if first { accent.opacity(0.18) } else { t.list_hover }))
                .on_click(move |_, _, cx| { s.update(cx, |m, cx| m.resolve_permission(&rid, &oid, cx)); })
                .child(
                    div().flex_none().size(px(16.)).flex().items_center().justify_center().rounded(px(4.)).text_size(px(10.)).font_weight(FontWeight::MEDIUM)
                        .bg(if first { accent } else { t.muted }).text_color(if first { t.primary_foreground } else { t.muted_foreground })
                        .child((i + 1).to_string()),
                )
                .child(div().flex_1().min_w_0().truncate().text_size(px(13.)).when(first, |d| d.text_color(accent).font_weight(FontWeight::MEDIUM)).child(opt.label.clone()))
                .when_some(tail, |d, k| d.child(key(k)))
        })))
        .child(
            h_flex().h(px(26.)).px_3().gap_2().items_center().border_t_1().border_color(t.border.opacity(0.6)).text_size(px(11.)).text_color(t.muted_foreground)
                .child(key("1-9")).child("直接选择")
                .child(key("⏎")).child("允许")
                .child(key("Esc")).child("拒绝")
                .child(div().flex_1())
                .child("输入框已让位,处理后恢复"),
        )
        .into_any_element()
}

/// 字母徽标(A、B、C…),选中时反色。
pub fn letter_badge(index: usize, selected: bool, cx: &App) -> Div {
    let t = cx.theme();
    let ch = char::from_u32(65 + index as u32).filter(|_| index < 26).unwrap_or('·');
    div().size(px(18.)).flex().items_center().justify_center().rounded(px(5.)).text_size(px(10.5)).font_weight(FontWeight::MEDIUM)
        .bg(if selected { t.primary } else { t.muted })
        .text_color(if selected { t.primary_foreground } else { t.muted_foreground })
        .child(ch.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[::core::prelude::v1::test]
    fn 类别标签() {
        assert_eq!(kind_badge(ToolKind::Edit), "写入");
        assert_eq!(kind_badge(ToolKind::Execute), "命令");
        assert_eq!(kind_badge(ToolKind::Subagent), "工具");
    }
}
