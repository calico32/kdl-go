package kdl_test

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

func ExampleDecode() {
	f := strings.NewReader(configKdl)

	var config Config
	err := kdl.Decode(f, &config)
	if err != nil {
		panic(fmt.Sprintf("Decode failed: %+v", err))
	}

	for _, host := range config.Hosts {
		fmt.Printf("%s: %s@%s:%d\n", host.Name, host.User, host.Hostname, host.Port)
	}

	// Output:
	// example1: root@example.com:22
	// example2: http@example.org:2022
}
