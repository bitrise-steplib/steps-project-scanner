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

const (
	scannerName = "fastlane"
)

const (
	fastFileBasePath = "Fastfile"
)

const (
	laneKey    = "lane"
	laneTitle  = "Fastlane lane"
	laneEnvKey = "FASTLANE_LANE"

	workDirKey    = "work_dir"
	workDirTitle  = "Working directory"
	workDirEnvKey = "FASTLANE_WORK_DIR"
)

var (
	logger = utility.NewLogger()
)

//--------------------------------------------------
// Utility
//--------------------------------------------------

func filterFastFiles(fileList []string) []string {
	fastFiles := utility.FilterFilesWithBasPaths(fileList, fastFileBasePath)
	sort.Sort(utility.ByComponents(fastFiles))

	return fastFiles
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

func inspectFastFile(fastFile string) ([]string, error) {
	content, err := fileutil.ReadStringFromFile(fastFile)
	if err != nil {
		return []string{}, err
	}

	return inspectFastfileContent(content)
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
	FastFiles []string
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
	logger.Info("Searching for Fastfiles")

	fastFiles := filterFastFiles(fileList)
	scanner.FastFiles = fastFiles

	logger.InfofDetails("%d Fastfile(s) detected:", len(fastFiles))
	for _, file := range fastFiles {
		logger.InfofDetails("  - %s", file)
	}

	if len(fastFiles) == 0 {
		logger.InfofDetails("platform not detected")
		return false, nil
	}

	logger.InfofReceipt("platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, error) {
	workDirOption := models.NewOptionModel(workDirTitle, workDirEnvKey)

	// Inspect Fastfiles
	for _, fastFile := range scanner.FastFiles {
		logger.InfofSection("Inspecting Fastfile: %s", fastFile)
		logger.InfoDetails("$ fastlane lanes --json")

		lanes, err := inspectFastFile(fastFile)
		if err != nil {
			return models.OptionModel{}, err
		}

		logger.InfofReceipt("found lanes: %v", lanes)

		// Check if `Fastfile` is in `./fastlane/Fastfile`
		// If no - generated fastlane step will require `work_dir` input too
		workDir := "./"
		relFastlaneDir := filepath.Dir(fastFile)
		if relFastlaneDir != "fastlane" {
			workDir = relFastlaneDir
		}

		logger.InfofReceipt("fastlane work dir: %s", workDir)

		configOption := models.NewEmptyOptionModel()
		configOption.Config = configName()

		laneOption := models.NewOptionModel(laneTitle, laneEnvKey)
		for _, lane := range lanes {
			laneOption.ValueMap[lane] = configOption
		}

		workDirOption.ValueMap[workDir] = laneOption
	}

	return workDirOption, nil
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
func (scanner *Scanner) Configs() (map[string]string, error) {
	stepList := []bitriseModels.StepListItemModel{}
	bitriseDataMap := map[string]string{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// CertificateAndProfileInstaller
	stepList = append(stepList, steps.CertificateAndProfileInstallerStepListItem())

	// Fastlane
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{laneKey: "$" + laneEnvKey},
		envmanModels.EnvironmentItemModel{workDirKey: "$" + workDirEnvKey},
	}

	stepList = append(stepList, steps.FastlaneStepListItem(inputs))

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return map[string]string{}, err
	}

	configName := configName()
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (map[string]string, error) {
	stepList := []bitriseModels.StepListItemModel{}
	bitriseDataMap := map[string]string{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// CertificateAndProfileInstaller
	stepList = append(stepList, steps.CertificateAndProfileInstallerStepListItem())

	// Fastlane
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{laneKey: "$" + laneEnvKey},
		envmanModels.EnvironmentItemModel{workDirKey: "$" + workDirEnvKey},
	}

	stepList = append(stepList, steps.FastlaneStepListItem(inputs))

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return map[string]string{}, err
	}

	configName := defaultConfigName()
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}
