package kdl

// #cgo CFLAGS: -I/usr/local/include
// #cgo LDFLAGS: -L/usr/local/lib -lkdl
// #include "kdl.h"
import "C"

import (
	"errors"
	"io"
	"runtime"
	"runtime/cgo"
	"slices"
	"unsafe"
)

// An Emitter is a wrapper around a ckdl emitter that writes to an
// [io.Writer].
type Emitter struct {
	emitterImpl
}

type emitterImpl struct {
	w    io.Writer
	h    cgo.Handle
	c    *C.kdl_emitter
	p    runtime.Pinner
	opts *C.kdl_emitter_options
}

// NewEmitter creates a new emitter that writes to the given [io.Writer]. It
// allocates the underlying C emitter and returns a [Emitter] instance.
//
// The emitter may be manually destroyed by calling the [Destroy] method, or it
// will be automatically destroyed when the emitter instance is no longer
// reachable. The emitter should not be used after it is destroyed.
func NewEmitter(ver KdlVersion, w io.Writer) *Emitter {
	v := C.kdl_version(C.KDL_VERSION_2)
	if ver == KdlVersion1 {
		v = C.kdl_version(C.KDL_VERSION_1)
	}
	impl := emitterImpl{w: w}
	impl.opts = &C.kdl_emitter_options{
		indent:          4,
		escape_mode:     C.KDL_ESCAPE_DEFAULT,
		identifier_mode: C.KDL_PREFER_BARE_IDENTIFIERS,
		version:         v,
		float_mode: C.kdl_float_printing_options{
			always_write_decimal_point_or_exponent: true,
			min_exponent:                           2,
			capital_e:                              true,
			exponent_plus:                          true,
		},
	}
	impl.p.Pin(impl.opts)
	impl.h = cgo.NewHandle(impl)
	impl.c = C.kdl_create_stream_emitter((C.kdl_write_func)(C.kdlgo_write), unsafe.Pointer(&impl.h), impl.opts)

	e := &Emitter{impl}
	runtime.AddCleanup(e, func(impl *emitterImpl) {
		C.kdl_destroy_emitter(impl.c)
		impl.p.Unpin()
		impl.h.Delete()
		impl.c = nil
		impl.opts = nil
		impl.w = nil
	}, &impl)
	return e
}

// Destroy destroys the emitter and releases all resources associated with it.
// The emitter should not be used after this method is called.
func (e *Emitter) Destroy() {
	C.kdl_destroy_emitter(e.c)
	e.p.Unpin()
	e.h.Delete()
	e.c = nil
	e.opts = nil
	e.w = nil
}

// EmitDocument emits the given [Document] to the underlying writer. It returns
// an error if the document is invalid or if there is an error writing to the
// writer.
func (e *Emitter) EmitDocument(doc *Document) error {
	for _, node := range doc.Nodes {
		if err := e.emitNode(node); err != nil {
			return err
		}
	}

	ok := C.kdl_emit_end(e.c)
	if !ok {
		return errors.New("failed to emit end")
	}

	return nil
}

func (e *Emitter) emitNode(node *Node) error {
	name, free := KdlString(node.Name)
	defer free()

	if node.TypeAnnotation == nil {
		if ok := C.kdl_emit_node(e.c, name); !ok {
			return errors.New("failed to emit node start")
		}
	} else {
		annot, free := KdlString(*node.TypeAnnotation)
		defer free()
		if ok := C.kdl_emit_node_with_type(e.c, annot, name); !ok {
			return errors.New("failed to emit node start with type")
		}
	}

	for _, arg := range node.Arguments {
		value, free := arg.c()
		defer free()
		if ok := C.kdl_emit_arg(e.c, &value); !ok {
			return errors.New("failed to emit argument")
		}
	}

	slices.Sort(node.PropertyOrder)
	for _, k := range node.PropertyOrder {
		v := node.Properties[k]

		key, free := KdlString(k)
		defer free()
		value, free := v.c()
		defer free()

		if ok := C.kdl_emit_property(e.c, key, &value); !ok {
			return errors.New("failed to emit property")
		}
	}

	if len(node.Children) > 0 {
		if ok := C.kdl_start_emitting_children(e.c); !ok {
			return errors.New("failed to emit children")
		}

		for _, child := range node.Children {
			if err := e.emitNode(child); err != nil {
				return err
			}
		}

		if ok := C.kdl_finish_emitting_children(e.c); !ok {
			return errors.New("failed to emit end children")
		}
	}

	return nil
}
