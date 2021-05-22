package beeParser

import (
	"fmt"
	"log"
)

func ExampleConfigGenerator() {
	const src = `
package p
import "http"

type StructB struct {
	Field1 string
}
type StructA struct {
	Field1 string
	Field2 StructB
	Field3 []string
	Field4 map[string]string
	Field5 http.SameSite
}
`
	cg, err := NewConfigGenerator("./sample.go", src, "StructA")
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
	// }
}
