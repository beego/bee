package beegopro

import (
	"database/sql"
	"errors"
	"github.com/beego/bee/logger"
)

type TableSchema struct {
	TableName              string
	ColumnName             string
	IsNullable             string
	DataType               string
	CharacterMaximumLength sql.NullInt64
	NumericPrecision       sql.NullInt64
	NumericScale           sql.NullInt64
	ColumnType             string
	ColumnKey              string
	Comment                string
}

type TableSchemas []TableSchema

func (tableSchemas TableSchemas) ToTableMap() (resp map[string]ModelInfos) {

	resp = make(map[string]ModelInfos)
	for _, value := range tableSchemas {
		if _, ok := resp[value.TableName]; !ok {
			resp[value.TableName] = make(ModelInfos, 0)
		}

		modelInfos := resp[value.TableName]
		inputType, goType, err := value.ToGoType()
		if err != nil {
			beeLogger.Log.Fatalf("parse go type err %s", err)
			return
		}

		modelInfo := ModelInfo{
			Name:      value.ColumnName,
			InputType: inputType,
			GoType:    goType,
			Comment:   value.Comment,
		}

		if value.ColumnKey == "PRI" {
			modelInfo.Orm = "pk"
		}
		resp[value.TableName] = append(modelInfos, modelInfo)
	}
	return
}

// GetGoDataType maps an SQL data type to Golang data type
func (col TableSchema) ToGoType() (inputType string, goType string, err error) {
	switch col.DataType {
	case "char", "varchar", "enum", "set", "text", "longtext", "mediumtext", "tinytext":
		goType = "string"
	case "blob", "mediumblob", "longblob", "varbinary", "binary":
		goType = "[]byte"
	case "date", "time", "datetime", "timestamp":
		goType, inputType = "time.Time", "dateTime"
	case "tinyint", "smallint", "int", "mediumint":
		goType = "int"
	case "bit", "bigint":
		goType = "int64"
	case "float", "decimal", "double":
		goType = "float64"
	}
	if goType == "" {
		err = errors.New("No compatible datatype (" + col.DataType + ", CamelName: " + col.ColumnName + ")  found")
	}
	return
}
