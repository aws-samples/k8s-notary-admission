package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DefaultEncoding string = "console"
)

var (
	Log    *zap.SugaredLogger
	Config *zap.Config
)

func Build(level string, encoding string) {
	if encoding == "" {
		encoding = DefaultEncoding
	}

	ec := zap.NewProductionEncoderConfig()
	ec.TimeKey = "timestamp"
	ec.EncodeTime = zapcore.ISO8601TimeEncoder

	Config = &zap.Config{
		Level:             zap.NewAtomicLevelAt(ResolveLevel(level)),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         encoding,
		EncoderConfig:    ec,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		InitialFields:    nil,
	}
}

func Start() error {
	logger, err := Config.Build()
	if err != nil {
		return err
	}

	Log = logger.Sugar()
	return nil
}

func ResolveLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zap.DebugLevel
	case "error":
		return zap.ErrorLevel
	case "fatal":
		return zap.FatalLevel
	case "warn":
		return zap.WarnLevel
	default:
		return zap.InfoLevel
	}
}
