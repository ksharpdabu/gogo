package commands

import (
	"log"
	"strings"
	"text/template"

	"github.com/dolab/logger"
)

var (
	box    *template.Template
	stderr *logger.Logger
)

type templateData struct {
	Namespace   string
	Application string
}

func init() {
	var err error

	// setup logger
	stderr, err = logger.New("stderr")
	if err != nil {
		panic(err.Error())
	}

	stderr.SetLevelByName("info")
	stderr.SetFlag(log.Lshortfile)

	// register templates
	box = template.New("gogo").Funcs(template.FuncMap{
		"lowercase": strings.ToLower,
	})

	template.Must(box.New("env").Parse(envTemplate))
	template.Must(box.New("makefile").Parse(Space2Tab(makefileTemplate)))
	template.Must(box.New("gitignore").Parse(gitIgnoreTemplate))
	template.Must(box.New("readme").Parse(readmeTemplate))
	template.Must(box.New("main").Parse(Space2Tab(mainTemplate)))
	// controllers
	template.Must(box.New("application").Parse(Space2Tab(applicationTemplates["application"])))
	template.Must(box.New("application_testing").Parse(Space2Tab(applicationTemplates["application_testing"])))
	template.Must(box.New("application_config").Parse(Space2Tab(applicationTemplates["application_config"])))
	template.Must(box.New("application_config_json").Parse(Space2Tab(applicationTemplates["application_config_json"])))
	template.Must(box.New("application_config_test").Parse(Space2Tab(applicationTemplates["application_config_test"])))
	template.Must(box.New("getting_start").Parse(Space2Tab(applicationTemplates["getting_start"])))
	template.Must(box.New("getting_start_test").Parse(Space2Tab(applicationTemplates["getting_start_test"])))
	// middlewares
	template.Must(box.New("middleware_testing").Parse(Space2Tab(applicationTemplates["middleware_testing"])))
	template.Must(box.New("middleware_recovery").Parse(Space2Tab(applicationTemplates["middleware_recovery"])))
	template.Must(box.New("middleware_recovery_test").Parse(Space2Tab(applicationTemplates["middleware_recovery_test"])))
	// models
	template.Must(box.New("model").Parse(Space2Tab(applicationTemplates["model"])))
	template.Must(box.New("model_test").Parse(Space2Tab(applicationTemplates["model_test"])))
	// templates
	template.Must(box.New("template_controller").Parse(Space2Tab(componentTemplates["controller"])))
	template.Must(box.New("template_controller_test").Parse(Space2Tab(componentTemplates["controller_test"])))
	template.Must(box.New("template_middleware").Parse(Space2Tab(componentTemplates["middleware"])))
	template.Must(box.New("template_middleware_test").Parse(Space2Tab(componentTemplates["middleware_test"])))
	template.Must(box.New("template_model").Parse(Space2Tab(componentTemplates["model"])))
	template.Must(box.New("template_model_test").Parse(Space2Tab(componentTemplates["model_test"])))
}

func Box() *template.Template {
	return box
}
