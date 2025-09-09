package kmp

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/detectors/direntry"
	"github.com/bitrise-io/bitrise-init/detectors/gradle"
	"github.com/bitrise-io/bitrise-init/detectors/kmp"
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/scanners/java"
	"github.com/bitrise-io/bitrise-init/steps"
	bitriseModels "github.com/bitrise-io/bitrise/v2/models"
	envmanModels "github.com/bitrise-io/envman/v2/models"
	"github.com/bitrise-io/go-utils/log"
)

/*
Relevant Gradle dependencies:
	plugins:
		org.jetbrains.kotlin.multiplatform -> kotlin("multiplatform")
			This plugin is used to enable Kotlin Multiplatform projects, allowing you to share code between different platforms (e.g., JVM, JS, Native).
*/

const (
	projectType = "kotlin-multiplatform"
)

// KMP project common options
const (
	gradleProjectRootDirInputEnvKey  = "PROJECT_ROOT_DIR"
	gradleProjectRootDirInputTitle   = "The root directory of the Kotlin Multiplatform project."
	gradleProjectRootDirInputSummary = "The root directory of the Kotlin Multiplatform project, which contains all source files from your project, as well as Gradle files, including the Gradle Wrapper (gradlew) file."
	optionValueYes                   = "yes"
	optionValueNo                    = "no"
)

// Android App project options
const (
	moduleInputTitle                  = "Android Application Module"
	moduleInputSummary                = "The name of the Android application module to build."
	variantInputTitle                 = "Android Application Variant"
	variantInputSummary               = "The name of the Android application variant to build."
	hasAndroidApplicationInputTitle   = "Has Android Application target?"
	HasAndroidApplicationInputSummary = "Indicates whether the project contains an Android Application target."
)

// iOS App project options
const (
	projectPathInputTitle          = "iOS Application Project or Workspace path"
	projectPathInputSummary        = "The path of iOS application Xcode project or workspace to build."
	schemeInputTitle               = "iOS Application Scheme"
	schemeInputSummary             = "The name of the iOS application scheme to build."
	distributionMethodInputTitle   = "iOS Application Distribution method"
	distributionMethodInputSummary = "The export method to use to build the iOS application IPA file."
	hasIOSApplicationInputTitle    = "Has iOS Application target?"
	hasIOSApplicationInputSummary  = "Indicates whether the project contains an iOS Application target."
)

// Config names
const (
	configName                                    = "kotlin-multiplatform-config"
	defaultConfigName                             = "default-kotlin-multiplatform-config"
	defaultConfigNameWithAndroidAndIOSApplication = "default-kotlin-multiplatform-config-android-ios"
	defaultConfigNameWithAndroidApplication       = "default-kotlin-multiplatform-config-android"
	defaultConfigNameWithIOSApplication           = "default-kotlin-multiplatform-config-ios"
)

// Workflow and Pipeline IDs
const (
	testWorkflowID         = "run_tests"
	androidBuildWorkflowID = "android_build"
	iosBuildWorkflowID     = "ios_build"
	buildPipelineID        = "build"
)

type Scanner struct {
	kmpProject *kmp.Project
}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Name() string {
	return projectType
}

func (s *Scanner) DetectPlatform(searchDir string) (bool, error) {
	log.TInfof("Searching for Gradle project files...")

	rootEntry, err := direntry.WalkDir(searchDir, 6)
	if err != nil {
		return false, err
	}

	gradleWrapperScripts := rootEntry.FindAllEntriesByName("gradlew", false)

	log.TDonef("%d Gradle wrapper script(s) found", len(gradleWrapperScripts))
	if len(gradleWrapperScripts) == 0 {
		return false, nil
	}
	gradleWrapperScript := gradleWrapperScripts[0]

	log.TInfof("Scanning project with Gradle wrapper script: %s", gradleWrapperScript.AbsPath)

	projectRootDir := gradleWrapperScript.Parent()
	if projectRootDir == nil {
		return false, fmt.Errorf("failed to get parent directory of %s", gradleWrapperScript.AbsPath)
	}
	gradleProject, err := gradle.ScanProject(*projectRootDir)
	if err != nil {
		return false, err
	}
	if gradleProject == nil {
		log.TWarnf("No Gradle project found in %s", projectRootDir.AbsPath)
		return false, nil
	}

	kmpProject, err := kmp.ScanProject(*gradleProject)
	if err != nil {
		return false, fmt.Errorf("failed to scan Kotlin Multiplatform project: %w", err)
	}

	if kmpProject == nil {
		return false, nil
	}

	printKMPProject(*kmpProject)

	s.kmpProject = kmpProject

	return true, nil
}

