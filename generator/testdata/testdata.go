package testdata

type SourceStruct struct {
	StringMatch string
	IntMatch    int
	BoolMatch   bool

	PointerMatch string
	DerefMatch   *string

	TypeAliasMatch string
}

type DestStruct struct {
	StringMatch string
	IntMatch    int
	BoolMatch   bool

	PointerMatch *string
	DerefMatch   string

	TypeAliasMatch stringAlias
}

type stringAlias string
