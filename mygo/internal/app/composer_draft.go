package app

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

// 输入框正文与图片附件分离:`a.draft` 仍是「正文 + 末尾的 ![](路径)」的完整草稿(发送、排队、会话恢复都读它),
// 但输入框只显示正文(编辑器文档 a.compDoc,见 composer_rich.go),图片附件以缩略图卡显示在输入框上方,
// 不再把 `![](/Users/…/xxx.png)` 这串路径露在输入框里(ChatGPT / Claude.ai 都是这么做的)。
// 正文里的胶囊(文件 / 文件夹 / 折叠文本)在草稿字符串里是原子标记串(composer_atoms.go)。
//
// @author ixxxxoooo

// draftTokenRE 匹配草稿里的图片附件(含前面的一个换行,拆出后正文不留空行)。
var draftTokenRE = regexp.MustCompile(`\n?!\[[^\]]*\]\([^)]+\)`)

// 图片路径写进 `![](路径)` 时转义 % ( ),否则像「舆图志-陕西省 (3).png」这样带括号的文件名会把语法截断
// (缩略图读不到、半截路径漏进输入框)。
var (
	imgPathEsc   = strings.NewReplacer("%", "%25", "(", "%28", ")", "%29")
	imgPathUnesc = strings.NewReplacer("%25", "%", "%28", "(", "%29", ")")
)

// escImgPath 把路径转义成可安全放进 `![](…)` 的形式。
func escImgPath(p string) string { return imgPathEsc.Replace(p) }

// unescImgPath 是 escImgPath 的逆运算。
func unescImgPath(p string) string { return imgPathUnesc.Replace(p) }

// splitDraftBody 把完整草稿拆成「正文」与「图片附件语法串」(按出现顺序)。
// 参数:draft 完整草稿。返回:去掉附件后的正文、各个 `![](路径)` 串。
func splitDraftBody(draft string) (body string, tokens []string) {
	body = draftTokenRE.ReplaceAllStringFunc(draft, func(m string) string {
		tokens = append(tokens, strings.TrimPrefix(m, "\n"))
		return ""
	})
	return body, tokens
}

// composeDraft 把正文与图片附件合成完整草稿(附件各占一行放在末尾),是 splitDraftBody 的逆运算。
// 参数:body 输入框正文;tokens 图片附件语法串。返回:完整草稿。
func composeDraft(body string, tokens []string) string {
	out := body
	for _, tk := range tokens {
		if out != "" {
			out += "\n"
		}
		out += tk
	}
	return out
}

// syncDraft 每帧在输入框构建前调用,让 a.draft 与输入框正文保持一致:
//   - 别处改了 a.draft(拖入、清空、切会话、插入文字……)→ 重新拆分,刷新输入框正文与附件;
//   - 否则是用户在输入框里打字改了 a.draftBody → 合成回 a.draft。
func (a *App) syncDraft(c *ui.Context) {
	if a.compDoc == nil {
		a.compDoc = richtext.NewDocument(richtext.NewText(""))
		a.compVer = a.compDoc.Version()
	}
	if a.draft != a.draftSeen {
		a.draftBody, a.draftImgs = splitDraftBody(a.draft)
		a.draftBody = dropOrphanImgAtoms(a.draftBody, a.draftImgs) // 缩略图被移除了,正文里它的位置胶囊也去掉
		a.imgPilled = imgAtomPaths(a.draftBody)
		a.compDoc.SetText(draftToText(a.draftBody, func(kind, path string) *richtext.Attachment {
			return a.pillAttachment(c, kind, path)
		}))
		a.compVer = a.compDoc.Version()
		a.draftSeen = a.draft
		return
	}
	if v := a.compDoc.Version(); v != a.compVer { // 用户在编辑器里改了:导出正文
		a.compVer = v
		a.draftBody = textToDraft(a.compDoc.Text())
		a.dropImgsWithoutPill()
	}
	if cur := composeDraft(a.draftBody, a.draftImgs); cur != a.draft {
		a.draft, a.draftSeen = cur, cur
	}
}

// draftParts 返回此刻的正文与图片附件(含编辑器里还没同步回 a.draft 的最新输入)。
// 别处刚改过 a.draft、还没同步时以 a.draft 为准。
func (a *App) draftParts() (body string, imgs []string) {
	if a.compDoc == nil || a.draft != a.draftSeen {
		return splitDraftBody(a.draft)
	}
	return textToDraft(a.compDoc.Text()), a.draftImgs
}

// draftNow 返回此刻的完整草稿(胶囊仍是原子标记串),发送 / 排队用。
func (a *App) draftNow() string { return composeDraft(a.draftParts()) }

// addDraftImage 在草稿里加一张图片附件(路径可为绝对路径);同一张不重复。
// 在输入回调(粘贴、拖放)里调用也安全:以编辑器最新内容为底,不会丢掉刚打的字。
// 返回:是否新增。
func (a *App) addDraftImage(path string) bool {
	tk := "![](" + escImgPath(path) + ")"
	body, imgs := a.draftParts()
	for _, x := range imgs {
		if x == tk {
			return false
		}
	}
	imgs = append(append([]string(nil), imgs...), tk)
	a.draft = composeDraft(body, imgs)
	if a.compDoc != nil {
		// 不让输入框按新草稿重建(重建会丢光标):正文与编辑器保持原样,只补上附件;位置胶囊稍后插在光标处
		a.draftBody, a.draftImgs, a.draftSeen = body, imgs, a.draft
	}
	a.queuePill(atomImg, path)
	a.ux.focusDraft = true
	return true
}

// imgAtomPaths 取正文里所有图片位置胶囊的路径。
func imgAtomPaths(body string) map[string]bool {
	out := map[string]bool{}
	for _, m := range atomRE.FindAllStringSubmatch(body, -1) {
		if kind, path, ok := atomParts(m[1]); ok && kind == atomImg {
			out[path] = true
		}
	}
	return out
}

// dropOrphanImgAtoms 去掉正文里指向「已不在附件里的图片」的位置胶囊。
func dropOrphanImgAtoms(body string, imgs []string) string {
	have := map[string]bool{}
	for _, tk := range imgs {
		for _, at := range parseDraftAttaches(tk) {
			have[at.Path] = true
		}
	}
	return atomRE.ReplaceAllStringFunc(body, func(m string) string {
		sm := atomRE.FindStringSubmatch(m)
		if kind, path, ok := atomParts(sm[1]); ok && kind == atomImg && !have[path] {
			return ""
		}
		return m
	})
}

// dropImgsWithoutPill 用户在输入框里删掉了某张图的位置胶囊:同时把这张图的附件移除(只处理「原来有、现在没了」的)。
func (a *App) dropImgsWithoutPill() {
	now := imgAtomPaths(a.draftBody)
	var gone []string
	for p := range a.imgPilled {
		if !now[p] {
			gone = append(gone, p)
		}
	}
	a.imgPilled = now
	if len(gone) == 0 {
		return
	}
	var keep []string
	for _, tk := range a.draftImgs {
		drop := false
		for _, at := range parseDraftAttaches(tk) {
			for _, g := range gone {
				if at.Path == g {
					drop = true
				}
			}
		}
		if !drop {
			keep = append(keep, tk)
		}
	}
	a.draftImgs = keep
}
