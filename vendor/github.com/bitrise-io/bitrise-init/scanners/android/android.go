package android

import (
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	envmanModels "github.com/bitrise-io/envman/models"
)

const (
	ScannerName       = "android"
	ConfigName        = "android-config"
	DefaultConfigName = "default-android-config"

	testsWorkflowID         = "run_tests"
	testsWorkflowSummary    = "Run your Android unit tests and get the test report."
	testWorkflowDescription = "The workflow will first clone your Git repository, cache your Gradle dependencies, install Android tools, run your Android unit tests and save the test report."

	buildWorkflowID          = "build_apk"
	buildWorkflowSummary     = "Run your Android unit tests and create an APK file to install your app on a device or share it with your team."
	buildWorkflowDescription = "The workflow will first clone your Git repository, install Android tools, set the project's version code based on the build number, run Android lint and unit tests, build the project's APK file and save it."

	ProjectLocationInputKey     = "project_location"
	ProjectLocationInputEnvKey  = "PROJECT_LOCATION"
	ProjectLocationInputTitle   = "The root directory of an Android project"
	ProjectLocationInputSummary = "The root directory of your Android project, stored as an Environment Variable. In your Workflows, you can specify paths relative to this path. You can change this at any time."

	ModuleBuildGradlePathInputKey = "build_gradle_path"

	VariantInputKey     = "variant"
	VariantInputEnvKey  = "VARIANT"
	VariantInputTitle   = "Variant"
	VariantInputSummary = "Your Android build variant. You can add variants at any time, as well as further configure your existing variants later."

	ModuleInputKey     = "module"
	ModuleInputEnvKey  = "MODULE"
	ModuleInputTitle   = "Module"
	ModuleInputSummary = "Modules provide a container for your Android project's source code, resource files, and app level settings, such as the module-level build file and Android manifest file. Each module can be independently built, tested, and debugged. You can add new modules to your Bitrise builds at any time."

	GradlewPathInputKey = "gradlew_path"

	CacheLevelInputKey = "cache_level"
	CacheLevelNone     = "none"
)

