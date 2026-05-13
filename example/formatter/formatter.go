// Demonstrates formatting and emitting KDL with various options.

package main

import (
	"fmt"
	"strings"

	"github.com/calico32/kdl-go"
)

const messy = `
server   host="localhost"   port=8080   tls=#true
database{driver "postgres"
host "db.internal"
port 5432
pool min=2 max=10}
tags "web" "api" "v2"
`

func main() {
	doc, err := kdl.Parse(strings.NewReader(messy))
	if err != nil {
		panic(err)
	}

	fmt.Println("=== formatted (default) ===")
	s, err := kdl.FormatToString(doc)
	if err != nil {
		panic(err)
	}
	fmt.Print(s)

	fmt.Println("=== formatted (2-space indent, sorted props) ===")
	s, err = kdl.FormatToString(doc,
		kdl.WithFormatIndentStr("  "),
		kdl.WithFormatSortProperties(true),
	)
	if err != nil {
		panic(err)
	}
	fmt.Print(s)

	fmt.Println("=== emitted (hex integers, quoted strings) ===")
	// Build a fresh doc with integers for the emit options demo
	demo := kdl.NewDocument()
	demo.AddNode(kdl.NewKV("flags", 0xFF))
	demo.AddNode(kdl.NewKV("mask", 0xDEAD))
	demo.AddNode(kdl.NewKV("label", "hello world"))

	s, err = kdl.EmitToString(demo,
		kdl.WithIntegerFormat(kdl.Hex),
		kdl.WithStringAlwaysQuote(true),
		kdl.WithIndent("\t"),
	)
	if err != nil {
		panic(err)
	}
	fmt.Print(s)

	fmt.Println("=== emitted (KDL v1) ===")
	s, err = kdl.EmitToString(doc, kdl.WithVersion(kdl.Version1))
	if err != nil {
		panic(err)
	}
	fmt.Print(s)
}
