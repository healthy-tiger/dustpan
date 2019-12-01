package dptxt

import (
	"bytes"
	"testing"
)

func compareParagraph(t *testing.T, a, b *Paragraph) {
	if len(a.Value) != len(b.Value) {
		t.Error("Paragraph.Value", len(a.Value), len(b.Value))
	} else {
		for i := 0; i < len(a.Value); i++ {
			x := a.Value[i]
			y := b.Value[i]
			if !bytes.Equal(x, y) {
				t.Error("Paragraph.Value.[]byte", x, y)
			}
		}
	}
}

func compareSection(t *testing.T, a, b *Section) {
	if len(a.Value) != len(b.Value) {
		t.Error("Section.Value", len(a.Value), len(b.Value))
	} else {
		for i := 0; i < len(a.Value); i++ {
			compareParagraph(t, a.Value[i], b.Value[i])
		}
		pa := a.PeekString()
		pb := b.PeekString()
		if pa != pb {
			t.Error("Section.PeekValue", pa, pb)
		}
	}
}

func compareDocument(t *testing.T, a, b *Document) {
	if a.Filename != b.Filename {
		t.Error("Document.Filename", a.Filename, b.Filename)
	}
	if len(a.Sections) != len(b.Sections) {
		t.Error("Document.Sections", len(a.Sections), len(b.Sections))
	} else {
		for k, v := range a.Sections {
			bv, ok := b.Sections[k]
			if !ok {
				t.Error("Document.Sections a", k, a.Sections)
				t.Error("Document.Sections b", k, b.Sections)
			} else {
				compareSection(t, v, bv)
			}
		}
	}
}

func TestParseRaw(t *testing.T) {
	src := `@test: hello
@date: 2019/1/2
@title: dptxt parse
@description: ほんじつは、
おひがらもよく、
云々。。。

あれこれ
これそれ

＠作者：ボブ
@  compile 　	 option　  : -O2  

@author:    

`
	expected := Document{
		"test1",
		map[string]*Section{
			"test": &Section{
				Value: []*Paragraph{
					&Paragraph{
						Value: [][]byte{
							[]byte("hello"),
						},
					},
				},
				peekedValue:"hello",
			},
			"date": &Section{
				Value:[]*Paragraph{
					&Paragraph{
						Value: [][]byte{
							[]byte("2019/1/2"),
						},
					},
				},
				peekedValue:"2019/1/2",
			},
			"title": &Section{
				Value: []*Paragraph{
					&Paragraph{
						Value: [][]byte{
							[]byte("dptxt parse"),
						},
					},
				},
				peekedValue:"dptxt parse",
			},
			"description": &Section{
				Value:[]*Paragraph{
					&Paragraph{
						Value:[][]byte{
							[]byte("ほんじつは、"),
							[]byte("おひがらもよく、"),
							[]byte("云々。。。"),
						},
					},
					&Paragraph{
						Value:[][]byte{
							[]byte("あれこれ"),
							[]byte("これそれ"),
						},
					},
				},
				peekedValue:"ほんじつは、",
			},
			"作者": &Section{
				Value:[]*Paragraph{
					&Paragraph{
						Value:[][]byte{
							[]byte("ボブ"),
						},
					},
				},
				peekedValue:"ボブ",
			},
			"compile option": &Section{
				Value:[]*Paragraph{
					&Paragraph{
						Value:[][]byte{
							[]byte("-O2"),
						},
					},
				},
				peekedValue:"-O2",
			},
			"author": &Section{
				Value:[]*Paragraph{},
				peekedValue:"",
			},
		},
		nil,
	}

	var doc Document
	err := ParseDocument("test1", bytes.NewBufferString(src), &doc)
	if err != nil {
		t.Error(err)
	} else {
		compareDocument(t, &doc, &expected)
	}
}

// 月と日が二桁
func TestParseDate1(t *testing.T) {
	dates := [][]byte{
		[]byte("2019/11/13"),
		[]byte("2019-11-13"),
		[]byte("2019.11.13"),
		[]byte("2019年11月13日"),
	}

	for _, d := range dates {
		year, month, day, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, err)
		}
		if year != 2019 || month != 11 || day != 13 {
			t.Error(string(d), year, month, day, err)
		}
	}
}

// 月と日が一桁
func TestParseDate2(t *testing.T) {
	dates := [][]byte{
		[]byte("2003/1/3"),
		[]byte("2003-1-3"),
		[]byte("2003.1.3"),
		[]byte("2003年1月3日"),
	}

	for _, d := range dates {
		year, month, day, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, err)
		}
	}
}

// 月と日がゼロ埋めありの二桁
func TestParseDate3(t *testing.T) {
	dates := [][]byte{
		[]byte("2003/01/03"),
		[]byte("2003-01-03"),
		[]byte("2003.01.03"),
		[]byte("2003年01月03日"),
	}

	for _, d := range dates {
		year, month, day, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, err)
		}
	}
}

