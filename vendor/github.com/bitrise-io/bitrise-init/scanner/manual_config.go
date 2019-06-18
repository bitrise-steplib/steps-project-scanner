package scanner

import (
	"fmt"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners"
)

// ManualConfig ...
func ManualConfig() (models.ScanResultModel, error) {
	scannerList := append(scanners.ProjectScanners, scanners.AutomationToolScanners...)
	scannerToOptionRoot := map[string]models.OptionNode{}
	scannerToBitriseConfigMap := map[string]models.BitriseConfigMap{}

	for _, scanner := range scannerList {
		options := scanner.DefaultOptions()

		scannerToOptionRoot[scanner.Name()] = options

		configs, err := scanner.DefaultConfigs()
		if err != nil {
			return models.ScanResultModel{}, fmt.Errorf("Failed create default configs, error: %s", err)
		}
		scannerToBitriseConfigMap[scanner.Name()] = configs
	}

	customConfig, err := scanners.CustomConfig()
	if err != nil {
		return models.ScanResultModel{}, fmt.Errorf("Failed create default custom configs, error: %s", err)
	}

	scannerToBitriseConfigMap[scanners.CustomProjectType] = customConfig

	return models.ScanResultModel{
		ScannerToOptionRoot:       scannerToOptionRoot,
		ScannerToBitriseConfigMap: scannerToBitriseConfigMap,
	}, nil
}
