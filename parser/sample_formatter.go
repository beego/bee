package beeParser

import "encoding/json"

type AnnotationJSONFormatter struct {
	Annotation Annotator
}

func (f *AnnotationJSONFormatter) Format(field *StructField) string {
	if field.Comment == "" && field.Doc == "" {
		return ""
	}
	kvs := f.Annotation.Annotate(field.Doc + field.Comment)
	res, _ := json.Marshal(kvs)
	return string(res)
}

func NewAnnotationJSONFormatter() *AnnotationJSONFormatter {
	return &AnnotationJSONFormatter{Annotation: &Annotation{}}
}

type AnnotationYAMLFormatter struct {
	Annotation Annotator
}

func (f *AnnotationYAMLFormatter) Format(field *StructField) string {
	if field.Comment == "" && field.Doc == "" {
		return ""
	}
	kvs := f.Annotation.Annotate(field.Doc + field.Comment)
	res, _ := json.Marshal(kvs)
	return string(res)
}

func NewAnnotationYAMLFormatter() *AnnotationYAMLFormatter {
	return &AnnotationYAMLFormatter{Annotation: &Annotation{}}
}

type AnnotationTextFromatter struct {
	Annotation Annotator
}
