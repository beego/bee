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
	"go/ast"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

//package model
//
//import (
//"sync"
//
//"example.com/pkgnotexist"
//"github.com/shopspring/decimal"
//)
//
//type Object struct {
//	Field1 decimal.Decimal
//	Field2 pkgnotexist.TestType
//	Field3 sync.Map
//}
func TestCheckAndLoadPackageOnGoMod(t *testing.T) {
	defer os.Setenv("GO111MODULE", os.Getenv("GO111MODULE"))
	os.Setenv("GO111MODULE", "on")

	testCases := []struct {
		pkgName       string
		pkgImportPath string
		imports       []*ast.ImportSpec
		realType      string
		curPkgName    string
		expected      bool
	}{
		{
			pkgName:       "decimal",
			pkgImportPath: "github.com/shopspring/decimal",
			imports: []*ast.ImportSpec{
				{
					Path: &ast.BasicLit{
						Value: "github.com/shopspring/decimal",
					},
				},
			},
			realType:   "decimal.Decimal",
			curPkgName: "model",
			expected:   true,
		},
		{
			pkgName:       "pkgnotexist",
			pkgImportPath: "example.com/pkgnotexist",
			imports: []*ast.ImportSpec{
				{
					Path: &ast.BasicLit{
						Value: "example.com/pkgnotexist",
					},
				},
			},
			realType:   "pkgnotexist.TestType",
			curPkgName: "model",
			expected:   false,
		},
		{
			pkgName:       "sync",
			pkgImportPath: "sync",
			imports: []*ast.ImportSpec{
				{
					Path: &ast.BasicLit{
						Value: "sync",
					},
				},
			},
			realType:   "sync.Map",
			curPkgName: "model",
			expected:   false,
		},
	}

	for _, test := range testCases {
		checkAndLoadPackage(test.imports, test.realType, test.curPkgName)
		result := false
		for _, v := range astPkgs {
			if v.Name == test.pkgName {
				result = true
				break
			}
		}
		if test.expected != result {
			t.Fatalf("load module error, expected: %v, result: %v", test.expected, result)
		}
	}
}

//package model
//
//import (
//"sync"
//
//"example.com/comm"
//"example.com/pkgnotexist"
//)
//
//type Object struct {
//	Field1 comm.Common
//	Field2 pkgnotexist.TestType
//	Field3 sync.Map
//}
func TestCheckAndLoadPackageOnGoPath(t *testing.T) {
	var (
		testCommPkg = `
package comm

type Common struct {
	Code  string
	Error string
}
`
	)

	gopath, err := ioutil.TempDir("", "gobuild-gopath")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(gopath)

	if err := os.MkdirAll(filepath.Join(gopath, "src/example.com/comm"), 0777); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filepath.Join(gopath, "src/example.com/comm/comm.go"), []byte(testCommPkg), 0666); err != nil {
		t.Fatal(err)
	}

	defer os.Setenv("GO111MODULE", os.Getenv("GO111MODULE"))
	os.Setenv("GO111MODULE", "off")
	defer os.Setenv("GOPATH", os.Getenv("GOPATH"))
	os.Setenv("GOPATH", gopath)
	build.Default.GOPATH = gopath

	testCases := []struct {
		pkgName       string
		pkgImportPath string
		imports       []*ast.ImportSpec
		realType      string
		curPkgName    string
		expected      bool
	}{
		{
			pkgName:       "comm",
			pkgImportPath: "example.com/comm",
			imports: []*ast.ImportSpec{
				{
					Path: &ast.BasicLit{
						Value: "example.com/comm",
					},
				},
			},
			realType:   "comm.Common",
			curPkgName: "model",
			expected:   true,
		},
		{
			pkgName:       "pkgnotexist",
			pkgImportPath: "example.com/pkgnotexist",
			imports: []*ast.ImportSpec{
				{
					Path: &ast.BasicLit{
						Value: "example.com/pkgnotexist",
					},
				},
			},
			realType:   "pkgnotexist.TestType",
			curPkgName: "model",
			expected:   false,
		},
		{
			pkgName:       "sync",
			pkgImportPath: "sync",
			imports: []*ast.ImportSpec{
				{
					Path: &ast.BasicLit{
						Value: "sync",
					},
				},
			},
			realType:   "sync.Map",
			curPkgName: "model",
			expected:   false,
		},
	}

	for _, test := range testCases {
		checkAndLoadPackage(test.imports, test.realType, test.curPkgName)
		result := false
		for _, v := range astPkgs {
			if v.Name == test.pkgName {
				result = true
				break
			}
		}
		if test.expected != result {
			t.Fatalf("load module error, expected: %v, result: %v", test.expected, result)
		}
	}
}
