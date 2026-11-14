//! 内置浏览器页(阅读模式):抓取网页后提取标题、正文与链接,用原生控件渲染。
//! GPUI 没有 WebView,所以不执行脚本也不排版复杂页面;需要完整网页时交给系统浏览器。

use crate::ui::{MONO, ask_claude, caption, empty_state, icon, pill};
use gpui_kit::assets::IconName;
use gpui_kit::component::button::{Button, ButtonVariants};
use gpui_kit::component::input::{Input, InputEvent, InputState};
use gpui_kit::component::{ActiveTheme, Disableable, Selectable, Sizable, h_flex, v_flex};
use gpui_kit::prelude::FluentBuilder as _;
use gpui_kit::*;
use mini_browser::{History, Page, PageBlock};

pub struct BrowserPage {
    input: Entity<InputState>,
    history: History,
    page: Option<Page>,
    loading: bool,
    error: Option<String>,
    /// 右侧「与此页面对话」栏是否展开
    aside: bool,
    /// 右栏分段:0 摘要 / 1 大纲 / 2 链接
    aside_tab: usize,
    scroll: ScrollHandle,
    _sub: Subscription,
}

impl BrowserPage {
    pub fn new(window: &mut Window, cx: &mut Context<Self>) -> Self {
        let input = cx.new(|cx| InputState::new(window, cx).placeholder("输入网址或搜索词,回车访问"));
        let sub = cx.subscribe_in(&input, window, |this: &mut Self, input, ev: &InputEvent, window, cx| {
            if let InputEvent::PressEnter { .. } = ev {
                let text = input.read(cx).value().to_string();
                if !text.trim().is_empty() {
                    this.open(&text, window, cx);
                }
            }
        });
        Self { input, history: History::default(), page: None, loading: false, error: None, aside: true, aside_tab: 0, scroll: ScrollHandle::new(), _sub: sub }
    }

    /// 访问地址栏输入(网址或搜索词)。
    pub fn open(&mut self, input: &str, window: &mut Window, cx: &mut Context<Self>) {
        let url = mini_browser::normalize_input(input);
        self.history.visit(&url);
        self.load(url, window, cx);
    }

    fn load(&mut self, url: String, window: &mut Window, cx: &mut Context<Self>) {
        self.loading = true;
        self.error = None;
        self.input.update(cx, |i, cx| i.set_value(url.clone(), window, cx));
        cx.notify();
        let bp = crate::prefs(cx).browser;
        let opts = mini_browser::FetchOptions { timeout_secs: bp.timeout_secs as u64, user_agent: bp.user_agent };
        cx.spawn_in(window, async move |this, cx| {
            let target = url.clone();
            let result = cx.background_spawn(async move { mini_browser::load_with(&target, &opts) }).await;
            let _ = this.update_in(cx, |this, window, cx| {
                // 期间用户又导航到别处时,丢弃过期结果
                if this.history.current() != Some(url.as_str()) {
                    return;
                }
                this.loading = false;
                match result {
                    Ok(page) => {
                        // 重定向后以最终地址为准
                        if page.url != url {
                            this.input.update(cx, |i, cx| i.set_value(page.url.clone(), window, cx));
                        }
                        this.page = Some(page);
                    }
                    Err(e) => this.error = Some(e.to_string()),
                }
                cx.notify();
            });
        })
        .detach();
    }

    fn back(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        if let Some(url) = self.history.back().map(String::from) {
            self.load(url, window, cx);
        }
    }

    fn forward(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        if let Some(url) = self.history.forward().map(String::from) {
            self.load(url, window, cx);
        }
    }

    fn reload(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        if let Some(url) = self.history.current().map(String::from) {
            self.load(url, window, cx);
        }
    }

