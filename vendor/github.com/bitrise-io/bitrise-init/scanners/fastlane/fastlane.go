package fastlane

import (
	"fmt"

	"github.com/bitrise-io/go-utils/pathutil"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	"github.com/bitrise-io/bitrise-init/toolscanner"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/log"
)

const scannerName = "fastlane"

const (
	unknownProjectType = "other"
	fastlaneWorkflowID = scannerName
)

const (
	configName              = "fastlane-config"
	defaultConfigNameFormat = "default-fastlane-%s-config"
)

const iosPlatform = "ios"

var platforms = []string{iosPlatform, "android"}

// Step Inputs
const (
	laneInputKey     = "lane"
	laneInputTitle   = "Fastlane lane"
	laneInputEnvKey  = "FASTLANE_LANE"
	laneInputSummary = "The lane that will be used in your builds, stored as an Environment Variable. You can change this at any time."
)

const (
	workDirInputKey     = "work_dir"
	workDirInputTitle   = "Working directory"
	workDirInputEnvKey  = "FASTLANE_WORK_DIR"
	workDirInputSummary = "The directory where your Fastfile is located."
)
const (
	projectTypeInputTitle   = "Project type"
	projectTypeInputSummary = "The project type of the app you added to Bitrise."
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
	Fastfiles    []string
	projectTypes []string
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
	fileList, err := pathutil.ListPathInDirSortedByComponents(searchDir, true)
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
func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	warnings := models.Warnings{}

	isValidFastfileFound := false

	// Inspect Fastfiles

	workDirOption := models.NewOption(workDirInputTitle, workDirInputSummary, workDirInputEnvKey, models.TypeSelector)

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

		laneOption := models.NewOption(laneInputTitle, laneInputSummary, laneInputEnvKey, models.TypeSelector)
		workDirOption.AddOption(workDir, laneOption)

		for _, lane := range lanes {
			log.TPrintf("- %s", lane)

			configOption := models.NewConfigOption(configName, nil)
			laneOption.AddConfig(lane, configOption)
		}
	}

	if !isValidFastfileFound {
		log.TErrorf("No valid Fastfile found")
		warnings = append(warnings, "No valid Fastfile found")
		return models.OptionNode{}, warnings, nil, nil
	}

	// Add project_type property option to decision tree
	optionWithProjectType := toolscanner.AddProjectTypeToOptions(*workDirOption, scanner.projectTypes)

	return optionWithProjectType, warnings, nil, nil
}

// DefaultOptions ...
func (*Scanner) DefaultOptions() models.OptionNode {
	workDirOption := models.NewOption(workDirInputTitle, workDirInputSummary, workDirInputEnvKey, models.TypeUserInput)

	laneOption := models.NewOption(laneInputTitle, laneInputSummary, laneInputEnvKey, models.TypeUserInput)
	workDirOption.AddOption("", laneOption)

	projectTypeOption := models.NewOption(projectTypeInputTitle, projectTypeInputSummary, "", models.TypeSelector)
	laneOption.AddOption("", projectTypeOption)

	for _, p := range platforms {
		configOption := models.NewConfigOption(fmt.Sprintf(defaultConfigNameFormat, p), nil)
		projectTypeOption.AddConfig(p, configOption)
	}

	return *workDirOption
}

// Configs ...
func (scanner *Scanner) Configs(_ bool) (models.BitriseConfigMap, error) {
	generateConfig := func(isIOS bool) (bitriseModels.BitriseDataModel, error) {
		configBuilder := models.NewDefaultConfigBuilder()
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)

		if isIOS {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
		}

		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.FastlaneStepListItem(
			envmanModels.EnvironmentItemModel{laneInputKey: "$" + laneInputEnvKey},
			envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey},
		))

		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

		// Fill in project type later, from the list of detected project types
		return configBuilder.Generate(unknownProjectType,
			envmanModels.EnvironmentItemModel{
				fastlaneXcodeListTimeoutEnvKey: fastlaneXcodeListTimeoutEnvValue,
			})
	}

	// Create list of possible configs with project types
	nameToConfigModel := map[string]bitriseModels.BitriseDataModel{}

	for _, platform := range scanner.projectTypes {
		config, err := generateConfig(platform == iosPlatform)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		for cfgName, dataModel := range toolscanner.AddProjectTypeToConfig(configName, config, []string{platform}) {
			nameToConfigModel[cfgName] = dataModel
		}
	}

	nameToConfigString := map[string]string{}
	for configName, config := range nameToConfigModel {
		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}
		nameToConfigString[configName] = string(data)
	}
	return nameToConfigString, nil
}

// DefaultConfigs ...
func (*Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	for _, p := range platforms {
		configBuilder := models.NewDefaultConfigBuilder()
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)

		if p == iosPlatform {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
		}

		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.FastlaneStepListItem(
			envmanModels.EnvironmentItemModel{laneInputKey: "$" + laneInputEnvKey},
			envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey},
		))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

		config, err := configBuilder.Generate(p, envmanModels.EnvironmentItemModel{fastlaneXcodeListTimeoutEnvKey: fastlaneXcodeListTimeoutEnvValue})
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		configMap[fmt.Sprintf(defaultConfigNameFormat, p)] = string(data)
	}

	return configMap, nil
}

// AutomationToolScannerInterface

// SetDetectedProjectTypes ...
func (scanner *Scanner) SetDetectedProjectTypes(detectedProjectTypes []string) {
	scanner.projectTypes = detectedProjectTypes
}
