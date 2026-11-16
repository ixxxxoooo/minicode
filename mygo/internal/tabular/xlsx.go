package tabular

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"path"
	"strconv"
	"strings"
	"time"
)

// 单个 zip 条目解压后的上限,防止「压缩炸弹」把内存吃光。
const maxEntry = 256 << 20

// ParseXLSX 读 .xlsx(Office Open XML)。只用标准库:zip + xml 流式解析,
// 每张表最多 MaxRows × MaxCols;日期按单元格数字格式换算成文本。
func ParseXLSX(data []byte) (*Workbook, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errors.New("不是有效的 xlsx 文件(zip 结构损坏或是旧版 .xls)")
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	open := func(name string) (io.ReadCloser, error) {
		f := files[strings.TrimPrefix(name, "/")]
		if f == nil {
			return nil, errors.New("缺少 " + name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		return limited{io.LimitReader(rc, maxEntry), rc}, nil
	}

	wb, err := readWorkbook(open)
	if err != nil {
		return nil, err
	}
	sst := readSharedStrings(open)
	dateXf := readDateStyles(open)

	out := &Workbook{}
	for _, sh := range wb.sheets {
		sheet, err := readSheet(open, sh.target, sst, dateXf, wb.date1904)
		if err != nil {
			continue // 一张表读不出来不影响其它表
		}
		sheet.Name = sh.name
		out.Sheets = append(out.Sheets, *sheet)
	}
	if len(out.Sheets) == 0 {
		return nil, errors.New("没有读到任何工作表")
	}
	return out, nil
}

type limited struct {
	io.Reader
	c io.Closer
}

func (l limited) Close() error { return l.c.Close() }

type sheetRef struct{ name, target string }

type bookInfo struct {
	sheets   []sheetRef
	date1904 bool
}

func readWorkbook(open func(string) (io.ReadCloser, error)) (*bookInfo, error) {
	rc, err := open("xl/workbook.xml")
	if err != nil {
		return nil, errors.New("不是有效的 xlsx 文件(缺少 workbook.xml)")
	}
	defer rc.Close()
	type sheetEl struct {
		Name  string `xml:"name,attr"`
		State string `xml:"state,attr"`
		RID   string `xml:"id,attr"` // r:id,命名空间不同只取本地名
	}
	var doc struct {
		Pr struct {
			Date1904 string `xml:"date1904,attr"`
		} `xml:"workbookPr"`
		Sheets struct {
			List []sheetEl `xml:"sheet"`
		} `xml:"sheets"`
	}
	if err := xml.NewDecoder(rc).Decode(&doc); err != nil {
		return nil, err
	}
	// 关系表:rId → 目标文件
	rels := map[string]string{}
	if rr, err := open("xl/_rels/workbook.xml.rels"); err == nil {
		var rd struct {
			Rel []struct {
				ID     string `xml:"Id,attr"`
				Target string `xml:"Target,attr"`
			} `xml:"Relationship"`
		}
		if xml.NewDecoder(rr).Decode(&rd) == nil {
			for _, r := range rd.Rel {
				t := r.Target
				if strings.HasPrefix(t, "/") {
					t = strings.TrimPrefix(t, "/")
				} else {
					t = path.Join("xl", t)
				}
				rels[r.ID] = t
			}
		}
		rr.Close()
	}
	info := &bookInfo{date1904: doc.Pr.Date1904 == "1" || doc.Pr.Date1904 == "true"}
	var hidden []sheetRef
	for i, s := range doc.Sheets.List {
		target := rels[s.RID]
		if target == "" {
			target = "xl/worksheets/sheet" + strconv.Itoa(i+1) + ".xml"
		}
		ref := sheetRef{name: s.Name, target: target}
		if s.State == "hidden" || s.State == "veryHidden" {
			hidden = append(hidden, ref)
			continue
		}
		info.sheets = append(info.sheets, ref)
	}
	if len(info.sheets) == 0 { // 全是隐藏表时还是给用户看
		info.sheets = hidden
	}
	return info, nil
}

// readSharedStrings 读共享字符串表;富文本取所有 <t> 拼接,拼音注释 <rPh> 跳过。
func readSharedStrings(open func(string) (io.ReadCloser, error)) []string {
	rc, err := open("xl/sharedStrings.xml")
	if err != nil {
		return nil
	}
	defer rc.Close()
	dec := xml.NewDecoder(rc)
	var out []string
	var cur strings.Builder
	inSI, inT, phon := false, false, 0
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch e := tok.(type) {
		case xml.StartElement:
			switch e.Name.Local {
			case "si":
				inSI = true
				cur.Reset()
			case "rPh":
				phon++
			case "t":
				inT = inSI && phon == 0
			}
		case xml.EndElement:
			switch e.Name.Local {
			case "si":
				out = append(out, cur.String())
				inSI = false
			case "rPh":
				phon--
			case "t":
				inT = false
			}
		case xml.CharData:
			if inT {
				cur.Write(e)
			}
		}
	}
	return out
}

// readDateStyles 返回「样式序号 → 是否日期格式」。
func readDateStyles(open func(string) (io.ReadCloser, error)) []bool {
	rc, err := open("xl/styles.xml")
	if err != nil {
		return nil
	}
	defer rc.Close()
	var doc struct {
		NumFmts struct {
			List []struct {
				ID   int    `xml:"numFmtId,attr"`
				Code string `xml:"formatCode,attr"`
			} `xml:"numFmt"`
		} `xml:"numFmts"`
		CellXfs struct {
			List []struct {
				NumFmt int `xml:"numFmtId,attr"`
			} `xml:"xf"`
		} `xml:"cellXfs"`
	}
	if xml.NewDecoder(rc).Decode(&doc) != nil {
		return nil
	}
	custom := map[int]bool{}
	for _, f := range doc.NumFmts.List {
		custom[f.ID] = isDateCode(f.Code)
	}
	out := make([]bool, len(doc.CellXfs.List))
	for i, x := range doc.CellXfs.List {
		if v, ok := custom[x.NumFmt]; ok {
			out[i] = v
		} else {
			out[i] = isBuiltinDate(x.NumFmt)
		}
	}
	return out
}

func isBuiltinDate(id int) bool {
	switch {
	case id >= 14 && id <= 22, id >= 27 && id <= 36, id >= 45 && id <= 47, id >= 50 && id <= 58:
		return true
	}
	return false
}

// isDateCode 判断自定义数字格式是不是日期 / 时间:去掉引号文本、[颜色] 与转义后看有没有 y d h s 等。
func isDateCode(code string) bool {
	var b strings.Builder
	inQ, inBr := false, false
	for i := 0; i < len(code); i++ {
		ch := code[i]
		switch {
		case ch == '"':
			inQ = !inQ
		case inQ:
		case ch == '[':
			inBr = true
		case ch == ']':
			inBr = false
		case inBr:
		case ch == '\\' || ch == '_' || ch == '*':
			i++
		default:
			b.WriteByte(ch)
		}
	}
	s := strings.ToLower(b.String())
	return strings.ContainsAny(s, "ydh") || (strings.Contains(s, "s") && strings.Contains(s, ":"))
}

// 单元格:r="B3" t="s" s="2" 下的 <v> 或 <is><t>。
func readSheet(open func(string) (io.ReadCloser, error), target string, sst []string, dateXf []bool, date1904 bool) (*Sheet, error) {
	rc, err := open(target)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	dec := xml.NewDecoder(rc)
	sh := &Sheet{}
	row, col := -1, -1
	var (
		cellT, cellV string
		cellStyle    int
		inV, inIS    bool
		isText       strings.Builder
		inT          bool
		cellActive   bool
		cellRow      int
		cellCol      int
	)
	put := func(r, c int, v string) {
		if r >= MaxRows {
			sh.Truncated = true
			return
		}
		if c >= MaxCols {
			sh.Truncated = true
			return
		}
		for len(sh.Rows) <= r {
			sh.Rows = append(sh.Rows, nil)
		}
		for len(sh.Rows[r]) <= c {
			sh.Rows[r] = append(sh.Rows[r], "")
		}
		sh.Rows[r][c] = v
		if c+1 > sh.Cols {
			sh.Cols = c + 1
		}
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch e := tok.(type) {
		case xml.StartElement:
			switch e.Name.Local {
			case "row":
				row++
				col = -1
				for _, a := range e.Attr {
					if a.Name.Local == "r" {
						if n, err := strconv.Atoi(a.Value); err == nil && n > 0 {
							row = n - 1
						}
					}
				}
				if row >= MaxRows {
					sh.Truncated = true
					return sh, nil // 行是按顺序出现的,后面的都超限
				}
			case "c":
				col++
				cellActive = true
				cellT, cellV, cellStyle = "", "", 0
				isText.Reset()
				cellRow = row
				for _, a := range e.Attr {
					switch a.Name.Local {
					case "r":
						if ci := colIndex(a.Value); ci >= 0 {
							col = ci
						}
					case "t":
						cellT = a.Value
					case "s":
						cellStyle, _ = strconv.Atoi(a.Value)
					}
				}
				cellCol = col
			case "v":
				inV = true
			case "is":
				inIS = true
			case "t":
				inT = inIS
			}
		case xml.EndElement:
			switch e.Name.Local {
			case "v":
				inV = false
			case "is":
				inIS = false
			case "t":
				inT = false
			case "c":
				if cellActive {
					val := cellValue(cellT, cellV, isText.String(), cellStyle, sst, dateXf, date1904)
					if val != "" {
						put(cellRow, cellCol, val)
					}
				}
				cellActive = false
			}
		case xml.CharData:
			switch {
			case inV:
				cellV += string(e)
			case inT:
				isText.Write(e)
			}
		}
	}
	return sh, nil
}

func cellValue(t, v, inline string, style int, sst []string, dateXf []bool, date1904 bool) string {
	switch t {
	case "s":
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && i >= 0 && i < len(sst) {
			return sst[i]
		}
		return ""
	case "inlineStr":
		return inline
	case "b":
		if strings.TrimSpace(v) == "1" {
			return "TRUE"
		}
		return "FALSE"
	case "str", "e", "d":
		return v
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return v
	}
	if style >= 0 && style < len(dateXf) && dateXf[style] {
		return excelDate(f, date1904)
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if len(s) > 16 { // 0.30000000000000004 这类浮点噪声:按 15 位有效数字
		s = strconv.FormatFloat(f, 'g', 15, 64)
	}
	return s
}

// excelDate 把 Excel 序列日期换成文本;整数天只给日期,有小数给日期时间,小于 1 只给时间。
func excelDate(serial float64, date1904 bool) string {
	epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	if date1904 {
		epoch = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	secs := int64(serial*86400 + 0.5)
	t := epoch.Add(time.Duration(secs) * time.Second)
	switch {
	case serial >= 0 && serial < 1:
		return t.Format("15:04:05")
	case serial == float64(int64(serial)):
		return t.Format("2006-01-02")
	}
	return t.Format("2006-01-02 15:04:05")
}
