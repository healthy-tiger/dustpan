package dpsh

import (
	"bufio"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"time"

	"github.com/healthy-tiger/dustpan/dptxt"
)

var br []byte = []byte("<br>")

var styleOpen = []byte(`<style type="text/css">`)
var styleClose = []byte("</style>")

var scriptOpen = []byte(`<script>`)
var scriptClose = []byte("</script>")

var divErrFmt string = `<div class="dp-err" data-msg="%v"></div>`
var divExpire []byte = []byte(`<div class="dp-expired"></div>`)
var divDateFmt string = `<div class="dp-date" data-year="%v" data-month="%v" data-day="%v"></div>`
var divDateWithSuffixFmt string = `<div class="dp-date" data-year="%v" data-month="%v" data-day="%v" data-suffix="%v"></div>`
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

var defaultstyle []byte = []byte(`body{background-color:#fff}body,html{padding:0;margin:0}body{font-family:Meiryo UI;font-size:9pt}.dp-heading{font-size:2em;margin:10pt;display:flex}.dp-heading>.dp-title{flex:initial}.dp-heading>.dp-update{font-size:.5em;flex:auto;text-align:right}.dp-heading>.dp-title:after{content:attr(data-title)}.dp-heading>.dp-update:after{content:attr(data-date) " "attr(date-time) " 更新"}.dp-t .dp-h{width:100%;font-weight:700}.dp-t,.dp-t .dp-b{width:100%}.dp-t .dp-r{width:100%;display:flex;justify-content:stretch;flex-wrap:nowrap;flex-direction:row;align-items:stretch}.dp-t .dp-r>.dp-c{flex-shrink:0;padding:3pt}.dp-t>.dp-b>.dp-r:nth-child(n+2){border-style:solid;border-color:#999;border-width:1px 0 0}.dp-t .dp-r>.dp-c:nth-child(n+2){border-style:solid;border-color:#999;border-width:0 0 0 1px}.dp-t .dp-h .dp-r{white-space:nowrap;vertical-align:bottom;text-align:center;border-bottom-width:3px;border-bottom-style:double;border-bottom-color:#999}.dp-t>.dp-b>.dp-r>.dp-c{vertical-align:top}.dp-t>.dp-b>.dp-r>.dp-c:empty{background-color:#eee;text-align:center}.dp-t .dp-b .dp-r .dp-c:empty:before{content:"?"}.dp-t>.dp-b>.dp-r>.dp-c .dp-err{display:inline-block;background-color:red;color:#fff;font-weight:700;font-size:.8em;padding:.1em}.dp-t>.dp-b>.dp-r>.dp-c .dp-err:before{content:"エラー："}.dp-t>.dp-b>.dp-r>.dp-c .dp-err:after{content:attr(data-msg)}.dp-t>.dp-b>.dp-r>.dp-c>.dp-date{text-align:center}.dp-t>.dp-b>.dp-r>.dp-c>.dp-date.dp-expired{color:red;font-weight:700}.dp-t>.dp-b>.dp-r>.dp-c>.dp-date.dp-expired:after,.dp-t>.dp-b>.dp-r>.dp-c>.dp-date:after{content:attr(data-year) "/"attr(data-month) "/"attr(data-day)}.dp-t>.dp-b>.dp-r>.dp-c .dp-p{padding-top:1.5em}.dp-t>.dp-b>.dp-r>.dp-c .dp-p:first-child{padding-top:0}.dp-t>.dp-b>.dp-r>.dp-c .dp-p:last-child{padding-bottom:0}.dp-t>.dp-b>.dp-r>.dp-c .dp-p>.dp-date{display:inline;margin-left:.5em}.dp-t>.dp-b>.dp-r>.dp-c .dp-p>.dp-date:after{content:"("attr(data-year) "/"attr(data-month) "/"attr(data-day) ")"}.dp-t>.dp-b>.dp-r>.dp-c .dp-p>.dp-date[data-suffix]:after{content:"("attr(data-year) "/"attr(data-month) "/"attr(data-day) " "attr(data-suffix) ")"}@media print{body,html{margin:0;padding:0}.dp-heading{display:none}.dp-t{font-size:7pt;border:1px solid #999;box-sizing:border-box}.dp-t .dp-h{break-inside:avoid}.dp-t .dp-b .dp-r{break-inside:auto}.dp-t .dp-b .dp-r .dp-c .dp-p{break-inside:avoid}.dp-t .dp-b .dp-r .dp-c:empty{background-color:transparent}.dp-t .dp-b .dp-r .dp-c .dp-err{display:none}}`)
var defaultColumnWithWidth string = ".dp-c[data-section=\"%s\"]{flex-grow:0;flex-basis:%s;width:%s;}"
var defaultColumnWithoutWidth string = ".dp-c[data-section=\"%s\"]{flex-grow:1;width:0px;}"

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

func byteIsNilOrEmpty(b []byte) bool {
	if b == nil || len(b) == 0 {
		return true
	}
	return false
}

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
		if byteIsNilOrEmpty(para.TimeSuffix) {
			_, err = w.WriteString(fmt.Sprintf(divDateFmt, year, int(month), day))
		} else {
			_, err = w.WriteString(fmt.Sprintf(divDateWithSuffixFmt, year, int(month), day, html.EscapeString(string(para.TimeSuffix))))
		}
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
		if err == nil {
			_, err = w.Write(styleOpen)
		}
		if err == nil {
			_, err = w.Write(defaultstyle)
		}
		for _, cd := range config.ColumnDefs {
			if cd.Width == "" {
				w.Write([]byte(fmt.Sprintf(defaultColumnWithoutWidth, cd.Name)))
			} else {
				w.Write([]byte(fmt.Sprintf(defaultColumnWithWidth, cd.Name, cd.Width, cd.Width)))
			}
		}
		if err == nil {
			_, err = w.Write(styleClose)
		}
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
	for _, cname := range config.Html.DisplayColumns {
		tdOpen := []byte(fmt.Sprintf(tdOpenFmt, cname))

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
