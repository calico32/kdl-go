# kdl-go

Package kdl implements a parser and emitter for the [KDL](https://kdl.dev/)
document format (version 1 and 2). It supports reading and writing KDL
documents, as well as decoding KDL documents into Go data structures and
encoding them back into KDL.

This implementation follows the KDL 1.0.0 and 2.0.0 specifications and passes
the upstream test suite for each (see `kdl_test.go`). Note that the parser
primarily targets the v2 spec and is somewhat more permissive when parsing v1
input.

Visit [pkg.go.dev](https://pkg.go.dev/github.com/calico32/kdl-go) for the full
documentation.

> [!Caution]
> This package's public 0.x API is not stable yet and may change at any time.
> Use at your own risk.

```go
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/calico32/kdl-go"
)

const configKdl = `
host example1 {
	user root
	hostname example.com
	port 22
}

host example2 {
	user http
	hostname example.org
	port 2022
}
`

type Config struct {
	Hosts []*Host `kdl:"host,multiple"`
}

type Host struct {
	Name     string `kdl:",argument"`
	User     string `kdl:"user"`
	Hostname string `kdl:"hostname"`
	Port     int    `kdl:"port"`
}

func main() {
    // open a KDL document (or, read from a string in this case)
	f := strings.NewReader(configKdl)

    // decode it into a Config
	var config Config
	err := kdl.Decode(f, &config)
	if err != nil {
		panic(err)
	}

    // print out the hosts
	for _, host := range config.Hosts {
		fmt.Printf("%s: %s@%s:%d\n", host.Name, host.User, host.Hostname, host.Port)
		// example1: root@example.com:22
		// example2: http@example.org:2022
	}

    // create an output file
	out, err := os.Create("output.kdl")
	if err != nil {
        panic(err)
	}

    // encode the Config back into KDL
	err = kdl.Encode(config, out)
	if err != nil {
		panic(err)
	}
}
```
