package scanners

import (
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners/android"
	"github.com/bitrise-core/bitrise-init/scanners/cordova"
	"github.com/bitrise-core/bitrise-init/scanners/fastlane"
	"github.com/bitrise-core/bitrise-init/scanners/ionic"
	"github.com/bitrise-core/bitrise-init/scanners/ios"
	"github.com/bitrise-core/bitrise-init/scanners/macos"
	"github.com/bitrise-core/bitrise-init/scanners/reactnative"
	"github.com/bitrise-core/bitrise-init/scanners/xamarin"
	"github.com/bitrise-core/bitrise-init/steps"
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

	// Should implement as minimal logic as possible to determine if searchDir contains the - in question - platform or not.
	// Inouts:
	// - searchDir: the directory where the project to scan exists.
	// Returns:
	// - platform detected
	// - error if (if any)
	DetectPlatform(string) (bool, error)

	// ExcludedScannerNames is used to mark, which scanners should be excluded, if the current scanner detects platform.
	ExcludedScannerNames() []string

	// OptionModel is the model, used to store the available configuration combintaions.
	// It defines option branches which leads different bitrise configurations.
	// Each branch should define a complete and valid options to build the final bitrise config model.
	// Every OptionModel branch's last options has to be the key of the workflow (in the BitriseConfigMap), which will be fulfilled with the selected options.
	// Returns:
	// - OptionModel
	// - Warnings (if any)
	// - error if (if any)
	Options() (models.OptionModel, models.Warnings, error)

	// Returns:
	// - default options for the platform.
	DefaultOptions() models.OptionModel

	// BitriseConfigMap's each element is a bitrise config template which will be fulfilled with the user selected options.
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
	reactnative.NewScanner(),
	ionic.NewScanner(),
	cordova.NewScanner(),
	ios.NewScanner(),
	macos.NewScanner(),
	android.NewScanner(),
	xamarin.NewScanner(),
	fastlane.NewScanner(),
}

// CustomProjectType ...
const CustomProjectType = "other"

// CustomConfigName ...
const CustomConfigName = "other-config"

// CustomConfig ...
func CustomConfig() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

	config, err := configBuilder.Generate(CustomProjectType)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		CustomConfigName: string(data),
	}, nil
}
