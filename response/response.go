package response

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/yusufsyaifudin/openapidoc/header"
	"github.com/yusufsyaifudin/openapidoc/utils"
)

type Config struct {
	Description string
}

type Response struct {
	Config Config

	// bodies map k = content type, v = struct data
	bodies map[string]interface{}

	// headers is the header map of this response
	headers []*header.Header
}

// NewResponse only return one openapi3respRef openapi3.ResponseRef.
// If one path can contain multiple response openapi3schema, then it must have different http code.
func NewResponse(cfg Config) *Response {
	return &Response{
		Config:  cfg,
		bodies:  map[string]interface{}{},
		headers: make([]*header.Header, 0),
	}
}

func (r *Response) Body(contentType string, data interface{}) *Response {
	r.bodies[contentType] = data
	return r
}

func (r *Response) Header(h *header.Header) *Response {
	r.headers = append(r.headers, h)

	return r
}

// Components returns openapi3.Components with the value of following fields:
// * openapi3.Schemas
// * openapi3.Headers
// * openapi3.Responses
func (r *Response) Components(gen *openapi3gen.Generator, responseName string) (components openapi3.Components, err error) {

	components = openapi3.NewComponents()

	// foreach added header key, generate it and save generated headerRef to this map
	// allHeaderRef contains KeyHeader:KeyHeaderRef
	// i.e: Signature:#/components/headers/Signature
	allHeaderRef := make(map[string]string)
	for _, h := range r.headers {
		if h == nil {
			continue
		}

		var headersComponents openapi3.Components
		headersComponents, err = h.Components(gen)
		if err != nil {
			err = fmt.Errorf("generate response header components error: %w", err)
			return
		}

		// merge schema from headers (schemas, params, and headers) to output components
		utils.MergeComponents(&components, headersComponents)

		for headerKey, headerRef := range h.RespHeaderRef() {
			allHeaderRef[headerKey] = headerRef
		}

	}

	openapi3schema := make(map[string]*openapi3.SchemaRef)
	openapi3respRef := &openapi3.ResponseRef{}

	for contentType, bodyPayload := range r.bodies {
		// generate schemaRef and add to the openapi3schema map
		schemaName := fmt.Sprintf("%T", bodyPayload)

		var schemaRef *openapi3.SchemaRef
		schemaRef, err = gen.NewSchemaRefForValue(bodyPayload, nil)
		if err != nil {
			err = fmt.Errorf("error generate openapi3schema response %s: %w", schemaName, err)
			return
		}

		openapi3schema[schemaName] = schemaRef

		if openapi3respRef == nil {
			openapi3respRef = &openapi3.ResponseRef{}
		}

		if openapi3respRef.Value == nil {
			openapi3respRef.Value = &openapi3.Response{}
		}

		// responses should have required property 'description'
		openapi3respRef.Value.Description = &r.Config.Description

		// response header should write here
		if openapi3respRef.Value.Headers == nil {
			openapi3respRef.Value.Headers = map[string]*openapi3.HeaderRef{}
		}

		for headerKey, headerRefName := range allHeaderRef {
			openapi3respRef.Value.Headers[headerKey] = &openapi3.HeaderRef{
				Ref: headerRefName,
			}
		}

		if openapi3respRef.Value.Content == nil {
			openapi3respRef.Value.Content = make(map[string]*openapi3.MediaType)
		}

		// refer the created openapi3schema to responses
		// if openapi3schema responses for the same contentType is already exist then replace it,
		// if contentType is different, then add it.
		openapi3respRef.Value.Content[contentType] = &openapi3.MediaType{
			Schema: &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", schemaName),
			},
		}

	}

	// only one response can be generated.
	// We design that 1 Response is represents only to specific method and path.
	// If user want to add multiple response with different method or path, they must call NewResponse() multiple times.
	openapi3responseBodies := make(map[string]*openapi3.ResponseRef)
	openapi3responseBodies[responseName] = openapi3respRef

	// output
	respComponents := openapi3.Components{
		Schemas:   openapi3schema,
		Responses: openapi3responseBodies,
	}

	// merge response components (schema, responses) to output components
	utils.MergeComponents(&components, respComponents)

	return
}
