package android

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-install-missing-android-tools/androidcomponents"
	"github.com/bitrise-tools/go-android/gradle"
	"github.com/bitrise-tools/go-android/sdk"
)

// Constants ...
const (
	ScannerName       = "android"
	ConfigName        = "android-config"
	DefaultConfigName = "default-android-config"

	ProjectLocationInputKey    = "project_location"
	ProjectLocationInputEnvKey = "PROJECT_LOCATION"
	ProjectLocationInputTitle  = "The root directory of an Android project"

	ModuleBuildGradlePathInputKey = "build_gradle_path"

	ModuleInputKey    = "module"
	ModuleInputEnvKey = "MODULE"
	ModuleInputTitle  = "Module"

	VariantInputKey         = "variant"
	TestVariantInputEnvKey  = "TEST_VARIANT"
	BuildVariantInputEnvKey = "BUILD_VARIANT"
	TestVariantInputTitle   = "Variant for testing"
	BuildVariantInputTitle  = "Variant for building"

	GradlewPathInputKey    = "gradlew_path"
	GradlewPathInputEnvKey = "GRADLEW_PATH"
	GradlewPathInputTitle  = "Gradlew file path"
)

func walk(src string, fn func(path string, info os.FileInfo) error) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}
		return fn(path, info)
	})
}

func checkFiles(path string, files ...string) (bool, error) {
	for _, file := range files {
		exists, err := pathutil.IsPathExists(filepath.Join(path, file))
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}
	return true, nil
}

func walkMultipleFiles(searchDir string, files ...string) (matches []string, err error) {
	match, err := checkFiles(searchDir, files...)
	if err != nil {
		return nil, err
	}
	if match {
		matches = append(matches, searchDir)
	}
	return matches, walk(searchDir, func(path string, info os.FileInfo) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			match, err := checkFiles(path, files...)
			if err != nil {
				return err
			}
			if match {
				matches = append(matches, path)
			}
		}
		return nil
	})
}

func checkLocalProperties(projectDir string) string {
	localPropertiesPth := filepath.Join(projectDir, "local.properties")
	exist, err := pathutil.IsPathExists(localPropertiesPth)
	if err == nil && exist {
		return fmt.Sprintf(`The local.properties file must NOT be checked into Version Control Systems, as it contains information specific to your local configuration.
The location of the file is: %s`, localPropertiesPth)
	}
	return ""
}

func checkGradlew(projectDir string) error {
	gradlewPth := filepath.Join(projectDir, "gradlew")
	exist, err := pathutil.IsPathExists(gradlewPth)
	if err != nil {
		return err
	}
	if !exist {
		return errors.New(`<b>No Gradle Wrapper (gradlew) found.</b> 
Using a Gradle Wrapper (gradlew) is required, as the wrapper is what makes sure
that the right Gradle version is installed and used for the build. More info/guide: <a>https://docs.gradle.org/current/userguide/gradle_wrapper.html</a>`)
	}
	return nil
}

