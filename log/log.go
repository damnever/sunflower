package log

import (
	"fmt"
	"os"

	isatty "github.com/mattn/go-isatty"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	encoderConf = zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
	logLevel      = zap.InfoLevel
	logTraceLevel = zap.PanicLevel
)

func init() {
	// Assume it's in development mode if terminal detected.
	fd := os.Stdout.Fd()
	if isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd) {
		encoderConf.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logLevel = zap.DebugLevel
		logTraceLevel = zap.ErrorLevel
	}
}

// New creates a logger with a name.
func New(format string, args ...interface{}) *zap.SugaredLogger {
	name := fmt.Sprintf(format, args...)
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConf), os.Stdout, logLevel)
	logger := zap.New(core).WithOptions(zap.AddStacktrace(logTraceLevel))
	return logger.Sugar().Named(name)
}
