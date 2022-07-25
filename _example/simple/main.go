package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
	"log"
)

func main() {

	desc := "Example description"

	x := openapi3.Components{
		Parameters: map[string]*openapi3.ParameterRef{
			"key": {
				Value: &openapi3.Parameter{
					In:          "header",
					Name:        "Signature",
					Description: "signature of the request",
					Schema: &openapi3.SchemaRef{
						Ref: "#/components/schemas/signatureHeaderRef",
					},
				},
			},
		},
		Headers: map[string]*openapi3.HeaderRef{
			"key": {
				Value: &openapi3.Header{
					Parameter: openapi3.Parameter{
						Description: "signature of the response",
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/schemas/signatureHeaderRef",
						},
					},
				},
			},
		},
		Schemas: map[string]*openapi3.SchemaRef{
			"signatureHeaderRef": {
				Value: &openapi3.Schema{
					Type:        "string",
					Title:       "Sign",
					Description: "signature...",
					Example:     "n4tst5f160l9dd034vd3669h0000gp",
				},
			},
			"reqRef": {
				Value: &openapi3.Schema{
					Title:    "request struct",
					Type:     "object",
					Required: []string{"id", "name"},
					Properties: map[string]*openapi3.SchemaRef{
						"id": {
							Value: &openapi3.Schema{
								Type:    "integer",
								Format:  "int64",
								Example: "123",
							},
						},
						"name": {
							Value: &openapi3.Schema{
								Type:    "string",
								Example: "my user name",
							},
						},
					},
				},
			},
		},
		RequestBodies: map[string]*openapi3.RequestBodyRef{
			"xxx": {
				Value: &openapi3.RequestBody{
					Description: "Body request for xxx",
					Required:    true,
					Content: map[string]*openapi3.MediaType{
						"application/json": {
							Schema: &openapi3.SchemaRef{
								Ref: "#/components/schemas/reqRef",
							},
						},
					},
				},
			},
		},
		Responses: map[string]*openapi3.ResponseRef{
			"respxxx": {
				Value: &openapi3.Response{
					Description: &desc,
					Headers: map[string]*openapi3.HeaderRef{
						"Signature": {
							Ref: "#/components/headers/key",
						},
					},
					Content: map[string]*openapi3.MediaType{
						"application/json": {
							Schema: &openapi3.SchemaRef{
								Ref: "#/components/schemas/reqRef",
							},
						},
					},
				},
			},
		},
	}

	paths := openapi3.Paths{
		"/api/v1": {
			Post: &openapi3.Operation{

				RequestBody: &openapi3.RequestBodyRef{
					Ref: "#/components/requestBodies/xxx",
				},
				Responses: map[string]*openapi3.ResponseRef{
					"200": {
						Ref: "#/components/responses/respxxx",
					},
				},
				Parameters: []*openapi3.ParameterRef{
					{
						Ref: "#/components/parameters/key",
					},
				},
			},
		},
	}

	d := openapi3.T{
		OpenAPI:    "3.0.0",
		Components: x,
		Info: &openapi3.Info{
			Title:       "My API",
			Description: "This is my api",
			Version:     "1.0.0",
		},
		Paths: paths,
	}

	out, err := d.MarshalJSON()
	if err != nil {
		log.Fatal(err)
		return
	}

	var outi interface{}
	err = json.Unmarshal(out, &outi)
	if err != nil {
		log.Fatal(err)
		return
	}

	outyaml := &bytes.Buffer{}
	enc := yaml.NewEncoder(outyaml)
	enc.SetIndent(2)
	if err := enc.Encode(outi); err != nil {
		log.Fatal(err)
	}

	fmt.Println(outyaml.String())
}
