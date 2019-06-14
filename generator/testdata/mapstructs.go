// +build typemapper

package testdata

import (
	typemapper "github.com/paultyng/go-typemapper"
)

func MapStructSrcDestParams(src SourceStruct, dst *DestStruct) {
	typemapper.CreateMap(src, dst)
}

func MapStructPtrSrcDestParams(src *SourceStruct, dst *DestStruct) {
	typemapper.CreateMap(src, dst)
}

func MapStructSrcParamsDestConst(src SourceStruct) DestStruct {
	var dst DestStruct
	typemapper.CreateMap(src, dst)
	return dst
}

func MapStructSrcParamsDestConstError(src SourceStruct) (DestStruct, error) {
	var dst DestStruct
	typemapper.CreateMap(src, dst)
	return dst, nil
}

func MapStructSrcParamsPtrDestConst(src SourceStruct) *DestStruct {
	var dst *DestStruct
	typemapper.CreateMap(src, dst)
	return dst
}

func MapStructPtrSrcParamsDestConst(src *SourceStruct) DestStruct {
	var dst DestStruct
	typemapper.CreateMap(src, dst)
	return dst
}

func MapStructSrcDstParamsError(src SourceStruct, dst *DestStruct) error {
	typemapper.CreateMap(src, dst)
	return nil
}

func (src SourceStruct) MapStructSrcRecvDestParams(dst *DestStruct) {
	typemapper.CreateMap(src, dst)
}

func (src SourceStruct) MapStructSrcRecvDestConst() DestStruct {
	var dst DestStruct
	typemapper.CreateMap(src, dst)
	return dst
}

func (src *SourceStruct) MapStructPtrSrcRecvDestConst() DestStruct {
	var dst DestStruct
	typemapper.CreateMap(src, dst)
	return dst
}

func (src SourceStruct) MapStructSrcRecvPtrDestConst() *DestStruct {
	var dst *DestStruct
	typemapper.CreateMap(src, dst)
	return dst
}

func (src *SourceStruct) MapStructPtrSrcRecvPtrDestConst() *DestStruct {
	var dst *DestStruct
	typemapper.CreateMap(src, dst)
	return dst
}

func (src *SourceStruct) MapStructPtrSrcRecvPtrDestConstError() (*DestStruct, error) {
	var dst *DestStruct
	typemapper.CreateMap(src, dst)
	return dst, nil
}
