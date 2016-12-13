package scanners

import (
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners/android"
	"github.com/bitrise-core/bitrise-init/scanners/fastlane"
	"github.com/bitrise-core/bitrise-init/scanners/ios"
	"github.com/bitrise-core/bitrise-init/scanners/xamarin"
	"github.com/bitrise-core/bitrise-init/steps"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"gopkg.in/yaml.v2"
)

// ScannerInterface ...
type ScannerInterface interface {
	// The name of the scanner is used for logging and
	// to store the scanner outputs, like warnings, options and configs.
	// The outputs are stored in a map[NAME]OUTPUT, like: warningMap[ios]warnings, optionsMap[android]options, configMap[xamarin]configs, ...,
	// this means, that the SCANNER NAME HAS TO BE UNIQUE.
	// Returns:
	// - the name of the scanner
	Name() string

	// Should implement as minimal logic as possible to determin if searchDir contains the - in question - platform or not.
	// Inouts:
	// - searchDir: the directory where the project to scann exists.
	// Returns:
	// - platform detected
	// - error if (if any)
	DetectPlatform(searchDir string) (bool, error)

	// OptionModel is the model, used to store the available configuration combintaions.
	// It defines option branches which leads different bitrise configurations.
	// Each branch should define a complete and valid options to build the final bitrise config model.
	// Every OptionModel branch's last options has to be the key of the workflow (in the BitriseConfigMap), which will fulfilled with the selected options.
	// Returns:
	// - an OptionModel
	// - Warnings (if any)
	// - error if (if any)
	Options() (models.OptionModel, models.Warnings, error)

	// Returns:
	// - default options for the platform.
	DefaultOptions() models.OptionModel

	// BitriseConfigMap's each element is a bitrise config template which will fulfilled with the user selected options.
	// Every config's key should be the last option one of the OptionModel branches.
	// Returns:
	// - platform BitriseConfigMap
	Configs() (models.BitriseConfigMap, error)

	// Returns:
	// - platform default BitriseConfigMap
	DefaultConfigs() (models.BitriseConfigMap, error)
}

// ActiveScanners ...
var ActiveScanners = []ScannerInterface{
	new(ios.Scanner),
	// new(macos.Scanner),
	new(android.Scanner),
	new(xamarin.Scanner),
	new(fastlane.Scanner),
}

func customConfigName() string {
	return "custom-config"
}

// CustomConfig ...
func CustomConfig() (models.BitriseConfigMap, error) {
	bitriseDataMap := models.BitriseConfigMap{}
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// Script
	stepList = append(stepList, steps.ScriptSteplistItem(steps.ScriptDefaultTitle))

	bitriseData := models.BitriseDataWithCIWorkflow([]envmanModels.EnvironmentItemModel{}, stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return map[string]string{}, err
	}

	configName := customConfigName()
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}
