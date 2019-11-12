package main

import (
	"fmt"
	"os"
)

func usage() {
	fmt.Println(`Dustpan shell Ver ????
dpsh <config file>`)
}

func argError() {
	fmt.Println("Too many arguments.")
}

func main() {
	switch len(os.Args) {
	case 1:
		usage()
	case 2:
		DoMain(os.Args[1])
	default:
		argError()
		usage()
	}
}
