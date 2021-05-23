package beeParser

import (
	"fmt"
	"log"
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
	cg, err := NewStructParser("src.go", src, "StructA")
	if err != nil {
		log.Fatal(err)
	}

	b, err := cg.ToJSON()
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
