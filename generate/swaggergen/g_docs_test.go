package swaggergen

import "testing"

func TestGenerateDocs(t *testing.T) {
	dir := "/Users/heldiam/Developer/GO/src/code.aliyun.com/zhizaofang/factoryshop/application/web"
	ParsePackagesFromDir(dir)
	GenerateDocs(dir)
}
