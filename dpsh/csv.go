package main

import (
	"bufio"
	"github.com/healthy-tiger/dustpan/dptxt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func csvWriteParagraph(para *dptxt.Paragraph, w *bufio.Writer) error {
	var err error
	sep := sepEmpty
	for _, v := range para.Value {
		_, err = w.Write(sep)
		if err != nil {
			return err
		}
		_, err = w.Write(v)
		if err != nil {
			return err
		}
		sep = sepNewline
	}
	return nil
}

func csvWriteSection(sec *dptxt.Section, w *bufio.Writer) error {
	_, err := w.Write(sepDq)
	if err != nil {
		return err
	}
	if sec != nil {
		sep := sepEmpty
		for _, p := range sec.Value {
			_, err = w.Write(sep)
			if err != nil {
				return err
			}
			err = csvWriteParagraph(p, w)
			if err != nil {
				return err
			}
			sep = sep2Newline
		}
	}
	_, err = w.Write(sepDq)
	if err != nil {
		return err
	}
	return nil
}

func csvWriteDocument(config *DustpanConfig, doc *dptxt.Document, w *bufio.Writer) error {
	var err error
	sep := sepEmpty
	for _, cname := range config.Columns {
		_, err = w.Write(sep)
		if err != nil {
			return err
		}
		err = csvWriteSection(doc.Sections[cname], w)
		if err != nil {
			return err
		}
		sep = sepComma
	}
	return nil
}

func WriteCsv(config *DustpanConfig, docs []*dptxt.Document) error {
	if len(config.Csv.DstPath) == 0 {
		return nil
	}
	dstname, err := filepath.Abs(config.Csv.DstPath)
	if err != nil {
		log.Fatal(config.Csv.DstPath, err)
	}

	tmpfile, err := ioutil.TempFile("", "_dustpan_csv.*.tmp")
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
	w := bufio.NewWriter(tmpfile)

	cols := make([]string, len(config.Columns))
	if config.Csv.AddHeading {
		for i, c := range config.Columns {
			cols[i] = "\"" + c + "\""
		}
		_, err = w.WriteString(strings.Join(cols, ","))
		if err != nil {
			return err
		}
		_, err = w.Write(sepNewline)
		if err != nil {
			return err
		}
	}

	for _, d := range docs {
		err = csvWriteDocument(config, d, w)
		if err != nil {
			return err
		}
		_, err = w.Write(sepNewline)
		if err != nil {
			return err
		}
	}

	w.Flush()
	return nil
}
