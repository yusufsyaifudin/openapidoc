package request

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/yusufsyaifudin/openapidoc/header"
	"github.com/yusufsyaifudin/openapidoc/utils"
	"reflect"
	"strings"
)

type PathParam struct {
	Name        string
	Value       interface{}
	Description string
}

type Request struct {
	// bodies map k = content type, v = struct data
	bodies map[string]interface{}

	// headers is the header map of this response
	headers []*header.Header

	// pathParams path parameters
	pathParams []PathParam

	// descriptions request description
	descriptions []string

	// required to mark whether this body payload is required or not
	required bool
}

func NewRequest() *Request {
	return &Request{
		bodies:     map[string]interface{}{},
		headers:    make([]*header.Header, 0),
		pathParams: make([]PathParam, 0),
	}
}

func (r *Request) Body(contentType string, data interface{}) *Request {
	r.bodies[contentType] = data
	return r
}

func (r *Request) Header(h *header.Header) *Request {
	r.headers = append(r.headers, h)
	return r
}

func (r *Request) PathParams(params ...PathParam) *Request {
	for _, param := range params {
		r.pathParams = append(r.pathParams, param)
	}

	return r
}

// Description will be added to new line each called.
func (r *Request) Description(desc string) *Request {
	r.descriptions = append(r.descriptions, desc)
	return r
}

// Required must be called once to mark whether this Request is required or not.
func (r *Request) Required(required bool) *Request {
	r.required = required
	return r
}

func (r *Request) Components(gen *openapi3gen.Generator, requestName string) (components openapi3.Components, err error) {

	components = openapi3.NewComponents()

	for _, h := range r.headers {
		if h == nil {
			continue
		}

		var headersComponents openapi3.Components
		headersComponents, err = h.Components(gen)
		if err != nil {
			err = fmt.Errorf("generate request header components error: %w", err)
			return
		}

		// merge schema from headers (schemas, params, and headers) to output components
		utils.MergeComponents(&components, headersComponents)
	}

	// generate path parameters
	// openapi3params used for parameters references for the path parameters
	// refer to https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#parameterObject
	openapi3params := make(map[string]*openapi3.ParameterRef)
	for _, param := range r.pathParams {
		paramName := fmt.Sprintf("pathParam.%s.%s", requestName, param.Name)

		// params is only simple value, and must not contain array or object
		paramType := "string"
		switch reflect.TypeOf(param.Value).Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			paramType = "integer"

		case reflect.Float32, reflect.Float64:
			paramType = "number"

		case reflect.Bool:
			paramType = "boolean"
		}

		openapi3params[paramName] = &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				In:          "path",
				Name:        param.Name,
				Description: param.Description,
				Example:     param.Value,
				Required:    true, // always true for in=path
				Style:       "simple",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: paramType,
					},
				},
			},
		}
	}

	openapi3schema := make(map[string]*openapi3.SchemaRef)

	// Define openapi3bodyRef here to ensure that this Request support multiple content type with different payload.
	// If openapi3bodyRef is first generated, then it will have Value.Content = nil
	openapi3bodyRef := &openapi3.RequestBodyRef{}
	openapi3bodyRef.Value = &openapi3.RequestBody{}
	openapi3bodyRef.Value.Required = r.required
	openapi3bodyRef.Value.Description = strings.Join(r.descriptions, "\n\n")
	openapi3bodyRef.Value.Content = make(map[string]*openapi3.MediaType)

	for contentType, bodyPayload := range r.bodies {
		// generate schemaRef and add to the schema map
		schemaName := fmt.Sprintf("%T", bodyPayload)

		var schemaRef *openapi3.SchemaRef
		schemaRef, err = gen.NewSchemaRefForValue(bodyPayload, nil)
		if err != nil {
			err = fmt.Errorf("error generate schema request %s: %w", schemaName, err)
			return
		}

		openapi3schema[schemaName] = schemaRef

		// refer the created schema to openapi3RequestBodies
		// if schema openapi3RequestBodies for the same contentType is already exist then replace it,
		// if contentType is different, then add it.
		openapi3bodyRef.Value.Content[contentType] = &openapi3.MediaType{
			Schema: &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", schemaName),
			},
		}

	}

	// this Request only returns one map value *openapi3.RequestBodyRef.
	// We design that 1 Request is represent only to specific method and path.
	// If user want to add multiple request with different method or path, they must call NewRequest() multiple times.
	openapi3RequestBodies := make(map[string]*openapi3.RequestBodyRef)
	openapi3RequestBodies[requestName] = openapi3bodyRef

	// output
	reqComponents := openapi3.Components{
		Schemas:       openapi3schema,
		RequestBodies: openapi3RequestBodies,
		Parameters:    openapi3params,
	}

	utils.MergeComponents(&components, reqComponents)
	return
}
