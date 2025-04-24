/*
Copyright © 2023-2024 Sean Laurent <o r g a n i c v e g g i e @ Google Mail>

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
	"html/template"

	"hass-ecowitt-proxy/controller"
	"hass-ecowitt-proxy/logging"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	defaultPort = 8181
)

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
	serveCmd.Flags().StringP(flagListenAddress, "l", "", "IP address to listen on (default is to listen on all addresses)")
	viper.BindPFlag(viperListenAddress, serveCmd.Flags().Lookup(flagListenAddress))
	viper.BindEnv(viperListenAddress, "SERVER_ADDRESS", "ECOWITT_PROXY_ADDRESS")

	serveCmd.Flags().IntP(flagListenPort, "p", defaultPort, "TCP port to listen on")
	viper.BindPFlag(viperListenPort, serveCmd.Flags().Lookup(flagListenPort))
	viper.BindEnv(viperListenPort, "SERVER_PORT", "ECOWITT_PROXY_PORT")

	serveCmd.Flags().StringP(flagHassUrl, "u", "", "Base URL for Home Assistant")
	serveCmd.MarkFlagRequired(flagHassUrl)
	viper.BindPFlag(flagHassUrl, serveCmd.Flags().Lookup(flagHassUrl))
	viper.BindEnv(flagHassUrl, "HASS_URL", "ECOWITT_PROXY_HASS_URL")

	serveCmd.Flags().StringP(flagHassAuthToken, "a", "", "Home Assistant auth token")
	serveCmd.MarkFlagRequired(flagHassAuthToken)
	viper.BindPFlag(flagHassAuthToken, serveCmd.Flags().Lookup(flagHassAuthToken))
	viper.BindEnv(flagHassAuthToken, "HASS_AUTH_TOKEN", "ECOWITT_PROXY_HASS_AUTH_TOKEN")

	serveCmd.Flags().StringP(flagHassWebhookId, "w", "", "Home Assistant webhook id")
	serveCmd.MarkFlagRequired(flagHassWebhookId)
	viper.BindPFlag(flagHassWebhookId, serveCmd.Flags().Lookup(flagHassWebhookId))
	viper.BindEnv(flagHassWebhookId, "HASS_WEBHOOK_ID", "ECOWITT_PROXY_HASS_WEBHOOK_ID")
}

func runServeCmd(_ *cobra.Command, _ []string) error {
	logLevelName := viper.GetString(flagLogLevel)
	logLevel, err := logging.LogLevelFromStr(logLevelName)
	if err != nil {
		return fmt.Errorf("error running serve command: %w", err)
	}

	// Setup Zap logging
	logConfig := zap.NewDevelopmentConfig()
	logConfig.Level = zap.NewAtomicLevelAt(logLevel.ToZap())
	logger, err := logConfig.Build()
	if err != nil {
		return fmt.Errorf("failed to create Zap logger: %w", err)
	}
	defer logger.Sync()

	hassURL := viper.GetString(flagHassUrl)
	hassAuthToken := viper.GetString(flagHassAuthToken)
	hassWebhookID := viper.GetString(flagHassWebhookId)

	ctrl := controller.New(hassURL, hassAuthToken, hassWebhookID, logger,
		controller.WithLogLevel(logLevel),
		controller.WithTemplates(template.Must(template.ParseGlob("html/*.html"))))
	defer ctrl.Close()

	serveAddress := viper.GetString(viperListenAddress)
	servePort := viper.GetInt(viperListenPort)
	addr := fmt.Sprintf("%s:%d", serveAddress, servePort)
	return ctrl.Serve(addr)
}
