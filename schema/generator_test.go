package schema_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/yusufsyaifudin/openapidoc/schema"
	"gopkg.in/yaml.v3"
	"testing"
)

type Pet struct {
	ID      int         `json:"id" openapi3:"ex:2"`
	Name    string      `json:"name"`
	Example interface{} `json:"example"`
}

type AnotherStruct struct {
	HOHO string `json:"HOHO"`
}

type Cat struct {
	Meow          string        `json:"meow"`
	Child         *Cat          `json:"child"`
	AnotherStruct AnotherStruct `json:"anotherStruct"`
}

func PetWrap(i interface{}) Pet {
	return Pet{
		Example: i,
	}
}

type RecursiveType struct {
	Field1     string           `json:"field1"`
	Field2     string           `json:"field2"`
	Field3     string           `json:"field3"`
	Components []*RecursiveType `json:"children,omitempty"`
}

func TestGenerator_Generate(t *testing.T) {
	v := PetWrap(Cat{
		Meow: "haha",
		Child: &Cat{
			Meow: "",
			Child: &Cat{
				Meow:          "",
				Child:         nil,
				AnotherStruct: AnotherStruct{},
			},
			AnotherStruct: AnotherStruct{},
		},
	})

	//v := PetWrap(&Cat{
	//	Meow:  "haha",
	//	Child: &Cat{},
	//})

	//v2 := RecursiveType{}
	g := schema.NewGenerator()
	schemas := g.Generate(v)

	c := openapi3.Components{
		Schemas: schemas,
	}

	doc := &openapi3.T{
		OpenAPI:    "3.0.0",
		Components: c,
	}

	b, _ := doc.MarshalJSON()

	var openapiDoc interface{}
	_ = json.Unmarshal(b, &openapiDoc)

	openapiDocBytes := &bytes.Buffer{}
	enc := yaml.NewEncoder(openapiDocBytes)
	enc.SetIndent(2)
	_ = enc.Encode(openapiDoc)

	fmt.Println(openapiDocBytes.String())
}
