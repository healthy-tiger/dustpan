package main

import (
	"bufio"
	"errors"
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
var divDateFmt string = `<div class="dp-date" data-year="%v" data-month="%v" data-day="%v"></div>`
var divDateExpiredFmt string = `<div class="dp-date dp-expired" data-year="%v" data-month="%v" data-day="%v"></div>`

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

var defaultstyle []byte = []byte(`
<style>
.dp-t { display: table; width: 100%; }
.dp-t .dp-h { display: table-header-group; }
.dp-t .dp-b { display: table-row-group; }
.dp-t .dp-r { display: table-row; }
.dp-t .dp-r .dp-c { display: table-cell;}
.dp-t .dp-b .dp-c:empty { background-color: #eee; }
.dp-t .dp-date:after { content: attr(data-year) "/" attr(data-month) "/" attr(data-day); }
.dp-t .dp-err { color: red; }
.dp-t .dp-err:after { content: "error: " attr(data-msg); }
.dp-t .dp-date.dp-expired:after { content: "expired"; }
</style>`)

var contentOpen2 string = `
</head>
<body>
<div class="dp-heading">
<div class="dp-title" data-title="%v"></div>
<div class="dp-update" data-date="%d/%d/%d" date-time="%d:%02d:%02d"></div>
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
	if para.Time != nil {
		year, month, day := para.Time.Date()
		_, err = w.WriteString(fmt.Sprintf(divDateFmt, year, int(month), day))
		if err != nil {
			return err
		}
	}
	if para.Error != nil {
		// para.ErrorはValueErrorの想定だけど、将来的に変更するかもしれないので、Unwrapする処理を入れておく。
		ierr := errors.Unwrap(para.Error)
		if ierr == nil {
			ierr = para.Error
		}
		_, err = w.WriteString(fmt.Sprintf(divErrFmt, html.EscapeString(ierr.Error())))
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
		if sec.Time != nil {
			year, month, day := sec.Time.Date()
			if sec.Expired {
				_, err = w.WriteString(fmt.Sprintf(divDateExpiredFmt, year, int(month), day))
			} else {
				_, err = w.WriteString(fmt.Sprintf(divDateFmt, year, int(month), day))
			}
			if err != nil {
				return err
			}
		} else {
			for _, p := range sec.Value {
				err = htmlWriteParagraph(p, w)
				if err != nil {
					return err
				}
			}
		}

		if sec.Error != nil {
			// para.ErrorはValueErrorの想定だけど、将来的に変更するかもしれないので、Unwrapする処理を入れておく。
			ierr := errors.Unwrap(sec.Error)
			if ierr == nil {
				ierr = sec.Error
			}
			_, err = w.WriteString(fmt.Sprintf(divErrFmt, html.EscapeString(ierr.Error())))
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
	} else { // CSSの指定がなければデフォルトのCSSを入れる。
		_, err = w.Write(defaultstyle)
		if err != nil {
			log.Println(err)
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

	now := time.Now()
	year, month, day := now.Date()
	hour, min, sec := now.Clock()
	_, err = w.WriteString(fmt.Sprintf(contentOpen2, html.EscapeString(title), year, month, day, hour, min, sec))
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
