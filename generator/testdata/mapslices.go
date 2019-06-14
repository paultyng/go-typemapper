// +build typemapper

package testdata

import (
	typemapper "github.com/paultyng/go-typemapper"
)

func MapSliceSrcDestParams(src []string, dst []string) {
	typemapper.CreateMap(src, dst)
}

func MapSliceSrcDestParamsError(src []string, dst []string) error {
	typemapper.CreateMap(src, dst)
	return nil
}

func MapSliceSrcParamsDestConst(src []string) []string {
	var dst []string
	typemapper.CreateMap(src, dst)
	return dst
}

func MapSliceSrcParamsDestConstTypeAlias(src []string) []stringAlias {
	var dst []stringAlias
	typemapper.CreateMap(src, dst)
	return dst
}

func MapSliceSrcParamsDestConstTypeAliasError(src []string) ([]stringAlias, error) {
	var dst []stringAlias
	typemapper.CreateMap(src, dst)
	return dst, nil
}

func MapSliceSrcParamsTypeAliasDestConst(src []stringAlias) []string {
	var dst []string
	typemapper.CreateMap(src, dst)
	return dst
}

func MapSliceSrcParamsTypeAliasDestConstTypeAlias(src []stringAlias) []stringAlias {
	var dst []stringAlias
	typemapper.CreateMap(src, dst)
	return dst
}