    fn toolbar(&self, cx: &mut Context<Self>) -> impl IntoElement {
        let t = cx.theme().clone();
        let (can_back, can_fwd, has_page) = (self.history.can_back(), self.history.can_forward(), self.history.current().is_some());
        let current = self.history.current().map(String::from);
        h_flex().h(px(44.)).px_3().gap_1().items_center().border_b_1().border_color(t.border)
            .child(Button::new("br-back").small().ghost().icon(IconName::ArrowLeft).disabled(!can_back).on_click(cx.listener(|this, _, window, cx| this.back(window, cx))))
            .child(Button::new("br-fwd").small().ghost().icon(IconName::ArrowRight).disabled(!can_fwd).on_click(cx.listener(|this, _, window, cx| this.forward(window, cx))))
            .child(Button::new("br-reload").small().ghost().icon(IconName::RefreshCw).disabled(!has_page).on_click(cx.listener(|this, _, window, cx| this.reload(window, cx))))
            .child(div().flex_1().px_1().child(Input::new(&self.input)))
            .child(Button::new("br-aside").small().ghost().icon(IconName::PanelRight).selected(self.aside).tooltip("与此页面对话").on_click(cx.listener(|this, _, _, cx| {
                this.aside = !this.aside;
                cx.notify();
            })))
            .child(Button::new("br-ext").small().ghost().icon(IconName::ExternalLink).label("系统浏览器").disabled(!has_page).on_click(move |_, _, cx| {
                if let Some(url) = &current {
                    cx.open_url(url);
                }
            }))
    }

    fn page_view(&self, page: &Page, cx: &mut Context<Self>) -> AnyElement {
        let t = cx.theme().clone();
        let blocks = page.blocks.iter().skip(Self::skip_first(page)).map(|b| match b {
            PageBlock::Heading { level, text } => {
                let size = match level {
                    1 => 24.,
                    2 => 19.,
                    _ => 15.5,
                };
                div().w_full().pt_2().text_size(px(size)).font_weight(FontWeight::SEMIBOLD).child(text.clone()).into_any_element()
            }
            PageBlock::Paragraph(text) => div().w_full().text_size(px(14.)).line_height(px(22.)).child(text.clone()).into_any_element(),
            PageBlock::ListItem(text) => h_flex().w_full().gap_2().items_start().text_size(px(14.)).child(div().text_color(t.muted_foreground).child("•")).child(div().flex_1().child(text.clone())).into_any_element(),
            PageBlock::Quote(text) => div().w_full().pl_3().border_l_2().border_color(t.border).text_color(t.muted_foreground).text_size(px(14.)).child(text.clone()).into_any_element(),
            PageBlock::Code(text) => div().w_full().p_2().rounded(px(6.)).bg(t.muted).font_family(MONO).text_size(px(12.)).child(text.clone()).into_any_element(),
        });
        let links = page.links.iter().enumerate().map(|(i, l)| {
            let url = l.url.clone();
            h_flex().id(SharedString::from(format!("br-link-{i}"))).w_full().gap_2().px_2().py_1().rounded(px(6.)).cursor_pointer().hover(|d| d.bg(t.list_hover))
                .on_click(cx.listener(move |this, _, window, cx| {
                    this.history.visit(&url);
                    this.load(url.clone(), window, cx);
                }))
                .child(icon(IconName::Link, 12., t.muted_foreground))
                .child(div().min_w_0().text_size(px(13.)).text_color(t.primary).truncate().child(l.text.clone()))
                .child(div().flex_1().min_w_0().truncate().text_size(px(11.)).text_color(t.muted_foreground).child(l.url.clone()))
        });
        let aside = self.aside;
        let mut shells: Vec<AnyElement> = Vec::new();
        let shell = |el: AnyElement| div().w_full().max_w(px(820.)).px_6().py_1p5().child(el).into_any_element();
        shells.push(shell(div().text_size(px(11.5)).text_color(t.muted_foreground).child("阅读模式 · 已移除脚本与样式").into_any_element()));
        shells.push(shell(div().text_size(px(26.)).font_weight(FontWeight::SEMIBOLD).child(page.title.clone()).into_any_element()));
        shells.extend(blocks.map(shell));
        if !aside && !page.links.is_empty() {
            shells.push(shell(div().w_full().h(px(1.)).my_2().bg(t.border).into_any_element()));
            shells.push(shell(caption(format!("页面链接 · {}", page.links.len()), cx).into_any_element()));
            shells.push(shell(v_flex().w_full().children(links).into_any_element()));
        }
        v_flex().id("browser-scroll").size_full().overflow_y_scroll().track_scroll(&self.scroll).items_center().py_4().children(shells).into_any_element()
    }

