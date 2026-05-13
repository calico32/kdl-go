// Demonstrates building a KDL document programmatically using the AST API.
package main

import (
	"fmt"
	"os"

	"github.com/calico32/kdl-go"
)

func main() {
	doc := kdl.NewDocument()

	// Top-level key-value nodes
	doc.AddNode(kdl.NewKV("version", 2))
	doc.AddNode(kdl.NewKV("name", "my-app"))

	// Node with arguments
	tags := kdl.NewNode("tags")
	tags.AddArgument(kdl.NewString("go"))
	tags.AddArgument(kdl.NewString("kdl"))
	tags.AddArgument(kdl.NewString("config"))
	doc.AddNode(tags)

	// Node with properties
	server := kdl.NewNode("server")
	server.AddProperty("host", kdl.NewString("0.0.0.0"))
	server.AddProperty("port", kdl.NewInt(8080))
	server.AddProperty("tls", kdl.NewBool(true))
	doc.AddNode(server)

	// Node with children
	db := kdl.NewNode("database")
	db.AddKV("driver", kdl.NewString("postgres"))
	db.AddKV("host", kdl.NewString("localhost"))
	db.AddKV("port", kdl.NewInt(5432))
	db.AddKV("name", kdl.NewString("mydb"))

	// Nested children
	pool := db.NewChild("pool")
	pool.AddKV("min", kdl.NewInt(2))
	pool.AddKV("max", kdl.NewInt(10))
	pool.AddKV("timeout", kdl.NewFloat(30.0))

	doc.AddNode(db)

	// Emit to stdout
	if err := kdl.Emit(doc, os.Stdout, kdl.WithIndent("  ")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
