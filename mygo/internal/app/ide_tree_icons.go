package app

// treeFolderIcon 资源管理器与面包屑下拉共用的文件夹图标名(与线条图标表一致)。
func treeFolderIcon(name string) string {
	switch name {
	case ".git":
		return "branch"
	case "node_modules", "vendor", "target", "dist", "build":
		return "folder"
	case "src", "lib", "test", "tests":
		return "folder"
	default:
		return "folder"
	}
}
