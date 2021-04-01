package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"html/template"
	"log"
	"regexp"
)

//go:embed  templates/*
var f embed.FS

func main() {
	configPath := flag.String("config", "config.yaml", "path to config")
	flag.Parse()
	cnf := Config{}
	if err := LoadConfig(*configPath, &cnf); err != nil {
		log.Fatalln(err.Error())
		return
	}

	gin.SetMode(cnf.Mode)
	r := gin.Default()
	temple := template.Must(template.New("").ParseFS(f, "templates/*.tmpl"))
	r.SetHTMLTemplate(temple)

	filterReg := regexp.MustCompile(cnf.Filter)
	watcher, err := NewWatcher(cnf.Folder, func(path string) bool {
		return filterReg.MatchString(path)
	})
	if err != nil {
		log.Fatalln(err.Error())
		return
	}

	defer watcher.Close()
	watcher.Watch()

	r.GET("", watcher.GetInfo)
	r.GET("api/*any", watcher.GetData)

	if cnf.TLSConfig.Enabled {
		log.Fatalln(r.RunTLS(fmt.Sprintf(":%d", cnf.Port), cnf.TLSConfig.Cert, cnf.TLSConfig.Key))
	} else {
		log.Fatalln(r.Run(fmt.Sprintf(":%d", cnf.Port)))
	}

}
