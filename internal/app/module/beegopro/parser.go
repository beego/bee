package beegopro

type Parser interface {
	RegisterOption(userOption UserOption, tmplOption TmplOption)
	Parse(descriptor Descriptor)
	GetRenderInfos(descriptor Descriptor) (output []RenderInfo)
	Unregister()
}

var ParserDriver = map[string]Parser{
	"text":  &TextParser{},
	"mysql": &MysqlParser{},
}
