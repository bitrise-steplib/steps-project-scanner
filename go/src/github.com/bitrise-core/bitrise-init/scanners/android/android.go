package android

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/log"
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

var defaultGradleTasks = []string{
	"assemble",
	"assembleDebug",
	"assembleRelease",
}

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

func filterRootBuildGradleFiles(fileList []string) ([]string, error) {
	allowBuildGradleBaseFilter := utility.BaseFilter(buildGradleBasePath, true)
	gradleFiles, err := utility.FilterPaths(fileList, allowBuildGradleBaseFilter)
	if err != nil {
		return []string{}, err
	}

	if len(gradleFiles) == 0 {
		return []string{}, nil
	}

	sortableFiles := []utility.SortablePath{}
	for _, pth := range gradleFiles {
		sortable, err := utility.NewSortablePath(pth)
		if err != nil {
			return []string{}, err
		}
		sortableFiles = append(sortableFiles, sortable)
	}

	sort.Sort(utility.BySortablePathComponents(sortableFiles))
	mindDepth := len(sortableFiles[0].Components)

	rootGradleFiles := []string{}
	for _, sortable := range sortableFiles {
		depth := len(sortable.Components)
		if depth == mindDepth {
			rootGradleFiles = append(rootGradleFiles, sortable.Pth)
		}
	}

	return rootGradleFiles, nil
}

func filterGradlewFiles(fileList []string) ([]string, error) {
	allowGradlewBaseFilter := utility.BaseFilter(gradlewBasePath, true)
	gradlewFiles, err := utility.FilterPaths(fileList, allowGradlewBaseFilter)
	if err != nil {
		return []string{}, err
	}

	fixedGradlewFiles := []string{}
	for _, gradlewFile := range gradlewFiles {
		fixed := fixedGradlewPath(gradlewFile)
		fixedGradlewFiles = append(fixedGradlewFiles, fixed)
	}

	return fixedGradlewFiles, nil
}

func configName() string {
	return "android-config"
}

func defaultConfigName() string {
	return "default-android-config"
}

//--------------------------------------------------
// Scanner
//--------------------------------------------------

// Scanner ...
type Scanner struct {
	FileList    []string
	GradleFiles []string
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
	scanner.FileList = fileList

	// Search for gradle file
	log.Infoft("Searching for build.gradle files")

	gradleFiles, err := filterRootBuildGradleFiles(fileList)
	if err != nil {
		return false, fmt.Errorf("failed to search for build.gradle files, error: %s", err)
	}
	scanner.GradleFiles = gradleFiles

	log.Printft("%d build.gradle files detected", len(gradleFiles))
	for _, file := range gradleFiles {
		log.Printft("- %s", file)
	}

	if len(gradleFiles) == 0 {
		log.Printft("platform not detected")
		return false, nil
	}

	log.Doneft("Platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	// Search for gradlew_path input
	log.Infoft("Searching for gradlew files")

	warnings := models.Warnings{}
	gradlewFiles, err := filterGradlewFiles(scanner.FileList)
	if err != nil {
		return models.OptionModel{}, warnings, fmt.Errorf("Failed to list gradlew files, error: %s", err)
	}

	log.Printft("%d gradlew files detected", len(gradlewFiles))
	for _, file := range gradlewFiles {
		log.Printft("- %s", file)
	}

	rootGradlewPath := ""
	if len(gradlewFiles) > 0 {
		rootGradlewPath = gradlewFiles[0]
		log.Printft("root gradlew path: %s", rootGradlewPath)
	} else {
		log.Errorft("No gradle wrapper (gradlew) found")
		return models.OptionModel{}, warnings, fmt.Errorf(`<b>No Gradle Wrapper (gradlew) found.</b> 
Using a Gradle Wrapper (gradlew) is required, as the wrapper is what makes sure
that the right Gradle version is installed and used for the build. More info/guide: <a>https://docs.gradle.org/current/userguide/gradle_wrapper.html</a>`)
	}

	// Inspect Gradle files
	gradleFileOption := models.NewOptionModel(gradleFileTitle, gradleFileEnvKey)

	for _, gradleFile := range scanner.GradleFiles {
		log.Infoft("Inspecting gradle file: %s", gradleFile)

		configs := defaultGradleTasks

		log.Printft("%d gradle tasks", len(configs))
		for _, config := range configs {
			log.Printft("- %s", config)
		}

		gradleTaskOption := models.NewOptionModel(gradleTaskTitle, gradleTaskEnvKey)

		for _, config := range configs {
			configOption := models.NewEmptyOptionModel()
			configOption.Config = configName()

			gradlewPathOption := models.NewOptionModel(gradlewPathTitle, gradlewPathEnvKey)
			gradlewPathOption.ValueMap[rootGradlewPath] = configOption

			gradleTaskOption.ValueMap[config] = gradlewPathOption
		}

		gradleFileOption.ValueMap[gradleFile] = gradleTaskOption
	}

	return gradleFileOption, warnings, nil
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
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// Script
	stepList = append(stepList, steps.ScriptSteplistItem(steps.ScriptDefaultTitle))

	// Install missing Android tools
	stepList = append(stepList, steps.InstallMissingAndroidToolsStepListItem())

	// GradleRunner
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{gradleFileKey: "$" + gradleFileEnvKey},
		envmanModels.EnvironmentItemModel{gradleTaskKey: "$" + gradleTaskEnvKey},
		envmanModels.EnvironmentItemModel{gradlewPathKey: "$" + gradlewPathEnvKey},
	}
	stepList = append(stepList, steps.GradleRunnerStepListItem(inputs))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	bitriseData := models.BitriseDataWithCIWorkflow([]envmanModels.EnvironmentItemModel{}, stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := configName()
	bitriseDataMap := models.BitriseConfigMap{
		configName: string(data),
	}

	return bitriseDataMap, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// Script
	stepList = append(stepList, steps.ScriptSteplistItem(steps.ScriptDefaultTitle))

	// Install missing Android tools
	stepList = append(stepList, steps.InstallMissingAndroidToolsStepListItem())

	// GradleRunner
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{gradleFileKey: "$" + gradleFileEnvKey},
		envmanModels.EnvironmentItemModel{gradleTaskKey: "$" + gradleTaskEnvKey},
		envmanModels.EnvironmentItemModel{gradlewPathKey: "$" + gradlewPathEnvKey},
	}
	stepList = append(stepList, steps.GradleRunnerStepListItem(inputs))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	bitriseData := models.BitriseDataWithCIWorkflow([]envmanModels.EnvironmentItemModel{}, stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := defaultConfigName()
	bitriseDataMap := models.BitriseConfigMap{
		configName: string(data),
	}

	return bitriseDataMap, nil
}
