package beeParser

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// type to default value
var builtInTypeMap = map[string]interface{}{
	"string": "",
	"int":    0,
	"int64":  0,
	"int32":  0,
	"uint":   0,
	"uint32": 0,
	"uint64": 0,
	"bool":   false,
	// @todo add more type
}

type StructField struct {
	Name       string
	Type       ast.Expr
	NestedType *StructNode
	Comment    string
	Doc        string
	Tag        string
}

func (sf *StructField) IsBuiltInType() bool {
	_, found := builtInTypeMap[fmt.Sprint(sf.Type)]
	return found
}

func (sf *StructField) Key() string {
	return sf.Name
}

func (sf *StructField) Value() interface{} {
	switch sf.Type.(type) {
	case *ast.Ident:
		val, found := builtInTypeMap[fmt.Sprint(sf.Type)]
		if found {
			return val
		}
		return sf.NestedType.ToKV()
	case *ast.ArrayType:
	case *ast.MapType:
	case *ast.SelectorExpr: // third party
	}

	return ""
}

type StructNode struct {
	Name   string
	Fields []*StructField
}

func (sn *StructNode) ToKV() map[string]interface{} {
	value := map[string]interface{}{}
	for _, field := range sn.Fields {
		value[field.Key()] = field.Value()
	}
	return value
}

type ConfigGenerator struct {
	StructMap  map[string]*StructNode // @todo key = {package}+{struct name}
	RootStruct string                 //match with the key of StructMap
}

func NewConfigGenerator(filePath string, src interface{}, rootStruct string) (*ConfigGenerator, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	structMap := map[string]*StructNode{}

	ast.Inspect(f, func(n ast.Node) bool {
		// ast.Print(nil, n)
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Type == nil {
			return true
		}

		structName := ts.Name.Name
		if structName != rootStruct {
			return true
		}

		s, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}

		structMap[structName] = ParseStruct(structName, s)

		return false
	})

	if _, found := structMap[rootStruct]; !found {
		return nil, errors.New("non-exist root struct")
	}

	return &ConfigGenerator{
		StructMap:  structMap,
		RootStruct: rootStruct,
	}, nil
}

func (cg *ConfigGenerator) ToJSON() ([]byte, error) {
	rootStruct := cg.StructMap[cg.RootStruct]
	value := rootStruct.ToKV()
	return json.MarshalIndent(value, "", "  ")
}

func ParseField(field *ast.Field) *StructField {
	fieldName := field.Names[0].Name
	fieldType := field.Type

	fieldTag := ""
	if field.Tag != nil {
		fieldTag = field.Tag.Value
	}
	fieldComment := ""
	if field.Comment != nil {
		fieldComment = field.Comment.Text()
	}
	fieldDoc := ""
	if field.Doc != nil {
		fieldDoc = field.Doc.Text()
	}

	switch field.Type.(type) {
	case *ast.Ident: // built-in or nested
		isNested := (field.Type.(*ast.Ident).Obj != nil)
		if !isNested {
			return &StructField{
				Name:    fieldName,
				Type:    fieldType,
				Tag:     fieldTag,
				Comment: fieldComment,
				Doc:     fieldDoc,
			}
		}
		ts, ok := field.Type.(*ast.Ident).Obj.Decl.(*ast.TypeSpec)
		if !ok || ts.Type == nil {
			return nil
		}

		s, ok := ts.Type.(*ast.StructType)
		if !ok {
			return nil
		}
		return &StructField{
			Name:       fieldName,
			Type:       fieldType,
			Tag:        fieldTag,
			Comment:    fieldComment,
			Doc:        fieldDoc,
			NestedType: ParseStruct(ts.Name.Name, s),
		}
	case *ast.ArrayType:
	case *ast.MapType:
	case *ast.SelectorExpr: // third party
	}

	return &StructField{
		Name:    fieldName,
		Type:    fieldType,
		Tag:     fieldTag,
		Comment: fieldComment,
		Doc:     fieldDoc,
	}
}

func ParseStruct(structName string, s *ast.StructType) *StructNode {
	fields := []*StructField{}
	for _, field := range s.Fields.List {
		parsedField := ParseField(field)
		if parsedField != nil {
			fields = append(fields, parsedField)
		}
	}

	return &StructNode{
		Name:   structName,
		Fields: fields,
	}
}
