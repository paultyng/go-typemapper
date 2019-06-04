package mapper

import (
	"go/types"
)

type FieldPair struct {
	Source      *types.Var
	Destination *types.Var
}

type Field struct {
	Var *types.Var
}

type MapConfiguration struct {
	Pairs   []FieldPair
	NoMatch []Field
}
