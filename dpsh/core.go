package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/healthy-tiger/dustpan/dptxt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var sepEmpty = []byte("")
var sepComma = []byte(",")
var sepNewline = []byte("\n")
var sep2Newline = []byte("\n\n")
var sepDq = []byte("\"")

var (
	ErrorInvalidDate       = errors.New("無効な日付")
	ErrorNoColumnName      = errors.New("カラム名が未指定")
	ErrorNoColumnType      = errors.New("カラム型が未指定")
	ErrorUndefinedColumn   = errors.New("未定義のカラム")
	ErrorUnknownColumnType = errors.New("未知のカラム型")
)

type DustpanConfig struct {
	SrcPath    []string       `json:"src"`
	Html       HtmlConfig     `json:"html"`
	Csv        CsvConfig      `json:"csv"`
	ColumnDefs []ColumnConfig `json:"columns"`
	SortOrder  []SortConfig   `json:"order"`
}

const (
	ColumnTypeText     = "text"
	ColumnTypeNumber   = "number"
	ColumnTypeDate     = "date"
	ColumnTypeDeadline = "deadline"
	ColumnTypeLog      = "log"
)

type ColumnConfig struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type CsvConfig struct {
	DstPath    string `json:"dst"`
	AddHeading bool   `json:"heading"`
}

type HtmlConfig struct {
	DstPath        string   `json:"dst"`
	CssPath        string   `json:"css"`
	JsPath         string   `json:"js"`
	Title          string   `json:"title"`
	DisplayColumns []string `json:"display"`
}

type SortConfig struct {
	Name       string `json:"name"`
	Descending bool   `json:"descending"` // falseなら降順
}

func validateColumnConfig(cc *ColumnConfig) error {
	if len(cc.Name) == 0 {
		return ErrorNoColumnName
	}
	if len(cc.Type) == 0 {
		return ErrorNoColumnType
	}
	switch strings.ToLower(cc.Type) {
	case ColumnTypeText, ColumnTypeNumber, ColumnTypeDate, ColumnTypeDeadline:
		return nil
	default:
		return ErrorUnknownColumnType
	}
}

func validateSortConfig(sc *SortConfig) error {
	if len(sc.Name) == 0 {
		return ErrorNoColumnName
	}
	return nil
}

func (config *DustpanConfig) GetColumnDef(name string) *ColumnConfig {
	for i, _ := range config.ColumnDefs {
		if config.ColumnDefs[i].Name == name {
			return &config.ColumnDefs[i]
		}
	}
	return nil
}

func LoadConfig(filename string, config *DustpanConfig) error {
	f, err := os.Open(filename)
	if f == nil {
		return err
	}
	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, &config)
	if err != nil {
		return err
	}

	for _, cc := range config.ColumnDefs {
		if err = validateColumnConfig(&cc); err != nil {
			log.Fatal("columns:", err)
		}
	}

	for _, sc := range config.SortOrder {
		if err = validateSortConfig(&sc); err != nil {
			log.Fatal("order:", err)
		}
		cc := config.GetColumnDef(sc.Name)
		if cc == nil {
			log.Fatal("order:", ErrorUndefinedColumn)
		}
	}
	return nil
}

func normalizePath(basepath string, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(basepath, path)
	}
	return filepath.Clean(path)
}

func LoadAllFiles(basepath string, paths []string) []*dptxt.Document {
	docs := make([]*dptxt.Document, 0)
	for _, p := range paths {
		ap := normalizePath(basepath, p)
		gp, err := filepath.Glob(ap)
		if err != nil {
			log.Println(ap, err)
		} else {
			for _, g := range gp {
				var doc *dptxt.Document = new(dptxt.Document)
				err := LoadFile(g, doc)
				if err != nil {
					log.Println(g, err)
				} else {
					docs = append(docs, doc)
				}
			}
		}
	}
	return docs
}

func LoadFile(filename string, doc *dptxt.Document) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	err = dptxt.ParseDocument(filename, f, doc)
	if err != nil {
		return err
	}
	return nil
}

