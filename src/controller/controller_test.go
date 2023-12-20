package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestHandleHealth(t *testing.T) {
	t.Run("should return 200 OK", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		ctrl := New("http://localhost/ecowitt", "test-token", "test-webhook")
		defer ctrl.Close()

		err := ctrl.HandleHealth(c)
		assert.Nil(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandleEventGet(t *testing.T) {
	t.Run("should return 200 OK", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		ctrl := New("http://localhost/ecowitt", "test-token", "test-webhook")
		defer ctrl.Close()

		err := ctrl.HandleEventGet(c)
		assert.Nil(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		want := EventResponse{Status: "OK"}
		var got EventResponse
		json.Unmarshal(rec.Body.Bytes(), &got)

		assert.Equal(t, want.Status, got.Status)
		assert.Equal(t, want.EventCount, got.EventCount)
		assert.Equal(t, want.ErrorCount, got.ErrorCount)
	})
}

func TestHandleEventPost(t *testing.T) {
	const token = "test-token"
	const webhookId = "test-webhook-id"

	tests := []struct {
		name        string
		statusCode  int
		wantStatus  int
		wantResp    *EventResponse
		wantErrResp *ErrorResponse
	}{
		{
			name:       "Handle Event POST",
			statusCode: http.StatusOK,
			wantStatus: http.StatusOK,
			wantResp:   &EventResponse{Status: "OK", EventCount: 1, ErrorCount: 0},
		},
		{
			name:        "Handle Event POST with error",
			statusCode:  http.StatusInternalServerError,
			wantStatus:  http.StatusInternalServerError,
			wantErrResp: &ErrorResponse{Status: "ERROR", EventCount: 0, ErrorCount: 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Printf("[%s] http.HandlerFunc() return %d\n", test.name, test.statusCode)
				w.WriteHeader(test.statusCode)
				fmt.Fprintf(w, "%d", test.statusCode)
			}))
			defer svr.Close()

			ctrl := New(svr.URL, token, webhookId)
			defer ctrl.Close()

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/event", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := ctrl.HandleEventPost(c)
			assert.Nil(t, err)

			assert.Equal(t, test.wantStatus, rec.Result().StatusCode)

			if test.wantResp != nil {
				var got EventResponse
				json.Unmarshal(rec.Body.Bytes(), &got)
				assert.Equal(t, test.wantResp.Status, got.Status)
				assert.Equal(t, test.wantResp.EventCount, got.EventCount)
				assert.Equal(t, test.wantResp.ErrorCount, got.ErrorCount)
			}

			if test.wantErrResp != nil {
				var got ErrorResponse
				json.Unmarshal(rec.Body.Bytes(), &got)
				assert.Equal(t, test.wantErrResp.Status, got.Status)
				assert.Equal(t, test.wantErrResp.EventCount, got.EventCount)
				assert.Equal(t, test.wantErrResp.ErrorCount, got.ErrorCount)
			}
		})
	}
}

func TestWebhookClient(t *testing.T) {
	const token = "test-token"

	tests := []struct {
		name       string
		statusCode int
		data       url.Values
	}{
		{
			name:       "Basic form data works",
			statusCode: http.StatusOK,
			data: url.Values{
				"k1": {"k1v1", "k1v2"},
				"k2": {"k2v1", "k2v2", "k2v3"},
				"k3": {"k3v1"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotValues url.Values
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.ParseForm()
				gotValues = r.PostForm
				w.WriteHeader(test.statusCode)
				fmt.Fprintf(w, "%d", test.statusCode)
			}))
			defer svr.Close()

			client := NewHassClient(svr.URL, token, test.data, WithOpenClientFn(func() *http.Client {
				return svr.Client()
			}))

			if err := client.PostData(context.Background()); err != nil {
				t.Errorf("unexpected error making Hass Client PostData call: %s", err)
			}
			assert.Equal(t, test.data, gotValues)
		})
	}
}

func TestServe(t *testing.T) {
	tests := []struct {
		name string
	}{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
		})
	}
}
