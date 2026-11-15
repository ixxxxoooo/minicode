package lsp

import (
	"net/url"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// ColToUTF16 把一行里的字节列换成 UTF-16 列(LSP 的 character)。
func ColToUTF16(line string, byteCol int) int {
	if byteCol > len(line) {
		byteCol = len(line)
	}
	n := 0
	for _, r := range line[:byteCol] {
		n += utf16Len(r)
	}
	return n
}

// ColFromUTF16 把 UTF-16 列换回一行里的字节列;超出行尾夹在行尾。
func ColFromUTF16(line string, u16 int) int {
	n := 0
	for i, r := range line {
		if n >= u16 {
			return i
		}
		n += utf16Len(r)
	}
	return len(line)
}

func utf16Len(r rune) int {
	if r == utf8.RuneError || r < 0x10000 {
		return 1
	}
	return len(utf16.Encode([]rune{r}))
}

// PathToURI 把绝对路径变成 file:// URI。
func PathToURI(p string) string {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(p)}
	return u.String()
}

// URIToPath 把 file:// URI 变回路径;不是 file 协议返回空串。
func URIToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return ""
	}
	return filepath.FromSlash(u.Path)
}

// HasPrefixPath 判断 p 是否在目录 dir 之内。
func HasPrefixPath(p, dir string) bool {
	dir = strings.TrimSuffix(dir, "/")
	return p == dir || strings.HasPrefix(p, dir+"/")
}
