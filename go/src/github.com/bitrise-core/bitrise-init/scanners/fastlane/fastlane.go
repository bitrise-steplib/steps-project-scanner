package fastlane

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"

	"bufio"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
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

func filterFastfiles(fileList []string) ([]string, error) {
	allowFastfileBaseFilter := utility.BaseFilter(fastfileBasePath, true)
	fastfiles, err := utility.FilterPaths(fileList, allowFastfileBaseFilter)
	if err != nil {
		return []string{}, err
	}

	return utility.SortPathsByComponents(fastfiles)
}

func inspectFastfileContent(content string) ([]string, error) {
	commonLanes := []string{}
	laneMap := map[string][]string{}

	// platform :ios do ...
	platformSectionStartRegexp := regexp.MustCompile(`platform\s+:(?P<platform>.*)\s+do`)
	platformSectionEndPattern := "end"
	platform := ""

	// lane :test_and_snapshot do
	laneRegexp := regexp.MustCompile(`^[\s]*lane\s+:(?P<lane>.*)\s+do`)

	reader := strings.NewReader(content)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " ")

		if platform != "" && line == platformSectionEndPattern {
			platform = ""
			continue
		}

		if platform == "" {
			if match := platformSectionStartRegexp.FindStringSubmatch(line); len(match) == 2 {
				platform = match[1]
				continue
			}
		}

		if match := laneRegexp.FindStringSubmatch(line); len(match) == 2 {
			lane := match[1]

			if platform != "" {
				lanes, found := laneMap[platform]
				if !found {
					lanes = []string{}
				}
				lanes = append(lanes, lane)
				laneMap[platform] = lanes
			} else {
				commonLanes = append(commonLanes, lane)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return []string{}, err
	}

	lanes := commonLanes
	for platform, platformLanes := range laneMap {
		for _, lane := range platformLanes {
			lanes = append(lanes, platform+" "+lane)
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
	Fastfiles []string
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (scanner Scanner) Name() string {
	return scannerName
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	fileList, err := utility.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}

	// Search for Fastfile
	log.Infoft("Searching for Fastfiles")

	fastfiles, err := filterFastfiles(fileList)
	if err != nil {
		return false, fmt.Errorf("failed to search for Fastfile in (%s), error: %s", searchDir, err)
	}

	scanner.Fastfiles = fastfiles

	log.Printft("%d Fastfiles detected", len(fastfiles))
	for _, file := range fastfiles {
		log.Printft("- %s", file)
	}

	if len(fastfiles) == 0 {
		log.Printft("platform not detected")
		return false, nil
	}

	log.Doneft("Platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	workDirOption := models.NewOptionModel(workDirTitle, workDirEnvKey)
	warnings := models.Warnings{}

	isValidFastfileFound := false

	// Inspect Fastfiles
	for _, fastfile := range scanner.Fastfiles {
		log.Infoft("Inspecting Fastfile: %s", fastfile)

		lanes, err := inspectFastfile(fastfile)
		if err != nil {
			log.Warnft("Failed to inspect Fastfile, error: %s", err)
			warnings = append(warnings, fmt.Sprintf("Failed to inspect Fastfile (%s), error: %s", fastfile, err))
			continue
		}

		log.Printft("%d lanes found", len(lanes))
		for _, lane := range lanes {
			log.Printft("- %s", lane)
		}

		if len(lanes) == 0 {
			log.Warnft("No lanes found")
			warnings = append(warnings, fmt.Sprintf("No lanes found for Fastfile: %s", fastfile))
			continue
		}

		isValidFastfileFound = true

		workDir := fastlaneWorkDir(fastfile)

		log.Printft("fastlane work dir: %s", workDir)

		configOption := models.NewEmptyOptionModel()
		configOption.Config = configName()

		laneOption := models.NewOptionModel(laneTitle, laneEnvKey)
		for _, lane := range lanes {
			laneOption.ValueMap[lane] = configOption
		}

		workDirOption.ValueMap[workDir] = laneOption
	}

	if !isValidFastfileFound {
		log.Errorft("No valid Fastfile found")
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
	stepList = append(stepList, steps.ScriptSteplistItem(steps.ScriptDefaultTitle))

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

	bitriseData := models.BitriseDataWithCIWorkflow(appEnvs, stepList)
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
	stepList = append(stepList, steps.ScriptSteplistItem(steps.ScriptDefaultTitle))

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

	bitriseData := models.BitriseDataWithCIWorkflow(appEnvs, stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := defaultConfigName()
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}
