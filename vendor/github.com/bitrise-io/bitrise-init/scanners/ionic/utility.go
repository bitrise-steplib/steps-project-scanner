package ionic

import (
	"github.com/bitrise-io/bitrise-init/utility"
)

// FilterRootFile ...
func FilterRootFile(fileList []string, fileName string) (string, error) {
	allowBaseFilter := utility.BaseFilter(fileName, true)
	files, err := utility.FilterPaths(fileList, allowBaseFilter)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", nil
	}

	return files[0], nil
}
