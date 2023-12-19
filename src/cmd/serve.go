/*
Copyright © 2023 Sean Laurent <o r g a n i c v e g g i e @ Google Mail>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"fmt"
	"hass-ecowitt-proxy/controller"
	"html/template"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultPort = 8181

	flagListenAddress = "listen_address"
	flagListenPort    = "port"

	flagHassUrl       = "hass_url"
	flagHassAuthToken = "hass_auth_token"
	flagHassWebhookId = "hass_webhook_id"

	viperListenAddress = "listen"
	viperListenPort    = "port"
)

type ServerOptions struct {
	Address string
	Port    int

	HassURL       string
	HassAuthToken string
	WebhookID     string
}

var serveOpts ServerOptions

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Listen for HTTP messages",
	Long: `Start server mode to listen for incoming HTTP messages. Does not
exit until it receives a SIGTERM.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServeCmd(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVarP(&serveOpts.Address, flagListenAddress, "l", "",
		"IP address to listen on (default is to listen on all addresses)")
	viper.BindPFlag(viperListenAddress, serveCmd.Flags().Lookup(flagListenAddress))
	viper.BindEnv(viperListenAddress, "SERVER_ADDRESS", "ECOWITT_PROXY_ADDRESS")

	serveCmd.Flags().IntVarP(&serveOpts.Port, flagListenPort, "p", defaultPort, "TCP port to listen on")
	viper.BindPFlag(viperListenPort, serveCmd.Flags().Lookup(flagListenPort))
	viper.BindEnv(viperListenPort, "SERVER_PORT", "ECOWITT_PROXY_PORT")

	serveCmd.Flags().StringVarP(&serveOpts.HassURL, flagHassUrl, "u", "",
		"Base URL for Home Assistant")
	serveCmd.MarkFlagRequired(flagHassUrl)
	viper.BindPFlag(flagHassUrl, serveCmd.Flags().Lookup(flagHassUrl))
	viper.BindEnv(flagHassUrl, "HASS_URL", "ECOWITT_PROXY_HASS_URL")

	serveCmd.Flags().StringVarP(&serveOpts.HassAuthToken, flagHassAuthToken, "a", "",
		"Home Assistant auth token")
	serveCmd.MarkFlagRequired(flagHassAuthToken)
	viper.BindPFlag(flagHassAuthToken, serveCmd.Flags().Lookup(flagHassAuthToken))
	viper.BindEnv(flagHassAuthToken, "HASS_AUTH_TOKEN", "ECOWITT_PROXY_HASS_AUTH_TOKEN")

	serveCmd.Flags().StringVarP(&serveOpts.WebhookID, flagHassWebhookId, "w", "",
		"Home Assistant webhook id")
	serveCmd.MarkFlagRequired(flagHassWebhookId)
	viper.BindPFlag(flagHassWebhookId, serveCmd.Flags().Lookup(flagHassWebhookId))
	viper.BindEnv(flagHassWebhookId, "HASS_WEBHOOK_ID", "ECOWITT_PROXY_HASS_WEBHOOK_ID")
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func runServeCmd(cmd *cobra.Command, args []string) {
	t := &Template{
		templates: template.Must(template.ParseGlob("html/*.html")),
	}

	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	e.Renderer = t
	e.HTTPErrorHandler = customHTTPErrorHandler

	ctrl := controller.New(serveOpts.HassURL, serveOpts.HassAuthToken, serveOpts.WebhookID)
	defer ctrl.Close()

	e.GET("/event", ctrl.HandleEventGet)
	e.POST("/event", ctrl.HandleEventPost)
	e.GET("/health", ctrl.HandleHealth)

	e.GET("/status", func(c echo.Context) error {
		data := struct {
			Opts       ServerOptions
			EventCount uint32
			ErrorCount uint32
		}{
			Opts:       serveOpts,
			EventCount: ctrl.GetEventCount(),
			ErrorCount: ctrl.GetErrorCount(),
		}
		return c.Render(http.StatusOK, "status", data)
	})

	addr := fmt.Sprintf("%s:%d", serveOpts.Address, serveOpts.Port)
	e.Logger.Fatal(e.Start(addr))
}

func customHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	c.Logger().Error(err)

	data := struct {
		Message string
	}{
		Message: "Internal Server Error",
	}
	c.JSON(code, data)
}
