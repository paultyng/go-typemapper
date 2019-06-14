package generator

import (
	"fmt"
	"go/types"

	. "github.com/dave/jennifer/jen"
	"github.com/pkg/errors"

	"github.com/paultyng/go-typemapper/mapper"
)

const (
	defaultSrcName = "src"
	defaultDstName = "dst"
)

func (g *Generator) generateSliceMapping(mf *mappingFunc) error {
	srcName := mf.srcName
	if srcName == "" {
		srcName = defaultSrcName
	}

	dstName := mf.dstName
	if dstName == "" {
		dstName = defaultDstName
	}

	returnsSuccess := []Code{}

	if mf.dstReturned {
		returnsSuccess = append(returnsSuccess, Id(dstName))
	}
	if mf.errReturned {
		returnsSuccess = append(returnsSuccess, Nil())
	}

	returnSuccess := Return(returnsSuccess...)

	body := []Code{}

	switch {
	case !mf.dstConstructed:
		body = append(body,
			Id(dstName).Op("=").Id(dstName).Index(Op(":").Lit(0)),
		)
	case mf.dstConstructed:
		// constructed but not a pointer type (no `new`)
		body = append(body,
			Var().Id(dstName).Add(g.genType(mf.dstType)),
		)
	}

	// TODO: nil checks
	srcElemType := unwrapSlice(mf.srcType).Elem()
	dstElemType := unwrapSlice(mf.dstType).Elem()
	iter := "x"
	srcExpr := g.convertSourceTo(mf.MapWith(g.cache), Id(iter), srcElemType, dstElemType)

	body = append(body,
		For(List(Id("_"), Id(iter)).Op(":=").Range().Id(srcName)).Block(
			Id(dstName).Op("=").Append(Id(dstName), srcExpr),
		),
	)

	params := []Code{}
	srcParam := Id(srcName).Add(g.genType(mf.srcType))

	s := g.file(mf.fileName).Func()
	if mf.srcReceiver {
		s = s.Params(srcParam.Clone())
	} else {
		params = append(params, srcParam.Clone())
	}

	if !mf.dstConstructed {
		params = append(params, Id(dstName).Add(g.genType(mf.dstType)))
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

	g.testFile(mf.fileName).Func().Id(fmt.Sprintf("Test%s", mf.name)).Params(Id("t").Op("*").Qual("testing", "T")).Block()

	return nil
}

func (g *Generator) generateStructMapping(mf *mappingFunc) error {
	m := mf.Mapper()

	if m == nil {
		return errors.Errorf("unable to create struct mapping for %v and %v", mf.srcType, mf.dstType)
	}

	mapConfig := m.Map()

	fileName := mf.fileName

	srcName := mf.srcName
	if srcName == "" {
		srcName = defaultSrcName
	}

	dstName := mf.dstName
	if dstName == "" {
		dstName = defaultDstName
	}

	returnsSuccess := []Code{}

	if mf.dstReturned {
		returnsSuccess = append(returnsSuccess, Id(dstName))
	}
	if mf.errReturned {
		returnsSuccess = append(returnsSuccess, Nil())
	}

	returnSuccess := Return(returnsSuccess...)

	body := []Code{}

	switch {
	case mf.dstConstructed && isPointer(mf.dstType):
		// construct with `new`
		body = append(body,
			Id(dstName).Op(":=").New(g.genType(unwrapPointer(mf.dstType))),
		)
	case mf.dstConstructed:
		// constructed but not a pointer type (no `new`)
		body = append(body,
			Id(dstName).Op(":=").Add(g.genType(mf.dstType)).Values(),
		)
	case isPointer(mf.dstType):
		// pointer but not constructed
		body = append(body, If(Id(dstName).Op("==").Nil()).Block(
			returnSuccess.Clone(),
		))
	}

	for _, p := range mapConfig.Pairs {
		body = append(body, g.generateFieldAssignment(mf.MapWith(g.cache), srcName, dstName, p)...)
	}
	for _, n := range mapConfig.NoMatch {
		body = append(body, Commentf("no match for %q", n.Name()))
	}

	params := []Code{}
	srcParam := Id(srcName).Add(g.genType(mf.srcType))

	s := g.file(fileName).Func()
	if mf.srcReceiver {
		s = s.Params(srcParam.Clone())
	} else {
		params = append(params, srcParam.Clone())
	}

	if !mf.dstConstructed {
		params = append(params, Id(dstName).Add(g.genType(mf.dstType)))
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

func (g *Generator) callMapWith(mapWith *mappingFunc, srcExpr *Statement) *Statement {
	if mapWith.errReturned {
		panic("err return values not yet supported for MapWith")
	}
	if !mapWith.dstConstructed {
		panic("destination parameters not yet supported for MapWith")
	}
	if mapWith.srcReceiver {
		return srcExpr.Dot(mapWith.name).Params()
	}

	return Id(mapWith.name).Params(srcExpr)
}

func (g *Generator) convertSourceTo(mapWith mappingCache, srcExpr *Statement, srcType, dstType types.Type) *Statement {
	for _, mw := range mapWith {
		if !types.AssignableTo(dstType, mw.dstType) {
			continue
		}

		if types.AssignableTo(srcType, mw.srcType) {
			return g.callMapWith(mw, srcExpr)
		}

		// TODO: handle multiple srcTypes when pointer receiver type?
		if mw.srcReceiver && isPointer(mw.srcType) && types.AssignableTo(srcType, unwrapPointer(mw.srcType)) {
			return g.callMapWith(mw, srcExpr)
		}
	}

	srcExpr = srcExpr.Clone()
	for !types.AssignableTo(srcType, dstType) {
		_, srcPtr := srcType.(*types.Pointer)
		_, dstPtr := dstType.(*types.Pointer)
		_, srcNamed := srcType.(*types.Named)
		_, dstNamed := dstType.(*types.Named)
		switch {
		case dstPtr:
			srcExpr = Op("&").Add(srcExpr)

			dstType = dstType.(*types.Pointer).Elem()
			continue
		case srcPtr:
			srcExpr = Op("*").Add(srcExpr)

			srcType = srcType.(*types.Pointer).Elem()
			continue
		case dstNamed:
			srcExpr = g.genType(dstType).Params(srcExpr)

			dstType = dstType.(*types.Named).Underlying()
			continue
		case srcNamed:
			srcExpr = g.genType(dstType).Params(srcExpr)

			srcType = srcType.(*types.Named).Underlying()
			continue
		}
		break
	}

	// if !types.AssignableTo(srcType, dstType) {
	// 	// TODO: return error here?
	// }

	return srcExpr
}

func (g *Generator) generateFieldAssignment(mapWith mappingCache, srcName, dstName string, p mapper.FieldPair) []Code {
	srcExpr := Id(srcName).Dot(p.Source.Name())
	srcExpr = g.convertSourceTo(mapWith, srcExpr, p.Source.Type(), p.Destination.Type())

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
	case *types.Basic:
		return Id(ty.Name())
	case *types.Slice:
		return Index().Add(g.genType(ty.Elem()))
	case *types.Map:
		return Map(g.genType(ty.Key())).Add(g.genType(ty.Elem()))
	}

	panic("?")
}
