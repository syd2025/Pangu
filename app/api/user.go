package api

func (api *Api) UserRoutes() {
	api.instance.POST("/v1/user/login", api.login)
	api.instance.GET("/v1/user/avatar", api.getUserAvatar)

	authGroup := api.instance.Group("/v1/user")
	authGroup.Use(api.middleware.Authenticate)
	authGroup.GET("/profile", api.getUserProfile)
	authGroup.POST("/logout", api.logout)
	authGroup.GET("/brief", api.getUserBrief)
	authGroup.PATCH("/avatar", api.uploadUserAvatar)
}
