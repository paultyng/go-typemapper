package generator

import (
	"fmt"
	"go/types"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/ssa"
)

const BuildTag = "typemapper"

type Generator struct {
	pkgName  string
	ssapkg   *ssa.Package
	comments []string

	cache mappingCache

	files map[string]*jen.File
}

func NewGenerator(ssapkg *ssa.Package, comments ...string) *Generator {
	g := &Generator{
		pkgName:  ssapkg.Pkg.Name(),
		ssapkg:   ssapkg,
		comments: comments,
		files:    map[string]*jen.File{},
	}

	return g
}

func (g *Generator) fileFactory(fileName string) *jen.File {
	if f, ok := g.files[fileName]; ok {
		return f
	}
	f := jen.NewFile(g.pkgName)
	for _, c := range g.comments {
		f.HeaderComment(c)
	}
	g.files[fileName] = f
	return f
}

func (g *Generator) file(fileName string) *jen.File {
	fileName = fmt.Sprintf("%s.generated.go", strings.TrimSuffix(fileName, ".go"))
	return g.fileFactory(fileName)
}

func (g *Generator) testFile(fileName string) *jen.File {
	fileName = fmt.Sprintf("%s.generated_test.go", strings.TrimSuffix(fileName, ".go"))
	return g.fileFactory(fileName)
}

func (g *Generator) AllFiles() []string {
	files := make([]string, 0, len(g.files))
	for fileName := range g.files {
		files = append(files, fileName)
	}
	sort.Strings(files)
	return files
}

func (g *Generator) Render(fileName string, w io.Writer) error {
	file, ok := g.files[fileName]
	if !ok {
		return errors.Errorf("unable to find file %q to render", fileName)
	}
	err := file.Render(w)
	if err != nil {
		return errors.Wrapf(err, "unable to render file %s", fileName)
	}
	return nil
}

func (g *Generator) GenerateMappings() error {
	mfs := mappingCache{}

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

					mf, err := g.parseFunction(ssaF)
					if err != nil {
						return errors.WithStack(err)
					}
					if mf == nil {
						continue
					}

					f := prog.Fset.File(ssaF.Pos())
					_, fileName := filepath.Split(f.Name())
					mf.fileName = fileName

					mfs = append(mfs, mf)
				}
			}
		case *ssa.Function:
			if mem.Name() == "init" || mem.Name() == "main" {
				continue
			}

			mf, err := g.parseFunction(mem)
			if err != nil {
				return errors.WithStack(err)
			}
			if mf == nil {
				continue
			}

			f := prog.Fset.File(mem.Pos())
			_, fileName := filepath.Split(f.Name())
			mf.fileName = fileName

			mfs = append(mfs, mf)
		}
	}

	g.cache = mfs
	sort.Slice(g.cache, func(i, j int) bool {
		if g.cache[i].name < g.cache[j].name {
			return true
		}

		if g.cache[i].name > g.cache[j].name {
			return false
		}

		return g.cache[i].fn.String() < g.cache[j].fn.String()
	})

	err := g.generateMappings()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (g *Generator) generateMappings() error {
	for _, mf := range g.cache {
		// not structs, so do some alternative mapping
		switch {
		default:
			return errors.Errorf("unexpected mapping type, unable to generate function")
		case mf.StructMapping():
			err := g.generateStructMapping(mf)
			if err != nil {
				return errors.WithStack(err)
			}
		case mf.SliceMapping():
			err := g.generateSliceMapping(mf)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return nil
}
