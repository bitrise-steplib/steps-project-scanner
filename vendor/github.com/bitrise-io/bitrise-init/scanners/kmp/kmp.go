package kmp

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/detectors/direntry"
	"github.com/bitrise-io/bitrise-init/detectors/gradle"
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/scanners/java"
	"github.com/bitrise-io/bitrise-init/steps"
	"github.com/bitrise-io/go-utils/log"
)

/*
Relevant Gradle dependencies:
	plugins:
		org.jetbrains.kotlin.multiplatform -> kotlin("multiplatform")
			This plugin is used to enable Kotlin Multiplatform projects, allowing you to share code between different platforms (e.g., JVM, JS, Native).
*/

const (
	projectType       = "kotlin-multiplatform"
	configName        = "kotlin-multiplatform-config"
	defaultConfigName = "default-kotlin-multiplatform-config"
	testWorkflowID    = "run_tests"

	gradlewPathInputEnvKey  = "GRADLEW_PATH"
	gradlewPathInputTitle   = "The project's Gradle Wrapper script (gradlew) path."
	gradlewPathInputSummary = "The project's Gradle Wrapper script (gradlew) path."
)

type Scanner struct {
	gradleProject gradle.Project
}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Name() string {
	return projectType
}

func printGradleProject(gradleProject gradle.Project) {
	log.TPrintf("Project root dir: %s", gradleProject.RootDirEntry.RelPath)
	log.TPrintf("Gradle wrapper script: %s", gradleProject.GradlewFileEntry.RelPath)
	if gradleProject.ConfigDirEntry != nil {
		log.TPrintf("Gradle config dir: %s", gradleProject.ConfigDirEntry.RelPath)
	}
	if gradleProject.VersionCatalogFileEntry != nil {
		log.TPrintf("Version catalog file: %s", gradleProject.VersionCatalogFileEntry.RelPath)
	}
	if gradleProject.SettingsGradleFileEntry != nil {
		log.TPrintf("Gradle settings file: %s", gradleProject.SettingsGradleFileEntry.RelPath)
	}
	if len(gradleProject.IncludedProjects) > 0 {
		log.TPrintf("Included projects:")
		for _, includedProject := range gradleProject.IncludedProjects {
			log.TPrintf("- %s: %s", includedProject.Name, includedProject.BuildScriptFileEntry.RelPath)
		}
	}
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

	printGradleProject(*gradleProject)

	log.TInfof("Searching for Kotlin Multiplatform dependencies...")
	kotlinMultiplatformDetected, err := gradleProject.DetectAnyDependencies([]string{
		"org.jetbrains.kotlin.multiplatform",
		`kotlin("multiplatform")`,
	})
	if err != nil {
		return false, err
	}

	log.TDonef("Kotlin Multiplatform dependencies found: %v", kotlinMultiplatformDetected)
	s.gradleProject = *gradleProject

	return kotlinMultiplatformDetected, nil
}

func (s *Scanner) ExcludedScannerNames() []string {
	return []string{
		android.ScannerName,
		string(ios.XcodeProjectTypeIOS),
		java.ProjectType,
	}
}

func (s *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	gradlewPathOption := models.NewOption(gradlewPathInputTitle, gradlewPathInputSummary, gradlewPathInputEnvKey, models.TypeSelector)
	configOption := models.NewConfigOption(configName, nil)
	gradlewPathOption.AddConfig(s.gradleProject.GradlewFileEntry.RelPath, configOption)
	return *gradlewPathOption, nil, nil, nil
}

func (s *Scanner) DefaultOptions() models.OptionNode {
	gradlewPathOption := models.NewOption(gradlewPathInputTitle, gradlewPathInputSummary, gradlewPathInputEnvKey, models.TypeUserInput)
	configOption := models.NewConfigOption(defaultConfigName, nil)
	gradlewPathOption.AddConfig(models.UserInputOptionDefaultValue, configOption)
	return *gradlewPathOption
}

func (s *Scanner) Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	gradlewPath := "$" + gradlewPathInputEnvKey

	configBuilder.AppendStepListItemsTo(testWorkflowID,
		steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: sshKeyActivation})...,
	)
	configBuilder.AppendStepListItemsTo(testWorkflowID,
		steps.GradleUnitTestStepListItem(gradlewPath),
	)
	configBuilder.AppendStepListItemsTo(testWorkflowID,
		steps.DefaultDeployStepList()...,
	)

	config, err := configBuilder.Generate(projectType)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	bitriseDataMap := models.BitriseConfigMap{}
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}

func (s *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	gradlewPath := "$" + gradlewPathInputEnvKey

	configBuilder.AppendStepListItemsTo(testWorkflowID,
		steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: models.SSHKeyActivationConditional})...,
	)
	configBuilder.AppendStepListItemsTo(testWorkflowID,
		steps.GradleUnitTestStepListItem(gradlewPath),
	)
	configBuilder.AppendStepListItemsTo(testWorkflowID,
		steps.DefaultDeployStepList()...,
	)

	config, err := configBuilder.Generate(projectType)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	bitriseDataMap := models.BitriseConfigMap{}
	bitriseDataMap[defaultConfigName] = string(data)

	return bitriseDataMap, nil
}
