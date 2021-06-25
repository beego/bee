package beeParser

import (
	"fmt"
	"strconv"
	"strings"
)

type Annotator interface {
	Annotate(string) map[string]interface{}
}

type Annotation struct {
	Key, Description string
	Default          interface{}
}

func isWhitespace(ch byte) bool { return ch == ' ' || ch == '\t' || ch == '\r' }

func handleHeadWhitespace(s string) string {
	i := 0
	for i < len(s) && isWhitespace(s[i]) {
		i++
	}
	return s[i:]
}

func handleTailWhitespace(s string) string {
	i := len(s)
	for i > 0 && isWhitespace(s[i-1]) {
		i--
	}
	return s[0:i]
}

//handle value to remove head and tail space.
func handleWhitespaceValues(values []string) []interface{} {
	res := make([]interface{}, 0)
	for _, v := range values {
		v = handleHeadWhitespace(v)
		v = handleTailWhitespace(v)
		res = append(res, transferType(v))
	}
	return res
}

//try to transfer string to original type
func transferType(str string) interface{} {
	if res, err := strconv.Atoi(str); err == nil {
		return res
	}
	if res, err := strconv.ParseBool(str); err == nil {
		return res
	}
	return str
}

//parse annotation to generate array with key and values
//start with "@" as a key-value pair,key and values are separated by a space,wrap to distinguish values.
func (a *Annotation) Annotate(annotation string) map[string]interface{} {
	results := make(map[string]interface{})
	//split annotation with '@'
	lines := strings.Split(annotation, "@")
	//skip first line whitespace
	for _, line := range lines[1:] {
		kvs := strings.Split(line, " ")
		key := kvs[0]
		values := strings.Split(strings.TrimSpace(line[len(kvs[0]):]), "\n")
		if len(values) == 1 {
			results[key] = handleWhitespaceValues(values)[0]
		} else {
			results[key] = handleWhitespaceValues(values)
		}
	}
	return results
}

func NewAnnotation(annotation string) *Annotation {
	a := &Annotation{}
	kvs := a.Annotate(annotation)
	if v, ok := kvs["Key"]; ok {
		a.Key = fmt.Sprintf("%v", v)
	}
	if v, ok := kvs["Description"]; ok {
		a.Description = fmt.Sprintf("%v", v)
	}
	if v, ok := kvs["Default"]; ok {
		a.Default = v
	}
	return a
}
