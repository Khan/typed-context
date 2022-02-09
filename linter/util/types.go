package lintutil

// This file defines utilities relating to types.

import "go/types"

// UnwrapMaybePointer returns T if passed any of T, *T, **T, etc.
func UnwrapMaybePointer(typ types.Type) types.Type {
	for {
		pointer, ok := typ.(*types.Pointer)
		if !ok {
			return typ
		}
		typ = pointer.Elem()
	}
}
