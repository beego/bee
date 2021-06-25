package beeParser

import (
	"encoding/json"
	"encoding/xml"

	"gopkg.in/yaml.v2"
)

type JsonFormatter struct {
}

func (f *JsonFormatter) FieldFormatFunc(field *StructField) ([]byte, error) {
	annotation := NewAnnotation(field.Doc + field.Comment)
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

func (f *YamlFormatter) FieldFormatFunc(field *StructField) ([]byte, error) {
	annotation := NewAnnotation(field.Doc + field.Comment)
	res := map[string]interface{}{}
	if field.NestedType != nil {
		res[annotation.Key] = field.NestedType
	} else {
		res[annotation.Key] = annotation.Default
	}
	return yaml.Marshal(res)
}

func (f *YamlFormatter) StructFormatFunc(node *StructNode) ([]byte, error) {
	return yaml.Marshal(node.Fields)
}

func (f *YamlFormatter) Marshal(node *StructNode) ([]byte, error) {
	return yaml.Marshal(node)
}

type XmlFormatter struct {
}

func (f *XmlFormatter) FieldFormatFunc(field *StructField) ([]byte, error) {
	annotation := NewAnnotation(field.Doc + field.Comment)
	if field.NestedType != nil {
		type xmlStruct struct {
			XMLName     xml.Name
			Default     interface{} `xml:",innerxml"`
			Description string      `xml:",comment"`
		}
		b, _ := field.NestedType.FormatFunc(field.NestedType)
		return xml.Marshal(&xmlStruct{
			XMLName:     xml.Name{Local: annotation.Key},
			Description: annotation.Description,
			Default:     b,
		})
	} else {
		type xmlStruct struct {
			XMLName     xml.Name
			Default     interface{} `xml:",chardata"`
			Description string      `xml:",comment"`
		}
		return xml.Marshal(&xmlStruct{
			XMLName:     xml.Name{Local: annotation.Key},
			Description: annotation.Description,
			Default:     annotation.Default,
		})
	}
}

func (f *XmlFormatter) StructFormatFunc(node *StructNode) ([]byte, error) {
	res := make([]byte, 0)
	for _, f := range node.Fields {
		b, _ := f.FormatFunc(f)
		res = append(res, b...)
		res = append(res, '\n')
	}
	return res, nil
}

func (f *XmlFormatter) Marshal(node *StructNode) ([]byte, error) {
	return node.FormatFunc(node)
}
