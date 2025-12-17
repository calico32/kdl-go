package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/calico32/kdl-go"
)

// var debug = flag.Bool("d", false, "Enable debug output to stderr")
var s = flag.Bool("s", false, "Output as s-expression")

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file.kdl>\n", os.Args[0])
		os.Exit(1)
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		panic(err)
	}

	doc, err := kdl.Parse(f)
	if err != nil {
		fmt.Println(err)
		return
	}

	if *s {
		fmt.Println(kdl.PrintDocument(doc))
		return
	}

	err = kdl.Emit(doc, os.Stdout)
	if err != nil {
		fmt.Println(err)
		return
	}
}
