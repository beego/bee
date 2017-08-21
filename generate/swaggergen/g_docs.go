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

package swaggergen

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
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"github.com/ghodss/yaml"

	"github.com/astaxie/beego/swagger"
	"github.com/astaxie/beego/utils"
	beeLogger "github.com/beego/bee/logger"
	bu "github.com/beego/bee/utils"
	"github.com/go-openapi/spec"
	"github.com/wy-z/tspec/tspec"
)

const (
	ajson  = "application/json"
	axml   = "application/xml"
	aplain = "text/plain"
	ahtml  = "text/html"
	aform  = "multipart/form-data"
)

// Swagger redefines definitions
type Swagger struct {
	swagger.Swagger
	Definitions spec.Definitions `json:"definitions,omitempty" yaml:"definitions,omitempty"`
}

var pkgCache map[string]struct{} //pkg:controller:function:comments comments: key:value
var controllerComments map[string]string
var importlist map[string]string
var controllerList map[string]map[string]*swagger.Item //controllername Paths items
var rootapi Swagger

var tparser *tspec.Parser
var controllerPkg *ast.Package

func parseModel(pkg *ast.Package, typeStr string) (typeTitle string, err error) {
	if pkg == nil {
		panic("pkg can not be nil")
	}
	schema, err := tparser.Parse(pkg, typeStr)
	if err != nil {
		return
	}
	typeTitle = schema.Title
	return
}

// refer to builtin.go
var basicTypes = map[string]string{
	"bool":       "boolean:",
	"uint":       "integer:int32",
	"uint8":      "integer:int32",
	"uint16":     "integer:int32",
	"uint32":     "integer:int32",
	"uint64":     "integer:int64",
	"int":        "integer:int64",
	"int8":       "integer:int32",
	"int16":      "integer:int32",
	"int32":      "integer:int32",
	"int64":      "integer:int64",
	"uintptr":    "integer:int64",
	"float32":    "number:float",
	"float64":    "number:double",
	"string":     "string:",
	"complex64":  "number:float",
	"complex128": "number:double",
	"byte":       "string:byte",
	"rune":       "string:byte",
	// builtin golang objects
	"time.Time": "string:string",
}

func init() {
	pkgCache = make(map[string]struct{})
	controllerComments = make(map[string]string)
	importlist = make(map[string]string)
	controllerList = make(map[string]map[string]*swagger.Item)
	tparser = tspec.NewParser()
}

