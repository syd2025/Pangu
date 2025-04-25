package api

func (api *Api) Login() {
	api.instance.POST("/v1/user/login", api.login)
	api.instance.GET("/v1/user/brief", api.getUserBrief)
	api.instance.POST("/v1/user/logout", api.logout)

	authGroup := api.instance.Group("/v1/user")
	authGroup.Use(api.middleware.Authenticate)
	authGroup.GET("/avatar", api.getUserAvatar)
	authGroup.GET("/profile", api.getUserProfile)

}
