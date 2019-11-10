package github.com/healthy-tiger/dustpan/dptxt

import (
	"bufio"
	"strings"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"io"
)

var (
	ErrorIllegalCharBeforeSectionName = errors.New("セクション名の前に不正な文字があります。")
	ErrorNoSectionNamePrefix = errors.New("セクション名プリフィックスがありません。")
	ErrorNoSectionNameSuffix = errors.New("セクション名サフィックスがありません。")
	ErrorSectionNameIsEmpty = errors.New("セクション名が空です。")
	ErrorUnexpectedText = errors.New("予期しない入力文字列です。")
)

const defaultSectionNameLen = 20
const defaultSectionParagraphNum = 20
const defaultParagraphLineNum = 10


func IsSp(r rune) bool {
	if r == ' ' || r == '　' {
		return true
	}
	return false
}

func IsNonSp(r rune) bool {
	return !IsSp(r)
}

func IsAt(r rune) bool {
	if r == '@' || r == '＠' {
		return true
	}
	return false
}

func IsNonAt(r rune) bool {
	return !IsAt(r)
}

func IsColon(r rune) bool {
	if r == ':' || r == '：' {
		return true
	}
	return false
}

func IsNonColon(r rune) bool {
	return !IsColon(r)
}

type Document struct {
	Filename string
	Sections []*Section
}

type Section struct {
	Name []byte
	Value []Paragraph
	IsCompound bool
}

type Paragraph struct {
	Value [][]byte
}

func (d *Document) String() string {
	var b strings.Builder
	b.WriteString("-------document-------\n")
	b.WriteString("filename=")
	b.WriteString(d.Filename)
	b.WriteString("\n")
	for _, s := range d.Sections {
		b.WriteString(s.String())
		b.WriteString("\n")
	}
	b.WriteString("---------end----------")
	return b.String()
}

func (s *Section) String() string {
	var b strings.Builder
	b.WriteString("-----section-----\n")
	if s.IsCompound {
		b.WriteString("name*=")
	} else {
		b.WriteString("name =")
	}
	b.Write(s.Name)
	b.WriteString("\n")
	for _, p := range s.Value {
		b.WriteString(p.String())
		b.WriteString("\n")
	}
	b.WriteString("-------end-------")
	return b.String()
}

func (p *Paragraph) String() string {
	var b strings.Builder
	b.WriteString("--paragraph--\n")
	for _, v := range p.Value {
		b.Write(v)
		b.WriteString("\n")
	}
	b.WriteString("-----end-----")
	return b.String()
}

type LineScanner struct {
	scanner *bufio.Scanner
	lastline []byte
	unread bool
}

func NewLineScanner(r io.Reader) *LineScanner {
	return &LineScanner{ bufio.NewScanner(r), nil, false }
}

func (ls *LineScanner) nextLine() ([]byte, error) {
	if ls.unread && ls.lastline != nil {
		ls.unread = false
		return ls.lastline, nil
	}
	if ls.scanner.Scan() {
		b := ls.scanner.Bytes()
		ls.lastline = b
		ls.unread = false
		return b, nil
	}
	err := ls.scanner.Err()
	if err == nil {
		// 入力の最後まで逹っした場合、s.Scan()はfalseを返すが、
		// エラーはnilになる。なので、敢えてio.EOFにして非nilな返り値を返すようにする。
		err = io.EOF
	}
	return nil, err
}

// 空白行を読み飛ばす。
func (ls *LineScanner) SkipEmptyLines() (int, error) {
	n := 0
	line, err := ls.nextLine()
	for err == nil {
		line = bytes.TrimFunc(line, IsSp)
		if len(line) > 0 {
			break
		}
		n++
		line, err = ls.nextLine()
	}
	if err != nil {
		return 0, err
	}
	ls.UnreadLine()
	return n, nil
}

func (ls *LineScanner) UnreadLine() bool {
	// 戻せるのは一行分だけ。
	// すでに戻してある分がある場合は、エラーになる。
	if ls.unread {
		return false
	}
	ls.unread = true
	return true
}

func processSection(ls *LineScanner) (*Section, error)  {
	// 空白行を読み飛ばす。
	ls.SkipEmptyLines()
	line, err := ls.nextLine()
	if err != nil {
		return nil, err
	}
	line = bytes.TrimFunc(line, IsSp)

	// セクション名の始まり
	i := bytes.IndexFunc(line, IsAt)  
	if i != 0 {
		return nil, ErrorNoSectionNamePrefix
	}
	line = line[i:]
	i = bytes.IndexFunc(line, IsNonAt)
	if i == -1 {
		return nil, ErrorSectionNameIsEmpty
	}
	line = line[i:]
	
	// セクション名の終わり
	i = bytes.IndexFunc(line, IsColon)
	if i == -1 { // コロンが見つからない
		return nil, ErrorNoSectionNameSuffix
	}
	name := bytes.TrimFunc(line[:i], IsSp)
	if len(name) == 0 { // セクション名が長さ0
		return nil, ErrorSectionNameIsEmpty
	}
	line = line[i:]

	// セクション本文の始まり
	i = bytes.IndexFunc(line, IsNonColon)
	if i == -1 {
		// 複合セクション
		ps, err := readCompoundValues(ls)
		if err != nil {
			return nil, err
		}
		return &Section{ name, ps, true }, nil
	} else {
		// 単純セクション

		// lineをトリミングしてあるので、必ずなにか文字列があるはず。
		value := bytes.TrimFunc(line[i:], IsSp)
		return &Section{ name, []Paragraph{ Paragraph{ [][]byte{ value } } }, false }, nil
	}

}

func readCompoundValues(ls *LineScanner) ([]Paragraph, error) {
	values := make([]Paragraph, 0, defaultSectionParagraphNum)
	pvalues := make([][]byte, 0, defaultParagraphLineNum)

	line, err := ls.nextLine()
	for err == nil {
		line = bytes.TrimFunc(line, IsSp)
		if len(line) > 0 {
			if bytes.IndexFunc(line, IsAt) == 0 { // 次のセクションまで来た。
				ls.UnreadLine()
				break
			} else {
				pvalues = append(pvalues, line)
			}
		} else {
			if len(pvalues) > 0 {
				values = append(values, Paragraph{ pvalues })
				pvalues = make([][]byte, 0, defaultParagraphLineNum)
			}
		}
		line, err = ls.nextLine()
	}

	if err != nil && err != io.EOF {
		return nil, err
	}

	if len(pvalues) > 0 {
		values = append(values, Paragraph{ pvalues })
	}
	return values, nil
}

func ProcessFile(filename string) (*Document, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ls := NewLineScanner(f)

	secs := make([]*Section, 0, 10)
	var sec *Section
	sec, err = processSection(ls)
	for err == nil {
		secs = append(secs, sec)
		sec, err = processSection(ls)
	}
	if err != io.EOF {
		return nil, err
	}
	return &Document{ filename, secs }, nil
}

func main() {
	for i := 1; i < len(os.Args); i++ {
		if paths, err := filepath.Glob(os.Args[i]); err != nil {
			log.Fatal(err)
		} else {
			for _, p := range paths {
				doc, err := ProcessFile(p)
				if err != nil {
					log.Println(p, err)
				} else {
					fmt.Println(doc)
				}
			}
		}
	}
}
