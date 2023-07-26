package flutter

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/steps"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-flutter/flutterproject"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	pathutilv2 "github.com/bitrise-io/go-utils/v2/pathutil"
	"gopkg.in/yaml.v2"
)

const (
	scannerName                 = "flutter"
	configName                  = "flutter-config"
	projectLocationInputKey     = "project_location"
	projectLocationInputEnvKey  = "BITRISE_FLUTTER_PROJECT_LOCATION"
	projectLocationInputTitle   = "Project location"
	projectLocationInputSummary = "The path to your Flutter project, stored as an Environment Variable. In your Workflows, you can specify paths relative to this path. You can change this at any time."
	platformInputKey            = "platform"
	platformInputTitle          = "Platform"
	platformInputSummary        = "The target platform for your first build. Your options are iOS, Android, both, or neither. You can change this in your Env Vars at any time."
	iosOutputTypeKey            = "ios_output_type"
	iosOutputTypeArchive        = "archive"
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
	flutterVersionToUse string
}

func (proj project) appPlatform() string {
	switch {
	case proj.hasAndroidProject && !proj.hasIosProject:
		return "android"
	case !proj.hasAndroidProject && proj.hasIosProject:
		return "ios"
	case proj.hasAndroidProject && proj.hasIosProject:
		return "both"
	default:
		return ""
	}
}

func (proj project) configName() string {
	name := configName
	if proj.hasTest {
		name += "-test"
	} else {
		name += "-notest"
	}
	if proj.appPlatform() != "" {
		name += "-" + proj.appPlatform()
	}
	name += fmt.Sprintf("-%d", proj.id)

	return name
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
		flutterProj, err := flutterproject.New(projectLocation, fileutil.NewFileManager(), pathutilv2.NewPathChecker())
		if err != nil {
			log.TErrorf(err.Error())
			continue
		}

		rootDir := flutterProj.RootDir()
		projectName := flutterProj.Pubspec().Name
		hasTest := flutterProj.TestDirPth() != ""
		hasIosProject := flutterProj.IOSProjectPth() != ""
		hasAndroidProject := flutterProj.AndroidProjectPth() != ""

		flutterVersion, err := flutterProj.FlutterSDKVersionToUse()
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
			flutterVersionToUse: flutterVersion,
		}

		scanner.projects = append(scanner.projects, proj)

		log.TPrintf("- Project path: %s", rootDir)
		log.TPrintf("  Project name: %s", projectName)
		log.TPrintf("  Has test: %v", hasTest)
		log.TPrintf("  Has Android project: %v", hasAndroidProject)
		log.TPrintf("  Has iOS project: %v", hasIosProject)
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
		android.ScannerName,
	}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	flutterProjectLocationOption := models.NewOption(projectLocationInputTitle, projectLocationInputSummary, projectLocationInputEnvKey, models.TypeSelector)

	for _, proj := range scanner.projects {
		configOption := models.NewConfigOption(proj.configName(), nil)
		flutterProjectLocationOption.AddConfig(proj.rootDir, configOption)
	}

	return *flutterProjectLocationOption, models.Warnings{}, nil, nil
}

var defaultProjects = []project{
	{hasTest: true, hasAndroidProject: true, hasIosProject: true},
	{hasTest: true, hasAndroidProject: false, hasIosProject: true},
	{hasTest: true, hasAndroidProject: true, hasIosProject: false},
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionNode {
	flutterProjectLocationOption := models.NewOption(projectLocationInputTitle, projectLocationInputSummary, projectLocationInputEnvKey, models.TypeUserInput)

	flutterPlatformOption := models.NewOption(platformInputTitle, platformInputSummary, "", models.TypeSelector)
	flutterProjectLocationOption.AddOption(models.UserInputOptionDefaultValue, flutterPlatformOption)

	for i, proj := range defaultProjects {
		proj.id = i
		configOption := models.NewConfigOption(proj.configName(), nil)
		flutterPlatformOption.AddConfig(proj.appPlatform(), configOption)
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

		configs[proj.configName()] = config
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
		configs[proj.configName()] = config
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
	flutterInstallStep := steps.FlutterInstallStepListItem(proj.flutterVersionToUse, false)
	deploySteps := steps.DefaultDeployStepList()

	// primary
	configBuilder.SetWorkflowDescriptionTo(models.PrimaryWorkflowID, primaryWorkflowDescription)

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, prepareSteps...)

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, flutterInstallStep)

	// restore cache is after flutter-installer, to prevent removal of pub system cache
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.RestoreDartCache())

	if proj.hasTest {
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.FlutterTestStepListItem(
			envmanModels.EnvironmentItemModel{projectLocationInputKey: "$" + projectLocationInputEnvKey},
		))
	} else {
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.FlutterAnalyzeStepListItem(
			envmanModels.EnvironmentItemModel{projectLocationInputKey: "$" + projectLocationInputEnvKey},
		))
	}

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.SaveDartCache())

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, deploySteps...)

	if proj.hasIosProject || proj.hasAndroidProject {
		// deploy
		configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, prepareSteps...)

		if proj.hasIosProject {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
		}

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, flutterInstallStep)

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.FlutterAnalyzeStepListItem(
			envmanModels.EnvironmentItemModel{projectLocationInputKey: "$" + projectLocationInputEnvKey},
		))

		if proj.hasTest {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.FlutterTestStepListItem(
				envmanModels.EnvironmentItemModel{projectLocationInputKey: "$" + projectLocationInputEnvKey},
			))
		}

		flutterBuildInputs := []envmanModels.EnvironmentItemModel{
			{projectLocationInputKey: "$" + projectLocationInputEnvKey},
			{platformInputKey: proj.appPlatform()},
		}
		if proj.hasIosProject {
			flutterBuildInputs = append(flutterBuildInputs, envmanModels.EnvironmentItemModel{iosOutputTypeKey: iosOutputTypeArchive})
		}
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.FlutterBuildStepListItem(flutterBuildInputs...))

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, deploySteps...)
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
