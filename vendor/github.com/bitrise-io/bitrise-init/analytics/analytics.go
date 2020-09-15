package analytics

import (
	"os"

	"github.com/bitrise-io/go-utils/log"
)

// LogError sends analytics log using log.RErrorf by setting the stepID and data/build_slug.
func LogError(tag string, data map[string]interface{}, format string, v ...interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	data["build_slug"] = os.Getenv("BITRISE_BUILD_SLUG")

	log.RErrorf("bitrise-init", tag, data, format, v...)
}
