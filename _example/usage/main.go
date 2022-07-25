package main

import (
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/yusufsyaifudin/openapidoc"
	"github.com/yusufsyaifudin/openapidoc/header"
	"github.com/yusufsyaifudin/openapidoc/request"
	"github.com/yusufsyaifudin/openapidoc/response"
	"github.com/yusufsyaifudin/openapidoc/utils"
	"log"
	"net/http"
	"reflect"
)

type Pet struct {
	Name  string `json:"name"`
	Sound string `json:"sound"`
}

type PetCreateReq struct {
	SimpleStr string   `json:"simpleStr"`
	Strings   []string `json:"strings"`
	Pet       *Pet     `json:"pet"`
}

type PetCreateResp struct {
	Data struct {
		Pet *Pet `json:"pet"`
	} `json:"data"`
}

func main() {
	customizer := func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		schema.Description = tag.Get("desc")

		switch t.Kind() {
		case reflect.String:
			schema.Example = tag.Get("ex")
		}

		return nil
	}

	gen := openapi3gen.NewGenerator(openapi3gen.SchemaCustomizer(customizer))

	info := &openapi3.Info{
		Title:          "My Server",
		Description:    "Description server",
		TermsOfService: "",
		Contact:        nil,
		License:        nil,
		Version:        "v1.0.0",
	}

	servers := openapi3.Servers{
		{
			URL:         "https://example.com/staging",
			Description: "URL Staging",
			Variables:   nil,
		},
		{
			URL:         "https://example.com/production",
			Description: "URL Production",
			Variables:   nil,
		},
	}

	cfg := openapidoc.Config{
		Generator:  gen,
		ServerInfo: info,
		Servers:    servers,
	}

	reg := openapidoc.NewRegistry(cfg)

	reg.Add(http.MethodPost, "/pets/{id}",
		request.NewRequest().
			Body("application/json", PetCreateReq{}).
			PathParams(request.PathParam{Name: "id", Value: 1, Description: "Pet ID"}).
			Header(header.NewHeader().
				Add("Signature", header.Map{Value: "H256", Required: true}),
			),

		map[string]*response.Response{
			// response 201
			fmt.Sprintf("%d", http.StatusCreated): response.NewResponse(response.Config{}).
				Body("application/json", PetCreateResp{}).
				Header(header.NewHeader().
					Add("Location", header.Map{Value: "/pets/:id", Description: "Newly created pets"}),
				),
		},
	)

	doc, err := reg.Generate()
	if err != nil {
		log.Fatalln(err)
		return
	}

	j, _ := doc.MarshalJSON()

	var i interface{}
	_ = json.Unmarshal(j, &i)

	y, _ := utils.YamlMarshalIndent(i)
	fmt.Println(string(y))
}
