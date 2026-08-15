package logger_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/pkg/logger"
)

func TestPrettyHandlerFormatting(t *testing.T) {
	buf := &bytes.Buffer{}
	l := logger.Setup(logger.Options{
		Level:   "debug",
		Format:  "pretty",
		NoColor: true,
		Writer:  buf,
	})

	l.Info("test message", slog.String("key", "val"))

	out := buf.String()
	if !strings.Contains(out, "[ INFO]") {
		t.Errorf("expected level tag '[ INFO]', got: %s", out)
	}
	if !strings.Contains(out, "test message") {
		t.Errorf("expected message 'test message', got: %s", out)
	}
	if !strings.Contains(out, "key=val") {
		t.Errorf("expected attribute 'key=val', got: %s", out)
	}
}

func TestJSONHandlerFormatting(t *testing.T) {
	buf := &bytes.Buffer{}
	l := logger.Setup(logger.Options{
		Level:  "info",
		Format: "json",
		Writer: buf,
	})

	l.Info("json log", slog.Int("count", 42))

	out := buf.String()
	if !strings.Contains(out, `"msg":"json log"`) {
		t.Errorf("expected json msg attribute, got: %s", out)
	}
	if !strings.Contains(out, `"count":42`) {
		t.Errorf("expected json count attribute, got: %s", out)
	}
}

func TestEchoPanicRecovery(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Setup(logger.Options{
		Level:   "debug",
		Format:  "pretty",
		NoColor: true,
		Writer:  buf,
	})

	e := echo.New()
	e.Use(logger.EchoPanicRecovery())

	e.GET("/panic", func(c echo.Context) error {
		panic("something went wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	out := buf.String()
	if !strings.Contains(out, "HTTP handler panic recovered") {
		t.Errorf("expected panic log message, got: %s", out)
	}
	if !strings.Contains(out, "something went wrong") {
		t.Errorf("expected panic error text in log, got: %s", out)
	}
}

func TestEchoHTTPErrorHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Setup(logger.Options{
		Level:   "debug",
		Format:  "pretty",
		NoColor: true,
		Writer:  buf,
	})

	e := echo.New()
	e.HTTPErrorHandler = logger.EchoHTTPErrorHandler

	e.GET("/error", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusInternalServerError, "db connection error")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	out := buf.String()
	if !strings.Contains(out, "HTTP server error") {
		t.Errorf("expected error log message, got: %s", out)
	}
	if !strings.Contains(out, "db connection error") {
		t.Errorf("expected error text in log, got: %s", out)
	}
}

func TestEchoNormalRequestNotLogged(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Setup(logger.Options{
		Level:   "debug",
		Format:  "pretty",
		NoColor: true,
		Writer:  buf,
	})

	e := echo.New()
	e.Use(logger.EchoPanicRecovery())
	e.Use(logger.EchoErrorLogger())
	e.HTTPErrorHandler = logger.EchoHTTPErrorHandler

	e.GET("/ok", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	out := buf.String()
	if strings.Contains(out, "/ok") {
		t.Errorf("expected NO request log for normal 200 response, but got: %s", out)
	}
}

func TestEchoErrorLogger_Warn4xx(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Setup(logger.Options{
		Level:   "debug",
		Format:  "pretty",
		NoColor: true,
		Writer:  buf,
	})

	e := echo.New()
	e.Use(logger.EchoErrorLogger())

	e.GET("/notfound", func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	})

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	out := buf.String()
	if !strings.Contains(out, "[ WARN]") {
		t.Errorf("expected WARN log for 404, got: %s", out)
	}
	if !strings.Contains(out, "HTTP request warning") {
		t.Errorf("expected 'HTTP request warning' msg, got: %s", out)
	}
	if !strings.Contains(out, "uri=/notfound") {
		t.Errorf("expected uri attribute in log, got: %s", out)
	}
	if !strings.Contains(out, "status=404") {
		t.Errorf("expected status=404 in log, got: %s", out)
	}
	if !strings.Contains(out, "method=GET") {
		t.Errorf("expected method=GET in log, got: %s", out)
	}
}

func TestEchoErrorLogger_Error5xx(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Setup(logger.Options{
		Level:   "debug",
		Format:  "pretty",
		NoColor: true,
		Writer:  buf,
	})

	e := echo.New()
	e.Use(logger.EchoErrorLogger())

	e.GET("/fail", func(c echo.Context) error {
		return errors.New("upstream service failed")
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, "[ERROR]") {
		t.Errorf("expected ERROR log for 500 error, got: %s", out)
	}
	if !strings.Contains(out, "HTTP request error") {
		t.Errorf("expected 'HTTP request error' msg, got: %s", out)
	}
	if !strings.Contains(out, "uri=/fail") {
		t.Errorf("expected uri attribute in log, got: %s", out)
	}
	if !strings.Contains(out, "status=500") {
		t.Errorf("expected status=500 in log, got: %s", out)
	}
	if !strings.Contains(out, "upstream service failed") {
		t.Errorf("expected error string in log, got: %s", out)
	}
}

func TestErrorAttributeFormatting(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Setup(logger.Options{
		Level:   "info",
		Format:  "pretty",
		NoColor: true,
		Writer:  buf,
	})

	errExample := errors.New("custom error object")
	slog.Error("failed operation", slog.Any("error", errExample))

	out := buf.String()
	if !strings.Contains(out, "error=custom error object") {
		t.Errorf("expected formatted error string, got: %s", out)
	}
}

func TestEchoErrorLogger_HandlerErrorContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Setup(logger.Options{
		Level:   "debug",
		Format:  "pretty",
		NoColor: true,
		Writer:  buf,
	})

	e := echo.New()
	e.Use(logger.EchoErrorLogger())

	e.GET("/custom-error", func(c echo.Context) error {
		c.Set("handler_error", "resource missing from catalog")
		return c.JSON(http.StatusNotFound, map[string]string{"error": "resource missing from catalog"})
	})

	req := httptest.NewRequest(http.MethodGet, "/custom-error", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	out := buf.String()
	if !strings.Contains(out, "[ WARN]") {
		t.Errorf("expected WARN log for 404, got: %s", out)
	}
	if !strings.Contains(out, "error=\"resource missing from catalog\"") && !strings.Contains(out, "error=resource missing from catalog") {
		t.Errorf("expected handler_error in log, got: %s", out)
	}
}

