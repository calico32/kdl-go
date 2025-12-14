package kdl_test

import (
	"fmt"
	"strings"

	"github.com/calico32/kdl-go"
)

const _configKdl = `
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
	Hosts []*_Host `kdl:"host,multiple"`
}

type _Host struct {
	User     string `kdl:"user"`
	Hostname string `kdl:"hostname"`
	Port     int    `kdl:"port"`
}

func ExampleConfig() {
	f := strings.NewReader(_configKdl)

	var config Config
	err := kdl.Decode(f, &config)
	if err != nil {
		panic(err)
	}

	for _, host := range config.Hosts {
		fmt.Printf("%s@%s:%d\n", host.User, host.Hostname, host.Port)
	}
}
