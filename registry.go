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
	"strings"
)

type Config struct {
	generator  *openapi3gen.Generator
	serverInfo *openapi3.Info
	servers    openapi3.Servers
}

func WithGenerator(gen *openapi3gen.Generator) func(*Config) {
	return func(config *Config) {
		config.generator = gen
	}
}

func WithServerInfo(serverInfo *openapi3.Info) func(*Config) {
	return func(config *Config) {
		config.serverInfo = serverInfo
	}
}

func WithServers(servers openapi3.Servers) func(*Config) {
	return func(config *Config) {
		config.servers = servers
	}
}

type Registry struct {
	Config *Config

	paths      map[string]*openapi3.PathItem
	components *openapi3.Components
	err        error
}

func NewRegistry(configs ...func(*Config)) *Registry {
	config := &Config{
		generator:  openapi3gen.NewGenerator(openapi3gen.SchemaCustomizer(customizer)),
		serverInfo: &openapi3.Info{},
		servers:    openapi3.Servers{},
	}

	for _, cfg := range configs {
		cfg(config)
	}

	r := &Registry{
		Config:     config,
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

	reqComp, err := req.Components(r.Config.generator, requestName)
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
		case http.MethodGet:
			if pathItem.Get == nil {
				pathItem.Get = &openapi3.Operation{}
			}

			pathItem.Get.RequestBody = reqBodyRef
			pathItem.Get.Parameters = reqParams

		case http.MethodHead:
			if pathItem.Head == nil {
				pathItem.Head = &openapi3.Operation{}
			}

			pathItem.Head.RequestBody = reqBodyRef
			pathItem.Head.Parameters = reqParams

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

		case http.MethodPatch:
			if pathItem.Patch == nil {
				pathItem.Patch = &openapi3.Operation{}
			}

			pathItem.Patch.RequestBody = reqBodyRef
			pathItem.Patch.Parameters = reqParams

		case http.MethodDelete:
			if pathItem.Delete == nil {
				pathItem.Delete = &openapi3.Operation{}
			}

			pathItem.Delete.RequestBody = reqBodyRef
			pathItem.Delete.Parameters = reqParams

		case http.MethodConnect:
			if pathItem.Connect == nil {
				pathItem.Connect = &openapi3.Operation{}
			}

			pathItem.Connect.RequestBody = reqBodyRef
			pathItem.Connect.Parameters = reqParams

		case http.MethodOptions:
			if pathItem.Options == nil {
				pathItem.Options = &openapi3.Operation{}
			}

			pathItem.Options.RequestBody = reqBodyRef
			pathItem.Options.Parameters = reqParams

		case http.MethodTrace:
			if pathItem.Trace == nil {
				pathItem.Trace = &openapi3.Operation{}
			}

			pathItem.Trace.RequestBody = reqBodyRef
			pathItem.Trace.Parameters = reqParams

		}

	}

	// generate response for each http status code,
	// i.e: http status 200 OK may have different schema for http status 404 Not Found
	for httpCode, respInstance := range resp {
		// TODO: validate http code, must valid range of http codes or 1xx, 2xx, etc
		httpCode = strings.ToLower(httpCode)

		if respInstance == nil {
			continue
		}

		respComp, err := respInstance.Components(r.Config.generator, requestName, httpCode)
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
			case http.MethodGet:
				if pathItem.Get == nil {
					pathItem.Get = &openapi3.Operation{}
				}

				if pathItem.Get.Responses == nil {
					pathItem.Get.Responses = make(map[string]*openapi3.ResponseRef)
				}

				pathItem.Get.Responses[httpCode] = &openapi3.ResponseRef{
					Ref: respBodyRefName,
				}

			case http.MethodHead:
				if pathItem.Head == nil {
					pathItem.Head = &openapi3.Operation{}
				}

				if pathItem.Head.Responses == nil {
					pathItem.Head.Responses = make(map[string]*openapi3.ResponseRef)
				}

				pathItem.Head.Responses[httpCode] = &openapi3.ResponseRef{
					Ref: respBodyRefName,
				}

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

			case http.MethodPatch:
				if pathItem.Patch == nil {
					pathItem.Patch = &openapi3.Operation{}
				}

				if pathItem.Patch.Responses == nil {
					pathItem.Patch.Responses = make(map[string]*openapi3.ResponseRef)
				}

				pathItem.Patch.Responses[httpCode] = &openapi3.ResponseRef{
					Ref: respBodyRefName,
				}

			case http.MethodDelete:
				if pathItem.Delete == nil {
					pathItem.Delete = &openapi3.Operation{}
				}

				if pathItem.Delete.Responses == nil {
					pathItem.Delete.Responses = make(map[string]*openapi3.ResponseRef)
				}

				pathItem.Delete.Responses[httpCode] = &openapi3.ResponseRef{
					Ref: respBodyRefName,
				}

			case http.MethodConnect:
				if pathItem.Connect == nil {
					pathItem.Connect = &openapi3.Operation{}
				}

				if pathItem.Connect.Responses == nil {
					pathItem.Connect.Responses = make(map[string]*openapi3.ResponseRef)
				}

				pathItem.Connect.Responses[httpCode] = &openapi3.ResponseRef{
					Ref: respBodyRefName,
				}

			case http.MethodOptions:
				if pathItem.Options == nil {
					pathItem.Options = &openapi3.Operation{}
				}

				if pathItem.Options.Responses == nil {
					pathItem.Options.Responses = make(map[string]*openapi3.ResponseRef)
				}

				pathItem.Options.Responses[httpCode] = &openapi3.ResponseRef{
					Ref: respBodyRefName,
				}

			case http.MethodTrace:
				if pathItem.Trace == nil {
					pathItem.Trace = &openapi3.Operation{}
				}

				if pathItem.Trace.Responses == nil {
					pathItem.Trace.Responses = make(map[string]*openapi3.ResponseRef)
				}

				pathItem.Trace.Responses[httpCode] = &openapi3.ResponseRef{
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

	if r.Config.serverInfo == nil {
		r.Config.serverInfo = &openapi3.Info{
			Title:          "My Server",
			Description:    "Description server",
			TermsOfService: "",
			Contact:        nil,
			License:        nil,
			Version:        "v0.0.0",
		}

	}

	if r.Config.servers == nil {
		r.Config.servers = openapi3.Servers{
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
		Info:           r.Config.serverInfo,
		Paths:          r.paths,
		Security:       nil,
		Servers:        r.Config.servers,
		Tags:           nil,
		ExternalDocs:   nil,
	}

	return t, nil
}
