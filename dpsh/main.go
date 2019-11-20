package main

import (
	"flag"
)

func main() {
	var configpath string
	flag.StringVar(&configpath, "c", "config.json", "config file path")
	flag.Parse()

	DoMain(configpath)
}
