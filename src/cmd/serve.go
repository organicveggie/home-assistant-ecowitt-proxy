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
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServeCmd(cmd, args)
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

func runServeCmd(_ *cobra.Command, _ []string) error {
	ctrl := controller.New(serveOpts.HassURL, serveOpts.HassAuthToken, serveOpts.WebhookID,
		controller.WithTemplates(template.Must(template.ParseGlob("html/*.html"))))
	defer ctrl.Close()

	addr := fmt.Sprintf("%s:%d", serveOpts.Address, serveOpts.Port)
	return ctrl.Serve(addr)
}
