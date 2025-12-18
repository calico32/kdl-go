package kdl

import (
	"fmt"
	"strings"
)

// A Printer formats a KDL document as an S-expression.
type Printer struct {
	builder     strings.Builder
	indent      int
	atLineStart bool
}

// PrintDocument formats the given KDL document as an S-expression.
func PrintDocument(doc *Document) string {
	p := NewPrinter()
	p.PrintDocument(doc)
	return p.String()
}

func NewPrinter() *Printer {
	return &Printer{}
}

func (p *Printer) String() string {
	return p.builder.String()
}

func (p *Printer) print(s string) {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if p.atLineStart {
			p.builder.WriteString(strings.Repeat("  ", p.indent))
			p.atLineStart = false
		}
		p.builder.WriteString(line)
		if i < len(lines)-1 {
			p.builder.WriteString("\n")
			p.atLineStart = true
		}
	}
}

func (p *Printer) printf(format string, args ...interface{}) {
	p.print(fmt.Sprintf(format, args...))
}

func (p *Printer) PrintDocument(doc *Document) {
	p.print("(document")
	p.indent++
	for _, node := range doc.Nodes {
		p.PrintNode(node)
	}
	p.indent--
	p.print(")")
}

func (p *Printer) PrintNode(node *Node) {
	if node == nil {
		p.print("\n(node nil)")
		return
	}
	p.printf("\n(node \"%s\"", node.name)
	p.indent++
	if ty, ok := node.TypeAnnotation(); ok {
		p.printf("\n(type \"%s\")", ty)
	}
	for _, arg := range node.Arguments() {
		p.print("\n(argument ")
		p.PrintValue(arg)
		p.print(")")
	}
	for _, prop := range node.PropertyOrder() {
		p.printf("\n(property \"%s\" ", prop)
		p.PrintValue(node.Properties()[prop])
		p.print(")")
	}
	for _, child := range node.Children().Nodes {
		p.PrintNode(child)
	}
	p.indent--
	p.print(")")
}

func (p *Printer) PrintValue(v Value) {
	switch v.Kind() {
	case String:
		p.printf("(string %q", v.String())
	case Int:
		p.printf("(integer %d", v.Int())
	case Float:
		p.printf("(float %f", v.Float())
	case BigInt:
		p.printf("(bigint %s", v.BigInt().String())
	case BigFloat:
		p.printf("(bigfloat %s", v.BigFloat().String())
	case Bool:
		p.printf("(boolean %t", v.Bool())
	case Null:
		p.print("(null")
	default:
		p.printf("(unknown %s", v)
	}

	typeAnnot, ok := v.TypeAnnotation()
	if ok {
		p.indent++
		p.printf("\n(type \"%s\")", typeAnnot)
		p.indent--
	}
	p.print(")")
}
