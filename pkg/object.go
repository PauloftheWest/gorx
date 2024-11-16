package gorx

/*
#cgo CFLAGS: -I ../ext/orx/code/include
#cgo LDFLAGS: -lorxd -L ../ext/orx/code/lib/dynamic/

#include "orx.h"
#include "object/orxObject.h"
*/
import "C"

// taking a struct
type Object struct {
	object *C.orxOBJECT
}

func NewObject() *Object {
	obj := C.orxObject_Create()
	return &Object{object: obj}
}

func (self *Object) Enable(enable uint) bool {
	C.orxObject_Enable(self.object, C.uint(enable))
	return true
}

func (self *orxOBJECT) orxObject_GetNext(_stGroupID C.orxSTRINGID) *C.orxOBJECT {
	return C.orxObject_GetNext(self, _stGroupID)
}
