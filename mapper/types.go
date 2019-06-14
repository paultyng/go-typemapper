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
	ty  types.Type
}

func (f *Field) Name() string {
	return f.key
}

func (f *Field) Type() types.Type {
	return f.ty
}

func fieldFromVar(v *types.Var) Field {
	return Field{
		key: v.Name(),
		ty:  v.Type(),
	}
}

type MapConfiguration struct {
	Pairs   []FieldPair
	NoMatch []Field
}
