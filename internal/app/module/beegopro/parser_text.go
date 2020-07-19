package beegopro

type TextParser struct {
	userOption UserOption
	tmplOption TmplOption
}

func (t *TextParser) RegisterOption(userOption UserOption, tmplOption TmplOption) {
	t.userOption = userOption
	t.tmplOption = tmplOption
}

func (*TextParser) Parse(descriptor Descriptor) {

}

func (t *TextParser) GetRenderInfos(descriptor Descriptor) (output []RenderInfo) {
	output = make([]RenderInfo, 0)
	// model table name, model table schema
	for modelName, content := range t.userOption.Models {
		output = append(output, RenderInfo{
			Module:     descriptor.Module,
			ModelName:  modelName,
			Content:    content.ToModelInfos(),
			Option:     t.userOption,
			Descriptor: descriptor,
			TmplPath:   t.tmplOption.RenderPath,
		})
	}
	return
}

func (t *TextParser) Unregister() {

}
