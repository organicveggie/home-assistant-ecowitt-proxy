package controller

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/labstack/echo/v4"
)

func New(hassURL, hassAuthToken, webhookID string) *Controller {
	return &Controller{
		hassURL:       hassURL,
		hassAuthToken: hassAuthToken,
		webhookID:     webhookID,
	}
}

type Controller struct {
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

func (c *Controller) MakeEventResponse(status string) EventResponse {
	return EventResponse{
		Status:     status,
		EventCount: c.eventCount.Load(),
		ErrorCount: c.errorCount.Load(),
	}
}

func (c *Controller) HandleEventGet(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, c.MakeEventResponse("OK"))
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
	return ctx.JSON(http.StatusOK, c.MakeEventResponse("OK"))
}

func (c *Controller) HandleHealth(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, struct{ Status string }{Status: "OK"})
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
