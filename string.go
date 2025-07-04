package kdl

// #cgo CFLAGS: -I/usr/local/include
// #cgo LDFLAGS: -L/usr/local/lib -lkdl
// #include "kdl.h"
// #include <stdlib.h>
import "C"

import "unsafe"

// goString converts a C kdl_str to a Go string.
func goString(kdlStr *C.kdl_str) string {
	return C.GoStringN(kdlStr.data, C.int(kdlStr.len))
}

// kdlString converts a Go string to a kdl_str. It returns a function to free
// the underlying C string after it is no longer needed. The kdl_str should not
// be referenced after free is called.
func kdlString(goStr string) (C.kdl_str, func()) {
	cstr := C.CString(goStr)
	return C.kdl_str_from_cstr(cstr), func() {
		C.free(unsafe.Pointer(cstr))
	}
}
