package scanners

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/pointers"
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
	gradleFileEnvKey = "GRADLE_BUILD_FILE_PATH"

	gradleTaskKey    = "gradle_task"
	gradleTaskTitle  = "Gradle task to run"
	gradleTaskEnvKey = "GRADLE_TASK"

	gradlewPathKey    = "gradlew_path"
	gradlewPathTitle  = "Gradlew file path"
	gradlewPathEnvKey = "GRADLEW_PATH"

	stepGradleRunnerIDComposite = "gradle-runner@1.3.1"
)

var (
	logger = utility.NewLogger()
)

//--------------------------------------------------
// Utility
//--------------------------------------------------

func fixedGradlewPath(gradlewPth string) string {
	split := strings.Split(gradlewPth, "/")
	if len(split) != 1 {
		return gradlewPth
	}

	if !strings.HasPrefix(gradlewPth, "./") {
		return "./" + gradlewPth
	}
	return gradlewPth
}

func filterGradleFiles(fileList []string) []string {
	gradleFiles := utility.FilterFilesWithBasPaths(fileList, buildGradleBasePath)
	sort.Sort(utility.ByComponents(gradleFiles))

	return gradleFiles
}

func filterGradlewFiles(fileList []string) []string {
	gradlewFiles := utility.FilterFilesWithBasPaths(fileList, gradlewBasePath)

	fixedGradlewFiles := []string{}
	for _, gradlewFile := range gradlewFiles {
		fixedGradlewFiles = append(fixedGradlewFiles, fixedGradlewPath(gradlewFile))
	}

	sort.Sort(utility.ByComponents(fixedGradlewFiles))

	return fixedGradlewFiles
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

func androidDefaultConfigName() string {
	return "default-android-config"
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
	logger.Info("Searching for gradle files")

	gradleFiles := filterGradleFiles(fileList)
	detector.GradleFiles = gradleFiles

	logger.InfofDetails("%d gradle file(s) detected", len(gradleFiles))

	if len(gradleFiles) == 0 {
		logger.InfofDetails("platform not detected")
		return false, nil
	}

	logger.InfofReceipt("platform detected")

	return true, nil
}

// Options ...
func (detector *Android) Options() (models.OptionModel, error) {
	// Search for gradlew_path input
	logger.InfoSection("Searching for gradlew files")

	gradlewFiles := filterGradlewFiles(detector.FileList)

	logger.InfofDetails("%d gradlew file(s) detected", len(gradlewFiles))

	rootGradlewPath := ""
	if len(gradlewFiles) > 0 {
		rootGradlewPath = gradlewFiles[0]
		detector.HasGradlewFile = true

		logger.InfofDetails("root gradlew path: %s", rootGradlewPath)
	}

	gradleBin := "gradle"
	if detector.HasGradlewFile {
		logger.InfofDetails("adding executable permission to gradlew file")

		err := os.Chmod(rootGradlewPath, 0770)
		if err != nil {
			return models.OptionModel{}, fmt.Errorf("failed to add executable permission on gradlew file (%s), error: %s", rootGradlewPath, err)
		}

		gradleBin = rootGradlewPath
	}

	logger.InfofReceipt("gradle bin to use by inspect: %s", gradleBin)

	// Inspect Gradle files
	gradleFileOption := models.NewOptionModel(gradleFileTitle, gradleFileEnvKey)

	for _, gradleFile := range detector.GradleFiles {
		logger.InfofSection("Inspecting gradle file: %s", gradleFile)
		logger.InfofDetails("$ %s tasks --build-file %s", gradleBin, gradleFile)

		configs, err := inspectGradleFile(gradleFile, gradleBin)
		if err != nil {
			return models.OptionModel{}, fmt.Errorf("failed to inspect gradle files, error: %s", err)
		}

		logger.InfofReceipt("found gradle tasks: %v", configs)

		gradleTaskOption := models.NewOptionModel(gradleTaskTitle, gradleTaskEnvKey)
		for _, config := range configs {

			configOption := models.NewEmptyOptionModel()
			configOption.Config = androidConfigName(detector.HasGradlewFile)

			if detector.HasGradlewFile {
				gradlewPathOption := models.NewOptionModel(gradlewPathTitle, gradlewPathEnvKey)
				gradlewPathOption.ValueMap[rootGradlewPath] = configOption

				gradleTaskOption.ValueMap[config] = gradlewPathOption
			} else {
				gradleTaskOption.ValueMap[config] = configOption
			}
		}

		gradleFileOption.ValueMap[gradleFile] = gradleTaskOption
	}

	return gradleFileOption, nil
}

// DefaultOptions ...
func (detector *Android) DefaultOptions() models.OptionModel {
	gradleFileOption := models.NewOptionModel(gradleFileTitle, gradleFileEnvKey)

	gradleTaskOption := models.NewOptionModel(gradleTaskTitle, gradleTaskEnvKey)

	configOption := models.NewEmptyOptionModel()
	configOption.Config = androidDefaultConfigName()

	gradlewPathOption := models.NewOptionModel(gradlewPathTitle, gradlewPathEnvKey)

	gradlewPathOption.ValueMap["_"] = configOption
	gradleTaskOption.ValueMap["_"] = gradlewPathOption
	gradleFileOption.ValueMap["_"] = gradleTaskOption

	return gradleFileOption
}

// Configs ...
func (detector *Android) Configs() (map[string]string, error) {
	steps := []bitriseModels.StepListItemModel{}

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

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(steps)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return map[string]string{}, err
	}

	configName := androidConfigName(detector.HasGradlewFile)
	bitriseDataMap := map[string]string{
		configName: string(data),
	}

	return bitriseDataMap, nil
}

// DefaultConfigs ...
func (detector *Android) DefaultConfigs() (map[string]string, error) {
	steps := []bitriseModels.StepListItemModel{}

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

	// GradleRunner
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{gradleFileKey: "$" + gradleFileEnvKey},
		envmanModels.EnvironmentItemModel{gradleTaskKey: "$" + gradleTaskEnvKey},
	}

	inputs = append(inputs, envmanModels.EnvironmentItemModel{
		gradlewPathKey: "$" + gradlewPathEnvKey,
	})

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

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(steps)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return map[string]string{}, err
	}

	configName := androidDefaultConfigName()
	bitriseDataMap := map[string]string{
		configName: string(data),
	}

	return bitriseDataMap, nil
}
