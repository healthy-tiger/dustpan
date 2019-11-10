package dptxt

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

// エラーメッセージ
var (
	ErrorIllegalCharBeforeSectionName = errors.New("セクション名の前に不正な文字があります。")
	ErrorNoSectionNamePrefix          = errors.New("セクション名プリフィックスがありません。")
	ErrorNoSectionNameSuffix          = errors.New("セクション名サフィックスがありません。")
	ErrorSectionNameIsEmpty           = errors.New("セクション名が空です。")
	ErrorUnexpectedText               = errors.New("予期しない入力文字列です。")
)

const defaultSectionNameLen = 20
const defaultSectionParagraphNum = 20
const defaultParagraphLineNum = 10

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
	Sections []Section
}

type Section struct {
	Name  []byte
	Value []Paragraph
}

type Paragraph struct {
	Value [][]byte
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

func processSection(ls *lineScanner) (*Section, error) {
	// 空白行を読み飛ばす。
	ls.SkipEmptyLines()
	line, err := ls.nextLine()
	if err != nil {
		return nil, err
	}
	line = bytes.TrimLeftFunc(line, isSp)

	// セクション名の始まり
	i := bytes.IndexFunc(line, isAt)
	if i != 0 {
		return nil, ErrorNoSectionNamePrefix
	}
	line = line[i:]
	i = bytes.IndexFunc(line, isNonAt)
	if i == -1 {
		return nil, ErrorSectionNameIsEmpty
	}
	line = line[i:]

	// セクション名の終わり
	i = bytes.IndexFunc(line, isColon)
	if i == -1 { // コロンが見つからない
		return nil, ErrorNoSectionNameSuffix
	}
	name := bytes.TrimFunc(line[:i], isSp) // セクション名の両端の空白を切り捨てる。
	if len(name) == 0 {                    // セクション名が長さ0
		return nil, ErrorSectionNameIsEmpty
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
		return nil, err
	}
	return &Section{name, ps}, nil
}

func readCompoundValues(ls *lineScanner, head []byte) ([]Paragraph, error) {
	values := make([]Paragraph, 0, defaultSectionParagraphNum)
	pvalues := make([][]byte, 0, defaultParagraphLineNum)
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
				values = append(values, Paragraph{pvalues})
				pvalues = make([][]byte, 0, defaultParagraphLineNum)
			}
		}
		line, err = ls.nextLine()
	}

	if err != nil && err != io.EOF {
		return nil, err
	}

	if len(pvalues) > 0 {
		values = append(values, Paragraph{pvalues})
	}
	return values, nil
}

func ParseDocument(filename string, r io.Reader) (*Document, error) {
	ls := newLineScanner(r)

	secs := make([]Section, 0, 10)
	var sec *Section
	sec, err := processSection(ls)
	for err == nil {
		secs = append(secs, *sec)
		sec, err = processSection(ls)
	}
	if err != io.EOF {
		return nil, err
	}
	return &Document{filename, secs}, nil
}
