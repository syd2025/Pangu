package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"example.com/myapp/app"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	var configPath string
	// 启动命令
	flag.StringVar(&configPath, "config", "./config-dev.yaml", "Path to the configuration file")
	flag.Parse()

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %v", err)
		return
	}

	if !strings.HasPrefix(configPath, "/") {
		configPath = filepath.Join(currentDir, configPath)
	}

	app, err := app.New(e, configPath, currentDir)
	if err != nil {
		log.Fatalf("Error creating application: %v", err)
		return
	}

	err = app.Run(e)
	if err != nil {
		log.Fatalf("Error running application: %v", err)
		return
	}
}
