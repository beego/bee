package beeParser

import (
	"fmt"
	"log"
)

const src = `
package p

type StructA struct {
	// @Key Field1
	// @Default test
	// @Description ddddddd
	Field1 string
	// @Key Field2
	Field2 struct{
		// @Key a
		// @Default https://github.com/beego/bee
		// 			https://github.com/beego
		a string
		// @Key b
		// @Default https://github.com/beego/bee https://github.com/beego
		b string
	}
	// @Key Field3
	// @Default 1
	Field3 int
	// @Key Field4
	// @Default false
	Field4 bool
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
	//		"Field3": 1
	//	},
	//	{
	//		"Field4": false
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
	//|
	//   - |
	//     Field1: test
	//   - |
	//     Field2: |
	//   	 - |
	//   	   a:
	//   	   - https://github.com/beego/bee
	//   	   - https://github.com/beego
	//   	 - |
	//   	   b: https://github.com/beego/bee https://github.com/beego
	//   - |
	//     Field3: 1
	//   - |
	//     Field4: false
}

func ExampleXmlFormatter() {
	sp, err := NewStructParser("src.go", src, "StructA", &XmlFormatter{})
	if err != nil {
		log.Fatal(err)
	}

	b, err := sp.Marshal()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

	// Output:
	//<Field1>test<!--ddddddd--></Field1>
	//<Field2><a></a>
	//<b>https://github.com/beego/bee https://github.com/beego</b>
	//</Field2>
	//<Field3>1</Field3>
	//<Field4>false</Field4>
}
