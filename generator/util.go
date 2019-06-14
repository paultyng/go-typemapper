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

func unwrapSlice(v types.Type) *types.Slice {
	if v == nil {
		return nil
	}
	var unwrap func(v types.Type) *types.Slice
	unwrap = func(v types.Type) *types.Slice {
		switch v := v.(type) {
		case *types.Pointer:
			return unwrap(v.Elem())
		case *types.Named:
			return unwrap(v.Underlying())
		case *types.Slice:
			return v
		}
		return nil
	}
	return unwrap(v)
}

func unwrapStruct(v types.Type) *types.Struct {
	if v == nil {
		return nil
	}
	var unwrap func(v types.Type) *types.Struct
	unwrap = func(v types.Type) *types.Struct {
		switch v := v.(type) {
		case *types.Pointer:
			return unwrap(v.Elem())
		case *types.Named:
			return unwrap(v.Underlying())
		case *types.Struct:
			return v
		}
		return nil
	}
	return unwrap(v)
}
