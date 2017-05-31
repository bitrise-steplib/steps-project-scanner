package ionic

import (
	"fmt"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners/android"
	"github.com/bitrise-core/bitrise-init/scanners/cordova"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/log"
)

const ScannerName = "ionic"

const (
	configName        = "ionic-config"
	defaultConfigName = "default-ionic-config"
)

// Step inputs

const (
	ionicProjectPathInputKey    = "path"
	ionicProjectPathInputTitle  = "Directory of Ionic project"
	ionicProjectPathInputEnvKey = "IONIC_PROJECT_PATH"
)

const (
	platformInputKey    = "build_for_platform"
	platformInputTitle  = "Platform to use in ionic-cli commands"
	platformInputEnvKey = "IONIC_PLATFORM"
)

//------------------
// ScannerInterface
//------------------

// Scanner ...
type Scanner struct {
	ionicConfigPth      string
	relIonicConfigDir   string
	searchDir           string
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (scanner Scanner) Name() string {
	return ScannerName
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	fileList, err := utility.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}

	// Search for ionic.config.json file
	log.Infoft("Searching for ionic.config.json file")

	configJsonPth, err := utility.FilterRootIonicConfigJsonFile(fileList)
	if err != nil {
		return false, fmt.Errorf("failed to search for ionic.config.json file, error: %s", err)
	}

	log.Printft("ionic.config.json: %s", configJsonPth)

	if configJsonPth == "" {
		log.Printft("platform not detected")
		return false, nil
	}

	log.Doneft("Platform detected")

	scanner.ionicConfigPth = configJsonPth
	scanner.searchDir = searchDir

	return true, nil
}

// ExcludedScannerNames ...
func (scanner *Scanner) ExcludedScannerNames() []string {
	return []string{
		string(utility.XcodeProjectTypeIOS),
		string(utility.XcodeProjectTypeMacOS),
		android.ScannerName,
		cordova.ScannerName,
	}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	warnings := models.Warnings{}

	// Get relative ionic.config.json dir
	configDir := filepath.Dir(scanner.ionicConfigPth)
	relIonicConfigDir, err := utility.RelPath(scanner.searchDir, configDir)
	if err != nil {
		return models.OptionModel{}, warnings, fmt.Errorf("Failed to get relative ionic.config.json dir path, error: %s", err)
	}
	if relIonicConfigDir == "." {
		// ionic.config.json placed in the search dir, no need to change-dir in the workflows
		relIonicConfigDir = ""
	}
	scanner.relIonicConfigDir = relIonicConfigDir
	// ---

	// Options
	var rootOption *models.OptionModel

	platforms := []string{"ios", "android"}

	if relIonicConfigDir != "" {
		rootOption = models.NewOption(ionicProjectPathInputTitle, ionicProjectPathInputEnvKey)

		projectTypeOption := models.NewOption(platformInputTitle, platformInputEnvKey)
		rootOption.AddOption(relIonicConfigDir, projectTypeOption)

		for _, platform := range platforms {
			configOption := models.NewConfigOption(configName)
			projectTypeOption.AddConfig(platform, configOption)
		}
	} else {
		rootOption = models.NewOption(platformInputTitle, platformInputEnvKey)

		for _, platform := range platforms {
			configOption := models.NewConfigOption(configName)
			rootOption.AddConfig(platform, configOption)
		}
	}

	return *rootOption, warnings, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionModel {
	ionicPlatform := models.NewOption(platformInputTitle, platformInputEnvKey)

	platforms := []string{"ios", "android"}
	for _, platform := range platforms {
		ionicPlatform.AddOption(platform, models.NewConfigOption(defaultConfigName))
	}

	return *ionicPlatform
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	if scanner.relIonicConfigDir != "" {
		configBuilder.AppendPreparStepList(steps.ChangeWorkDirStepListItem(envmanModels.EnvironmentItemModel{ionicProjectPathInputKey: "$" + ionicProjectPathInputEnvKey}))
	}

	configBuilder.AppendMainStepList(steps.IonicBuildStepListItem(envmanModels.EnvironmentItemModel{
		platformInputKey: "$" + platformInputEnvKey,
	}))

	config, err := configBuilder.Generate(ScannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		configName: string(data),
	}, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	configBuilder.AppendMainStepList(steps.IonicBuildStepListItem(envmanModels.EnvironmentItemModel{
		platformInputKey: "$" + platformInputEnvKey,
	}))

	config, err := configBuilder.Generate(ScannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		defaultConfigName: string(data),
	}, nil
}
