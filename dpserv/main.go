package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/healthy-tiger/dustpan/dpsh"
)

// Usage コマンドラインオプションのヘルプを表示
func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Dustpan Shell\nUsage:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = Usage

	var configpath string
	var addr string
	flag.StringVar(&configpath, "c", "config.json", "config file path")
	flag.StringVar(&addr, "a", ":8080", "listen address")
	flag.Parse()

	configname, err := filepath.Abs(configpath)
	if err != nil {
		log.Fatal(err)
	}

	var config dpsh.DustpanConfig
	err = dpsh.LoadConfig(configname, &config)
	if err != nil {
		log.Fatal(err)
	}

	basepath := filepath.Dir(configname)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		docs := dpsh.LoadAllFiles(basepath, config.SrcPath)

		dpsh.PreprocessAllDocs(&config, docs)
		dpsh.SortDocs(&config, docs)

		err = dpsh.WriteHTMLTo(w, basepath, &config, docs)
		if err != nil {
			log.Println("html", err)
		}
	})

	log.Fatal(http.ListenAndServe(addr, nil))
}
