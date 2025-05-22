package kdl

// #cgo CFLAGS: -I/usr/local/include
// #cgo LDFLAGS: -L/usr/local/lib -lkdl
// #include "kdl.h"
// #include <stdlib.h>
import "C"

import "unsafe"

func GoString(kdlStr *C.kdl_str) string {
	return C.GoStringN(kdlStr.data, C.int(kdlStr.len))
}

func KdlString(goStr string) (C.kdl_str, func()) {
	cstr := C.CString(goStr)
	return C.kdl_str_from_cstr(cstr), func() {
		C.free(unsafe.Pointer(cstr))
	}
}
