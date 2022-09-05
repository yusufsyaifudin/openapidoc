package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"io"
	"reflect"
	"strings"
	"time"
)

type Opt func(*Generator) error

type Generator struct {
	logEnabled   bool
	logWriter    io.Writer
	goTag        []string
	schemaPrefix string
}

// WithLog enables debug log to see how Schema will be appended as final Schemas.
func WithLog(writer io.Writer) Opt {
	return func(gen *Generator) error {
		if writer == nil {
			return fmt.Errorf("nil writer for logging")
		}

		gen.logEnabled = true
		gen.logWriter = writer
		return nil
	}
}

// WithGoTag will try to get the first non-empty field name based on Golang tag.
// For example, if you use ["json", "yaml"], then json tag will be used as field name if the value is not empty.
// If json tag is empty, then it will try using yaml tag name.
// If all the list is still return empty string, it will use golang default field name.
//
// By default, this will use json tag.
func WithGoTag(tags []string) Opt {
	unique := make(map[string]struct{})
	return func(gen *Generator) error {
		// validate unique tag
		for idx, tag := range tags {
			if _, exist := unique[tag]; exist {
				return fmt.Errorf("tag %s already defined before on index %d", tag, idx)
			}

			unique[tag] = struct{}{}
		}

		gen.goTag = tags
		return nil
	}
}

func WithSchemaPrefix(prefix string) Opt {
	return func(gen *Generator) error {
		prefix = strings.TrimSpace(prefix)
		gen.schemaPrefix = prefix
		return nil
	}
}

func NewGenerator(options ...Opt) (*Generator, error) {

	gen := &Generator{
		logEnabled: false,
		logWriter:  &noopWriter{},
		goTag:      []string{"json"},
	}

	for _, option := range options {
		if option != nil {
			err := option(gen)
			if err != nil {
				return nil, err
			}
		}
	}

	return gen, nil
}

func getSchemaName(prefix string, structValue interface{}) string {
	schemaName := fmt.Sprintf("%T", structValue)
	schemaName = strings.TrimPrefix(schemaName, "*")
	for strings.HasPrefix(schemaName, "*") {
		schemaName = strings.TrimPrefix(schemaName, "*")
	}

	return fmt.Sprintf("%s%s", prefix, schemaName)
}

type GenerateOut struct {
	ParentSchemaName string
	Schemas          map[string]*openapi3.SchemaRef
	Examples         openapi3.Examples
}

func (g *Generator) Generate(ctx context.Context, structValue interface{}) (out GenerateOut, err error) {
	parentSchemaName := getSchemaName(g.schemaPrefix, structValue)

	schemaRef := make(map[string]*openapi3.SchemaRef)
	err = g.generate(ctx, 0, "", structValue, schemaRef)
	if err != nil {
		return
	}

	out = GenerateOut{
		ParentSchemaName: parentSchemaName,
		Schemas:          schemaRef,
	}
	return
}

