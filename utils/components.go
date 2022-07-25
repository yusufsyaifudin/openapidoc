package utils

import "github.com/getkin/kin-openapi/openapi3"

// MergeComponents merge component from src to dst.
func MergeComponents(dst *openapi3.Components, src openapi3.Components) {
	if dst == nil {
		return
	}

	// merge schema
	if dst.Schemas == nil {
		dst.Schemas = make(map[string]*openapi3.SchemaRef)
	}

	for schemaName, schemaRef := range src.Schemas {
		dst.Schemas[schemaName] = schemaRef
	}

	// merge header
	if dst.Headers == nil {
		dst.Headers = make(map[string]*openapi3.HeaderRef)
	}

	for headerName, headerRef := range src.Headers {
		dst.Headers[headerName] = headerRef
	}

	// merge parameters
	if dst.Parameters == nil {
		dst.Parameters = make(map[string]*openapi3.ParameterRef)
	}

	for paramName, paramRef := range src.Parameters {
		dst.Parameters[paramName] = paramRef
	}

	// merge request body
	if dst.RequestBodies == nil {
		dst.RequestBodies = make(map[string]*openapi3.RequestBodyRef)
	}

	for reqBodyName, reqBodyRef := range src.RequestBodies {
		dst.RequestBodies[reqBodyName] = reqBodyRef
	}

	// merge responses
	if dst.Responses == nil {
		dst.Responses = make(map[string]*openapi3.ResponseRef)
	}

	for respName, respRef := range src.Responses {
		dst.Responses[respName] = respRef
	}

}
