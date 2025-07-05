package logger

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
	once   sync.Once
)

// GetLogger returns zap.Logger instance, but using singleton pattern creates only one reusable instace
// development config by default
func GetLogger() *zap.Logger {
	once.Do(func() {
		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

		var err error
		logger, err = config.Build()
		if err != nil {
			panic("failed logger setup : " + err.Error())
		}

	})
	return logger
}
