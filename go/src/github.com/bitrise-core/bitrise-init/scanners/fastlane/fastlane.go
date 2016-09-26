package fastlane

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/fileutil"
)

var (
	log = utility.NewLogger()
)

const (
	scannerName = "fastlane"
)

const (
	fastfileBasePath = "Fastfile"
)

const (
	laneKey    = "lane"
	laneTitle  = "Fastlane lane"
	laneEnvKey = "FASTLANE_LANE"

	workDirKey    = "work_dir"
	workDirTitle  = "Working directory"
	workDirEnvKey = "FASTLANE_WORK_DIR"

	fastlaneXcodeListTimeoutEnvKey   = "FASTLANE_XCODE_LIST_TIMEOUT"
	fastlaneXcodeListTimeoutEnvValue = "120"
)

//--------------------------------------------------
// Utility
//--------------------------------------------------

func filterFastfiles(fileList []string) []string {
	fastfiles := utility.FilterFilesWithBasPaths(fileList, fastfileBasePath)
	sort.Sort(utility.ByComponents(fastfiles))

	return fastfiles
}

func inspectFastfileContent(content string) ([]string, error) {
	lanes := []string{}

	// lane :test_and_snapshot do
	regexp := regexp.MustCompile(`^ *lane :(.+) do`)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		matches := regexp.FindStringSubmatch(line)
		if len(matches) == 2 {
			lane := matches[1]
			lanes = append(lanes, lane)
		}
	}

	return lanes, nil
}

func inspectFastfile(fastFile string) ([]string, error) {
	content, err := fileutil.ReadStringFromFile(fastFile)
	if err != nil {
		return []string{}, err
	}

	return inspectFastfileContent(content)
}

// Returns:
//  - fastlane dir's parent, if Fastfile is in fastlane dir (test/fastlane/Fastfile)
//  - Fastfile's dir, if Fastfile is NOT in fastlane dir (test/Fastfile)
func fastlaneWorkDir(fastfilePth string) string {
	dirPth := filepath.Dir(fastfilePth)
	dirName := filepath.Base(dirPth)
	if dirName == "fastlane" {
		return filepath.Dir(dirPth)
	}
	return dirPth
}

func configName() string {
	return "fastlane-config"
}

func defaultConfigName() string {
	return "default-fastlane-config"
}

//--------------------------------------------------
// Scanner
//--------------------------------------------------

// Scanner ...
type Scanner struct {
	SearchDir string
	Fastfiles []string
}

// Name ...
func (scanner Scanner) Name() string {
	return scannerName
}

// Configure ...
func (scanner *Scanner) Configure(searchDir string) {
	scanner.SearchDir = searchDir
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform() (bool, error) {
	fileList, err := utility.FileList(scanner.SearchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", scanner.SearchDir, err)
	}

	// Search for Fastfile
	log.Info("Searching for Fastfiles")

	fastfiles := filterFastfiles(fileList)
	scanner.Fastfiles = fastfiles

	log.Details("%d Fastfile(s) detected", len(fastfiles))
	for _, file := range fastfiles {
		log.Details("- %s", file)
	}

	if len(fastfiles) == 0 {
		log.Details("platform not detected")
		return false, nil
	}

	log.Done("Platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	workDirOption := models.NewOptionModel(workDirTitle, workDirEnvKey)
	warnings := models.Warnings{}

	isValidFastfileFound := false

	// Inspect Fastfiles
	for _, fastfile := range scanner.Fastfiles {
		log.Info("Inspecting Fastfile: %s", fastfile)

		lanes, err := inspectFastfile(fastfile)
		if err != nil {
			log.Warn("Failed to inspect Fastfile, error: %s", err)
			warnings = append(warnings, fmt.Sprintf("Failed to inspect Fastfile (%s), error: %s", fastfile, err))
			continue
		}

		log.Details("%d lane(s) found", len(lanes))
		for _, lane := range lanes {
			log.Details("- %s", lane)
		}

		if len(lanes) == 0 {
			log.Warn("No lanes found")
			warnings = append(warnings, fmt.Sprintf("No lanes found for Fastfile: %s", fastfile))
			continue
		}

		isValidFastfileFound = true

		workDir := fastlaneWorkDir(fastfile)

		log.Details("fastlane work dir: %s", workDir)

		configOption := models.NewEmptyOptionModel()
		configOption.Config = configName()

		laneOption := models.NewOptionModel(laneTitle, laneEnvKey)
		for _, lane := range lanes {
			laneOption.ValueMap[lane] = configOption
		}

		workDirOption.ValueMap[workDir] = laneOption
	}

	if !isValidFastfileFound {
		log.Error("No valid Fastfile found")
		warnings = append(warnings, "No valid Fastfile found")
		return models.OptionModel{}, warnings, nil
	}

	return workDirOption, warnings, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionModel {
	configOption := models.NewEmptyOptionModel()
	configOption.Config = defaultConfigName()

	workDirOption := models.NewOptionModel(workDirTitle, workDirEnvKey)
	laneOption := models.NewOptionModel(laneTitle, laneEnvKey)

	laneOption.ValueMap["_"] = configOption
	workDirOption.ValueMap["_"] = laneOption

	return workDirOption
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	stepList := []bitriseModels.StepListItemModel{}
	bitriseDataMap := models.BitriseConfigMap{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// Script
	stepList = append(stepList, steps.ScriptSteplistItem())

	// CertificateAndProfileInstaller
	stepList = append(stepList, steps.CertificateAndProfileInstallerStepListItem())

	// Fastlane
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{laneKey: "$" + laneEnvKey},
		envmanModels.EnvironmentItemModel{workDirKey: "$" + workDirEnvKey},
	}
	stepList = append(stepList, steps.FastlaneStepListItem(inputs))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	// App envs
	appEnvs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{fastlaneXcodeListTimeoutEnvKey: fastlaneXcodeListTimeoutEnvValue},
	}

	bitriseData := models.BitriseDataWithDefaultTriggerMapAndAppEnvsAndPrimaryWorkflowSteps(appEnvs, stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := configName()
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	stepList := []bitriseModels.StepListItemModel{}
	bitriseDataMap := models.BitriseConfigMap{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// Script
	stepList = append(stepList, steps.ScriptSteplistItem())

	// CertificateAndProfileInstaller
	stepList = append(stepList, steps.CertificateAndProfileInstallerStepListItem())

	// Fastlane
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{laneKey: "$" + laneEnvKey},
		envmanModels.EnvironmentItemModel{workDirKey: "$" + workDirEnvKey},
	}
	stepList = append(stepList, steps.FastlaneStepListItem(inputs))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	// App envs
	appEnvs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{fastlaneXcodeListTimeoutEnvKey: fastlaneXcodeListTimeoutEnvValue},
	}

	bitriseData := models.BitriseDataWithDefaultTriggerMapAndAppEnvsAndPrimaryWorkflowSteps(appEnvs, stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := defaultConfigName()
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}
