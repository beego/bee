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
	"fmt"
	"os"
	"path"
)

// recipe
// admin/recipe
func generateView(viewpath, currpath string) {
	w := NewColorWriter(os.Stdout)

	logger.Info("Generating view...")

	absViewPath := path.Join(currpath, "views", viewpath)
	err := os.MkdirAll(absViewPath, os.ModePerm)
	if err != nil {
		logger.Fatalf("Could not create '%s' view: %s", viewpath, err)
	}

	cfile := path.Join(absViewPath, "index.tpl")
	if f, err := os.OpenFile(cfile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer CloseFile(f)
		f.WriteString(cfile)
		fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", cfile, "\x1b[0m")
	} else {
		logger.Fatalf("Could not create view file: %s", err)
	}

	cfile = path.Join(absViewPath, "show.tpl")
	if f, err := os.OpenFile(cfile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer CloseFile(f)
		f.WriteString(cfile)
		fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", cfile, "\x1b[0m")
	} else {
		logger.Fatalf("Could not create view file: %s", err)
	}

	cfile = path.Join(absViewPath, "create.tpl")
	if f, err := os.OpenFile(cfile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer CloseFile(f)
		f.WriteString(cfile)
		fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", cfile, "\x1b[0m")
	} else {
		logger.Fatalf("Could not create view file: %s", err)
	}

	cfile = path.Join(absViewPath, "edit.tpl")
	if f, err := os.OpenFile(cfile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer CloseFile(f)
		f.WriteString(cfile)
		fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", cfile, "\x1b[0m")
	} else {
		logger.Fatalf("Could not create view file: %s", err)
	}
}
