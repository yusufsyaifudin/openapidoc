package schema

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/yusufsyaifudin/openapidoc/schema/testasset"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"testing"
)

func TestRecursiveArray(t *testing.T) {
	type Children struct {
		NationalID float64 `json:"national_id"`
		Name       string  `json:"name"`
		Age        int     `json:"age"`

		Childrens []Children `json:"childrens"` // childrens can have many childrens
	}

	type Parent struct {
		NationalID float64    `json:"national_id"`
		Name       string     `json:"name"`
		Deceased   bool       `json:"deceased"`
		Age        int        `json:"age"`
		Childrens  []Children `json:"childrens,omitempty"`
	}

	v := Parent{
		NationalID: 100,
		Name:       "John",
		Deceased:   true,
		Age:        70,
		Childrens: []Children{
			{
				NationalID: 100.1,
				Name:       "Aji",
				Age:        35,
				Childrens: []Children{
					{
						NationalID: 100.11,
						Name:       "Aji Child 1",
						Age:        3,
					},
					{
						NationalID: 100.12,
						Name:       "Aji Child 2",
						Age:        1,
					},
				},
			},
			{
				NationalID: 100.2,
				Name:       "Bayu",
				Age:        30,
				Childrens: []Children{
					{
						NationalID: 100.21,
						Name:       "Bayu Child 1",
						Age:        1,
					},
				},
			},
			{
				NationalID: 100.3,
				Name:       "Chandra",
				Age:        28,
			},
		},
	}

	g, err := NewGenerator(WithLog(os.Stdout))
	assert.NotNil(t, g)
	assert.NoError(t, err)

	out, err := g.Generate(context.Background(), v)
	assert.NoError(t, err)

	reqBody := openapi3.NewRequestBody()
	reqBody.WithJSONSchemaRef(&openapi3.SchemaRef{
		Ref: fmt.Sprintf("#/components/schemas/%s", out.ParentSchemaName),
	})

	components := openapi3.Components{
		RequestBodies: openapi3.RequestBodies{
			"myReqBodyName": &openapi3.RequestBodyRef{
				Value: reqBody,
			},
		},
		Examples: out.Examples,
		Schemas:  out.Schemas,
	}

	op := openapi3.NewOperation()
	op.RequestBody = &openapi3.RequestBodyRef{
		Ref: "#/components/requestBodies/myReqBodyName", // refer to generated name we define above
	}
	op.AddResponse(http.StatusOK, openapi3.NewResponse().WithJSONSchemaRef(&openapi3.SchemaRef{
		Ref: fmt.Sprintf("#/components/schemas/%s", out.ParentSchemaName),
	}).WithDescription("desc"))

	doc := &openapi3.T{
		OpenAPI:    "3.0.0",
		Components: components,
		Info: &openapi3.Info{
			ExtensionProps: openapi3.ExtensionProps{},
			Title:          "My API",
			Version:        "1.0.0",
		},
		Paths: openapi3.Paths{
			"/family-tree": &openapi3.PathItem{
				Post: op,
			},
		},
	}

	b, _ := doc.MarshalJSON()

	var openapiDoc interface{}
	_ = json.Unmarshal(b, &openapiDoc)

	openapiDocBytes := &bytes.Buffer{}
	enc := yaml.NewEncoder(openapiDocBytes)
	enc.SetIndent(2)
	_ = enc.Encode(openapiDoc)

	curDir, err := os.Getwd()
	assert.NotEmpty(t, curDir)
	assert.NoError(t, err)

	//err = os.WriteFile(fmt.Sprintf("%s/tmp.yaml", curDir), openapiDocBytes.Bytes(), os.ModePerm)
	//assert.NoError(t, err)

	assert.EqualValues(t, testasset.NestedArray, openapiDocBytes.String())
}
