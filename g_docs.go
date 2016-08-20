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

const (
	ajson  = "application/json"
	axml   = "application/xml"
	aplain = "text/plain"
	ahtml  = "text/html"
)

var pkgCache map[string]struct{} //pkg:controller:function:comments comments: key:value
var controllerComments map[string]string
var importlist map[string]string
var controllerList map[string]map[string]*swagger.Item //controllername Paths items
var modelsList map[string]map[string]swagger.Schema
var rootapi swagger.Swagger

func init() {
	pkgCache = make(map[string]struct{})
	controllerComments = make(map[string]string)
	importlist = make(map[string]string)
	controllerList = make(map[string]map[string]*swagger.Item)
	modelsList = make(map[string]map[string]swagger.Schema)
}

func generateDocs(curpath string) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, path.Join(curpath, "routers", "router.go"), nil, parser.ParseComments)

	if err != nil {
		ColorLog("[ERRO] parse router.go error\n")
		os.Exit(2)
	}

	rootapi.Infos = swagger.Information{}
	rootapi.SwaggerVersion = "2.0"
	//analysis API comments
	if f.Comments != nil {
		for _, c := range f.Comments {
			for _, s := range strings.Split(c.Text(), "\n") {
				if strings.HasPrefix(s, "@APIVersion") {
					rootapi.Infos.Version = strings.TrimSpace(s[len("@APIVersion"):])
				} else if strings.HasPrefix(s, "@Title") {
					rootapi.Infos.Title = strings.TrimSpace(s[len("@Title"):])
				} else if strings.HasPrefix(s, "@Description") {
					rootapi.Infos.Description = strings.TrimSpace(s[len("@Description"):])
				} else if strings.HasPrefix(s, "@TermsOfServiceUrl") {
					rootapi.Infos.TermsOfService = strings.TrimSpace(s[len("@TermsOfServiceUrl"):])
				} else if strings.HasPrefix(s, "@Contact") {
					rootapi.Infos.Contact.EMail = strings.TrimSpace(s[len("@Contact"):])
				} else if strings.HasPrefix(s, "@License") {
					rootapi.Infos.License.Name = strings.TrimSpace(s[len("@License"):])
				} else if strings.HasPrefix(s, "@LicenseUrl") {
					rootapi.Infos.License.URL = strings.TrimSpace(s[len("@LicenseUrl"):])
				}
			}
		}
	}
	// analisys controller package
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
				switch stmt := l.(type) {
				case *ast.AssignStmt:
					for _, l := range stmt.Rhs {
						if v, ok := l.(*ast.CallExpr); ok {
							// analisys NewNamespace, it will return version and the subfunction
							if selName := v.Fun.(*ast.SelectorExpr).Sel.String(); selName != "NewNamespace" {
								continue
							}
							version, params := analisysNewNamespace(v)
							if rootapi.BasePath == "" && version != "" {
								rootapi.BasePath = version
							}
							for _, p := range params {
								switch pp := p.(type) {
								case *ast.CallExpr:
									controllerName := ""
									if selname := pp.Fun.(*ast.SelectorExpr).Sel.String(); selname == "NSNamespace" {
										s, params := analisysNewNamespace(pp)
										for _, sp := range params {
											switch pp := sp.(type) {
											case *ast.CallExpr:
												if pp.Fun.(*ast.SelectorExpr).Sel.String() == "NSInclude" {
													controllerName = analisysNSInclude(s, pp)
													if v, ok := controllerComments[controllerName]; ok {
														rootapi.Tags = append(rootapi.Tags, swagger.Tag{
															Name:        strings.Trim(s, "/"),
															Description: v,
														})
													}
												}
											}
										}
									} else if selname == "NSInclude" {
										controllerName = analisysNSInclude("", pp)
										if v, ok := controllerComments[controllerName]; ok {
											rootapi.Tags = append(rootapi.Tags, swagger.Tag{
												Name:        controllerName, // if the NSInclude has no prefix, we use the controllername as the tag
												Description: v,
											})
										}
									}
								}
							}
						}

					}
				}
			}
		}
	}
	os.Mkdir(path.Join(curpath, "swagger"), 0755)
	fd, err := os.Create(path.Join(curpath, "swagger", "swagger.json"))
	if err != nil {
		panic(err)
	}
	defer fd.Close()
	dt, err := json.MarshalIndent(rootapi, "", "    ")
	if err != nil {
		panic(err)
	}
	_, err = fd.Write(dt)
	if err != nil {
		panic(err)
	}
}

