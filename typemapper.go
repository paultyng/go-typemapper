package typemapper // import "github.com/paultyng/go-typemapper"

const panicNotRuntime = "this should not be invoked at runtime"

func CreateMap(src interface{}, dst interface{}) {
	panic(panicNotRuntime)
}

func RecognizePrefixes(prefix... string) {
	panic(panicNotRuntime)
}

func MapField(srcField interface{}, dstField interface{}) {
	panic(panicNotRuntime)
}

func IgnoreFields(dstFields... interface{}) {
	panic(panicNotRuntime)
}