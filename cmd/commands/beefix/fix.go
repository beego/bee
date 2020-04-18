package beefix

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/logger/colors"
)

var CmdFix = &commands.Command{
	UsageLine: "fix",
	Short:     "Fixes your application by making it compatible with newer versions of Beego",
	Long: `As of {{"Beego 1.6"|bold}}, there are some backward compatibility issues.

  The command 'fix' will try to solve those issues by upgrading your code base
  to be compatible  with Beego version 1.6+.
`,
}

func init() {
	CmdFix.Run = runFix
	CmdFix.PreRun = func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() }
	commands.AvailableCommands = append(commands.AvailableCommands, CmdFix)
}

func runFix(cmd *commands.Command, args []string) int {
	output := cmd.Out()

	beeLogger.Log.Info("Upgrading the application...")

	dir, err := os.Getwd()
	if err != nil {
		beeLogger.Log.Fatalf("Error while getting the current working directory: %s", err)
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".exe") {
			return nil
		}
		err = fixFile(path)
		fmt.Fprintf(output, colors.GreenBold("\tfix\t")+"%s\n", path)
		if err != nil {
			beeLogger.Log.Errorf("Could not fix file: %s", err)
		}
		return err
	})
	beeLogger.Log.Success("Upgrade Done!")
	return 0
}

var rules = []string{
	"beego.AppName", "beego.BConfig.AppName",
	"beego.RunMode", "beego.BConfig.RunMode",
	"beego.RecoverPanic", "beego.BConfig.RecoverPanic",
	"beego.RouterCaseSensitive", "beego.BConfig.RouterCaseSensitive",
	"beego.BeegoServerName", "beego.BConfig.ServerName",
	"beego.EnableGzip", "beego.BConfig.EnableGzip",
	"beego.ErrorsShow", "beego.BConfig.EnableErrorsShow",
	"beego.CopyRequestBody", "beego.BConfig.CopyRequestBody",
	"beego.MaxMemory", "beego.BConfig.MaxMemory",
	"beego.Graceful", "beego.BConfig.Listen.Graceful",
	"beego.HttpAddr", "beego.BConfig.Listen.HTTPAddr",
	"beego.HttpPort", "beego.BConfig.Listen.HTTPPort",
	"beego.ListenTCP4", "beego.BConfig.Listen.ListenTCP4",
	"beego.EnableHttpListen", "beego.BConfig.Listen.EnableHTTP",
	"beego.EnableHttpTLS", "beego.BConfig.Listen.EnableHTTPS",
	"beego.HttpsAddr", "beego.BConfig.Listen.HTTPSAddr",
	"beego.HttpsPort", "beego.BConfig.Listen.HTTPSPort",
	"beego.HttpCertFile", "beego.BConfig.Listen.HTTPSCertFile",
	"beego.HttpKeyFile", "beego.BConfig.Listen.HTTPSKeyFile",
	"beego.EnableAdmin", "beego.BConfig.Listen.EnableAdmin",
	"beego.AdminHttpAddr", "beego.BConfig.Listen.AdminAddr",
	"beego.AdminHttpPort", "beego.BConfig.Listen.AdminPort",
	"beego.UseFcgi", "beego.BConfig.Listen.EnableFcgi",
	"beego.HttpServerTimeOut", "beego.BConfig.Listen.ServerTimeOut",
	"beego.AutoRender", "beego.BConfig.WebConfig.AutoRender",
	"beego.ViewsPath", "beego.BConfig.WebConfig.ViewsPath",
	"beego.StaticDir", "beego.BConfig.WebConfig.StaticDir",
	"beego.StaticExtensionsToGzip", "beego.BConfig.WebConfig.StaticExtensionsToGzip",
	"beego.DirectoryIndex", "beego.BConfig.WebConfig.DirectoryIndex",
	"beego.FlashName", "beego.BConfig.WebConfig.FlashName",
	"beego.FlashSeperator", "beego.BConfig.WebConfig.FlashSeparator",
	"beego.EnableDocs", "beego.BConfig.WebConfig.EnableDocs",
	"beego.XSRFKEY", "beego.BConfig.WebConfig.XSRFKey",
	"beego.EnableXSRF", "beego.BConfig.WebConfig.EnableXSRF",
	"beego.XSRFExpire", "beego.BConfig.WebConfig.XSRFExpire",
	"beego.TemplateLeft", "beego.BConfig.WebConfig.TemplateLeft",
	"beego.TemplateRight", "beego.BConfig.WebConfig.TemplateRight",
	"beego.SessionOn", "beego.BConfig.WebConfig.Session.SessionOn",
	"beego.SessionProvider", "beego.BConfig.WebConfig.Session.SessionProvider",
	"beego.SessionName", "beego.BConfig.WebConfig.Session.SessionName",
	"beego.SessionGCMaxLifetime", "beego.BConfig.WebConfig.Session.SessionGCMaxLifetime",
	"beego.SessionSavePath", "beego.BConfig.WebConfig.Session.SessionProviderConfig",
	"beego.SessionCookieLifeTime", "beego.BConfig.WebConfig.Session.SessionCookieLifeTime",
	"beego.SessionAutoSetCookie", "beego.BConfig.WebConfig.Session.SessionAutoSetCookie",
	"beego.SessionDomain", "beego.BConfig.WebConfig.Session.SessionDomain",
	"Ctx.Input.CopyBody(", "Ctx.Input.CopyBody(beego.BConfig.MaxMemory",
	".UrlFor(", ".URLFor(",
	".ServeJson(", ".ServeJSON(",
	".ServeXml(", ".ServeXML(",
	".ServeJsonp(", ".ServeJSONP(",
	".XsrfToken(", ".XSRFToken(",
	".CheckXsrfCookie(", ".CheckXSRFCookie(",
	".XsrfFormHtml(", ".XSRFFormHTML(",
	"beego.UrlFor(", "beego.URLFor(",
	"beego.GlobalDocApi", "beego.GlobalDocAPI",
	"beego.Errorhandler", "beego.ErrorHandler",
	"Output.Jsonp(", "Output.JSONP(",
	"Output.Json(", "Output.JSON(",
	"Output.Xml(", "Output.XML(",
	"Input.Uri()", "Input.URI()",
	"Input.Url()", "Input.URL()",
	"Input.AcceptsHtml()", "Input.AcceptsHTML()",
	"Input.AcceptsXml()", "Input.AcceptsXML()",
	"Input.AcceptsJson()", "Input.AcceptsJSON()",
	"Ctx.XsrfToken()", "Ctx.XSRFToken()",
	"Ctx.CheckXsrfCookie()", "Ctx.CheckXSRFCookie()",
	"session.SessionStore", "session.Store",
	".TplNames", ".TplName",
	"swagger.ApiRef", "swagger.APIRef",
	"swagger.ApiDeclaration", "swagger.APIDeclaration",
	"swagger.Api", "swagger.API",
	"swagger.ApiRef", "swagger.APIRef",
	"swagger.Infomation", "swagger.Information",
	"toolbox.UrlMap", "toolbox.URLMap",
	"logs.LoggerInterface", "logs.Logger",
	"Input.Request", "Input.Context.Request",
	"Input.Params)", "Input.Params())",
	"httplib.BeegoHttpSettings", "httplib.BeegoHTTPSettings",
	"httplib.BeegoHttpRequest", "httplib.BeegoHTTPRequest",
	".TlsClientConfig", ".TLSClientConfig",
	".JsonBody", ".JSONBody",
	".ToJson", ".ToJSON",
	".ToXml", ".ToXML",
	"beego.Html2str", "beego.HTML2str",
	"beego.AssetsCss", "beego.AssetsCSS",
	"orm.DR_Sqlite", "orm.DRSqlite",
	"orm.DR_Postgres", "orm.DRPostgres",
	"orm.DR_MySQL", "orm.DRMySQL",
	"orm.DR_Oracle", "orm.DROracle",
	"orm.Col_Add", "orm.ColAdd",
	"orm.Col_Minus", "orm.ColMinus",
	"orm.Col_Multiply", "orm.ColMultiply",
	"orm.Col_Except", "orm.ColExcept",
	"GenerateOperatorSql", "GenerateOperatorSQL",
	"OperatorSql", "OperatorSQL",
	"orm.Debug_Queries", "orm.DebugQueries",
	"orm.COMMA_SPACE", "orm.CommaSpace",
	".SendOut()", ".DoRequest()",
	"validation.ValidationError", "validation.Error",
}

