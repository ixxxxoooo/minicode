//! 编辑器没有打开任何文件时的「工作台概览」:项目名 + 状态一行,三张卡片(最近文件 / 待审查改动 / 让 Claude 来),
//! 下面是常用快捷键。嵌在 Agent 右栏「文件」标签里很窄时,卡片自动折成一列。
//!
//! @author ixxxxoooo

use crate::file_icons::file_icon_img;
use crate::ide::IdeView;
use crate::ui::icon;
use gpui_kit::assets::IconName;
use gpui_kit::component::{ActiveTheme, h_flex, v_flex};
use gpui_kit::prelude::FluentBuilder as _;
use gpui_kit::*;
use std::path::{Path, PathBuf};

/// 「让 Claude 来」的三个起手式:(标题, 填进对话输入框的提示词)。
const ASKS: [(&str, &str); 3] = [
    ("解释项目结构", "请梳理这个项目的目录结构与关键流程,给出简洁说明。"),
    ("补充缺失的测试", "找出缺少测试的模块,并为它们补充单元测试。"),
    ("审查未提交改动", "请审查当前未提交的改动,指出问题与改进建议。"),
];

/// 常用快捷键(名称, 键位)。
const KEYS: [(&str, &str); 6] = [
    ("快速打开文件", "⌘P"),
    ("命令面板", "⇧⌘P"),
    ("在项目中搜索", "⇧⌘F"),
    ("转到符号", "⇧⌘O"),
    ("显示 / 隐藏终端", "⌃`"),
    ("引用选中内容到对话", "⌘L"),
];

/// 状态一行:分支 · 未提交改动 · 上次打开的文件(没有的项省略)。
pub fn status_line(branch: Option<&str>, changes: usize, last: Option<&str>) -> String {
    let mut parts: Vec<String> = Vec::new();
    if let Some(b) = branch {
        parts.push(b.to_string());
    }
    if changes > 0 {
        parts.push(format!("{changes} 项未提交改动"));
    }
    if let Some(l) = last {
        parts.push(format!("上次打开 {l}"));
    }
    parts.join(" · ")
}

/// 把路径拆成「文件名」与「相对项目根的上级目录」(根目录下的文件上级目录为空)。
pub fn split_name(path: &Path, root: &str) -> (String, String) {
    let name = path.file_name().map(|n| n.to_string_lossy().to_string()).unwrap_or_default();
    let dir = path.parent().and_then(|p| p.strip_prefix(root).ok()).map(|p| p.to_string_lossy().to_string()).unwrap_or_default();
    (name, if dir.is_empty() { String::new() } else { format!("{dir}/") })
}

/// 一张概览卡片的外壳:标题行 + 内容。
fn card(title: &'static str, ic: IconName, body: Vec<AnyElement>, cx: &App) -> Div {
    let t = cx.theme();
    v_flex().flex_1().min_w(px(210.)).p_3().gap_1p5().rounded(px(10.)).border_1().border_color(t.border).bg(t.popover)
        .child(h_flex().gap_1p5().items_center().child(icon(ic, 12., t.muted_foreground)).child(div().text_size(px(12.)).font_weight(FontWeight::SEMIBOLD).child(title)))
        .child(v_flex().gap(px(1.)).children(body))
}

