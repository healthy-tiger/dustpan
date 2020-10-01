package dpsh

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/healthy-tiger/dustpan/dptxt"
)

var jsonErrFmt string = `, "error":"%v"`
var jsonSecDateFmt string = `"date":{ "year":%d, "month":%d, "day":%d }`
var jsonSecDateExpiredFmt string = `"date":{ "year":%d, "month":%d, "day":%d, "expired":true }`
var jsonParaDateFmt string = `, "date":{ "year":%d, "month":%d, "day":%d }`
var jsonParaDateWithSuffixFmt string = `, "date":{ "year":%d, "month":%d, "day":%d, "suffix":"%v" }`
var jsonLastUpdateFmt string = `, "lastupdate":{ "year":%d, "month":%d, "day":%d, "hour":%d, "min":%d, "sec":%d }`

func jsonEscapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func jsonWriteParagraph(para *dptxt.Paragraph, w *bufio.Writer) error {
	_, err := w.WriteString(`{ "value":[`)
	if err != nil {
		return err
	}

	sep := sepEmpty
	for _, v := range para.Value {
		_, err = w.Write(sep)
		if err != nil {
			return err
		}
		_, err = w.WriteString(`"` + jsonEscapeString(string(v)) + `"`)
		if err != nil {
			return err
		}
		sep = sepComma
	}
	_, err = w.WriteString(`]`)
	if err != nil {
		return err
	}

	if para.Time != nil {
		year, month, day := para.Time.Date()
		if byteIsNilOrEmpty(para.TimeSuffix) {
			_, err = w.WriteString(fmt.Sprintf(jsonParaDateFmt, year, int(month), day))
		} else {
			_, err = w.WriteString(fmt.Sprintf(jsonParaDateWithSuffixFmt, year,
				int(month),
				day,
				jsonEscapeString(string(para.TimeSuffix))))
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
		_, err = w.WriteString(fmt.Sprintf(jsonErrFmt, jsonEscapeString(ierr.Error())))
		if err != nil {
			return err
		}
	}
	_, err = w.WriteString(`}`)
	if err != nil {
		return err
	}

	return nil
}

func jsonWriteSection(sec *dptxt.Section, secname string, w *bufio.Writer) error {
	// secがnilでも開始タグと閉じタグは出力する。
	_, err := w.WriteString(fmt.Sprintf(`"%v":`, jsonEscapeString(secname)))
	if err != nil {
		return err
	}

	if sec != nil && sec.Time == nil && len(sec.Value) == 1 && len(sec.Value[0].Value) == 1 && sec.Error == nil {
		_, err = w.WriteString(fmt.Sprintf(`"%v"`, jsonEscapeString(string(sec.Value[0].Value[0]))))
	} else {
		_, err = w.WriteString("{")
		if sec != nil {
			if sec.Time != nil {
				year, month, day := sec.Time.Date()
				if sec.Expired {
					_, err = w.WriteString(fmt.Sprintf(jsonSecDateExpiredFmt, year, int(month), day))
				} else {
					_, err = w.WriteString(fmt.Sprintf(jsonSecDateFmt, year, int(month), day))
				}
				if err != nil {
					return err
				}
			} else {
				w.WriteString(`"value":[`)
				sep := sepEmpty
				for _, p := range sec.Value {
					_, err = w.Write(sep)
					err = jsonWriteParagraph(p, w)
					if err != nil {
						return err
					}
					sep = sepComma
				}
				w.WriteString(`]`)
			}

			if sec.Error != nil {
				// para.ErrorはValueErrorの想定だけど、将来的に変更するかもしれないので、Unwrapする処理を入れておく。
				ierr := errors.Unwrap(sec.Error)
				if ierr == nil {
					ierr = sec.Error
				}
				_, err = w.WriteString(fmt.Sprintf(jsonErrFmt, jsonEscapeString(ierr.Error())))
				if err != nil {
					return err
				}
			}
		}

		_, err = w.WriteString(`}`)
		if err != nil {
			return err
		}
	}
	return nil
}

func jsonWriteDocument(config *DustpanConfig, doc *dptxt.Document, w *bufio.Writer) error {
	_, err := w.WriteString(fmt.Sprintf(`{"filename":"%v","sections":{`, jsonEscapeString(doc.Filename)))
	if err != nil {
		return err
	}
	sep := sepEmpty
	for _, cname := range config.HTML.DisplayColumns {
		_, err = w.Write(sep)
		err = jsonWriteSection(doc.Sections[cname], cname, w)
		if err != nil {
			return err
		}
		sep = sepComma
	}
	_, err = w.WriteString(`} }`)
	if err != nil {
		return err
	}
	return nil
}

func writeJSONTo(w *bufio.Writer, basepath string, config *DustpanConfig, docs []*dptxt.Document) error {
	if len(config.HTML.Header) > 0 {
		w.WriteString(config.HTML.Header)
	}

	title := config.HTML.Title
	if len(title) == 0 {
		title = defaultTitle
	}

	_, err := w.WriteString(fmt.Sprintf(`{"title":"%v"`, jsonEscapeString(title)))
	if err != nil {
		return err
	}

	now := time.Now()
	year, month, day := now.Date()
	hour, min, sec := now.Clock()
	_, err = w.WriteString(fmt.Sprintf(jsonLastUpdateFmt, year, month, day, hour, min, sec))
	if err != nil {
		return err
	}

	_, err = w.WriteString(`, "documents":[`)
	if err != nil {
		return err
	}
	sep := sepEmpty
	for _, d := range docs {
		_, err = w.Write(sep)
		err = jsonWriteDocument(config, d, w)
		if err != nil {
			return err
		}
		sep = sepComma
	}
	_, err = w.WriteString(`]`)
	if err != nil {
		return err
	}

	_, err = w.WriteString(`}`)
	if err != nil {
		return err
	}

	w.Flush()
	return nil
}

// WriteJSON 設定ファイルに従ってHTML出力を実行する。
func WriteJSON(basepath string, config *DustpanConfig, docs []*dptxt.Document) error {
	var w *bufio.Writer

	if len(config.HTML.DstPath) == 0 {
		// 出力先の指定がない場合は標準出力に出力する。
		w = bufio.NewWriter(os.Stdout)
	} else {
		dstname := normalizePath(basepath, config.HTML.DstPath)

		// 一時ファイルの生成
		tmpfile, err := openTempFile("json")
		if err != nil {
			return err
		}
		// ファイルの後始末
		defer func() {
			closeTempFile(dstname, tmpfile, err)
		}()

		w = bufio.NewWriter(tmpfile)
	}

	return writeJSONTo(w, basepath, config, docs)
}
