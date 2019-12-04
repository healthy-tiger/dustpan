package main

import (
	"flag"
	"fmt"
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

	dpsh.DoMain(configpath)
}
