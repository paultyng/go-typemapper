package generator

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/go-typemapper/mapper"
)

type nameAndType interface {
	Name() string
	Type() types.Type
}

type dstType struct {
	ty types.Type
}

var _ nameAndType = &dstType{}

func (d *dstType) Name() string {
	return "dst"
}

func (d *dstType) Type() types.Type {
	return d.ty
}

func (g *Generator) MapFunction(f *ssa.Function) error {
	var (
		m   *mapper.StructMapper
		src nameAndType
		dst nameAndType
		err error
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
					// not a mapping function
					return nil
					// return errors.Errorf("expected *ssa.Function, got %T", inst.Common().Value)
				}

				if m == nil {
					if callF.Name() != "CreateMap" {
						// not a mapping function
						return nil
						//return errors.Errorf("expected call to CreateMap, but got %s", callF.Name())
					}
					m, src, dst, err = handleCreateMap(inst)
					if err != nil {
						return errors.WithStack(err)
					}
					continue
				}

				// TODO: do this via map of funcs?
				switch callF.Name() {
				default:
					return errors.Errorf("unexpected typemapper call %s", callF.Name())
				case "RecognizePrefixes":
					err = handleRecognizePrefixes(m, inst)
					if err != nil {
						return errors.WithStack(err)
					}
				case "MapField":
					err = handleMapField(m, inst)
					if err != nil {
						return errors.WithStack(err)
					}
				case "IgnoreFields":
					err = handleIgnoreFields(m, inst)
					if err != nil {
						return errors.WithStack(err)
					}
				}
			}
		}
	}

	return g.generateTypeMapping(f.Name(), f.Signature, src, dst, m)
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

func field(v ssa.Value) (*types.Var, error) {
	switch v := v.(type) {
	default:
		return nil, errors.Errorf("unexpected value type %T", v)
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

func handleMapField(m *mapper.StructMapper, call ssa.CallInstruction) error {
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
	m.MapField(srcField.Name(), dstField.Name())
	return nil
}

func handleRecognizePrefixes(m *mapper.StructMapper, call ssa.CallInstruction) error {
	if argLen := len(call.Common().Args); argLen != 1 {
		return errors.Errorf("expected 1 arg for RecognizePrefixes, found %d", argLen)
	}
	prefixValues, err := literalStringSlice(call.Common().Args[0])
	if err != nil {
		return errors.WithStack(err)
	}
	m.RecognizePrefixes(prefixValues...)
	return nil
}

func handleIgnoreFields(m *mapper.StructMapper, call ssa.CallInstruction) error {
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
	m.IgnoreFields(ignoreNames...)
	return nil
}

func handleCreateMap(call ssa.CallInstruction) (m *mapper.StructMapper, srcP, dstP nameAndType, err error) {
	src := call.Common().Args[0]
	srcP, err = param(src)
	if err != nil {
		return nil, nil, nil, err
	}
	dst := call.Common().Args[1]
	dstP, err = param(dst)
	if err != nil {
		return nil, nil, nil, err
	}

	m = mapper.NewStructMapper(srcP.Type(), dstP.Type())
	return m, srcP, dstP, nil
}

func param(v ssa.Value) (nameAndType, error) {
	switch v := v.(type) {
	case *ssa.Parameter:
		return v, nil
	case *ssa.MakeInterface:
		return param(v.X)
	case *ssa.Alloc:
		// alloc is only possible for destination type
		return &dstType{
			ty: v.Type(),
		}, nil
	}
	return nil, errors.Errorf("unexpected value %T %#v", v, v)
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

func (g *Generator) generateTypeMapping(name string, sig *types.Signature, src, dst nameAndType, m *mapper.StructMapper) error {
	mapConfig := m.Map()

	// options are zero or one results, and if one, its `error`
	isErrorResult := sig.Results().Len() == 1

	returnSuccess := Return()
	if isErrorResult {
		returnSuccess = Return(Nil())
	}

	body := []Code{}

	if isPointer(dst.Type()) {
		body = append(body, If(Id(dst.Name()).Op("==").Nil()).Block(
			returnSuccess.Clone(),
		))
	}

	for _, p := range mapConfig.Pairs {
		body = append(body, g.generateAssignment(src, dst, p)...)
	}
	for _, n := range mapConfig.NoMatch {
		body = append(body, Commentf("no match for %q", n.Name()))
	}

	params := []Code{}
	for i := 0; i < sig.Params().Len(); i++ {
		p := sig.Params().At(i)
		params = append(params, Id(p.Name()).Add(g.genType(p.Type())))
	}

	s := g.file.Func()
	if sig.Recv() != nil {
		s = s.Params(Id(sig.Recv().Name()).Add(g.genType(sig.Recv().Type())))
	}
	s = s.Id(name).Params(params...)
	if isErrorResult {
		s = s.Error()
	}
	body = append(body, returnSuccess.Clone())
	s.Block(body...)
	return nil
}

func (g *Generator) generateAssignment(src, dst nameAndType, p mapper.FieldPair) []Code {
	srcExpr := Id(src.Name()).Dot(p.Source.Name())
	if p.Destination.IsPointer() && !p.Source.IsPointer() {
		srcExpr = Op("&").Add(srcExpr)
	}

	return []Code{
		Id(dst.Name()).Dot(p.Destination.Name()).Op("=").Add(srcExpr),
	}
}

func (g *Generator) genType(ty types.Type) *Statement {
	switch ty := ty.(type) {
	default:
		fmt.Printf("%T %#v\n", ty, ty)
	case *types.Pointer:
		return Op("*").Add(g.genType(ty.Elem()))
	case *types.Named:
		if ty.Obj().Pkg() == g.ssapkg.Pkg {
			return Id(ty.Obj().Name())
		}
		return Qual(ty.Obj().Pkg().Path(), ty.Obj().Name())
	}

	panic("?")
}
