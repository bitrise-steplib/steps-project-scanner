package utility

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/bitrise-io/go-utils/colorstring"
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

// Info ...
func (logger *LoggerModel) Info(args ...interface{}) {
	logger.Logger.Info(colorstring.Blue(args...))
}

// Infof ...
func (logger *LoggerModel) Infof(format string, args ...interface{}) {
	logger.Logger.Info(colorstring.Bluef(format, args...))
}

// Warnf ...
func (logger *LoggerModel) Warnf(format string, args ...interface{}) {
	logger.Logger.Info(colorstring.Yellowf(format, args...))
}

// InfoSection ...
func (logger *LoggerModel) InfoSection(args ...interface{}) {
	logger.Logger.Info()
	logger.Logger.Info(colorstring.Blue(args...))
}

// InfofSection ...
func (logger *LoggerModel) InfofSection(format string, args ...interface{}) {
	logger.Logger.Info()
	logger.Logger.Info(colorstring.Bluef(format, args...))
}

// InfofDetails ...
func (logger *LoggerModel) InfofDetails(format string, args ...interface{}) {
	logger.Logger.Infof("  " + fmt.Sprintf(format, args...))
}

// InfoDetails ...
func (logger *LoggerModel) InfoDetails(args ...interface{}) {
	logger.Logger.Info("  " + fmt.Sprint(args...))
}

// InfofReceipt ...
func (logger *LoggerModel) InfofReceipt(format string, args ...interface{}) {
	logger.Logger.Info(colorstring.Green("  " + fmt.Sprintf(format, args...)))
}
