package api

import (
	"net/http"

	"example.com/myapp/app/middleware"
	"example.com/myapp/models"
	"example.com/myapp/utils/helper"
	"github.com/labstack/echo/v4"
)

type Api struct {
	instance   *echo.Echo
	helper     *helper.Helper
	models     *models.Models
	middleware *middleware.Middleware
	config     *models.Config
}

func (api *Api) Routes() *echo.Echo {
	// 处理路由未找到的情况
	api.instance.RouteNotFound("/*", func(c echo.Context) error {
		return api.helper.NotFoundResponse(c)
	})

	// 自定义错误处理器来处理方法不允许的情况
	api.instance.HTTPErrorHandler = func(err error, c echo.Context) {
		if httpErr, ok := err.(*echo.HTTPError); ok {
			if httpErr.Code == http.StatusMethodNotAllowed {
				returnErr := api.helper.MethodNotAllowedResponse(c)
				if returnErr != nil {
					api.instance.Logger.Error(returnErr)
				}
				return
			}
		}
		// 其他错误使用默认的错误处理器
		api.instance.DefaultHTTPErrorHandler(err, c)
	}

	// 健康检查接口
	api.instance.GET("/v1/healthcheck", func(c echo.Context) error {
		data := map[string]interface{}{
			"status": "available",
			"system_info": map[string]string{
				"environment": api.config.Env,
				"version":     api.Version(),
			},
		}
		resp := api.helper.NewResponse(0, data)
		return api.helper.WriteJSON(c.Response().Writer, http.StatusOK, resp, nil)
	})

	// 注册路由
	api.UserRoutes()

	// 注册中间件
	api.instance.Use(api.middleware.LogRequest)
	api.instance.Use(api.middleware.RecoverPanic)
	api.instance.Use(api.middleware.RateLimit)

	return api.instance
}

func New(handler *echo.Echo, helper *helper.Helper, config *models.Config) (*Api, error) {
	models, err := models.NewModels(&config.Database, helper)
	if err != nil {
		return nil, err
	}
	middlerware := middleware.New(helper, &config.Limiter, models)
	return &Api{
		config:     config,
		helper:     helper,
		models:     models,
		instance:   handler,
		middleware: middlerware,
	}, nil
}

func (api *Api) Version() string {
	return "v1.0.0"
}
