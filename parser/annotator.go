package beeParser

import (
	"fmt"
	"go/types"
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

// Handle value to remove head and tail space.
func handleWhitespaceValues(values []string) []interface{} {
	res := make([]interface{}, 0)
	for _, v := range values {
		v = handleHeadWhitespace(v)
		v = handleTailWhitespace(v)
		res = append(res, transferType(v))
	}
	return res
}

// Transfer string to original type
func transferType(str string) interface{} {
	if res, err := strconv.Atoi(str); err == nil {
		return res
	}
	if res, err := strconv.ParseBool(str); err == nil {
		return res
	}
	return str
}

// Parse annotation to generate array with key and values
// start with "@" as a key-value pair,key and values are separated by a space,wrap to distinguish values.
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

// Create new annotation,parse "Key","Default","Description" by annotation.
// If key and default value is empty by annotaion, set default key and value
// by params, default value according defaultType to generate
func NewAnnotation(annotation, defaultKey string, defaultType types.Type) *Annotation {
	a := &Annotation{}
	kvs := a.Annotate(annotation)
	if v, ok := kvs["Key"]; ok {
		a.Key = fmt.Sprintf("%v", v)
	}
	if v, ok := kvs["Description"]; ok {
		if ss, ok := v.([]interface{}); ok {
			for i, s := range ss {
				if i == 0 {
					a.Description += s.(string)
					continue
				}
				a.Description += "\n" + s.(string)
			}
		} else {
			a.Description = fmt.Sprintf("%v", v)
		}
	}
	if v, ok := kvs["Default"]; ok {
		a.Default = v
	}
	if a.Key == "" {
		//if key by parse is empty, set a default key
		a.Key = defaultKey
	}
	if a.Default == nil {
		//if default value is nil, set the default value according to the defaultType
		a.Default = getDefaultValue(defaultType)
	}
	return a
}

// Get the default value according to the t, process bool/string/int
func getDefaultValue(t types.Type) interface{} {
	switch tys := t.(type) {
	case *types.Basic:
		switch tys.Kind() {
		case types.Bool:
			return false
		case types.Int, types.Int16, types.Int8, types.Int32, types.Int64, types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr, types.Float32, types.Float64:
			return 0
		case types.String:
			return ""
		}
	}
	return nil
}
