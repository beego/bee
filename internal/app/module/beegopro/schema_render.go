package beegopro

type RenderInfo struct {
	Module       string
	ModelName    string
	Option       UserOption
	Content      ModelInfos
	Descriptor   Descriptor
	TmplPath     string
	GenerateTime string
}