func (scanner *Scanner) generateOptions(searchDir string) (models.OptionModel, models.Warnings, error) {
	warnings := models.Warnings{}

	projectLocationOption := models.NewOption(ProjectLocationInputTitle, ProjectLocationInputEnvKey)

	androidSdk, err := sdk.New(os.Getenv("ANDROID_HOME"))
	if err != nil {
		return models.OptionModel{}, warnings, err
	}

	if err := androidcomponents.InstallLicences(androidSdk); err != nil {
		return models.OptionModel{}, warnings, err
	}

	for _, projectRoot := range scanner.ProjectRoots {
		if warning := checkLocalProperties(projectRoot); warning != "" {
			warnings = append(warnings, warning)
		}

		if err := checkGradlew(projectRoot); err != nil {
			return models.OptionModel{}, warnings, err
		}

		gradlewPath := filepath.Join(projectRoot, "gradlew")

		if err := os.Chmod(gradlewPath, 0770); err != nil {
			return models.OptionModel{}, warnings, err
		}

		if err := androidcomponents.Ensure(androidSdk, gradlewPath); err != nil {
			return models.OptionModel{}, warnings, err
		}

		proj, err := gradle.NewProject(projectRoot)
		if err != nil {
			return models.OptionModel{}, warnings, err
		}
		testVariantsMap, err := proj.GetTask("test").GetVariants()
		if err != nil {
			return models.OptionModel{}, warnings, err
		}
		buildVariantsMap, err := proj.GetTask("assemble").GetVariants()
		if err != nil {
			return models.OptionModel{}, warnings, err
		}

		moduleOption := models.NewOption(ModuleInputTitle, ModuleInputEnvKey)

		for module, variants := range buildVariantsMap {
			testVariantOption := models.NewOption(TestVariantInputTitle, TestVariantInputEnvKey)
			buildVariantOption := models.NewOption(BuildVariantInputTitle, BuildVariantInputEnvKey)

			configOption := models.NewConfigOption(ConfigName)

			if !scanner.ExcludeTest {
				for _, variant := range testVariantsMap[module] {
					variant = strings.TrimSuffix(variant, "UnitTest")
					testVariantOption.AddOption(variant, configOption)
				}
				testVariantOption.AddOption("", configOption)
			}

			for _, variant := range variants {
				if !scanner.ExcludeTest {
					configOption = testVariantOption
				}
				buildVariantOption.AddOption(variant, configOption)
			}
			buildVariantOption.AddOption("", configOption)

			moduleOption.AddOption(module, buildVariantOption)
		}

		relProjectRoot, err := filepath.Rel(scanner.SearchDir, projectRoot)
		if err != nil {
			return models.OptionModel{}, warnings, err
		}

		gradlewPthOption := models.NewOption(GradlewPathInputTitle, GradlewPathInputEnvKey)
		gradlewPthOption.AddOption(filepath.Join(relProjectRoot, "gradlew"), moduleOption)

		projectLocationOption.AddOption(relProjectRoot, gradlewPthOption)
	}
	return *projectLocationOption, warnings, nil
}

func (scanner *Scanner) generateConfigBuilder(isIncludeCache bool) models.ConfigBuilderModel {
	configBuilder := models.NewDefaultConfigBuilder()

	projectLocationEnv, moduleEnv, testVariantEnv, buildVariantEnv := "$"+ProjectLocationInputEnvKey, "$"+ModuleInputEnvKey, "$"+TestVariantInputEnvKey, "$"+BuildVariantInputEnvKey

	//-- primary
	if !scanner.ExcludeTest {
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(isIncludeCache)...)
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.InstallMissingAndroidToolsStepListItem())
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.AndroidLintStepListItem(
			envmanModels.EnvironmentItemModel{ProjectLocationInputKey: projectLocationEnv},
			envmanModels.EnvironmentItemModel{ModuleInputKey: moduleEnv},
			envmanModels.EnvironmentItemModel{VariantInputKey: testVariantEnv},
		))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.AndroidUnitTestStepListItem(
			envmanModels.EnvironmentItemModel{ProjectLocationInputKey: projectLocationEnv},
			envmanModels.EnvironmentItemModel{ModuleInputKey: moduleEnv},
			envmanModels.EnvironmentItemModel{VariantInputKey: testVariantEnv},
		))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(isIncludeCache)...)
	}
	//-- deploy
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(isIncludeCache)...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem())

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.ChangeAndroidVersionCodeAndVersionNameStepListItem(
		envmanModels.EnvironmentItemModel{ModuleBuildGradlePathInputKey: filepath.Join(projectLocationEnv, moduleEnv, "build.gradle")},
	))
	if !scanner.ExcludeTest {
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidLintStepListItem(
			envmanModels.EnvironmentItemModel{ProjectLocationInputKey: projectLocationEnv},
			envmanModels.EnvironmentItemModel{ModuleInputKey: moduleEnv},
			envmanModels.EnvironmentItemModel{VariantInputKey: testVariantEnv},
		))
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidUnitTestStepListItem(
			envmanModels.EnvironmentItemModel{ProjectLocationInputKey: projectLocationEnv},
			envmanModels.EnvironmentItemModel{ModuleInputKey: moduleEnv},
			envmanModels.EnvironmentItemModel{VariantInputKey: testVariantEnv},
		))
	}
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
		envmanModels.EnvironmentItemModel{ProjectLocationInputKey: projectLocationEnv},
		envmanModels.EnvironmentItemModel{ModuleInputKey: moduleEnv},
		envmanModels.EnvironmentItemModel{VariantInputKey: buildVariantEnv},
	))
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.SignAPKStepListItem())
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(isIncludeCache)...)

	configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)

	return *configBuilder
}
