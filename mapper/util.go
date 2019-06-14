package mapper

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

// difference returns the elements in `a` that aren't in `b`.
// from https://stackoverflow.com/a/45428032
func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
