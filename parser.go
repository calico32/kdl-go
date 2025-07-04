package kdl

// #cgo CFLAGS: -I/usr/local/include
// #cgo LDFLAGS: -L/usr/local/lib -lkdl
// #include "kdl.h"
import "C"

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"runtime/cgo"
	"unsafe"

	"github.com/pkg/errors"
)

type KdlVersion int

const (
	KdlVersionAuto KdlVersion = iota
	KdlVersion1
	KdlVersion2
)

// A kdlEvent is a Go equivalent for the C kdl_event_data struct.
type kdlEvent struct {
	Type  C.kdl_event
	Name  string
	Value Value
}

func (d *kdlEvent) String() string {
	return fmt.Sprintf("kdlEvent{Type: %s, Name: %v, Value: %v}", kdlEventName(d.Type), d.Name, d.Value)
}

// A Parser is a wrapper around a ckdl Parser that reads from an
// [io.Reader].
type Parser struct {
	parserImpl
}

type parserImpl struct {
	ev    *kdlEvent
	debug io.Writer
	r     io.Reader
	h     cgo.Handle
	c     *C.kdl_parser
}

func kdlEventName(ev C.kdl_event) string {
	switch ev {
	case C.KDL_EVENT_ARGUMENT:
		return "argument"
	case C.KDL_EVENT_END_NODE:
		return "end_node"
	case C.KDL_EVENT_EOF:
		return "eof"
	case C.KDL_EVENT_PARSE_ERROR:
		return "parse_error"
	case C.KDL_EVENT_PROPERTY:
		return "property"
	case C.KDL_EVENT_START_NODE:
		return "start_node"
	}
	return "unknown"
}

// NewParser creates a new parser that reads from the given [io.Reader]. It
// allocates the underlying C parser and returns a [parser] instance.
//
// The parser may be manually destroyed by calling [parser.Destroy], or it
// will be automatically destroyed if the parser instance is no longer
// reachable. The parser should not be used after it is destroyed.
func NewParser(kdlVersion KdlVersion, r io.Reader) *Parser {
	impl := parserImpl{r: r}
	var v C.kdl_parse_option
	switch kdlVersion {
	case KdlVersionAuto:
		v = C.KDL_DETECT_VERSION
	case KdlVersion1:
		v = C.KDL_READ_VERSION_1
	case KdlVersion2:
		v = C.KDL_READ_VERSION_2
	default:
		panic("invalid kdl version")
	}

	impl.h = cgo.NewHandle(impl)
	impl.c = C.kdl_create_stream_parser((C.kdl_read_func)(C.kdlgo_read), unsafe.Pointer(&impl.h), v)

	p := &Parser{impl}
	runtime.AddCleanup(p, func(impl *parserImpl) {
		C.kdl_destroy_parser(impl.c)
		impl.h.Delete()
		impl.c = nil
		impl.r = nil
	}, &impl)

	return p
}

// SetDebug sets the writer to which debug output will be written. If the writer
// is nil, debug output will be disabled.
func (p *Parser) SetDebug(w io.Writer) {
	p.debug = w
}

// Destroy destroys the parser and releases all resources associated with it.
// The parser should not be used after this method is called.
func (p *Parser) Destroy() {
	C.kdl_destroy_parser(p.c)
	p.c = nil
	p.ev = nil
	p.r = nil
	p.h.Delete()
}

func (p *Parser) debugf(format string, args ...any) {
	if p.debug != nil {
		fmt.Fprintf(p.debug, format, args...)
	}
}

