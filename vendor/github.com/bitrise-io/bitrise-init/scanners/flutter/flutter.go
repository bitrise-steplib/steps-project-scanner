package flutter

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/scanners/java"
	"github.com/bitrise-io/bitrise-init/steps"
	envmanModels "github.com/bitrise-io/envman/v2/models"
	"github.com/bitrise-io/go-flutter/flutterproject"
	"github.com/bitrise-io/go-flutter/fluttersdk"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	pathutilv2 "github.com/bitrise-io/go-utils/v2/pathutil"
	"gopkg.in/yaml.v2"
)

const (
	scannerName                 = "flutter"
	configName                  = "flutter-config"
	testWorkflowID              = "run_tests"
	buildWorkflowID             = "build_app"
	projectLocationInputKey     = "project_location"
	projectLocationInputEnvKey  = "BITRISE_FLUTTER_PROJECT_LOCATION"
	projectLocationInputTitle   = "Project location"
	projectLocationInputSummary = "The path to your Flutter project, stored as an Environment Variable. In your Workflows, you can specify paths relative to this path. You can change this at any time."
	platformInputKey            = "platform"
	iosOutputTypeKey            = "ios_output_type"
	iosOutputTypeArchive        = "archive"
)

const (
	testWorkflowDescription = `Runs tests or analysis.

Runs flutter-test if a test directory is present, otherwise runs flutter-analyze.

Next steps:
- Check out [Getting started with Flutter apps](https://docs.bitrise.io/en/bitrise-ci/getting-started/quick-start-guides/getting-started-with-flutter-projects.html).
`

	buildAppWorkflowDescription = `Builds and deploys app using [Deploy to bitrise.io Step](https://docs.bitrise.io/en/bitrise-ci/getting-started/quick-start-guides/getting-started-with-flutter-projects.html#deploying-a-flutter-app).

If you build for iOS, make sure to set up code signing secrets on Bitrise for a successful build.

Next steps:
- Check out [Getting started with Flutter apps](https://docs.bitrise.io/en/bitrise-ci/getting-started/quick-start-guides/getting-started-with-flutter-projects.html) for signing and deployment options.
- Check out the Code signing guide for [iOS](https://docs.bitrise.io/en/bitrise-ci/code-signing/ios-code-signing.html) and [Android](https://docs.bitrise.io/en/bitrise-ci/code-signing/android-code-signing.html).
`
)

//------------------
// ScannerInterface
//------------------

type project struct {
	id                  int
	rootDir             string
	hasTest             bool
	hasIosProject       bool
	hasAndroidProject   bool
	hasWebProject       bool
	flutterVersionToUse string
}

// Scanner ...
type Scanner struct {
	projects []project
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (scanner *Scanner) Name() string {
	return scannerName
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	log.TInfof("Search for project(s)")
	projectLocations, err := findProjectLocations(searchDir)
	if err != nil {
		return false, err
	}

	log.TPrintf("Paths containing pubspec.yaml(%d):", len(projectLocations))
	for _, p := range projectLocations {
		log.TPrintf("- %s", p)
	}
	log.TPrintf("")

	log.TInfof("Fetching pubspec.yaml files")

	currentID := -1
	for _, projectLocation := range projectLocations {
		flutterProj, err := flutterproject.New(projectLocation, fileutil.NewFileManager(), pathutilv2.NewPathChecker(), fluttersdk.NewSDKVersionFinder())
		if err != nil {
			log.TErrorf(err.Error())
			continue
		}

		rootDir := flutterProj.RootDir()
		projectName := flutterProj.Pubspec().Name
		hasTest := flutterProj.TestDirPth() != ""
		hasIosProject := flutterProj.IOSProjectPth() != ""
		hasAndroidProject := flutterProj.AndroidProjectPth() != ""
		hasWebProject := flutterProj.WebProjectDir() != ""

		// TODO: The second return value (flutterChannel) is omitted,
		//  because the Flutter Installer step is not able to install a Flutter SDK version from a specific channel.
		//  This is not a huge issue, because just a few SDK versions are available on multiple channels (like 2.2.2).
		flutterVersion, _, err := flutterProj.FlutterSDKVersionToUse()
		if err != nil {
			log.Warnf(err.Error())
		}

		currentID++
		proj := project{
			id:                  currentID,
			rootDir:             rootDir,
			hasTest:             hasTest,
			hasIosProject:       hasIosProject,
			hasAndroidProject:   hasAndroidProject,
			hasWebProject:       hasWebProject,
			flutterVersionToUse: flutterVersion,
		}

		scanner.projects = append(scanner.projects, proj)

		log.TPrintf("- Project path: %s", rootDir)
		log.TPrintf("  Project name: %s", projectName)
		log.TPrintf("  Has test: %v", hasTest)
		log.TPrintf("  Has Android project: %v", hasAndroidProject)
		log.TPrintf("  Has iOS project: %v", hasIosProject)
		log.TPrintf("  Has Web project: %v", hasWebProject)
		if flutterVersion != "" {
			log.TPrintf("  Flutter version to use: %s", proj.flutterVersionToUse)
		}
	}

	return len(scanner.projects) > 0, nil
}

// ExcludedScannerNames ...
func (scanner *Scanner) ExcludedScannerNames() []string {
	return []string{
		string(ios.XcodeProjectTypeIOS),
		string(ios.XcodeProjectTypeMacOS),
		android.ScannerName,
		java.ProjectType,
	}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	flutterProjectLocationOption := models.NewOption(projectLocationInputTitle, projectLocationInputSummary, projectLocationInputEnvKey, models.TypeSelector)

	for _, proj := range scanner.projects {
		configOption := models.NewConfigOption(configNameFor(proj), nil)
		flutterProjectLocationOption.AddConfig(proj.rootDir, configOption)
	}

	return *flutterProjectLocationOption, models.Warnings{}, nil, nil
}

var defaultProjects = []project{
	{hasTest: true, hasAndroidProject: true, hasIosProject: true, hasWebProject: true},
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionNode {
	flutterProjectLocationOption := models.NewOption(projectLocationInputTitle, projectLocationInputSummary, projectLocationInputEnvKey, models.TypeUserInput)

	for i, proj := range defaultProjects {
		proj.id = i
		configOption := models.NewConfigOption(configNameFor(proj), nil)
		flutterProjectLocationOption.AddConfig(models.UserInputOptionDefaultValue, configOption)
	}

	return *flutterProjectLocationOption
}

func (scanner *Scanner) Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	configs := models.BitriseConfigMap{}

	for _, proj := range scanner.projects {
		config, err := generateConfig(sshKeyActivation, proj)
		if err != nil {
			return nil, err
		}

		configs[configNameFor(proj)] = config
	}

	return configs, nil
}

func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configs := models.BitriseConfigMap{}

	for i, proj := range defaultProjects {
		proj.id = i

		config, err := generateConfig(models.SSHKeyActivationConditional, proj)
		if err != nil {
			return nil, err
		}
		configs[configNameFor(proj)] = config
	}

	return configs, nil
}