// return version and the others params
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
	for _, p := range ce.Args {
		x := p.(*ast.UnaryExpr).X.(*ast.CompositeLit).Type.(*ast.SelectorExpr)
		if v, ok := importlist[fmt.Sprint(x.X)]; ok {
			cname = v + x.Sel.Name
		}
		if apis, ok := controllerList[cname]; ok {
			for rt, item := range apis {
				tag := ""
				if baseurl != "" {
					rt = baseurl + rt
					tag = strings.Trim(baseurl, "/")
				} else {
					tag = cname
				}
				if item.Get != nil {
					item.Get.Tags = []string{tag}
				}
				if item.Post != nil {
					item.Post.Tags = []string{tag}
				}
				if item.Put != nil {
					item.Put.Tags = []string{tag}
				}
				if item.Patch != nil {
					item.Patch.Tags = []string{tag}
				}
				if item.Head != nil {
					item.Head.Tags = []string{tag}
				}
				if item.Delete != nil {
					item.Delete.Tags = []string{tag}
				}
				if item.Options != nil {
					item.Options.Tags = []string{tag}
				}
				if len(rootapi.Paths) == 0 {
					rootapi.Paths = make(map[string]*swagger.Item)
				}
				rt = urlReplace(rt)
				rootapi.Paths[rt] = item
			}
		}
	}
	return cname
}

