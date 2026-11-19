package lsp

// Mirror 是托管 Node 与语言包的下载源(偏好 lsp.mirror,默认官方源)。
// 不自动测速、不自动切换:该用哪个由用户选,下载失败时界面只提示可以换源。
type Mirror string

const (
	MirrorOfficial Mirror = "official" // nodejs.org / registry.npmjs.org
	MirrorCN       Mirror = "cn"       // npmmirror.com(Node 与 npm 包都走它)
)

// ParseMirror 把偏好值转成 Mirror;空 / 未知一律当官方源。
func ParseMirror(s string) Mirror {
	if s == string(MirrorCN) {
		return MirrorCN
	}
	return MirrorOfficial
}

// NodeDist 是 Node 发行包的根地址(其下是 v24.x.y/ 目录)。
func (m Mirror) NodeDist() string {
	if m == MirrorCN {
		return "https://npmmirror.com/mirrors/node"
	}
	return "https://nodejs.org/dist"
}

// Registry 是 npm 源。
func (m Mirror) Registry() string {
	if m == MirrorCN {
		return "https://registry.npmmirror.com"
	}
	return "https://registry.npmjs.org"
}

// Label 是界面上的名字。
func (m Mirror) Label() string {
	if m == MirrorCN {
		return "国内镜像"
	}
	return "官方源"
}
