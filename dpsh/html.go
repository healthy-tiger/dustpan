package main

import (
	"bufio"
	"fmt"
	"github.com/healthy-tiger/dustpan/dptxt"
	"html"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"
)

var br []byte = []byte("<br>")

var styleOpen = []byte(`<style type="text/css">`)
var styleClose = []byte("</style>")

var scriptOpen = []byte(`<script>`)
var scriptClose = []byte("</script>")

var pOpen []byte = []byte("<p>")
var pClose []byte = []byte("</p>")

var tdOpenFmt string = `<div class="dp-cell c%d">`
var tdClose []byte = []byte("</div>")

var trOpenFmt string = `<div class="dp-row" data-filename="%s">`
var trOpen []byte = []byte(`<div class="dp-row">`)
var trClose []byte = []byte("</div>")

var theadOpen []byte = []byte(`<div class="dp-header">`)
var theadClose []byte = []byte("</div>")

var tbodyOpen []byte = []byte(`<div class="dp-body">`)
var tbodyClose []byte = []byte("</div>")

const defaultTitle = "Dustpan HTML"

var contentOpen1 string = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta http-equiv="X-UA-Compatible" content="IE=Edge" />
<title>%s</title>`

var contentOpen2 string = `
</head>
<body data-update="%d/%d/%d">
<div class="dp-table">`

var contentClose string = `</div>
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
		_, err = w.Write(v) // TODO vのエスケープ
		if err != nil {
			return err
		}
		sep = br
	}
	return nil
}

func htmlWriteSection(sec *dptxt.Section, tdOpen []byte, w *bufio.Writer) error {
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

func htmlWriteDocument(config *DustpanConfig, doc *dptxt.Document, tdOpenMap map[string][]byte, w *bufio.Writer) error {
	var err error
	_, err = w.WriteString(fmt.Sprintf(trOpenFmt, html.EscapeString(doc.Filename)))
	if err != nil {
		return err
	}
	for _, cname := range config.Columns {
		err = htmlWriteSection(doc.Sections[cname], tdOpenMap[cname], w)
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
		return err
	}

	// 一時ファイルの生成
	tmpfile, err := openTempFile("html")
	if err != nil {
		return err
	}
	// ファイルの後始末
	defer func() {
		closeTempFile(dstname, tmpfile, err)
	}()

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
			log.Println(config.Html.CssPath, err)
		} else {
			var cssbytes []byte
			cssbytes, err = ioutil.ReadFile(cssname)
			if err == nil {
				_, err = w.Write(styleOpen)
			}
			if err == nil {
				_, err = w.Write(cssbytes)
			}
			if err == nil {
				_, err = w.Write(styleClose)
			}
			if err != nil {
				log.Println(cssname, err)
			}
		}
	}

	if len(config.Html.JsPath) > 0 {
		jsname, err := filepath.Abs(config.Html.JsPath)
		if err != nil {
			log.Println(config.Html.JsPath, err)
		} else {
			var jsbytes []byte
			jsbytes, err = ioutil.ReadFile(jsname)
			if err == nil {
				_, err = w.Write(scriptOpen)
			}
			if err == nil {
				_, err = w.Write(jsbytes)
			}
			if err == nil {
				_, err = w.Write(scriptClose)
			}
			if err != nil {
				log.Println(jsname, err)
			}
		}
	}

	year, month, day := time.Now().Date()
	_, err = w.WriteString(fmt.Sprintf(contentOpen2, year, month, day))
	if err != nil {
		return err
	}

	// あとでドキュメントの出力に使うので、カラムの開始タグをキャッシュする。
	tdOpenMap := make(map[string][]byte)

	// theadを出力する。
	_, err = w.Write(theadOpen)
	if err != nil {
		return err
	}
	_, err = w.Write(trOpen)
	if err != nil {
		return err
	}
	for i, c := range config.Columns {
		tdOpen := []byte(fmt.Sprintf(tdOpenFmt, i))

		_, err = w.Write(tdOpen)
		if err != nil {
			return err
		}
		_, err = w.WriteString(c)
		if err != nil {
			return err
		}
		_, err = w.Write(tdClose)
		if err != nil {
			return err
		}

		tdOpenMap[c] = tdOpen // キャッシュに入れる。
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
		err = htmlWriteDocument(config, d, tdOpenMap, w)
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
