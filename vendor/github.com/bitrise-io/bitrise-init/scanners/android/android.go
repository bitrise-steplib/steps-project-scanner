package android

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/analytics"
	"github.com/bitrise-io/bitrise-init/detectors/direntry"
	"github.com/bitrise-io/bitrise-init/detectors/gradle"
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/java"
	"github.com/bitrise-io/bitrise-init/steps"
	bitriseModels "github.com/bitrise-io/bitrise/v2/models"
	envmanModels "github.com/bitrise-io/envman/v2/models"
	"github.com/bitrise-io/go-utils/log"
)

/*
Relevant Gradle dependencies:
	plugins:
		com.android.application
			This plugin is used to configure and build Android application projects.
*/

const (
	ScannerName                   = "android"
	ConfigName                    = "android-config"
	ConfigNameKotlinScript        = "android-config-kts"
	DefaultConfigName             = "default-android-config"
	DefaultConfigNameKotlinScript = "default-android-config-kts"

	testsWorkflowID         = "run_tests"
	testsWorkflowSummary    = "Run your Android unit tests and get the test report."
	testWorkflowDescription = "The workflow will first clone your Git repository, cache your Gradle dependencies, install Android tools, run your Android unit tests and save the test report."

	testPipelineID = "run_tests"

	runInstrumentedTestsWorkflowID          = "run_instrumented_tests"
	runInstrumentedTestsWorkflowSummary     = "Run your Android instrumented tests and get the test report."
	runInstrumentedTestsWorkflowDescription = "The workflow will first clone your Git repository, cache your Gradle dependencies, install Android tools, run your Android instrumented tests and save the test report."
	TestShardCountEnvKey                    = "TEST_SHARD_COUNT"
	TestShardCountEnvValue                  = 2
	ParallelTotalEnvKey                     = "BITRISE_IO_PARALLEL_TOTAL"
	ParallelIndexEnvKey                     = "BITRISE_IO_PARALLEL_INDEX"

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

	BuildScriptInputTitle   = "Does your app use Kotlin build scripts?"
	BuildScriptInputSummary = "The workflow configuration slightly differs based on what language (Groovy or Kotlin) you used in your build scripts."

	GradlewPathInputKey = "gradlew_path"

	CacheLevelInputKey = "cache_level"
	CacheLevelNone     = "none"

	gradleKotlinBuildFile = "build.gradle.kts"
)

type gradleModule struct {
	ModulePath     string
	BuildScriptPth string
	UsesKotlinDSL  bool
}

// Scanner ...
type Scanner struct {
	Results []DetectResult
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
	return []string{java.ProjectType}
}

