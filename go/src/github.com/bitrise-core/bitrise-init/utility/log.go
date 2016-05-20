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

// InfoSection ...
func (logger *LoggerModel) InfoSection(args ...interface{}) {
	fmt.Println()
	logger.Logger.Info(colorstring.Blue(args...))
}

// InfofSection ...
func (logger *LoggerModel) InfofSection(format string, args ...interface{}) {
	fmt.Println()
	logger.Logger.Info(colorstring.Bluef(format, args...))
}

// InfofDetails ...
func (logger *LoggerModel) InfofDetails(format string, args ...interface{}) {
	logger.Logger.Infof("  " + fmt.Sprintf(format, args...))
}

// InfofReceipt ...
func (logger *LoggerModel) InfofReceipt(format string, args ...interface{}) {
	logger.Logger.Info(colorstring.Green("  " + fmt.Sprintf(format, args...)))
}
