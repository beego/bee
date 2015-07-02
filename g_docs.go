// Copyright 2013 bee authors
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
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"github.com/astaxie/beego/swagger"
	"github.com/astaxie/beego/utils"
)

var globalDocsTemplate = `package docs

import (
	"encoding/json"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/swagger"
)

const (
    Rootinfo string = {{.rootinfo}}
    Subapi string = {{.subapi}}
    BasePath string= "{{.version}}"
)

var rootapi swagger.ResourceListing
var apilist map[string]*swagger.ApiDeclaration

func init() {
	if beego.EnableDocs {
		err := json.Unmarshal([]byte(Rootinfo), &rootapi)
		if err != nil {
			beego.Error(err)
		}
		err = json.Unmarshal([]byte(Subapi), &apilist)
		if err != nil {
			beego.Error(err)
		}
		beego.GlobalDocApi["Root"] = rootapi
		for k, v := range apilist {
			for i, a := range v.Apis {
				a.Path = urlReplace(k + a.Path)
				v.Apis[i] = a
			}
			v.BasePath = BasePath
			beego.GlobalDocApi[strings.Trim(k, "/")] = v
		}
	}
}


func urlReplace(src string) string {
	pt := strings.Split(src, "/")
	for i, p := range pt {
		if len(p) > 0 {
			if p[0] == ':' {
				pt[i] = "{" + p[1:] + "}"
			} else if p[0] == '?' && p[1] == ':' {
				pt[i] = "{" + p[2:] + "}"
			}
		}
	}
	return strings.Join(pt, "/")
}
`

const (
	ajson  = "application/json"
	axml   = "application/xml"
	aplain = "text/plain"
	ahtml  = "text/html"
)

var pkgCache map[string]bool //pkg:controller:function:comments comments: key:value
var controllerComments map[string]string
var importlist map[string]string
var apilist map[string]*swagger.ApiDeclaration
var controllerList map[string][]swagger.Api
var modelsList map[string]map[string]swagger.Model
var rootapi swagger.ResourceListing

func init() {
	pkgCache = make(map[string]bool)
	controllerComments = make(map[string]string)
	importlist = make(map[string]string)
	apilist = make(map[string]*swagger.ApiDeclaration)
	controllerList = make(map[string][]swagger.Api)
	modelsList = make(map[string]map[string]swagger.Model)
}

func generateDocs(curpath string) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, path.Join(curpath, "routers", "router.go"), nil, parser.ParseComments)

	if err != nil {
		ColorLog("[ERRO] parse router.go error\n")
		os.Exit(2)
	}

	rootapi.Infos = swagger.Infomation{}
	rootapi.SwaggerVersion = swagger.SwaggerVersion
	//analysis API comments
	if f.Comments != nil {
		for _, c := range f.Comments {
			for _, s := range strings.Split(c.Text(), "\n") {
				if strings.HasPrefix(s, "@APIVersion") {
					rootapi.ApiVersion = strings.TrimSpace(s[len("@APIVersion"):])
				} else if strings.HasPrefix(s, "@Title") {
					rootapi.Infos.Title = strings.TrimSpace(s[len("@Title"):])
				} else if strings.HasPrefix(s, "@Description") {
					rootapi.Infos.Description = strings.TrimSpace(s[len("@Description"):])
				} else if strings.HasPrefix(s, "@TermsOfServiceUrl") {
					rootapi.Infos.TermsOfServiceUrl = strings.TrimSpace(s[len("@TermsOfServiceUrl"):])
				} else if strings.HasPrefix(s, "@Contact") {
					rootapi.Infos.Contact = strings.TrimSpace(s[len("@Contact"):])
				} else if strings.HasPrefix(s, "@License") {
					rootapi.Infos.License = strings.TrimSpace(s[len("@License"):])
				} else if strings.HasPrefix(s, "@LicenseUrl") {
					rootapi.Infos.LicenseUrl = strings.TrimSpace(s[len("@LicenseUrl"):])
				}
			}
		}
	}
	for _, im := range f.Imports {
		localName := ""
		if im.Name != nil {
			localName = im.Name.Name
		}
		analisyscontrollerPkg(localName, im.Path.Value)
	}
	for _, d := range f.Decls {
		switch specDecl := d.(type) {
		case *ast.FuncDecl:
			for _, l := range specDecl.Body.List {
				switch smtp := l.(type) {
				case *ast.AssignStmt:
					for _, l := range smtp.Rhs {
						if v, ok := l.(*ast.CallExpr); ok {
							f, params := analisysNewNamespace(v)
							globalDocsTemplate = strings.Replace(globalDocsTemplate, "{{.version}}", f, -1)
							for _, p := range params {
								switch pp := p.(type) {
								case *ast.CallExpr:
									if selname := pp.Fun.(*ast.SelectorExpr).Sel.String(); selname == "NSNamespace" {
										s, params := analisysNewNamespace(pp)
										subapi := swagger.ApiRef{Path: s}
										controllerName := ""
										for _, sp := range params {
											switch pp := sp.(type) {
											case *ast.CallExpr:
												if pp.Fun.(*ast.SelectorExpr).Sel.String() == "NSInclude" {
													controllerName = analisysNSInclude(s, pp)
												}
											}
										}
										if v, ok := controllerComments[controllerName]; ok {
											subapi.Description = v
										}
										rootapi.Apis = append(rootapi.Apis, subapi)
									} else if selname == "NSInclude" {
										analisysNSInclude(f, pp)
									}
								}
							}
						}

					}
				}
			}
		}
	}
	apiinfo, err := json.Marshal(rootapi)
	if err != nil {
		panic(err)
	}
	subapi, err := json.Marshal(apilist)
	if err != nil {
		panic(err)
	}
	os.Mkdir(path.Join(curpath, "docs"), 0755)
	fd, err := os.Create(path.Join(curpath, "docs", "docs.go"))
	if err != nil {
		panic(err)
	}
	defer fd.Close()
	a := strings.Replace(globalDocsTemplate, "{{.rootinfo}}", "`"+string(apiinfo)+"`", -1)
	a = strings.Replace(a, "{{.subapi}}", "`"+string(subapi)+"`", -1)
	fd.WriteString(a)
}