func analisyscontrollerPkg(localName, pkgpath string) {
	pkgpath = strings.Trim(pkgpath, "\"")
	if isSystemPackage(pkgpath) {
		return
	}
	if pkgpath == "github.com/astaxie/beego" {
		return
	}
	if localName != "" {
		importlist[localName] = pkgpath
	} else {
		pps := strings.Split(pkgpath, "/")
		importlist[pps[len(pps)-1]] = pkgpath
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
		pkgCache[pkgpath] = struct{}{}
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
							// parse controller method
							parserComments(specDecl.Doc, specDecl.Name.String(), fmt.Sprint(t.X), pkgpath)
						}
					}
				case *ast.GenDecl:
					if specDecl.Tok == token.TYPE {
						for _, s := range specDecl.Specs {
							switch tp := s.(*ast.TypeSpec).Type.(type) {
							case *ast.StructType:
								_ = tp.Struct
								//parse controller definition comments
								if strings.TrimSpace(specDecl.Doc.Text()) != "" {
									controllerComments[pkgpath+s.(*ast.TypeSpec).Name.String()] = specDecl.Doc.Text()
								}
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
	var routerPath string
	var HTTPMethod string
	opts := swagger.Operation{
		Responses: make(map[string]swagger.Response),
	}
	if comments != nil && comments.List != nil {
		for _, c := range comments.List {
			t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
			if strings.HasPrefix(t, "@router") {
				elements := strings.TrimSpace(t[len("@router"):])
				e1 := strings.SplitN(elements, " ", 2)
				if len(e1) < 1 {
					return errors.New("you should has router infomation")
				}
				routerPath = e1[0]
				if len(e1) == 2 && e1[1] != "" {
					e1 = strings.SplitN(e1[1], " ", 2)
					HTTPMethod = strings.ToUpper(strings.Trim(e1[0], "[]"))
				} else {
					HTTPMethod = "GET"
				}
			} else if strings.HasPrefix(t, "@Title") {
				opts.OperationID = controllerName + "." + strings.TrimSpace(t[len("@Title"):])
			} else if strings.HasPrefix(t, "@Description") {
				opts.Summary = strings.TrimSpace(t[len("@Description"):])
			} else if strings.HasPrefix(t, "@Success") {
				ss := strings.TrimSpace(t[len("@Success"):])
				rs := swagger.Response{}
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
							j++
							start = false
							if j == 1 {
								continue
							} else {
								st[j] = strings.TrimSpace(ss[i+1:])
								break

							}
						}
					} else {
						start = true
						tmp = append(tmp, c)
					}
				}
				if len(tmp) > 0 && st[2] == "" {
					st[2] = strings.TrimSpace(string(tmp))
				}
				rs.Description = st[2]
				if st[1] == "{object}" {
					if st[2] == "" {
						panic(controllerName + " " + funcName + " has no object")
					}
					cmpath, m, mod, realTypes := getModel(st[2])
					//ll := strings.Split(st[2], ".")
					//opts.Type = ll[len(ll)-1]
					rs.Schema = &swagger.Schema{
						Ref: "#/definitions/" + m,
					}
					if _, ok := modelsList[pkgpath+controllerName]; !ok {
						modelsList[pkgpath+controllerName] = make(map[string]swagger.Schema, 0)
					}
					modelsList[pkgpath+controllerName][st[2]] = mod
					appendModels(cmpath, pkgpath, controllerName, realTypes)
				}
				opts.Responses[st[0]] = rs
			} else if strings.HasPrefix(t, "@Param") {
				para := swagger.Parameter{}
				p := getparams(strings.TrimSpace(t[len("@Param "):]))
				if len(p) < 4 {
					panic(controllerName + "_" + funcName + "'s comments @Param at least should has 4 params")
				}
				para.Name = p[0]
				switch p[1] {
				case "query":
					fallthrough
				case "header":
					fallthrough
				case "path":
					fallthrough
				case "formData":
					fallthrough
				case "body":
					break
				default:
					fmt.Fprintf(os.Stderr, "[%s.%s] Unknow param location: %s, Possible values are `query`, `header`, `path`, `formData` or `body`.\n", controllerName, funcName, p[1])
				}
				para.In = p[1]
				pp := strings.Split(p[2], ".")
				typ := pp[len(pp)-1]
				if len(pp) >= 2 {
					cmpath, m, mod, realTypes := getModel(p[2])
					para.Schema = &swagger.Schema{
						Ref: "#/definitions/" + m,
					}
					if _, ok := modelsList[pkgpath+controllerName]; !ok {
						modelsList[pkgpath+controllerName] = make(map[string]swagger.Schema, 0)
					}
					modelsList[pkgpath+controllerName][typ] = mod
					appendModels(cmpath, pkgpath, controllerName, realTypes)
				} else {
					isArray := false
					paraType := ""
					paraFormat := ""
					if strings.HasPrefix(typ, "[]") {
						typ = typ[2:]
						isArray = true
					}
					if typ == "string" || typ == "number" || typ == "integer" || typ == "boolean" ||
						typ == "array" || typ == "file" {
						paraType = typ
					} else if sType, ok := basicTypes[typ]; ok {
						typeFormat := strings.Split(sType, ":")
						paraType = typeFormat[0]
						paraFormat = typeFormat[1]
					} else {
						fmt.Fprintf(os.Stderr, "[%s.%s] Unknow param type: %s\n", controllerName, funcName, typ)
					}
					if isArray {
						para.Type = "array"
						para.Items = &swagger.ParameterItems{
							Type:   paraType,
							Format: paraFormat,
						}
					} else {
						para.Type = paraType
						para.Format = paraFormat
					}
				}
				if len(p) > 4 {
					para.Required, _ = strconv.ParseBool(p[3])
					para.Description = strings.Trim(p[4], `" `)
				} else {
					para.Description = strings.Trim(p[3], `" `)
				}
				opts.Parameters = append(opts.Parameters, para)
			} else if strings.HasPrefix(t, "@Failure") {
				rs := swagger.Response{}
				st := strings.TrimSpace(t[len("@Failure"):])
				var cd []rune
				var start bool
				for i, s := range st {
					if unicode.IsSpace(s) {
						if start {
							rs.Description = strings.TrimSpace(st[i+1:])
							break
						} else {
							continue
						}
					}
					start = true
					cd = append(cd, s)
				}
				opts.Responses[string(cd)] = rs
			} else if strings.HasPrefix(t, "@Deprecated") {
				opts.Deprecated, _ = strconv.ParseBool(strings.TrimSpace(t[len("@Deprecated"):]))
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
	if routerPath != "" {
		var item *swagger.Item
		if itemList, ok := controllerList[pkgpath+controllerName]; ok {
			if it, ok := itemList[routerPath]; !ok {
				item = &swagger.Item{}
			} else {
				item = it
			}
		} else {
			controllerList[pkgpath+controllerName] = make(map[string]*swagger.Item)
			item = &swagger.Item{}
		}
		switch HTTPMethod {
		case "GET":
			item.Get = &opts
		case "POST":
			item.Post = &opts
		case "PUT":
			item.Put = &opts
		case "PATCH":
			item.Patch = &opts
		case "DELETE":
			item.Delete = &opts
		case "HEAD":
			item.Head = &opts
		case "OPTIONS":
			item.Options = &opts
		}
		controllerList[pkgpath+controllerName][routerPath] = item
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

func getModel(str string) (pkgpath, objectname string, m swagger.Schema, realTypes []string) {
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
	m.Type = "object"
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
					m.Title = k
					if st.Fields.List != nil {
						m.Properties = make(map[string]swagger.Propertie)
						for _, field := range st.Fields.List {
							isSlice, realType, sType := typeAnalyser(field)
							realTypes = append(realTypes, realType)
							mp := swagger.Propertie{}
							// add type slice
							if isSlice {
								mp.Type = "array"
								if isBasicType(realType) {
									typeFormat := strings.Split(sType, ":")
									mp.Items = &swagger.Propertie{
										Type:   typeFormat[0],
										Format: typeFormat[1],
									}
								} else {
									mp.Items = &swagger.Propertie{
										Ref: "#/definitions/" + realType,
									}
								}
							} else {
								if isBasicType(realType) {
									typeFormat := strings.Split(sType, ":")
									mp.Type = typeFormat[0]
									mp.Format = typeFormat[1]
								} else if sType == "object" {
									mp.Ref = "#/definitions/" + realType
								}
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
				}
			}
		}
	}
	if m.Title == "" {
		ColorLog("can't find the object: %s", str)
		os.Exit(1)
	}
	if len(rootapi.Definitions) == 0 {
		rootapi.Definitions = make(map[string]swagger.Schema)
	}
	rootapi.Definitions[objectname] = m
	return
}

func typeAnalyser(f *ast.Field) (isSlice bool, realType, swaggerType string) {
	if arr, ok := f.Type.(*ast.ArrayType); ok {
		if isBasicType(fmt.Sprint(arr.Elt)) {
			return false, fmt.Sprintf("[]%v", arr.Elt), basicTypes[fmt.Sprint(arr.Elt)]
		}
		if mp, ok := arr.Elt.(*ast.MapType); ok {
			return false, fmt.Sprintf("map[%v][%v]", mp.Key, mp.Value), "object"
		}
		if star, ok := arr.Elt.(*ast.StarExpr); ok {
			return true, fmt.Sprint(star.X), "object"
		}
		return true, fmt.Sprint(arr.Elt), "object"
	}
	switch t := f.Type.(type) {
	case *ast.StarExpr:
		return false, fmt.Sprint(t.X), "object"
	case *ast.MapType:
		return false, fmt.Sprint(t.Value), "object"
	}
	if k, ok := basicTypes[fmt.Sprint(f.Type)]; ok {
		return false, fmt.Sprint(f.Type), k
	}
	return false, fmt.Sprint(f.Type), "object"
}

func isBasicType(Type string) bool {
	if _, ok := basicTypes[Type]; ok {
		return true
	}
	return false
}

// refer to builtin.go
var basicTypes = map[string]string{
	"bool": "boolean:",
	"uint": "integer:int32", "uint8": "integer:int32", "uint16": "integer:int32", "uint32": "integer:int32", "uint64": "integer:int64",
	"int": "integer:int64", "int8": "integer:int32", "int16:int32": "integer:int32", "int32": "integer:int32", "int64": "integer:int64",
	"uintptr": "integer:int64",
	"float32": "number:float", "float64": "number:double",
	"string":    "string:",
	"complex64": "number:float", "complex128": "number:double",
	"byte": "string:byte", "rune": "string:byte",
}

// regexp get json tag
func grepJSONTag(tag string) string {
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