func (g *Generator) generate(
	ctx context.Context,
	called int,
	jsonPath string,
	structValue interface{},
	schemaRef openapi3.Schemas,
) (err error) {
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

	numField := fields.NumField()
	numVal := values.NumField()
	if numField != numVal {
		err = fmt.Errorf("reflect.Type numField fields is %d but reflect.Value numField fields is %d", numField, numVal)
		return
	}

	schemaName := getSchemaName(g.schemaPrefix, structValue)

	if g.logEnabled {
		msg := make([]byte, 0)

		_val, _exist := schemaRef[schemaName]
		if _exist && _val != nil {
			if _val.Value != nil {
				b, _ := json.Marshal(_val.Value.Properties)
				msg = fmt.Appendf(msg, "%d exist schema name %s %s\n", called, schemaName, b)
			}
		} else {
			msg = fmt.Appendf(msg, "%d not exist, will append schema name %s\n", called, schemaName)
			msg = fmt.Appendf(msg, "%d not exist, type %s json path '%s'\n", called, reflect.TypeOf(structValue).Kind(), jsonPath)
		}

		_, _ = g.logWriter.Write(msg)

	}

	for i := 0; i < numField; i++ {

		field := fields.Field(i)
		value := values.Field(i)

		// propertyFieldName by default using Go field name, but will try to look up from the go tag
		propertyFieldName := field.Name
		for _, goTag := range g.goTag {
			tagVal, tagValExist := field.Tag.Lookup(goTag)
			tagVal = strings.TrimSpace(tagVal)
			if !tagValExist || tagVal == "" {
				continue
			}

			if s := strings.Split(tagVal, ","); len(s) > 0 {
				propertyFieldName = strings.TrimSpace(s[0])
				if propertyFieldName != "" {
					break // break only if property field name is not empty
				}
			}
		}

		// finally use Golang field name as default if empty
		if propertyFieldName == "" {
			propertyFieldName = field.Name
		}

		if g.logEnabled {
			msg := make([]byte, 0)
			msg = fmt.Appendf(msg,
				"%d iterate field name '%s' as '%s' with type %s\n",
				called, field.Name, propertyFieldName, value.Kind(),
			)
			_, _ = g.logWriter.Write(msg)
		}

		if !value.CanInterface() {
			msg := make([]byte, 0)
			msg = fmt.Appendf(msg,
				"%d field '%s' on struct '%T' is unexported and Interface cannot be used without panicking\n",
				called, field.Name, structValue,
			)
			_, _ = g.logWriter.Write(msg)

			continue
		}

		// iterate over the values
		// With this methodology, we expect that all values must be set in the struct when we add as open api schema.
		// All values in this struct also be used as example.
		// OpenAPI 3 schema only support type: array, boolean, integer, number, object, string
		switch value.Kind() {

		case reflect.Bool:
			currentSchema[propertyFieldName] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:    "boolean",
					Example: value.Interface(),
				},
			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:

			currentSchema[propertyFieldName] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:    "integer",
					Example: value.Interface(),
				},
			}

		case reflect.Float32, reflect.Float64:

			currentSchema[propertyFieldName] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:    "number",
					Example: value.Interface(),
				},
			}

		case reflect.String:

			currentSchema[propertyFieldName] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:    "string",
					Example: value.String(),
				},
			}

		case reflect.Interface, reflect.Struct:
			// if implements fmt.Stringer
			example := value.Interface()
			if stringer, ok := value.Interface().(interface{ String() string }); ok {
				example = stringer.String()
			}

			if _, ok := value.Interface().(time.Time); ok {
				currentSchema[propertyFieldName] = &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:    "string",
						Example: example,
					},
				}

				continue
			}

			//objSchemaRefs := make(map[string]*openapi3.SchemaRef)

			err = g.generate(ctx, called+1, fmt.Sprintf("%s.%s", jsonPath, propertyFieldName), value.Interface(), schemaRef)
			if err != nil {
				return
			}

			// This is enough if we have simple object.
			// But, if we have multiple object or recursive object, then we need allOf method
			newSchemaName := getSchemaName(g.schemaPrefix, value.Interface())
			currentSchema[propertyFieldName] = &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", newSchemaName),
			}

			//allOfSchemaRef := make([]*openapi3.SchemaRef, 0)
			//for objSchemaName, objSchemaRef := range objSchemaRefs {
			//
			//	// add to the final output
			//	schemaRef[objSchemaName] = objSchemaRef
			//
			//	// then add to array to be referenced later in type object
			//	allOfSchemaRef = append(allOfSchemaRef, &openapi3.SchemaRef{
			//		Ref: fmt.Sprintf("#/components/schemas/%s", objSchemaName),
			//	})
			//}
			//
			//currentSchema[propertyFieldName] = &openapi3.SchemaRef{
			//	Value: &openapi3.Schema{
			//		Type:    "object",
			//		AllOf:   allOfSchemaRef,
			//		Example: value.Interface(),
			//	},
			//}

		case reflect.Slice, reflect.Array:

			msg := make([]byte, 0)
			msg = fmt.Appendf(msg, "%d slice", called)
			_, _ = g.logWriter.Write(msg)

			allObjInArr := make(map[string]*openapi3.SchemaRef)

			// expected behavior: recursive generate wrong example as specified in this issue
			// https://github.com/swagger-api/swagger-ui/issues/3325

			// we use decrement to ensure that first element of the array is generated as final example.
			// see log for debugging purpose.
			for j := value.Len() - 1; j >= 0; j-- {
				arrVal := value.Index(j)
				err = g.generate(ctx, called+1, fmt.Sprintf("%s[].%s", jsonPath, propertyFieldName), arrVal.Interface(), allObjInArr)
				if err != nil {
					return
				}
			}

			anyOfSchemaRef := openapi3.SchemaRefs{}
			for multipleArrSchemaName, arrSchemaRefValue := range allObjInArr {

				// add to the final output
				schemaRef[multipleArrSchemaName] = arrSchemaRefValue

				// then add to array to be referenced later in type array
				anyOfSchemaRef = append(anyOfSchemaRef, &openapi3.SchemaRef{
					Ref: fmt.Sprintf("#/components/schemas/%s", multipleArrSchemaName),
				})
			}

			// https://stackoverflow.com/a/47657131
			currentSchema[propertyFieldName] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: "array",
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							AnyOf: anyOfSchemaRef,
						},
					},
					// TODO add example on type array
					Example: value.Interface(),
				},
			}

		case reflect.Ptr:
			// check if pointer is nil
			if value.IsZero() {
				msg := make([]byte, 0)
				msg = fmt.Appendf(msg, "%d zero pointer\n", called)
				_, _ = g.logWriter.Write(msg)

				continue
			}

			// dereference pointer value
			ptrVal := values.Field(i)
			for ptrVal.Kind() == reflect.Ptr {
				ptrVal = ptrVal.Elem()
			}

			if !ptrVal.CanInterface() {
				msg := make([]byte, 0)
				msg = fmt.Appendf(msg, "%d not exported field '%s'\n", called, field.Name)
				_, _ = g.logWriter.Write(msg)

				continue
			}

			err = g.generate(ctx, called+1, fmt.Sprintf("%s.%s", jsonPath, propertyFieldName), ptrVal.Interface(), schemaRef)
			if err != nil {
				return
			}

			newSchemaName := getSchemaName(g.schemaPrefix, ptrVal.Interface())
			currentSchema[propertyFieldName] = &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", newSchemaName),
			}

		case reflect.Map:

			mapSchemaProps := make(map[string]*openapi3.SchemaRef)
			mapExample := make(map[string]interface{})

			mapIter := value.MapRange()
			for mapIter.Next() {
				// for type map, use key as-is it defined on the struct value as an example.
				// For value, we will try to generate it again
				mapKeyName := fmt.Sprintf("%s", mapIter.Key().Interface())
				mapValue := mapIter.Value()

				// use string format as value example
				example := fmt.Sprintf("%+v", mapValue.Interface())
				if stringer, ok := mapValue.Interface().(interface{ String() string }); ok {
					example = stringer.String()
				}

				mapExample[mapKeyName] = example

				jsonPath = fmt.Sprintf("%s.%s", jsonPath, mapKeyName)
				err = g.generate(ctx, called+1, jsonPath, mapValue.Interface(), mapSchemaProps)
				if err != nil {
					return
				}
			}

			mapSchemaPropsRef := make(map[string]*openapi3.SchemaRef)
			for mapSchemaName, mapSchemaRef := range mapSchemaProps {
				// add to final output schema map
				schemaRef[mapSchemaName] = mapSchemaRef

				mapSchemaPropsRef[mapSchemaName] = &openapi3.SchemaRef{
					Ref: fmt.Sprintf("#/components/schemas/%s", mapSchemaName),
				}
			}

			mapSchema := &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:       "object",
					Example:    mapExample,
					Properties: mapSchemaPropsRef,
				},
			}

			currentSchema[propertyFieldName] = mapSchema

		default:
			msg := make([]byte, 0)
			msg = fmt.Appendf(msg, "%d default", called)
			_, _ = g.logWriter.Write(msg)

			// error still using field.Name for clarity. It refers to the original Go field name,
			// not from golang tag
			err = fmt.Errorf("not supported type %s on field '%s'", value.Kind(), field.Name)
			return
		}
	}

	// ensure that we don't need to check if schemaName already exist in the map,
	// because when we enable log, it will print in reverse order (from the last recursive element to the first),
	// so, first value will end-up as the final value.
	if g.logEnabled {
		msg := make([]byte, 0)

		b, _err := json.Marshal(currentSchema)
		if _err != nil {
			msg = fmt.Appendf(msg, "%d cannot marshal current schema: %+v\n", called, currentSchema)
		}

		msg = fmt.Appendf(msg, "%d appended %s %s\n", called, schemaName, b)
		_, _ = g.logWriter.Write(msg)
	}

	schemaRef[schemaName] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:       "object",
			Properties: currentSchema,
		},
	}

	return
}
