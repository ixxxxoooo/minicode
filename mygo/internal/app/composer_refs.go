package app

import "strings"

// 文件 / 文件夹引用(拖入、添加、文件树「引用到对话」)写进输入框正文,是和文字在一起、可以夹在文字中间的胶囊
// (composer_pill.go),插在光标处;只有图片是输入框上方的缩略图附件。发送时胶囊展开成 `@路径`。
//
// @author ixxxxoooo

// insertRefs 把一批「@路径」引用(文件夹以 / 结尾)排队成胶囊,下一帧插进输入框光标处;已有的不重复。
// 返回:实际新增了几个。
func (a *App) insertRefs(refs ...string) int {
	n := 0
	for _, r := range refs {
		if !strings.HasPrefix(r, "@") || len(r) < 2 {
			continue
		}
		p := strings.TrimPrefix(r, "@")
		kind := atomFile
		if strings.HasSuffix(p, "/") {
			kind, p = atomDir, strings.TrimSuffix(p, "/")
		}
		if a.queuePill(kind, p) {
			n++
		}
	}
	return n
}
