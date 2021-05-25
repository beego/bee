package beeParser

import "encoding/json"

type AnnotationFormatter struct {
	Annotation Annotator
}

func (f *AnnotationFormatter) Format(field *StructField) string {
	if field.Comment == "" && field.Doc == "" {
		return ""
	}
	kvs := f.Annotation.Annotate(field.Doc + field.Comment)
	res, _ := json.Marshal(kvs)
	return string(res)
}

func NewAnnotationFormatter() *AnnotationFormatter {
	return &AnnotationFormatter{Annotation: &Annotation{}}
}
