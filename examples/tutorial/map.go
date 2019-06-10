// +build typemapper

package main

import (
    "github.com/paultyng/go-typemapper"
)

func MapFooToBar(src Foo) Bar {
    dst := Bar{}
    typemapper.CreateMap(src, dst)
    return dst
}
