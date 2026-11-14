//! 「自定义」页:一个搜索框加七个标签(插件 / MCP / 技能 / 命令 / 子 Agent / 规则 / Hooks),
//! 内容复用对应的管理页。左侧栏的「自定义」入口(整页)与设置里的「自定义」分类共用这一份渲染。
//!
//! @author ixxxxoooo

use super::widgets::chips;
use super::{claude_pages, hooks_page, items_page, mcp_page, plugins_page};
use crate::ui::icon;
use crate::workbench::Workbench;
use gpui_kit::assets::IconName;
use gpui_kit::component::button::{Button, ButtonVariants};
use gpui_kit::component::input::Input;
use gpui_kit::component::{ActiveTheme, Sizable, h_flex, v_flex};
use gpui_kit::prelude::FluentBuilder as _;
use gpui_kit::*;
use mini_config::items::ItemKind;

/// 自定义页的标签。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CustomizeTab {
    Plugins,
    Mcp,
    Skills,
    Commands,
    Agents,
    Rules,
    Hooks,
}

impl CustomizeTab {
    pub const ALL: [CustomizeTab; 7] = [Self::Plugins, Self::Mcp, Self::Skills, Self::Commands, Self::Agents, Self::Rules, Self::Hooks];

    /// 标签名。
    pub fn label(self) -> &'static str {
        match self {
            Self::Plugins => "插件",
            Self::Mcp => "MCP",
            Self::Skills => "技能",
            Self::Commands => "命令",
            Self::Agents => "子 Agent",
            Self::Rules => "规则",
            Self::Hooks => "Hooks",
        }
    }

    /// 稳定的小写名(`MINI_CUSTOMIZE`、记住上次标签用)。
    pub fn key(self) -> &'static str {
        match self {
            Self::Plugins => "plugins",
            Self::Mcp => "mcp",
            Self::Skills => "skills",
            Self::Commands => "commands",
            Self::Agents => "agents",
            Self::Rules => "rules",
            Self::Hooks => "hooks",
        }
    }

    /// 按小写名查找。
    pub fn from_name(name: &str) -> Option<Self> {
        Self::ALL.into_iter().find(|t| t.key() == name)
    }
}

/// 当前标签的内容(插件 / MCP / 技能…的管理页),`render` 与整页布局共用。
fn body(wb: &mut Workbench, window: &mut Window, cx: &mut Context<Workbench>) -> AnyElement {
    wb.ensure_claude_cfg();
    wb.ensure_forms(window, cx);
    let tab = wb.customize_tab;
    let filter = wb.field_value("search", cx).to_lowercase();
    match tab {
        CustomizeTab::Plugins => plugins_page::render(wb, &filter, cx),
        CustomizeTab::Mcp => mcp_page::mcp(wb, &filter, cx),
        CustomizeTab::Skills => items_page::render(wb, ItemKind::Skill, tab, &filter, cx),
        CustomizeTab::Commands => items_page::render(wb, ItemKind::Command, tab, &filter, cx),
        CustomizeTab::Agents => items_page::render(wb, ItemKind::Agent, tab, &filter, cx),
        CustomizeTab::Rules => claude_pages::memory(wb, window, cx),
        CustomizeTab::Hooks => hooks_page::hooks(wb, &filter, cx),
    }
}

/// 渲染自定义页内容(不含外层滚动容器):搜索框 + 标签条 + 内容。设置里的「自定义」分类用它。
///
/// 参数:`wb` 工作台;`window` / `cx` gpui 上下文。返回:页面元素。
pub fn render(wb: &mut Workbench, window: &mut Window, cx: &mut Context<Workbench>) -> AnyElement {
    let tab = wb.customize_tab;
    let body = body(wb, window, cx);
    let t = cx.theme().clone();
    v_flex().w_full().gap_4()
        .child(v_flex().w_full().gap_3()
            .child(h_flex().w_full().items_center().gap_2()
                .child(div().flex_1().child(Input::new(wb.field("search")).small().prefix(icon(IconName::Search, 14., t.muted_foreground))))
                .child(Button::new("customize-refresh").small().ghost().icon(IconName::RefreshCw).tooltip("重新读取")
                    .on_click(cx.listener(|this, _, _, cx| { this.reload_claude_cfg(); this.refresh_plugins(); cx.notify(); }))))
            .child(chips("cz", CustomizeTab::ALL.into_iter().map(|c| (c, c.label().to_string())).collect(), tab, cx, |this, c, cx| {
                this.customize_tab = c;
                this.update_settings(cx, |s| s.last_customize_tab = c.key().to_string());
            })))
        .child(div().w_full().h(px(1.)).bg(t.border))
        .child(body)
        .into_any_element()
}

