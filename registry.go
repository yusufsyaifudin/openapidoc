package openapidoc

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/hashicorp/go-multierror"
	"github.com/yusufsyaifudin/openapidoc/request"
	"github.com/yusufsyaifudin/openapidoc/response"
	"github.com/yusufsyaifudin/openapidoc/utils"
	"hash/fnv"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

var customizer = func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
	tagValue := tag.Get("openapi3")
	tagSplit := strings.Split(tagValue, ",")

	tagMap := make(map[string]string)
	for _, val := range tagSplit {
		kv := strings.Split(val, ":")
		kvLen := len(kv)
		switch {
		case kvLen >= 2:
			tagMap[kv[0]] = strings.ReplaceAll(strings.Join(kv[1:], " "), "'", "")
		case kvLen == 1:
			tagMap[kv[0]] = ""
		}
	}

	for k, v := range tagMap {
		var (
			desc          string
			exampleVal    interface{}
			requiredField []string // only valid for type object
		)

		switch k {
		case "desc":
			desc = v

		case "ex":
			switch t.Kind() {
			case reflect.String:
				vArr := strings.Split(v, ";")
				if len(vArr) > 0 {
					exampleVal = vArr[len(vArr)-1]
				} else {
					exampleVal = v
				}

			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				vInt, _ := strconv.Atoi(v)
				exampleVal = vInt

			case reflect.Float32, reflect.Float64:
				vFloat, _ := strconv.ParseFloat(v, 64)
				exampleVal = vFloat

			case reflect.Bool:
				vBool, _ := strconv.ParseBool(v)
				exampleVal = vBool

			case reflect.Array, reflect.Slice:
				exampleVal = strings.Split(v, ";")

			default:
				exampleVal = v
			}

		case "required":
			switch t.Kind() {
			case reflect.Map, reflect.Struct, reflect.Pointer:
				requiredField = strings.Split(v, ";")
			}
		}

		if desc != "" {
			schema.Description = desc
		}

		if exampleVal != nil {
			schema.Example = exampleVal
		}

		if len(requiredField) > 0 {
			schema.Required = requiredField
		}
	}

	return nil
}

type Config struct {
	Generator  *openapi3gen.Generator
	ServerInfo *openapi3.Info
	Servers    openapi3.Servers
}

type Registry struct {
	Config Config

	paths      map[string]*openapi3.PathItem
	components *openapi3.Components
	err        error
}

func NewRegistry(cfg Config) *Registry {
	if cfg.Generator == nil {
		cfg.Generator = openapi3gen.NewGenerator(openapi3gen.SchemaCustomizer(customizer))
	}

	r := &Registry{
		Config:     cfg,
		paths:      make(map[string]*openapi3.PathItem),
		components: &openapi3.Components{},
		err:        nil,
	}
	return r
}

