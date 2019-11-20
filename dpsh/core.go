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
	ErrorInvalidDate = errors.New("無効な日付")
)

type DustpanConfig struct {
	SrcPath     []string     `json:"src"`
	Html        HtmlConfig   `json:"html"`
	Csv         CsvConfig    `json:"csv"`
	OrderBy     []SortConfig `json:"order"`
	Columns     []string     `json:"columns"`
	DateColumns []string     `json:"dates"`
}

type CsvConfig struct {
	DstPath    string `json:"dst"`
	AddHeading bool   `json:"heading"`
}

type HtmlConfig struct {
	DstPath string `json:"dst"`
	CssPath string `json:"css"`
	JsPath  string `json:"js"`
	Title   string `json:"title"`
}

type SortConfig struct {
	Name       string `json:"name"`
	Numerical  bool   `json:"numerical"`  // falseなら辞書式
	Descending bool   `json:"descending"` // falseなら降順
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

	return nil
}

func LoadAllFiles(basepath string, paths []string) []*dptxt.Document {
	docs := make([]*dptxt.Document, 0)
	for _, p := range paths {
		ap := filepath.Clean(filepath.Join(basepath, p))
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
	if len(config.OrderBy) == 0 {
		return
	}

	sort.Slice(docs, func(i, j int) bool {
		a := docs[i]
		b := docs[j]
		for _, c := range config.OrderBy {
			var r int64
			as := a.Sections[c.Name]
			bs := b.Sections[c.Name]
			if c.Numerical {
				av := int64(0)
				bv := int64(0)
				if as != nil {
					av = bytesToInt64(as.PeekBytes())
				}
				if bs != nil {
					bv = bytesToInt64(bs.PeekBytes())
				}
				r = av - bv
			} else {
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
			}
			if r != 0 {
				return (r < 0) != c.Descending
			}
		}
		return false
	})
}

// 日付の妥当性をチェックする。time.Date()を使った結果に対して、日付けが正規化されていないことを確認する。
func validateDate(year, month, day int) bool {
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	if t.Year() == year && t.Month() == time.Month(month) && t.Day() == day {
		return true
	}
	return false
}

func dateCheck(config *DustpanConfig, docs []*dptxt.Document) {
	if config.DateColumns == nil || len(config.DateColumns) == 0 {
		return
	}

	for _, d := range docs {
		for _, dc := range config.DateColumns {
			c := d.Sections[dc]
			if c != nil {
				pb := c.PeekBytes()
				year, month, day, err := dptxt.ParseDate(pb)
				if err != nil {
					c.Error = err
					log.Println(err, string(pb), year, month, day)
				} else if !validateDate(year, month, day) {
					c.Error = ErrorInvalidDate
					log.Println(ErrorInvalidDate, string(pb), year, month, day)
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
		r, d, s := dptxt.DecodeDigit(b)
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
