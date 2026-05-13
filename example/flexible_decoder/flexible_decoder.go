package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/calico32/kdl-go"
)

type Example struct {
	Name        string `kdl:"example"`
	Description string `kdl:"description"`
	Url         string `kdl:"url"`
	Image       string `kdl:"image"`
}

type Document struct {
	First  Example `kdl:"first"`
	Second Example `kdl:"second"`
}

func main() {
	// by default, the decoder can fill struct fields from properties, children,
	// or a mix of both
	input := `
		first name=Example description="foo\nbar" url="https://example.com" image=foo.png

		second name=Example {
			description """
				foo
				bar
				"""
			url "https://example.com"
			image foo.png
		}
	`

	var doc Document
	err := kdl.Decode(strings.NewReader(input), &doc)
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(doc.First, doc.Second) {
		panic("not equal")
	}

	fmt.Println("equal")
}
