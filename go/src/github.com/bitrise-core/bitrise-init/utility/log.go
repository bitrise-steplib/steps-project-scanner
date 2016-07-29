package utility

import (
	"fmt"

	"github.com/Sirupsen/logrus"
)

// LoggerModel ...
type LoggerModel struct {
	Logger *logrus.Logger
}

// NewLogger ...
func NewLogger() LoggerModel {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{
		FullTimestamp:   true,
		ForceColors:     true,
		TimestampFormat: "15:04:05",
	}
	return LoggerModel{
		Logger: logger,
	}
}

// Fail ...
func (logger *LoggerModel) Fail(format string, v ...interface{}) {
	errorMsg := fmt.Sprintf(format, v...)
	logger.Logger.Fatalf("\x1b[31;1m%s\x1b[0m", errorMsg)
}

// Error ...
func (logger *LoggerModel) Error(format string, v ...interface{}) {
	errorMsg := fmt.Sprintf(format, v...)
	logger.Logger.Errorf("\x1b[31;1m%s\x1b[0m", errorMsg)
}

// Warn ...
func (logger *LoggerModel) Warn(format string, v ...interface{}) {
	errorMsg := fmt.Sprintf(format, v...)
	logger.Logger.Warnf("\x1b[33;1m%s\x1b[0m", errorMsg)
}

// Info ...
func (logger *LoggerModel) Info(format string, v ...interface{}) {
	logger.Logger.Info("")
	errorMsg := fmt.Sprintf(format, v...)
	logger.Logger.Infof("\x1b[34;1m%s\x1b[0m", errorMsg)
}

// Details ...
func (logger *LoggerModel) Details(format string, v ...interface{}) {
	errorMsg := fmt.Sprintf(format, v...)
	logger.Logger.Infof("  %s", errorMsg)
}

// Done ...
func (logger *LoggerModel) Done(format string, v ...interface{}) {
	errorMsg := fmt.Sprintf(format, v...)
	logger.Logger.Infof("\x1b[32;1m%s\x1b[0m", errorMsg)
}
