package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
)

func main() {
	var configpath string
	flag.StringVar(&configpath, "config", "config.json", "config file path")

	flag.Parse()
	configname, err := filepath.Abs(configpath)
	if err != nil {
		log.Fatal(configpath, err)
	}

	config, err := LoadConfig(configname)
	if err != nil {
		log.Fatal(configname, err)
	}

	docs := LoadAllFiles(flag.Args())
	SortDocs(config, docs)

	for i, d := range docs {
		fmt.Println(i, d)
	}
}
