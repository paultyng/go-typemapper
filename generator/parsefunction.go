package generator

import (
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/ssa"
)

func (g *Generator) parseFunction(f *ssa.Function) (*mappingFunc, error) {
	var (
		err error
		m   *mappingFunc
	)
	for _, blk := range f.Blocks {
		for _, inst := range blk.Instrs {
			switch inst := inst.(type) {
			case *ssa.Alloc:
				if m != nil {
					// ignore allocations after the CreateMap call
					continue
				}

				//TODO: check that one of the reffers is the an `*ssa.Return`, also that the type matches dstP type
			case ssa.CallInstruction:
				if !isTypeMapperCall(inst) {
					continue
				}
				callF, ok := inst.Common().Value.(*ssa.Function)
				if !ok {
					return nil, nil
				}

				if m == nil {
					if callF.Name() != "CreateMap" {
						return nil, nil
					}
					m, err = handleCreateMap(inst, f)
					if err != nil {
						return nil, errors.WithStack(err)
					}
					continue
				}

				// TODO: do this via map of funcs?
				switch callF.Name() {
				default:
					return nil, errors.Errorf("unexpected typemapper call %s", callF.Name())
				case "RecognizePrefixes":
					err = handleRecognizePrefixes(m, inst)
					if err != nil {
						return nil, errors.WithStack(err)
					}
				case "MapField":
					err = handleMapField(m, inst)
					if err != nil {
						return nil, errors.WithStack(err)
					}
				case "IgnoreFields":
					err = handleIgnoreFields(m, inst)
					if err != nil {
						return nil, errors.WithStack(err)
					}
				case "MapWith":
					err = handleMapWith(m, inst)
					if err != nil {
						return nil, errors.WithStack(err)
					}
				}
			}
		}
	}
	if m == nil {
		return nil, nil
	}

	//validations
	_, dstPointer := m.dstType.(*types.Pointer)
	_, dstSlice := m.dstType.(*types.Slice)
	_, dstMap := m.dstType.(*types.Map)
	if !(dstPointer || dstSlice || dstMap) && !m.dstConstructed {
		return nil, errors.Errorf("non-pointer destination types cannot be passed as parameters")
	}

	return m, nil
}

func walkReferrers(v ssa.Value, cb func(ssa.Instruction) bool) bool {
	refs := v.Referrers()
	if refs == nil {
		return true
	}
	for _, ref := range *refs {
		cont := cb(ref)
		if !cont {
			return false
		}
		if cv, ok := ref.(ssa.Value); ok {
			cont := walkReferrers(cv, cb)
			if !cont {
				return false
			}
		}
	}
	return true
}

func literalString(v ssa.Value) (string, error) {
	switch v := v.(type) {
	case *ssa.Const:
		s, err := strconv.Unquote(v.Value.ExactString())
		if err != nil {
			return "", errors.Wrapf(err, "unable to unquote string %q", v.String())
		}
		return s, nil
	}
	return "", errors.Errorf("unexpected value %T %#v", v, v)
}

