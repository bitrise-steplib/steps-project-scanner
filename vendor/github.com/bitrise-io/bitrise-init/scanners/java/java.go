package java

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/detectors/direntry"
	"github.com/bitrise-io/bitrise-init/detectors/gradle"
	"github.com/bitrise-io/bitrise-init/detectors/maven"
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	envmanModels "github.com/bitrise-io/envman/v2/models"
	"github.com/bitrise-io/go-utils/log"
)

const (
	ProjectType = "java"

	buildToolInputTitle   = "Build tool"
	buildToolInputSummary = "The build tool used in the project. Supported options: Gradle, Maven."
	buildToolGradle       = "Gradle"
	buildToolMaven        = "Maven"

	testWorkflowID = "run_tests"

	gradleConfigName                 = "java-gradle-config"
	defaultGradleConfigName          = "default-java-gradle-config"
	gradleProjectRootDirInputEnvKey  = "PROJECT_ROOT_DIR"
	gradleProjectRootDirInputTitle   = "The root directory of the Gradle project."
	gradleProjectRootDirInputSummary = "The root directory of the Gradle project, which contains all source files from your project, as well as Gradle files, including the Gradle Wrapper (`gradlew`) file."

	mavenConfigName                 = "java-maven-config"
	defaultMavenConfigName          = "default-java-maven-config"
	mavenProjectRootDirInputEnvKey  = "PROJECT_ROOT_DIR"
	mavenProjectRootDirInputTitle   = "The root directory of the Maven project."
	mavenProjectRootDirInputSummary = "The root directory of the Maven project, which contains all source files from your project, as well as Maven files, including the Maven Wrapper (`mvn`) file."
	mavenTestScriptTitle            = `Run Maven tests`
	mavenTestScriptContent          = `#!/usr/bin/env bash
# fail if any commands fails
set -e
# make pipelines' return status equal the last command to exit with a non-zero status, or zero if all commands exit successfully
set -o pipefail
# debug log
set -x

./mvn test
`
)

type Scanner struct {
	gradleProject *gradle.Project
	mavenProject  *maven.Project
}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Name() string {
	return ProjectType
}

func (s *Scanner) DetectPlatform(searchDir string) (bool, error) {
	log.TInfof("Searching for Gradle project files...")

	rootEntry, err := direntry.WalkDir(searchDir, 6)
	if err != nil {
		return false, err
	}

	gradleWrapperScripts := rootEntry.FindAllEntriesByName("gradlew", false)
	log.TDonef("%d Gradle wrapper script(s) found", len(gradleWrapperScripts))

	if len(gradleWrapperScripts) > 0 {
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

		if gradleProject != nil {
			s.gradleProject = gradleProject
			printGradleProject(*gradleProject)
			return true, nil
		} else {
			log.TWarnf("No Gradle project found in %s", projectRootDir.AbsPath)
		}
	}

	log.TInfof("Searching for Maven project files...")

	projectObjectModels := rootEntry.FindAllEntriesByName("pom.xml", false)
	log.TDonef("%d POM file(s) found", len(projectObjectModels))

	if len(projectObjectModels) > 0 {
		projectObjectModel := projectObjectModels[0]

		log.TInfof("Scanning project with POM file: %s", projectObjectModel.AbsPath)

		projectRootDir := projectObjectModel.Parent()
		if projectRootDir == nil {
			return false, fmt.Errorf("failed to get parent directory of %s", projectObjectModel.AbsPath)
		}
		mavenProject, err := maven.ScanProject(*projectRootDir)
		if err != nil {
			return false, err
		}

		if mavenProject != nil {
			s.mavenProject = mavenProject
			printMavenProject(*mavenProject)
			return true, nil
		} else {
			log.Warnf("No Maven project found in %s", projectRootDir.AbsPath)
		}
	}

	return false, nil
}

func (s *Scanner) ExcludedScannerNames() []string {
	return []string{}
}

func (s *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	if s.gradleProject != nil {
		gradleProjectRootDirOption := models.NewOption(gradleProjectRootDirInputTitle, gradleProjectRootDirInputSummary, gradleProjectRootDirInputEnvKey, models.TypeSelector)
		configOption := models.NewConfigOption(gradleConfigName, nil)
		gradleProjectRootDirOption.AddConfig(s.gradleProject.RootDirEntry.RelPath, configOption)
		return *gradleProjectRootDirOption, nil, nil, nil
	}

	if s.mavenProject != nil {
		mavenProjectRootDirOption := models.NewOption(mavenProjectRootDirInputTitle, mavenProjectRootDirInputSummary, mavenProjectRootDirInputEnvKey, models.TypeSelector)
		configOption := models.NewConfigOption(mavenConfigName, nil)
		mavenProjectRootDirOption.AddConfig(s.mavenProject.RootDirEntry.RelPath, configOption)
		return *mavenProjectRootDirOption, nil, nil, nil
	}

	return models.OptionNode{}, nil, nil, nil
}

