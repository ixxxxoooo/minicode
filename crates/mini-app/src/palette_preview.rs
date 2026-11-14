//! 命令面板右侧的预览栏(对标设计稿「列表 + 预览」):选中项是文件时显示开头几行内容,
//! 其余条目显示标题、来源与回车会做什么。

use crate::palette::{Group, PaletteAction, PaletteItem};
use crate::ui::{MONO, icon};
use gpui_kit::component::{ActiveTheme, h_flex, v_flex};
use gpui_kit::prelude::FluentBuilder as _;
use gpui_kit::*;

/// 预览最多读取的字节数。
const MAX_BYTES: usize = 8 * 1024;
/// 预览最多显示的行数。
pub const MAX_LINES: usize = 16;
/// 单行最多保留的字符数。
const MAX_COLS: usize = 96;

/// 把条目里的路径补成绝对路径:已是绝对路径原样返回,否则接在项目根后面。
pub fn resolve(root: &str, path: &str) -> String {
    if path.starts_with('/') || root.is_empty() { path.to_string() } else { format!("{}/{}", root.trim_end_matches('/'), path) }
}

/// 读文件开头:前 `MAX_LINES` 行,每行截到 `MAX_COLS` 个字符。二进制 / 非 UTF-8 / 读失败返回 `None`。
pub fn file_head(path: &str) -> Option<Vec<String>> {
    use std::io::Read;
    let mut buf = Vec::with_capacity(MAX_BYTES);
    std::fs::File::open(path).ok()?.take(MAX_BYTES as u64).read_to_end(&mut buf).ok()?;
    if buf.contains(&0) {
        return None;
    }
    // 截断点可能落在多字节字符中间:退回到最近的合法边界
    let text = match String::from_utf8(buf) {
        Ok(t) => t,
        Err(e) => {
            let ok = e.utf8_error().valid_up_to();
            if e.as_bytes().len() - ok > 3 {
                return None;
            }
            String::from_utf8_lossy(&e.as_bytes()[..ok]).into_owned()
        }
    };
    Some(text.lines().take(MAX_LINES).map(|l| l.chars().take(MAX_COLS).collect::<String>().replace('\t', "    ")).collect())
}

/// 回车会做什么(预览栏里的一句话)。
fn enter_hint(item: &PaletteItem) -> &'static str {
    match &item.action {
        PaletteAction::OpenFile(_) => "回车在编辑器中打开",
        PaletteAction::OpenSession(..) => "回车切换到这个会话",
        PaletteAction::OpenProject(_) => "回车展开项目并打开最近会话",
        PaletteAction::RunScenario(_) => "回车从头播放这个演示场景",
        PaletteAction::Goto(..) => "回车跳到这个位置",
        PaletteAction::SetTheme(_) | PaletteAction::PickTheme => "回车选择主题",
        PaletteAction::OpenSettings(_) | PaletteAction::OpenCustomize(_) | PaletteAction::OpenPage(_) => "回车打开这个页面",
        _ => "回车执行这条命令",
    }
}

/// 渲染预览栏。`head` 是文件开头(只对文件条目有意义)。
pub fn render(item: Option<&PaletteItem>, head: Option<&[String]>, cx: &App) -> AnyElement {
    let t = cx.theme();
    let Some(item) = item else {
        return v_flex().size_full().items_center().justify_center().text_size(px(12.)).text_color(t.muted_foreground).child("选中一项查看详情").into_any_element();
    };
    let is_file = matches!(item.group, Group::File | Group::Recent);
    v_flex().size_full().p_3().gap_2()
        .child(h_flex().gap_2().items_center()
            .child(icon(item.icon, 16., t.primary))
            .child(div().flex_1().min_w_0().truncate().text_size(px(13.5)).font_weight(FontWeight::MEDIUM).child(item.title.clone())))
        .when(!item.subtitle.is_empty(), |d| d.child(div().text_size(px(11.5)).text_color(t.muted_foreground).font_family(if is_file { MONO } else { ".SystemUIFont" }).child(item.subtitle.clone())))
        .when(is_file, |d| match head {
            Some(lines) if !lines.is_empty() => d.child(
                v_flex().w_full().flex_1().min_h_0().overflow_hidden().p_2().rounded(px(8.)).bg(t.muted).font_family(MONO).text_size(px(11.)).text_color(t.foreground)
                    .children(lines.iter().map(|l| div().w_full().h(px(15.)).truncate().child(if l.is_empty() { " ".to_string() } else { l.clone() }))),
            ),
            Some(_) => d.child(div().text_size(px(12.)).text_color(t.muted_foreground).child("空文件")),
            None => d.child(div().text_size(px(12.)).text_color(t.muted_foreground).child("无法预览(二进制、过大或不可读)")),
        })
        .child(div().flex_none().pt_1().text_size(px(11.5)).text_color(t.muted_foreground).child(enter_hint(item)))
        .into_any_element()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[::core::prelude::v1::test]
    fn 文件开头只取前若干行并截断长行() {
        let dir = std::env::temp_dir().join(format!("mini-pal-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let p = dir.join("a.txt");
        let body: String = (0..40).map(|i| format!("第{i}行{}\n", "x".repeat(200))).collect();
        std::fs::write(&p, body).unwrap();
        let head = file_head(p.to_str().unwrap()).unwrap();
        assert_eq!(head.len(), MAX_LINES);
        assert!(head.iter().all(|l| l.chars().count() <= MAX_COLS));
        std::fs::remove_dir_all(&dir).ok();
    }

    #[::core::prelude::v1::test]
    fn 相对路径接在项目根后面() {
        assert_eq!(resolve("/a/b/", "src/x.rs"), "/a/b/src/x.rs");
        assert_eq!(resolve("/a/b", "/c/d.rs"), "/c/d.rs");
        assert_eq!(resolve("", "x.rs"), "x.rs");
    }

    #[::core::prelude::v1::test]
    fn 二进制与不存在的文件不预览() {
        let dir = std::env::temp_dir().join(format!("mini-pal-bin-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let p = dir.join("b.bin");
        std::fs::write(&p, [0u8, 1, 2, 3]).unwrap();
        assert!(file_head(p.to_str().unwrap()).is_none());
        assert!(file_head("/不存在/文件").is_none());
        std::fs::remove_dir_all(&dir).ok();
    }

    #[::core::prelude::v1::test]
    fn 多字节字符被截断时仍能预览() {
        let dir = std::env::temp_dir().join(format!("mini-pal-cn-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let p = dir.join("c.txt");
        std::fs::write(&p, "汉".repeat(MAX_BYTES)).unwrap();
        assert!(file_head(p.to_str().unwrap()).is_some());
        std::fs::remove_dir_all(&dir).ok();
    }
}
