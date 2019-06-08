package generator

import (
	"go/types"
)

func unwrapPointer(v types.Type) types.Type {
	if v == nil {
		return nil
	}
	if p, ok := v.(*types.Pointer); ok {
		return p.Elem()
	}
	return v
}

func isPointer(t types.Type) bool {
	if _, ok := t.(*types.Pointer); ok {
		return true
	}
	return false
}