type DetectResult struct {
	GradleProject gradle.Project
	Modules       []gradleModule
	Icons         models.Icons
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (_ bool, err error) {
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

	var results []DetectResult
	for i, gradleWrapperScript := range gradleWrapperScripts {
		if i > 0 {
			log.TPrintf("")
		}
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
			continue
		}

		printGradleProject(*gradleProject)

		if len(gradleProject.AllBuildScriptFileEntries) == 0 {
			analytics.LogInfo("android-no-build-scripts-found", nil, "no build script files found")
			return false, fmt.Errorf("no Gradle build script file found")
		}

		log.TPrintf("Searching for Android dependencies...")
		androidDetected, err := gradleProject.DetectAnyDependencies([]string{
			"com.android.application",
		})
		if err != nil {
			return false, err
		}

		log.TDonef("Android dependencies found: %v", androidDetected)
		if !androidDetected {
			log.TDonef("No Android dependencies found, skipping this project")
			continue
		}

		result := DetectResult{
			GradleProject: *gradleProject,
		}

		if gradleProject.SettingsGradleFileEntry != nil && len(gradleProject.IncludedProjects) == 0 {
			log.TWarnf("No included projects found in settings.gradle file")
			remoteLogNoIncludedProjectsFound(gradleProject.SettingsGradleFileEntry.AbsPath)
		}

		log.TPrintf("Scanning Gradle modules...")
		var modules []gradleModule
		if len(gradleProject.IncludedProjects) > 0 {
			for _, includedProject := range gradleProject.IncludedProjects {
				modulePath := modulePathFromBuildScriptPath(gradleProject.RootDirEntry.RelPath, includedProject.BuildScriptFileEntry.RelPath)
				modules = append(modules, gradleModule{
					ModulePath:     modulePath,
					BuildScriptPth: includedProject.BuildScriptFileEntry.RelPath,
					UsesKotlinDSL:  strings.HasSuffix(includedProject.BuildScriptFileEntry.RelPath, ".kts"),
				})
			}

			log.TDonef("%d included module(s) found:", len(modules))
			for _, module := range modules {
				log.TPrintf("- %s", module.ModulePath)
			}
		} else {
			for _, buildScript := range gradleProject.AllBuildScriptFileEntries {
				modulePath := modulePathFromBuildScriptPath(gradleProject.RootDirEntry.RelPath, buildScript.RelPath)
				if modulePath == "" {
					// Skipp top-level build script file
					continue
				}
				modules = append(modules, gradleModule{
					ModulePath:     modulePath,
					BuildScriptPth: buildScript.RelPath,
					UsesKotlinDSL:  strings.HasSuffix(buildScript.RelPath, ".kts"),
				})
			}

			log.TDonef("%d module(s) found:", len(modules))
			for _, module := range modules {
				log.TPrintf("- %s", module.ModulePath)
			}
		}
		result.Modules = modules

		log.TPrintf("Searching for project icons...")
		result.Icons, err = LookupIcons(result.GradleProject.RootDirEntry.AbsPath, searchDir)
		if err != nil {
			log.TWarnf("Failed to find icons: %v", err)
			analytics.LogInfo("android-icon-lookup", analytics.DetectorErrorData("android", err), "Failed to lookup android icon")
		}
		log.TDonef("%d icon(s) found", len(result.Icons))

		results = append(results, result)
	}

	if len(results) == 0 {
		log.TDonef("No Android projects found")
		return false, nil
	}

	scanner.Results = results

	return len(results) > 0, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	projectLocationOption := models.NewOption(ProjectLocationInputTitle, ProjectLocationInputSummary, ProjectLocationInputEnvKey, models.TypeSelector)
	var allIcons models.Icons

	for _, result := range scanner.Results {
		moduleOption := models.NewOption(ModuleInputTitle, ModuleInputSummary, ModuleInputEnvKey, models.TypeUserInput)
		variantOption := models.NewOption(VariantInputTitle, VariantInputSummary, VariantInputEnvKey, models.TypeOptionalUserInput)

		iconIDs := make([]string, len(result.Icons))
		for i, icon := range result.Icons {
			iconIDs[i] = icon.Filename
		}
		allIcons = append(allIcons, result.Icons...)

		for _, module := range result.Modules {
			var configOption *models.OptionNode
			if module.UsesKotlinDSL {
				configOption = models.NewConfigOption(ConfigNameKotlinScript, iconIDs)
			} else {
				configOption = models.NewConfigOption(ConfigName, iconIDs)
			}

			projectLocationOption.AddOption(result.GradleProject.RootDirEntry.RelPath, moduleOption)
			moduleOption.AddOption(module.ModulePath, variantOption)
			variantOption.AddConfig("", configOption)
		}
	}

	return *projectLocationOption, nil, allIcons, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionNode {
	projectLocationOption := models.NewOption(ProjectLocationInputTitle, ProjectLocationInputSummary, ProjectLocationInputEnvKey, models.TypeUserInput)
	moduleOption := models.NewOption(ModuleInputTitle, ModuleInputSummary, ModuleInputEnvKey, models.TypeUserInput)
	variantOption := models.NewOption(VariantInputTitle, VariantInputSummary, VariantInputEnvKey, models.TypeOptionalUserInput)

	buildScriptOption := models.NewOption(BuildScriptInputTitle, BuildScriptInputSummary, "", models.TypeSelector)
	regularConfigOption := models.NewConfigOption(DefaultConfigName, nil)
	kotlinScriptConfigOption := models.NewConfigOption(DefaultConfigNameKotlinScript, nil)

	projectLocationOption.AddOption(models.UserInputOptionDefaultValue, moduleOption)
	moduleOption.AddOption(models.UserInputOptionDefaultValue, variantOption)
	variantOption.AddOption(models.UserInputOptionDefaultValue, buildScriptOption)

	buildScriptOption.AddConfig("yes", kotlinScriptConfigOption)
	buildScriptOption.AddOption("no", regularConfigOption)

	return *projectLocationOption
}

type configBuildingParams struct {
	name            string
	useKotlinScript bool
}

// Configs ...
func (scanner *Scanner) Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	var usesGradleDSL, usesKotlinDSL bool
	for _, result := range scanner.Results {
		for _, module := range result.Modules {
			if module.UsesKotlinDSL {
				usesKotlinDSL = true
			} else {
				usesGradleDSL = true
			}
		}
	}

	var params []configBuildingParams
	if usesGradleDSL {
		params = append(params, configBuildingParams{
			name:            ConfigName,
			useKotlinScript: false,
		})
	}
	if usesKotlinDSL {
		params = append(params, configBuildingParams{
			name:            ConfigNameKotlinScript,
			useKotlinScript: true,
		})
	}
	return scanner.generateConfigs(sshKeyActivation, params)
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	params := []configBuildingParams{
		{name: DefaultConfigName, useKotlinScript: false},
		{name: DefaultConfigNameKotlinScript, useKotlinScript: true},
	}
	return scanner.generateConfigs(models.SSHKeyActivationConditional, params)
}

