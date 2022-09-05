package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/yusufsyaifudin/openapidoc/schema"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Requirements struct {
	ResourceRequirements *corev1.ResourceRequirements `json:"resource_requirements"`
}

func main() {

	//defer func() {
	//	if r := recover(); r != nil {
	//		log.Println(r)
	//	}
	//}()
	//
	//gen := openapi3gen.NewGenerator()
	//sch, err := gen.GenerateSchemaRef(reflect.TypeOf(Requirements{}))
	//if err != nil {
	//	log.Fatalln(err)
	//	return
	//}

	//out := map[string]*openapi3.Schemas{}
	//out["project"] = sch
	//
	//fmt.Println(generateYAML(out))

	schemaGen := schema.NewGenerator()

	schemaReference := schemaGen.Generate(Requirements{
		ResourceRequirements: &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(200, resource.Milli),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: *resource.NewScaledQuantity(1, resource.Giga),
			},
		},
	})

	fmt.Println(generateYAML(schemaReference))

	return
}

func generateYAML(in map[string]*openapi3.SchemaRef) string {
	a, _ := json.Marshal(in)

	var b interface{}
	_ = yaml.Unmarshal(a, &b)

	c := &bytes.Buffer{}
	enc := yaml.NewEncoder(c)
	enc.SetIndent(2)
	_ = enc.Encode(b)
	return c.String()
}
