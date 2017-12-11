package fastlane

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/log"
)

const scannerName = "fastlane"

const (
	configName        = "fastlane-config"
	defaultConfigName = "default-fastlane-config"
)

// Step Inputs
const (
	laneInputKey    = "lane"
	laneInputTitle  = "Fastlane lane"
	laneInputEnvKey = "FASTLANE_LANE"
)

const (
	workDirInputKey    = "work_dir"
	workDirInputTitle  = "Working directory"
	workDirInputEnvKey = "FASTLANE_WORK_DIR"
)

const (
	fastlaneXcodeListTimeoutEnvKey   = "FASTLANE_XCODE_LIST_TIMEOUT"
	fastlaneXcodeListTimeoutEnvValue = "120"
)

//------------------
// ScannerInterface
//------------------

// Scanner ...
type Scanner struct {
	Fastfiles []string
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (Scanner) Name() string {
	return scannerName
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	fileList, err := utility.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}

	// Search for Fastfile
	log.TInfof("Searching for Fastfiles")

	fastfiles, err := FilterFastfiles(fileList)
	if err != nil {
		return false, fmt.Errorf("failed to search for Fastfile in (%s), error: %s", searchDir, err)
	}

	scanner.Fastfiles = fastfiles

	log.TPrintf("%d Fastfiles detected", len(fastfiles))
	for _, file := range fastfiles {
		log.TPrintf("- %s", file)
	}

	if len(fastfiles) == 0 {
		log.TPrintf("platform not detected")
		return false, nil
	}

	log.TSuccessf("Platform detected")

	return true, nil
}

// ExcludedScannerNames ...
func (*Scanner) ExcludedScannerNames() []string {
	return []string{}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	warnings := models.Warnings{}

	isValidFastfileFound := false

	// Inspect Fastfiles

	workDirOption := models.NewOption(workDirInputTitle, workDirInputEnvKey)

	for _, fastfile := range scanner.Fastfiles {
		log.TInfof("Inspecting Fastfile: %s", fastfile)

		workDir := WorkDir(fastfile)
		log.TPrintf("fastlane work dir: %s", workDir)

		lanes, err := InspectFastfile(fastfile)
		if err != nil {
			log.TWarnf("Failed to inspect Fastfile, error: %s", err)
			warnings = append(warnings, fmt.Sprintf("Failed to inspect Fastfile (%s), error: %s", fastfile, err))
			continue
		}

		log.TPrintf("%d lanes found", len(lanes))

		if len(lanes) == 0 {
			log.TWarnf("No lanes found")
			warnings = append(warnings, fmt.Sprintf("No lanes found for Fastfile: %s", fastfile))
			continue
		}

		isValidFastfileFound = true

		laneOption := models.NewOption(laneInputTitle, laneInputEnvKey)
		workDirOption.AddOption(workDir, laneOption)

		for _, lane := range lanes {
			log.TPrintf("- %s", lane)

			configOption := models.NewConfigOption(configName)
			laneOption.AddConfig(lane, configOption)
		}
	}

	if !isValidFastfileFound {
		log.TErrorf("No valid Fastfile found")
		warnings = append(warnings, "No valid Fastfile found")
		return models.OptionModel{}, warnings, nil
	}

	return *workDirOption, warnings, nil
}

// DefaultOptions ...
func (*Scanner) DefaultOptions() models.OptionModel {
	workDirOption := models.NewOption(workDirInputTitle, workDirInputEnvKey)

	laneOption := models.NewOption(laneInputTitle, laneInputEnvKey)
	workDirOption.AddOption("_", laneOption)

	configOption := models.NewConfigOption(defaultConfigName)
	laneOption.AddConfig("_", configOption)

	return *workDirOption
}

// Configs ...
func (*Scanner) Configs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.FastlaneStepListItem(
		envmanModels.EnvironmentItemModel{laneInputKey: "$" + laneInputEnvKey},
		envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey},
	))

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

	config, err := configBuilder.Generate(scannerName, envmanModels.EnvironmentItemModel{fastlaneXcodeListTimeoutEnvKey: fastlaneXcodeListTimeoutEnvValue})
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
func (*Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.FastlaneStepListItem(
		envmanModels.EnvironmentItemModel{laneInputKey: "$" + laneInputEnvKey},
		envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey},
	))
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

	config, err := configBuilder.Generate(scannerName, envmanModels.EnvironmentItemModel{fastlaneXcodeListTimeoutEnvKey: fastlaneXcodeListTimeoutEnvValue})
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
