package main

import (
	"flag"
	"fmt"
	"hass-ecowitt-proxy/controller"
	"html/template"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type OsEnvOption int

const (
	EnvServerAddress OsEnvOption = iota
	EnvServerPort
	EnvHassURL
	EnvHassAuthToken
	EnvHassWebhookId
)

func (o OsEnvOption) String() string {
	return getEnvVarNames()[o]
}

func (o OsEnvOption) Index() int {
	return int(o)
}

func getEnvVarNames() []string {
	return []string{"SERVER_ADDRESS", "SERVER_PORT", "HASS_URL", "HASS_AUTH_TOKEN", "HASS_WEBHOOK_ID"}
}

const (
	FlagHassUrl       = "hass_url"
	FlagHassAuthToken = "hass_auth_token"
	FlagHassWebhookId = "hass_webhook_id"
)

type ServerOptions struct {
	Address string
	Port    int

	HassURL       string
	HassAuthToken string
	WebhookID     string
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func makeEnvVarMap(environ []string) map[string]string {
	serverEnvNames := getEnvVarNames()

	var m = make(map[string]string)
	for _, name := range serverEnvNames {
		m[name] = ""
	}

	for _, ev := range os.Environ() {
		pair := strings.SplitN(ev, "=", 2)
		if slices.Contains(serverEnvNames, pair[0]) {
			m[pair[0]] = pair[1]
		}
	}
	return m
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

func getEnvOrOpt(envName, optValue string) string {
	if v := os.Getenv(envName); len(v) > 0 {
		return v
	}
	if len(optValue) > 0 {
		return optValue
	}
	return ""
}

func getOptOrEnv(optValue, envName string) string {
	if len(optValue) > 0 {
		return optValue
	}

	if v := os.Getenv(envName); len(v) > 0 {
		return v
	}

	return ""
}

func main() {
	// Setup command line flags
	var printHelp bool
	flag.BoolVar(&printHelp, "help", false, "Show all flags and their default values")
	flag.BoolVar(&printHelp, "?", false, "Show all flags and their default values")

	opts := ServerOptions{}
	flag.StringVar(&opts.Address, "address", "", "IP address to listen on (default is to listen on all addresses)")
	flag.IntVar(&opts.Port, "port", 8181, "TCP port to listen on")
	flag.StringVar(&opts.HassURL, FlagHassUrl, "",
		fmt.Sprintf("Base URL for Home Assistant (defaults to env var %s)", EnvHassURL))
	flag.StringVar(&opts.HassAuthToken, FlagHassAuthToken, "",
		fmt.Sprintf("Home Assistant auth token (defaults to env var %s)", EnvHassAuthToken))
	flag.StringVar(&opts.WebhookID, FlagHassWebhookId, "",
		fmt.Sprintf("Home Assistant webhook id (defaults to env var %s)", EnvHassWebhookId))
	flag.Parse()

	opts.Address = getEnvOrOpt(EnvServerAddress.String(), opts.Address)
	if portStr := os.Getenv(EnvServerPort.String()); len(portStr) > 0 {
		var err error
		if opts.Port, err = strconv.Atoi(portStr); err != nil {
			fmt.Printf("Error parsing server port %q: %v", portStr, err)
			return
		}
	}

	opts.HassURL = getOptOrEnv(opts.HassURL, EnvHassURL.String())
	opts.HassAuthToken = getOptOrEnv(opts.HassAuthToken, EnvHassAuthToken.String())
	opts.WebhookID = getOptOrEnv(opts.WebhookID, EnvHassWebhookId.String())

	missingArgs := len(opts.HassURL) == 0 || len(opts.HassAuthToken) == 0 || len(opts.WebhookID) == 0
	if len(opts.HassURL) == 0 {
		fmt.Fprintf(os.Stderr, "Missing required flag -%s (%s)\n", FlagHassUrl, EnvHassURL.String())
	}
	if len(opts.HassAuthToken) == 0 {
		fmt.Fprintf(os.Stderr, "Missing required flag -%s (%s)\n", FlagHassAuthToken, EnvHassAuthToken.String())
	}
	if len(opts.WebhookID) == 0 {
		fmt.Fprintf(os.Stderr, "Missing required flag -%s (%s)\n", FlagHassWebhookId, EnvHassWebhookId.String())
	}
	if missingArgs {
		fmt.Fprintln(os.Stderr)
	}

	if printHelp || missingArgs {
		fmt.Fprintln(os.Stderr, "Usage: server [OPTIONS]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "options:")
		flag.PrintDefaults()
		return
	}

	t := &Template{
		templates: template.Must(template.ParseGlob("html/*.html")),
	}

	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	e.Renderer = t
	e.HTTPErrorHandler = customHTTPErrorHandler

	ctrl := controller.New(opts.HassURL, opts.HassAuthToken, opts.WebhookID)
	defer ctrl.Close()

	e.GET("/event", ctrl.HandleEventGet)
	e.POST("/event", ctrl.HandleEventPost)
	e.GET("/health", ctrl.HandleHealth)

	e.GET("/status", func(c echo.Context) error {
		data := struct {
			Opts       ServerOptions
			Env        map[string]string
			EventCount uint32
			ErrorCount uint32
		}{
			Opts:       opts,
			Env:        makeEnvVarMap(os.Environ()),
			EventCount: ctrl.GetEventCount(),
			ErrorCount: ctrl.GetErrorCount(),
		}
		return c.Render(http.StatusOK, "status", data)
	})

	addr := fmt.Sprintf("%s:%d", opts.Address, opts.Port)
	e.Logger.Fatal(e.Start(addr))
}
