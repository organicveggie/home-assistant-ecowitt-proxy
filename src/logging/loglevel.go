package logging

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	gl "github.com/labstack/gommon/log"
	"go.uber.org/zap/zapcore"
)

type LogLevel uint8

const (
	OffLevel LogLevel = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	InvalidLogLevel
)

var levelNames = map[LogLevel]string{
	OffLevel:   "OFF",
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
}

func (l LogLevel) String() string {
	return levelNames[l]
}

func (l LogLevel) ToUint() uint8 {
	return uint8(l)
}

func (l LogLevel) ToGommon() gl.Lvl {
	switch l {
	case DebugLevel:
		return gl.DEBUG
	case InfoLevel:
		return gl.INFO
	case WarnLevel:
		return gl.WARN
	case ErrorLevel:
		return gl.ERROR
	default:
		return gl.OFF
	}
}

func (l LogLevel) ToZap() zapcore.Level {
	switch l {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	default:
		return zapcore.DPanicLevel
	}
}

func LogLevelNames() []string {
	return slices.Collect(maps.Values(levelNames))
}

func LogLevelFromStr(name string) (LogLevel, error) {
	switch strings.ToUpper(name) {
	case "DEBUG":
		return DebugLevel, nil
	case "INFO":
		return InfoLevel, nil
	case "WARN":
		return WarnLevel, nil
	case "ERROR":
		return ErrorLevel, nil
	case "OFF":
		return OffLevel, nil
	default:
		return InvalidLogLevel, fmt.Errorf("invalid log level %q", name)
	}
}
