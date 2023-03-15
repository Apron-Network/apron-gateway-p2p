package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"apron.network/gateway-p2p/internal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.Logger

	defaultEncoderConfig = zapcore.EncoderConfig{
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

	defaultLogConfig = LogConfig{
		BaseDir:    "/var/log/",
		Level:      "Warn",
		MaxSizeMb:  10,
		MaxBackups: 10,
		MaxAgeDay:  7,
	}
)

func InitLogger(c LogConfig, name string) {
	mergedLogConfig := defaultLogConfig
	mergedLogConfig = *internal.MergeTwoStruct(&mergedLogConfig, &c).(*LogConfig)

	logFileName := filepath.Join(mergedLogConfig.BaseDir, fmt.Sprintf("%s.log", name))
	logWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFileName,
		MaxSize:    mergedLogConfig.MaxSizeMb,
		MaxAge:     mergedLogConfig.MaxAgeDay,
		MaxBackups: mergedLogConfig.MaxBackups,
		LocalTime:  false,
		Compress:   true,
	})

	encoderConfig := defaultEncoderConfig
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	var l zapcore.Level
	if err := l.Set(c.Level); err != nil {
		panic(err)
	}

	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(os.Stdout),
		logWriter,
	), zap.NewAtomicLevelAt(l))
	logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel), zap.Fields(zap.String("logger_name", name)))
}

// GetLogger build zap logger object and return to invoker.
// TODO: change to channel based logger system ()
func GetLogger() *zap.Logger {
	return logger
}
