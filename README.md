# Go TypeMapper

**main.go**
```
package main

type Foo struct {
    FieldOne string
    FieldTwo int
}

type Bar struct {
    FieldOne string
    Field2 int
}
```

**typemapper.go**
```
// +build typemapper

package main

func MapFooToBar(src Foo, dst Bar) error {
    return nil
}
```

Running `typemapper` will generate:

**typemapper.generated.go**
```
// +build !typemapper

package main

func MapFooToBar(src Foo, dst Bar) error {
    dst.FieldOne = src.FieldOne
    // no match for "FieldTwo"
    return nil
}
```

## TODO

* [ ] Generate test file with failing unit test if not fully mapped