package beeParser

import (
	"encoding/json"
	"fmt"
	"log"
)

type sampleFormatter struct {
	Annotation Annotation
}

func (f *sampleFormatter) FieldFormatFunc(field *StructField) ([]byte, error) {
	// @todo update annotationResult by parsing with annotation struct
	annotationResult := field.Comment + field.Doc
	return json.Marshal(&struct {
		Key        string
		Annotation string
		NestedType *StructNode `json:"NestedType,omitempty"`
	}{
		Key:        field.Name,
		Annotation: annotationResult,
		NestedType: field.NestedType,
	})
}

func (f *sampleFormatter) StructFormatFunc(node *StructNode) ([]byte, error) {
	return json.Marshal(&struct {
		Key    string
		Fields []*StructField `json:"Fields,omitempty"`
	}{
		Key:    node.Name,
		Fields: node.Fields,
	})
}

func (f *sampleFormatter) Marshal(node *StructNode) ([]byte, error) {
	return json.Marshal(node)
}

func ExampleJSONMarshal() {
	const src = `
package p

import (
	"net/http"
)

type StructB struct {
	Field1 string
}
type StructA struct {
	// doc
	Field1 string //comment
	// @Name Field1
	// @Path https://github.com/beego/bee
	// 		  https://github.com/beego
	Field2 struct{
		a string
		b string
	}
	Field3 []string
	Field4 map[string]string
	Field5 http.SameSite
	Field6 func(int)
	Field7 StructB
}
`
	formatter := &sampleFormatter{}

	sp, err := NewStructParser("src.go", src, "StructA", formatter)
	if err != nil {
		log.Fatal(err)
	}

	b, err := sp.Marshal()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

	// Output:
}
