# kdl-go

Package kdl is a Go wrapper for [ckdl](https://github.com/tjol/ckdl), a C library for reading and writing KDL documents. It provides a Parser and Emitter for reading and writing KDL documents and a Decode for unmarshaling KDL documents into Go types.

> [!Caution]
> This package's public API is not stable and may change at any time.
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
