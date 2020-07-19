package beegopro

import (
	beeLogger "github.com/beego/bee/logger"
)

type TextModel struct {
	Names    []string
	Orms     []string
	Comments []string
	Extends  []string
}

func (content TextModel) ToModelInfos() (output []ModelInfo) {
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
