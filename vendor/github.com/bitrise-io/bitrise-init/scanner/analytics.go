package scanner

func detectorErrorData(detector string, err error) map[string]interface{} {
	return map[string]interface{}{
		"detector": detector,
		"error":    err.Error(),
	}
}