// 区切り文字に全角と半角が混在
func TestParseDate4(t *testing.T) {
	dates := [][]byte{
		[]byte("2003／01/03"),
		[]byte("2003―01-03"),
		[]byte("2003．01.03"),
		[]byte("2003／1/3"),
		[]byte("2003―1-3"),
		[]byte("2003．1.3"),
		[]byte("2003/01／03"),
		[]byte("2003-01―03"),
		[]byte("2003.01．03"),
		[]byte("2003/1／3"),
		[]byte("2003-1―3"),
		[]byte("2003.1．3"),
	}

	for _, d := range dates {
		year, month, day, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, err)
		}
	}
}

// 前後に空白、空白のあとに単語
func TestParseDate5(t *testing.T) {
	dates := [][]byte{
		[]byte("  2003.1.3"),
		[]byte("2003.1.3  "),
		[]byte("  2003.1.3  "),
		[]byte("  2003年1月3日"),
		[]byte("2003年1月3日  "),
		[]byte("  2003年01月03日  "),
		[]byte("2003   年01月03日"),
		[]byte("2003   年1月3日"),
		[]byte("2003年   01月03日"),
		[]byte("2003年01  月03日"),
		[]byte("2003年   1月03日"),
		[]byte("2003年1   月03日"),
		[]byte("2003年   01  月03日"),
		[]byte("2003年   1   月03日"),
		[]byte("2003年01月   03日"),
		[]byte("2003年01月03   日"),
		[]byte("2003年01月   3日"),
		[]byte("2003年01月3   日"),
		[]byte("2003年01月   03   日"),
		[]byte("2003年01月   3   日"),
		[]byte("  2003.1.3 b "),
		[]byte("2003年01月   3   日 太郎"),
	}

	for _, d := range dates {
		year, month, day, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, err)
		}
	}
}

// 数字に全角と半角が混在
func TestParseDate6(t *testing.T) {
	dates := [][]byte{
		[]byte("  2０0３.1.3"),
		[]byte("2003.１.3  "),
		[]byte("  2003.1.３  "),
		[]byte("  ２003年1月3日"),
		[]byte("200３年1月3日  "),
		[]byte("  2003年０1月0３日  "),
	}

	for _, d := range dates {
		year, month, day, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, err)
		}
	}
}

func TestParseDateErr(t *testing.T) {
	dates := [][]byte{
		[]byte(".1.3"),
		[]byte("2０３.1.3"),
		[]byte("2０0３..3"),
		[]byte("2０0３.1."),
		[]byte("2０0３.2. "),
		[]byte("2０0３4.1.3"),
		[]byte("２003年1日3日"),
		[]byte("2０0３.1-3"),
		[]byte("2０0３/1.3"),
		[]byte("2０0３-1.3"),
		[]byte("a  2003.1.3  "),
		[]byte("  2003.1.3b "),
		[]byte("  2003年1月3日太郎  "),
		[]byte("  2003 "),
		[]byte("  2003.1 "),
		[]byte("  2003.111.3 "),
		[]byte("  003.1.3333 "),
	}

	for _, d := range dates {
		year, month, day, err := ParseDate(d)
		if err == nil {
			t.Error(string(d), year, month, day)
		}
	}
}

func TestParseLogDate1(t *testing.T) {
	dates := [][]byte{
		[]byte("abc(2019/11/13)"),
		[]byte("abc(2019-11-13)"),
		[]byte("abc(2019.11.13)"),
		[]byte("abc(2019年11月13日)"),
	}
	for _, d := range dates {
		year, month, day, err := ParseLogDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, err)
		}
		if year != 2019 || month != 11 || day != 13 {
			t.Error(string(d), year, month, day, err)
		}
	}
}

func TestParseLogDate2(t *testing.T) {
	dates := [][]byte{
		[]byte("hello ( 2003/01/03   ) "),
		[]byte("hello （ 2003/01/03   ) "),
		[]byte("hello（2003.01.03)"),
		[]byte("hello(2003/1/3）"),
		[]byte("hello ( 2003/01/03   ) "),
		[]byte("hello（　　2003.01.03　)"),
		[]byte("hello(2003/1/3）"),
		[]byte("hello(2003/1/3 world）"),
		[]byte("(2003/1/3)"),
	}

	for _, d := range dates {
		year, month, day, err := ParseLogDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, err)
		}
	}
}

func TestParseLogDateErr(t *testing.T) {
	dates := [][]byte{
		[]byte("hello(2003/01/03"),
		[]byte("hello 2003/01/03)"),
		[]byte("hello 2003/01/03 "),
		[]byte("hello(203/01/03)"),
		[]byte("hello(world)"),
		[]byte("hello ( 2003/01/03   )) "),
		[]byte("hello ( 2003/01/03   )   ) "),
		[]byte("hello ((((( 2003/01/03   )   ) "),
		[]byte("hello( 2003-01-03) world"),
		[]byte("hello　　( 2003-01-03 ) world"),
		[]byte("(2003/1/3) hello world"),
	}

	for _, d := range dates {
		year, month, day, err := ParseLogDate(d)
		if err == nil {
			t.Error(string(d), year, month, day)
		}
	}
}
