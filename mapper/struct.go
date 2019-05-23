package mapper

import (
	"strings"
	"go/types"
)

type structMapper struct {
	//TODO: multiple...
	prefix string
}

func (m *structMapper) TypesMappable(src, dst types.Type) bool {
	if types.AssignableTo(src, dst) {
		return true
	}

	// int => string? etc... probably need an explicit listing of valid options
	// if types.ConvertibleTo(src, dst) {
	// 	return true
	// }

	if dstP, ok := dst.(*types.Pointer); ok {
		return m.TypesMappable(src, dstP.Elem())
	}

	return false
}

func (m *structMapper) FieldsMappable(src, dst *types.Var) bool {
	srcNames := []string{src.Name()}
	dstNames := []string{dst.Name()}

	if m.prefix != "" {
		if strings.HasPrefix(srcNames[0], m.prefix) {
			srcNames = append(srcNames, strings.TrimPrefix(srcNames[0], m.prefix))
		}

		if strings.HasPrefix(dstNames[0], m.prefix) {
			dstNames = append(dstNames, strings.TrimPrefix(dstNames[0], m.prefix))
		}
	}

	// TODO: add options for this, casing, type conversions, etc

	matchFound := func() bool {
		for _, dstName := range dstNames {
			if dstName == "" {
				continue
			}
			for _, srcName := range srcNames {
				if srcName == "" {
					continue
				}
				if dstName == srcName {
					return true
				}
			}
		}
		return false
	}()

	if !matchFound {
		return false
	}

	return m.TypesMappable(src.Type(), dst.Type())
}

func (m *structMapper) FindPair(src, dst *types.Struct, dstField *types.Var) *types.Var {
	for i := 0; i < src.NumFields(); i++ {
		srcField := src.Field(i)
		if m.FieldsMappable(srcField, dstField) {
			return srcField
		}
	}
	return nil
}

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

func (m *structMapper) Map(src, dst *types.Struct) MapConfiguration {
	noMatch := []Field{}
	pairs := []FieldPair{}

	for i := 0; i < dst.NumFields(); i++ {
		dstField := dst.Field(i)

		if dstField.Name() == "_" {
			continue
		}

		srcField := m.FindPair(src, dst, dstField)
		if srcField == nil {
			noMatch = append(noMatch, Field{dstField})
			continue
		}
		pairs = append(pairs, FieldPair{
			Source:      srcField,
			Destination: dstField,
		})
	}
	return MapConfiguration{
		Pairs:   pairs,
		NoMatch: noMatch,
	}
}
