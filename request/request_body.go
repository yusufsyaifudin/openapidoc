package request

import "strings"

type bodyOpt struct {
	withPrefixRequestName bool
	withSchemaName        string
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

// UniquePerRequest if set to true, then we will add hash request method:path to the schema name prefix.
// This to make sure if we add the same struct to multiple request, then each request will have different schema.
// This useful when you have general struct as parent wrapper. For example, you have this struct:
//
// type ReqWrapper struct {
// 	 Data interface{} `json:"data"`
// }
//
// and `data` field will be assigned to different struct on each request.
// I.e: req1 has body: {"data": {"username": "user-name"}}
// I.e: req2 has body: {"data": {"teamName": "team-name"}}
// req1 and req2 can share the same 'wrapper' struct such as ReqWrapper, and calling this UniquePerRequest
// will ensure that those two payload is generated with different schema name.
func UniquePerRequest() func(*bodyOpt) {
	return func(o *bodyOpt) {
		o.withPrefixRequestName = true
	}
}