func literalStringSlice(v ssa.Value) ([]string, error) {
	sli, ok := v.(*ssa.Slice)
	if !ok {
		return nil, errors.Errorf("expected value of type Slice, got %T", v)
	}
	alloc, ok := sli.X.(*ssa.Alloc)
	if !ok {
		return nil, errors.Errorf("expected value of type Alloc, got %T", v)
	}

	var (
		err    error
		values []string
	)
	walkReferrers(alloc, func(inst ssa.Instruction) bool {
		switch inst := inst.(type) {
		case *ssa.Store:
			var v string
			v, err = literalString(inst.Val)
			if err != nil {
				return false
			}
			values = append(values, v)
		}
		return true
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return values, nil
}

func structType(t types.Type) (*types.Struct, error) {
	switch t := t.(type) {
	default:
		return nil, errors.Errorf("unexpected type %T", t)
	case *types.Pointer:
		return structType(t.Elem())
	case *types.Named:
		return structType(t.Underlying())
	case *types.Struct:
		return t, nil
	}
}

func call(v ssa.Value) (*ssa.Call, error) {
	switch v := v.(type) {
	default:
		return nil, errors.Errorf("unexpected call value type %T %#v", v, v)
	case *ssa.MakeInterface:
		return call(v.X)
	case *ssa.MakeClosure:
		fn, ok := v.Fn.(*ssa.Function)
		if !ok {
			return nil, errors.Errorf("unexpected closure Fn type %T %#v", v.Fn, v.Fn)
		}
		// TODO: only do this for "bound method wrapper for %s"?
		// https://github.com/golang/tools/blob/149740340b5f650cf618fb4e02716f04efac92bb/go/ssa/wrappers.go#L182

		// find first Call
		for _, blk := range fn.Blocks {
			for _, inst := range blk.Instrs {
				switch inst := inst.(type) {
				case *ssa.Call:
					return inst, nil
				}

			}
		}

		return nil, errors.Errorf("unable to find call in closure")
	}
}

func field(v ssa.Value) (*types.Var, error) {
	switch v := v.(type) {
	default:
		return nil, errors.Errorf("unexpected field value type %T %#v", v, v)
	case *ssa.MakeInterface:
		return field(v.X)
	case *ssa.UnOp:
		return field(v.X)
	case *ssa.FieldAddr:
		st, err := structType(v.X.Type())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return st.Field(v.Field), nil
	}
}

func callInterfaceSlice(v ssa.Value) ([]*ssa.Call, error) {
	sli, ok := v.(*ssa.Slice)
	if !ok {
		return nil, errors.Errorf("expected value of type Slice, got %T", v)
	}
	alloc, ok := sli.X.(*ssa.Alloc)
	if !ok {
		return nil, errors.Errorf("expected value of type Alloc, got %T", v)
	}

	var (
		err    error
		values []*ssa.Call
	)
	walkReferrers(alloc, func(inst ssa.Instruction) bool {
		switch inst := inst.(type) {
		case *ssa.Store:
			var c *ssa.Call
			c, err = call(inst.Val)
			if err != nil {
				return false
			}
			values = append(values, c)
		}
		return true
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return values, nil
}

func fieldInterfaceSlice(v ssa.Value) ([]*types.Var, error) {
	sli, ok := v.(*ssa.Slice)
	if !ok {
		return nil, errors.Errorf("expected value of type Slice, got %T", v)
	}
	alloc, ok := sli.X.(*ssa.Alloc)
	if !ok {
		return nil, errors.Errorf("expected value of type Alloc, got %T", v)
	}

	var (
		err    error
		values []*types.Var
	)
	walkReferrers(alloc, func(inst ssa.Instruction) bool {
		switch inst := inst.(type) {
		case *ssa.Store:
			var v *types.Var
			v, err = field(inst.Val)
			if err != nil {
				return false
			}
			values = append(values, v)
		}
		return true
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return values, nil
}

func handleMapField(m *mappingFunc, call ssa.CallInstruction) error {
	if argLen := len(call.Common().Args); argLen != 2 {
		return errors.Errorf("expected 2 args for MapField, found %d", argLen)
	}
	srcField, err := field(call.Common().Args[0])
	if err != nil {
		return errors.WithStack(err)
	}
	dstField, err := field(call.Common().Args[1])
	if err != nil {
		return errors.WithStack(err)
	}
	m.manualMaps[dstField.Name()] = srcField.Name()
	return nil
}

func handleMapWith(m *mappingFunc, call ssa.CallInstruction) error {
	if argLen := len(call.Common().Args); argLen != 1 {
		return errors.Errorf("expected 1 arg for MapWith, found %d", argLen)
	}
	calls, err := callInterfaceSlice(call.Common().Args[0])
	if err != nil {
		return errors.WithStack(err)
	}
	for _, c := range calls {
		m.mapWith = append(m.mapWith, c.Call.StaticCallee())
	}
	return nil
}

func handleRecognizePrefixes(m *mappingFunc, call ssa.CallInstruction) error {
	if argLen := len(call.Common().Args); argLen != 1 {
		return errors.Errorf("expected 1 arg for RecognizePrefixes, found %d", argLen)
	}
	prefixValues, err := literalStringSlice(call.Common().Args[0])
	if err != nil {
		return errors.WithStack(err)
	}
	m.prefixes = append(m.prefixes, prefixValues...)
	return nil
}

func handleIgnoreFields(m *mappingFunc, call ssa.CallInstruction) error {
	if argLen := len(call.Common().Args); argLen != 1 {
		return errors.Errorf("expected 1 arg for IgnoreFields, found %d", argLen)
	}
	ignores, err := fieldInterfaceSlice(call.Common().Args[0])
	if err != nil {
		return errors.WithStack(err)
	}
	ignoreNames := make([]string, 0, len(ignores))
	for _, ig := range ignores {
		ignoreNames = append(ignoreNames, ig.Name())
	}
	m.ignores = append(m.ignores, ignoreNames...)
	return nil
}

func handleCreateMap(call ssa.CallInstruction, f *ssa.Function) (*mappingFunc, error) {
	var err error
	m := &mappingFunc{
		fn:   f,
		name: f.Name(),

		ignores:    []string{},
		manualMaps: map[string]string{},
		prefixes:   []string{},
		mapWith:    []*ssa.Function{},
	}

	src := call.Common().Args[0]
	m.srcName, m.srcType, _, err = param(call, src)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dst := call.Common().Args[1]
	m.dstName, m.dstType, m.dstConstructed, err = param(call, dst)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if m.dstType == nil {
		return nil, errors.Errorf("unable to determine destination type, %T %#v", dst, dst)
	}

	if recv := f.Signature.Recv(); recv != nil {
		// TODO: validate that its src type / name?
		// for now just assuming if there is a receiver its the src
		m.srcReceiver = true
	}

	results := f.Signature.Results()
	if resultsLen := results.Len(); resultsLen > 0 {
		last := results.At(results.Len() - 1)
		if nm, ok := last.Type().(*types.Named); ok {
			if nm.Obj().Name() == "error" && nm.Obj().Pkg() == nil {
				m.errReturned = true
				resultsLen -= 1
			}
		}

		if resultsLen > 1 {
			return nil, errors.Errorf("functions can only return the destination and/or an error, found %d results", results.Len())
		}

		if resultsLen == 1 {
			// TODO: validate its the dest type
			// for now just assuming if there is a result, its the destination
			m.dstReturned = true
		}
	}

	return m, nil
}

func param(call ssa.CallInstruction, v ssa.Value) (name string, ty types.Type, constructed bool, err error) {
	switch v := v.(type) {
	case *ssa.Const:
		return "", v.Type(), true, nil
	case *ssa.UnOp:
		if v.Op == token.MUL {
			name, ty, constructed, err = param(call, v.X)
			if err != nil {
				return "", nil, false, err
			}
			return name, unwrapPointer(ty), constructed, nil
		}
	case *ssa.Parameter:
		return v.Name(), v.Type(), false, nil
	case *ssa.MakeInterface:
		return param(call, v.X)
	case *ssa.Alloc:
		for _, ref := range *v.Referrers() {
			if ref == call {
				continue
			}
			switch ref := ref.(type) {
			case *ssa.Store:
				if ref.Addr == v {
					return param(call, ref.Val)
				}
			}
		}
		return "", v.Type(), true, nil
	}
	return "", nil, false, errors.Errorf("unexpected value %T %#v", v, v)
}

func isTypeMapperCall(inst ssa.CallInstruction) bool {
	callF, ok := inst.Common().Value.(*ssa.Function)
	if !ok {
		return false
	}
	if !strings.HasSuffix(callF.Pkg.Pkg.Path(), "github.com/paultyng/go-typemapper") {
		return false
	}
	return true
}
