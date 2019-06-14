package generator

import (
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/go-typemapper/mapper"
)

type mappingFunc struct {
	fn       *ssa.Function
	name     string
	fileName string

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
	mapWith    []*ssa.Function
}

func (mf *mappingFunc) MapWith(cache mappingCache) mappingCache {
	mw := mappingCache{}
	for _, mwf := range mf.mapWith {
		for _, c := range cache {
			if c.fn == mwf {
				mw = append(mw, c)
				break
			}
		}
	}
	return mw
}

type mappingCache []*mappingFunc

func (mf *mappingFunc) SliceMapping() bool {
	src := unwrapSlice(mf.srcType)
	dst := unwrapSlice(mf.dstType)
	return src != nil && dst != nil
}

func (mf *mappingFunc) StructMapping() bool {
	src := unwrapStruct(mf.srcType)
	dst := unwrapStruct(mf.dstType)
	return src != nil && dst != nil
}

func (mf *mappingFunc) Mapper() *mapper.StructMapper {
	m := mapper.NewStructMapper(mf.srcType, mf.dstType)
	if m == nil {
		return nil
	}

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
