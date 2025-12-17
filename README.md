# kdl-go

Package kdl implements a parser and emitter for the [KDL](https://kdl.dev/) document format (version 2 only). It supports reading and writing KDL documents, as well as decoding KDL documents into Go data structures.

This implementation follows the KDL 2.0.0 specification and passes the upstream test suite (see `kdl_test.go`).

> [!Caution]
> This package's public 0.x API is not stable yet and may change at any time.
> Use at your own risk.

```go
package main

import (
	"fmt"
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
	f := strings.NewReader(configKdl)

	var config Config
	err := kdl.Decode(f, &config)
	if err != nil {
		panic(err)
	}

	for _, host := range config.Hosts {
		fmt.Printf("%s: %s@%s:%d\n", host.Name, host.User, host.Hostname, host.Port)
	}

	// Output:
	// example1: root@example.com:22
	// example2: http@example.org:2022
}
```
