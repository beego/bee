package beeParser

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var BeeAnnotator Annotator

const (
	Annotation1 = `
		@Name Field1
		@Type string
		@Path https://github.com/beego/bee
			  https://github.com/beego
	`
	Annotation2 = `
		@Number 2
		@Projects 	https://github.com/beego/bee

https://github.com/beego
	`
)

func TestMain(m *testing.M) {
	BeeAnnotator = &Annotation{}
	retCode := m.Run() //run test
	os.Exit(retCode)
}

func TestAnnotate(t *testing.T) {
	expect1 := map[string]interface{}{
		"Name": "Field1",
		"Type": "string",
		"Path": []interface{}{"https://github.com/beego/bee", "https://github.com/beego"},
	}

	expect2 := map[string]interface{}{
		"Number":   2,
		"Projects": []interface{}{"https://github.com/beego/bee", "", "https://github.com/beego"},
	}

	actual := BeeAnnotator.Annotate(Annotation1)
	actual2 := BeeAnnotator.Annotate(Annotation2)

	assert.Equal(t, expect1, actual)
	assert.Equal(t, expect2, actual2)
}

func TestHandleWhitespaceValues(t *testing.T) {
	src := []string{
		"    beego",
		"",
		"  	bee 	",
		"  	bee beego 	",
		"  	1	",
		" false	",
	}

	expect := []interface{}{
		"beego",
		"",
		"bee",
		"bee beego",
		1,
		false,
	}

	actual := handleWhitespaceValues(src)

	assert.Equal(t, expect, actual)
}

//benchmark test
func BenchmarkAnnotate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BeeAnnotator.Annotate(Annotation1)
	}
}