func (s *Scanner) ExcludedScannerNames() []string {
	return []string{
		android.ScannerName,
		string(ios.XcodeProjectTypeIOS),
		java.ProjectType,
	}
}

func (s *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	gradleProjectRootDirOption := models.NewOption(gradleProjectRootDirInputTitle, gradleProjectRootDirInputSummary, gradleProjectRootDirInputEnvKey, models.TypeSelector)

	var nextOption models.OptionNode
	var nextOptionValue string
	if s.kmpProject.AndroidAppDetectResult != nil {
		moduleOption := models.NewOption(moduleInputTitle, moduleInputSummary, android.ModuleInputEnvKey, models.TypeSelector)
		gradleProjectRootDirOption.AddOption(s.kmpProject.GradleProject.RootDirEntry.RelPath, moduleOption)

		variantOption := models.NewOption(variantInputTitle, variantInputSummary, android.VariantInputEnvKey, models.TypeOptionalUserInput)
		moduleOption.AddOption(s.kmpProject.AndroidAppDetectResult.Modules[0].ModulePath, variantOption)

		nextOption = *variantOption
		nextOptionValue = models.UserInputOptionDefaultValue
	} else {
		nextOption = *gradleProjectRootDirOption
		nextOptionValue = s.kmpProject.GradleProject.RootDirEntry.RelPath
	}

	if s.kmpProject.IOSAppDetectResult != nil {
		projectPathOption := models.NewOption(projectPathInputTitle, projectPathInputSummary, ios.ProjectPathInputEnvKey, models.TypeSelector)
		nextOption.AddOption(nextOptionValue, projectPathOption)

		schemeOption := models.NewOption(schemeInputTitle, schemeInputSummary, ios.SchemeInputEnvKey, models.TypeSelector)
		projectPathOption.AddOption(s.kmpProject.IOSAppDetectResult.Projects[0].RelPath, schemeOption)

		for _, scheme := range s.kmpProject.IOSAppDetectResult.Projects[0].Schemes {
			exportMethodOption := models.NewOption(distributionMethodInputTitle, distributionMethodInputSummary, ios.DistributionMethodEnvKey, models.TypeSelector)
			schemeOption.AddOption(scheme.Name, exportMethodOption)

			for _, exportMethod := range ios.IosExportMethods {
				configOption := models.NewConfigOption(configName, nil)
				exportMethodOption.AddConfig(exportMethod, configOption)
			}
		}
	} else {
		configOption := models.NewConfigOption(configName, nil)
		nextOption.AddConfig(models.UserInputOptionDefaultValue, configOption)
	}

	return *gradleProjectRootDirOption, nil, nil, nil
}

