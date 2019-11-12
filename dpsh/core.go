package main

import (
	"encoding/json"
	"errors"
	"github.com/healthy-tiger/dustpan/dptxt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const csvExt = ".csv"
const htmlExt = ".html"
const htmExt = ".htm"

var sepEmpty = []byte("")
var sepComma = []byte(",")
var sepNewline = []byte("\n")
var sep2Newline = []byte("\n\n")
var sepDq = []byte("\"")

type DustpanConfig struct {
	SrcPath    []string     `json:"src"`
	DstPath    string       `json:"dst"`
	AddHeading bool         `json:"heading"`
	CssPath    string       `json:"css"`
	OrderBy    []SortConfig `json:"order"`
	Columns    []string     `json:"columns"`
}

type SortConfig struct {
	Name  string `json:"name"`
	Order bool   `json:"order"` // trueなら昇順
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

func LoadAllFiles(paths []string) []*dptxt.Document {
	docs := make([]*dptxt.Document, 0)
	for _, p := range paths {
		ap, err := filepath.Abs(p)
		if err != nil {
			log.Println(p, err)
		} else {
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

func SortDocs(config DustpanConfig, docs []*dptxt.Document) {
	if len(config.OrderBy) == 0 {
		return
	}

	sort.Slice(docs, func(i, j int) bool {
		a := docs[i]
		b := docs[j]
		for _, c := range config.OrderBy {
			// 対応するセクションがなければ空文字列として扱う。
			av := ""
			bv := ""
			as, ok := a.Sections[c.Name]
			if ok {
				av = as.PeekValue()
			}
			bs, ok := b.Sections[c.Name]
			if ok {
				bv = bs.PeekValue()
			}
			r := strings.Compare(av, bv)
			if r != 0 {
				return (r < 0) == c.Order
			}
		}
		return false
	})
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

	dstname, err := filepath.Abs(config.DstPath)
	if err != nil {
		log.Fatal(config.DstPath, err)
	}

	docs := LoadAllFiles(config.SrcPath)
	SortDocs(config, docs)

	tmpfile, err := ioutil.TempFile("", "_dustpan.*.tmp")
	if err != nil {
		log.Fatal(err)
	}
	// ファイルの後始末
	defer (func() {
		// とりあえず閉じて
		tmpfile.Close()
		// エラーがあればtmpfileを削除する
		if err != nil {
			os.Remove(tmpfile.Name())
		} else {
			// エラーがなければ、出力先ファイルにリネームして、パーミッションを変更する。
			err = os.Rename(tmpfile.Name(), dstname)
			if err != nil {
				log.Fatal(err)
			}
			err = os.Chmod(dstname, 0644)
			if err != nil {
				log.Fatal(err)
			}
		}
	})()

	// TODO ファイルの種類によって呼ぶ関数を変える。
	switch filepath.Ext(dstname) {
	case csvExt:
		err = WriteCsv(&config, docs, tmpfile)
	case htmlExt, htmExt:
	default:
		err = errors.New("Unsupported output filetype")
	}
	if err != nil {
		log.Fatal(err)
	}
}
