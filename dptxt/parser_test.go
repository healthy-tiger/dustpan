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
			if x != y {
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
			"test": {
				Value: []*Paragraph{
					{
						Value: []string{
							"hello",
						},
					},
				},
				peekedValue: "hello",
			},
			"date": {
				Value: []*Paragraph{
					{
						Value: []string{
							"2019/1/2",
						},
					},
				},
				peekedValue: "2019/1/2",
			},
			"title": {
				Value: []*Paragraph{
					{
						Value: []string{
							"dptxt parse",
						},
					},
				},
				peekedValue: "dptxt parse",
			},
			"description": {
				Value: []*Paragraph{
					{
						Value: []string{
							"ほんじつは、",
							"おひがらもよく、",
							"云々。。。",
						},
					},
					{
						Value: []string{
							"あれこれ",
							"これそれ",
						},
					},
				},
				peekedValue: "ほんじつは、",
			},
			"作者": {
				Value: []*Paragraph{
					{
						Value: []string{
							"ボブ",
						},
					},
				},
				peekedValue: "ボブ",
			},
			"compile option": {
				Value: []*Paragraph{
					{
						Value: []string{
							"-O2",
						},
					},
				},
				peekedValue: "-O2",
			},
			"author": {
				Value:       []*Paragraph{},
				peekedValue: "",
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
	dates := []string{
		"2019/11/13",
		"2019-11-13",
		"2019.11.13",
		"2019年11月13日",
	}

	for _, d := range dates {
		year, month, day, pre, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, string(pre), err)
		}
		if year != 2019 || month != 11 || day != 13 {
			t.Error(string(d), year, month, day, string(pre), err)
		}
	}
}

// 月と日が一桁
func TestParseDate2(t *testing.T) {
	dates := []string{
		"2003/1/3",
		"2003-1-3",
		"2003.1.3",
		"2003年1月3日",
	}

	for _, d := range dates {
		year, month, day, pre, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, string(pre), err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, string(pre), err)
		}
	}
}

// 月と日がゼロ埋めありの二桁
func TestParseDate3(t *testing.T) {
	dates := []string{
		"2003/01/03",
		"2003-01-03",
		"2003.01.03",
		"2003年01月03日",
	}

	for _, d := range dates {
		year, month, day, pre, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, string(pre), err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, string(pre), err)
		}
	}
}

// 区切り文字に全角と半角が混在
func TestParseDate4(t *testing.T) {
	dates := []string{
		"2003／01/03",
		"2003―01-03",
		"2003．01.03",
		"2003／1/3",
		"2003―1-3",
		"2003．1.3",
		"2003/01／03",
		"2003-01―03",
		"2003.01．03",
		"2003/1／3",
		"2003-1―3",
		"2003.1．3",
	}

	for _, d := range dates {
		year, month, day, pre, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, string(pre), err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, string(pre), err)
		}
	}
}

// 前後に空白、空白のあとに単語
func TestParseDate5(t *testing.T) {
	dates := []string{
		"  2003.1.3",
		"2003.1.3  ",
		"  2003.1.3  ",
		"  2003年1月3日",
		"2003年1月3日  ",
		"  2003年01月03日  ",
		"2003   年01月03日",
		"2003   年1月3日",
		"2003年   01月03日",
		"2003年01  月03日",
		"2003年   1月03日",
		"2003年1   月03日",
		"2003年   01  月03日",
		"2003年   1   月03日",
		"2003年01月   03日",
		"2003年01月03   日",
		"2003年01月   3日",
		"2003年01月3   日",
		"2003年01月   03   日",
		"2003年01月   3   日",
		"  2003.1.3 b ",
		"2003年01月   3   日 太郎",
	}

	for _, d := range dates {
		year, month, day, pre, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, string(pre), err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, string(pre), err)
		}
	}
}

// 数字に全角と半角が混在
func TestParseDate6(t *testing.T) {
	dates := []string{
		"  2０0３.1.3",
		"2003.１.3  ",
		"  2003.1.３  ",
		"  ２003年1月3日",
		"200３年1月3日  ",
		"  2003年０1月0３日  ",
	}

	for _, d := range dates {
		year, month, day, pre, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, string(pre), err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, string(pre), err)
		}
	}
}

