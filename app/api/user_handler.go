package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example.com/myapp/models"
	"example.com/myapp/utils/validator"
	"github.com/labstack/echo/v4"
)

// login 用户登录接口
func (api *Api) login(c echo.Context) error {
	var input struct {
		Type    int    `json:"type"`
		Account string `json:"account"`
		Code    string `json:"code"`
	}
	err := api.helper.ReadJSON(c.Response().Writer, c.Request(), &input)
	if err != nil {
		return api.helper.BadRequestResponse(c, err)
	}

	v := validator.New()
	v.Check(validator.Matches(input.Account, validator.EmailRX), "account", "must be a valid email address")
	v.Check(input.Code != "", "code", "must be provided")
	if !v.Valid() {
		return api.helper.FailedValidationResponse(c, v.Errors)
	}

	user, err := api.models.Users.GetUserBriefByAccountOrID(input.Account, 0)
	if err != nil {
		return api.helper.InvalidCredentialsResponse(c)
	}

	if user.PasswdHash == nil {
		return api.helper.ErrorResponse(c, http.StatusUnauthorized, "Password is Not Set")
	}

	math, err := api.helper.ComparePasswordAndHash(input.Code, user.PasswdHash)
	if err != nil {
		return api.helper.ServerErrorResponse(c, err)
	} else if !math {
		return api.helper.InvalidCredentialsResponse(c)
	}

	token, err := api.models.Tokens.New(user.ID, 30*24*time.Hour, models.ScopeAuthentication)
	if err != nil {
		return api.helper.ServerErrorResponse(c, err)
	}

	data := map[string]interface{}{
		"token": token.PlainText,
		"brief": user,
	}

	resp := api.helper.NewResponse(0, data)
	err = api.helper.WriteJSON(c.Response().Writer, http.StatusOK, resp, nil)
	if err != nil {
		return api.helper.ServerErrorResponse(c, err)
	}
	return nil
}

// getUserAvatarById 获取用户头像接口
func (api *Api) getUserAvatar(c echo.Context) error {
	qs := c.Request().URL.Query()
	avatar := api.helper.ReadString(qs, "k", "")
	if avatar == "" {
		user := api.middleware.ContextGetUser(c)
		avatarFromDB, err := api.models.Users.GetUserAvatarById(user.ID)
		if err != nil {
			return api.helper.ServerErrorResponse(c, err)
		}
		if avatarFromDB == nil {
			return api.helper.NotFoundResponse(c)
		}
		avatar = *avatarFromDB
	}
	filepath := filepath.Join(api.config.Path.AvatarsPath(), avatar)
	http.ServeFile(c.Response().Writer, c.Request(), filepath)
	return nil
}

// uploadUserAvatar 上传用户头像接口
func (api *Api) uploadUserAvatar(c echo.Context) error {
	user := api.middleware.ContextGetUser(c)
	err := c.Request().ParseMultipartForm(10 << 20)
	if err != nil {
		return api.helper.BadRequestResponse(c, err)
	}
	file, header, err := c.Request().FormFile("avatar")
	if err != nil {
		return api.helper.BadRequestResponse(c, err)
	}
	defer file.Close()

	dstExt := filepath.Ext(header.Filename)
	dstName := api.helper.RandomString(16) + dstExt
	dstPath := filepath.Join(api.config.Path.AvatarsPath(), dstName)
	dst, err := os.Create(dstPath)
	if err != nil {
		return api.helper.ServerErrorResponse(c, err)

	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		return api.helper.ServerErrorResponse(c, err)
	}

	old, err := api.models.Users.UpdateUserAvatar(user.ID, dstName)
	if err != nil {
		os.Remove(dstPath)
		return api.helper.ServerErrorResponse(c, err)
	}

	if old != "" && old != "default_avatar.png" {
		os.Remove(filepath.Join(api.config.Path.AvatarsPath(), old))
	}

	var output = struct {
		ID     int64  `json:"id"`
		Field  string `json:"field"`
		Avatar string `json:"value"`
	}{
		ID:     user.ID,
		Field:  "avatar",
		Avatar: dstName,
	}

	data := map[string]interface{}{
		"output": output,
	}

	resp := api.helper.NewResponse(0, data)
	err = api.helper.WriteJSON(c.Response().Writer, http.StatusOK, resp, nil)
	if err != nil {
		return api.helper.ServerErrorResponse(c, err)
	}
	return nil
}

// getUserProfile 获取用户信息接口
func (api *Api) getUserProfile(c echo.Context) error {
	user := api.middleware.ContextGetUser(c)
	profile, err := api.models.Users.GetUserProfileByID(user.ID)
	if err != nil {
		api.helper.ServerErrorResponse(c, err)
		return nil
	}
	data := map[string]interface{}{
		"profile": profile,
	}
	resp := api.helper.NewResponse(0, data)
	err = api.helper.WriteJSON(c.Response().Writer, http.StatusOK, resp, nil)
	if err != nil {
		api.helper.ServerErrorResponse(c, err)
		return err
	}
	return nil
}

// updateUserProfile 获取用户信息接口
func (api *Api) getUserBrief(c echo.Context) error {
	user := api.middleware.ContextGetUser(c)
	brief, err := api.models.Users.GetUserBriefByAccountOrID("", user.ID)
	if err != nil {
		api.helper.ServerErrorResponse(c, err)
		return err
	}

	data := map[string]interface{}{
		"brief": brief,
	}

	resp := api.helper.NewResponse(0, data)
	err = api.helper.WriteJSON(c.Response().Writer, http.StatusOK, resp, nil)
	if err != nil {
		api.helper.ServerErrorResponse(c, err)
	}
	return nil
}

// logout 用户登出接口
func (api *Api) logout(c echo.Context) error {
	authorizationHeader := c.Request().Header.Get("Authorization")
	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return api.helper.InvalidCredentialsResponse(c)
	}

	token := headerParts[1]
	err := api.models.Tokens.DeleteToken(token)
	if err != nil {
		return api.helper.ServerErrorResponse(c, err)
	}

	data := map[string]interface{}{}
	resp := api.helper.NewResponse(0, data)
	err = api.helper.WriteJSON(c.Response().Writer, http.StatusOK, resp, nil)
	if err != nil {
		return api.helper.ServerErrorResponse(c, err)
	}
	return nil
}
