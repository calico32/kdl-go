// Demonstrates traversing a KDL document tree with Walk.

package main

import (
	"fmt"
	strs "strings"

	"github.com/calico32/kdl-go"
)

const input = `
project "my-app" version="1.0.0" {
	dependencies {
		dep "github.com/foo/bar" version="2.3.0"
		dep "github.com/baz/qux" version="0.1.0" optional=#true
	}
	build {
		output "dist/"
		target "linux/amd64"
		target "darwin/arm64"
		flags "-trimpath" "-ldflags=-s -w"
	}
	scripts {
		run "go run ./cmd/app"
		test "go test ./..."
		lint "golangci-lint run"
	}
}
`

func main() {
	doc, err := kdl.Parse(strs.NewReader(input))
	if err != nil {
		panic(err)
	}

	fmt.Println("=== all nodes ===")
	kdl.Walk(doc, func(node *kdl.Node, depth int) bool {
		indent := strs.Repeat("  ", depth)
		fmt.Printf("%s%s\n", indent, node.Name())
		return true // visit children
	})

	fmt.Println()
	fmt.Println("=== deps only ===")
	kdl.Walk(doc, func(node *kdl.Node, depth int) bool {
		if node.Name() == "dep" {
			args := node.Arguments()
			ver := node.Prop("version")
			optional := node.Prop("optional")

			opt := ""
			if optional.IsValid() && optional.Bool() {
				opt = " (optional)"
			}
			fmt.Printf("  %s @ %s%s\n", args[0].String(), ver.String(), opt)
			return false // no children to visit anyway
		}
		return true
	})

	fmt.Println()
	fmt.Println("=== collect all string args ===")
	var strings2 []string
	kdl.Walk(doc, func(node *kdl.Node, depth int) bool {
		for _, arg := range node.Arguments() {
			if arg.Kind() == kdl.String {
				strings2 = append(strings2, arg.String())
			}
		}
		return true
	})
	for _, s := range strings2 {
		fmt.Printf("  %q\n", s)
	}
}
