/*
Copyright © 2023-2024 Sean Laurent <o r g a n i c v e g g i e @ Google Mail>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package controller

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sync/atomic"

	"hass-ecowitt-proxy/logging"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

func New(url string, authToken string, webhookID string, logger *zap.Logger, opts ...Option) *Controller {
	c := &Controller{
		echoSrv:       echo.New(),
		logger:        logger.Sugar(),
		logLevel:      logging.InfoLevel,
		hassURL:       url,
		hassAuthToken: authToken,
		webhookID:     webhookID,
	}

	for _, opt := range opts {
		opt(c)
	}

	// Setup request logging
	c.echoSrv.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info("request",
				zap.String("URI", v.URI),
				zap.Int("status", v.Status),
				zap.String("host", v.Host),
				zap.String("protocol", v.Protocol),
				zap.String("method", v.Method),
			)

			return nil
		},
	}))
	c.logger.Info("Request logging middleware for Echo enabled.")

	c.echoSrv.Logger.SetLevel(c.logLevel.ToGommon())
	c.echoSrv.Renderer = c
	c.echoSrv.HTTPErrorHandler = customHTTPErrorHandler

	return c
}

type Option func(*Controller)

func WithTemplates(templates *template.Template) Option {
	return func(c *Controller) {
		c.templates = templates
	}
}

func WithEchoServer(echoSrv *echo.Echo) Option {
	return func(c *Controller) {
		c.echoSrv = echoSrv
	}
}

func WithLogLevel(level logging.LogLevel) Option {
	return func(c *Controller) {
		c.logLevel = level
	}
}

type Controller struct {
	echoSrv   *echo.Echo
	templates *template.Template

	logLevel logging.LogLevel
	logger   *zap.SugaredLogger

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
		ctx.Logger().Errorf("Error retrieving form parameters: %s", err)
		return ctx.JSON(http.StatusInternalServerError,
			c.NewErrorResponse("Error retrieving form parameters", err))
	}

	forwardUrl := fmt.Sprintf("%s/api/webhook/%s", c.hassURL, c.webhookID)
	ctx.Logger().Infof("Forwarding Ecowitt event data to %s", forwardUrl)
	ctx.Logger().Debugf("Ecowitt event data: %v", values)

	haClient := NewHassClient(forwardUrl, c.hassAuthToken, values)
	if err := haClient.PostData(ctx.Request().Context()); err != nil {
		c.errorCount.Add(1)
		ctx.Logger().Errorf("Error posting event data to Home Assistant: %s", err)
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

func (c *Controller) Render(w io.Writer, name string, data interface{}, ctx echo.Context) error {
	return c.templates.ExecuteTemplate(w, name, data)
}

func (c *Controller) Serve(addr string) error {
	c.echoSrv.GET("/event", c.HandleEventGet)
	c.echoSrv.POST("/event", c.HandleEventPost)
	c.echoSrv.GET("/health", c.HandleHealth)

	c.echoSrv.GET("/status", func(ctx echo.Context) error {
		return c.HandleStatus(ctx, addr)
	})

	return c.echoSrv.Start(addr)
}

func customHTTPErrorHandler(err error, ctx echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	if code == http.StatusNotFound {
		ctx.Logger().Errorf("%s: %q", err, ctx.Request().URL)
	} else {
		ctx.Logger().Error(err)
	}
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
