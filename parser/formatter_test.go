package beeParser

import (
	"fmt"
	"log"
)

const src = `
package p

type StructB struct {
	// @Key FieldB1
	FieldB1 interface{}
}

type StructA struct {
	// @Key Field1
	// @Default test
	// @Description comment of field1
	Field1 string
	// @Key Field2
	// @Description comment of field2
	Field2 struct{
		// @Key a
		// @Default https://github.com/beego/bee
		// 			https://github.com/beego
		// @Description comment of a of field2
		a string
		// @Key b
		// @Default https://github.com/beego/bee https://github.com/beego
		// @Description comment of b of field2
		b map[int]string
	}
	// @Description comment of field3
	Field3 int
	// @Default false
	Field4 bool
	// @Key NestField
	// @Description comment of NestField
	NestField StructB
}
`

func ExampleJsonFormatter() {
	sp, err := NewStructParser("src.go", src, "StructA", &JsonFormatter{})
	if err != nil {
		log.Fatal(err)
	}

	b, err := sp.Marshal()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

	// Output:
	//[
	//	{
	//		"Field1": "test"
	//	},
	//	{
	//		"Field2": [
	//			{
	//				"a": [
	//					"https://github.com/beego/bee",
	//					"https://github.com/beego"
	//				]
	//			},
	//			{
	//				"b": "https://github.com/beego/bee https://github.com/beego"
	//			}
	//		]
	//	},
	//	{
	//		"Field3": null
	//	},
	//	{
	//		"Field4": false
	//	},
	//	{
	//		"NestField": [
	//			{
	//				"FieldB1": null
	//			}
	//		]
	//	}
	//]
}

func ExampleYamlFormatter() {
	sp, err := NewStructParser("src.go", src, "StructA", &YamlFormatter{})
	if err != nil {
		log.Fatal(err)
	}

	b, err := sp.Marshal()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

	// Output:
	// # comment of field1
	// Field1: test
	// # comment of b of field2
	// Field2: |
	//     # comment of a of field2
	//     a:
	//         - https://github.com/beego/bee
	//         - https://github.com/beego
	//     # comment of b of field2
	//     b: https://github.com/beego/bee https://github.com/beego
	// # comment of field3
	// Field3: null
	// Field4: false
	// NestField: |
	//     FieldB1: null
}