/// 渲染概览。
///
/// 参数:`view` 编辑器(取根目录与最近文件);`cx` 编辑器上下文。
pub fn render(view: &IdeView, cx: &mut Context<IdeView>) -> AnyElement {
    let t = cx.theme().clone();
    let root = view.root.clone();
    let project = Path::new(&root).file_name().map(|n| n.to_string_lossy().to_string()).unwrap_or_default();
    // 工作台里的分支与 Git 改动(编辑器单独使用时没有工作台,这两项为空)
    let wb = cx.try_global::<crate::WorkbenchRef>().and_then(|r| r.0.upgrade());
    let (branch, changed): (Option<String>, Vec<(String, char)>) = match &wb {
        Some(wb) => {
            let w = wb.read(cx);
            let scm = w.scm.read(cx);
            let top = scm.top.clone().unwrap_or_else(|| root.clone());
            let items = scm.groups.staged.iter().chain(&scm.groups.changes).map(|i| (format!("{top}/{}", i.path()), i.kind)).collect::<Vec<_>>();
            (w.active_branch(), items)
        }
        None => (None, Vec::new()),
    };
    // 同一文件两组都有改动时只列一次
    let mut seen = std::collections::HashSet::new();
    let changed: Vec<(String, char)> = changed.into_iter().filter(|(p, _)| seen.insert(p.clone())).collect();
    let recent: Vec<PathBuf> = view.recent.iter().map(PathBuf::from).filter(|p| p.exists()).take(4).collect();
    let last = recent.first().map(|p| split_name(p, &root).0);
    let status = status_line(branch.as_deref(), changed.len(), last.as_deref());

    let file_row = |id: String, path: PathBuf, tail: Option<(char, Hsla)>, cx: &mut Context<IdeView>| {
        let (name, dir) = split_name(&path, &root);
        h_flex().id(SharedString::from(id)).h(px(24.)).px_1().gap_1p5().items_center().rounded(px(5.)).cursor_pointer()
            .hover(|d| d.bg(t.list_hover))
            .on_click(cx.listener(move |this, _, window, cx| this.open(path.clone(), false, window, cx)))
            .child(file_icon_img(&name, false, false, 13.))
            .child(div().flex_none().text_size(px(12.5)).child(name.clone()))
            .child(div().flex_1().min_w_0().truncate().text_size(px(11.)).text_color(t.muted_foreground).child(dir))
            .when_some(tail, |d, (k, c)| d.child(div().flex_none().text_size(px(11.)).font_weight(FontWeight::MEDIUM).text_color(c).child(k.to_string())))
            .into_any_element()
    };

    let recent_rows: Vec<AnyElement> = if recent.is_empty() {
        vec![div().py_1().text_size(px(12.)).text_color(t.muted_foreground).child("还没有打开过文件").into_any_element()]
    } else {
        recent.iter().enumerate().map(|(i, p)| file_row(format!("ov-recent-{i}"), p.clone(), None, cx)).collect()
    };
    let change_rows: Vec<AnyElement> = if changed.is_empty() {
        vec![div().py_1().text_size(px(12.)).text_color(t.muted_foreground).child("工作区很干净").into_any_element()]
    } else {
        changed.iter().take(4).enumerate().map(|(i, (p, k))| {
            let color = match k { 'A' | 'U' => t.success, 'D' => t.danger, _ => t.warning };
            file_row(format!("ov-change-{i}"), PathBuf::from(p), Some((*k, color)), cx)
        }).collect()
    };
    let ask_rows: Vec<AnyElement> = ASKS.iter().enumerate().map(|(i, (title, prompt))| {
        let (wb, prompt) = (wb.clone(), *prompt);
        h_flex().id(SharedString::from(format!("ov-ask-{i}"))).h(px(24.)).px_1().gap_1p5().items_center().rounded(px(5.)).cursor_pointer()
            .hover(|d| d.bg(t.list_hover))
            .on_click(move |_, window, cx| {
                // 把提示词填进当前会话的输入框(不自动发送)
                if let Some(wb) = &wb {
                    wb.update(cx, |w, cx| {
                        if let Some((_, composer)) = w.chat_parts() {
                            composer.update(cx, |c, cx| { c.set_text(prompt, window, cx); c.focus(window, cx); });
                        }
                    });
                }
            })
            .child(icon(IconName::ArrowRight, 11., t.primary))
            .child(div().text_size(px(12.5)).child(*title))
            .into_any_element()
    }).collect();

    let keys = h_flex().flex_wrap().gap_x_4().gap_y_1().children(KEYS.iter().map(|(name, key)| {
        h_flex().gap_1p5().items_center().text_size(px(12.)).text_color(t.muted_foreground)
            .child(div().h(px(18.)).px_1().flex().items_center().rounded(px(4.)).border_1().border_color(t.border).text_size(px(10.5)).child(*key))
            .child(*name)
    }));

    div().id("ide-overview").size_full().overflow_y_scroll().flex().justify_center().child(
        v_flex().w_full().max_w(px(760.)).p_6().gap_4().justify_center()
            .child(
                v_flex().gap_0p5()
                    .child(div().text_size(px(20.)).font_weight(FontWeight::SEMIBOLD).child(project))
                    .when(!status.is_empty(), |d| d.child(div().text_size(px(12.)).text_color(t.muted_foreground).child(status))),
            )
            .child(
                h_flex().flex_wrap().gap_3().items_start()
                    .child(card("最近文件", IconName::Clock, recent_rows, cx))
                    .child(card("待审查改动", IconName::FileDiff, change_rows, cx))
                    .child(card("让 Claude 来", IconName::Sparkles, ask_rows, cx)),
            )
            .child(v_flex().gap_1p5().child(div().text_size(px(12.)).font_weight(FontWeight::SEMIBOLD).child("常用快捷键")).child(keys)),
    ).into_any_element()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[::core::prelude::v1::test]
    fn 状态行拼接() {
        assert_eq!(status_line(Some("main"), 3, Some("router.rs")), "main · 3 项未提交改动 · 上次打开 router.rs");
        assert_eq!(status_line(None, 0, None), "");
    }

    #[::core::prelude::v1::test]
    fn 拆出文件名与相对目录() {
        assert_eq!(split_name(Path::new("/p/src/a.rs"), "/p"), ("a.rs".to_string(), "src/".to_string()));
        assert_eq!(split_name(Path::new("/p/Cargo.toml"), "/p"), ("Cargo.toml".to_string(), String::new()));
    }
}
