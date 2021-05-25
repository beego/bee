package beeParser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormat(t *testing.T) {
	except := `{
		"Name": [
			"Field1"
		],
		"Path":[
			"https://github.com/beego/bee",
			"https://github.com/beego"
		],
		"test":[
			"test comment"
		]
	}`

	field := &StructField{
		Comment: "@test test comment",
		Doc: `@Name Field1
		@Path https://github.com/beego/bee
			  https://github.com/beego`,
	}

	actual := NewAnnotationFormatter().Format(field)

	assert.JSONEq(t, except, actual)
}
