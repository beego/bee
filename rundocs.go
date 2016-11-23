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
	"net/http"
	"os"
	"strings"
)

var cmdRundocs = &Command{
	UsageLine: "rundocs [-isDownload=true] [-docport=8888]",
	Short:     "rundocs will run the docs server,default is 8089",
	Long: `
-d meaning will download the docs file from github
-p meaning server the Server on which port, default is 8089

`,
}

var (
	swaggerVersion = "2"
	swaggerlink    = "https://github.com/beego/swagger/archive/v" + swaggerVersion + ".zip"
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
	cmdRundocs.PreRun = func(cmd *Command, args []string) { ShowShortVersionBanner() }
	cmdRundocs.Flag.Var(&isDownload, "isDownload", "weather download the Swagger Docs")
	cmdRundocs.Flag.Var(&docport, "docport", "doc server port")
}

func runDocs(cmd *Command, args []string) int {
	if isDownload == "true" {
		downloadFromURL(swaggerlink, "swagger.zip")
		err := unzipAndDelete("swagger.zip")
		if err != nil {
			logger.Errorf("Error while unzipping 'swagger.zip' file: %s", err)
		}
	}
	if docport == "" {
		docport = "8089"
	}
	if _, err := os.Stat("swagger"); err != nil && os.IsNotExist(err) {
		logger.Fatal("No Swagger dist found. Run: bee rundocs -isDownload=true")
	}

	logger.Infof("Starting the docs server on: http://127.0.0.1:%s", docport)

	err := http.ListenAndServe(":"+string(docport), http.FileServer(http.Dir("swagger")))
	if err != nil {
		logger.Fatalf("%s", err)
	}
	return 0
}

func downloadFromURL(url, fileName string) {
	var down bool
	if fd, err := os.Stat(fileName); err != nil && os.IsNotExist(err) {
		down = true
	} else if fd.Size() == int64(0) {
		down = true
	} else {
		logger.Infof("'%s' already exists", fileName)
		return
	}
	if down {
		logger.Infof("Downloading '%s' to '%s'...", url, fileName)
		output, err := os.Create(fileName)
		if err != nil {
			logger.Errorf("Error while creating '%s': %s", fileName, err)
			return
		}
		defer output.Close()

		response, err := http.Get(url)
		if err != nil {
			logger.Errorf("Error while downloading '%s': %s", url, err)
			return
		}
		defer response.Body.Close()

		n, err := io.Copy(output, response.Body)
		if err != nil {
			logger.Errorf("Error while downloading '%s': %s", url, err)
			return
		}
		logger.Successf("%d bytes downloaded!", n)
	}
}

func unzipAndDelete(src string) error {
	logger.Infof("Unzipping '%s'...", src)
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	rp := strings.NewReplacer("swagger-"+swaggerVersion, "swagger")
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		fname := rp.Replace(f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fname, f.Mode())
		} else {
			f, err := os.OpenFile(
				fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
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
	logger.Successf("Done! Deleting '%s'...", src)
	return os.RemoveAll(src)
}
