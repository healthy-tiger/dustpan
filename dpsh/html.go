package main

import (
	"bufio"
	"fmt"
	"github.com/healthy-tiger/dustpan/dptxt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var br []byte = []byte("<br>")

var styleOpen = []byte(`<style type="text/css">`)
var styleClose = []byte("</style>")

var pOpen []byte = []byte("<p>")
var pClose []byte = []byte("</p>")

var tdOpen []byte = []byte("<td>")
var tdClose []byte = []byte("</td>")

var trOpen []byte = []byte("<tr>")
var trClose []byte = []byte("</tr>")

var thOpen []byte = []byte("<th>")
var thClose []byte = []byte("</th>")

var theadOpen []byte = []byte("<thead>")
var theadClose []byte = []byte("</thead>")

var tbodyOpen []byte = []byte("<tbody>")
var tbodyClose []byte = []byte("</tbody>")

const defaultTitle = "Dustpan HTML"

var contentOpen1 string = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta http-equiv="X-UA-Compatible" content="IE=Edge" />
<title>%s</title>`

var contentOpen2 string = `
</head>
<body>
<table>`

var contentClose string = `</table>
</body>
</html>`

func htmlWriteParagraph(para *dptxt.Paragraph, w *bufio.Writer) error {
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
		sep = br
	}
	return nil
}

func htmlWriteSection(sec *dptxt.Section, w *bufio.Writer) error {
	_, err := w.Write(tdOpen)
	if err != nil {
		return err
	}
	// secがnilでも開始タグと閉じタグは出力する。
	if sec != nil {
		for _, p := range sec.Value {
			_, err = w.Write(pOpen)
			if err != nil {
				return err
			}
			err = htmlWriteParagraph(p, w)
			if err != nil {
				return err
			}
			_, err = w.Write(pClose)
			if err != nil {
				return err
			}
		}
	}
	_, err = w.Write(tdClose)
	if err != nil {
		return err
	}
	return nil
}

func htmlWriteDocument(config *DustpanConfig, doc *dptxt.Document, w *bufio.Writer) error {
	var err error
	_, err = w.Write(trOpen)
	if err != nil {
		return err
	}
	for _, cname := range config.Columns {
		err = htmlWriteSection(doc.Sections[cname], w)
		if err != nil {
			return err
		}
	}
	_, err = w.Write(trClose)
	if err != nil {
		return err
	}
	return nil
}

func WriteHtml(config *DustpanConfig, docs []*dptxt.Document) error {
	if len(config.Html.DstPath) == 0 {
		return nil
	}
	dstname, err := filepath.Abs(config.Html.DstPath)
	if err != nil {
		log.Fatal(config.Html.DstPath, err)
	}

	tmpfile, err := ioutil.TempFile("", "_dustpan_html.*.tmp")
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

	title := config.Html.Title
	if len(title) == 0 {
		title = defaultTitle
	}

	_, err = w.WriteString(fmt.Sprintf(contentOpen1, title))
	if err != nil {
		return err
	}

	// CSSの指定があれば読み込んで埋め込む。読み込みエラーがあっても中断はしない。
	if len(config.Html.CssPath) > 0 {
		cssname, err := filepath.Abs(config.Html.CssPath)
		if err != nil {
			log.Println(cssname, err)
		} else {
			cssbytes, err := ioutil.ReadFile(cssname)
			if err != nil {
				log.Println(cssname, err)
			} else {
				_, err = w.Write(styleOpen)
				if err != nil {
					return err
				}
				_, err = w.Write(cssbytes)
				if err != nil {
					return err
				}
				_, err = w.Write(styleClose)
				if err != nil {
					return err
				}
			}
		}
	}

	_, err = w.WriteString(contentOpen2)
	if err != nil {
		return err
	}

	// theadを出力する。
	_, err = w.Write(theadOpen)
	if err != nil {
		return err
	}
	_, err = w.Write(trOpen)
	if err != nil {
		return err
	}
	for _, c := range config.Columns {
		_, err = w.Write(thOpen)
		if err != nil {
			return err
		}
		_, err = w.WriteString(c)
		if err != nil {
			return err
		}
		_, err = w.Write(thClose)
		if err != nil {
			return err
		}
	}
	_, err = w.Write(trClose)
	if err != nil {
		return err
	}
	_, err = w.Write(theadClose)
	if err != nil {
		return err
	}

	_, err = w.Write(tbodyOpen)
	if err != nil {
		return err
	}
	for _, d := range docs {
		err = htmlWriteDocument(config, d, w)
		if err != nil {
			return err
		}
	}
	_, err = w.Write(tbodyClose)
	if err != nil {
		return err
	}

	_, err = w.WriteString(contentClose)
	if err != nil {
		return err
	}

	w.Flush()
	return nil
}
