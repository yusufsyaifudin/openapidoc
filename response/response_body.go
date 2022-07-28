package response

import "strings"

type bodyOpt struct {
	withSchemaName string
}

type body struct {
	data interface{}
	opts *bodyOpt
}

// WithSchemaName to override generated schema name from struct name to this name.
// For example, if you don't specify this it will use package.StructName as the schema name,
// but if you define this, then we will use this as the schema name.
func WithSchemaName(name string) func(*bodyOpt) {
	return func(o *bodyOpt) {
		o.withSchemaName = strings.ReplaceAll(strings.TrimSpace(name), "\n", "")
	}
}
