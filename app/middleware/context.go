package middleware

import (
	"context"
	"net/http"

	"example.com/myapp/models"
	"github.com/labstack/echo/v4"
)

type contextKey string

const userContextKey contextKey = "user"

func (m *Middleware) ContextSetUser(c echo.Context, user *models.User) *http.Request {
	ctx := context.WithValue(c.Request().Context(), userContextKey, user)
	return c.Request().WithContext(ctx)
}

func (m *Middleware) ContextGetUser(c echo.Context) *models.User {
	req := c.Request()
	user, ok := req.Context().Value(userContextKey).(*models.User)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}
