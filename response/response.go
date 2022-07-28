package response

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/yusufsyaifudin/openapidoc/header"
	"github.com/yusufsyaifudin/openapidoc/utils"
	"strings"
)

type bodiesSchema struct {
	schemaName string
	schemaRef  *openapi3.SchemaRef
}

type Response struct {

	// bodies map k = content type, v = struct data
	bodies map[string]body

	// bodiesSchema map k = content type, v = struct data
	bodiesSchema map[string]bodiesSchema

	// headers is the header map of this response
	headers []*header.Header

	// descriptions response description
	descriptions []string
}

// NewResponse only return one openapi3respRef openapi3.ResponseRef.
// If one path can contain multiple response openapi3schema, then it must have different http code.
func NewResponse() *Response {
	return &Response{
		bodies:       map[string]body{},
		headers:      make([]*header.Header, 0),
		descriptions: make([]string, 0),
	}
}

func (r *Response) Body(contentType string, data interface{}, opts ...func(*bodyOpt)) *Response {
	reqData := body{
		data: data,
		opts: &bodyOpt{},
	}

	// override the options
	for _, opt := range opts {
		opt(reqData.opts)
	}

	r.bodies[contentType] = reqData
	return r
}

func (r *Response) BodyWithSchema(contentType, schemaName string, schemaRef *openapi3.SchemaRef) *Response {
	r.bodiesSchema[contentType] = bodiesSchema{
		schemaName: schemaName,
		schemaRef:  schemaRef,
	}
	return r
}

func (r *Response) Header(h *header.Header) *Response {
	r.headers = append(r.headers, h)

	return r
}

// Description will be added to new line each called.
func (r *Response) Description(desc string) *Response {
	r.descriptions = append(r.descriptions, desc)
	return r
}

// Components returns openapi3.Components with the value of following fields:
// * openapi3.Schemas
// * openapi3.Headers
// * openapi3.Responses
func (r *Response) Components(gen *openapi3gen.Generator, responseName, httpCode string) (components openapi3.Components, err error) {

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
	openapi3respRef.Value = &openapi3.Response{}

	// responses should have required property 'description'
	desc := strings.Join(r.descriptions, "\n\n")
	openapi3respRef.Value.Description = &desc

	// response header should write here, because even if we have multiple content type,
	// it will share the same response headers.
	openapi3respRef.Value.Headers = map[string]*openapi3.HeaderRef{}
	for headerKey, headerRefName := range allHeaderRef {
		openapi3respRef.Value.Headers[headerKey] = &openapi3.HeaderRef{
			Ref: headerRefName,
		}
	}

	// same response can have multiple response schema for different content type
	openapi3respRef.Value.Content = make(map[string]*openapi3.MediaType)

	// only one response can be generated.
	// We design that 1 Response is represents only to specific method and path.
	// If user want to add multiple response with different method or path, they must call NewResponse() multiple times.
	openapi3responseBodies := make(map[string]*openapi3.ResponseRef)

	for contentType, bodyPayload := range r.bodies {
		// generate schemaRef and add to the openapi3schema map
		// if bodyPayload is come from the same struct that we defined in some places, then it will use the same schema name
		schemaName := fmt.Sprintf("%T", bodyPayload.data)
		if bodyPayload.opts.withSchemaName != "" {
			schemaName = bodyPayload.opts.withSchemaName
		}

		var schemaRef *openapi3.SchemaRef
		schemaRef, err = gen.NewSchemaRefForValue(bodyPayload.data, nil)
		if err != nil {
			err = fmt.Errorf("error generate openapi3schema response %s: %w", schemaName, err)
			return
		}

		// by default, this struct is just added as is
		openapi3schema[schemaName] = schemaRef

		// refer the created openapi3schema to responses.
		openapi3respRef.Value.Content[contentType] = &openapi3.MediaType{
			Schema: &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", schemaName),
			},
		}

		// responses can have the same content type (i.e: application/json), but different payload
		// i.e: http 200 OK, can have different payload depending on request body.
		// But, please note that if we already define the content type with specific http code, the values will be overrided.
		// i.e: we add 200 OK and application/json with payload {"foo": "bar"}
		// then we add again 200 OK and text/html with payload <html></html>
		// the text/html will be used as the response for http status 200 because it is the latest data we push
		responseName = fmt.Sprintf("%s-%s", responseName, httpCode)
		openapi3responseBodies[responseName] = openapi3respRef
	}

	// current body content type will be overridden by the schema
	for contentType, bodySchema := range r.bodiesSchema {
		// add current schema name to body
		openapi3schema[bodySchema.schemaName] = bodySchema.schemaRef

		// refer the created openapi3schema to responses.
		openapi3respRef.Value.Content[contentType] = &openapi3.MediaType{
			Schema: &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", bodySchema.schemaName),
			},
		}

		// responses can have the same content type (i.e: application/json), but different payload
		// i.e: http 200 OK, can have different payload depending on request body.
		// But, please note that if we already define the content type with specific http code, the values will be overrided.
		// i.e: we add 200 OK and application/json with payload {"foo": "bar"}
		// then we add again 200 OK and text/html with payload <html></html>
		// the text/html will be used as the response for http status 200 because it is the latest data we push
		responseName = fmt.Sprintf("%s-%s", responseName, httpCode)
		openapi3responseBodies[responseName] = openapi3respRef
	}

	// output
	respComponents := openapi3.Components{
		Schemas:   openapi3schema,
		Responses: openapi3responseBodies,
	}

	// merge response components (schema, responses) to output components
	utils.MergeComponents(&components, respComponents)

	return
}
