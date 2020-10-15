package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/healthy-tiger/dustpan/dpsh"
)

func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Dustpan Shell\nUsage:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = Usage

	var configpath string
	flag.StringVar(&configpath, "c", "config.json", "config file path")
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

	docs := dpsh.LoadAllFiles(basepath, config.SrcPath)

	dpsh.PreprocessAllDocs(&config, docs)
	dpsh.SortDocs(&config, docs)

	err = dpsh.WriteCsv(basepath, &config, docs)
	if err != nil {
		log.Println("csv", err)
	}
	err = dpsh.WriteHTML(basepath, &config, docs)
	if err != nil {
		log.Println("html", err)
	}
}
