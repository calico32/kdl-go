package kdl

import (
	"fmt"
	"strings"
)

type Printer struct {
	builder     strings.Builder
	indent      int
	atLineStart bool
}

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
	p.printf("\n(node \"%s\"", node.Name)
	p.indent++
	if node.TypeAnnotation != nil {
		p.printf("\n(type \"%s\")", *node.TypeAnnotation)
	}
	for _, arg := range node.Arguments {
		p.print("\n(argument ")
		p.PrintValue(arg)
		p.print(")")
	}
	for _, prop := range node.PropertyOrder {
		p.printf("\n(property \"%s\" ", prop)
		p.PrintValue(node.Properties[prop])
		p.print(")")
	}
	for _, child := range node.Children {
		p.PrintNode(child)
	}
	p.indent--
	p.print(")")
}

func (p *Printer) PrintValue(v Value) {
	switch v := v.(type) {
	case String:
		p.printf("(string \"%s\"", v.value)
	case Integer:
		p.printf("(integer %d", v.value)
	case Float:
		p.printf("(float %f", v.value)
	case BigInt:
		p.printf("(bigint %s", v.value.String())
	case BigFloat:
		p.printf("(bigfloat %s", v.value.String())
	case Boolean:
		p.printf("(boolean %t", v.value)
	case Null:
		p.print("(null")
	default:
		p.printf("(unknown %T", v)
	}

	typeAnnot := v.TypeAnnotation()
	if typeAnnot != nil {
		p.indent++
		p.printf("\n(type \"%s\")", *typeAnnot)
		p.indent--
	}
	p.print(")")
}
