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
package cmd

import (
	"fmt"
	"os"
	"strings"

	"hass-ecowitt-proxy/logging"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hass-ecowitt-proxy",
	Short: "A lightweight proxy from Ecowitt to Home Assistant",
	Long: `A small server application which accepts HTTP messages with weather
data from Ecowitt devices and proxies them to Home Assistant.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, flagConfig, "", "Config file (default is $HOME/.hass-ecowitt-proxy.yaml)")
	rootCmd.PersistentFlags().StringP(flagOutput, "o", "stdout",
		"Output target for log messages. One ofstdout, stderr, or filename. Defaults to stderr. Ignored if loglevel is off.")
	viper.BindPFlag(flagOutput, rootCmd.PersistentFlags().Lookup(flagOutput))
	viper.BindEnv(flagOutput, "ECOWITT_PROXY_OUTPUT")

	rootCmd.PersistentFlags().StringP(flagLogLevel, "l", logging.InfoLevel.String(),
		"Log level. One of: "+strings.Join(logging.LogLevelNames(), ", "))
	viper.BindPFlag(flagLogLevel, rootCmd.PersistentFlags().Lookup(flagLogLevel))
	viper.BindEnv(flagLogLevel, "ECOWITT_PROXY_LOGLEVEL")

	rootCmd.AddCommand(serveCmd)

	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		level := viper.GetString(flagLogLevel)
		if _, err := logging.LogLevelFromStr(level); err != nil {
			return err
		}

		return nil
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".hass-ecowitt-proxy" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".hass-ecowitt-proxy")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
