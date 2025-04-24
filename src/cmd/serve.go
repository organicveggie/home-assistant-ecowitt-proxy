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
	"strings"

	"hass-ecowitt-proxy/controller"
	"hass-ecowitt-proxy/logging"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	defaultPort = 8181

	envListenAddress = "ECOWITT_PROXY_ADDRESS"
	envListenPort    = "ECOWITT_PROXY_PORT"

	envHassURL       = "ECOWITT_PROXY_HASS_URL"
	envHassAuthToken = "ECOWITT_PROXY_HASS_AUTH_TOKEN"
	envHassWebhookID = "ECOWITT_PROXY_HASS_WEBHOOK_ID"
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
	serveCmd.Flags().StringP(flagListenAddress, "a", "", fmt.Sprintf("IP address to listen on "+
		"(%s) (default is to listen on all addresses)", envListenAddress))
	viper.BindPFlag(viperListenAddress, serveCmd.Flags().Lookup(flagListenAddress))
	viper.BindEnv(viperListenAddress, "SERVER_ADDRESS", envListenAddress)

	serveCmd.Flags().IntP(flagListenPort, "p", defaultPort, fmt.Sprintf("TCP port to listen on. "+
		"(%s)", envListenPort))
	viper.BindPFlag(viperListenPort, serveCmd.Flags().Lookup(flagListenPort))
	viper.BindEnv(viperListenPort, "SERVER_PORT", envListenPort)

	serveCmd.Flags().StringP(flagHassUrl, "u", "", fmt.Sprintf("Base URL for Home Assistant. (%s)", envHassURL))
	viper.BindPFlag(flagHassUrl, serveCmd.Flags().Lookup(flagHassUrl))
	viper.BindEnv(flagHassUrl, "HASS_URL", envHassURL)

	serveCmd.Flags().StringP(flagHassAuthToken, "t", "", fmt.Sprintf("Home Assistant auth token. "+
		"(%s)", envHassAuthToken))
	viper.BindPFlag(flagHassAuthToken, serveCmd.Flags().Lookup(flagHassAuthToken))
	viper.BindEnv(flagHassAuthToken, "HASS_AUTH_TOKEN", envHassAuthToken)

	serveCmd.Flags().StringP(flagHassWebhookId, "w", "", fmt.Sprintf("Home Assistant webhook id. "+
		"(%s)", envHassWebhookID))
	viper.BindPFlag(flagHassWebhookId, serveCmd.Flags().Lookup(flagHassWebhookId))
	viper.BindEnv(flagHassWebhookId, "HASS_WEBHOOK_ID", envHassWebhookID)

	serveCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		hassURL := viper.GetString(flagHassUrl)
		hassAuthToken := viper.GetString(flagHassAuthToken)
		hassWebhookID := viper.GetString(flagHassWebhookId)

		missingOptions := []string{}
		if hassURL == "" {
			missingOptions = append(missingOptions, flagHassUrl)
		}
		if hassAuthToken == "" {
			missingOptions = append(missingOptions, flagHassAuthToken)
		}
		if hassWebhookID == "" {
			missingOptions = append(missingOptions, flagHassWebhookId)
		}

		if len(missingOptions) > 0 {
			return fmt.Errorf("missing required config options: %s", strings.Join(missingOptions, ", "))
		}

		return nil
	}
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
	zap.RedirectStdLog(logger)

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
