package analytics

import (
	"github.com/bitrise-io/go-utils/log"
)

const stepName = "bitrise-init"

func initData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{}
	}
	data["source"] = "scanner"
	return data
}

// LogError sends analytics log using log.RErrorf by setting the stepID.
func LogError(tag string, data map[string]interface{}, format string, v ...interface{}) {
	log.RErrorf(stepName, tag, initData(data), format, v...)
}

// LogInfo sends analytics log using log.RInfof by setting the stepID.
func LogInfo(tag string, data map[string]interface{}, format string, v ...interface{}) {
	log.RInfof(stepName, tag, initData(data), format, v...)
}

// DetectorErrorData creates analytics data that includes the platform and error
func DetectorErrorData(detector string, err error) map[string]interface{} {
	return map[string]interface{}{
		"detector": detector,
		"error":    err.Error(),
	}
}