// Scanner ...
type Scanner struct {
	Projects []Project
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (scanner *Scanner) Name() string {
	return ScannerName
}

// ExcludedScannerNames ...
func (scanner *Scanner) ExcludedScannerNames() []string {
	return nil
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (_ bool, err error) {
	projects, err := detect(searchDir)
	scanner.Projects = projects

	detected := len(projects) > 0
	return detected, err
}

// Options ...
func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	projectLocationOption := models.NewOption(ProjectLocationInputTitle, ProjectLocationInputSummary, ProjectLocationInputEnvKey, models.TypeSelector)
	warnings := models.Warnings{}
	appIconsAllProjects := models.Icons{}

	for _, project := range scanner.Projects {
		warnings = append(warnings, project.Warnings...)
		appIconsAllProjects = append(appIconsAllProjects, project.Icons...)

		iconIDs := make([]string, len(project.Icons))
		for i, icon := range project.Icons {
			iconIDs[i] = icon.Filename
		}

		configOption := models.NewConfigOption(ConfigName, iconIDs)
		moduleOption := models.NewOption(ModuleInputTitle, ModuleInputSummary, ModuleInputEnvKey, models.TypeUserInput)
		variantOption := models.NewOption(VariantInputTitle, VariantInputSummary, VariantInputEnvKey, models.TypeOptionalUserInput)

		projectLocationOption.AddOption(project.RelPath, moduleOption)
		moduleOption.AddOption("app", variantOption)
		variantOption.AddConfig("", configOption)
	}

	return *projectLocationOption, warnings, appIconsAllProjects, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionNode {
	projectLocationOption := models.NewOption(ProjectLocationInputTitle, ProjectLocationInputSummary, ProjectLocationInputEnvKey, models.TypeUserInput)
	moduleOption := models.NewOption(ModuleInputTitle, ModuleInputSummary, ModuleInputEnvKey, models.TypeUserInput)
	variantOption := models.NewOption(VariantInputTitle, VariantInputSummary, VariantInputEnvKey, models.TypeOptionalUserInput)
	configOption := models.NewConfigOption(DefaultConfigName, nil)

	projectLocationOption.AddOption(models.UserInputOptionDefaultValue, moduleOption)
	moduleOption.AddOption(models.UserInputOptionDefaultValue, variantOption)
	variantOption.AddConfig("", configOption)

	return *projectLocationOption
}

// Configs ...
func (scanner *Scanner) Configs(repoAccess models.RepoAccess) (models.BitriseConfigMap, error) {
	configBuilder := scanner.generateConfigBuilder(repoAccess)

	config, err := configBuilder.Generate(ScannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		ConfigName: string(data),
	}, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := scanner.generateConfigBuilder(models.RepoAccessUnknown)

	config, err := configBuilder.Generate(ScannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		DefaultConfigName: string(data),
	}, nil
}

func (scanner *Scanner) generateConfigBuilder(repoAccess models.RepoAccess) models.ConfigBuilderModel {
	configBuilder := models.NewDefaultConfigBuilder()

	projectLocationEnv, gradlewPath, moduleEnv, variantEnv := "$"+ProjectLocationInputEnvKey, "$"+ProjectLocationInputEnvKey+"/gradlew", "$"+ModuleInputEnvKey, "$"+VariantInputEnvKey

	//-- test
	configBuilder.AppendStepListItemsTo(testsWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		RepoAccess: repoAccess})...)
	configBuilder.AppendStepListItemsTo(testsWorkflowID, steps.RestoreGradleCache())
	configBuilder.AppendStepListItemsTo(testsWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
		envmanModels.EnvironmentItemModel{GradlewPathInputKey: gradlewPath},
	))
	configBuilder.AppendStepListItemsTo(testsWorkflowID, steps.AndroidUnitTestStepListItem(
		envmanModels.EnvironmentItemModel{
			ProjectLocationInputKey: projectLocationEnv,
		},
		envmanModels.EnvironmentItemModel{
			VariantInputKey: variantEnv,
		},
		envmanModels.EnvironmentItemModel{
			CacheLevelInputKey: CacheLevelNone,
		},
	))
	configBuilder.AppendStepListItemsTo(testsWorkflowID, steps.SaveGradleCache())
	configBuilder.AppendStepListItemsTo(testsWorkflowID, steps.DefaultDeployStepList()...)
	configBuilder.SetWorkflowSummaryTo(testsWorkflowID, testsWorkflowSummary)
	configBuilder.SetWorkflowDescriptionTo(testsWorkflowID, testWorkflowDescription)

	//-- build
	configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		RepoAccess: repoAccess,
	})...)
	configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
		envmanModels.EnvironmentItemModel{GradlewPathInputKey: gradlewPath},
	))

	configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.ChangeAndroidVersionCodeAndVersionNameStepListItem(
		envmanModels.EnvironmentItemModel{ModuleBuildGradlePathInputKey: filepath.Join(projectLocationEnv, moduleEnv, "build.gradle")},
	))

	configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.AndroidLintStepListItem(
		envmanModels.EnvironmentItemModel{
			ProjectLocationInputKey: projectLocationEnv,
		},
		envmanModels.EnvironmentItemModel{
			VariantInputKey: variantEnv,
		},
		envmanModels.EnvironmentItemModel{
			CacheLevelInputKey: CacheLevelNone,
		},
	))
	configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.AndroidUnitTestStepListItem(
		envmanModels.EnvironmentItemModel{
			ProjectLocationInputKey: projectLocationEnv,
		},
		envmanModels.EnvironmentItemModel{
			VariantInputKey: variantEnv,
		},
		envmanModels.EnvironmentItemModel{
			CacheLevelInputKey: CacheLevelNone,
		},
	))

	configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.AndroidBuildStepListItem(
		envmanModels.EnvironmentItemModel{
			ProjectLocationInputKey: projectLocationEnv,
		},
		envmanModels.EnvironmentItemModel{
			ModuleInputKey: moduleEnv,
		},
		envmanModels.EnvironmentItemModel{
			VariantInputKey: variantEnv,
		},
		envmanModels.EnvironmentItemModel{
			CacheLevelInputKey: CacheLevelNone,
		},
	))
	configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.SignAPKStepListItem())
	configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.DefaultDeployStepList()...)

	configBuilder.SetWorkflowDescriptionTo(buildWorkflowID, buildWorkflowDescription)
	configBuilder.SetWorkflowSummaryTo(buildWorkflowID, buildWorkflowSummary)

	return *configBuilder
}
