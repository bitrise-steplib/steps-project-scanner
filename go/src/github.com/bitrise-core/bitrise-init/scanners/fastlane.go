package scanners

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/pointers"
	stepmanModels "github.com/bitrise-io/stepman/models"
)

const (
	fastlaneDetectorName = "fastlane"
)

const (
	fastFileBasePath = "Fastfile"
)

const (
	laneKey    = "lane"
	laneTitle  = "fastlane lane"
	laneEnvKey = "FASTLANE_LANE"

	workDirKey    = "work_dir"
	workDirTitle  = "Working directory"
	workDirEnvKey = "FASTLANE_WORK_DIR"

	stepFastlaneIDComposite = "fastlane@2.1.3"
)

//--------------------------------------------------
// Utility
//--------------------------------------------------

func filterFastFiles(fileList []string) []string {
	fastFiles := utility.FilterFilesWithBasPaths(fileList, fastFileBasePath)
	sort.Sort(utility.ByComponents(fastFiles))

	return fastFiles
}

func inspectFastFile(fastFile string) ([]string, error) {
	dir := filepath.Dir(fastFile)

	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer

	cmd := exec.Command("fastlane", "lanes", "--json")
	cmd.Dir = dir
	cmd.Stdout = io.Writer(&outBuffer)
	cmd.Stderr = io.Writer(&errBuffer)

	if err := cmd.Run(); err != nil {
		return []string{}, err
	}

	linesStr := outBuffer.String()
	lines := strings.Split(linesStr, "\n")

	expectedLines := []string{}
	expectedLinesStart := false
	for _, line := range lines {
		if line == "{" {
			expectedLinesStart = true
		}
		if expectedLinesStart {
			expectedLines = append(expectedLines, line)
		}
		if line == "}" {
			expectedLinesStart = false
		}
	}

	expectedLinesStr := strings.Join(expectedLines, "\n")

	laneMap := map[string]map[string]interface{}{}

	if err := json.Unmarshal([]byte(expectedLinesStr), &laneMap); err != nil {
		return []string{}, err
	}

	lanes := []string{}
	for _, laneConfig := range laneMap {
		for name := range laneConfig {
			lanes = append(lanes, name)
		}
	}

	return lanes, nil
}

func fastlaneConfigName() string {
	name := "fastlane-"
	return name + "config"
}

func fastlaneDefaultConfigName() string {
	return "default-fastlane-config"
}

//--------------------------------------------------
// Detector
//--------------------------------------------------

// Fastlane ...
type Fastlane struct {
	SearchDir string
	FastFiles []string
}

// Name ...
func (detector Fastlane) Name() string {
	return fastlaneDetectorName
}

// Configure ...
func (detector *Fastlane) Configure(searchDir string) {
	detector.SearchDir = searchDir
}

// DetectPlatform ...
func (detector *Fastlane) DetectPlatform() (bool, error) {
	fileList, err := utility.FileList(detector.SearchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", detector.SearchDir, err)
	}

	// Search for Fastfile
	logger.Info("Searching for Fastfiles")

	fastFiles := filterFastFiles(fileList)
	detector.FastFiles = fastFiles

	logger.InfofDetails("%d Fastfile(s) detected", len(fastFiles))

	if len(fastFiles) == 0 {
		logger.InfofDetails("platform not detected")
		return false, nil
	}

	logger.InfofReceipt("platform detected")

	return true, nil
}

// Options ...
func (detector *Fastlane) Options() (models.OptionModel, error) {
	workDirOption := models.NewOptionModel(workDirTitle, workDirEnvKey)

	// Inspect Fastfiles
	for _, fastFile := range detector.FastFiles {
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
		configOption.Config = fastlaneConfigName()

		laneOption := models.NewOptionModel(laneTitle, laneEnvKey)
		for _, lane := range lanes {
			laneOption.ValueMap[lane] = configOption
		}

		workDirOption.ValueMap[workDir] = laneOption
	}

	return workDirOption, nil
}

// DefaultOptions ...
func (detector *Fastlane) DefaultOptions() models.OptionModel {
	workDirOption := models.NewOptionModel(workDirTitle, workDirEnvKey)

	configOption := models.NewEmptyOptionModel()
	configOption.Config = fastlaneDefaultConfigName()

	laneOption := models.NewOptionModel(laneTitle, laneEnvKey)

	laneOption.ValueMap["_"] = configOption

	workDirOption.ValueMap["_"] = laneOption

	return workDirOption
}

// Configs ...
func (detector *Fastlane) Configs() map[string]bitriseModels.BitriseDataModel {
	steps := []bitriseModels.StepListItemModel{}
	bitriseDataMap := map[string]bitriseModels.BitriseDataModel{}

	// ActivateSSHKey
	steps = append(steps, bitriseModels.StepListItemModel{
		stepActivateSSHKeyIDComposite: stepmanModels.StepModel{
			RunIf: pointers.NewStringPtr(`{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}`),
		},
	})

	// GitClone
	steps = append(steps, bitriseModels.StepListItemModel{
		stepGitCloneIDComposite: stepmanModels.StepModel{},
	})

	// CertificateAndProfileInstaller
	steps = append(steps, bitriseModels.StepListItemModel{
		stepCertificateAndProfileInstallerIDComposite: stepmanModels.StepModel{},
	})

	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{laneKey: "$" + laneEnvKey},
		envmanModels.EnvironmentItemModel{workDirKey: "$" + workDirEnvKey},
	}

	// Fastlane
	steps = append(steps, bitriseModels.StepListItemModel{
		stepFastlaneIDComposite: stepmanModels.StepModel{
			Inputs: inputs,
		},
	})

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(steps)

	configName := fastlaneConfigName()
	bitriseDataMap[configName] = bitriseData

	return bitriseDataMap
}

// DefaultConfigs ...
func (detector *Fastlane) DefaultConfigs() map[string]bitriseModels.BitriseDataModel {
	steps := []bitriseModels.StepListItemModel{}
	bitriseDataMap := map[string]bitriseModels.BitriseDataModel{}

	// ActivateSSHKey
	steps = append(steps, bitriseModels.StepListItemModel{
		stepActivateSSHKeyIDComposite: stepmanModels.StepModel{
			RunIf: pointers.NewStringPtr(`{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}`),
		},
	})

	// GitClone
	steps = append(steps, bitriseModels.StepListItemModel{
		stepGitCloneIDComposite: stepmanModels.StepModel{},
	})

	// CertificateAndProfileInstaller
	steps = append(steps, bitriseModels.StepListItemModel{
		stepCertificateAndProfileInstallerIDComposite: stepmanModels.StepModel{},
	})

	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{laneKey: "$" + laneEnvKey},
		envmanModels.EnvironmentItemModel{workDirKey: "$" + workDirEnvKey},
	}

	// Fastlane
	steps = append(steps, bitriseModels.StepListItemModel{
		stepFastlaneIDComposite: stepmanModels.StepModel{
			Inputs: inputs,
		},
	})

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(steps)

	configName := fastlaneDefaultConfigName()
	bitriseDataMap[configName] = bitriseData

	return bitriseDataMap
}
