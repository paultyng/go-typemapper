package generator

import (
	"go/types"
)

func isPointer(t types.Type) bool {
	if _, ok := t.(*types.Pointer); ok {
		return true
	}
	return false
}
