package openapidoc

import (
	"github.com/getkin/kin-openapi/openapi3"
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
