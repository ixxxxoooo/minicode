package update

import (
	"encoding/json"
	"io"
	"net/url"
	"path"
	"strings"
)

// appcastEntry 是发布源里描述一个版本的条目。
type appcastEntry struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	Sha256  string `json:"sha256"`
}

// parseAppcast 解析 appcast.json,同时兼容两种格式:
//   - 平铺:{"version","url","sha256"}(本应用读取的格式)
//   - 按通道嵌套:{"stable":{…},"beta":{…}}(旧版发布脚本写的格式,旧客户端读它)
//
// 发布脚本两种都写,新旧客户端都能读到。
// 参数:r 为 appcast 内容,channel 为通道名(空 = stable;没有该通道时退回 stable)。
// 返回值:版本、下载地址、sha256;没有可用条目时三者皆空、err 为 nil。
func parseAppcast(r io.Reader, channel string) (ver, pkgURL, sha string, err error) {
	var raw map[string]json.RawMessage
	if err = json.NewDecoder(r).Decode(&raw); err != nil {
		return "", "", "", err
	}
	var e appcastEntry
	pick := func(key string) bool {
		b, ok := raw[key]
		if !ok {
			return false
		}
		return json.Unmarshal(b, &e) == nil && e.Version != "" && e.URL != ""
	}
	switch {
	case channel != "" && channel != "stable" && pick(channel):
	case pick("stable"):
	default:
		e = appcastEntry{}
		for k, dst := range map[string]*string{"version": &e.Version, "url": &e.URL, "sha256": &e.Sha256} {
			if b, ok := raw[k]; ok {
				_ = json.Unmarshal(b, dst)
			}
		}
	}
	if e.Version == "" || e.URL == "" {
		return "", "", "", nil
	}
	return e.Version, e.URL, e.Sha256, nil
}

// pkgFileName 按下载地址的扩展名决定本地文件名:解包靠扩展名区分 dmg / zip,
// 不带扩展名会走到「当目录复制」的分支而失败。
// 参数:u 为下载地址。返回值:download.dmg / download.zip,其他情况 download.pkg。
func pkgFileName(u string) string {
	p := u
	if pu, err := url.Parse(u); err == nil {
		p = pu.Path
	}
	switch ext := strings.ToLower(path.Ext(p)); ext {
	case ".dmg", ".zip":
		return "download" + ext
	}
	return "download.pkg"
}