// GenerateDocs ...
func GenerateDocs(curpath string) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, filepath.Join(curpath, "routers", "router.go"), nil, parser.ParseComments)
	if err != nil {
		beeLogger.Log.Fatalf("Error while parsing router.go: %s", err)
	}

	rootapi.Infos = swagger.Information{}
	rootapi.SwaggerVersion = "2.0"

	// Analyse API comments
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
				} else if strings.HasPrefix(s, "@Name") {
					rootapi.Infos.Contact.Name = strings.TrimSpace(s[len("@Name"):])
				} else if strings.HasPrefix(s, "@URL") {
					rootapi.Infos.Contact.URL = strings.TrimSpace(s[len("@URL"):])
				} else if strings.HasPrefix(s, "@LicenseUrl") {
					if rootapi.Infos.License == nil {
						rootapi.Infos.License = &swagger.License{URL: strings.TrimSpace(s[len("@LicenseUrl"):])}
					} else {
						rootapi.Infos.License.URL = strings.TrimSpace(s[len("@LicenseUrl"):])
					}
				} else if strings.HasPrefix(s, "@License") {
					if rootapi.Infos.License == nil {
						rootapi.Infos.License = &swagger.License{Name: strings.TrimSpace(s[len("@License"):])}
					} else {
						rootapi.Infos.License.Name = strings.TrimSpace(s[len("@License"):])
					}
				} else if strings.HasPrefix(s, "@Schemes") {
					rootapi.Schemes = strings.Split(strings.TrimSpace(s[len("@Schemes"):]), ",")
				} else if strings.HasPrefix(s, "@Host") {
					rootapi.Host = strings.TrimSpace(s[len("@Host"):])
				} else if strings.HasPrefix(s, "@SecurityDefinition") {
					if len(rootapi.SecurityDefinitions) == 0 {
						rootapi.SecurityDefinitions = make(map[string]swagger.Security)
					}
					var out swagger.Security
					p := getparams(strings.TrimSpace(s[len("@SecurityDefinition"):]))
					if len(p) < 2 {
						beeLogger.Log.Fatalf("Not enough params for security: %d\n", len(p))
					}
					out.Type = p[1]
					switch out.Type {
					case "oauth2":
						if len(p) < 6 {
							beeLogger.Log.Fatalf("Not enough params for oauth2: %d\n", len(p))
						}
						if !(p[3] == "implicit" || p[3] == "password" || p[3] == "application" || p[3] == "accessCode") {
							beeLogger.Log.Fatalf("Unknown flow type: %s. Possible values are `implicit`, `password`, `application` or `accessCode`.\n", p[1])
						}
						out.AuthorizationURL = p[2]
						out.Flow = p[3]
						if len(p)%2 != 0 {
							out.Description = strings.Trim(p[len(p)-1], `" `)
						}
						out.Scopes = make(map[string]string)
						for i := 4; i < len(p)-1; i += 2 {
							out.Scopes[p[i]] = strings.Trim(p[i+1], `" `)
						}
					case "apiKey":
						if len(p) < 4 {
							beeLogger.Log.Fatalf("Not enough params for apiKey: %d\n", len(p))
						}
						if !(p[3] == "header" || p[3] == "query") {
							beeLogger.Log.Fatalf("Unknown in type: %s. Possible values are `query` or `header`.\n", p[4])
						}
						out.Name = p[2]
						out.In = p[3]
						if len(p) > 4 {
							out.Description = strings.Trim(p[4], `" `)
						}
					case "basic":
						if len(p) > 2 {
							out.Description = strings.Trim(p[2], `" `)
						}
					default:
						beeLogger.Log.Fatalf("Unknown security type: %s. Possible values are `oauth2`, `apiKey` or `basic`.\n", p[1])
					}
					rootapi.SecurityDefinitions[p[0]] = out
				} else if strings.HasPrefix(s, "@Security") {
					if len(rootapi.Security) == 0 {
						rootapi.Security = make([]map[string][]string, 0)
					}
					rootapi.Security = append(rootapi.Security, getSecurity(s))
				}
			}
		}
	}
	// Analyse controller package
	for _, im := range f.Imports {
		localName := ""
		if im.Name != nil {
			localName = im.Name.Name
		}
		analyseControllerPkg(path.Join(curpath, "vendor"), localName, im.Path.Value)
	}
	for _, d := range f.Decls {
		switch specDecl := d.(type) {
		case *ast.FuncDecl:
			for _, l := range specDecl.Body.List {
				switch stmt := l.(type) {
				case *ast.AssignStmt:
					for _, l := range stmt.Rhs {
						if v, ok := l.(*ast.CallExpr); ok {
							// Analyse NewNamespace, it will return version and the subfunction
							if selName := v.Fun.(*ast.SelectorExpr).Sel.String(); selName != "NewNamespace" {
								continue
							}
							version, params := analyseNewNamespace(v)
							if rootapi.BasePath == "" && version != "" {
								rootapi.BasePath = version
							}
							for _, p := range params {
								switch pp := p.(type) {
								case *ast.CallExpr:
									var controllerName string
									if selname := pp.Fun.(*ast.SelectorExpr).Sel.String(); selname == "NSNamespace" {
										s, params := analyseNewNamespace(pp)
										for _, sp := range params {
											switch pp := sp.(type) {
											case *ast.CallExpr:
												if pp.Fun.(*ast.SelectorExpr).Sel.String() == "NSInclude" {
													controllerName = analyseNSInclude(s, pp)
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
										controllerName = analyseNSInclude("", pp)
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
	rootapi.Definitions = tparser.Definitions()

	os.Mkdir(path.Join(curpath, "swagger"), 0755)
	fd, err := os.Create(path.Join(curpath, "swagger", "swagger.json"))
	if err != nil {
		panic(err)
	}
	fdyml, err := os.Create(path.Join(curpath, "swagger", "swagger.yml"))
	if err != nil {
		panic(err)
	}
	defer fdyml.Close()
	defer fd.Close()
	dt, err := json.MarshalIndent(rootapi, "", "    ")
	if err != nil {
		msg := fmt.Sprintf("failed to marshal api doc: %s", err)
		panic(msg)
	}
	dtyml, erryml := yaml.JSONToYAML(dt)
	if erryml != nil {
		msg := fmt.Sprintf("failed to convert json bytes to yaml bytes: %s", erryml)
		panic(msg)
	}
	_, err = fd.Write(dt)
	_, erryml = fdyml.Write(dtyml)
	if err != nil || erryml != nil {
		panic(err)
	}
}

// analyseNewNamespace returns version and the others params
func analyseNewNamespace(ce *ast.CallExpr) (first string, others []ast.Expr) {
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

func analyseNSInclude(baseurl string, ce *ast.CallExpr) string {
	cname := ""
	for _, p := range ce.Args {
		x := p.(*ast.UnaryExpr).X.(*ast.CompositeLit).Type.(*ast.SelectorExpr)
		if v, ok := importlist[fmt.Sprint(x.X)]; ok {
			cname = v + x.Sel.Name
		}
		if apis, ok := controllerList[cname]; ok {
			for rt, item := range apis {
				tag := cname
				if baseurl != "" {
					rt = baseurl + rt
					tag = strings.Trim(baseurl, "/")
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

func analyseControllerPkg(vendorPath, localName, pkgpath string) {
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
	gopaths := bu.GetGOPATHs()
	if len(gopaths) == 0 {
		beeLogger.Log.Fatal("GOPATH environment variable is not set or empty")
	}
	pkgRealpath := ""

	wg, _ := filepath.EvalSymlinks(filepath.Join(vendorPath, pkgpath))
	if utils.FileExists(wg) {
		pkgRealpath = wg
	} else {
		wgopath := gopaths
		for _, wg := range wgopath {
			wg, _ = filepath.EvalSymlinks(filepath.Join(wg, "src", pkgpath))
			if utils.FileExists(wg) {
				pkgRealpath = wg
				break
			}
		}
	}
	if pkgRealpath != "" {
		if _, ok := pkgCache[pkgpath]; ok {
			return
		}
		pkgCache[pkgpath] = struct{}{}
	} else {
		beeLogger.Log.Fatalf("Package '%s' does not exist in the GOPATH or vendor path", pkgpath)
	}

	fileSet := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fileSet, pkgRealpath, func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
	}, parser.ParseComments)
	if err != nil {
		beeLogger.Log.Fatalf("Error while parsing dir at '%s': %s", pkgpath, err)
	}
	for _, pkg := range astPkgs {
		if pkg.Name == "controllers" {
			controllerPkg = pkg
		}

		for _, fl := range pkg.Files {
			for _, d := range fl.Decls {
				switch specDecl := d.(type) {
				case *ast.FuncDecl:
					if specDecl.Recv != nil && len(specDecl.Recv.List) > 0 {
						if t, ok := specDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
							// Parse controller method
							parserComments(specDecl, fmt.Sprint(t.X), pkgpath)
						}
					}
				case *ast.GenDecl:
					if specDecl.Tok == token.TYPE {
						for _, s := range specDecl.Specs {
							switch tp := s.(*ast.TypeSpec).Type.(type) {
							case *ast.StructType:
								_ = tp.Struct
								// Parse controller definition comments
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
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		goroot = runtime.GOROOT()
	}
	if goroot == "" {
		beeLogger.Log.Fatalf("GOROOT environment variable is not set or empty")
	}

	wg, _ := filepath.EvalSymlinks(filepath.Join(goroot, "src", "pkg", pkgpath))
	if utils.FileExists(wg) {
		return true
	}

	//TODO(zh):support go1.4
	wg, _ = filepath.EvalSymlinks(filepath.Join(goroot, "src", pkgpath))
	return utils.FileExists(wg)
}

func peekNextSplitString(ss string) (s string, spacePos int) {
	spacePos = strings.IndexFunc(ss, unicode.IsSpace)
	if spacePos < 0 {
		s = ss
		spacePos = len(ss)
	} else {
		s = strings.TrimSpace(ss[:spacePos])
	}
	return
}

// parse the func comments
func parserComments(f *ast.FuncDecl, controllerName, pkgpath string) error {
	var routerPath string
	var HTTPMethod string
	opts := swagger.Operation{
		Responses: make(map[string]swagger.Response),
	}
	funcName := f.Name.String()
	comments := f.Doc
	funcParamMap := buildParamMap(f.Type.Params)
	//TODO: resultMap := buildParamMap(f.Type.Results)
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
				opts.Description = strings.TrimSpace(t[len("@Description"):])
			} else if strings.HasPrefix(t, "@Summary") {
				opts.Summary = strings.TrimSpace(t[len("@Summary"):])
			} else if strings.HasPrefix(t, "@Success") {
				ss := strings.TrimSpace(t[len("@Success"):])
				rs := swagger.Response{}
				respCode, pos := peekNextSplitString(ss)
				ss = strings.TrimSpace(ss[pos:])
				respType, pos := peekNextSplitString(ss)
				if respType == "{object}" || respType == "{array}" {
					isArray := respType == "{array}"
					ss = strings.TrimSpace(ss[pos:])
					schemaName, pos := peekNextSplitString(ss)
					if schemaName == "" {
						beeLogger.Log.Fatalf("[%s.%s] Schema must follow {object} or {array}", controllerName, funcName)
					}
					if strings.HasPrefix(schemaName, "[]") {
						schemaName = schemaName[2:]
						isArray = true
					}
					schema := swagger.Schema{}
					if sType, ok := basicTypes[schemaName]; ok {
						typeFormat := strings.Split(sType, ":")
						schema.Type = typeFormat[0]
						schema.Format = typeFormat[1]
					} else {
						typeTitle, err := parseModel(controllerPkg, schemaName)
						if err != nil {
							beeLogger.Log.Fatalf("failed to parse model %s: %s", schemaName, err)
						}
						schema.Ref = "#/definitions/" + typeTitle
					}
					if isArray {
						rs.Schema = &swagger.Schema{
							Type:  "array",
							Items: &schema,
						}
					} else {
						rs.Schema = &schema
					}
					rs.Description = strings.TrimSpace(ss[pos:])
				} else {
					rs.Description = strings.TrimSpace(ss)
				}
				opts.Responses[respCode] = rs
			} else if strings.HasPrefix(t, "@Param") {
				para := swagger.Parameter{}
				p := getparams(strings.TrimSpace(t[len("@Param "):]))
				if len(p) < 4 {
					beeLogger.Log.Fatal(controllerName + "_" + funcName + "'s comments @Param should have at least 4 params")
				}
				paramNames := strings.SplitN(p[0], "=>", 2)
				para.Name = paramNames[0]
				funcParamName := para.Name
				if len(paramNames) > 1 {
					funcParamName = paramNames[1]
				}
				paramType, ok := funcParamMap[funcParamName]
				if ok {
					delete(funcParamMap, funcParamName)
				}

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
					beeLogger.Log.Warnf("[%s.%s] Unknown param location: %s. Possible values are `query`, `header`, `path`, `formData` or `body`.\n", controllerName, funcName, p[1])
				}
				para.In = p[1]
				pp := strings.Split(p[2], ".")
				typ := pp[len(pp)-1]
				if len(pp) >= 2 {
					typeTitle, err := parseModel(controllerPkg, p[2])
					if err != nil {
						beeLogger.Log.Fatalf("failed to parse model %s: %s", p[2], err)
					}
					para.Schema = &swagger.Schema{
						Ref: "#/definitions/" + typeTitle,
					}
				} else {
					if typ == "auto" {
						typ = paramType
					}
					setParamType(&para, typ, pkgpath, controllerName)
				}
				switch len(p) {
				case 5:
					para.Required, _ = strconv.ParseBool(p[3])
					para.Description = strings.Trim(p[4], `" `)
				case 6:
					para.Default = str2RealType(p[3], para.Type)
					para.Required, _ = strconv.ParseBool(p[4])
					para.Description = strings.Trim(p[5], `" `)
				default:
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
					case "form":
						opts.Consumes = append(opts.Consumes, aform)
					}
				}
			} else if strings.HasPrefix(t, "@Security") {
				if len(opts.Security) == 0 {
					opts.Security = make([]map[string][]string, 0)
				}
				opts.Security = append(opts.Security, getSecurity(t))
			}
		}
	}

	if routerPath != "" {
		//Go over function parameters which were not mapped and create swagger params for them
		for name, typ := range funcParamMap {
			para := swagger.Parameter{}
			para.Name = name
			setParamType(&para, typ, pkgpath, controllerName)
			if paramInPath(name, routerPath) {
				para.In = "path"
			} else {
				para.In = "query"
			}
			opts.Parameters = append(opts.Parameters, para)
		}

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
		for _, hm := range strings.Split(HTTPMethod, ",") {
			switch hm {
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
		}
		controllerList[pkgpath+controllerName][routerPath] = item
	}
	return nil
}

func setParamType(para *swagger.Parameter, typ string, pkgpath, controllerName string) {
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
		typeTitle, err := parseModel(controllerPkg, typ)
		if err != nil {
			beeLogger.Log.Fatalf("failed to parse model %s: %s", typ, err)
		}
		para.Schema = &swagger.Schema{
			Ref: "#/definitions/" + typeTitle,
		}
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

func paramInPath(name, route string) bool {
	return strings.HasSuffix(route, ":"+name) ||
		strings.Contains(route, ":"+name+"/")
}

func getFunctionParamType(t ast.Expr) string {
	switch paramType := t.(type) {
	case *ast.Ident:
		return paramType.Name
	// case *ast.Ellipsis:
	// 	result := getFunctionParamType(paramType.Elt)
	// 	result.array = true
	// 	return result
	case *ast.ArrayType:
		return "[]" + getFunctionParamType(paramType.Elt)
	case *ast.StarExpr:
		return getFunctionParamType(paramType.X)
	case *ast.SelectorExpr:
		return getFunctionParamType(paramType.X) + "." + paramType.Sel.Name
	default:
		return ""

	}
}

func buildParamMap(list *ast.FieldList) map[string]string {
	i := 0
	result := map[string]string{}
	if list != nil {
		funcParams := list.List
		for _, fparam := range funcParams {
			param := getFunctionParamType(fparam.Type)
			var paramName string
			if len(fparam.Names) > 0 {
				paramName = fparam.Names[0].Name
			} else {
				paramName = fmt.Sprint(i)
				i++
			}
			result[paramName] = param
		}
	}
	return result
}

// analisys params return []string
// @Param	query		form	 string	true		"The email for login"
// [query form string true "The email for login"]
func getparams(str string) []string {
	var s []rune
	var j int
	var start bool
	var r []string
	var quoted int8
	for _, c := range str {
		if unicode.IsSpace(c) && quoted == 0 {
			if !start {
				continue
			} else {
				start = false
				j++
				r = append(r, string(s))
				s = make([]rune, 0)
				continue
			}
		}

		start = true
		if c == '"' {
			quoted ^= 1
			continue
		}
		s = append(s, c)
	}
	if len(s) > 0 {
		r = append(r, string(s))
	}
	return r
}

func getSecurity(t string) (security map[string][]string) {
	security = make(map[string][]string)
	p := getparams(strings.TrimSpace(t[len("@Security"):]))
	if len(p) == 0 {
		beeLogger.Log.Fatalf("No params for security specified\n")
	}
	security[p[0]] = make([]string, 0)
	for i := 1; i < len(p); i++ {
		security[p[0]] = append(security[p[0]], p[i])
	}
	return
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

func str2RealType(s string, typ string) interface{} {
	var err error
	var ret interface{}

	switch typ {
	case "int", "int64", "int32", "int16", "int8":
		ret, err = strconv.Atoi(s)
	case "bool":
		ret, err = strconv.ParseBool(s)
	case "float64":
		ret, err = strconv.ParseFloat(s, 64)
	case "float32":
		ret, err = strconv.ParseFloat(s, 32)
	default:
		return s
	}

	if err != nil {
		beeLogger.Log.Warnf("Invalid default value type '%s': %s", typ, s)
		return s
	}

	return ret
}
