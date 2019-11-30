package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/healthy-tiger/dustpan/dptxt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
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
	ErrorMultipleValue     = errors.New("複数の値")
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
	ColumnTypeNumber   = "number" // 符号付き整数
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
	Descending bool   `json:"descending"` // trueなら降順
}

func validateColumnConfig(cc *ColumnConfig) error {
	if len(cc.Name) == 0 {
		return ErrorNoColumnName
	}
	if len(cc.Type) == 0 {
		return ErrorNoColumnType
	}
	switch strings.ToLower(cc.Type) {
	case ColumnTypeText, ColumnTypeNumber, ColumnTypeDate, ColumnTypeDeadline, ColumnTypeLog:
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

	now := time.Now()

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
			} else if cd.Type == ColumnTypeDate || cd.Type == ColumnTypeDeadline {
				// Timeメンバがnilの場合は現在時刻が入っているものとして扱う。
				at := &now
				if as != nil && as.Time != nil {
					at = as.Time
				}
				bt := &now
				if bs != nil && bs.Time != nil {
					bt = bs.Time
				}
				r = int64(at.Sub(*bt))
			} else if cd.Type == ColumnTypeNumber {
				av := int64(0)
				bv := int64(0)
				if as != nil {
					av = as.Number
				}
				if bs != nil {
					bv = bs.Number
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

func preprocessDoc(config *DustpanConfig, now *time.Time, doc *dptxt.Document) {
	for _, cd := range config.ColumnDefs {
		c := doc.Sections[cd.Name]
		if c == nil {
			continue
		}
		switch cd.Type {
		case ColumnTypeNumber:
 			if len(c.Value) > 1 || len(c.Value[0].Value) > 1 {
				c.Error = ErrorMultipleValue
			} else {
				num, err := strconv.ParseInt(c.PeekString(), 10, 64)
				if err != nil {
					c.Error = err
				} else {
					c.Number = num
				}
			}
		case ColumnTypeDate, ColumnTypeDeadline:
 			if len(c.Value) > 1 || len(c.Value[0].Value) > 1 {
				c.Error = ErrorMultipleValue
			} else {
				pb := c.PeekBytes()
				year, month, day, err := dptxt.ParseDate(pb)
				if err != nil {
					c.Error = err
					log.Println(err, string(pb), year, month, day)
				} else {
					// 日付の妥当性をチェックする。time.Date()を使った結果に対して、日付けが正規化されていないことを確認する。
					t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
					if t.Year() == year && t.Month() == time.Month(month) && t.Day() == day {
						c.Time = new(time.Time)
						*(c.Time) = t
						// 有効期限型で値が現在日時より前なら有効期限切れのフラグを立てる。
						if cd.Type == ColumnTypeDeadline && t.Before(*now) {
							c.Expired = true
						}
					} else {
						c.Error = ErrorInvalidDate
						log.Println(ErrorInvalidDate, string(pb), year, month, day)
					}
				}
			}
		case ColumnTypeLog:
			for _, p := range c.Value {
				nv := len(p.Value)
				if nv > 0 {
					lp := p.Value[nv-1]
					year, month, day, err := dptxt.ParseLogDate(p.Value[nv-1])
					if err != nil {
						p.Error = err
						log.Println(err, string(lp), year, month, day)
					} else {
						t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
						if t.Year() == year && t.Month() == time.Month(month) && t.Day() == day {
							p.Time = new(time.Time)
							*(p.Time) = t
						} else {
							p.Error = ErrorInvalidDate
							log.Println(ErrorInvalidDate, string(lp), year, month, day)
						}
					}
				}
			}
		}
	}
}

func preprocessAllDocs(config *DustpanConfig, docs []*dptxt.Document) {
	if config.ColumnDefs == nil || len(config.ColumnDefs) == 0 {
		return
	}

	now := time.Now()

	for _, d := range docs {
		preprocessDoc(config, &now, d)
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

	preprocessAllDocs(&config, docs)
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
