package main

import (
	"encoding/json"
	"github.com/healthy-tiger/dustpan/dptxt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type SectionConfig struct {
	Name  string `json:"name"`
	Order bool   `json:"order"` // trueなら昇順
}

func LoadConfig(filename string) ([]SectionConfig, error) {
	f, err := os.Open(filename)
	if f == nil {
		return nil, err
	}
	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var config []SectionConfig
	err = json.Unmarshal(buf, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func LoadAllFiles(paths []string) []*dptxt.Document {
	docs := make([]*dptxt.Document, 0)
	for _, a := range paths {
		ms, err := filepath.Glob(a)
		if err != nil {
			log.Println(a, err)
		} else {
			for _, m := range ms {
				p, err := filepath.Abs(m)
				if err != nil {
					log.Println(ms, err)
				} else {
					var doc *dptxt.Document = new(dptxt.Document)
					err := LoadFile(p, doc)
					if err != nil {
						log.Println(p, err)
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

func SortDocs(config []SectionConfig, docs []*dptxt.Document) {
	if len(config) == 0 {
		return
	}

	sort.Slice(docs, func(i, j int) bool {
		a := docs[i]
		b := docs[j]
		for _, c := range config {
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
