package scanners

import (
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	"gopkg.in/yaml.v2"
)

// ScannerInterface ...
type ScannerInterface interface {
	Name() string
	Configure(searchDir string)

	DetectPlatform() (bool, error)

	Options() (models.OptionModel, error)
	DefaultOptions() models.OptionModel

	Configs() (map[string]string, error)
	DefaultConfigs() (map[string]string, error)
}

func customConfigName() string {
	return "custom-config"
}

// CustomConfig ...
func CustomConfig() (map[string]string, error) {
	bitriseDataMap := map[string]string{}
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return map[string]string{}, err
	}

	configName := customConfigName()
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}