// ParseDocument parses a document from the underlying reader and returns a
// [Document] instance. It returns an error if the document is invalid or if
// there is an error reading from the reader.
func (d *Parser) ParseDocument() (*Document, error) {
	d.next()
	if d.ev == nil {
		return nil, errors.New("no event data")
	}

	doc := &Document{
		Nodes: []*Node{},
	}

	for {
		if d.ev.Type == C.KDL_EVENT_START_NODE {
			n, err := d.nextNode(nil)
			if err != nil {
				return nil, err
			}
			doc.Nodes = append(doc.Nodes, n)
			continue
		}

		break
	}

	_, err := d.accept(C.KDL_EVENT_EOF)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// nextNode parses and returns the next complete node from the underlying
// reader, or an error if the node is invalid or if there is an error reading
// from the reader.
func (d *Parser) nextNode(parent *Node) (*Node, error) {
	start, err := d.accept(C.KDL_EVENT_START_NODE)
	if err != nil {
		return nil, err
	}
	node := &Node{
		Name:          start.Name,
		Children:      []*Node{},
		Arguments:     []Value{},
		Properties:    map[string]Value{},
		PropertyOrder: []string{},
		Parent:        parent,
	}

	switch s := start.Value.(type) {
	case Null:
		node.TypeAnnotation = s.typeAnnotation
	default:
		return nil, errors.New("invalid type annotation")
	}

	// accept arguments and properties
	for {
		if d.ev.Type == C.KDL_EVENT_ARGUMENT {
			arg, err := d.accept(C.KDL_EVENT_ARGUMENT)
			if err != nil {
				return nil, err
			}
			node.Arguments = append(node.Arguments, arg.Value)
			continue
		}

		if d.ev.Type == C.KDL_EVENT_PROPERTY {
			prop, err := d.accept(C.KDL_EVENT_PROPERTY)
			if err != nil {
				return nil, err
			}
			_, repeated := node.Properties[prop.Name]
			node.Properties[prop.Name] = prop.Value
			if !repeated {
				node.PropertyOrder = append(node.PropertyOrder, prop.Name)
			}
			continue
		}

		break
	}

	// accept children
	for {
		if d.ev.Type == C.KDL_EVENT_START_NODE {
			n, err := d.nextNode(node)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, n)
			continue
		}

		break
	}

	_, err = d.accept(C.KDL_EVENT_END_NODE)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// next gets the next event from the ckdl parser and returns an equivalent
// [kdlEvent], or an error if the ckdl parser returns an error or an EOF is
// reached.
func (p *Parser) next() (*kdlEvent, error) {
	if p.ev != nil && p.ev.Type == C.KDL_EVENT_PARSE_ERROR {
		return nil, errors.New("parse error already reached")
	}
	if p.ev != nil && p.ev.Type == C.KDL_EVENT_EOF {
		return p.ev, nil
	}

	ev := C.kdl_parser_next_event(p.c)
	if ev == nil {
		return nil, errors.New("no event data")
	}
	if ev.event == C.KDL_EVENT_PARSE_ERROR {
		return nil, errors.New("ckdl parse error")
	}

	name := goString(&ev.name)
	v, err := newKdlValue(ev.value)
	if err != nil {
		return nil, err
	}

	p.ev = &kdlEvent{
		Type:  C.kdl_event(ev.event),
		Name:  name,
		Value: v,
	}

	p.debugf("--> %s\n", p.ev)

	return p.ev, nil
}

// accept checks if the current event is of the expected type and returns it if
// it is, or an error if it is not. It also advances the parser to the next
// event.
func (p *Parser) accept(eventType C.kdl_event) (ev *kdlEvent, err error) {
	_, file, line, _ := runtime.Caller(1)
	// always advance the parser
	defer func() {
		_, err = p.next()
	}()

	file = filepath.Base(file)
	caller := fmt.Sprintf("%s:%d", file, line)

	if p.ev == nil {
		p.debugf(" <-- %s NOACCEPT no data\n", caller)
		return nil, errors.New("no event data")
	}

	if p.ev.Type != eventType {
		p.debugf(" <-- %s NOACCEPT expected %s, got %s\n", caller, kdlEventName(eventType), kdlEventName(p.ev.Type))
		return nil, errors.Errorf("expected %s, got %s", kdlEventName(eventType), kdlEventName(p.ev.Type))
	}

	p.debugf(" <-- %s ACCEPT\n", caller)
	ev = p.ev
	return
}
