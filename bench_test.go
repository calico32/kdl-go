package kdl_test

import (
	"io"
	"strings"
	"testing"

	"github.com/calico32/kdl-go"
)

var benchSmall = `node 1 2 3 key="value"`

var benchMedium = strings.Repeat(
	`package name=acme version="1.2.3" {
	dep left-pad version="1.0.0"
	dep right-pad version="2.0.0" optional=#true
	script build "go build ./..."
	script test "go test -race ./..."
}
`,
	16,
)

var benchLarge = strings.Repeat(benchMedium, 16)

type benchPackage struct {
	Name    string        `kdl:"name,prop"`
	Version string        `kdl:"version,prop"`
	Deps    []benchDep    `kdl:"dep,multiple"`
	Scripts []benchScript `kdl:"script,multiple"`
}

type benchDep struct {
	Name     string `kdl:",arg"`
	Version  string `kdl:"version,prop"`
	Optional bool   `kdl:"optional,prop,omitzero"`
}

type benchScript struct {
	Name string `kdl:",arg"`
	Cmd  string `kdl:",arg"`
}

type benchDoc struct {
	Packages []benchPackage `kdl:"package,multiple"`
}

func benchInputs() []struct {
	name string
	src  string
} {
	return []struct {
		name string
		src  string
	}{
		{"small", benchSmall},
		{"medium", benchMedium},
		{"large", benchLarge},
	}
}

func BenchmarkParse(b *testing.B) {
	for _, in := range benchInputs() {
		b.Run(in.name, func(b *testing.B) {
			b.SetBytes(int64(len(in.src)))
			b.ReportAllocs()
			for b.Loop() {
				_, err := kdl.Parse(strings.NewReader(in.src))
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDecode(b *testing.B) {
	for _, in := range benchInputs()[1:] {
		b.Run(in.name, func(b *testing.B) {
			b.SetBytes(int64(len(in.src)))
			b.ReportAllocs()
			for b.Loop() {
				var dst benchDoc
				if err := kdl.Decode(strings.NewReader(in.src), &dst); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEncode(b *testing.B) {
	var doc benchDoc
	if err := kdl.Decode(strings.NewReader(benchLarge), &doc); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := kdl.Encode(doc, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEmit(b *testing.B) {
	for _, in := range benchInputs() {
		b.Run(in.name, func(b *testing.B) {
			parsed, err := kdl.Parse(strings.NewReader(in.src))
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(in.src)))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if err := kdl.Emit(parsed, io.Discard); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkFormat(b *testing.B) {
	for _, in := range benchInputs() {
		b.Run(in.name, func(b *testing.B) {
			parsed, err := kdl.Parse(strings.NewReader(in.src))
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(in.src)))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if err := kdl.Format(parsed, io.Discard); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