func analisysNewNamespace(ce *ast.CallExpr) (first string, others []ast.Expr) {
	for i, p := range ce.Args {
		if i == 0 {
			switch pp := p.(type) {
			case *ast.BasicLit:
				first = strings.Trim(pp.Value, `"`)
			}
			continue
		}
		others = append(others, p)
	}
	return
}

func analisysNSInclude(baseurl string, ce *ast.CallExpr) string {
	cname := ""
	a := &swagger.ApiDeclaration{}
	a.ApiVersion = rootapi.ApiVersion
	a.SwaggerVersion = swagger.SwaggerVersion
	a.ResourcePath = baseurl
	a.Produces = []string{"application/json", "application/xml", "text/plain", "text/html"}
	a.Apis = make([]swagger.Api, 0)
	a.Models = make(map[string]swagger.Model)
	for _, p := range ce.Args {
		x := p.(*ast.UnaryExpr).X.(*ast.CompositeLit).Type.(*ast.SelectorExpr)
		if v, ok := importlist[fmt.Sprint(x.X)]; ok {
			cname = v + x.Sel.Name
		}
		if apis, ok := controllerList[cname]; ok {
			if len(a.Apis) > 0 {
				a.Apis = append(a.Apis, apis...)
			} else {
				a.Apis = apis
			}
		}
		if models, ok := modelsList[cname]; ok {
			for _, m := range models {
				a.Models[m.Id] = m
			}
		}
	}
	apilist[baseurl] = a
	return cname
}

