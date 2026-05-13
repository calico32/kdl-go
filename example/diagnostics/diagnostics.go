// Demonstrates parsing KDL with diagnostics — collecting errors/warnings
// without aborting on the first failure.

package main

import (
	"fmt"
	"strings"

	"github.com/calico32/kdl-go"
)

// intentionally malformed input with recoverable errors
const broken = `
name "Alice"
age not-a-number
email "alice@example.com"
score 9.
unknown-key #invalid
tags "go" "kdl"
`

func main() {
	result, err := kdl.ParseNamedWithDiagnostics("config.kdl", strings.NewReader(broken))
	if err != nil {
		panic(err)
	}

	if len(result.Diagnostics) > 0 {
		fmt.Println("=== diagnostics ===")
		for _, d := range result.Diagnostics {
			severity := [...]string{"error", "warning", "info", "hint"}[d.Severity]
			fmt.Printf("  [%s] %s: %s\n", severity, d.Start, d.Message)
		}
		fmt.Println()
	}

	if result.HasErrors() {
		fmt.Println("document has errors — partial result:")
	} else {
		fmt.Println("document parsed successfully:")
	}

	// Even with errors, the parser recovers and returns whatever it could parse.
	for _, node := range result.Document.Nodes {
		args := node.Arguments()
		if len(args) > 0 {
			fmt.Printf("  %s = %v (kind: %s)\n", node.Name(), args[0].RawValue(), args[0].Kind())
		} else {
			fmt.Printf("  %s (no args)\n", node.Name())
		}
	}
}
