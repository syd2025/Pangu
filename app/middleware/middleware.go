package middleware

import (
	"context"
	"expvar"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"example.com/myapp/models"
	"example.com/myapp/utils/helper"
	"example.com/myapp/utils/validator"
	"github.com/felixge/httpsnoop"
	"github.com/labstack/echo/v4"
	"github.com/tomasen/realip"
	"golang.org/x/time/rate"
)

type Middleware struct {
	helper      *helper.Helper
	limitConfig *models.LimiterConfig
	models      *models.Models
}

func New(helper *helper.Helper, limitConfig *models.LimiterConfig, models *models.Models) *Middleware {
	return &Middleware{
		helper:      helper,
		limitConfig: limitConfig,
		models:      models,
	}
}

func (m *Middleware) RecoverPanic(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if err := recover(); err != nil {
				c.Response().Writer.Header().Set("Connection", "close")
				m.helper.ServerErrorResponse(c, fmt.Errorf("%s", err))
			}
		}()
		return next(c)
	}
}

func (m *Middleware) LogRequest(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		msg := fmt.Sprintf("%s - %s %s %s",
			c.Request().RemoteAddr,
			c.Request().Proto,
			c.Request().Method,
			c.Request().URL.RequestURI(),
		)
		m.helper.Logger.Info(msg, nil)
		return next(c)
	}
}

func (m *Middleware) RateLimit(next echo.HandlerFunc) echo.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c echo.Context) error {
		if m.limitConfig.Enabled {
			ip := realip.FromRequest(c.Request())
			mu.Lock()
			if _, found := clients[ip]; !found {
				rps := m.limitConfig.Rps
				burst := m.limitConfig.Burst
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(rps), burst),
				}
			}
			clients[ip].lastSeen = time.Now()
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				m.helper.RateLimitExceededResponse(c)
				return nil
			}
			mu.Unlock()
		}
		return next(c)
	}
}

func (m *Middleware) Metrics(next echo.HandlerFunc) echo.HandlerFunc {
	totalRequestsReceived := expvar.NewInt("total_requests_received")
	totalResponseSent := expvar.NewInt("total_response_sent")
	totalProcessingTimeMicroseconds := expvar.NewInt("total_processing_time_μs")
	totalResponseSentByStatus := expvar.NewMap("total_response_sent_by_status")

	return func(c echo.Context) error {
		totalRequestsReceived.Add(1)
		metrics := httpsnoop.CaptureMetrics(c.Echo().Server.Handler, c.Response().Writer, c.Request())
		totalResponseSent.Add(1)
		totalProcessingTimeMicroseconds.Add(metrics.Duration.Microseconds())
		totalResponseSentByStatus.Add(strconv.Itoa(metrics.Code), 1)
		return next(c)
	}
}

func (m *Middleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Writer.Header().Add("Vary", "Authorization")

		// Get existing context and ensure proper propagation
		ctx := c.Request().Context()

		authorizationHeader := c.Request().Header.Get("Authorization")

		if authorizationHeader == "" {
			// Set anonymous user in context
			ctx = context.WithValue(ctx, userContextKey, models.AnonymousUser)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			return m.helper.InvalidAuthenticationTokenResponse(c)

		}

		token := headerParts[1]
		v := validator.New()
		if models.ValidateTokenPlaintext(v, token); !v.Valid() {
			return m.helper.InvalidAuthenticationTokenResponse(c)

		}
		user, err := m.models.Users.GetUserForToken(models.ScopeAuthentication, token)
		if err != nil {
			switch {
			case err == helper.ErrRecordNotFound:
				return m.helper.InvalidAuthenticationTokenResponse(c)
			default:
				return m.helper.ServerErrorResponse(c, err)
			}
		}

		// Set authenticated user
		ctx = context.WithValue(ctx, userContextKey, user)
		c.SetRequest(c.Request().WithContext(ctx))
		return next(c)
	}
}
