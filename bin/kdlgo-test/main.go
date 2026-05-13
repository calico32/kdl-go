package main

import (
	"encoding/json"
	"os"

	"github.com/calico32/kdl-go"
)

func main() {
	doc, err := kdl.Parse(os.Stdin, kdl.WithVersion(kdl.Version2))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	output, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	os.Stdout.Write(output)
}
