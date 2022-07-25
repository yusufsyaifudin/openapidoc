package main

import (
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/yusufsyaifudin/openapidoc"
	"github.com/yusufsyaifudin/openapidoc/header"
	"github.com/yusufsyaifudin/openapidoc/request"
	"github.com/yusufsyaifudin/openapidoc/response"
	"github.com/yusufsyaifudin/openapidoc/utils"
	"log"
	"net/http"
)

type Pet struct {
	ID              int      `json:"id" openapi3:"ex:2"`
	Name            string   `json:"name"`
	Sound           string   `json:"sound"`
	IsBark          bool     `json:"isBark" openapi3:"ex:true"`
	OtherAttributes []string `json:"otherAttributes" openapi3:"ex:'a;b'"`
}

type PetCreateReq struct {
	Pet *Pet `json:"pet" openapi3:"required:'id;name;sound'"`
}

type PetCreateResp struct {
	Data struct {
		Pet *Pet `json:"pet" openapi3:"desc:'pet schema'"`
	} `json:"data"`
}

type ErrorData struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type Error struct {
	Error ErrorData `json:"error,omitempty" openapi3:"desc:xxx"`
}

func main() {

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
		ServerInfo: info,
		Servers:    servers,
	}

	reg := openapidoc.NewRegistry(cfg)

	reg.Add(http.MethodPost, "/pets/{id}",
		request.NewRequest().
			Required(true).
			Body("application/json", PetCreateReq{}).
			PathParams(request.PathParam{Name: "id", Value: 1, Description: "Pet ID"}).
			Header(header.NewHeader().
				Add("Signature", header.Map{Value: "H256", Required: true}),
			),

		map[string]*response.Response{
			// response 201
			fmt.Sprintf("%d", http.StatusCreated): response.NewResponse().
				Body("application/json", PetCreateResp{}).
				Header(header.NewHeader().
					Add("Location", header.Map{Value: "/pets/:id", Description: "Newly created pets"}),
				),
			fmt.Sprintf("%d", http.StatusUnprocessableEntity): response.NewResponse().
				Body("application/json", Error{}),
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
