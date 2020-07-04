package beegopro

import (
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/utils"
	"strings"
)

type ModelContent struct {
	Names    []string
	Orms     []string
	Comments []string
	Extends  []string
}

type ModelInfo struct {
	Name      string `json:"name"`      // mysql name
	InputType string `json:"inputType"` // user input type
	MysqlType string `json:"mysqlType"` // mysql type
	GoType    string `json:"goType"`    // go type
	Orm       string `json:"orm"`       // orm tag
	Comment   string `json:"comment"`   // mysql comment
	Extend    string `json:"extend"`    // user extend info
}

func (m ModelInfo) GetColumnKey() (columnKey string) {
	if m.InputType == "auto" || m.Orm == "pk" {
		columnKey = "PRI"
	}
	return
}

func (m ModelInfo) IsPrimaryKey() (flag bool) {
	if m.Orm == "auto" || m.Orm == "pk" || strings.ToLower(m.Name) == "id" {
		flag = true
	}
	return
}

func (content ModelContent) ToModelInfoArr() (output []ModelInfo) {
	namesLen := len(content.Names)
	ormsLen := len(content.Orms)
	commentsLen := len(content.Comments)
	if namesLen != ormsLen && namesLen != commentsLen {
		beeLogger.Log.Fatalf("length error, namesLen is %d, ormsLen is %d, commentsLen is %d", namesLen, ormsLen, commentsLen)
	}
	extendLen := len(content.Extends)
	if extendLen != 0 && extendLen != namesLen {
		beeLogger.Log.Fatalf("extend length error, namesLen is %d, extendsLen is %d", namesLen, extendLen)
	}

	output = make([]ModelInfo, 0)
	for i, name := range content.Names {
		comment := content.Comments[i]
		if comment == "" {
			comment = name
		}
		inputType, goType, mysqlType, ormTag := getModelType(content.Orms[i])

		m := ModelInfo{
			Name:      name,
			InputType: inputType,
			GoType:    goType,
			Orm:       ormTag,
			Comment:   comment,
			MysqlType: mysqlType,
			Extend:    "",
		}
		// extend value
		if extendLen != 0 {
			m.Extend = content.Extends[i]
		}
		output = append(output, m)
	}
	return
}

func (content ModelContent) ToModelSchemas() (output ModelSchemas) {
	modelInfoArr := content.ToModelInfoArr()

	output = make(ModelSchemas, 0)
	for i, value := range modelInfoArr {
		if i == 0 && !value.IsPrimaryKey() {
			inputType, goType, mysqlType, ormTag := getModelType("auto")
			output = append(output, &ModelSchema{
				Name:      "id",
				InputType: inputType,
				ColumnKey: "PRI",
				Comment:   "ID",
				MysqlType: mysqlType,
				GoType:    goType,
				OrmTag:    ormTag,
				Extend:    value.Extend,
			})
		}

		modelSchema := &ModelSchema{
			Name:      value.Name,
			InputType: value.InputType,
			ColumnKey: value.GetColumnKey(),
			MysqlType: value.MysqlType,
			Comment:   value.Comment,
			GoType:    value.GoType,
			OrmTag:    value.Orm,
		}
		output = append(output, modelSchema)
	}
	return
}

type ModelSchema struct {
	Name      string // column name
	InputType string // user input type
	MysqlType string // mysql type
	ColumnKey string // PRI
	Comment   string // comment
	GoType    string // go type
	OrmTag    string // orm tag
	Extend    string
}

type ModelSchemas []*ModelSchema

func (m ModelSchemas) IsExistTime() bool {
	for _, value := range m {
		if value.InputType == "datetime" {
			return true
		}
	}
	return false
}

func (m ModelSchemas) GetPrimaryKey() string {
	camelPrimaryKey := ""
	for _, value := range m {
		if value.ColumnKey == "PRI" {
			camelPrimaryKey = utils.CamelString(value.Name)
		}
	}
	return camelPrimaryKey
}
