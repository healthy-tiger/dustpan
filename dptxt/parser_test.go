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
		pa := a.PeekValue()
		pb := b.PeekValue()
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
			},
			"author": &Section{
				[]*Paragraph{},
				"",
			},
		},
	}

	var doc Document
	err := ParseDocument("test1", bytes.NewBufferString(src), &doc)
	if err != nil {
		t.Error(err)
	} else {
		compareDocument(t, &doc, &expected)
	}
}
