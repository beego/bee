package beeParser

import (
	"errors"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
)

// FieldFormatter transfers the field value to expected format
type Formatter interface {
	FieldFormatFunc(field *StructField) ([]byte, error)
	StructFormatFunc(node *StructNode) ([]byte, error)
	Marshal(root *StructNode) ([]byte, error)
}

// StructField defines struct field
type StructField struct {
	Name       string
	Type       types.Type
	NestedType *StructNode
	Comment    string
	Doc        string
	Tag        string
	FormatFunc func(field *StructField) ([]byte, error)
}

func (sf *StructField) MarshalText() ([]byte, error) {
	if sf.FormatFunc == nil {
		return nil, errors.New("format func is missing")
	}
	return sf.FormatFunc(sf)
}

func (sf *StructField) MarshalJSON() ([]byte, error) {
	if sf.FormatFunc == nil {
		return nil, errors.New("format func is missing")
	}
	return sf.FormatFunc(sf)
}

// StructNode defines struct node
type StructNode struct {
	Name       string
	Fields     []*StructField
	FormatFunc func(node *StructNode) ([]byte, error)
}

func (sn *StructNode) MarshalText() ([]byte, error) {
	if sn.FormatFunc == nil {
		return nil, errors.New("format func is missing")
	}

	return sn.FormatFunc(sn)
}

func (sn *StructNode) MarshalJSON() ([]byte, error) {
	if sn.FormatFunc == nil {
		return nil, errors.New("format func is missing")
	}

	return sn.FormatFunc(sn)
}

// StructParser parses structs in given file or string
type StructParser struct {
	MainStruct *StructNode
	Info       types.Info
	Formatter  Formatter
}

// NewStructParser is the constructor of StructParser
// filePath and src follow the same rule with go/parser.ParseFile
// If src != nil, ParseFile parses the source from src and the filename is only used when recording position information. The type of the argument for the src parameter must be string, []byte, or io.Reader. If src == nil, ParseFile parses the file specified by filename.
// rootStruct is the root struct we want to use
func NewStructParser(filePath string, src interface{}, rootStruct string, formatter Formatter) (*StructParser, error) {
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

	sp := &StructParser{
		Formatter: formatter,
		Info:      info,
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

		sp.MainStruct = sp.ParseStruct(structName, s)
		return false
	})

	if sp.MainStruct == nil {
		return nil, errors.New("non-exist root struct")
	}

	return sp, nil
}

// ParseField parses struct field in nested way
func (sp *StructParser) ParseField(field *ast.Field) *StructField {
	// ast.Print(nil, field)
	fieldName := field.Names[0].Name
	fieldType := sp.Info.TypeOf(field.Type)

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
		nestedStruct = sp.ParseStruct("", s)
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
		nestedStruct = sp.ParseStruct(ts.Name.Name, s)
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
		FormatFunc: sp.Formatter.FieldFormatFunc,
	}
}

// ParseStruct parses struct in nested way
func (sp *StructParser) ParseStruct(structName string, s *ast.StructType) *StructNode {
	fields := []*StructField{}
	for _, field := range s.Fields.List {
		parsedField := sp.ParseField(field)
		if parsedField != nil {
			fields = append(fields, parsedField)
		}
	}

	return &StructNode{
		Name:       structName,
		Fields:     fields,
		FormatFunc: sp.Formatter.StructFormatFunc,
	}
}

func (sp *StructParser) Marshal() ([]byte, error) {
	return sp.Formatter.Marshal(sp.MainStruct)
}
