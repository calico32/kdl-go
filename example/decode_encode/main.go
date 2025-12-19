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
	// Open a KDL document (or read from a string, in this case)
	f := strings.NewReader(configKdl)

	// Decode the KDL document into a Config
	var config Config
	err := kdl.Decode(f, &config)
	if err != nil {
		panic(err)
	}

	// Print out the hosts
	for _, host := range config.Hosts {
		fmt.Printf("%s: %s@%s:%d\n", host.Name, host.User, host.Hostname, host.Port)
		// example1: root@example.com:22
		// example2: http@example.org:2022
	}

	// Open a file to write the output KDL
	out, err := os.Create("output.kdl")
	if err != nil {
		panic(err)
	}

	// Encode the config back to KDL
	err = kdl.Encode(config, out)
	if err != nil {
		panic(err)
	}
}
