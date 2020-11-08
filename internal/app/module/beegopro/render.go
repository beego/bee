package beegopro

import (
	"go/format"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/flosch/pongo2"
	"github.com/smartwalle/pongo2render"

	"github.com/beego/bee/internal/pkg/system"
	beeLogger "github.com/beego/bee/logger"
)

// render
type RenderFile struct {
	*pongo2render.Render
	Context      pongo2.Context
	GenerateTime string
	Option       UserOption
	ModelName    string
	PackageName  string
	FlushFile    string
	PkgPath      string
	TmplPath     string
	Descriptor   Descriptor
}

func NewRender(m RenderInfo) *RenderFile {
	var (
		pathCtx       pongo2.Context
		newDescriptor Descriptor
	)

	// parse descriptor, get flush file path, beego path, etc...
	newDescriptor, pathCtx = m.Descriptor.Parse(m.ModelName, m.Option.Path)

	obj := &RenderFile{
		Context:      make(pongo2.Context),
		Option:       m.Option,
		ModelName:    m.ModelName,
		GenerateTime: m.GenerateTime,
		Descriptor:   newDescriptor,
	}

	obj.FlushFile = newDescriptor.DstPath

	// new render
	obj.Render = pongo2render.NewRender(path.Join(obj.Option.GitLocalPath, obj.Option.ProType, m.TmplPath))

	filePath := path.Dir(obj.FlushFile)
	err := createPath(filePath)
	if err != nil {
		beeLogger.Log.Fatalf("Could not create the controllers directory: %s", err)
	}
	// get go package path
	obj.PkgPath = getPackagePath()

	relativePath, err := filepath.Rel(system.CurrentDir, obj.FlushFile)
	if err != nil {
		beeLogger.Log.Fatalf("Could not get the relative path: %s", err)
	}

	modelSchemas := m.Content.ToModelSchemas()
	camelPrimaryKey := modelSchemas.GetPrimaryKey()
	importMaps := make(map[string]struct{})
	if modelSchemas.IsExistTime() {
		importMaps["time"] = struct{}{}
	}
	obj.PackageName = filepath.Base(filepath.Dir(relativePath))
	beeLogger.Log.Infof("Using '%s' as name", obj.ModelName)

	beeLogger.Log.Infof("Using '%s' as package name from %s", obj.ModelName, obj.PackageName)

	// package
	obj.SetContext("packageName", obj.PackageName)
	obj.SetContext("packageImports", importMaps)

	// todo optimize
	// todo Set the beego directory, should recalculate the package
	if pathCtx["pathRelBeego"] == "." {
		obj.SetContext("packagePath", obj.PkgPath)
	} else {
		obj.SetContext("packagePath", obj.PkgPath+"/"+pathCtx["pathRelBeego"].(string))
	}

	obj.SetContext("packageMod", obj.PkgPath)

	obj.SetContext("modelSchemas", modelSchemas)
	obj.SetContext("modelPrimaryKey", camelPrimaryKey)

	for key, value := range pathCtx {
		obj.SetContext(key, value)
	}

	obj.SetContext("apiPrefix", obj.Option.ApiPrefix)
	obj.SetContext("generateTime", obj.GenerateTime)

	if obj.Option.ContextDebug {
		spew.Dump(obj.Context)
	}
	return obj
}

func (r *RenderFile) SetContext(key string, value interface{}) {
	r.Context[key] = value
}

func (r *RenderFile) Exec(name string) {
	var (
		buf string
		err error
	)
	buf, err = r.Render.Template(name).Execute(r.Context)
	if err != nil {
		beeLogger.Log.Fatalf("Could not create the %s render tmpl: %s", name, err)
		return
	}
	_, err = os.Stat(r.Descriptor.DstPath)
	var orgContent []byte
	if err == nil {
		if org, err := os.OpenFile(r.Descriptor.DstPath, os.O_RDONLY, 0666); err == nil {
			orgContent,_ = ioutil.ReadAll(org)
			org.Close()
		} else {
			beeLogger.Log.Infof("file err %s", err)
		}
	}
	// Replace or create when content changes
	output := []byte(buf)
	ext := filepath.Ext(r.FlushFile)
	if r.Option.EnableFormat && ext == ".go" {
		// format code
		var bts []byte
		bts, err = format.Source([]byte(buf))
		if err != nil {
			beeLogger.Log.Warnf("format buf error %s", err.Error())
		}
		output = bts
	}

	if FileContentChange(orgContent, output, GetSeg(ext)) {
		err = r.write(r.FlushFile, output)
		if err != nil {
			beeLogger.Log.Fatalf("Could not create file: %s", err)
			return
		}
		beeLogger.Log.Infof("create file '%s' from %s", r.FlushFile, r.PackageName)
	}
}

