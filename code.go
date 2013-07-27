// Copyright 2011 Gary Burd
// Copyright 2013 Unknown
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"go/ast"
	"go/printer"
	"go/scanner"
	"go/token"
	"math"
	"strconv"
)

const (
	notPredeclared = iota
	predeclaredType
	predeclaredConstant
	predeclaredFunction
)

// predeclared represents the set of all predeclared identifiers.
var predeclared = map[string]int{
	"bool":       predeclaredType,
	"byte":       predeclaredType,
	"complex128": predeclaredType,
	"complex64":  predeclaredType,
	"error":      predeclaredType,
	"float32":    predeclaredType,
	"float64":    predeclaredType,
	"int16":      predeclaredType,
	"int32":      predeclaredType,
	"int64":      predeclaredType,
	"int8":       predeclaredType,
	"int":        predeclaredType,
	"rune":       predeclaredType,
	"string":     predeclaredType,
	"uint16":     predeclaredType,
	"uint32":     predeclaredType,
	"uint64":     predeclaredType,
	"uint8":      predeclaredType,
	"uint":       predeclaredType,
	"uintptr":    predeclaredType,

	"true":  predeclaredConstant,
	"false": predeclaredConstant,
	"iota":  predeclaredConstant,
	"nil":   predeclaredConstant,

	"append":  predeclaredFunction,
	"cap":     predeclaredFunction,
	"close":   predeclaredFunction,
	"complex": predeclaredFunction,
	"copy":    predeclaredFunction,
	"delete":  predeclaredFunction,
	"imag":    predeclaredFunction,
	"len":     predeclaredFunction,
	"make":    predeclaredFunction,
	"new":     predeclaredFunction,
	"panic":   predeclaredFunction,
	"print":   predeclaredFunction,
	"println": predeclaredFunction,
	"real":    predeclaredFunction,
	"recover": predeclaredFunction,
}

const (
	ExportLinkAnnotation AnnotationKind = iota
	AnchorAnnotation
	CommentAnnotation
	PackageLinkAnnotation
	BuiltinAnnotation
)

// annotationVisitor collects annotations.
type annotationVisitor struct {
	annotations []Annotation
}

func (v *annotationVisitor) add(kind AnnotationKind, importPath string) {
	v.annotations = append(v.annotations, Annotation{Kind: kind, ImportPath: importPath})
}

func (v *annotationVisitor) ignoreName() {
	v.add(-1, "")
}

func (v *annotationVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.TypeSpec:
		v.ignoreName()
		ast.Walk(v, n.Type)
	case *ast.FuncDecl:
		if n.Recv != nil {
			ast.Walk(v, n.Recv)
		}
		v.ignoreName()
		ast.Walk(v, n.Type)
	case *ast.Field:
		for _ = range n.Names {
			v.ignoreName()
		}
		ast.Walk(v, n.Type)
	case *ast.ValueSpec:
		for _ = range n.Names {
			v.add(AnchorAnnotation, "")
		}
		if n.Type != nil {
			ast.Walk(v, n.Type)
		}
		for _, x := range n.Values {
			ast.Walk(v, x)
		}
	case *ast.Ident:
		switch {
		case n.Obj == nil && predeclared[n.Name] != notPredeclared:
			v.add(BuiltinAnnotation, "")
		case n.Obj != nil && ast.IsExported(n.Name):
			v.add(ExportLinkAnnotation, "")
		default:
			v.ignoreName()
		}
	case *ast.SelectorExpr:
		if x, _ := n.X.(*ast.Ident); x != nil {
			if obj := x.Obj; obj != nil && obj.Kind == ast.Pkg {
				if spec, _ := obj.Decl.(*ast.ImportSpec); spec != nil {
					if path, err := strconv.Unquote(spec.Path.Value); err == nil {
						v.add(PackageLinkAnnotation, path)
						if path == "C" {
							v.ignoreName()
						} else {
							v.add(ExportLinkAnnotation, path)
						}
						return nil
					}
				}
			}
		}
		ast.Walk(v, n.X)
		v.ignoreName()
	default:
		return v
	}
	return nil
}

func printDecl(decl ast.Node, fset *token.FileSet, buf []byte) (Code, []byte) {
	v := &annotationVisitor{}
	ast.Walk(v, decl)

	buf = buf[:0]
	err := (&printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}).Fprint(sliceWriter{&buf}, fset, decl)
	if err != nil {
		return Code{Text: err.Error()}, buf
	}

	var annotations []Annotation
	var s scanner.Scanner
	fset = token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(buf))
	s.Init(file, buf, nil, scanner.ScanComments)
loop:
	for {
		pos, tok, lit := s.Scan()
		switch tok {
		case token.EOF:
			break loop
		case token.COMMENT:
			p := file.Offset(pos)
			e := p + len(lit)
			if p > math.MaxInt16 || e > math.MaxInt16 {
				break loop
			}
			annotations = append(annotations, Annotation{Kind: CommentAnnotation, Pos: int16(p), End: int16(e)})
		case token.IDENT:
			if len(v.annotations) == 0 {
				// Oops!
				break loop
			}
			annotation := v.annotations[0]
			v.annotations = v.annotations[1:]
			if annotation.Kind == -1 {
				continue
			}
			p := file.Offset(pos)
			e := p + len(lit)
			if p > math.MaxInt16 || e > math.MaxInt16 {
				break loop
			}
			annotation.Pos = int16(p)
			annotation.End = int16(e)
			if len(annotations) > 0 && annotation.Kind == ExportLinkAnnotation {
				prev := annotations[len(annotations)-1]
				if prev.Kind == PackageLinkAnnotation &&
					prev.ImportPath == annotation.ImportPath &&
					prev.End+1 == annotation.Pos {
					// merge with previous
					annotation.Pos = prev.Pos
					annotations[len(annotations)-1] = annotation
					continue loop
				}
			}
			annotations = append(annotations, annotation)
		}
	}
	return Code{Text: string(buf), Annotations: annotations}, buf
}

type AnnotationKind int16

type Annotation struct {
	Pos, End   int16
	Kind       AnnotationKind
	ImportPath string
}

type Code struct {
	Text        string
	Annotations []Annotation
}

func commentAnnotations(src string) []Annotation {
	var annotations []Annotation
	var s scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	s.Init(file, []byte(src), nil, scanner.ScanComments)
	for {
		pos, tok, lit := s.Scan()
		switch tok {
		case token.EOF:
			return annotations
		case token.COMMENT:
			p := file.Offset(pos)
			e := p + len(lit)
			if p > math.MaxInt16 || e > math.MaxInt16 {
				return annotations
			}
			annotations = append(annotations, Annotation{Kind: CommentAnnotation, Pos: int16(p), End: int16(e)})
		}
	}
	return nil
}

type sliceWriter struct{ p *[]byte }

func (w sliceWriter) Write(p []byte) (int, error) {
	*w.p = append(*w.p, p...)
	return len(p), nil
}
