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
	"time"
)

type ResourceList map[testasset.ResourceName]testasset.Quantity

func TestRecursiveObject(t *testing.T) {
	type Chain struct {
		Hash  string `json:"hash"`
		Chain *Chain `json:"chain"` // always pointer for recursive
	}

	type BlockChain struct {
		FirstInitTime time.Time `json:"first_init_time"`
		ChainPointer  *Chain    `json:"chain_pointer"`
		ChainValue    Chain     `json:"chain_value"`
	}

	v := BlockChain{
		FirstInitTime: time.Date(2022, 9, 2, 17, 21, 0, 0, time.UTC),
		ChainPointer: &Chain{
			Hash: "ptr.hash1",
			Chain: &Chain{
				Hash: "ptr.hash1.child1",
				Chain: &Chain{
					Hash:  "ptr.hash1.child1.grandchild1",
					Chain: nil,
				},
			},
		},
		ChainValue: Chain{
			Hash: "val.hash1",
			Chain: &Chain{
				Hash: "val.hash1.child1",
				Chain: &Chain{
					Hash:  "val.hash1.child1.grandchild1",
					Chain: nil,
				},
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
			"/block-chain": &openapi3.PathItem{
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

	err = os.WriteFile(fmt.Sprintf("%s/tmp.yaml", curDir), openapiDocBytes.Bytes(), os.ModePerm)
	assert.NoError(t, err)

	//assert.EqualValues(t, testasset.NestedArray, openapiDocBytes.String())
}

func TestNestedObject(t *testing.T) {
	type Twig struct {
		NumLeaves  int `json:"numLeaves"`
		NumFlowers int `json:"numFlowers"`
	}

	type Branch struct {
		Twig Twig `json:"twig"`
	}

	type Tree struct {
		Branch Branch `json:"branch"`
	}

	v := Tree{
		Branch: Branch{
			Twig: Twig{
				NumLeaves:  10,
				NumFlowers: 2,
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
			"/block-chain": &openapi3.PathItem{
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

	err = os.WriteFile(fmt.Sprintf("%s/tmp.yaml", curDir), openapiDocBytes.Bytes(), os.ModePerm)
	assert.NoError(t, err)

	//assert.EqualValues(t, testasset.NestedArray, openapiDocBytes.String())
}

func TestCustomType(t *testing.T) {
	type Custom struct {
		Limits   ResourceList `json:"limits,omitempty"`
		Requests ResourceList `json:"requests,omitempty"`
	}

	v := Custom{
		Limits: ResourceList{
			"cpu":    testasset.NewQuantity("100m"),
			"memory": testasset.NewQuantity("1Gi"),
		},
		Requests: ResourceList{
			"cpu":    testasset.NewQuantity("100m"),
			"memory": testasset.NewQuantity("1Gi"),
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
			"/block-chain": &openapi3.PathItem{
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

	err = os.WriteFile(fmt.Sprintf("%s/tmp.yaml", curDir), openapiDocBytes.Bytes(), os.ModePerm)
	assert.NoError(t, err)

	//assert.EqualValues(t, testasset.NestedArray, openapiDocBytes.String())
}

func TestMapType(t *testing.T) {
	// ResourceList is a set of (resource name, quantity) pairs.
	// This similar like corev1.ResourceList on package k8s.io/api/core/v1
	// https://github.com/kubernetes/api/blob/v0.25.0/core/v1/types.go#L5285-L5286
	type ResourceList map[string]string

	type ResourceRequirements struct {
		Limits   ResourceList `json:"limits,omitempty"`
		Requests ResourceList `json:"requests,omitempty"`
	}

	type Deployment struct {
		ContainerResources ResourceRequirements `json:"container_resources,omitempty"`
	}

	value := Deployment{
		ContainerResources: ResourceRequirements{
			Limits: map[string]string{
				"cpu":    "100m",
				"memory": "1Gi",
			},
			Requests: map[string]string{
				"cpu":    "100m",
				"memory": "1Gi",
			},
		},
	}

	g, err := NewGenerator(WithLog(os.Stdout), WithSchemaPrefix("prefix."))
	assert.NotNil(t, g)
	assert.NoError(t, err)

	out, err := g.Generate(context.Background(), value)
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
			"/block-chain": &openapi3.PathItem{
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

	err = os.WriteFile(fmt.Sprintf("%s/tmp.yaml", curDir), openapiDocBytes.Bytes(), os.ModePerm)
	assert.NoError(t, err)

}
