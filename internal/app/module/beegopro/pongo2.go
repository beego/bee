package beegopro

import (
	"github.com/beego/bee/utils"
	"github.com/flosch/pongo2"
	"strings"
	"unicode/utf8"
)

func init() {
	_ = pongo2.RegisterFilter("lowerFirst", pongo2LowerFirst)
	_ = pongo2.RegisterFilter("upperFirst", pongo2UpperFirst)
	_ = pongo2.RegisterFilter("snakeString", pongo2SnakeString)
	_ = pongo2.RegisterFilter("camelString", pongo2CamelString)
}

func pongo2LowerFirst(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if in.Len() <= 0 {
		return pongo2.AsValue(""), nil
	}
	t := in.String()
	r, size := utf8.DecodeRuneInString(t)
	return pongo2.AsValue(strings.ToLower(string(r)) + t[size:]), nil
}

func pongo2UpperFirst(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if in.Len() <= 0 {
		return pongo2.AsValue(""), nil
	}
	t := in.String()
	return pongo2.AsValue(strings.Replace(t, string(t[0]), strings.ToUpper(string(t[0])), 1)), nil
}

// snake string, XxYy to xx_yy
func pongo2SnakeString(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if in.Len() <= 0 {
		return pongo2.AsValue(""), nil
	}
	t := in.String()
	return pongo2.AsValue(utils.SnakeString(t)), nil
}

// snake string, XxYy to xx_yy
func pongo2CamelString(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if in.Len() <= 0 {
		return pongo2.AsValue(""), nil
	}
	t := in.String()
	return pongo2.AsValue(utils.CamelString(t)), nil
}

//func upperFirst(str string) string {
//	return strings.Replace(str, string(str[0]), strings.ToUpper(string(str[0])), 1)
//}

func lowerFirst(str string) string {
	return strings.Replace(str, string(str[0]), strings.ToLower(string(str[0])), 1)
}
