package android

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/cmdex"
)

const (
	scannerName = "android"
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

func configName(hasGradlew bool) string {
	name := "android-"
	if hasGradlew {
		name = name + "gradlew-"
	}
	return name + "config"
}

func defaultConfigName() string {
	return "default-android-config"
}

//--------------------------------------------------
// Scanner
//--------------------------------------------------

// Scanner ...
type Scanner struct {
	SearchDir   string
	FileList    []string
	GradleFiles []string

	HasGradlewFile bool
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
	scanner.FileList = fileList

	// Search for gradle file
	logger.Info("Searching for gradle files")

	gradleFiles := filterGradleFiles(fileList)
	scanner.GradleFiles = gradleFiles

	logger.InfofDetails("%d gradle file(s) detected", len(gradleFiles))

	if len(gradleFiles) == 0 {
		logger.InfofDetails("platform not detected")
		return false, nil
	}

	logger.InfofReceipt("platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, error) {
	// Search for gradlew_path input
	logger.InfoSection("Searching for gradlew files")

	gradlewFiles := filterGradlewFiles(scanner.FileList)

	logger.InfofDetails("%d gradlew file(s) detected", len(gradlewFiles))

	rootGradlewPath := ""
	if len(gradlewFiles) > 0 {
		rootGradlewPath = gradlewFiles[0]
		scanner.HasGradlewFile = true

		logger.InfofDetails("root gradlew path: %s", rootGradlewPath)
	}

	gradleBin := "gradle"
	if scanner.HasGradlewFile {
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

	for _, gradleFile := range scanner.GradleFiles {
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
			configOption.Config = configName(scanner.HasGradlewFile)

			if scanner.HasGradlewFile {
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
func (scanner *Scanner) DefaultOptions() models.OptionModel {
	configOption := models.NewEmptyOptionModel()
	configOption.Config = defaultConfigName()

	gradleFileOption := models.NewOptionModel(gradleFileTitle, gradleFileEnvKey)
	gradleTaskOption := models.NewOptionModel(gradleTaskTitle, gradleTaskEnvKey)
	gradlewPathOption := models.NewOptionModel(gradlewPathTitle, gradlewPathEnvKey)

	gradlewPathOption.ValueMap["_"] = configOption
	gradleTaskOption.ValueMap["_"] = gradlewPathOption
	gradleFileOption.ValueMap["_"] = gradleTaskOption

	return gradleFileOption
}

// Configs ...
func (scanner *Scanner) Configs() (map[string]string, error) {
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// GradleRunner
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{gradleFileKey: "$" + gradleFileEnvKey},
		envmanModels.EnvironmentItemModel{gradleTaskKey: "$" + gradleTaskEnvKey},
	}

	if scanner.HasGradlewFile {
		inputs = append(inputs, envmanModels.EnvironmentItemModel{
			gradlewPathKey: "$" + gradlewPathEnvKey,
		})
	}

	stepList = append(stepList, steps.GradleRunnerStepListItem(inputs))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return map[string]string{}, err
	}

	configName := configName(scanner.HasGradlewFile)
	bitriseDataMap := map[string]string{
		configName: string(data),
	}

	return bitriseDataMap, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (map[string]string, error) {
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// GradleRunner
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{gradleFileKey: "$" + gradleFileEnvKey},
		envmanModels.EnvironmentItemModel{gradleTaskKey: "$" + gradleTaskEnvKey},
	}

	inputs = append(inputs, envmanModels.EnvironmentItemModel{
		gradlewPathKey: "$" + gradlewPathEnvKey,
	})

	stepList = append(stepList, steps.GradleRunnerStepListItem(inputs))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return map[string]string{}, err
	}

	configName := defaultConfigName()
	bitriseDataMap := map[string]string{
		configName: string(data),
	}

	return bitriseDataMap, nil
}
