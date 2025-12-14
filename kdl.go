package kdl

// #cgo CFLAGS: -I/usr/local/include
// #cgo LDFLAGS: -L/usr/local/lib -lkdl -lm
// #include "kdl.h"
import "C"

import (
	"io"
	"runtime/cgo"
	"unsafe"
)

//export kdlgo_read
func kdlgo_read(ptr unsafe.Pointer, buf *C.char, size C.size_t) C.size_t {
	impl := (*cgo.Handle)(ptr).Value().(parserImpl)
	slice := unsafe.Slice((*byte)(unsafe.Pointer(buf)), size)
	n, err := impl.r.Read(slice)
	if n > 0 {
		return C.size_t(n)
	}
	if err != io.EOF {
		impl.err = err
	}
	return 0
}

//export kdlgo_write
func kdlgo_write(ptr unsafe.Pointer, buf C.kdlgo_char_const_ptr, size C.size_t) C.size_t {
	impl := (*cgo.Handle)(ptr).Value().(emitterImpl)
	slice := unsafe.Slice((*uint8)(unsafe.Pointer(buf)), size)
	n, err := impl.w.Write(slice)
	if err != nil {
		return 0
	}
	return C.size_t(n)
}
