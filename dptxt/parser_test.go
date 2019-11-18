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
				[]*Paragraph{
					&Paragraph{
						[][]byte{
							[]byte("hello"),
						},
					},
				},
				"hello",
				nil,
			},
			"date": &Section{
				[]*Paragraph{
					&Paragraph{
						[][]byte{
							[]byte("2019/1/2"),
						},
					},
				},
				"2019/1/2",
				nil,
			},
			"title": &Section{
				[]*Paragraph{
					&Paragraph{
						[][]byte{
							[]byte("dptxt parse"),
						},
					},
				},
				"dptxt parse",
				nil,
			},
			"description": &Section{
				[]*Paragraph{
					&Paragraph{
						[][]byte{
							[]byte("ほんじつは、"),
							[]byte("おひがらもよく、"),
							[]byte("云々。。。"),
						},
					},
					&Paragraph{
						[][]byte{
							[]byte("あれこれ"),
							[]byte("これそれ"),
						},
					},
				},
				"ほんじつは、",
				nil,
			},
			"作者": &Section{
				[]*Paragraph{
					&Paragraph{
						[][]byte{
							[]byte("ボブ"),
						},
					},
				},
				"ボブ",
				nil,
			},
			"compile option": &Section{
				[]*Paragraph{
					&Paragraph{
						[][]byte{
							[]byte("-O2"),
						},
					},
				},
				"-O2",
				nil,
			},
			"author": &Section{
				[]*Paragraph{},
				"",
				nil,
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

func TestParseDate(t *testing.T) {
	dates := [][]byte{
		[]byte("2019/11/13"),
		[]byte("2019-11-13"),
		[]byte("2019.11.13"),
		[]byte("2019年11月13日"),
	}
	dates2 := [][]byte{
		[]byte("2003/01/03"),
		[]byte("2003-01-03"),
		[]byte("2003.01.03"),
		[]byte("2003/1/3"),
		[]byte("2003-1-3"),
		[]byte("2003.1.3"),
		[]byte("2003／01/03"),
		[]byte("2003ー01-03"),
		[]byte("2003．01.03"),
		[]byte("2003／1/3"),
		[]byte("2003ー1-3"),
		[]byte("2003．1.3"),
		[]byte("2003/01／03"),
		[]byte("2003-01ー03"),
		[]byte("2003.01．03"),
		[]byte("2003/1／3"),
		[]byte("2003-1ー3"),
		[]byte("2003.1．3"),
		[]byte("2003年1月3日"),
		[]byte("2003年01月03日"),
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
	for _, d := range dates2 {
		year, month, day, err := ParseDate(d)
		if err != nil {
			t.Error(string(d), year, month, day, err)
		}
		if year != 2003 || month != 1 || day != 3 {
			t.Error(string(d), year, month, day, err)
		}
	}
}