// Add
// method contains http method: GET, POST, PUT, PATCH, HEAD, DELETE
// path contains URL path such as /api/v1/xxx
// req contains request body, header, params, etc..
// resp contains map of http status as key and response body as value, for example: 200:&response.Response{}
//
// Multiple path with different method can be added.
func (r *Registry) Add(method string, path string, req *request.Request, resp map[string]*response.Response) {
	if req == nil {
		return
	}

	if resp == nil {
		return
	}

	hasher := fnv.New32()
	_, err := hasher.Write([]byte(fmt.Sprintf("%s.%s", method, path)))
	if err != nil {
		err = fmt.Errorf("cannot hash method and path request: %w", err)
		r.err = multierror.Append(r.err, err)
		return
	}

	requestName := fmt.Sprintf("%d", hasher.Sum32())

	reqComp, err := req.Components(r.Config.Generator, requestName)
	if err != nil {
		err = fmt.Errorf("cannot create components for the request payload: %w", err)
		r.err = multierror.Append(r.err, err)
		return
	}

	// add parameters from request.Request to this specific method:path
	// this includes header params, path params, query params, etc
	reqParams := make([]*openapi3.ParameterRef, 0)
	for parameterRefName := range reqComp.Parameters {
		reqParams = append(reqParams, &openapi3.ParameterRef{
			Ref: fmt.Sprintf("#/components/parameters/%s", parameterRefName),
		})
	}

	// merge components from request to current T
	utils.MergeComponents(r.components, reqComp)

	if r.paths[path] == nil {
		r.paths[path] = &openapi3.PathItem{}
	}

	pathItem := r.paths[path]

	// refer request schema to this method:path request URI
	for reqBodyName := range reqComp.RequestBodies {
		// The Add method can be called multiple times to add different method for the same path.

		reqBodyRefName := fmt.Sprintf("#/components/requestBodies/%s", reqBodyName)
		reqBodyRef := &openapi3.RequestBodyRef{
			Ref: reqBodyRefName,
		}

		switch method {
		case http.MethodPost:
			if pathItem.Post == nil {
				pathItem.Post = &openapi3.Operation{}
			}

			pathItem.Post.RequestBody = reqBodyRef
			pathItem.Post.Parameters = reqParams

		case http.MethodPut:
			if pathItem.Put == nil {
				pathItem.Put = &openapi3.Operation{}
			}

			pathItem.Put.RequestBody = reqBodyRef
			pathItem.Put.Parameters = reqParams
		}
	}

	// generate response for each http status code,
	// i.e: http status 200 OK may have different schema for http status 404 Not Found
	for httpCode, respInstance := range resp {
		// TODO: validate http code, must valid range of http codes or 1XX, 2XX, etc
		httpCode = strings.ToUpper(httpCode)

		if respInstance == nil {
			continue
		}

		respComp, err := respInstance.Components(r.Config.Generator, requestName, httpCode)
		if err != nil {
			err = fmt.Errorf("cannot create components for the response payload %s: %w", httpCode, err)
			r.err = multierror.Append(r.err, err)
			return
		}

		// merge components from request to current T
		utils.MergeComponents(r.components, respComp)

		// add responses schema to current components
		// same method and path can multiple response with different content type
		for respBodyName := range respComp.Responses {
			respBodyRefName := fmt.Sprintf("#/components/responses/%s", respBodyName)

			switch method {
			case http.MethodPost:
				if pathItem.Post == nil {
					pathItem.Post = &openapi3.Operation{}
				}

				if pathItem.Post.Responses == nil {
					pathItem.Post.Responses = make(map[string]*openapi3.ResponseRef)
				}

				pathItem.Post.Responses[httpCode] = &openapi3.ResponseRef{
					Ref: respBodyRefName,
				}

			case http.MethodPut:
				if pathItem.Put == nil {
					pathItem.Put = &openapi3.Operation{}
				}

				if pathItem.Put.Responses == nil {
					pathItem.Put.Responses = make(map[string]*openapi3.ResponseRef)
				}

				pathItem.Put.Responses[httpCode] = &openapi3.ResponseRef{
					Ref: respBodyRefName,
				}
			}

		}
	}

	r.paths[path] = pathItem
}

func (r *Registry) Generate() (*openapi3.T, error) {
	if r.err != nil {
		return nil, r.err
	}

	if r.Config.ServerInfo == nil {
		r.Config.ServerInfo = &openapi3.Info{
			Title:          "My Server",
			Description:    "Description server",
			TermsOfService: "",
			Contact:        nil,
			License:        nil,
			Version:        "v0.0.0",
		}

	}

	if r.Config.Servers == nil {
		r.Config.Servers = openapi3.Servers{
			{
				URL:         "https://example.com/",
				Description: "This is example URL",
				Variables:   nil,
			},
		}
	}

	t := &openapi3.T{
		ExtensionProps: openapi3.ExtensionProps{},
		OpenAPI:        "3.0.3",
		Components:     *r.components,
		Info:           r.Config.ServerInfo,
		Paths:          r.paths,
		Security:       nil,
		Servers:        r.Config.Servers,
		Tags:           nil,
		ExternalDocs:   nil,
	}

	return t, nil
}
