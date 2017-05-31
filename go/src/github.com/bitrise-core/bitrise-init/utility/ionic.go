package utility

const ionicConfigJsonBasePath = "ionic.config.json"

// FilterRootIonicConfigJsonFile ...
func FilterRootIonicConfigJsonFile(fileList []string) (string, error) {
	allowIonicConfigJsonBaseFilter := BaseFilter(ionicConfigJsonBasePath, true)
	ionicConfigJsons, err := FilterPaths(fileList, allowIonicConfigJsonBaseFilter)
	if err != nil {
		return "", err
	}

	if len(ionicConfigJsons) == 0 {
		return "", nil
	}

	return ionicConfigJsons[0], nil
}