func analisyscontrollerPkg(localName, pkgpath string) {
	pkgpath = strings.Trim(pkgpath, "\"")
	if isSystemPackage(pkgpath) {
		return
	}
	if localName != "" {
		importlist[localName] = pkgpath
	} else {
		pps := strings.Split(pkgpath, "/")
		importlist[pps[len(pps)-1]] = pkgpath
	}
	if pkgpath == "github.com/astaxie/beego" {
		return
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		panic("please set gopath")
	}
	pkgRealpath := ""

	wgopath := filepath.SplitList(gopath)
	for _, wg := range wgopath {
		wg, _ = filepath.EvalSymlinks(filepath.Join(wg, "src", pkgpath))
		if utils.FileExists(wg) {
			pkgRealpath = wg
			break
		}
	}
	if pkgRealpath != "" {
		if _, ok := pkgCache[pkgpath]; ok {
			return
		}
	} else {
		ColorLog("[ERRO] the %s pkg not exist in gopath\n", pkgpath)
		os.Exit(1)
	}
	fileSet := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fileSet, pkgRealpath, func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
	}, parser.ParseComments)

	if err != nil {
		ColorLog("[ERRO] the %s pkg parser.ParseDir error\n", pkgpath)
		os.Exit(1)
	}
	for _, pkg := range astPkgs {
		for _, fl := range pkg.Files {
			for _, d := range fl.Decls {
				switch specDecl := d.(type) {
				case *ast.FuncDecl:
					if specDecl.Recv != nil && len(specDecl.Recv.List) > 0 {
						if t, ok := specDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
							parserComments(specDecl.Doc, specDecl.Name.String(), fmt.Sprint(t.X), pkgpath)
						}
					}
				case *ast.GenDecl:
					if specDecl.Tok.String() == "type" {
						for _, s := range specDecl.Specs {
							switch tp := s.(*ast.TypeSpec).Type.(type) {
							case *ast.StructType:
								_ = tp.Struct
								controllerComments[pkgpath+s.(*ast.TypeSpec).Name.String()] = specDecl.Doc.Text()
							}
						}
					}
				}
			}
		}
	}
}

func isSystemPackage(pkgpath string) bool {
	goroot := runtime.GOROOT()
	if goroot == "" {
		panic("goroot is empty, do you install Go right?")
	}
	wg, _ := filepath.EvalSymlinks(filepath.Join(goroot, "src", "pkg", pkgpath))
	if utils.FileExists(wg) {
		return true
	}

	//TODO(zh):support go1.4
	wg, _ = filepath.EvalSymlinks(filepath.Join(goroot, "src", pkgpath))
	if utils.FileExists(wg) {
		return true
	}

	return false
}