func (s *Scanner) DefaultOptions() models.OptionNode {
	gradleProjectRootDirOption := models.NewOption(gradleProjectRootDirInputTitle, gradleProjectRootDirInputSummary, gradleProjectRootDirInputEnvKey, models.TypeUserInput)
	hasAndroidAppTarget := models.NewOption(hasAndroidApplicationInputTitle, HasAndroidApplicationInputSummary, "", models.TypeSelector)
	gradleProjectRootDirOption.AddOption(models.UserInputOptionDefaultValue, hasAndroidAppTarget)

	// Has Android app target
	{
		moduleOption := models.NewOption(moduleInputTitle, moduleInputSummary, android.ModuleInputEnvKey, models.TypeUserInput)
		hasAndroidAppTarget.AddOption(optionValueYes, moduleOption)

		variantOption := models.NewOption(variantInputTitle, variantInputSummary, android.VariantInputEnvKey, models.TypeOptionalUserInput)
		moduleOption.AddOption("", variantOption)

		hasIosAppTarget := models.NewOption(hasIOSApplicationInputTitle, hasIOSApplicationInputSummary, "", models.TypeSelector)
		variantOption.AddOption("", hasIosAppTarget)

		// Has iOS app target
		{
			projectPathOption := models.NewOption(projectPathInputTitle, projectPathInputSummary, ios.ProjectPathInputEnvKey, models.TypeUserInput)
			hasIosAppTarget.AddOption(optionValueYes, projectPathOption)

			schemeOption := models.NewOption(schemeInputTitle, schemeInputSummary, ios.SchemeInputEnvKey, models.TypeUserInput)
			projectPathOption.AddOption("", schemeOption)

			exportMethodOption := models.NewOption(distributionMethodInputTitle, distributionMethodInputSummary, ios.DistributionMethodEnvKey, models.TypeSelector)
			schemeOption.AddOption("", exportMethodOption)

			for _, exportMethod := range ios.IosExportMethods {
				configOption := models.NewConfigOption(defaultConfigNameWithAndroidAndIOSApplication, nil)
				exportMethodOption.AddConfig(exportMethod, configOption)
			}
		}

		// Has no iOS app target
		{
			configOption := models.NewConfigOption(defaultConfigNameWithAndroidApplication, nil)
			hasIosAppTarget.AddConfig(optionValueNo, configOption)
		}
	}

	// Has no Android app target
	{
		hasIosAppTarget := models.NewOption(hasIOSApplicationInputTitle, hasIOSApplicationInputSummary, "", models.TypeSelector)
		hasAndroidAppTarget.AddOption(optionValueNo, hasIosAppTarget)

		// Has iOS app target
		{
			projectPathOption := models.NewOption(projectPathInputTitle, projectPathInputSummary, ios.ProjectPathInputEnvKey, models.TypeUserInput)
			hasIosAppTarget.AddOption(optionValueYes, projectPathOption)

			schemeOption := models.NewOption(schemeInputTitle, schemeInputSummary, ios.SchemeInputEnvKey, models.TypeUserInput)
			projectPathOption.AddOption("", schemeOption)

			exportMethodOption := models.NewOption(distributionMethodInputTitle, distributionMethodInputSummary, ios.DistributionMethodEnvKey, models.TypeSelector)
			schemeOption.AddOption("", exportMethodOption)

			for _, exportMethod := range ios.IosExportMethods {
				configOption := models.NewConfigOption(defaultConfigNameWithIOSApplication, nil)
				exportMethodOption.AddConfig(exportMethod, configOption)
			}
		}

		// Has no iOS app target
		{
			configOption := models.NewConfigOption(defaultConfigName, nil)
			hasIosAppTarget.AddConfig(optionValueNo, configOption)
		}
	}

	return *gradleProjectRootDirOption
}

func (s *Scanner) Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	bitriseDataMap := models.BitriseConfigMap{}
	configBuilder := models.NewDefaultConfigBuilder()

	// Test workflow
	{
		// Repository clone steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: sshKeyActivation,
		})...)

		// Cache setup steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.RestoreGradleCache())
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.ActivateBuildCacheForGradle())

		// Test step
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.GradleUnitTestStepListItem("$"+gradleProjectRootDirInputEnvKey))

		// Cache teardown steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.SaveGradleCache())

		// Deploy step
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.DefaultDeployStepList()...)
	}

	// Android build workflow
	if s.kmpProject.AndroidAppDetectResult != nil {
		// Repository clone steps
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: sshKeyActivation,
		})...)

		// Cache setup steps
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.RestoreGradleCache())
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.ActivateBuildCacheForGradle())

		// Build step
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.AndroidBuildStepListItem(
			envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: "$" + gradleProjectRootDirInputEnvKey},
			envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
			envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
		))
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.SignAPKStepListItem())

		// Cache teardown steps
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.SaveGradleCache())

		// Deploy step
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.DefaultDeployStepList()...)
	}

	// iOS build workflow
	if s.kmpProject.IOSAppDetectResult != nil {
		// Repository clone steps
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: sshKeyActivation,
		})...)

		// Dependency install & cache setup steps
		if s.kmpProject.IOSAppDetectResult.HasSPMDependencies {
			configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.RestoreSPMCache())
		}
		if s.kmpProject.IOSAppDetectResult.Projects[0].IsPodWorkspace {
			configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.RestoreCocoapodsCache())
			configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.CocoapodsInstallStepListItem())
		}
		if s.kmpProject.IOSAppDetectResult.Projects[0].CarthageCommand != "" {
			configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.RestoreCarthageCache())
			configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.CarthageStepListItem(
				envmanModels.EnvironmentItemModel{ios.CarthageCommandInputKey: s.kmpProject.IOSAppDetectResult.Projects[0].CarthageCommand},
			))
		}
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.RestoreGradleCache())
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.ActivateBuildCacheForGradle())

		// Build step
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.XcodeArchiveStepListItem(
			envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.DistributionMethodInputKey: "$" + ios.DistributionMethodEnvKey},
			envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
			envmanModels.EnvironmentItemModel{ios.AutomaticCodeSigningInputKey: ios.AutomaticCodeSigningInputAPIKeyValue},
		))

		// Cache teardown steps
		if s.kmpProject.IOSAppDetectResult.HasSPMDependencies {
			configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.SaveSPMCache())
		}
		if s.kmpProject.IOSAppDetectResult.Projects[0].IsPodWorkspace {
			configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.SaveCocoapodsCache())
		}
		if s.kmpProject.IOSAppDetectResult.Projects[0].CarthageCommand != "" {
			configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.SaveCarthageCache())
		}
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.SaveGradleCache())

		// Deploy step
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.DefaultDeployStepList()...)
	}

	if s.kmpProject.AndroidAppDetectResult != nil && s.kmpProject.IOSAppDetectResult != nil {
		configBuilder.SetGraphPipelineWorkflowTo(buildPipelineID, androidBuildWorkflowID, bitriseModels.GraphPipelineWorkflowModel{})
		configBuilder.SetGraphPipelineWorkflowTo(buildPipelineID, iosBuildWorkflowID, bitriseModels.GraphPipelineWorkflowModel{})
	}

	config, err := configBuilder.Generate(projectType)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}

