package beeParser

import (
	"encoding/json"
	"errors"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
)

// StructField defines struct field
type StructField struct {
	Name       string
	Type       types.Type
	NestedType *StructNode
	Comment    string
	Doc        string
	Tag        string
}

// Key returns the key of the field
func (sf *StructField) Key() string {
	return sf.Name
}

// Value returns the value of the field
// if the field contains nested struct, it will return a nested result
func (sf *StructField) Value() interface{} {
	if sf.NestedType != nil {
		return sf.NestedType.ToKV()
	}

	return ""
}

// StructNode defines struct node
type StructNode struct {
	Name   string
	Fields []*StructField
}

// ToKV transfers struct to key value pair
func (sn *StructNode) ToKV() map[string]interface{} {
	value := map[string]interface{}{}
	for _, field := range sn.Fields {
		value[field.Key()] = field.Value()
	}
	return value
}

// StructParser parses structs in given file or string
type StructParser struct {
	MainStruct *StructNode
	Info       types.Info
}

// NewStructParser is the constructor of StructParser
// filePath and src follow the same rule with go/parser.ParseFile
// If src != nil, ParseFile parses the source from src and the filename is only used when recording position information. The type of the argument for the src parameter must be string, []byte, or io.Reader. If src == nil, ParseFile parses the file specified by filename.
// rootStruct is the root struct we want to use
func NewStructParser(filePath string, src interface{}, rootStruct string) (*StructParser, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	info := types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{
		Importer: importer.ForCompiler(fset, "source", nil),
	}
	_, err = conf.Check("src", fset, []*ast.File{f}, &info)
	if err != nil {
		return nil, err
	}

	cg := &StructParser{
		Info: info,
	}

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

		cg.MainStruct = cg.ParseStruct(structName, s)
		return false
	})

	if cg.MainStruct == nil {
		return nil, errors.New("non-exist root struct")
	}

	return cg, nil
}

func (cg *StructParser) ToJSON() ([]byte, error) {
	value := cg.MainStruct.ToKV()
	return json.MarshalIndent(value, "", "  ")
}

// ParseField parses struct field in nested way
func (cg *StructParser) ParseField(field *ast.Field) *StructField {
	// ast.Print(nil, field)
	fieldName := field.Names[0].Name
	fieldType := cg.Info.TypeOf(field.Type)

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

	var nestedStruct *StructNode
	if s, isInlineStruct := field.Type.(*ast.StructType); isInlineStruct {
		nestedStruct = cg.ParseStruct("", s)
	}

	if _, isNamedStructorBasic := field.Type.(*ast.Ident); isNamedStructorBasic && field.Type.(*ast.Ident).Obj != nil {
		ts, ok := field.Type.(*ast.Ident).Obj.Decl.(*ast.TypeSpec)
		if !ok || ts.Type == nil {
			return nil
		}

		s, ok := ts.Type.(*ast.StructType)
		if !ok {
			return nil
		}
		nestedStruct = cg.ParseStruct(ts.Name.Name, s)
	}
	// fieldType.(*types.Basic) // basic type
	// *ast.ArrayType:
	// *ast.MapType:
	// *ast.SelectorExpr: // third party

	return &StructField{
		Name:       fieldName,
		Type:       fieldType,
		Tag:        fieldTag,
		Comment:    fieldComment,
		Doc:        fieldDoc,
		NestedType: nestedStruct,
	}
}

// ParseStruct parses struct in nested way
func (cg *StructParser) ParseStruct(structName string, s *ast.StructType) *StructNode {
	fields := []*StructField{}
	for _, field := range s.Fields.List {
		parsedField := cg.ParseField(field)
		if parsedField != nil {
			fields = append(fields, parsedField)
		}
	}

	return &StructNode{
		Name:   structName,
		Fields: fields,
	}
}
