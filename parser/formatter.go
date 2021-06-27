package beeParser

import (
	"encoding/json"

	"github.com/talos-systems/talos/pkg/machinery/config/encoder"
)

type JsonFormatter struct {
}

func (f *JsonFormatter) FieldFormatFunc(field *StructField) ([]byte, error) {
	annotation := NewAnnotation(field.Doc + field.Comment)
	if annotation.Key == "" {
		annotation.Key = field.Name
	}
	res := map[string]interface{}{}
	if field.NestedType != nil {
		res[annotation.Key] = field.NestedType
	} else {
		res[annotation.Key] = annotation.Default
	}
	return json.Marshal(res)
}

func (f *JsonFormatter) StructFormatFunc(node *StructNode) ([]byte, error) {
	return json.Marshal(node.Fields)
}

func (f *JsonFormatter) Marshal(node *StructNode) ([]byte, error) {
	return json.MarshalIndent(node, "", "	")
}

type YamlFormatter struct {
}

var result encoder.Doc

type Result map[string]interface{}

func (c Result) Doc() *encoder.Doc {
	return &result
}
func (f *YamlFormatter) FieldFormatFunc(field *StructField) ([]byte, error) {
	annotation := NewAnnotation(field.Doc + field.Comment)
	if annotation.Key == "" {
		annotation.Key = field.Name
	}
	res := Result{}
	// add head comment for this field
	res.Doc().Comments[encoder.HeadComment] = annotation.Description
	if field.NestedType != nil {
		b, _ := field.NestedType.FormatFunc(field.NestedType)
		res[annotation.Key] = string(b)
	} else {
		res[annotation.Key] = annotation.Default
	}
	encoder := encoder.NewEncoder(&res, []encoder.Option{
		encoder.WithComments(encoder.CommentsAll),
	}...)
	return encoder.Encode()
}

func (f *YamlFormatter) StructFormatFunc(node *StructNode) ([]byte, error) {
	res := make([]byte, 0)
	for _, f := range node.Fields {
		b, _ := f.FormatFunc(f)
		res = append(res, b...)
	}
	return res, nil
}

func (f *YamlFormatter) Marshal(node *StructNode) ([]byte, error) {
	return node.FormatFunc(node)
}