// parse the func comments
func parserComments(comments *ast.CommentGroup, funcName, controllerName, pkgpath string) error {
	innerapi := swagger.Api{}
	opts := swagger.Operation{}
	if comments != nil && comments.List != nil {
		for _, c := range comments.List {
			t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
			if strings.HasPrefix(t, "@router") {
				elements := strings.TrimSpace(t[len("@router"):])
				e1 := strings.SplitN(elements, " ", 2)
				if len(e1) < 1 {
					return errors.New("you should has router infomation")
				}
				innerapi.Path = e1[0]
				if len(e1) == 2 && e1[1] != "" {
					e1 = strings.SplitN(e1[1], " ", 2)
					opts.HttpMethod = strings.ToUpper(strings.Trim(e1[0], "[]"))
				} else {
					opts.HttpMethod = "GET"
				}
			} else if strings.HasPrefix(t, "@Title") {
				opts.Nickname = strings.TrimSpace(t[len("@Title"):])
			} else if strings.HasPrefix(t, "@Description") {
				opts.Summary = strings.TrimSpace(t[len("@Description"):])
			} else if strings.HasPrefix(t, "@Success") {
				ss := strings.TrimSpace(t[len("@Success"):])
				rs := swagger.ResponseMessage{}
				st := make([]string, 3)
				j := 0
				var tmp []rune
				start := false

				for i, c := range ss {
					if unicode.IsSpace(c) {
						if !start && j < 2 {
							continue
						}
						if j == 0 || j == 1 {
							st[j] = string(tmp)
							tmp = make([]rune, 0)
							j += 1
							start = false
							continue
						} else {
							st[j] = strings.TrimSpace(ss[i+1:])
							break
						}
					} else {
						start = true
						tmp = append(tmp, c)
					}
				}
				if len(tmp) > 0 && st[2] == "" {
					st[2] = strings.TrimSpace(string(tmp))
				}
				rs.Message = st[2]
				if st[1] == "{object}" {
					if st[2] == "" {
						panic(controllerName + " " + funcName + " has no object")
					}
					cmpath, m, mod, realTypes := getModel(st[2])
					//ll := strings.Split(st[2], ".")
					//opts.Type = ll[len(ll)-1]
					rs.ResponseModel = m
					if _, ok := modelsList[pkgpath+controllerName]; !ok {
						modelsList[pkgpath+controllerName] = make(map[string]swagger.Model, 0)
					}
					modelsList[pkgpath+controllerName][st[2]] = mod
					appendModels(cmpath, pkgpath, controllerName, realTypes)
				}

				rs.Code, _ = strconv.Atoi(st[0])
				opts.ResponseMessages = append(opts.ResponseMessages, rs)
			} else if strings.HasPrefix(t, "@Param") {
				para := swagger.Parameter{}
				p := getparams(strings.TrimSpace(t[len("@Param "):]))
				if len(p) < 4 {
					panic(controllerName + "_" + funcName + "'s comments @Param at least should has 4 params")
				}
				para.Name = p[0]
				para.ParamType = p[1]
				pp := strings.Split(p[2], ".")
				para.DataType = pp[len(pp)-1]
				if len(p) > 4 {
					para.Required, _ = strconv.ParseBool(p[3])
					para.Description = p[4]
				} else {
					para.Description = p[3]
				}
				opts.Parameters = append(opts.Parameters, para)
			} else if strings.HasPrefix(t, "@Failure") {
				rs := swagger.ResponseMessage{}
				st := strings.TrimSpace(t[len("@Failure"):])
				var cd []rune
				var start bool
				for i, s := range st {
					if unicode.IsSpace(s) {
						if start {
							rs.Message = strings.TrimSpace(st[i+1:])
							break
						} else {
							continue
						}
					}
					start = true
					cd = append(cd, s)
				}
				rs.Code, _ = strconv.Atoi(string(cd))
				opts.ResponseMessages = append(opts.ResponseMessages, rs)
			} else if strings.HasPrefix(t, "@Type") {
				opts.Type = strings.TrimSpace(t[len("@Type"):])
			} else if strings.HasPrefix(t, "@Accept") {
				accepts := strings.Split(strings.TrimSpace(strings.TrimSpace(t[len("@Accept"):])), ",")
				for _, a := range accepts {
					switch a {
					case "json":
						opts.Consumes = append(opts.Consumes, ajson)
						opts.Produces = append(opts.Produces, ajson)
					case "xml":
						opts.Consumes = append(opts.Consumes, axml)
						opts.Produces = append(opts.Produces, axml)
					case "plain":
						opts.Consumes = append(opts.Consumes, aplain)
						opts.Produces = append(opts.Produces, aplain)
					case "html":
						opts.Consumes = append(opts.Consumes, ahtml)
						opts.Produces = append(opts.Produces, ahtml)
					}
				}
			}
		}
	}
	innerapi.Operations = append(innerapi.Operations, opts)
	if innerapi.Path != "" {
		if _, ok := controllerList[pkgpath+controllerName]; ok {
			controllerList[pkgpath+controllerName] = append(controllerList[pkgpath+controllerName], innerapi)
		} else {
			controllerList[pkgpath+controllerName] = make([]swagger.Api, 1)
			controllerList[pkgpath+controllerName][0] = innerapi
		}
	}
	return nil
}

// analisys params return []string
// @Param	query		form	 string	true		"The email for login"
// [query form string true "The email for login"]
func getparams(str string) []string {
	var s []rune
	var j int
	var start bool
	var r []string
	for i, c := range []rune(str) {
		if unicode.IsSpace(c) {
			if !start {
				continue
			} else {
				if j == 3 {
					r = append(r, string(s))
					r = append(r, strings.TrimSpace((str[i+1:])))
					break
				}
				start = false
				j++
				r = append(r, string(s))
				s = make([]rune, 0)
				continue
			}
		}
		start = true
		s = append(s, c)
	}
	return r
}

