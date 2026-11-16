// Package tabular 读取表格类文件(csv / tsv / xlsx)供只读查看器显示。
// 与界面无关;为了不卡界面,行 / 列数都有上限,超出的部分只记「被截断」。
package tabular

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	// MaxRows / MaxCols 是查看器最多载入的行数 / 列数。
	MaxRows = 5000
	MaxCols = 100
)

// ErrEncoding 表示文件既不是 UTF-8 也读不出 GBK。
var ErrEncoding = errors.New("编码无法识别(只支持 UTF-8 与 GBK)")

// Sheet 是一张工作表。Rows 的每行长度可以不同,取格子时用 Cell。
type Sheet struct {
	Name      string
	Rows      [][]string
	Cols      int  // 最宽一行的列数(已按 MaxCols 截断)
	Truncated bool // 行或列超出上限被截掉了
}

// Cell 返回第 r 行第 c 列的文本,越界返回空串。
func (s *Sheet) Cell(r, c int) string {
	if r < 0 || r >= len(s.Rows) || c < 0 || c >= len(s.Rows[r]) {
		return ""
	}
	return s.Rows[r][c]
}

// Workbook 是一个文件里的全部工作表(csv 只有一张)。
type Workbook struct {
	Sheets []Sheet
	// Note 是需要提示用户的附注(例如「按 GBK 解码」「只读了前 32 MB」)。
	Note string
}

// decodeText 把字节转成 UTF-8 文本:去 BOM;不是合法 UTF-8 时按 GBK 试一次。
func decodeText(data []byte) (string, string, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if utf8.Valid(data) {
		return string(data), "", nil
	}
	// 读取上限处可能把最后一个多字节字符截成半个:去掉尾部至多 3 字节再试
	for cut := 1; cut <= 3 && cut <= len(data); cut++ {
		if utf8.Valid(data[:len(data)-cut]) {
			return string(data[:len(data)-cut]), "", nil
		}
	}
	out, err := simplifiedchinese.GBK.NewDecoder().Bytes(data)
	if err != nil || bytes.ContainsRune(out, utf8.RuneError) {
		return "", "", ErrEncoding
	}
	return string(out), "按 GBK 解码", nil
}

// detectComma 在前几行里数候选分隔符,取出现最多的;ext 为 ".tsv" 时直接用制表符。
func detectComma(text, ext string) rune {
	if strings.EqualFold(ext, ".tsv") {
		return '\t'
	}
	head := text
	if len(head) > 8192 {
		head = head[:8192]
	}
	lines := strings.SplitN(head, "\n", 6)
	best, bestN := ',', 0
	for _, c := range []rune{',', ';', '\t', '|'} {
		n := 0
		for _, ln := range lines {
			n += strings.Count(ln, string(c))
		}
		if n > bestN {
			best, bestN = c, n
		}
	}
	return best
}

// ParseCSV 解析 csv / tsv。ext 带点(".csv" / ".tsv"),partial 表示传入的只是文件开头的一部分。
func ParseCSV(data []byte, ext string, partial bool) (*Workbook, error) {
	text, note, err := decodeText(data)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = detectComma(text, ext)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	sh := Sheet{}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if len(sh.Rows) == 0 {
				return nil, err
			}
			sh.Truncated = true
			break
		}
		if len(sh.Rows) >= MaxRows {
			sh.Truncated = true
			break
		}
		if len(rec) > MaxCols {
			rec = rec[:MaxCols]
			sh.Truncated = true
		}
		if len(rec) > sh.Cols {
			sh.Cols = len(rec)
		}
		sh.Rows = append(sh.Rows, rec)
	}
	if partial {
		sh.Truncated = true
		if note != "" {
			note += ";"
		}
		note += "文件较大,只读了开头一部分"
	}
	return &Workbook{Sheets: []Sheet{sh}, Note: note}, nil
}

// ColName 把 0 起的列号转成表格里的列字母:0→A,25→Z,26→AA。
func ColName(i int) string {
	var b []byte
	for i++; i > 0; i = (i - 1) / 26 {
		b = append([]byte{byte('A' + (i-1)%26)}, b...)
	}
	return string(b)
}

// colIndex 是 ColName 的反向:"A"→0,"AA"→26;遇到非字母停止,没有字母返回 -1。
func colIndex(ref string) int {
	n, seen := 0, false
	for i := 0; i < len(ref); i++ {
		ch := ref[i]
		switch {
		case ch >= 'A' && ch <= 'Z':
			n = n*26 + int(ch-'A') + 1
		case ch >= 'a' && ch <= 'z':
			n = n*26 + int(ch-'a') + 1
		default:
			return n - 1
		}
		seen = true
	}
	if !seen {
		return -1
	}
	return n - 1
}
