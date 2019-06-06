package typemapper // import "github.com/paultyng/go-typemapper"

const panicNotRuntime = "this should not be invoked at runtime"

// CreateMap initiates a mapping from the src type to the dst type.
func CreateMap(src interface{}, dst interface{}) {
	panic(panicNotRuntime)
}

// RecognizePrefixes tells the map to match fields where the names
// match ignoring certain prefixes.
func RecognizePrefixes(prefix ...string) {
	panic(panicNotRuntime)
}

// MapField tells the map to explicitly match fields that would
// otherwise not match.
func MapField(srcField interface{}, dstField interface{}) {
	panic(panicNotRuntime)
}

// IgnoreFields tells the map to ignore certain destination fields.
func IgnoreFields(dstFields ...interface{}) {
	panic(panicNotRuntime)
}

// MapWith provides additional mapping functions to use for
// type conversions.
func MapWith(mappingFuncs ...interface{}) {
	panic(panicNotRuntime)
}
