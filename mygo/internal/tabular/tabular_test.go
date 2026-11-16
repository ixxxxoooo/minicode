package tabular

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestColName(t *testing.T) {
	for i, want := range map[int]string{0: "A", 25: "Z", 26: "AA", 27: "AB", 701: "ZZ", 702: "AAA"} {
		if got := ColName(i); got != want {
			t.Errorf("ColName(%d)=%s want %s", i, got, want)
		}
		if got := colIndex(want + "12"); got != i {
			t.Errorf("colIndex(%s12)=%d want %d", want, got, i)
		}
	}
}

func TestParseCSVDelimiters(t *testing.T) {
	cases := map[string]string{
		"逗号": "a,b,c\n1,2,3\n",
		"分号": "a;b;c\n1;2;3\n",
		"竖线": "a|b|c\n1|2|3\n",
		"制表": "a\tb\tc\n1\t2\t3\n",
	}
	for name, in := range cases {
		wb, err := ParseCSV([]byte(in), ".csv", false)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		sh := wb.Sheets[0]
		if len(sh.Rows) != 2 || sh.Cols != 3 || sh.Cell(1, 2) != "3" {
			t.Errorf("%s: rows=%v cols=%d", name, sh.Rows, sh.Cols)
		}
	}
}

func TestParseCSVQuotesBOMAndRagged(t *testing.T) {
	in := "\xEF\xBB\xBF名称,备注\n\"a,b\",\"他说\"\"好\"\"\"\n短\n"
	wb, err := ParseCSV([]byte(in), ".csv", false)
	if err != nil {
		t.Fatal(err)
	}
	sh := wb.Sheets[0]
	if sh.Cell(0, 0) != "名称" || sh.Cell(1, 0) != "a,b" || sh.Cell(1, 1) != `他说"好"` || sh.Cell(2, 1) != "" {
		t.Errorf("rows=%q", sh.Rows)
	}
}

func TestParseCSVGBKAndBadEncoding(t *testing.T) {
	gbk, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte("姓名,城市\n张三,北京\n"))
	if err != nil {
		t.Fatal(err)
	}
	wb, err := ParseCSV(gbk, ".csv", false)
	if err != nil || wb.Sheets[0].Cell(1, 1) != "北京" || !strings.Contains(wb.Note, "GBK") {
		t.Fatalf("GBK: %v %+v", err, wb)
	}
	if _, err := ParseCSV([]byte{0xff, 0xfe, 0xfd, 0x80, 0x81}, ".csv", false); err == nil {
		t.Error("乱码应报编码错误")
	}
}

func TestParseCSVLimits(t *testing.T) {
	var b strings.Builder
	for i := 0; i < MaxRows+10; i++ {
		b.WriteString("1,2\n")
	}
	wb, err := ParseCSV([]byte(b.String()), ".csv", false)
	if err != nil {
		t.Fatal(err)
	}
	if sh := wb.Sheets[0]; len(sh.Rows) != MaxRows || !sh.Truncated {
		t.Errorf("rows=%d truncated=%v", len(sh.Rows), sh.Truncated)
	}
}

// makeXLSX 现场拼一个最小的 xlsx:两张表,含共享字符串、内联字符串、日期、布尔与跳格。
func makeXLSX(t *testing.T) []byte {
	t.Helper()
	files := map[string]string{
		"xl/workbook.xml":            `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="销售" sheetId="1" r:id="rId1"/><sheet name="隐藏" sheetId="2" state="hidden" r:id="rId2"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Target="worksheets/sheet2.xml"/></Relationships>`,
		"xl/sharedStrings.xml":       `<sst><si><t>地区</t></si><si><r><t>华</t></r><r><t>东</t></r></si><si><t>x</t><rPh><t>拼音</t></rPh></si></sst>`,
		"xl/styles.xml":              `<styleSheet><numFmts><numFmt numFmtId="164" formatCode="yyyy&quot;年&quot;m&quot;月&quot;"/></numFmts><cellXfs><xf numFmtId="0"/><xf numFmtId="14"/><xf numFmtId="164"/></cellXfs></styleSheet>`,
		"xl/worksheets/sheet1.xml": `<worksheet><sheetData>` +
			`<row r="1"><c r="A1" t="s"><v>0</v></c><c r="C1" t="inlineStr"><is><t>内联</t></is></c></row>` +
			`<row r="3"><c r="A3" t="s"><v>1</v></c><c r="B3"><v>0.30000000000000004</v></c><c r="C3" s="1"><v>45292</v></c><c r="D3" t="b"><v>1</v></c><c r="E3" t="s"><v>2</v></c></row>` +
			`</sheetData></worksheet>`,
		"xl/worksheets/sheet2.xml": `<worksheet><sheetData><row r="1"><c r="A1"><v>7</v></c></row></sheetData></worksheet>`,
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestParseXLSX(t *testing.T) {
	wb, err := ParseXLSX(makeXLSX(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(wb.Sheets) != 1 || wb.Sheets[0].Name != "销售" {
		t.Fatalf("隐藏表应被跳过: %+v", wb.Sheets)
	}
	sh := wb.Sheets[0]
	checks := []struct {
		r, c int
		want string
	}{
		{0, 0, "地区"}, {0, 2, "内联"}, {2, 0, "华东"}, {2, 1, "0.3"},
		{2, 2, "2024-01-01"}, {2, 3, "TRUE"}, {2, 4, "x"}, {1, 0, ""},
	}
	for _, c := range checks {
		if got := sh.Cell(c.r, c.c); got != c.want {
			t.Errorf("cell(%d,%d)=%q want %q", c.r, c.c, got, c.want)
		}
	}
	if sh.Cols != 5 || len(sh.Rows) != 3 {
		t.Errorf("rows=%d cols=%d", len(sh.Rows), sh.Cols)
	}
}

func TestParseXLSXRejectsGarbage(t *testing.T) {
	if _, err := ParseXLSX([]byte("not a zip")); err == nil {
		t.Error("非 zip 应报错")
	}
}

func TestIsDateCode(t *testing.T) {
	for code, want := range map[string]bool{
		`yyyy"年"m"月"`: true, "hh:mm:ss": true, "0.00": false, `"d"0`: false, "[Red]0.0": false, "#,##0": false,
	} {
		if got := isDateCode(code); got != want {
			t.Errorf("isDateCode(%q)=%v want %v", code, got, want)
		}
	}
}
