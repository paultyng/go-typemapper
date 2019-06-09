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

const (
	defaultSrcName = "src"
	defaultDstName = "dst"
)

type mappingFunc struct {
	name string

	srcName     string
	srcType     types.Type
	srcReceiver bool

	dstName        string
	dstType        types.Type
	dstConstructed bool
	dstReturned    bool

	errReturned bool

	prefixes   []string
	ignores    []string
	manualMaps map[string]string
}

func (mf *mappingFunc) Mapper() *mapper.StructMapper {
	m := mapper.NewStructMapper(mf.srcType, mf.dstType)
	if len(mf.prefixes) > 0 {
		m = m.RecognizePrefixes(mf.prefixes...)
	}
	if len(mf.ignores) > 0 {
		m = m.IgnoreFields(mf.ignores...)
	}
	if len(mf.manualMaps) > 0 {
		for dst, src := range mf.manualMaps {
			m = m.MapField(src, dst)
		}
	}
	return m
}

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
				}
			}
		}
	}

	return m, nil
}

func (g *Generator) MapFunction(fileName string, f *ssa.Function) error {
	mf, err := g.parseFunction(f)
	if err != nil {
		return errors.WithStack(err)
	}
	if mf == nil {
		return nil
	}

	if mf.srcName == "" {
		mf.srcName = defaultSrcName
	}

	if mf.dstName == "" {
		mf.dstName = defaultDstName
	}

	return g.generateTypeMapping(fileName, mf)
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
		name: f.Name(),

		ignores:    []string{},
		manualMaps: map[string]string{},
		prefixes:   []string{},
	}

	src := call.Common().Args[0]
	m.srcName, m.srcType, _, err = param(src)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dst := call.Common().Args[1]
	m.dstName, m.dstType, m.dstConstructed, err = param(dst)
	if err != nil {
		return nil, errors.WithStack(err)
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

func param(v ssa.Value) (name string, ty types.Type, constructed bool, err error) {
	switch v := v.(type) {
	case *ssa.Parameter:
		return v.Name(), v.Type(), false, nil
	case *ssa.MakeInterface:
		return param(v.X)
	case *ssa.Alloc:
		// alloc is only possible for destination type
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

func (g *Generator) generateTypeMapping(fileName string, mf *mappingFunc) error {
	m := mf.Mapper()
	mapConfig := m.Map()

	returnsSuccess := []Code{}

	if mf.dstReturned {
		returnsSuccess = append(returnsSuccess, Id(mf.dstName))
	}
	if mf.errReturned {
		returnsSuccess = append(returnsSuccess, Nil())
	}

	returnSuccess := Return(returnsSuccess...)

	body := []Code{}

	if mf.dstConstructed {
		body = append(body,
			Id(mf.dstName).Op(":=").New(g.genType(unwrapPointer(mf.dstType))),
		)
	} else {
		if isPointer(mf.dstType) {
			body = append(body, If(Id(mf.dstName).Op("==").Nil()).Block(
				returnSuccess.Clone(),
			))
		}
	}

	for _, p := range mapConfig.Pairs {
		body = append(body, g.generateAssignment(mf.srcName, mf.dstName, p)...)
	}
	for _, n := range mapConfig.NoMatch {
		body = append(body, Commentf("no match for %q", n.Name()))
	}

	// TODO: generate unit test code

	params := []Code{}
	srcParam := Id(mf.srcName).Add(g.genType(mf.srcType))

	s := g.file(fileName).Func()
	if mf.srcReceiver {
		s = s.Params(srcParam.Clone())
	} else {
		params = append(params, srcParam.Clone())
	}

	if !mf.dstConstructed {
		params = append(params, Id(mf.dstName).Add(g.genType(mf.dstType)))
	}

	s = s.Id(mf.name).Params(params...)

	switch {
	case mf.dstReturned && mf.errReturned:
		s = s.Params(g.genType(mf.dstType), Error())
	case mf.errReturned:
		s = s.Error()
	case mf.dstReturned:
		s = s.Add(g.genType(mf.dstType))
	}

	body = append(body, returnSuccess.Clone())
	s.Block(body...)

	//generate test func
	testBody := []Code{}
	noMatchNames := []string{}
	for _, n := range mapConfig.NoMatch {
		noMatchNames = append(noMatchNames, n.Name())
	}
	if len(noMatchNames) > 0 {
		testBody = append(testBody,
			Id("t").Dot("Fatal").Params(Lit(fmt.Sprintf("no mapping for: %v", noMatchNames))),
		)
	}

	g.testFile(fileName).Func().Id(fmt.Sprintf("Test%s", mf.name)).Params(Id("t").Op("*").Qual("testing", "T")).Block(testBody...)
	return nil
}

func (g *Generator) generateAssignment(srcName, dstName string, p mapper.FieldPair) []Code {
	srcExpr := Id(srcName).Dot(p.Source.Name())
	if p.Destination.IsPointer() && !p.Source.IsPointer() {
		srcExpr = Op("&").Add(srcExpr)
	}

	return []Code{
		Id(dstName).Dot(p.Destination.Name()).Op("=").Add(srcExpr),
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
