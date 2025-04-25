package helper

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
	ErrDuplicateEmail = errors.New("duplicate email")
)

func (helper *Helper) LogError(r *http.Request, err error) {
	helper.Logger.Error(err, map[string]string{
		"request_method": r.Method,
		"request_url":    r.URL.String(),
	})
}

func (helper *Helper) ErrorResponse(c echo.Context, status int, message string) error {
	data := map[string]interface{}{"message": message}
	resp := helper.NewResponse(-1, data)
	return helper.WriteJSON(c.Response().Writer, status, resp, nil)
	// err := helper.WriteJSON(c.Response().Writer, status, resp, nil)
	// if err != nil {
	// 	helper.LogError(c.Request(), err)
	// 	c.Response().Writer.WriteHeader(http.StatusInternalServerError)
	// }
}

func (helper *Helper) ForbiddenResponse(c echo.Context) {
	message := "you do not have permission to access this resource"
	helper.ErrorResponse(c, http.StatusForbidden, message)
}

func (helper *Helper) ServerErrorResponse(c echo.Context, err error) error {
	helper.LogError(c.Request(), err)
	message := "the server encountered a problem and could not process your request"
	return helper.ErrorResponse(c, http.StatusInternalServerError, message)
}

func (helper *Helper) NotFoundResponse(c echo.Context) error {
	message := "the requested resource could not be found"
	return helper.ErrorResponse(c, http.StatusNotFound, message)
}

func (helper *Helper) MethodNotAllowedResponse(c echo.Context) error {
	message := "the request method is not allowed"
	return helper.ErrorResponse(c, http.StatusMethodNotAllowed, message)
}

func (helper *Helper) BadRequestResponse(c echo.Context, err error) error {
	return helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
}

func (helper *Helper) FailedValidationResponse(c echo.Context, errors map[string]string) error {
	var strBuilder strings.Builder
	for key, value := range errors {
		strBuilder.WriteString(fmt.Sprintf("%s: %s, ", key, value))
	}
	result := strBuilder.String()
	if len(result) > 0 {
		result = result[:len(result)-2]
		result += "."
	}
	return helper.ErrorResponse(c, http.StatusUnprocessableEntity, result)
}

func (helper *Helper) EditConflictResponse(c echo.Context) {
	message := "unable to update the resource due to an edit conflict"
	helper.ErrorResponse(c, http.StatusConflict, message)
}

func (helper *Helper) RateLimitExceededResponse(c echo.Context) {
	message := "rate limit exceeded"
	helper.ErrorResponse(c, http.StatusTooManyRequests, message)
}

func (helper *Helper) InvalidCredentialsResponse(c echo.Context) error {
	message := "invalid authentication credentials"
	return helper.ErrorResponse(c, http.StatusUnauthorized, message)
}

func (helper *Helper) InvalidAuthenticationTokenResponse(c echo.Context) error {
	message := "invalid or expired authentication token"
	return helper.ErrorResponse(c, http.StatusUnauthorized, message)
}

func (helper *Helper) AuthenticationRequiredResponse(c echo.Context) {
	message := "authentication required to access this resource"
	helper.ErrorResponse(c, http.StatusUnauthorized, message)
}

func (helper *Helper) NotPermittedResponse(c echo.Context) {
	message := "you do not have permission to perform this action"
	helper.ErrorResponse(c, http.StatusForbidden, message)
}
