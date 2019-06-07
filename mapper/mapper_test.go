package mapper

import (
	"fmt"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindMatchingField(t *testing.T) {
	pkg := types.NewPackage("example.com/mappertest", "mappertest")

	var (
		stringType    = types.Universe.Lookup("string").Type()
		ptrStringType = types.NewPointer(stringType)
		intType       = types.Universe.Lookup("int").Type()
	)

	var (
		fooStringVar    = types.NewVar(1, pkg, "Foo", stringType)
		fooPtrStringVar = types.NewVar(2, pkg, "Foo", ptrStringType)
		fooIntVar       = types.NewVar(3, pkg, "Foo", intType)

		getFooStringVar = types.NewVar(1, pkg, "GetFoo", stringType)

		barIntVar = types.NewVar(4, pkg, "Bar", intType)
	)

	for i, c := range []struct {
		expected *types.Var
		src      *types.Struct
		dst      *types.Struct
		dstField *types.Var
	}{
		{nil, types.NewStruct(nil, nil), types.NewStruct([]*types.Var{fooStringVar}, nil), fooStringVar},
		{nil, types.NewStruct([]*types.Var{barIntVar}, nil), types.NewStruct([]*types.Var{fooStringVar}, nil), fooStringVar},

		{nil, types.NewStruct([]*types.Var{fooIntVar}, nil), types.NewStruct([]*types.Var{fooStringVar}, nil), fooStringVar},

		{fooStringVar, types.NewStruct([]*types.Var{fooStringVar}, nil), types.NewStruct([]*types.Var{fooStringVar}, nil), fooStringVar},
		{fooStringVar, types.NewStruct([]*types.Var{fooStringVar}, nil), types.NewStruct([]*types.Var{fooPtrStringVar}, nil), fooStringVar},
		{fooStringVar, types.NewStruct([]*types.Var{fooStringVar, barIntVar}, nil), types.NewStruct([]*types.Var{fooStringVar}, nil), fooStringVar},
		{fooStringVar, types.NewStruct([]*types.Var{fooStringVar, barIntVar}, nil), types.NewStruct([]*types.Var{fooStringVar, barIntVar}, nil), fooStringVar},

		// prefix tests, both directions
		{fooStringVar, types.NewStruct([]*types.Var{fooStringVar}, nil), types.NewStruct([]*types.Var{getFooStringVar}, nil), getFooStringVar},
		{getFooStringVar, types.NewStruct([]*types.Var{getFooStringVar}, nil), types.NewStruct([]*types.Var{fooStringVar}, nil), fooStringVar},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			sm := NewStructMapper(c.src, c.dst)
			sm.RecognizePrefixes("Get")
			sm.MapField("MapFieldSrc", "MapFieldDst")

			actual := sm.findPair(c.src, c.dst, c.dstField)
			assert.Equal(t, c.expected, actual)
		})
	}
}

// TODO: test IgnoreFields
