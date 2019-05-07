package utility

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
)

// ConvertPathsToUniqueFileNames returns a map whose values are the input array elements
// keys are a sha256 hash of input strings as a base file name with the original file extension appened
func ConvertPathsToUniqueFileNames(absoluteIconPaths []string, basepath string) (models.Icons, error) {
	iconIDToPath := models.Icons{}
	for _, appIconPath := range absoluteIconPaths {
		relativePath, err := filepath.Rel(basepath, appIconPath)
		if err != nil {
			return nil, err
		}
		hash := sha256.Sum256([]byte(relativePath))
		hashStr := fmt.Sprintf("%x", hash) + filepath.Ext(appIconPath)

		iconIDToPath[hashStr] = appIconPath
	}
	return iconIDToPath, nil
}
