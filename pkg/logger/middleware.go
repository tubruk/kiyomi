package logger

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/labstack/echo/v4"
)

const errorLoggedContextKey = "_error_logged"

// EchoErrorLogger returns an Echo middleware that intercepts HTTP responses and logs
// failed requests (status >= 400 or err != nil) via slog.Error or slog.Warn.
// Requests with status < 400 and err == nil are not logged.
// If an error was already logged for this request, it avoids duplicate logging.
func EchoErrorLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start)

			status := c.Response().Status
			if err != nil {
				var he *echo.HTTPError
				if errors.As(err, &he) {
					status = he.Code
				} else if status < 400 {
					status = http.StatusInternalServerError
				}
			}

			if (status >= 400 || err != nil) && c.Get(errorLoggedContextKey) != true {
				c.Set(errorLoggedContextKey, true)
				req := c.Request()
				uri := req.RequestURI
				if uri == "" && req.URL != nil {
					uri = req.URL.String()
				}

				attrs := []any{
					slog.String("method", req.Method),
					slog.String("uri", uri),
					slog.Int("status", status),
					slog.Duration("latency", latency),
					slog.String("remote_ip", c.RealIP()),
				}
				errMsg := ""
				if err != nil {
					errMsg = err.Error()
				} else if val := c.Get("handler_error"); val != nil {
					errMsg = fmt.Sprintf("%v", val)
				}
				if errMsg != "" {
					attrs = append(attrs, slog.String("error", errMsg))
				}

				if status >= 500 {
					slog.Error("HTTP request error", attrs...)
				} else {
					slog.Warn("HTTP request warning", attrs...)
				}
			}

			return err
		}
	}
}

// EchoPanicRecovery returns an Echo middleware that catches panics in HTTP handlers,
// logs panic details and stack traces via slog.Error, and responds with a 500 error.
func EchoPanicRecovery() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}
					stack := debug.Stack()

					slog.Error("HTTP handler panic recovered",
						slog.String("method", c.Request().Method),
						slog.String("uri", c.Request().RequestURI),
						slog.String("remote_ip", c.RealIP()),
						slog.String("error", err.Error()),
						slog.String("stack", string(stack)),
					)

					c.Set(errorLoggedContextKey, true)
					c.Error(echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error"))
				}
			}()
			return next(c)
		}
	}
}

// EchoHTTPErrorHandler is a custom HTTP error handler for Echo that logs server-side (5xx)
// errors using structured slog.Error logging if not already logged.
func EchoHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	var he *echo.HTTPError
	if errors.As(err, &he) {
		code = he.Code
	}

	if code >= 500 && c.Get(errorLoggedContextKey) != true {
		c.Set(errorLoggedContextKey, true)
		slog.Error("HTTP server error",
			slog.String("method", c.Request().Method),
			slog.String("uri", c.Request().RequestURI),
			slog.String("remote_ip", c.RealIP()),
			slog.Int("status", code),
			slog.String("error", err.Error()),
		)
	}

	c.Echo().DefaultHTTPErrorHandler(err, c)
}
