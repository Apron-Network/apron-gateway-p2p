package logger

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger
var err error

var once sync.Once

const defaultLogConfig = ""

func GetLogger() *zap.Logger {
	once.Do(func() {
		logEncoderConfig := zapcore.EncoderConfig{
			MessageKey:     "msg",
			LevelKey:       "level",
			NameKey:        "name",
			TimeKey:        "ts",
			CallerKey:      "caller",
			FunctionKey:    "func",
			StacktraceKey:  "stacktrace",
			LineEnding:     "\n",
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		loggerConfig := zap.NewProductionConfig()
		loggerConfig.EncoderConfig = logEncoderConfig
		logger = zap.Must(loggerConfig.Build())
	})

	return logger
}
