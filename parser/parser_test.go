package beeParser

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ExampleStructParser() {
	const src = `
package p

import (
	"net/http"
)

type StructB struct {
	Field1 string
}
type StructA struct {
	Field1 string
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
	annotator := &Annotator{}

	sp, err := NewStructParser("src.go", src, "StructA", annotator)
	if err != nil {
		log.Fatal(err)
	}

	b, err := sp.ToJSON()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

	// Output:
	// {
	//   "Field1": "",
	//   "Field2": {
	//     "a": "",
	//     "b": ""
	//   },
	//   "Field3": "",
	//   "Field4": "",
	//   "Field5": "",
	//   "Field6": "",
	//   "Field7": {
	//     "Field1": ""
	//   }
	// }
}

func TestParseStructByFieldAnnotation(t *testing.T) {
	const src = `
package p

type StructA struct {
	//@Name Field1
	//@DefaultValues bee test
	//				 beego test
	Field1 string
}
`

	expect := `[
  {
    "Name": [
      "Field1"
    ]
  },
  {
    "DefaultValues": [
      "bee test",
      "beego test"
    ]
  }
]`

	annotator := &Annotator{}

	sp, err := NewStructParser("src.go", src, "StructA", annotator)
	if err != nil {
		log.Fatal(err)
	}

	actual := sp.FieldFormatter.Format(sp.MainStruct.Fields[0])

	assert.Equal(t, expect, actual)
}
