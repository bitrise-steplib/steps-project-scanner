package scanners

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/cmdex"
	stepmanModels "github.com/bitrise-io/stepman/models"
)

const (
	androidDetectorName = "android"
)

const (
	buildGradleBasePath = "build.gradle"
	gradlewBasePath     = "gradlew"
)

const (
	gradleFileKey    = "gradle_file"
	gradleFileTitle  = "Path to the gradle file to use"
	gradleFileEnvKey = "BITRISE_PROJECT_PATH"

	gradleTaskKey    = "gradle_task"
	gradleTaskTitle  = "Gradle task to run"
	gradleTaskEnvKey = "GRADLE_TASK"

	gradlewPathKey    = "gradlew_path"
	gradlewPathTitle  = "Gradlew file path"
	gradlewPathEnvKey = "GRADLEW_PATH"

	stepGradleRunnerIDComposite = "gradle-runner@1.3.1"
)

//--------------------------------------------------
// Utility
//--------------------------------------------------

func filterGradleFiles(fileList []string) []string {
	gradleFiles := utility.FilterFilesWithBasPaths(fileList, buildGradleBasePath)
	sort.Sort(utility.ByComponents(gradleFiles))

	return gradleFiles
}

func filterGradlewFiles(fileList []string) []string {
	gradlewFiles := utility.FilterFilesWithBasPaths(fileList, gradlewBasePath)
	sort.Sort(utility.ByComponents(gradlewFiles))

	return gradlewFiles
}

func inspectGradleFile(gradleFile string, gradleBin string) ([]string, error) {
	out, err := cmdex.RunCommandAndReturnCombinedStdoutAndStderr(gradleBin, "tasks", "--build-file", gradleFile)
	if err != nil {
		return []string{}, fmt.Errorf("output: %s, error: %s", out, err)
	}

	lines := strings.Split(out, "\n")
	isBuildTaskSection := false
	buildTasksExp := regexp.MustCompile(`^Build tasks`)
	configurationExp := regexp.MustCompile(`^(assemble\S+)(\s*-\s*.*)*`)

	configurations := []string{}
	for _, line := range lines {
		if !isBuildTaskSection && buildTasksExp.FindString(line) != "" {
			isBuildTaskSection = true
			continue
		} else if line == "" {
			isBuildTaskSection = false
			continue
		}

		if !isBuildTaskSection {
			continue
		}

		match := configurationExp.FindStringSubmatch(line)
		if len(match) > 1 {
			configurations = append(configurations, match[1])
		}
	}

	return configurations, nil
}

func androidConfigName(hasGradlew bool) string {
	name := "android-"
	if hasGradlew {
		name = name + "gradlew-"
	}
	return name + "config"
}

//--------------------------------------------------
// Detector
//--------------------------------------------------

// Android ...
type Android struct {
	SearchDir   string
	FileList    []string
	GradleFiles []string

	HasGradlewFile bool
}

// Name ...
func (detector Android) Name() string {
	return androidDetectorName
}

// Configure ...
func (detector *Android) Configure(searchDir string) {
	detector.SearchDir = searchDir
}

// DetectPlatform ...
func (detector *Android) DetectPlatform() (bool, error) {
	fileList, err := utility.FileList(detector.SearchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", detector.SearchDir, err)
	}
	detector.FileList = fileList

	// Search for gradle file
	gradleFiles := filterGradleFiles(fileList)
	detector.GradleFiles = gradleFiles

	if len(gradleFiles) == 0 {
		return false, nil
	}

	return true, nil
}

// Analyze ...
func (detector *Android) Analyze() ([]models.OptionModel, error) {
	//
	// Search for gradlew_path input
	gradlewFiles := filterGradlewFiles(detector.FileList)

	rootGradlewPath := ""
	if len(gradlewFiles) > 0 {
		rootGradlewPath = gradlewFiles[0]
		detector.HasGradlewFile = true
	}

	var gradlewPathOption models.OptionModel
	gradleBin := "gradle"
	if detector.HasGradlewFile {
		err := os.Chmod(rootGradlewPath, 0770)
		if err != nil {
			return []models.OptionModel{}, fmt.Errorf("failed to add executable permission on gradlew file (%s), error: %s", rootGradlewPath, err)
		}

		gradleBin = rootGradlewPath

		gradlewPathOption = models.NewOptionModel(gradlewPathKey, gradlewPathTitle, gradlewPathEnvKey)
		gradlewPathOption.AddValueMapItems(rootGradlewPath)
	}

	// Inspect Gradle files
	gradleFileOption := models.NewOptionModel(gradleFileKey, gradleFileTitle, gradleFileEnvKey)
	for _, gradleFile := range detector.GradleFiles {
		log.Infof("Inspecting gradle file: %s", gradleFile)
		log.Infof(" $ %s tasks --build-file %s", gradleBin, gradleFile)

		configs, err := inspectGradleFile(gradleFile, gradleBin)
		if err != nil {
			return []models.OptionModel{}, fmt.Errorf("failed to inspect gradle files, error: %s", err)
		}

		gradleTaskOption := models.NewOptionModel(gradleTaskKey, gradleTaskTitle, gradleTaskEnvKey)
		for _, config := range configs {
			configOption := models.NewEmptyOptionModel()
			configOption.Config = androidConfigName(detector.HasGradlewFile)

			gradleTaskOption.AddValueMapItems(config, configOption)
		}

		gradleFileOption.AddValueMapItems(gradleFile, gradleTaskOption)
	}

	options := []models.OptionModel{
		gradleFileOption,
	}

	if rootGradlewPath != "" {
		options = append(options, gradlewPathOption)
	}

	return options, nil
}

// Configs ...
func (detector *Android) Configs(isPrivate bool) map[string]bitriseModels.BitriseDataModel {
	steps := []bitriseModels.StepListItemModel{}

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

	// GradleRunner
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{gradleFileKey: "$" + gradleFileEnvKey},
		envmanModels.EnvironmentItemModel{gradleTaskKey: "$" + gradleTaskEnvKey},
	}

	if detector.HasGradlewFile {
		inputs = append(inputs, envmanModels.EnvironmentItemModel{
			gradlewPathKey: "$" + gradlewPathEnvKey,
		})
	}

	// GradleRunner
	steps = append(steps, bitriseModels.StepListItemModel{
		stepGradleRunnerIDComposite: stepmanModels.StepModel{
			Inputs: inputs,
		},
	})

	// DeployToBitriseIo
	steps = append(steps, bitriseModels.StepListItemModel{
		stepDeployToBitriseIoIDComposite: stepmanModels.StepModel{},
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

	configName := androidConfigName(detector.HasGradlewFile)
	bitriseDataMap := map[string]bitriseModels.BitriseDataModel{
		configName: bitriseData,
	}

	return bitriseDataMap
}
