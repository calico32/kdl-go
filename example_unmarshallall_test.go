package kdl_test

import (
	"fmt"
	"strings"

	"github.com/calico32/kdl-go"
)

const hostsKdl = `
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

type Host struct {
	Name     string
	User     string
	Hostname string
	Port     int
}

func (h *Host) UnmarshalKDL(node *kdl.Node) (err error) {
	h.Name, err = kdl.Get(node, 0, kdl.AsString)
	if err != nil {
		return err
	}
	h.User, err = kdl.GetKV(node, "user", kdl.AsString)
	if err != nil {
		return err
	}
	h.Hostname, err = kdl.GetKV(node, "hostname", kdl.AsString)
	if err != nil {
		return err
	}
	h.Port, err = kdl.GetKV(node, "port", kdl.AsInt)
	if err != nil {
		return err
	}
	return nil
}

func ExampleUnmarshalAll() {

	f := strings.NewReader(hostsKdl)
	doc, err := kdl.NewParser(kdl.KdlVersion2, f).ParseDocument()
	if err != nil {
		panic(err)
	}

	hosts, err := kdl.UnmarshalAll[Host](doc.Nodes)
	if err != nil {
		panic(err)
	}

	for _, host := range hosts {
		fmt.Printf("host %s: %s@%s:%d\n", host.Name, host.User, host.Hostname, host.Port)
	}

	// Output:
	// host example1: root@example.com:22
	// host example2: http@example.org:2022
}
