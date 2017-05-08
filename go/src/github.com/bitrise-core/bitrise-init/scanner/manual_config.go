package scanner

import (
	"fmt"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners"
)

// ManualConfig ...
func ManualConfig() (models.ScanResultModel, error) {
	projectScanners := scanners.ActiveScanners
	projectTypeOptionMap := map[string]models.OptionModel{}
	projectTypeConfigMap := map[string]models.BitriseConfigMap{}

	for _, detector := range projectScanners {
		detectorName := detector.Name()

		option := detector.DefaultOptions()
		projectTypeOptionMap[detectorName] = option

		configs, err := detector.DefaultConfigs()
		if err != nil {
			return models.ScanResultModel{}, fmt.Errorf("Failed create default configs, error: %s", err)
		}
		projectTypeConfigMap[detectorName] = configs
	}

	customConfig, err := scanners.CustomConfig()
	if err != nil {
		return models.ScanResultModel{}, fmt.Errorf("Failed create default custom configs, error: %s", err)
	}

	projectTypeConfigMap["other"] = customConfig

	return models.ScanResultModel{
		PlatformOptionMap:    projectTypeOptionMap,
		PlatformConfigMapMap: projectTypeConfigMap,
	}, nil
}
