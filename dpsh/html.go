package main

import (
	"bufio"
	"fmt"
	"github.com/healthy-tiger/dustpan/dptxt"
	"html"
	"io/ioutil"
	"log"
	"time"
)

var br []byte = []byte("<br>")

var styleOpen = []byte(`<style type="text/css">`)
var styleClose = []byte("</style>")

var scriptOpen = []byte(`<script>`)
var scriptClose = []byte("</script>")

var divErrFmt string = `<div class="dp-err" data-msg="%v"></div>`
var divExpire []byte = []byte(`<div class="dp-expired"></div>`)

var pOpen []byte = []byte(`<div class="dp-p">`)
var pClose []byte = []byte("</div>")

var tdOpenFmt string = `<div class="dp-c" data-section="%v">`
var tdClose []byte = []byte("</div>")

var trOpenFmt string = `<div class="dp-r" data-filename="%v">`
var trOpen []byte = []byte(`<div class="dp-r">`)
var trClose []byte = []byte("</div>")

var theadOpen []byte = []byte(`<div class="dp-h">`)
var theadClose []byte = []byte("</div>")

var tbodyOpen []byte = []byte(`<div class="dp-b">`)
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
<body>
<div class="dp-heading">
<div class="dp-title" data-title="%v"></div>
<div class="dp-update" data-update="%d/%d/%d"></div>
</div>
<div class="dp-t">`

var contentClose string = `</div>
</body>
</html>`

func htmlWriteParagraph(para *dptxt.Paragraph, w *bufio.Writer) error {
	_, err := w.Write(pOpen)
	if err != nil {
		return err
	}

	sep := sepEmpty
	for _, v := range para.Value {
		_, err = w.Write(sep)
		if err != nil {
			return err
		}
		_, err = w.WriteString(html.EscapeString(string(v)))
		if err != nil {
			return err
		}
		sep = br
	}
	if para.Error != nil {
		_, err = w.WriteString(fmt.Sprintf(divErrFmt, html.EscapeString(para.Error.Error())))
		if err != nil {
			return err
		}
	}

	_, err = w.Write(pClose)
	if err != nil {
		return err
	}

	return nil
}

func htmlWriteSection(sec *dptxt.Section, secname string, w *bufio.Writer) error {
	// secがnilでも開始タグと閉じタグは出力する。
	_, err := w.WriteString(fmt.Sprintf(tdOpenFmt, html.EscapeString(secname)))
	if err != nil {
		return err
	}

	if sec != nil {
		for _, p := range sec.Value {
			err = htmlWriteParagraph(p, w)
			if err != nil {
				return err
			}
		}

		if sec.Error != nil {
			_, err = w.WriteString(fmt.Sprintf(divErrFmt, html.EscapeString(sec.Error.Error())))
			if err != nil {
				return err
			}
		}

		if sec.Expired {
			_, err = w.Write(divExpire)
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
	_, err := w.WriteString(fmt.Sprintf(trOpenFmt, html.EscapeString(doc.Filename)))
	if err != nil {
		return err
	}
	for _, cname := range config.Html.DisplayColumns {
		err = htmlWriteSection(doc.Sections[cname], cname, w)
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

func WriteHtml(basepath string, config *DustpanConfig, docs []*dptxt.Document) error {
	if len(config.Html.DstPath) == 0 {
		return nil
	}
	dstname := normalizePath(basepath, config.Html.DstPath)

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
		cssname := normalizePath(basepath, config.Html.CssPath)
		cssbytes, err := ioutil.ReadFile(cssname)
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

	// JavaScriptの指定があれば読み込んで埋め込む。読み込みエラーがあっても中断はしない。
	if len(config.Html.JsPath) > 0 {
		jsname := normalizePath(basepath, config.Html.JsPath)
		jsbytes, err := ioutil.ReadFile(jsname)
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

	year, month, day := time.Now().Date()
	_, err = w.WriteString(fmt.Sprintf(contentOpen2, html.EscapeString(title), year, month, day))
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
	for i, cname := range config.Html.DisplayColumns {
		tdOpen := []byte(fmt.Sprintf(tdOpenFmt, i))

		_, err = w.Write(tdOpen)
		if err != nil {
			return err
		}
		_, err = w.WriteString(cname)
		if err != nil {
			return err
		}
		_, err = w.Write(tdClose)
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
