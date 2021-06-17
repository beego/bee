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

func ExamplesampleFormatter() {
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
	// "{\"Key\":\"StructA\",\"Fields\":[\"{\\\"Key\\\":\\\"Field1\\\",\\\"Annotation\\\":\\\"comment\\\\ndoc\\\\n\\\"}\",\"{\\\"Key\\\":\\\"Field2\\\",\\\"Annotation\\\":\\\"@Name Field1\\\\n@Path https://github.com/beego/bee\\\\n\\\\t\\\\t  https://github.com/beego\\\\n\\\",\\\"NestedType\\\":\\\"{\\\\\\\"Key\\\\\\\":\\\\\\\"\\\\\\\",\\\\\\\"Fields\\\\\\\":[\\\\\\\"{\\\\\\\\\\\\\\\"Key\\\\\\\\\\\\\\\":\\\\\\\\\\\\\\\"a\\\\\\\\\\\\\\\",\\\\\\\\\\\\\\\"Annotation\\\\\\\\\\\\\\\":\\\\\\\\\\\\\\\"\\\\\\\\\\\\\\\"}\\\\\\\",\\\\\\\"{\\\\\\\\\\\\\\\"Key\\\\\\\\\\\\\\\":\\\\\\\\\\\\\\\"b\\\\\\\\\\\\\\\",\\\\\\\\\\\\\\\"Annotation\\\\\\\\\\\\\\\":\\\\\\\\\\\\\\\"\\\\\\\\\\\\\\\"}\\\\\\\"]}\\\"}\",\"{\\\"Key\\\":\\\"Field3\\\",\\\"Annotation\\\":\\\"\\\"}\",\"{\\\"Key\\\":\\\"Field4\\\",\\\"Annotation\\\":\\\"\\\"}\",\"{\\\"Key\\\":\\\"Field5\\\",\\\"Annotation\\\":\\\"\\\"}\",\"{\\\"Key\\\":\\\"Field6\\\",\\\"Annotation\\\":\\\"\\\"}\",\"{\\\"Key\\\":\\\"Field7\\\",\\\"Annotation\\\":\\\"\\\",\\\"NestedType\\\":\\\"{\\\\\\\"Key\\\\\\\":\\\\\\\"StructB\\\\\\\",\\\\\\\"Fields\\\\\\\":[\\\\\\\"{\\\\\\\\\\\\\\\"Key\\\\\\\\\\\\\\\":\\\\\\\\\\\\\\\"Field1\\\\\\\\\\\\\\\",\\\\\\\\\\\\\\\"Annotation\\\\\\\\\\\\\\\":\\\\\\\\\\\\\\\"\\\\\\\\\\\\\\\"}\\\\\\\"]}\\\"}\"]}"
}
