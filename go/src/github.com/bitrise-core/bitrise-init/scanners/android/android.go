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
)

var (
	log = utility.NewLogger()
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

	scriptContentKey = "content"
)

const (
	updateAndroidExtraPackagesScriptContent = `#!/bin/bash
set -ex

echo y | android update sdk --no-ui --all --filter platform-tools | grep 'package installed'

echo y | android update sdk --no-ui --all --filter extra-android-m2repository | grep 'package installed'
echo y | android update sdk --no-ui --all --filter extra-google-m2repository | grep 'package installed'
echo y | android update sdk --no-ui --all --filter extra-google-google_play_services | grep 'package installed'
`

	updateAndroidExtraPackagesScriptTite = "Update Android Extra packages"
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
	gradleFiles := utility.FilterFilesWithBasPaths(fileList, buildGradleBasePath)
	sort.Sort(utility.ByComponents(gradleFiles))

	if len(gradleFiles) == 0 {
		return []string{}, nil
	}

	mindDepth, err := utility.PathDept(gradleFiles[0])
	if err != nil {
		return []string{}, err
	}

	rootGradleFiles := []string{}
	for _, gradleFile := range gradleFiles {
		depth, err := utility.PathDept(gradleFile)
		if err != nil {
			return []string{}, err
		}

		if depth == mindDepth {
			rootGradleFiles = append(rootGradleFiles, gradleFile)
		}
	}

	return rootGradleFiles, nil
}

func filterGradlewFiles(fileList []string) []string {
	gradlewFiles := utility.FilterFilesWithBasPaths(fileList, gradlewBasePath)
	sort.Sort(utility.ByComponents(gradlewFiles))

	fixedGradlewFiles := []string{}
	for _, gradlewFile := range gradlewFiles {
		fixed := fixedGradlewPath(gradlewFile)
		fixedGradlewFiles = append(fixedGradlewFiles, fixed)
	}

	return fixedGradlewFiles
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
	fileList, err := utility.FileList(searchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}
	scanner.FileList = fileList

	// Search for gradle file
	log.Info("Searching for build.gradle files")

	gradleFiles, err := filterRootBuildGradleFiles(fileList)
	if err != nil {
		return false, fmt.Errorf("failed to search for build.gradle files, error: %s", err)
	}
	scanner.GradleFiles = gradleFiles

	log.Details("%d build.gradle file(s) detected", len(gradleFiles))
	for _, file := range gradleFiles {
		log.Details("- %s", file)
	}

	if len(gradleFiles) == 0 {
		log.Details("platform not detected")
		return false, nil
	}

	log.Done("Platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	// Search for gradlew_path input
	log.Info("Searching for gradlew files")

	warnings := models.Warnings{}
	gradlewFiles := filterGradlewFiles(scanner.FileList)

	log.Details("%d gradlew file(s) detected", len(gradlewFiles))
	for _, file := range gradlewFiles {
		log.Details("- %s", file)
	}

	rootGradlewPath := ""
	if len(gradlewFiles) > 0 {
		rootGradlewPath = gradlewFiles[0]
		log.Details("root gradlew path: %s", rootGradlewPath)
	} else {
		log.Error("No gradle wrapper (gradlew) found")
		return models.OptionModel{}, warnings, fmt.Errorf(`<b>No Gradle Wrapper (gradlew) found.</b> 
Using a Gradle Wrapper (gradlew) is required, as the wrapper is what makes sure
that the right Gradle version is installed and used for the build. More info/guide: <a>https://docs.gradle.org/current/userguide/gradle_wrapper.html</a>`)
	}

	// Inspect Gradle files
	gradleFileOption := models.NewOptionModel(gradleFileTitle, gradleFileEnvKey)

	for _, gradleFile := range scanner.GradleFiles {
		log.Info("Inspecting gradle file: %s", gradleFile)

		configs := defaultGradleTasks

		log.Details("%d gradle task(s)", len(configs))
		for _, config := range configs {
			log.Details("- %s", config)
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

	// Script - Update unversioned main android packages
	stepList = append(stepList, steps.ScriptSteplistItem(updateAndroidExtraPackagesScriptTite, envmanModels.EnvironmentItemModel{
		scriptContentKey: updateAndroidExtraPackagesScriptContent,
	}))

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

	// Script - Update unversioned main android packages
	stepList = append(stepList, steps.ScriptSteplistItem(updateAndroidExtraPackagesScriptTite, envmanModels.EnvironmentItemModel{
		scriptContentKey: updateAndroidExtraPackagesScriptContent,
	}))

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
