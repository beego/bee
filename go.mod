module github.com/beego/bee/v2

go 1.16

require (
	github.com/beego/beego/v2 v2.0.1
	github.com/davecgh/go-spew v1.1.1
	github.com/flosch/pongo2 v0.0.0-20200529170236-5abacdfa4915
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/go-delve/delve v1.5.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/gorilla/websocket v1.4.2
	github.com/lib/pq v1.7.0
	github.com/pelletier/go-toml v1.8.1
	github.com/smartwalle/pongo2render v1.0.1
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/talos-systems/talos/pkg/machinery v0.0.0-20210625144407-2060ceaa0b16
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/text v0.3.6
	golang.org/x/tools v0.1.3
	gopkg.in/yaml.v2 v2.4.0
	honnef.co/go/tools v0.1.4 // indirect
)

//replace github.com/beego/beego/v2 => ../beego
