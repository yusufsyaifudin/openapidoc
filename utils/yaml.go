package utils

import (
	"bytes"
	"gopkg.in/yaml.v3"
)

func YamlMarshalIndent(i interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	yamlEncoder := yaml.NewEncoder(buf)
	yamlEncoder.SetIndent(2)
	_err := yamlEncoder.Encode(i)
	return buf.Bytes(), _err
}
