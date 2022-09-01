package schema

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"reflect"
	"strings"
	"sync/atomic"
)

var inc atomic.Int64

type Generator struct {
	// Types save the same struct name as key.
	// If the type name is pointer (using '*' prefix), then we remove it
	schemas map[string]*openapi3.SchemaRef
}

func NewGenerator() *Generator {
	return &Generator{
		schemas: map[string]*openapi3.SchemaRef{},
	}
}

// Generate will generate each fields on struct based on reflect.Value and get the tag on reflect.Type at the same time.
// Unlike the openapi3gen.Generator which only sees the reflect.Type,
// by watching the reflect.Value we can get the exact type if we pass interface{} or any Go native values.
// This useful when we share the struct to create standard response structure, etc.
//
// TODO: this function is in WIP state
func (g *Generator) Generate(structValue interface{}) map[string]*openapi3.SchemaRef {

	currentSchema := map[string]*openapi3.SchemaRef{}

	fields := reflect.TypeOf(structValue)
	values := reflect.ValueOf(structValue)

	// dereference fields and values
	for fields.Kind() == reflect.Ptr {
		fields = fields.Elem()
	}

	for values.Kind() == reflect.Ptr {
		values = values.Elem()
	}

	num := fields.NumField()
	numVal := values.NumField()
	if num != numVal {
		err := fmt.Errorf("reflect.Type num fields is %d but reflect.Value num fields is %d", num, numVal)
		panic(err)
	}

	for i := 0; i < num; i++ {

		field := fields.Field(i)
		value := values.Field(i)

		//fmt.Printf("%T    : Type: %s %s: %+v\n",
		//	structValue,
		//	field.Type, field.Name, value,
		//)

		fmt.Printf("%d_ %T %+v %+v\n", inc.Add(1), value.Interface(), value.Interface(), field.Name)

		// iterate over the values
		// With this methodology, we expect that all values must be set in the struct when we add as open api schema.
		// All values in this struct also be used as example.
		switch value.Kind() {
		case reflect.Ptr:
			// check if pointer is nil
			if value.IsZero() {
				continue
			}

			// dereference pointer value
			ptrVal := values.Field(i)
			for ptrVal.Kind() == reflect.Ptr {
				ptrVal = ptrVal.Elem()
			}

			// get value fields
			schemaName, schema := g.generateWithoutSaving(field.Name, ptrVal.Interface())
			if g.schemas[schemaName] != nil {
				// if schema name already exist then don't generate it again
				currentSchema[field.Name] = &openapi3.SchemaRef{
					Ref: fmt.Sprintf("#/components/schemas/%s", schemaName),
				}
				continue
			}

			g.schemas[schemaName] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:       "object",
					Properties: schema,
				},
			}

			currentSchema[field.Name] = &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", schemaName),
			}
			continue

		case reflect.Interface:

			// generate new schema for this type and refer generated schema to this one.
			schemaName, schema := g.generateWithoutSaving(field.Name, value.Interface())
			if g.schemas[schemaName] != nil {

				currentSchema[field.Name] = &openapi3.SchemaRef{
					Ref: fmt.Sprintf("#/components/schemas/%s", schemaName),
				}
				continue
			}

			g.schemas[schemaName] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:       "object",
					Properties: schema,
				},
			}

			currentSchema[field.Name] = &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", schemaName),
			}
			continue

		default:

			currentSchema[field.Name] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:    value.Type().Name(),
					Example: value.Interface(),
				},
			}
			continue
		}

	}

	schemaName := fmt.Sprintf("%T", structValue)
	schemaName = strings.TrimPrefix(schemaName, "*")
	for strings.HasPrefix(schemaName, "*") {
		schemaName = strings.TrimPrefix(schemaName, "*")
	}

	g.schemas[schemaName] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:       "object",
			Properties: currentSchema,
		},
	}

	return g.schemas
}

func (g *Generator) generateWithoutSaving(parentFieldName string, structValue interface{}) (schemaName string, currentSchema map[string]*openapi3.SchemaRef) {
	currentSchema = map[string]*openapi3.SchemaRef{}

	fields := reflect.TypeOf(structValue)
	values := reflect.ValueOf(structValue)

	// dereference fields and values
	for fields.Kind() == reflect.Ptr {
		fields = fields.Elem()
	}

	for values.Kind() == reflect.Ptr {
		values = values.Elem()
	}

	num := fields.NumField()
	numVal := values.NumField()
	if num != numVal {
		err := fmt.Errorf("reflect.Type num fields is %d but reflect.Value num fields is %d", num, numVal)
		panic(err)
	}

	// this points to output named params
	schemaName = fmt.Sprintf("%T", structValue)
	schemaName = strings.TrimPrefix(schemaName, "*")
	for strings.HasPrefix(schemaName, "*") {
		schemaName = strings.TrimPrefix(schemaName, "*")
	}

	//fmt.Printf("%d %T %s %+v\n", inc.Add(1), values.Interface(), parentFieldName, values.Interface())

	for i := 0; i < num; i++ {
		field := fields.Field(i)
		value := values.Field(i)

		// if schema name already exist then don't generate it again
		if g.schemas[schemaName] != nil {
			//fmt.Println(parentFieldName, schemaName, field.Name, "this")
			currentSchema[field.Name] = &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", schemaName),
			}
			continue
		}

		switch value.Kind() {
		case reflect.Ptr:
			// check if pointer is nil
			if value.IsZero() {
				continue
			}

			// dereference pointer value
			ptrVal := values.Field(i)
			for ptrVal.Kind() == reflect.Ptr {
				ptrVal = ptrVal.Elem()
			}

			newSchemaName, newSchema := g.generateWithoutSaving(field.Name, ptrVal.Interface())
			if g.schemas[newSchemaName] != nil {
				//fmt.Printf("%d %+v %+v\n", inc.Add(1), newSchemaName, ptrVal.Interface())

				// if schema name already exist then don't generate it again
				currentSchema[field.Name] = &openapi3.SchemaRef{
					Ref: fmt.Sprintf("#/components/schemas/%s", newSchemaName),
				}
				continue
			}

			g.schemas[newSchemaName] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:       "object",
					Properties: newSchema,
				},
			}

			currentSchema[field.Name] = &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", newSchemaName),
			}

			continue

		case reflect.Interface, reflect.Struct:

			newSchemaName, newSchema := g.generateWithoutSaving(field.Name, value.Interface())
			if g.schemas[newSchemaName] != nil {
				//fmt.Printf("%d %+v %+v\n", inc.Add(1), newSchemaName, value.Interface())

				// if schema name already exist then don't generate it again
				currentSchema[field.Name] = &openapi3.SchemaRef{
					Ref: fmt.Sprintf("#/components/schemas/%s", newSchemaName),
				}
				continue
			}

			g.schemas[newSchemaName] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:       "object",
					Properties: newSchema,
				},
			}

			currentSchema[field.Name] = &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", newSchemaName),
			}

			continue

		default:

			//fmt.Printf("%d %+v %+v\n", inc.Add(1), field.Name, value.Interface())

			//g.schemas[newSchemaName] = &openapi3.SchemaRef{
			//	Value: &openapi3.Schema{
			//		Type:       "object",
			//		Properties: newSchema,
			//	},
			//}

			currentSchema[field.Name] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:    value.Type().Name(),
					Example: value.Interface(),
				},
			}
		}

	}

	return
}
