package scanners

import (
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/cordova"
	"github.com/bitrise-io/bitrise-init/scanners/fastlane"
	"github.com/bitrise-io/bitrise-init/scanners/flutter"
	"github.com/bitrise-io/bitrise-init/scanners/ionic"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/scanners/macos"
	"github.com/bitrise-io/bitrise-init/scanners/nodejs"
	"github.com/bitrise-io/bitrise-init/scanners/reactnative"
	"github.com/bitrise-io/bitrise-init/steps"
	"gopkg.in/yaml.v2"
)

// ScannerInterface ...
type ScannerInterface interface {
	// The name of the scanner is used for logging and
	// to store the scanner outputs, like warnings, options and configs.
	// The outputs are stored in a map[NAME]OUTPUT, like: warningMap[ios]warnings, optionsMap[android]options, ...,
	// this means, that the SCANNER NAME HAS TO BE UNIQUE.
	// Returns:
	// - the name of the scanner
	Name() string

	// Should implement as minimal logic as possible to determine if searchDir contains the - in question - platform or not.
	// Inputs:
	// - searchDir: the directory where the project to scan exists.
	// Returns:
	// - platform detected
	// - error if (if any)
	DetectPlatform(string) (bool, error)

	// ExcludedScannerNames is used to mark, which scanners should be excluded, if the current scanner detects platform.
	ExcludedScannerNames() []string

	// OptionNode is the model, an n-ary tree, used to store the available configuration combinations.
	// It defines an option decision tree whose every branch maps to a bitrise configuration.
	// Each branch should define a complete and valid options to build the final bitrise config model.
	// Every leaf node has to be the key of the workflow (in the BitriseConfigMap), which will be fulfilled with the selected options.
	// Returns:
	// - OptionNode
	// - Warnings (if any)
	// - error if (if any)
	Options() (models.OptionNode, models.Warnings, models.Icons, error)

	// Returns:
	// - default options for the platform.
	DefaultOptions() models.OptionNode

	// BitriseConfigMap's each element is a bitrise config template which will be fulfilled with the user selected options.
	// Every config's key should be the last option one of the OptionNode branches.
	// Returns:
	// - platform BitriseConfigMap
	Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error)

	// Returns:
	// - platform default BitriseConfigMap
	DefaultConfigs() (models.BitriseConfigMap, error)
}

// AutomationToolScanner contains additional methods (relative to ScannerInterface)
// implemented by an AutomationToolScanner
type AutomationToolScanner interface {
	// Set the project types detected
	SetDetectedProjectTypes(projectTypes []string)
}

// ProjectScanners ...
func ProjectScanners() []ScannerInterface {
	return []ScannerInterface{
		reactnative.NewScanner(),
		flutter.NewScanner(),
		ionic.NewScanner(),
		cordova.NewScanner(),
		ios.NewScanner(),
		macos.NewScanner(),
		android.NewScanner(),
		nodejs.NewScanner(),
	}
}

// AutomationToolScanners returns active automation tool scanners
func AutomationToolScanners() []ScannerInterface {
	return []ScannerInterface{
		fastlane.NewScanner(),
	}
}

// CustomProjectType ...
const CustomProjectType = "other"

// CustomConfigName ...
const CustomConfigName = "other-config"

// CustomConfig ...
func CustomConfig() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		SSHKeyActivation: models.SSHKeyActivationConditional,
	})...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList()...)

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