func (s *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	bitriseDataMap := models.BitriseConfigMap{}

	// No Android and no iOS config
	{
		configBuilder := models.NewDefaultConfigBuilder()

		//
		// Test workflow

		// Repository clone steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: models.SSHKeyActivationConditional,
		})...)

		// Cache setup steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.RestoreGradleCache())
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.ActivateBuildCacheForGradle())

		// Test step
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.GradleUnitTestStepListItem("$"+gradleProjectRootDirInputEnvKey))

		// Cache teardown steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.SaveGradleCache())

		// Deploy step
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.DefaultDeployStepList()...)

		config, err := configBuilder.Generate(projectType)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		bitriseDataMap[defaultConfigName] = string(data)
	}

	// Android and no iOS config
	{
		configBuilder := models.NewDefaultConfigBuilder()

		//
		// Test workflow

		// Repository clone steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: models.SSHKeyActivationConditional,
		})...)

		// Cache setup steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.RestoreGradleCache())
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.ActivateBuildCacheForGradle())

		// Test step
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.GradleUnitTestStepListItem("$"+gradleProjectRootDirInputEnvKey))

		// Cache teardown steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.SaveGradleCache())

		// Deploy step
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.DefaultDeployStepList()...)

		//
		// Android build workflow

		// Repository clone steps
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: models.SSHKeyActivationConditional,
		})...)

		// Cache setup steps
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.RestoreGradleCache())
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.ActivateBuildCacheForGradle())

		// Build step
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.AndroidBuildStepListItem(
			envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: "$" + gradleProjectRootDirInputEnvKey},
			envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
			envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
		))
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.SignAPKStepListItem())

		// Cache teardown steps
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.SaveGradleCache())

		// Deploy step
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.DefaultDeployStepList()...)

		config, err := configBuilder.Generate(projectType)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		bitriseDataMap[defaultConfigNameWithAndroidApplication] = string(data)
	}

	// iOS and no Android config
	{
		configBuilder := models.NewDefaultConfigBuilder()

		//
		// Test workflow

		// Repository clone steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: models.SSHKeyActivationConditional,
		})...)

		// Cache setup steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.RestoreGradleCache())
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.ActivateBuildCacheForGradle())

		// Test step
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.GradleUnitTestStepListItem("$"+gradleProjectRootDirInputEnvKey))

		// Cache teardown steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.SaveGradleCache())

		// Deploy step
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.DefaultDeployStepList()...)

		//
		// iOS build workflow

		// Repository clone steps
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: models.SSHKeyActivationConditional,
		})...)

		// Cache setup step
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.RestoreGradleCache())
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.ActivateBuildCacheForGradle())

		// Build step
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.XcodeArchiveStepListItem(
			envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.DistributionMethodInputKey: "$" + ios.DistributionMethodEnvKey},
			envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
			envmanModels.EnvironmentItemModel{ios.AutomaticCodeSigningInputKey: ios.AutomaticCodeSigningInputAPIKeyValue},
		))

		// Cache teardown steps
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.SaveGradleCache())

		// Deploy step
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.DefaultDeployStepList()...)

		config, err := configBuilder.Generate(projectType)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		bitriseDataMap[defaultConfigNameWithIOSApplication] = string(data)
	}

	// Android and iOS config
	{
		configBuilder := models.NewDefaultConfigBuilder()

		//
		// Test workflow

		// Repository clone steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: models.SSHKeyActivationConditional,
		})...)

		// Cache setup steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.RestoreGradleCache())
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.ActivateBuildCacheForGradle())

		// Test step
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.GradleUnitTestStepListItem("$"+gradleProjectRootDirInputEnvKey))

		// Cache teardown steps
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.SaveGradleCache())

		// Deploy step
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.DefaultDeployStepList()...)

		//
		// Android build workflow

		// Repository clone steps
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: models.SSHKeyActivationConditional,
		})...)

		// Cache setup steps
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.RestoreGradleCache())
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.ActivateBuildCacheForGradle())

		// Build step
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.AndroidBuildStepListItem(
			envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: "$" + gradleProjectRootDirInputEnvKey},
			envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
			envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
		))
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.SignAPKStepListItem())

		// Cache teardown steps
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.SaveGradleCache())

		// Deploy step
		configBuilder.AppendStepListItemsTo(androidBuildWorkflowID, steps.DefaultDeployStepList()...)

		//
		// iOS build workflow

		// Repository clone steps
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: models.SSHKeyActivationConditional,
		})...)

		// Cache setup step
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.RestoreGradleCache())
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.ActivateBuildCacheForGradle())

		// Build step
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.XcodeArchiveStepListItem(
			envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.DistributionMethodInputKey: "$" + ios.DistributionMethodEnvKey},
			envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
			envmanModels.EnvironmentItemModel{ios.AutomaticCodeSigningInputKey: ios.AutomaticCodeSigningInputAPIKeyValue},
		))

		// Cache teardown steps
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.SaveGradleCache())

		// Deploy step
		configBuilder.AppendStepListItemsTo(iosBuildWorkflowID, steps.DefaultDeployStepList()...)

		//
		// iOS and Android build pipeline
		configBuilder.SetGraphPipelineWorkflowTo(buildPipelineID, androidBuildWorkflowID, bitriseModels.GraphPipelineWorkflowModel{})
		configBuilder.SetGraphPipelineWorkflowTo(buildPipelineID, iosBuildWorkflowID, bitriseModels.GraphPipelineWorkflowModel{})

		config, err := configBuilder.Generate(projectType)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		bitriseDataMap[defaultConfigNameWithAndroidAndIOSApplication] = string(data)
	}

	return bitriseDataMap, nil
}

