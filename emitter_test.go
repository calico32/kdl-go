package kdl

import (
	"bytes"
	"math/big"
	"testing"
)

func TestEmitFloat(t *testing.T) {
	tests := []struct {
		name     string
		val      Value
		opts     []EmitterOption
		expected string
	}{
		{
			name:     "float64 simple",
			val:      NewFloat(1.23),
			expected: "node 1.23\n",
		},
		{
			name:     "float64 zero",
			val:      NewFloat(0.0),
			expected: "node 0.0\n",
		},
		{
			name:     "float64 large fixed",
			val:      NewFloat(100000.0),
			expected: "node 100000.0\n",
		},
		{
			name:     "float64 scientific default",
			val:      NewFloat(1e15),
			expected: "node 1e15\n",
		},
		{
			name:     "float64 small scientific default",
			val:      NewFloat(1e-15),
			expected: "node 1e-15\n",
		},
		{
			name: "float64 with options",
			val:  NewFloat(123.0),
			opts: []EmitterOption{
				WithTestSuiteFloatOptions(),
			},
			expected: "node 1.23E+02\n",
		},
		{
			name: "float64 zero with options",
			val:  NewFloat(0.0),
			opts: []EmitterOption{
				WithTestSuiteFloatOptions(),
			},
			expected: "node 0.0\n",
		},
		{
			name:     "bigFloat simple",
			val:      NewBigFloat(big.NewFloat(1.23)),
			expected: "node 1.23\n",
		},
		{
			name: "bigFloat high precision",
			val: func() Value {
				f, _, _ := big.ParseFloat("1.2345678901234567890123456789", 10, 100, big.ToNearestEven)
				return NewBigFloat(f)
			}(),
			expected: "node 1.2345678901234567890123456789\n",
		},
		{
			name: "bigFloat very large",
			val: func() Value {
				f := new(big.Float).SetInf(false) // Just to init
				f.Parse("1e1000", 10)
				return NewBigFloat(f)
			}(),
			expected: "node 1e1000\n",
		},
		{
			name: "floatPlus option",
			val:  NewFloat(1.23),
			opts: []EmitterOption{
				WithFloatPlus(true),
			},
			expected: "node +1.23\n",
		},
		{
			name: "floatPlus option negative",
			val:  NewFloat(-1.23),
			opts: []EmitterOption{
				WithFloatPlus(true),
			},
			expected: "node -1.23\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &Document{
				Nodes: []*Node{
					{
						name:  "node",
						args:  []Value{tt.val},
						props: map[string]Value{},
					},
				},
			}
			var buf bytes.Buffer
			err := Emit(doc, &buf, tt.opts...)
			if err != nil {
				t.Fatalf("Emit() error = %v", err)
			}
			if got := buf.String(); got != tt.expected {
				t.Errorf("Emit() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEmitStringAlwaysQuote(t *testing.T) {
	doc := &Document{
		Nodes: []*Node{
			{
				name:  "node",
				args:  []Value{NewString("simple")},
				props: map[string]Value{},
			},
			{
				name:  "node2",
				args:  []Value{NewString("")},
				props: map[string]Value{},
			},
		},
	}
	var buf bytes.Buffer
	err := Emit(doc, &buf, WithStringAlwaysQuote(true))
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	expected := `"node" "simple"
"node2" ""
`
	if got := buf.String(); got != expected {
		t.Errorf("Emit() = %q, want %q", got, expected)
	}
}

func TestEmitBareIdents(t *testing.T) {
	tests := []struct {
		name     string
		val      Value
		expected string
	}{
		{
			name:     "simple ident",
			val:      NewString("simple"),
			expected: "simple",
		},
		{
			name:     "ident with spaces",
			val:      NewString("with spaces"),
			expected: `"with spaces"`,
		},
		{
			name:     "ident with special chars",
			val:      NewString("special!@#"),
			expected: `"special!@#"`,
		},
		{
			name:     "empty string",
			val:      NewString(""),
			expected: `""`,
		},
		{
			name:     "reserved bool",
			val:      NewString("true"),
			expected: `"true"`,
		},
		{
			name:     "reserved float",
			val:      NewString("nan"),
			expected: `"nan"`,
		},
		{
			name:     "ident with leading digit",
			val:      NewString("1abc"),
			expected: `"1abc"`,
		},
		{
			name:     "ident with leading dot and digit",
			val:      NewString(".1abc"),
			expected: `".1abc"`,
		},
		{
			name:     "ident with leading sign and digit",
			val:      NewString("-1abc"),
			expected: `"-1abc"`,
		},
		{
			name:     "ident with leading sign and dot and digit",
			val:      NewString("+.1abc"),
			expected: `"+.1abc"`,
		},
		{
			name:     "ident with leading dot",
			val:      NewString(".abc"),
			expected: `.abc`,
		},
		{
			name:     "ident with leading sign",
			val:      NewString("-abc"),
			expected: `-abc`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &Document{
				Nodes: []*Node{
					{
						name:  "node",
						args:  []Value{tt.val},
						props: map[string]Value{},
					},
				},
			}
			var buf bytes.Buffer
			err := Emit(doc, &buf)
			if err != nil {
				t.Fatalf("Emit() error = %v", err)
			}
			got := buf.String()
			if got != "node "+tt.expected+"\n" {
				t.Errorf("Emit() = %q, want %q", got, "node "+tt.expected+"\n")
			}
		})
	}
}