func fixFile(file string) error {
	rp := strings.NewReplacer(rules...)
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	fixed := rp.Replace(string(content))

	// Forword the RequestBody from the replace
	// "Input.Request", "Input.Context.Request",
	fixed = strings.Replace(fixed, "Input.Context.RequestBody", "Input.RequestBody", -1)

	// Regexp replace
	pareg := regexp.MustCompile(`(Input.Params\[")(.*)("])`)
	fixed = pareg.ReplaceAllString(fixed, "Input.Param(\"$2\")")
	pareg = regexp.MustCompile(`(Input.Data\[\")(.*)(\"\])(\s)(=)(\s)(.*)`)
	fixed = pareg.ReplaceAllString(fixed, "Input.SetData(\"$2\", $7)")
	pareg = regexp.MustCompile(`(Input.Data\[\")(.*)(\"\])`)
	fixed = pareg.ReplaceAllString(fixed, "Input.Data(\"$2\")")
	// Fix the cache object Put method
	pareg = regexp.MustCompile(`(\.Put\(\")(.*)(\",)(\s)(.*)(,\s*)([^\*.]*)(\))`)
	if pareg.MatchString(fixed) && strings.HasSuffix(file, ".go") {
		fixed = pareg.ReplaceAllString(fixed, ".Put(\"$2\", $5, $7*time.Second)")
		fset := token.NewFileSet() // positions are relative to fset
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			panic(err)
		}
		// Print the imports from the file's AST.
		hasTimepkg := false
		for _, s := range f.Imports {
			if s.Path.Value == `"time"` {
				hasTimepkg = true
				break
			}
		}
		if !hasTimepkg {
			fixed = strings.Replace(fixed, "import (", "import (\n\t\"time\"", 1)
		}
	}
	// Replace the v.Apis in docs.go
	if strings.Contains(file, "docs.go") {
		fixed = strings.Replace(fixed, "v.Apis", "v.APIs", -1)
	}
	// Replace the config file
	if strings.HasSuffix(file, ".conf") {
		fixed = strings.Replace(fixed, "HttpCertFile", "HTTPSCertFile", -1)
		fixed = strings.Replace(fixed, "HttpKeyFile", "HTTPSKeyFile", -1)
		fixed = strings.Replace(fixed, "EnableHttpListen", "HTTPEnable", -1)
		fixed = strings.Replace(fixed, "EnableHttpTLS", "EnableHTTPS", -1)
		fixed = strings.Replace(fixed, "EnableHttpTLS", "EnableHTTPS", -1)
		fixed = strings.Replace(fixed, "BeegoServerName", "ServerName", -1)
		fixed = strings.Replace(fixed, "AdminHttpAddr", "AdminAddr", -1)
		fixed = strings.Replace(fixed, "AdminHttpPort", "AdminPort", -1)
		fixed = strings.Replace(fixed, "HttpServerTimeOut", "ServerTimeOut", -1)
	}
	err = os.Truncate(file, 0)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, []byte(fixed), 0666)
}
