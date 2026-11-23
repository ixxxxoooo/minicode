// Package macos 是 macOS 平台层:用 cgo 调 Cocoa 的一小组自包含能力(辅助功能偏好、访达图标、Dock 进度 / 弹跳、
// 系统深浅色)。不依赖界面,也不回调进应用;非 macOS 或关闭 cgo 时各函数退化成「没有 / 不支持」的空实现,
// 调用方不用写平台判断。
//
// 通知(mac_notify_*)带 //export 回调进应用,不是自包含的,仍留在 internal/app。
package macos
