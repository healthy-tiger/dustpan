package dptxt

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// エラーメッセージ
var (
	ErrorIllegalCharBeforeSectionName = errors.New("セクション名の前に不正な文字があります。")
	ErrorNoSectionNamePrefix          = errors.New("セクション名プリフィックスがありません。")
	ErrorNoSectionNameSuffix          = errors.New("セクション名サフィックスがありません。")
	ErrorSectionNameIsEmpty           = errors.New("セクション名が空です。")
	ErrorUnexpectedText               = errors.New("予期しない入力文字列です。")
	ErrorInvalidDateFormat            = errors.New("日付の書式に誤りがあります。")
	ErrorNoMonthSpecified             = errors.New("日付の書式に誤りがあります。")
	ErrorNoDaySpecified               = errors.New("日付の書式に誤りがあります。")
	ErrorYearIsOutOfRange             = errors.New("日付の書式に誤りがあります。")
	ErrorMonthIsOutOfRange            = errors.New("日付の書式に誤りがあります。")
	ErrorDayIsOutOfRange              = errors.New("日付の書式に誤りがあります。")
	ErrorInvalidMonthSuffix           = errors.New("日付の書式に誤りがあります。")
	ErrorInvalidDaySuffix             = errors.New("日付の書式に誤りがあります。")
	ErrorUnknownDateSuffix            = errors.New("日付の書式に誤りがあります。")
	ErrorNoOpenParenthesis            = errors.New("日付を指定してください。")
	ErrorNoCloseParenthesis           = errors.New("日付を指定してください。")
	ErrorExtraTextAfterDate           = errors.New("日付を指定してください。")
)

type ParseError struct {
	Filename string
	Line     int
	Err      error
}

func (pe *ParseError) Unwrap() error {
	return pe.Err
}

func (pe *ParseError) Error() string {
	return pe.Filename + ":" + strconv.FormatInt(int64(pe.Line), 10) + ": " + pe.Err.Error()
}

func NewParseError(filename string, linenum int, err error) *ParseError {
	pe := new(ParseError)
	pe.Filename = filename
	pe.Line = linenum
	pe.Err = err
	return pe
}

const empty = ""

var emptyBytes []byte = make([]byte, 0, 0)

func isSp(r rune) bool {
	if r == ' ' || r == '　' || r == '\t' {
		return true
	}
	return false
}

func isAt(r rune) bool {
	if r == '@' || r == '＠' {
		return true
	}
	return false
}

func isColon(r rune) bool {
	if r == ':' || r == '：' {
		return true
	}
	return false
}

func isOpenParenthesis(r rune) bool {
	if r == '(' || r == '（' {
		return true
	}
	return false
}

func isCloseParenthesis(r rune) bool {
	if r == ')' || r == '）' {
		return true
	}
	return false
}

type Document struct {
	Filename string
	Sections map[string]*Section
	Error    error
}

type Section struct {
	Linenum     int
	Value       []*Paragraph
	peekedValue string
	Error       error
	Expired     bool
	Time        *time.Time
	Number      int64
}

type Paragraph struct {
	Linenum    int
	Value      [][]byte
	Error      error
	Time       *time.Time
	TimeSuffix []byte
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

func (s *Section) PeekString() string {
	if len(s.Value) == 0 {
		return empty
	}
	if len(s.Value[0].Value) == 0 {
		return empty
	}
	if len(s.peekedValue) == 0 {
		s.peekedValue = string(s.Value[0].Value[0])
	}
	return s.peekedValue
}

func (s *Section) PeekBytes() []byte {
	if len(s.Value) == 0 {
		return emptyBytes
	}
	if len(s.Value[0].Value) == 0 {
		return emptyBytes
	}
	return s.Value[0].Value[0]
}

type lineScanner struct {
	scanner  *bufio.Scanner
	lastline []byte
	unread   bool
	Filename string
	Linenum  int
}

func newLineScanner(filename string, r io.Reader) *lineScanner {
	return &lineScanner{bufio.NewScanner(r), nil, false, filename, 0}
}

func (ls *lineScanner) nextLine() ([]byte, error) {
	if ls.unread && ls.lastline != nil {
		ls.unread = false
		ls.Linenum++
		return ls.lastline, nil
	}
	if ls.scanner.Scan() {
		b := ls.scanner.Bytes()
		t := make([]byte, len(b), len(b))
		copy(t, b)
		ls.lastline = t
		ls.unread = false
		ls.Linenum++
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
	ls.Linenum--
	ls.unread = true
	return true
}

func (ls *lineScanner) NewParseError(err error) *ParseError {
	return NewParseError(ls.Filename, ls.Linenum, err)
}

func processSection(ls *lineScanner, sec *Section) (string, error) {
	// 空白行を読み飛ばす。
	ls.SkipEmptyLines()
	line, err := ls.nextLine()
	if err != nil {
		return empty, ls.NewParseError(err)
	}
	linenum := ls.Linenum
	line = bytes.TrimLeftFunc(line, isSp)

	// セクション名の始まり
	i, s := IndexFuncWithSize(line, isAt)
	if i != 0 {
		return empty, ls.NewParseError(ErrorNoSectionNamePrefix)
	}
	line = line[i+s:]

	// セクション名の終わり
	i, s = IndexFuncWithSize(line, isColon)
	if i == -1 { // コロンが見つからない
		return empty, ls.NewParseError(ErrorNoSectionNameSuffix)
	}
	name, err := normalizeText(line[:i]) // セクション名を正規化する。
	if err != nil {
		return empty, ls.NewParseError(err)
	}
	line = line[i+s:]

	// セクション本文の始まり
	var head []byte = nil
	line = bytes.TrimFunc(line, isSp)
	if len(line) > 0 {
		head = line
	}
	ps, err := readCompoundValues(ls, head)
	if err != nil {
		return empty, ls.NewParseError(err)
	}
	*sec = Section{Linenum: linenum, Value: ps, peekedValue: empty}
	return name, nil
}

func readCompoundValues(ls *lineScanner, head []byte) ([]*Paragraph, error) {
	values := make([]*Paragraph, 0)
	pvalues := make([][]byte, 0)
	var linenum int
	if head != nil {
		linenum = ls.Linenum
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
				if len(pvalues) == 0 {
					linenum = ls.Linenum
				}
				pvalues = append(pvalues, line)
			}
		} else {
			if len(pvalues) > 0 {
				values = append(values, &Paragraph{Linenum: linenum, Value: pvalues})
				pvalues = make([][]byte, 0)
			}
		}
		line, err = ls.nextLine()
	}

	// io.EOFはファイルの末尾なのでエラー扱いにしない。
	if err != nil && err != io.EOF {
		return nil, ls.NewParseError(err)
	}

	if len(pvalues) > 0 {
		values = append(values, &Paragraph{Linenum: linenum, Value: pvalues})
	}
	return values, nil
}

