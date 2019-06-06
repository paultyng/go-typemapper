package mapper

import (
	"go/types"
)

type FieldPair struct {
	Source      Field
	Destination Field
}

type Field struct {
	key string
	ptr bool
}

func (f *Field) Name() string {
	return f.key
}

// TODO: should expose typing information differently
func (f *Field) IsPointer() bool {
	return f.ptr
}

func fieldFromVar(v *types.Var) Field {
	_, ptr := v.Type().(*types.Pointer)
	return Field{
		key: v.Name(),
		ptr: ptr,
	}
}

type MapConfiguration struct {
	Pairs   []FieldPair
	NoMatch []Field
}