func printKMPProject(kmpProject kmp.Project) {
	log.TPrintf("Project root dir: %s", kmpProject.GradleProject.RootDirEntry.RelPath)
	log.TPrintf("Gradle wrapper script: %s", kmpProject.GradleProject.GradlewFileEntry.RelPath)
	if kmpProject.GradleProject.ConfigDirEntry != nil {
		log.TPrintf("Gradle config dir: %s", kmpProject.GradleProject.ConfigDirEntry.RelPath)
	}
	if kmpProject.GradleProject.VersionCatalogFileEntry != nil {
		log.TPrintf("Version catalog file: %s", kmpProject.GradleProject.VersionCatalogFileEntry.RelPath)
	}
	if kmpProject.GradleProject.SettingsGradleFileEntry != nil {
		log.TPrintf("Gradle settings file: %s", kmpProject.GradleProject.SettingsGradleFileEntry.RelPath)
	}
	if len(kmpProject.GradleProject.IncludedProjects) > 0 {
		log.TPrintf("Included projects:")
		for _, includedProject := range kmpProject.GradleProject.IncludedProjects {
			log.TPrintf("- %s: %s", includedProject.Name, includedProject.BuildScriptFileEntry.RelPath)
		}
	}

	if kmpProject.IOSAppDetectResult != nil {
		log.TPrintf("iOS App target: %s", kmpProject.IOSAppDetectResult.Projects[0].RelPath)
	}
	if kmpProject.AndroidAppDetectResult != nil {
		log.TPrintf("Android App target: %s", kmpProject.AndroidAppDetectResult.Modules[0].BuildScriptPth)
	}
}