func ParseDocument(filename string, r io.Reader, doc *Document) error {
	ls := newLineScanner(filename, r)

	secs := make(map[string]*Section)
	var sec *Section = new(Section)
	name, err := processSection(ls, sec)
	for err == nil {
		secs[name] = sec
		sec = new(Section)
		name, err = processSection(ls, sec)
	}
	if !errors.Is(err, io.EOF) {
		log.Println(err)
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

func IndexFuncWithSize(b []byte, f func(r rune) bool) (int, int) {
	i := 0
	for len(b) > 0 {
		r, s := utf8.DecodeRune(b)
		if f(r) {
			return i, s
		}
		i += s
		b = b[s:]
	}
	return -1, 0
}

func LastIndexFuncWithSize(b []byte, f func(r rune) bool) (int, int) {
	i := 0
	index := -1
	size := 0
	for len(b) > 0 {
		r, s := utf8.DecodeRune(b)
		if f(r) {
			index = i
			size = s
		}
		i += s
		b = b[s:]
	}
	return index, size
}

func DecodeSingleDigit(b []byte) (rune, int, int) {
	r, s := utf8.DecodeRune(b)
	switch r {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return r, int(r - '0'), s
	case '０', '１', '２', '３', '４', '５', '６', '７', '８', '９':
		return r, int(r - '０'), s
	default:
		return r, -1, s
	}
}

func DecodeDigit(b []byte) ([]byte, int, rune, int) {
	var (
		r rune
		s int
		d int
		v int = 0
		n int = 0
	)
	for len(b) > 0 {
		r, d, s = DecodeSingleDigit(b)
		if d >= 0 {
			v = v*10 + d
			b = b[s:]
			n++
		} else {
			break
		}
	}
	return b, v, r, n
}

// 年のサフィックスから月のサフィックスへのマップ
var year2monthSuffix map[rune]rune = map[rune]rune{
	'年': '月',
	'/': '/',
	'／': '/',
	// ハイフンマイナス
	'\u002D': '-',
	'\uFE63': '-',
	'\uFF0D': '-',
	// ハイフン
	'\u2010': '-',
	'\u2011': '-',
	'\u2043': '-',
	// マイナス
	'\u02D7': '-',
	'\u2212': '-',
	'\u29FF': '-',
	'\u2796': '-',
	// ダッシュ
	'\u2012': '-',
	'\u2013': '-',
	'\u2014': '-',
	'\u2015': '-',
	'\u2E3A': '-',
	'\u2E3B': '-',
	'\uFE58': '-',
	// 罫線
	'\u2500': '-',
	'\u2501': '-',
	'\u2574': '-',
	'\u2576': '-',
	'\u2578': '-',
	'\u257A': '-',
	'\u257C': '-',
	'\u257E': '-',
	'.':      '.',
	'．':      '.',
}

// 月のサフィックスを正規化するためのマップ
var monthSuffixes map[rune]rune = map[rune]rune{
	'月': '月',
	'/': '/',
	'／': '/',
	// ハイフンマイナス
	'\u002D': '-',
	'\uFE63': '-',
	'\uFF0D': '-',
	// ハイフン
	'\u2010': '-',
	'\u2011': '-',
	'\u2043': '-',
	// マイナス
	'\u02D7': '-',
	'\u2212': '-',
	'\u29FF': '-',
	'\u2796': '-',
	// ダッシュ
	'\u2012': '-',
	'\u2013': '-',
	'\u2014': '-',
	'\u2015': '-',
	'\u2E3A': '-',
	'\u2E3B': '-',
	'\uFE58': '-',
	// 罫線
	'\u2500': '-',
	'\u2501': '-',
	'\u2574': '-',
	'\u2576': '-',
	'\u2578': '-',
	'\u257A': '-',
	'\u257C': '-',
	'\u257E': '-',
	'.':      '.',
	'．':      '.',
}

func ParseDate(b []byte) (int, int, int, []byte, error) {
	var (
		year, month, day       int = 0, 0, 0
		monthsuffix, daysuffix rune
		r                      rune
		n, s                   int
	)

	b = bytes.TrimLeftFunc(b, isSp)

	b, year, r, n = DecodeDigit(b)
	if r == utf8.RuneError {
		return year, month, day, nil, ErrorInvalidDateFormat
	} else if n != 4 {
		return year, month, day, nil, ErrorYearIsOutOfRange
	}

	b = bytes.TrimLeftFunc(b, isSp)

	// 年のサフィックスをデコード
	r, s = utf8.DecodeRune(b)
	if r == utf8.RuneError {
		return year, month, day, nil, ErrorInvalidDateFormat
	}
	monthsuffix, ok := year2monthSuffix[r]
	if !ok {
		return year, month, day, nil, ErrorInvalidDateFormat
	}
	if r == '年' {
		daysuffix = '日'
	} else {
		daysuffix = 0
	}
	b = b[s:]

	b = bytes.TrimLeftFunc(b, isSp)

	b, month, r, n = DecodeDigit(b)
	if r == utf8.RuneError {
		return year, month, day, nil, ErrorInvalidDateFormat
	} else if n == 0 {
		return year, month, day, nil, ErrorNoMonthSpecified
	} else if n > 2 {
		return year, month, day, nil, ErrorMonthIsOutOfRange
	}

	b = bytes.TrimLeftFunc(b, isSp)

	// 月のサフィックスをデコード
	r, s = utf8.DecodeRune(b)
	if r == utf8.RuneError {
		return year, month, day, nil, ErrorInvalidDateFormat
	}
	if ms, ok := monthSuffixes[r]; !ok || ms != monthsuffix {
		return year, month, day, nil, ErrorInvalidMonthSuffix
	}
	b = b[s:]

	b = bytes.TrimLeftFunc(b, isSp)

	b, day, r, n = DecodeDigit(b)
	if r == utf8.RuneError && len(b) > 0 { // 終端まで逹っしていないのにエラー
		return year, month, day, nil, ErrorInvalidDateFormat
	} else if n == 0 {
		return year, month, day, nil, ErrorNoDaySpecified
	} else if n > 2 {
		return year, month, day, nil, ErrorDayIsOutOfRange
	}

	// 日のサフィックスをデコード(ある場合のみ)
	if daysuffix != 0 {
		b = bytes.TrimLeftFunc(b, isSp)

		r, s = utf8.DecodeRune(b)
		if r == utf8.RuneError {
			return year, month, day, nil, ErrorInvalidDateFormat
		}
		if r != daysuffix {
			return year, month, day, nil, ErrorInvalidDaySuffix
		}
		b = b[s:] // 日のサフィックスを読み飛す
		r, s = utf8.DecodeRune(b)
	}

	// 日付けの直後の文字がないか、空白でなければエラー
	if len(b) == 0 || isSp(r) {
		return year, month, day, bytes.TrimFunc(b, isSp), nil
	}
	return year, month, day, nil, ErrorUnknownDateSuffix
}

// 日付より前の部分と、カッコの中の日付より後ろの部分も返す。
func ParseLogDate(b []byte) (int, int, int, []byte, []byte, error) {
	var (
		year, month, day int
		i, s             int
		err              error
	)
	i, s = LastIndexFuncWithSize(b, isOpenParenthesis)
	if i < 0 {
		return year, month, day, nil, nil, ErrorNoOpenParenthesis
	}
	pre := b[:i] // 日付けよりも前の部分
	b = b[i+s:]
	// 最後の左括弧から一番近い右括弧までの間を日付が入っていると想定してパースする。
	i, s = IndexFuncWithSize(b, isCloseParenthesis)
	if i < 0 {
		return year, month, day, pre, nil, ErrorNoCloseParenthesis
	}
	// 閉じカッコの後に文字が続く場合は、日付けとみなさない。
	if len(bytes.TrimLeftFunc(b[i+s:], isSp)) > 0 {
		return year, month, day, pre, nil, ErrorExtraTextAfterDate
	}
	year, month, day, post, err := ParseDate(b[:i])
	return year, month, day, pre, post, err
}
