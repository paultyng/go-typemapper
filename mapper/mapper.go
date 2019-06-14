package mapper

import (
	"go/types"
	"strings"
)

type StructMapper struct {
	prefixes  []string
	ignore    []string
	manualMap map[string]string

	src *types.Struct
	dst *types.Struct
}

func NewStructMapper(src, dst types.Type) *StructMapper {
	m := &StructMapper{}

	m.src = unwrapStruct(src)
	m.dst = unwrapStruct(dst)

	if m.src == nil || m.dst == nil {
		return nil
	}

	return m
}

func (m *StructMapper) RecognizePrefixes(prefixes ...string) *StructMapper {
	m.prefixes = prefixes
	return m
}

func (m *StructMapper) MapField(srcField, dstField string) *StructMapper {
	if m.manualMap == nil {
		m.manualMap = map[string]string{}
	}
	m.manualMap[dstField] = srcField
	return m
}

func (m *StructMapper) IgnoreFields(dstFields ...string) *StructMapper {
	m.ignore = append(m.ignore, dstFields...)
	return m
}

func (m *StructMapper) typesMappable(src, dst types.Type) bool {
	var unwrap func(types.Type) types.Type
	unwrap = func(t types.Type) types.Type {
		switch t := t.(type) {
		case *types.Pointer:
			return unwrap(t.Elem())
		case *types.Named:
			return unwrap(t.Underlying())
		}
		return t
	}

	src = unwrap(src)
	dst = unwrap(dst)

	return types.AssignableTo(src, dst)
}

func (m *StructMapper) fieldsMappable(src, dst *types.Var) bool {
	srcNames := []string{src.Name()}
	dstNames := []string{dst.Name()}

	for _, prefix := range m.prefixes {
		if prefix != "" {
			if strings.HasPrefix(srcNames[0], prefix) {
				srcNames = append(srcNames, strings.TrimPrefix(srcNames[0], prefix))
			}

			if strings.HasPrefix(dstNames[0], prefix) {
				dstNames = append(dstNames, strings.TrimPrefix(dstNames[0], prefix))
			}
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

	return m.typesMappable(src.Type(), dst.Type())
}

func (m *StructMapper) findPair(src, dst *types.Struct, dstField *types.Var) *types.Var {
	for i := 0; i < src.NumFields(); i++ {
		srcField := src.Field(i)
		if m.fieldsMappable(srcField, dstField) {
			return srcField
		}
	}
	return nil
}

func (m *StructMapper) Map() MapConfiguration {
	noMatch := []Field{}
	pairs := []FieldPair{}

	for i := 0; i < m.dst.NumFields(); i++ {
		dstField := m.dst.Field(i)

		if dstField.Name() == "_" {
			continue
		}

		ignoreDst := false
		for _, ig := range m.ignore {
			if dstField.Name() == ig {
				ignoreDst = true
				break
			}
		}
		if ignoreDst {
			continue
		}

		var srcField *types.Var
		if mm, ok := m.manualMap[dstField.Name()]; ok {
			for i := 0; i < m.src.NumFields(); i++ {
				f := m.src.Field(i)
				if f.Name() == mm {
					srcField = f
					break
				}
			}
		} else {
			srcField = m.findPair(m.src, m.dst, dstField)
		}
		if srcField == nil {
			noMatch = append(noMatch, fieldFromVar(dstField))
			continue
		}
		pairs = append(pairs, FieldPair{
			Source:      fieldFromVar(srcField),
			Destination: fieldFromVar(dstField),
		})
	}
	return MapConfiguration{
		Pairs:   pairs,
		NoMatch: noMatch,
	}
}