func sortDocs(config *DustpanConfig, docs []*dptxt.Document) {
	if len(config.SortOrder) == 0 {
		return
	}

	cdefs := make([]*ColumnConfig, len(config.SortOrder))
	for si, sc := range config.SortOrder {
		cdefs[si] = config.GetColumnDef(sc.Name)
	}

	sort.Slice(docs, func(i, j int) bool {
		a := docs[i]
		b := docs[j]
		for si, c := range config.SortOrder {
			var r int64
			as := a.Sections[c.Name]
			bs := b.Sections[c.Name]
			cd := cdefs[si]
			if cd.Type == ColumnTypeText || cd.Type == ColumnTypeLog {
				// 対応するセクションがなければ空文字列として扱う。
				av := ""
				bv := ""
				if as != nil {
					av = as.PeekString()
				}
				if bs != nil {
					bv = bs.PeekString()
				}
				r = int64(strings.Compare(av, bv))
			} else {
				av := int64(0)
				bv := int64(0)
				if as != nil {
					av = bytesToInt64(as.PeekBytes())
				}
				if bs != nil {
					bv = bytesToInt64(bs.PeekBytes())
				}
				r = av - bv
			}
			if r != 0 {
				return (r < 0) != c.Descending
			}
		}
		return false
	})
}

func dateCheck(config *DustpanConfig, docs []*dptxt.Document) {
	if config.ColumnDefs == nil || len(config.ColumnDefs) == 0 {
		return
	}

	now := time.Now()

	for _, d := range docs {
		for _, cd := range config.ColumnDefs {
			c := d.Sections[cd.Name]
			if c != nil && (cd.Type == ColumnTypeDate || cd.Type == ColumnTypeDeadline) {
				pb := c.PeekBytes()
				year, month, day, err := dptxt.ParseDate(pb)
				if err != nil {
					c.Error = err
					log.Println(err, string(pb), year, month, day)
				} else {
					// 日付の妥当性をチェックする。time.Date()を使った結果に対して、日付けが正規化されていないことを確認する。
					t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
					if t.Year() == year && t.Month() == time.Month(month) && t.Day() == day {
						// 有効期限型で値が現在日時より前なら有効期限切れのフラグを立てる。
						if cd.Type == ColumnTypeDeadline && t.Before(now) {
							c.Expired = true
						}
					} else {
						c.Error = ErrorInvalidDate
						log.Println(ErrorInvalidDate, string(pb), year, month, day)
					}
				}
			}
		}
	}
}

func DoMain(configpath string) {
	configname, err := filepath.Abs(configpath)
	if err != nil {
		log.Fatal(configpath, err)
	}

	var config DustpanConfig
	err = LoadConfig(configname, &config)
	if err != nil {
		log.Fatal(configname, err)
	}

	basepath := filepath.Dir(configname)

	docs := LoadAllFiles(basepath, config.SrcPath)

	dateCheck(&config, docs)
	sortDocs(&config, docs)

	err = WriteCsv(basepath, &config, docs)
	if err != nil {
		log.Println("csv", err)
	}
	err = WriteHtml(basepath, &config, docs)
	if err != nil {
		log.Println("html", err)
	}
}

var nbyteToInt64 int

func init() {
	nbyteToInt64 = len(strconv.FormatInt(math.MaxInt64, 10)) - 1
}

func bytesToInt64(b []byte) int64 {
	n := 0
	var v int64 = 0

	for len(b) > 0 && n < nbyteToInt64 {
		r, d, s := dptxt.DecodeSingleDigit(b)
		if r == utf8.RuneError {
			// bの長さをチェックしているので、s==0にはならない。
			return math.MaxInt64
		}
		if d >= 0 {
			v = v*10 + int64(d)
		}
		b = b[s:]
		n++
	}
	return v
}

func numericCompare(a, b []byte) int {
	return int(bytesToInt64(a) - bytesToInt64(b))
}

const tempfile_template = "_dustpan_%s.*.tmp"

func openTempFile(filetype string) (*os.File, error) {
	return ioutil.TempFile("", fmt.Sprintf(tempfile_template, filetype))
}

func closeTempFile(dstname string, tmpfile *os.File, lasterr error) {
	// とりあえず閉じて
	tmpfile.Close()
	// エラーがあればtmpfileを削除する
	if lasterr != nil {
		os.Remove(tmpfile.Name())
	} else {
		// エラーがなければ、出力先ファイルにリネームして、パーミッションを変更する。
		err := os.Rename(tmpfile.Name(), dstname)
		if err != nil {
			os.Remove(tmpfile.Name())
			log.Println(err)
		} else {
			err = os.Chmod(dstname, 0644)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
