// Demonstrates implementing custom Marshaler/Unmarshaler interfaces.
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/calico32/kdl-go"
)

// Duration wraps time.Duration with KDL marshal support.
type Duration time.Duration

func (d Duration) MarshalKDLValue() (kdl.Value, error) {
	return kdl.NewString(time.Duration(d).String()), nil
}

func (d *Duration) UnmarshalKDLValue(v kdl.Value) error {
	if v.Kind() != kdl.String {
		return fmt.Errorf("duration: expected string, got %s", v.Kind())
	}
	parsed, err := time.ParseDuration(v.String())
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

// Job marshals/unmarshals as a single KDL node with custom shape.
type Job struct {
	Name     string
	Command  string
	Interval Duration
	Retries  int
}

var _ kdl.Marshaler = &Job{}
var _ kdl.Unmarshaler = &Job{}

func (j *Job) MarshalKDL() (*kdl.Node, error) {
	node := kdl.NewNode("job", kdl.NewString(j.Name))
	node.AddProperty("command", kdl.NewString(j.Command))
	node.AddProperty("interval", kdl.NewString(time.Duration(j.Interval).String()))
	node.AddProperty("retries", kdl.NewInt(j.Retries))
	return node, nil
}

func (j *Job) UnmarshalKDL(node *kdl.Node) error {
	args := node.Arguments()
	if len(args) < 1 {
		return fmt.Errorf("job: missing name argument")
	}
	j.Name = args[0].String()

	if cmd := node.Prop("command"); cmd.IsValid() {
		j.Command = cmd.String()
	}
	if iv := node.Prop("interval"); iv.IsValid() {
		d, err := time.ParseDuration(iv.String())
		if err != nil {
			return err
		}
		j.Interval = Duration(d)
	}
	if r := node.Prop("retries"); r.IsValid() {
		j.Retries = r.Int()
	}
	return nil
}

const input = `
job "sync-db" command="./sync.sh" interval="5m" retries=3
job "cleanup" command="./cleanup.sh" interval="1h" retries=1
`

func main() {
	doc, err := kdl.Parse(strings.NewReader(input))
	if err != nil {
		panic(err)
	}

	jobs, err := kdl.UnmarshalAll[Job](doc.GetNodes("job"))
	if err != nil {
		panic(err)
	}

	for _, j := range jobs {
		fmt.Printf("job %q: %s every %s (%d retries)\n",
			j.Name, j.Command, time.Duration(j.Interval), j.Retries)
	}

	fmt.Println()

	// Round-trip: marshal back to KDL
	out := kdl.NewDocument()
	for _, j := range jobs {
		if err := out.MarshalNodes(j); err != nil {
			panic(err)
		}
	}

	s, err := kdl.EmitToString(out)
	if err != nil {
		panic(err)
	}
	fmt.Print(s)
}