func TestParseDateErr(t *testing.T) {
	dates := []string{
		".1.3",
		"2０３.1.3",
		"2０0３..3",
		"2０0３.1.",
		"2０0３.2. ",
		"2０0３4.1.3",
		"２003年1日3日",
		"2０0３.1-3",
		"2０0３/1.3",
		"2０0３-1.3",
		"a  2003.1.3  ",
		"  2003.1.3b ",
		"  2003年1月3日太郎  ",
		"  2003 ",
		"  2003.1 ",
		"  2003.111.3 ",
		"  003.1.3333 ",
	}

	for _, d := range dates {
		year, month, day, pre, err := ParseDate(d)
		if err == nil {
			t.Error(string(d), year, month, day, string(pre))
		}
	}
}

func TestParseLogDate1(t *testing.T) {
	dates := []string{
		"abc(2019/11/13)",
		"abc(2019-11-13)",
		"abc(2019.11.13)",
		"abc(2019年11月13日)",
	}
	for _, d := range dates {
		year, month, day, pre, post, err := ParseLogDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, string(pre), string(post), err)
		}
		if year != 2019 || month != 11 || day != 13 {
			t.Error(string(d), year, month, day, string(pre), string(post), err)
		}
	}
}

func TestParseLogDate2(t *testing.T) {
	dates := []string{
		"hello ( 2003/01/03   ) ",
		"hello （ 2003/01/03   ) ",
		"hello（2003.01.03)",
		"hello(2003/1/3）",
		"hello ( 2003/01/03   ) ",
		"hello（　　2003.01.03　)",
		"hello(2003/1/3）",
		"hello(2003/1/3 world）",
		"(2003/1/3)",
		"(2003/1/3 john)",
	}

	for _, d := range dates {
		year, month, day, pre, post, err := ParseLogDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, string(pre), string(post), err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, string(pre), string(post), err)
		}
	}
}

func TestParseLogDate3(t *testing.T) {
	d := "hello(2003/1/3 world）"
	year, month, day, pre, post, err := ParseLogDate(d)
	if err != nil {
		t.Error(d, year, month, day, pre, post, err)
	}
	if year != 2003 || month != 1 || day != 3 {
		t.Error(d, year, month, day, pre, post, err)
	}
	if pre != "hello" {
		t.Error(d, year, month, day, pre, post, err)
	}
	if post != "world" {
		t.Error(d, year, month, day, pre, post, err)
	}

	d = "(2003/1/3 john)"
	year, month, day, pre, post, err = ParseLogDate(d)
	if err != nil {
		t.Error(d, year, month, day, pre, post, err)
	}
	if year != 2003 || month != 1 || day != 3 {
		t.Error(d, year, month, day, pre, post, err)
	}
	if pre != "" {
		t.Error(d, year, month, day, pre, post, err)
	}
	if post != "john" {
		t.Error(d, year, month, day, pre, post, err)
	}

	d = "hello(2003年1月3日 world)"
	year, month, day, pre, post, err = ParseLogDate(d)
	if err != nil {
		t.Error(d, year, month, day, pre, post, err)
	}
	if year != 2003 || month != 1 || day != 3 {
		t.Error(d, year, month, day, pre, post, err)
	}
	if pre != "hello" {
		t.Error(d, year, month, day, pre, post, err)
	}
	if post != "world" {
		t.Error(d, year, month, day, pre, post, err)
	}
}

func TestParseLogDateErr(t *testing.T) {
	dates := []string{
		"hello(2003/01/03",
		"hello 2003/01/03)",
		"hello 2003/01/03 ",
		"hello(203/01/03)",
		"hello(world)",
		"hello ( 2003/01/03   )) ",
		"hello ( 2003/01/03   )   ) ",
		"hello ((((( 2003/01/03   )   ) ",
		"hello( 2003-01-03) world",
		"hello　　( 2003-01-03 ) world",
		"(2003/1/3) hello world",
	}

	for _, d := range dates {
		year, month, day, pre, post, err := ParseLogDate(d)
		if err == nil {
			t.Error(d, year, month, day, pre, post)
		}
	}
}