func (s *Scanner) DefaultOptions() models.OptionNode {
	buildToolOption := models.NewOption(buildToolInputTitle, buildToolInputSummary, "", models.TypeSelector)

	gradleProjectRootDirOption := models.NewOption(gradleProjectRootDirInputTitle, gradleProjectRootDirInputSummary, gradleProjectRootDirInputEnvKey, models.TypeUserInput)
	buildToolOption.AddOption(buildToolGradle, gradleProjectRootDirOption)

	gradleConfigOption := models.NewConfigOption(defaultGradleConfigName, nil)
	gradleProjectRootDirOption.AddConfig(models.UserInputOptionDefaultValue, gradleConfigOption)

	mavenProjectRootDirOption := models.NewOption(mavenProjectRootDirInputTitle, mavenProjectRootDirInputSummary, mavenProjectRootDirInputEnvKey, models.TypeUserInput)
	buildToolOption.AddOption(buildToolMaven, mavenProjectRootDirOption)

	mavenConfigOption := models.NewConfigOption(defaultMavenConfigName, nil)
	mavenProjectRootDirOption.AddConfig(models.UserInputOptionDefaultValue, mavenConfigOption)

	return *buildToolOption
}

func (s *Scanner) Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()
	bitriseDataMap := models.BitriseConfigMap{}

	if s.gradleProject != nil {
		gradleProjectRootDir := "$" + gradleProjectRootDirInputEnvKey
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: sshKeyActivation})...,
		)
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.GradleUnitTestStepListItem(gradleProjectRootDir),
		)
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.DefaultDeployStepList()...,
		)

		config, err := configBuilder.Generate(ProjectType)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		bitriseDataMap[gradleConfigName] = string(data)
	}

	if s.mavenProject != nil {
		mavenProjectRootDir := "$" + mavenProjectRootDirInputEnvKey
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: sshKeyActivation})...,
		)
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.ScriptStepListItem(mavenTestScriptTitle, mavenTestScriptContent, envmanModels.EnvironmentItemModel{
				"working_dir": mavenProjectRootDir,
			}),
		)
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.DefaultDeployStepList()...,
		)
		config, err := configBuilder.Generate(ProjectType)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}
		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		bitriseDataMap[mavenConfigName] = string(data)
	}

	return bitriseDataMap, nil
}

func (s *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	bitriseDataMap := models.BitriseConfigMap{}

	{
		configBuilder := models.NewDefaultConfigBuilder()

		gradleProjectRootDir := "$" + gradleProjectRootDirInputEnvKey
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: models.SSHKeyActivationConditional})...,
		)
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.GradleUnitTestStepListItem(gradleProjectRootDir),
		)
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.DefaultDeployStepList()...,
		)

		config, err := configBuilder.Generate(ProjectType)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}
		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}
		bitriseDataMap[defaultGradleConfigName] = string(data)
	}

	{
		configBuilder := models.NewDefaultConfigBuilder()

		mavenProjectRootDir := "$" + mavenProjectRootDirInputEnvKey
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: models.SSHKeyActivationConditional})...,
		)
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.ScriptStepListItem(mavenTestScriptTitle, mavenTestScriptContent, envmanModels.EnvironmentItemModel{
				"working_dir": mavenProjectRootDir,
			}),
		)
		configBuilder.AppendStepListItemsTo(testWorkflowID,
			steps.DefaultDeployStepList()...,
		)

		config, err := configBuilder.Generate(ProjectType)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}
		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}
		bitriseDataMap[defaultMavenConfigName] = string(data)
	}

	return bitriseDataMap, nil
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

func printMavenProject(mavenProject maven.Project) {
	log.TPrintf("Project root dir: %s", mavenProject.RootDirEntry.RelPath)
	log.TPrintf("Maven POM file: %s", mavenProject.ProjectObjectModelFileEntry.RelPath)
	log.TPrintf("Maven wrapper file: %s", mavenProject.MavenWrapperFileEntry.RelPath)
}
