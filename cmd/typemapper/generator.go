package main

import (
	"fmt"
	"go/types"

	. "github.com/dave/jennifer/jen"
	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/go-typemapper/mapper"
)

func isPointer(t types.Type) bool {
	if _, ok := t.(*types.Pointer); ok {
		return true
	}
	return false
}

func (g *generator) generateTypeMapping(name string, sig *types.Signature, src, dst *ssa.Parameter, m *mapper.StructMapper) error {
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
		body = append(body, Commentf("no match for %q", n.Var.Name()))
	}

	params := []Code{}
	for i := 0; i < sig.Params().Len(); i++ {
		p := sig.Params().At(i)
		params = append(params, Id(p.Name()).Add(g.genType(p.Type())))
	}

	s := g.Func()
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

func (g *generator) generateAssignment(src, dst *ssa.Parameter, p mapper.FieldPair) []Code {
	srcExpr := Id(src.Name()).Dot(p.Source.Name())
	if isPointer(p.Destination.Type()) && !isPointer(p.Source.Type()) {
		srcExpr = Op("&").Add(srcExpr)
	}

	return []Code{
		Id(dst.Name()).Dot(p.Destination.Name()).Op("=").Add(srcExpr),
	}
}

func (g *generator) genType(ty types.Type) *Statement {
	switch ty := ty.(type) {
	default:
		fmt.Printf("%T %#v\n", ty, ty)
	case *types.Pointer:
		return Op("*").Add(g.genType(ty.Elem()))
	case *types.Named:
		if ty.Obj().Pkg() == g.typesPkg {
			return Id(ty.Obj().Name())
		}
		return Qual(ty.Obj().Pkg().Path(), ty.Obj().Name())
	}

	panic("?")
}