impl CustomizeTab {
    /// 左侧分类栏里的图标。
    fn icon(self) -> IconName {
        match self {
            Self::Plugins => IconName::Plug,
            Self::Mcp => IconName::Server,
            Self::Skills => IconName::Zap,
            Self::Commands => IconName::Hash,
            Self::Agents => IconName::Bot,
            Self::Rules => IconName::FileText,
            Self::Hooks => IconName::Webhook,
        }
    }
}

/// 某个分类下的条目数(规则 / Hooks 没有统一的计数,返回 `None`)。
fn tab_count(wb: &Workbench, tab: CustomizeTab) -> Option<usize> {
    let items = |k: ItemKind| wb.forms().items.get(&k).map(|v| v.len());
    match tab {
        CustomizeTab::Plugins => Some(wb.forms().plugins.installed.len()),
        CustomizeTab::Mcp => Some(wb.claude_cfg.mcp_servers.len()),
        CustomizeTab::Skills => items(ItemKind::Skill),
        CustomizeTab::Commands => items(ItemKind::Command),
        CustomizeTab::Agents => items(ItemKind::Agent),
        CustomizeTab::Rules | CustomizeTab::Hooks => None,
    }
}

/// 左侧分类栏:七个分类(图标 + 名称 + 条目数),当前项高亮;底部「重新读取」。
fn category_rail(wb: &Workbench, cx: &mut Context<Workbench>) -> AnyElement {
    let t = cx.theme().clone();
    let cur = wb.customize_tab;
    v_flex().w(px(176.)).h_full().flex_none().p_2().gap_0p5().border_r_1().border_color(t.border).bg(t.sidebar)
        .child(div().px_2().pt_1().pb_1().text_size(px(11.)).text_color(t.muted_foreground).child("类别"))
        .children(CustomizeTab::ALL.into_iter().map(|c| {
            let on = c == cur;
            h_flex().id(SharedString::from(format!("cz-cat-{}", c.key()))).h(px(30.)).px_2().gap_2().items_center().rounded(px(6.)).cursor_pointer()
                .when(on, |d| d.bg(t.primary.opacity(0.12)))
                .hover(|d| d.bg(if on { t.primary.opacity(0.16) } else { t.list_hover }))
                .on_click(cx.listener(move |this, _, _, cx| {
                    this.customize_tab = c;
                    this.update_settings(cx, |s| s.last_customize_tab = c.key().to_string());
                }))
                .child(icon(c.icon(), 14., if on { t.primary } else { t.muted_foreground }))
                .child(div().flex_1().text_size(px(13.)).when(on, |d| d.text_color(t.primary).font_weight(FontWeight::MEDIUM)).child(c.label()))
                .when_some(tab_count(wb, c), |d, n| d.child(div().text_size(px(11.)).text_color(t.muted_foreground).child(n.to_string())))
        }))
        .child(div().flex_1())
        .child(
            h_flex().id("cz-reload").h(px(28.)).px_2().gap_2().items_center().rounded(px(6.)).cursor_pointer().hover(|d| d.bg(t.list_hover))
                .on_click(cx.listener(|this, _, _, cx| { this.reload_claude_cfg(); this.refresh_plugins(); cx.notify(); }))
                .child(icon(IconName::RefreshCw, 13., t.muted_foreground))
                .child(div().text_size(px(12.)).text_color(t.muted_foreground).child("重新读取")),
        )
        .into_any_element()
}

/// 主窗口里的整页(左分类栏 + 中间搜索与内容):内容区居中可滚动,宽度比设置页稍宽。
pub fn render_page(wb: &mut Workbench, window: &mut Window, cx: &mut Context<Workbench>) -> AnyElement {
    let body = body(wb, window, cx);
    let rail = category_rail(wb, cx);
    let t = cx.theme().clone();
    let search = h_flex().w_full().items_center().gap_2()
        .child(div().flex_1().child(Input::new(wb.field("search")).small().prefix(icon(IconName::Search, 14., t.muted_foreground))));
    // 纵向 flex + flex_none:内容比窗口高时不被压缩,外层才有可滚动的溢出(横向 flex 会把子项拉伸到容器高度,导致滚不动)
    h_flex().size_full()
        .child(rail)
        .child(
            div().id("customize-scroll").flex_1().min_w_0().h_full().overflow_y_scroll().flex().flex_col().items_center().px_6().pb_10()
                .child(v_flex().w_full().max_w(px(820.)).flex_none().pt_4().gap_4().child(search).child(body)),
        )
        .into_any_element()
}
