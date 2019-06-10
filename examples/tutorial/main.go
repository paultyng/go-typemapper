package main

import (
	"fmt"
)

type Foo struct {
    FieldOne string
    FieldTwo int
}

type Bar struct {
    FieldOne string
    Field2 int
}

func main() {
	src := Foo{
		FieldOne: "one",
		FieldTwo: 2,
	}
	dst := MapFooToBar(src)
	fmt.Printf("Src: %#v\nDst: %#v\n", src, dst)
}