func (scanner *Scanner) generateConfigs(sshKeyActivation models.SSHKeyActivation, params []configBuildingParams) (models.BitriseConfigMap, error) {
	bitriseDataMap := models.BitriseConfigMap{}

	for _, param := range params {
		configBuilder := scanner.generateConfigBuilder(sshKeyActivation, param.useKotlinScript)

		config, err := configBuilder.Generate(ScannerName,
			envmanModels.EnvironmentItemModel{TestShardCountEnvKey: TestShardCountEnvValue},
		)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		bitriseDataMap[param.name] = string(data)
	}

	return bitriseDataMap, nil
}

func (scanner *Scanner) generateConfigBuilder(sshKeyActivation models.SSHKeyActivation, useKotlinBuildScript bool) models.ConfigBuilderModel {
	configBuilder := models.NewDefaultConfigBuilder()

	projectLocationEnv, gradlewPath, moduleEnv, variantEnv := "$"+ProjectLocationInputEnvKey, "$"+ProjectLocationInputEnvKey+"/gradlew", "$"+ModuleInputEnvKey, "$"+VariantInputEnvKey

	//-- test
	configBuilder.AppendStepListItemsTo(testsWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		SSHKeyActivation: sshKeyActivation})...)
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

	//-- instrumented test
	configBuilder.AppendStepListItemsTo(runInstrumentedTestsWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		SSHKeyActivation: sshKeyActivation,
	})...)
	configBuilder.AppendStepListItemsTo(runInstrumentedTestsWorkflowID, steps.RestoreGradleCache())
	configBuilder.AppendStepListItemsTo(runInstrumentedTestsWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
		envmanModels.EnvironmentItemModel{GradlewPathInputKey: gradlewPath},
	))
	configBuilder.AppendStepListItemsTo(runInstrumentedTestsWorkflowID, steps.AvdManagerStepListItem())
	configBuilder.AppendStepListItemsTo(runInstrumentedTestsWorkflowID, steps.WaitForAndroidEmulatorStepListItem())
	configBuilder.AppendStepListItemsTo(runInstrumentedTestsWorkflowID, steps.GradleRunnerStepListItem(
		gradlewPath,
		fmt.Sprintf("connectedAndroidTest \\\n  -Pandroid.testInstrumentationRunnerArguments.numShards=$%s \\\n  -Pandroid.testInstrumentationRunnerArguments.shardIndex=$%s",
			ParallelTotalEnvKey,
			ParallelIndexEnvKey,
		),
	))
	configBuilder.AppendStepListItemsTo(runInstrumentedTestsWorkflowID, steps.SaveGradleCache())
	configBuilder.AppendStepListItemsTo(runInstrumentedTestsWorkflowID, steps.DefaultDeployStepList()...)
	configBuilder.SetWorkflowSummaryTo(runInstrumentedTestsWorkflowID, runInstrumentedTestsWorkflowSummary)
	configBuilder.SetWorkflowDescriptionTo(runInstrumentedTestsWorkflowID, runInstrumentedTestsWorkflowDescription)

	configBuilder.SetGraphPipelineWorkflowTo(testPipelineID, runInstrumentedTestsWorkflowID, bitriseModels.GraphPipelineWorkflowModel{
		Parallel: "$" + TestShardCountEnvKey,
	})

	//-- build
	configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		SSHKeyActivation: sshKeyActivation,
	})...)
	configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
		envmanModels.EnvironmentItemModel{GradlewPathInputKey: gradlewPath},
	))

	basePath := filepath.Join(projectLocationEnv, moduleEnv)
	path := filepath.Join(basePath, "build.gradle")
	if useKotlinBuildScript {
		path = filepath.Join(basePath, gradleKotlinBuildFile)
	}
	configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.ChangeAndroidVersionCodeAndVersionNameStepListItem(
		envmanModels.EnvironmentItemModel{ModuleBuildGradlePathInputKey: path},
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

func modulePathFromBuildScriptPath(projectRootDir, buildScriptPth string) string {
	relBuildScriptPath := strings.TrimPrefix(buildScriptPth, projectRootDir)
	relBuildScriptPath = strings.TrimPrefix(relBuildScriptPath, "/")
	pathComponents := strings.Split(relBuildScriptPath, "/")
	if len(pathComponents) < 2 {
		return ""
	}

	return strings.Join(pathComponents[:len(pathComponents)-1], "/")
}

func remoteLogNoIncludedProjectsFound(settingGradlePth string) {
	file, err := os.Open(settingGradlePth)
	if err != nil {
		analytics.LogInfo("android-no-included-projects", map[string]interface{}{
			"error": err.Error(),
		}, "Failed to open settings.gradle file")
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.TWarnf("Unable to close file %s: %s", settingGradlePth, err)
		}
	}()

	var includeLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "include") {
			includeLines = append(includeLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		analytics.LogInfo("android-no-included-projects", map[string]interface{}{
			"error": err.Error(),
		}, "Failed to read settings.gradle file")
		return
	}

	analytics.LogInfo("android-no-included-projects", map[string]interface{}{
		"include_lines": strings.Join(includeLines, "\n"),
	}, "settings.gradle file exists, but no included projects found")
}
