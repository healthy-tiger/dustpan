package dptxt

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
)

// エラーメッセージ
var (
	ErrorIllegalCharBeforeSectionName = errors.New("セクション名の前に不正な文字があります。")
	ErrorNoSectionNamePrefix          = errors.New("セクション名プリフィックスがありません。")
	ErrorNoSectionNameSuffix          = errors.New("セクション名サフィックスがありません。")
	ErrorSectionNameIsEmpty           = errors.New("セクション名が空です。")
	ErrorUnexpectedText               = errors.New("予期しない入力文字列です。")
)

const empty = ""

func isSp(r rune) bool {
	if r == ' ' || r == '　' || r == '\t' {
		return true
	}
	return false
}

func isNonSp(r rune) bool {
	return !isSp(r)
}

func isAt(r rune) bool {
	if r == '@' || r == '＠' {
		return true
	}
	return false
}

func isNonAt(r rune) bool {
	return !isAt(r)
}

func isColon(r rune) bool {
	if r == ':' || r == '：' {
		return true
	}
	return false
}

func isNonColon(r rune) bool {
	return !isColon(r)
}

type Document struct {
	Filename string
	Sections map[string]*Section
}

type Section struct {
	Value       []*Paragraph
	peekedValue string
}

type Paragraph struct {
	Value [][]byte
}

func (p *Paragraph) String() string {
	return string(bytes.Join(p.Value, []byte("\\n")))
}

func (s *Section) String() string {
	buf := make([]string, 0, len(s.Value))
	for _, p := range s.Value {
		buf = append(buf, p.String())
	}
	return "\"" + strings.Join(buf, "\\n") + "\""
}

func (d *Document) String() string {
	buf := make([]string, 0, len(d.Sections)+1)
	buf = append(buf, d.Filename)
	for n, v := range d.Sections {
		buf = append(buf, "\""+n+"\":"+v.String())
	}
	return strings.Join(buf, ",")
}

func (s *Section) PeekValue() string {
	if len(s.Value) == 0 {
		return empty
	}
	if s.peekedValue == empty {
		s.peekedValue = string(s.Value[0].Value[0])
	}
	return s.peekedValue
}

type lineScanner struct {
	scanner  *bufio.Scanner
	lastline []byte
	unread   bool
}

func newLineScanner(r io.Reader) *lineScanner {
	return &lineScanner{bufio.NewScanner(r), nil, false}
}

func (ls *lineScanner) nextLine() ([]byte, error) {
	if ls.unread && ls.lastline != nil {
		ls.unread = false
		return ls.lastline, nil
	}
	if ls.scanner.Scan() {
		b := ls.scanner.Bytes()
		t := make([]byte, len(b), len(b))
		copy(t, b)
		ls.lastline = t
		ls.unread = false
		return t, nil
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
func (ls *lineScanner) SkipEmptyLines() (int, error) {
	n := 0
	line, err := ls.nextLine()
	for err == nil {
		line = bytes.TrimLeftFunc(line, isSp)
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

func (ls *lineScanner) UnreadLine() bool {
	// 戻せるのは一行分だけ。
	// すでに戻してある分がある場合は、エラーになる。
	if ls.unread {
		return false
	}
	ls.unread = true
	return true
}

func processSection(ls *lineScanner, sec *Section) (string, error) {
	// 空白行を読み飛ばす。
	ls.SkipEmptyLines()
	line, err := ls.nextLine()
	if err != nil {
		return empty, err
	}
	line = bytes.TrimLeftFunc(line, isSp)

	// セクション名の始まり
	i := bytes.IndexFunc(line, isAt)
	if i != 0 {
		return empty, ErrorNoSectionNamePrefix
	}
	line = line[i:]
	i = bytes.IndexFunc(line, isNonAt)
	if i == -1 {
		return empty, ErrorSectionNameIsEmpty
	}
	line = line[i:]

	// セクション名の終わり
	i = bytes.IndexFunc(line, isColon)
	if i == -1 { // コロンが見つからない
		return empty, ErrorNoSectionNameSuffix
	}
	name, err := normalizeText(line[:i]) // セクション名を正規化する。
	if err != nil {
		return empty, err
	}
	line = line[i:]

	// セクション本文の始まり
	i = bytes.IndexFunc(line, isNonColon)
	var head []byte = nil
	if i != -1 {
		// lineをトリミングしてあるので、必ずなにか文字列があるはず。
		head = bytes.TrimFunc(line[i:], isSp)
	}
	ps, err := readCompoundValues(ls, head)
	if err != nil {
		return empty, err
	}
	*sec = Section{ps, empty}
	return name, nil
}

func readCompoundValues(ls *lineScanner, head []byte) ([]*Paragraph, error) {
	values := make([]*Paragraph, 0)
	pvalues := make([][]byte, 0)
	if head != nil {
		pvalues = append(pvalues, head)
	}

	line, err := ls.nextLine()
	for err == nil {
		line = bytes.TrimFunc(line, isSp)
		if len(line) > 0 {
			if bytes.IndexFunc(line, isAt) == 0 { // 次のセクションまで来た。
				ls.UnreadLine()
				break
			} else {
				pvalues = append(pvalues, line)
			}
		} else {
			if len(pvalues) > 0 {
				values = append(values, &Paragraph{pvalues})
				pvalues = make([][]byte, 0)
			}
		}
		line, err = ls.nextLine()
	}

	if err != nil && err != io.EOF {
		return nil, err
	}

	if len(pvalues) > 0 {
		values = append(values, &Paragraph{pvalues})
	}
	return values, nil
}

func ParseDocument(filename string, r io.Reader, doc *Document) error {
	ls := newLineScanner(r)

	secs := make(map[string]*Section)
	var sec *Section = new(Section)
	name, err := processSection(ls, sec)
	for err == nil {
		secs[name] = sec
		sec = new(Section)
		name, err = processSection(ls, sec)
	}
	if err != io.EOF {
		return err
	}
	doc.Filename = filename
	doc.Sections = secs
	return nil
}

func normalizeText(b []byte) (string, error) {
	ps := make([][]byte, 0)
	b = bytes.TrimLeftFunc(b, isSp)
	for len(b) > 0 {
		i := bytes.IndexFunc(b, isSp)
		if i > 0 {
			ps = append(ps, b[:i])
			b = b[i:]
		} else if i < 0 {
			ps = append(ps, b)
			break
		}
		b = bytes.TrimLeftFunc(b, isSp)
	}
	if len(ps) == 0 {
		return empty, ErrorSectionNameIsEmpty
	}
	return string(bytes.Join(ps, []byte(" "))), nil
}
