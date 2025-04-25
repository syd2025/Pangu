package app

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"example.com/myapp/app/api"
	"example.com/myapp/models"
	"example.com/myapp/utils/helper"
	"example.com/myapp/utils/jsonlog"
	"github.com/labstack/echo/v4"
)

type Application struct {
	config *models.Config
	server *http.Server
	api    *api.Api
	helper *helper.Helper
}

func New(handler *echo.Echo, configPath, currentDir string) (*Application, error) {
	config, err := models.NewConfig(configPath, currentDir)
	if err != nil {
		return nil, err
	}

	prettyLog := config.Env == "dev"

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo, prettyLog)
	helper := helper.New(logger)
	api, err := api.New(handler, helper, config)
	if err != nil {
		return nil, err
	}

	return &Application{
		config: config,
		api:    api,
		helper: helper,
	}, nil
}

func (app *Application) runServer() error {
	tlsCfg := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		MinVersion:       tls.VersionTLS13,
	}

	addr := fmt.Sprintf("%s:%d", app.config.Server, app.config.Port)
	app.server = &http.Server{
		Addr:      addr,
		Handler:   app.api.Routes(),
		TLSConfig: tlsCfg,

		IdleTimeout:    time.Minute,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 8192,
	}

	app.helper.Logger.Info(fmt.Sprintf("Starting server on %s", addr), map[string]string{
		"addr": addr,
		"env":  app.config.Env,
	})
	return app.server.ListenAndServeTLS(app.config.Path.CertPath(), app.config.Path.KeyPath())
}

func (app *Application) Run(e *echo.Echo) error {
	err := app.runServer()
	if err != nil {
		return err
	}
	return nil
}