func findProjectLocations(searchDir string) ([]string, error) {
	fileList, err := pathutil.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return nil, err
	}

	filters := []pathutil.FilterFunc{
		pathutil.BaseFilter("pubspec.yaml", true),
		pathutil.ComponentFilter("node_modules", false),
	}

	paths, err := pathutil.FilterPaths(fileList, filters...)
	if err != nil {
		return nil, err
	}

	for i, path := range paths {
		paths[i] = filepath.Dir(path)
	}

	return paths, nil
}

func generateConfig(sshKeyActivation models.SSHKeyActivation, proj project) (string, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	// Common steps to all workflows
	prepareSteps := steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: sshKeyActivation})
	flutterInstallStep := steps.FlutterInstallStepListItem(proj.flutterVersionToUse)
	deploySteps := steps.DefaultDeployStepList()

	// primary
	configBuilder.SetWorkflowDescriptionTo(testWorkflowID, testWorkflowDescription)

	configBuilder.AppendStepListItemsTo(testWorkflowID, prepareSteps...)

	configBuilder.AppendStepListItemsTo(testWorkflowID, flutterInstallStep)

	// restore cache is after flutter-installer, to prevent removal of pub system cache
	configBuilder.AppendStepListItemsTo(testWorkflowID, steps.RestoreDartCache())

	if proj.hasTest {
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.FlutterTestStepListItem(
			envmanModels.EnvironmentItemModel{projectLocationInputKey: "$" + projectLocationInputEnvKey},
		))
	} else {
		configBuilder.AppendStepListItemsTo(testWorkflowID, steps.FlutterAnalyzeStepListItem(
			envmanModels.EnvironmentItemModel{projectLocationInputKey: "$" + projectLocationInputEnvKey},
		))
	}

	configBuilder.AppendStepListItemsTo(testWorkflowID, steps.SaveDartCache())

	configBuilder.AppendStepListItemsTo(testWorkflowID, deploySteps...)

	if proj.hasIosProject || proj.hasAndroidProject {
		// deploy
		configBuilder.SetWorkflowDescriptionTo(buildWorkflowID, buildAppWorkflowDescription)

		configBuilder.AppendStepListItemsTo(buildWorkflowID, prepareSteps...)

		if proj.hasIosProject {
			configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
		}

		configBuilder.AppendStepListItemsTo(buildWorkflowID, flutterInstallStep)

		configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.FlutterAnalyzeStepListItem(
			envmanModels.EnvironmentItemModel{projectLocationInputKey: "$" + projectLocationInputEnvKey},
		))

		if proj.hasTest {
			configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.FlutterTestStepListItem(
				envmanModels.EnvironmentItemModel{projectLocationInputKey: "$" + projectLocationInputEnvKey},
			))
		}

		flutterBuildInputs := []envmanModels.EnvironmentItemModel{
			{projectLocationInputKey: "$" + projectLocationInputEnvKey},
			{platformInputKey: targetPlatformInputValueFor(proj)},
		}
		if proj.hasIosProject {
			flutterBuildInputs = append(flutterBuildInputs, envmanModels.EnvironmentItemModel{iosOutputTypeKey: iosOutputTypeArchive})
		}
		configBuilder.AppendStepListItemsTo(buildWorkflowID, steps.FlutterBuildStepListItem(flutterBuildInputs...))

		configBuilder.AppendStepListItemsTo(buildWorkflowID, deploySteps...)
	}

	config, err := configBuilder.Generate(scannerName)
	if err != nil {
		return "", err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func targetPlatformInputValueFor(proj project) string {
	switch {
	case proj.hasIosProject && proj.hasAndroidProject:
		return "both"
	case proj.hasIosProject:
		return "ios"
	case proj.hasAndroidProject:
		return "android"
	default:
		return ""
	}
}

func configNameFor(proj project) string {
	name := configName
	if proj.hasTest {
		name += "-test"
	} else {
		name += "-notest"
	}
	if proj.hasIosProject {
		name += "-ios"
	}
	if proj.hasAndroidProject {
		name += "-android"
	}
	if proj.hasWebProject {
		name += "-web"
	}
	name += fmt.Sprintf("-%d", proj.id)

	return name
}
