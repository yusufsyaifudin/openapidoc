package header

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

type Map struct {
	Value       string
	Description string
	Required    bool
}

type Header struct {
	headerMap map[string]Map

	respHeaderRef      map[string]string
	reqHeaderRefParams []string
}

func NewHeader() *Header {
	h := &Header{
		headerMap:          map[string]Map{},
		respHeaderRef:      map[string]string{},
		reqHeaderRefParams: make([]string, 0),
	}
	return h
}

func (h *Header) Add(k string, v Map) *Header {
	h.headerMap[k] = v
	return h
}

// RespHeaderRef returns header actual key name as the map key,
// and string of referenced header as the value.
// Components must be called before call this function, otherwise this will return empty map.
func (h *Header) RespHeaderRef() (refHeader map[string]string) {
	refHeader = h.respHeaderRef
	return
}

// Components returns openapi3.Components with the value of following fields:
// * openapi3.Schemas
// * openapi3.Headers
// * openapi3.Parameters
func (h *Header) Components(gen *openapi3gen.Generator) (components openapi3.Components, err error) {

	// openapi3schema used for schema references
	openapi3schema := make(map[string]*openapi3.SchemaRef)

	// openapi3headers used for response body header references
	openapi3headers := make(map[string]*openapi3.HeaderRef)

	// openapi3params used for parameters references for the generated header.
	// parameters needed for the request body header
	openapi3params := make(map[string]*openapi3.ParameterRef)

	for key, header := range h.headerMap {
		// add each header to schemas with the name headerSchema.{actualHeaderKey}
		schemaName := fmt.Sprintf("headerSchema.%s", key)
		openapi3schema[schemaName] = &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:        "string",
				Title:       fmt.Sprintf("header.%s", key), // this is Schema name, not used as the real Header key
				Example:     header.Value,
				Description: fmt.Sprintf("[header properties] %s", header.Description),
			},
		}

		// use original header name key as the map key and reference it to the schema
		openapi3headers[key] = &openapi3.HeaderRef{
			Value: &openapi3.Header{
				Parameter: openapi3.Parameter{
					Description: header.Description,
					Schema: &openapi3.SchemaRef{
						Ref: fmt.Sprintf("#/components/schemas/%s", schemaName),
					},
				},
			},
		}

		// reference the header name to the openapi3.HeaderRef
		// By using this approach, we can reuse the same header name for many responses.
		h.respHeaderRef[key] = fmt.Sprintf("#/components/headers/%s", key)

		// use prefix headerParam.{actualHeaderKey} as the parameter key, so it similar with the schema name
		paramName := fmt.Sprintf("headerParam.%s", key)
		openapi3params[paramName] = &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				In:          "header",
				Name:        key,
				Description: header.Description,
				Required:    header.Required,
				Schema: &openapi3.SchemaRef{
					Ref: fmt.Sprintf("#/components/schemas/%s", schemaName),
				},
			},
		}

		h.reqHeaderRefParams = append(h.reqHeaderRefParams, fmt.Sprintf("#/components/parameters/%s", paramName))
	}

	components = openapi3.NewComponents()
	components.Schemas = openapi3schema
	components.Headers = openapi3headers
	components.Parameters = openapi3params
	return
}
