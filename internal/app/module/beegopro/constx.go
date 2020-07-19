package beegopro

const BeegoToml = `
	dsn = "root:123456@tcp(127.0.0.1:3306)/beego"
	driver = "mysql"
	proType = "default"
	enableModule = []
	apiPrefix = "/"
	gitRemotePath = "https://github.com/beego-dev/beego-pro.git"
	format = true
	sourceGen = "text"
	gitPull = true
	[models.user]
		name = ["uid"]
		orm = ["auto"]
		comment = ["Uid"]
		
`
