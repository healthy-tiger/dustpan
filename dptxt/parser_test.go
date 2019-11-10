package dptxt

import (
	"bytes"
	"testing"
)

func compareParagraph(t *testing.T, a, b Paragraph) {
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

func compareSection(t *testing.T, a, b Section) {
	if !bytes.Equal(a.Name, b.Name) {
		t.Error("Section.Name", string(a.Name), string(b.Name))
	}
	if len(a.Value) != len(b.Value) {
		t.Error("Section.Value", len(a.Value), len(b.Value))
	} else {
		for i := 0; i < len(a.Value); i++ {
			compareParagraph(t, a.Value[i], b.Value[i])
		}
	}
}

func compareDocument(t *testing.T, a, b Document) {
	if a.Filename != b.Filename {
		t.Error("Document.Filename", a.Filename, b.Filename)
	}
	if len(a.Sections) != len(b.Sections) {
		t.Error("Document.Sections", len(a.Sections), len(b.Sections))
	} else {
		for i := 0; i < len(a.Sections); i++ {
			compareSection(t, a.Sections[i], b.Sections[i])
		}
	}
}

func TestParse(t *testing.T) {
	src := `@test: hello
@date: 2019/1/2
@title: dptxt parse
@description: ほんじつは、
おひがらもよく、
云々。。。`
	expected := Document{
		"test1",
		[]Section{
			Section{
				[]byte("test"),
				[]Paragraph{
					Paragraph{
						[][]byte{
							[]byte("hello"),
						},
					},
				},
			},
			Section{
				[]byte("date"),
				[]Paragraph{
					Paragraph{
						[][]byte{
							[]byte("2019/1/2"),
						},
					},
				},
			},
			Section{
				[]byte("title"),
				[]Paragraph{
					Paragraph{
						[][]byte{
							[]byte("dptxt parse"),
						},
					},
				},
			},
			Section{
				[]byte("description"),
				[]Paragraph{
					Paragraph{
						[][]byte{
							[]byte("ほんじつは、"),
							[]byte("おひがらもよく、"),
							[]byte("云々。。。"),
						},
					},
				},
			},
		},
	}

	doc, err := ParseDocument("test1", bytes.NewBufferString(src))
	if err != nil {
		t.Error(err)
	} else {
		compareDocument(t, *doc, expected)
	}
}
