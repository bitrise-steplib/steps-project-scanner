package android

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners/xamarin"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/pathutil"
)

const (
	// ScannerName ...
	ScannerName = "android"
	// ConfigName ...
	ConfigName = "android-config"
	// DefaultConfigName ...
	DefaultConfigName = "default-android-config"

	// GradlewPathInputKey ...
	GradlewPathInputKey = "gradlew_path"
	// GradlewPathInputEnvKey ...
	GradlewPathInputEnvKey = "GRADLEW_PATH"
	// GradlewPathInputTitle ...
	GradlewPathInputTitle = "Gradlew file path"

	// GradleFileInputKey ...
	GradleFileInputKey = "gradle_file"
	// GradleFileInputEnvKey ...
	GradleFileInputEnvKey = "GRADLE_BUILD_FILE_PATH"
	// GradleFileInputTitle ...
	GradleFileInputTitle = "Path to the gradle file to use"

	// GradleTaskInputKey ...
	GradleTaskInputKey = "gradle_task"

	buildGradleBasePath = "build.gradle"
)

// CollectRootBuildGradleFiles - Collects the most root (mint path depth) build.gradle files
// May the searchDir contains multiple android projects, this case it return multiple builde.gradle path
// searchDir/android-project1/build.gradle, searchDir/android-project2/build.gradle, ...
func CollectRootBuildGradleFiles(searchDir string) ([]string, error) {
	fileList, err := utility.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return nil, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}

	return FilterRootBuildGradleFiles(fileList)
}

// CheckLocalProperties - Returns warning if local.properties exists
// Local properties may contains absolute paths (sdk.dir=/Users/xyz/Library/Android/sdk),
// it should be gitignored
func CheckLocalProperties(buildGradleFile string) string {
	projectDir := filepath.Dir(buildGradleFile)
	localPropertiesPth := filepath.Join(projectDir, "local.properties")
	exist, err := pathutil.IsPathExists(localPropertiesPth)
	if err == nil && exist {
		return fmt.Sprintf(`The local.properties file must NOT be checked into Version Control Systems, as it contains information specific to your local configuration.
The location of the file is: %s`, localPropertiesPth)
	}
	return ""
}

// EnsureGradlew - Retuns the gradle wrapper path, or error if not exists
func EnsureGradlew(buildGradleFile string) (string, error) {
	projectDir := filepath.Dir(buildGradleFile)
	gradlewPth := filepath.Join(projectDir, "gradlew")
	if exist, err := pathutil.IsPathExists(gradlewPth); err != nil {
		return "", err
	} else if !exist {
		return "", errors.New(`<b>No Gradle Wrapper (gradlew) found.</b> 
Using a Gradle Wrapper (gradlew) is required, as the wrapper is what makes sure
that the right Gradle version is installed and used for the build. More info/guide: <a>https://docs.gradle.org/current/userguide/gradle_wrapper.html</a>`)
	}

	return FixedGradlewPath(gradlewPth), nil
}

// GenerateOptions ...
func GenerateOptions(searchDir string) (models.OptionModel, models.Warnings, error) {
	warnings := models.Warnings{}

	buildGradlePths, err := CollectRootBuildGradleFiles(searchDir)
	if err != nil {
		return models.OptionModel{}, warnings, err
	}

	gradleFileOption := models.NewOption(GradleFileInputTitle, GradleFileInputEnvKey)

	for _, buildGradlePth := range buildGradlePths {
		if warning := CheckLocalProperties(buildGradlePth); warning != "" {
			warnings = append(warnings, warning)
		}

		gradlewPth, err := EnsureGradlew(buildGradlePth)
		if err != nil {
			return models.OptionModel{}, warnings, err
		}

		gradlewPthOption := models.NewOption(GradlewPathInputTitle, GradlewPathInputEnvKey)
		gradleFileOption.AddOption(buildGradlePth, gradlewPthOption)

		configOption := models.NewConfigOption(ConfigName)
		gradlewPthOption.AddConfig(gradlewPth, configOption)
	}

	return *gradleFileOption, warnings, nil
}

// GenerateConfigBuilder ...
func GenerateConfigBuilder(isIncludeCache bool) models.ConfigBuilderModel {
	configBuilder := models.NewDefaultConfigBuilder()

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(isIncludeCache)...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.InstallMissingAndroidToolsStepListItem())
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.GradleRunnerStepListItem(
		envmanModels.EnvironmentItemModel{GradleFileInputKey: "$" + GradleFileInputEnvKey},
		envmanModels.EnvironmentItemModel{GradleTaskInputKey: "assembleDebug"},
		envmanModels.EnvironmentItemModel{GradlewPathInputKey: "$" + GradlewPathInputEnvKey},
	))
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(isIncludeCache)...)

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(isIncludeCache)...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem())
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.GradleRunnerStepListItem(
		envmanModels.EnvironmentItemModel{GradleFileInputKey: "$" + GradleFileInputEnvKey},
		envmanModels.EnvironmentItemModel{GradleTaskInputKey: "assembleRelease"},
		envmanModels.EnvironmentItemModel{GradlewPathInputKey: "$" + GradlewPathInputEnvKey},
	))
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(isIncludeCache)...)

	return *configBuilder
}

// FixedGradlewPath ...
func FixedGradlewPath(gradlewPth string) string {
	split := strings.Split(gradlewPth, "/")
	if len(split) != 1 {
		return gradlewPth
	}

	if !strings.HasPrefix(gradlewPth, "./") {
		return "./" + gradlewPth
	}
	return gradlewPth
}

// FilterRootBuildGradleFiles ...
func FilterRootBuildGradleFiles(fileList []string) ([]string, error) {
	allowBuildGradleBaseFilter := utility.BaseFilter(buildGradleBasePath, true)
	denyNodeModulesComponent := utility.ComponentFilter(xamarin.NodeModulesDirName, false)
	gradleFiles, err := utility.FilterPaths(fileList, allowBuildGradleBaseFilter, denyNodeModulesComponent)
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
