//! 权限模式的图标与颜色:菜单里每一项、输入框外的模式标签用同一套,一眼分清当前是哪种模式。
//!
//! @author ixxxxoooo

use gpui_kit::assets::IconName;
use gpui_kit::component::ActiveTheme;
use gpui_kit::*;
use mini_core::PermissionMode;

/// 模式对应的图标。
pub fn mode_icon(mode: PermissionMode) -> IconName {
    match mode {
        PermissionMode::Default => IconName::Shield,
        PermissionMode::AcceptEdits => IconName::FilePen,
        PermissionMode::Plan => IconName::ClipboardList,
        PermissionMode::Auto => IconName::Sparkles,
        PermissionMode::DontAsk => IconName::ShieldBan,
        PermissionMode::BypassPermissions => IconName::ShieldAlert,
    }
}

/// 模式对应的颜色:默认灰、接受编辑绿、计划蓝、自动主题色、不再询问黄、绕过权限红。
pub fn mode_color(mode: PermissionMode, cx: &App) -> Hsla {
    let t = cx.theme();
    match mode {
        PermissionMode::Default => t.muted_foreground,
        PermissionMode::AcceptEdits => t.success,
        PermissionMode::Plan => t.info,
        PermissionMode::Auto => t.primary,
        PermissionMode::DontAsk => t.warning,
        PermissionMode::BypassPermissions => t.danger,
    }
}

/// 模式的一句话说明(运行配置弹层里每行右侧的灰字)。
pub fn mode_hint(mode: PermissionMode) -> &'static str {
    match mode {
        PermissionMode::Default => "每次修改与命令都问我",
        PermissionMode::AcceptEdits => "自动改文件,命令仍需确认",
        PermissionMode::Plan => "只读规划,批准后再动手",
        PermissionMode::Auto => "模型自行判断风险",
        PermissionMode::DontAsk => "不询问,未授权的直接拒绝",
        PermissionMode::BypassPermissions => "全部放行(谨慎)",
    }
}
