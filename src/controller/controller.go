package controller

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

func New(url string, authToken string, webhookID string, opts ...Option) *Controller {
	c := &Controller{
		hassURL:       url,
		hassAuthToken: authToken,
		webhookID:     webhookID,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type Option func(*Controller)

func WithTemplates(templates *template.Template) Option {
	return func(c *Controller) {
		c.templates = templates
	}
}

type Controller struct {
	templates *template.Template

	hassURL       string
	hassAuthToken string
	webhookID     string

	eventCount atomic.Uint32
	errorCount atomic.Uint32
}

func (c *Controller) Close() {}

func (c *Controller) GetEventCount() uint32 {
	return c.eventCount.Load()
}

func (c *Controller) GetErrorCount() uint32 {
	return c.errorCount.Load()
}

func (c *Controller) makeEventResponse(status string) EventResponse {
	return EventResponse{
		Status:     status,
		EventCount: c.eventCount.Load(),
		ErrorCount: c.errorCount.Load(),
	}
}

func (c *Controller) HandleEventGet(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, c.makeEventResponse("OK"))
}

func (c *Controller) HandleEventPost(ctx echo.Context) error {
	values, err := ctx.FormParams()
	if err != nil {
		c.errorCount.Add(1)
		ctx.Logger().Errorf("Error retrieving form parameters: %w", err)
		return ctx.JSON(http.StatusInternalServerError,
			c.NewErrorResponse("Error retrieving form parameters", err))
	}

	forwardUrl := fmt.Sprintf("%s/api/webhook/%s", c.hassURL, c.webhookID)
	ctx.Logger().Infof("Forwarding Ecowitt event data to %q: %v", forwardUrl, values)

	haClient := NewHassClient(forwardUrl, c.hassAuthToken, values)
	if err := haClient.PostData(ctx.Request().Context()); err != nil {
		c.errorCount.Add(1)
		ctx.Logger().Errorf("Error posting event data to Home Assistant: %w", err)
		return ctx.JSON(http.StatusInternalServerError, c.NewErrorResponse(forwardUrl, err))
	}

	c.eventCount.Add(1)
	return ctx.JSON(http.StatusOK, c.makeEventResponse("OK"))
}

func (c *Controller) HandleHealth(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, struct{ Status string }{Status: "OK"})
}

func (c *Controller) HandleStatus(ctx echo.Context, addr string) error {
	data := struct {
		Address string

		HassURL       string
		HassAuthToken string
		WebhookID     string

		EventCount uint32
		ErrorCount uint32
	}{
		Address:       addr,
		HassURL:       c.hassURL,
		HassAuthToken: c.hassAuthToken,
		WebhookID:     c.webhookID,
		EventCount:    c.GetEventCount(),
		ErrorCount:    c.GetErrorCount(),
	}

	if c.templates != nil {
		return ctx.Render(http.StatusOK, "statuscobra", data)
	}

	return ctx.JSON(http.StatusOK, data)
}

func (c *Controller) NewErrorResponse(msg string, err error) ErrorResponse {
	return ErrorResponse{
		Status:     "ERROR",
		Message:    msg,
		Error:      fmt.Sprint(err),
		EventCount: c.eventCount.Load(),
		ErrorCount: c.errorCount.Load(),
	}
}

func (c *Controller) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return c.templates.ExecuteTemplate(w, name, data)
}

func (c *Controller) Serve(addr string) error {
	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	e.Renderer = c
	e.HTTPErrorHandler = customHTTPErrorHandler

	e.GET("/event", c.HandleEventGet)
	e.POST("/event", c.HandleEventPost)
	e.GET("/health", c.HandleHealth)

	e.GET("/status", func(ctx echo.Context) error {
		return c.HandleStatus(ctx, addr)
	})

	return e.Start(addr)
}

func customHTTPErrorHandler(err error, ctx echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	ctx.Logger().Error(err)
	ctx.JSON(code, struct{ Message string }{Message: "Internal Server Error"})
}

type EventResponse struct {
	Status     string
	EventCount uint32
	ErrorCount uint32
}

type ErrorResponse struct {
	Status     string
	Message    string
	Error      string
	EventCount uint32
	ErrorCount uint32
}
