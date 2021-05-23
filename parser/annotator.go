package beeParser

import (
	"encoding/json"
	"strings"
)

type Annotator interface {
	Annotate(string) []map[string]interface{}
	AnnotateToJson(string) (string, error)
}

type Annotation struct {
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
func handleWhitespaceValues(values []string) []string {
	res := make([]string, 0)
	for _, v := range values {
		v = handleHeadWhitespace(v)
		v = handleTailWhitespace(v)
		res = append(res, v)
	}
	return res
}

//parse annotation to generate array with key and values
//start with "@" as a key-value pair,key and values are separated by a space,wrap to distinguish values.
func (a *Annotation) Annotate(comment string) []map[string]interface{} {
	results := make([]map[string]interface{}, 0)
	//split annotation with '@'
	lines := strings.Split(comment, "@")
	//skip first line whitespace
	for _, line := range lines[1:] {
		kvs := strings.Split(line, " ")
		key := kvs[0]
		values := strings.Split(strings.TrimSpace(line[len(kvs[0]):]), "\n")
		annotation := make(map[string]interface{})
		annotation[key] = handleWhitespaceValues(values)
		results = append(results, annotation)
	}
	return results
}

//parse annotation to json
func (a *Annotation) AnnotateToJson(comment string) (string, error) {
	annotate := a.Annotate(comment)
	result, err := json.MarshalIndent(annotate, "", "  ")
	return string(result), err
}
