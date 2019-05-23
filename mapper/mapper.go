package mapper

import (
	"go/types"
)

func unwrapPointer(v types.Type) types.Type {
	if p, ok := v.(*types.Pointer); ok {
		return p.Elem()
	}
	return v
}

func Map(src, dst types.Type) (MapConfiguration, error) {
	//fmt.Printf("Src %T %#v\nDst %T %#v\n", src, src, dst, dst)

	structSrc := unwrapPointer(src).(*types.Named).Underlying().(*types.Struct)
	structDst := unwrapPointer(dst).(*types.Named).Underlying().(*types.Struct)

	sm := &structMapper{
		prefix: "Service",
	}

	config := sm.Map(structSrc, structDst)

	return config, nil
}
