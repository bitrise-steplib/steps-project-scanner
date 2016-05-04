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

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
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

func fastlaneConfigName(isWorkDirSet bool) string {
	name := "fastlane-"
	if isWorkDirSet {
		name = name + "workdir-"
	}
	return name + "config"
}

//--------------------------------------------------
// Detector
//--------------------------------------------------

// Fastlane ...
type Fastlane struct {
	SearchDir string
	FastFiles []string

	IsWorkDirSet bool
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
	// Search for Fastfile
	fileList, err := utility.FileList(detector.SearchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", detector.SearchDir, err)
	}

	fastFiles := filterFastFiles(fileList)
	detector.FastFiles = fastFiles
	return len(fastFiles) > 0, nil
}

// Analyze ...
func (detector *Fastlane) Analyze() ([]models.OptionModel, error) {
	options := []models.OptionModel{}

	// Inspect Fastfiles
	for _, fastFile := range detector.FastFiles {
		log.Infof("Inspecting Fastfile: %s", fastFile)

		lanes, err := inspectFastFile(fastFile)
		if err != nil {
			return []models.OptionModel{}, err
		}

		// Check if `Fastfile` is in `PROJECT_ROOT/fastlane/Fastfile`
		// If no - generated fastlane step will require `work_dir` input too
		relFastfile, err := filepath.Rel(detector.SearchDir, fastFile)
		if err != nil {
			return []models.OptionModel{}, err
		}

		relFastlaneDir := filepath.Dir(relFastfile)

		if relFastlaneDir != "fastlane" {
			detector.IsWorkDirSet = true

			configOption := models.NewEmptyOptionModel()
			configOption.Config = fastlaneConfigName(true)

			laneOption := models.NewOptionModel(laneKey, laneTitle, laneEnvKey)
			for _, lane := range lanes {
				laneOption.AddValueMapItems(lane, configOption)
			}

			fastFileDir := filepath.Dir(fastFile)

			workDirOption := models.NewOptionModel(workDirKey, workDirTitle, workDirEnvKey)
			workDirOption.AddValueMapItems(fastFileDir, laneOption)

			options = append(options, workDirOption)
		} else {
			configOption := models.NewEmptyOptionModel()
			configOption.Config = fastlaneConfigName(false)

			laneOption := models.NewOptionModel(laneKey, laneTitle, laneEnvKey)
			for _, lane := range lanes {
				laneOption.AddValueMapItems(lane, configOption)
			}

			options = append(options, laneOption)
		}
	}

	return options, nil
}

// Configs ...
func (detector *Fastlane) Configs(isPrivate bool) map[string]bitriseModels.BitriseDataModel {
	steps := []bitriseModels.StepListItemModel{}
	bitriseDataMap := map[string]bitriseModels.BitriseDataModel{}

	// ActivateSSHKey
	if isPrivate {
		steps = append(steps, bitriseModels.StepListItemModel{
			stepActivateSSHKeyIDComposite: stepmanModels.StepModel{},
		})
	}

	// GitClone
	steps = append(steps, bitriseModels.StepListItemModel{
		stepGitCloneIDComposite: stepmanModels.StepModel{},
	})

	// CertificateAndProfileInstaller
	steps = append(steps, bitriseModels.StepListItemModel{
		stepCertificateAndProfileInstallerIDComposite: stepmanModels.StepModel{},
	})

	inputs := []envmanModels.EnvironmentItemModel{envmanModels.EnvironmentItemModel{laneKey: "$" + laneEnvKey}}

	if detector.IsWorkDirSet {
		workDirInputs := append(inputs, envmanModels.EnvironmentItemModel{workDirKey: "$" + workDirEnvKey})

		// Fastlane
		workDirSteps := append(steps, bitriseModels.StepListItemModel{
			stepFastlaneIDComposite: stepmanModels.StepModel{
				Inputs: workDirInputs,
			},
		})

		workflows := map[string]bitriseModels.WorkflowModel{
			"primary": bitriseModels.WorkflowModel{
				Steps: workDirSteps,
			},
		}

		bitriseData := bitriseModels.BitriseDataModel{
			Workflows:            workflows,
			FormatVersion:        "1.1.0",
			DefaultStepLibSource: "https://github.com/bitrise-io/bitrise-steplib.git",
		}

		configName := fastlaneConfigName(true)
		bitriseDataMap[configName] = bitriseData
	}

	// Fastlane
	steps = append(steps, bitriseModels.StepListItemModel{
		stepFastlaneIDComposite: stepmanModels.StepModel{
			Inputs: inputs,
		},
	})

	workflows := map[string]bitriseModels.WorkflowModel{
		"primary": bitriseModels.WorkflowModel{
			Steps: steps,
		},
	}

	bitriseData := bitriseModels.BitriseDataModel{
		Workflows:            workflows,
		FormatVersion:        "1.1.0",
		DefaultStepLibSource: "https://github.com/bitrise-io/bitrise-steplib.git",
	}

	configName := fastlaneConfigName(false)
	bitriseDataMap[configName] = bitriseData

	return bitriseDataMap
}