    /// 正文第一块与页面标题相同(常见的 `<h1>` 重复)时跳过它,返回要跳过的块数。
    fn skip_first(page: &Page) -> usize {
        match page.blocks.first() {
            Some(PageBlock::Heading { text, .. }) if text.trim() == page.title.trim() => 1,
            _ => 0,
        }
    }

    /// 页面大纲:标题块的 (块序号 + 2 [前面有两个固定块], 级别, 文字)。
    fn outline(page: &Page) -> Vec<(usize, u8, String)> {
        let skip = Self::skip_first(page);
        page.blocks.iter().enumerate().skip(skip).filter_map(|(i, b)| match b {
            PageBlock::Heading { level, text } => Some((i - skip + 2, level.saturating_sub(1).min(2), text.clone())),
            _ => None,
        }).collect()
    }

    /// 右侧栏(对标设计稿「页面 + 对话分栏」):摘要 / 大纲 / 链接三个分段,底部是「让 Claude 处理此页面」的快捷提示。
    fn aside_view(&self, page: &Page, cx: &mut Context<Self>) -> AnyElement {
        let t = cx.theme().clone();
        let chars: usize = page.blocks.iter().map(|b| match b {
            PageBlock::Heading { text, .. } | PageBlock::Paragraph(text) | PageBlock::ListItem(text) | PageBlock::Quote(text) | PageBlock::Code(text) => text.chars().count(),
        }).sum();
        let outline = Self::outline(page);
        let tabs = [("摘要", 0usize), ("大纲", 1), ("链接", 2)];
        let seg = h_flex().px_3().py_2().gap_1().flex_none().children(tabs.iter().map(|(label, ix)| {
            let ix = *ix;
            Button::new(SharedString::from(format!("br-seg-{ix}"))).xsmall().ghost().label(*label).selected(self.aside_tab == ix)
                .on_click(cx.listener(move |this, _, _, cx| { this.aside_tab = ix; cx.notify(); }))
        }));
        let body: AnyElement = match self.aside_tab {
            0 => v_flex().w_full().px_3().gap_2()
                .child(div().text_size(px(13.)).font_weight(FontWeight::MEDIUM).child(page.title.clone()))
                .child(div().text_size(px(11.)).text_color(t.muted_foreground).truncate().child(page.url.clone()))
                .child(h_flex().gap_1().flex_wrap()
                    .child(pill(format!("约 {chars} 字"), t.muted_foreground, t.muted))
                    .child(pill(format!("{} 个段落", page.blocks.len()), t.muted_foreground, t.muted))
                    .child(pill(format!("{} 条链接", page.links.len()), t.muted_foreground, t.muted)))
                .child(div().text_size(px(12.)).text_color(t.muted_foreground).child("阅读模式只在本地提取正文。想要总结或解答,点下面的快捷提示,Claude 会读取这个页面。"))
                .into_any_element(),
            1 => v_flex().w_full().px_2().gap_0p5()
                .when(outline.is_empty(), |d| d.child(div().px_1().text_size(px(12.)).text_color(t.muted_foreground).child("页面里没有标题")))
                .children(outline.into_iter().map(|(ix, lv, text)| {
                    let handle = self.scroll.clone();
                    div().id(SharedString::from(format!("br-ol-{ix}"))).h(px(24.)).pr_1().pl(px(4. + lv as f32 * 12.)).flex().items_center().rounded(px(5.)).cursor_pointer()
                        .hover(|d| d.bg(t.list_hover)).on_click(move |_, _, _| handle.scroll_to_item(ix))
                        .child(div().min_w_0().truncate().text_size(px(12.5)).when(lv == 0, |d| d.font_weight(FontWeight::MEDIUM)).child(text))
                })).into_any_element(),
            _ => v_flex().w_full().px_2().gap_0p5().children(page.links.iter().take(40).enumerate().map(|(i, l)| {
                let url = l.url.clone();
                v_flex().id(SharedString::from(format!("br-al-{i}"))).w_full().px_1().py_0p5().rounded(px(5.)).cursor_pointer().hover(|d| d.bg(t.list_hover))
                    .on_click(cx.listener(move |this, _, window, cx| {
                        this.history.visit(&url);
                        this.load(url.clone(), window, cx);
                    }))
                    .child(div().truncate().text_size(px(12.5)).text_color(t.primary).child(l.text.clone()))
                    .child(div().truncate().text_size(px(10.5)).text_color(t.muted_foreground).child(l.url.clone()))
            })).into_any_element(),
        };
        let asks = ["总结这个页面", "提取关键要点", "翻译成中文", "围绕这个页面回答我的问题"];
        let target = format!("{}({})", page.title, page.url);
        v_flex().w(px(300.)).h_full().flex_none().border_l_1().border_color(t.border).bg(t.sidebar)
            .child(h_flex().h(px(36.)).px_3().gap_1p5().items_center().flex_none().border_b_1().border_color(t.border)
                .child(icon(IconName::Bot, 14., t.primary))
                .child(div().text_size(px(12.5)).font_weight(FontWeight::MEDIUM).child("与此页面对话")))
            .child(seg)
            .child(div().id("br-aside-body").flex_1().min_h_0().overflow_y_scroll().child(body))
            .child(v_flex().flex_none().px_2().py_2().gap_0p5().border_t_1().border_color(t.border)
                .child(div().px_1().pb_1().text_size(px(11.)).text_color(t.muted_foreground).child("让 Claude 处理此页面"))
                .children(asks.iter().enumerate().map(|(i, a)| {
                    let prompt = if *a == "围绕这个页面回答我的问题" { format!("请先阅读这个页面:{target}。之后我会就它向你提问。") } else { format!("请阅读页面 {target},然后{a}。") };
                    h_flex().id(SharedString::from(format!("br-ask-{i}"))).h(px(26.)).px_1().gap_1p5().items_center().rounded(px(5.)).cursor_pointer()
                        .hover(|d| d.bg(t.list_hover))
                        .on_click(move |_, window, cx| ask_claude(&prompt, window, cx))
                        .child(icon(IconName::ArrowRight, 11., t.primary))
                        .child(div().text_size(px(12.5)).child(a.to_string()))
                })))
            .into_any_element()
    }
}

impl Render for BrowserPage {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let t = cx.theme().clone();
        let body: AnyElement = if self.loading {
            empty_state(IconName::Globe, "正在加载…", self.history.current().unwrap_or_default(), cx).into_any_element()
        } else if let Some(e) = &self.error {
            empty_state(IconName::TriangleAlert, "无法打开页面", e, cx).into_any_element()
        } else if let Some(page) = self.page.clone() {
            self.page_view(&page, cx)
        } else {
            empty_state(IconName::Globe, "内置浏览器", "输入网址或搜索词开始浏览(阅读模式,不执行网页脚本)", cx).into_any_element()
        };
        let aside = match (&self.page, self.aside && !self.loading && self.error.is_none()) {
            (Some(page), true) => Some(self.aside_view(&page.clone(), cx)),
            _ => None,
        };
        v_flex().size_full().bg(t.background).child(self.toolbar(cx)).child(h_flex().flex_1().min_h_0().child(div().flex_1().min_w_0().h_full().child(body)).children(aside))
    }
}
