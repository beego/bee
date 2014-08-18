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
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var cmdRundocs = &Command{
	UsageLine: "rundocs [-isDownload=true] [-docport=8888]",
	Short:     "rundocs will run the docs server,default is 8089",
	Long: `
-d meaning will download the docs file from github
-p meaning server the Server on which port, default is 8089

`,
}

const (
	swaggerlink = "https://github.com/beego/swagger/archive/v1.zip"
)

type docValue string

func (d *docValue) String() string {
	return fmt.Sprint(*d)
}

func (d *docValue) Set(value string) error {
	*d = docValue(value)
	return nil
}

var isDownload docValue
var docport docValue

func init() {
	cmdRundocs.Run = runDocs
	cmdRundocs.Flag.Var(&isDownload, "isDownload", "weather download the Swagger Docs")
	cmdRundocs.Flag.Var(&docport, "docport", "doc server port")
}

func runDocs(cmd *Command, args []string) int {
	if isDownload == "true" {
		downloadFromUrl(swaggerlink, "swagger.zip")
		err := unzipAndDelete("swagger.zip", "swagger")
		if err != nil {
			fmt.Println("has err exet unzipAndDelete", err)
		}
	}
	if docport == "" {
		docport = "8089"
	}
	if _, err := os.Stat("swagger"); err != nil && os.IsNotExist(err) {
		fmt.Println("there's no swagger, please use bee rundocs -isDownload=true downlaod first")
		os.Exit(2)
	}
	fmt.Println("start the docs server on: http://127.0.0.1:" + docport)
	log.Fatal(http.ListenAndServe(":"+string(docport), http.FileServer(http.Dir("swagger"))))
	return 0
}

func downloadFromUrl(url, fileName string) {
	fmt.Println("Downloading", url, "to", fileName)

	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}

	fmt.Println(n, "bytes downloaded.")
}

func unzipAndDelete(src, dest string) error {
	fmt.Println("start to unzip file from " + src + " to " + dest)
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			f, err := os.OpenFile(
				path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("Start delete src file " + src)
	err = os.RemoveAll(src)
	if err != nil {
		return err
	}
	return nil
}
