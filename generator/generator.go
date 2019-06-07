package generator

import (
	"go/types"
	"io"

	"github.com/dave/jennifer/jen"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/ssa"
)

const BuildTag = "typemapper"

type Generator struct {
	file     *jen.File
	testFile *jen.File

	ssapkg *ssa.Package
}

func NewGenerator(ssapkg *ssa.Package, comments ...string) *Generator {
	g := &Generator{
		file:     jen.NewFile(ssapkg.Pkg.Name()),
		testFile: jen.NewFile(ssapkg.Pkg.Name()),

		ssapkg: ssapkg,
	}

	for _, c := range comments {
		g.file.HeaderComment(c)
	}

	return g
}

func (g *Generator) Render(w io.Writer) error {
	return g.file.Render(w)
}

func (g *Generator) GenerateMappings() error {
	prog := g.ssapkg.Prog
	for _, mem := range g.ssapkg.Members {
		switch mem := mem.(type) {
		case *ssa.Type:
			for _, ms := range []*types.MethodSet{
				prog.MethodSets.MethodSet(mem.Type()),
				prog.MethodSets.MethodSet(types.NewPointer(mem.Type())),
			} {
				for i := 0; i < ms.Len(); i++ {
					sel := ms.At(i)
					ssaF := prog.MethodValue(sel)
					err := g.MapFunction(ssaF)
					if err != nil {
						return errors.WithStack(err)
					}
				}
			}
		case *ssa.Function:
			err := g.MapFunction(mem)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