func getModel(str string) (pkgpath, objectname string, m swagger.Model, realTypes []string) {
	strs := strings.Split(str, ".")
	objectname = strs[len(strs)-1]
	pkgpath = strings.Join(strs[:len(strs)-1], "/")
	curpath, _ := os.Getwd()
	pkgRealpath := path.Join(curpath, pkgpath)
	fileSet := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fileSet, pkgRealpath, func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
	}, parser.ParseComments)

	if err != nil {
		ColorLog("[ERRO] the model %s parser.ParseDir error\n", str)
		os.Exit(1)
	}

	for _, pkg := range astPkgs {
		for _, fl := range pkg.Files {
			for k, d := range fl.Scope.Objects {
				if d.Kind == ast.Typ {
					if k != objectname {
						continue
					}
					ts, ok := d.Decl.(*ast.TypeSpec)
					if !ok {
						ColorLog("Unknown type without TypeSec: %v", d)
						os.Exit(1)
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					m.Id = k
					if st.Fields.List != nil {
						m.Properties = make(map[string]swagger.ModelProperty)
						for _, field := range st.Fields.List {
							isSlice, realType := typeAnalyser(field)
							realTypes = append(realTypes, realType)
							mp := swagger.ModelProperty{}
							// add type slice
							if isSlice {
								if isBasicType(realType) {
									mp.Type = "[]" + realType
								} else {
									mp.Type = "array"
									mp.Items = make(map[string]string)
									mp.Items["$ref"] = realType
								}
							} else {
								mp.Type = realType
							}

							// dont add property if anonymous field
							if field.Names != nil {

								// set property name as field name
								var name = field.Names[0].Name

								// if no tag skip tag processing
								if field.Tag == nil {
									m.Properties[name] = mp
									continue
								}

								var tagValues []string
								stag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
								tag := stag.Get("json")

								if tag != "" {
									tagValues = strings.Split(tag, ",")
								}

								// dont add property if json tag first value is "-"
								if len(tagValues) == 0 || tagValues[0] != "-" {

									// set property name to the left most json tag value only if is not omitempty
									if len(tagValues) > 0 && tagValues[0] != "omitempty" {
										name = tagValues[0]
									}

									if thrifttag := stag.Get("thrift"); thrifttag != "" {
										ts := strings.Split(thrifttag, ",")
										if ts[0] != "" {
											name = ts[0]
										}
									}
									if required := stag.Get("required"); required != "" {
										m.Required = append(m.Required, name)
									}
									if desc := stag.Get("description"); desc != "" {
										mp.Description = desc
									}

									m.Properties[name] = mp
								}
								if ignore := stag.Get("ignore"); ignore != "" {
									continue
								}
							}
						}
					}
					return
				}
			}
		}
	}
	if m.Id == "" {
		ColorLog("can't find the object: %v", str)
		os.Exit(1)
	}
	return
}

func typeAnalyser(f *ast.Field) (isSlice bool, realType string) {
	if arr, ok := f.Type.(*ast.ArrayType); ok {
		if isBasicType(fmt.Sprint(arr.Elt)) {
			return false, fmt.Sprintf("[]%v", arr.Elt)
		}
		if mp, ok := arr.Elt.(*ast.MapType); ok {
			return false, fmt.Sprintf("map[%v][%v]", mp.Key, mp.Value)
		}
		if star, ok := arr.Elt.(*ast.StarExpr); ok {
			return true, fmt.Sprint(star.X)
		} else {
			return true, fmt.Sprint(arr.Elt)
		}
	} else {
		switch t := f.Type.(type) {
		case *ast.StarExpr:
			return false, fmt.Sprint(t.X)
		}
		return false, fmt.Sprint(f.Type)
	}
}

func isBasicType(Type string) bool {
	for _, v := range basicTypes {
		if v == Type {
			return true
		}
	}
	return false
}

// refer to builtin.go
var basicTypes = []string{
	"bool",
	"uint", "uint8", "uint16", "uint32", "uint64",
	"int", "int8", "int16", "int32", "int64",
	"float32", "float64",
	"string",
	"complex64", "complex128",
	"byte", "rune", "uintptr",
}

// regexp get json tag
func grepJsonTag(tag string) string {
	r, _ := regexp.Compile(`json:"([^"]*)"`)
	matches := r.FindAllStringSubmatch(tag, -1)
	if len(matches) > 0 {
		return matches[0][1]
	}
	return ""
}

// append models
func appendModels(cmpath, pkgpath, controllerName string, realTypes []string) {
	var p string
	if cmpath != "" {
		p = strings.Join(strings.Split(cmpath, "/"), ".") + "."
	} else {
		p = ""
	}
	for _, realType := range realTypes {
		if realType != "" && !isBasicType(strings.TrimLeft(realType, "[]")) &&
			!strings.HasPrefix(realType, "map") && !strings.HasPrefix(realType, "&") {
			if _, ok := modelsList[pkgpath+controllerName][p+realType]; ok {
				continue
			}
			//fmt.Printf(pkgpath + ":" + controllerName + ":" + cmpath + ":" + realType + "\n")
			_, _, mod, newRealTypes := getModel(p + realType)
			modelsList[pkgpath+controllerName][p+realType] = mod
			appendModels(cmpath, pkgpath, controllerName, newRealTypes)
		}
	}
}
