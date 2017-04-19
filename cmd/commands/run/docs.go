package run

import (
	"archive/zip"
	"io"
	"net/http"
	"os"
	"strings"

	beeLogger "github.com/beego/bee/logger"
)

var (
	swaggerVersion = "3"
	swaggerlink    = "https://github.com/beego/swagger/archive/v" + swaggerVersion + ".zip"
)

func downloadFromURL(url, fileName string) {
	var down bool
	if fd, err := os.Stat(fileName); err != nil && os.IsNotExist(err) {
		down = true
	} else if fd.Size() == int64(0) {
		down = true
	} else {
		beeLogger.Log.Infof("'%s' already exists", fileName)
		return
	}
	if down {
		beeLogger.Log.Infof("Downloading '%s' to '%s'...", url, fileName)
		output, err := os.Create(fileName)
		if err != nil {
			beeLogger.Log.Errorf("Error while creating '%s': %s", fileName, err)
			return
		}
		defer output.Close()

		response, err := http.Get(url)
		if err != nil {
			beeLogger.Log.Errorf("Error while downloading '%s': %s", url, err)
			return
		}
		defer response.Body.Close()

		n, err := io.Copy(output, response.Body)
		if err != nil {
			beeLogger.Log.Errorf("Error while downloading '%s': %s", url, err)
			return
		}
		beeLogger.Log.Successf("%d bytes downloaded!", n)
	}
}

func unzipAndDelete(src string) error {
	beeLogger.Log.Infof("Unzipping '%s'...", src)
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
	beeLogger.Log.Successf("Done! Deleting '%s'...", src)
	return os.RemoveAll(src)
}
