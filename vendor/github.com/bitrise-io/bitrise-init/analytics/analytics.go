package analytics

import (
	"github.com/bitrise-io/go-utils/log"
)

// LogError sends analytics log using log.RErrorf by setting the stepID.
func LogError(tag string, data map[string]interface{}, format string, v ...interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	data["source"] = "scanner"
	log.RErrorf("bitrise-init", tag, data, format, v...)
}
