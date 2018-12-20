package flutter

import (
	"path/filepath"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners/android"
	"github.com/bitrise-core/bitrise-init/scanners/ios"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-tools/xcode-project/xcworkspace"
	yaml "gopkg.in/yaml.v2"
)

const (
	scannerName                = "flutter"
	configName                 = "flutter-config"
	projectLocationInputKey    = "project_location"
	defaultIOSConfiguration    = "Release"
	projectLocationInputEnvKey = "BITRISE_FLUTTER_PROJECT_LOCATION"
	projectLocationInputTitle  = "Project Location"
)

//------------------
// ScannerInterface
//------------------

// Scanner ...
type Scanner struct {
	projects []project
}

type project struct {
	path              string
	xcodeProjectPaths map[string][]string
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (Scanner) Name() string {
	return scannerName
}

func findProjectLocations(searchDir string) ([]string, error) {
	fileList, err := utility.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return nil, err
	}

	filters := []utility.FilterFunc{
		utility.BaseFilter("pubspec.yaml", true),
	}

	paths, err := utility.FilterPaths(fileList, filters...)
	if err != nil {
		return nil, err
	}

	for i, path := range paths {
		paths[i] = filepath.Dir(path)
	}

	return paths, nil
}

func findWorkspaceLocations(projectLocation string) ([]string, error) {
	fileList, err := utility.ListPathInDirSortedByComponents(projectLocation, true)
	if err != nil {
		return nil, err
	}

	for i, file := range fileList {
		fileList[i] = filepath.Join(projectLocation, file)
	}

	filters := []utility.FilterFunc{
		ios.AllowXCWorkspaceExtFilter,
		ios.AllowIsDirectoryFilter,
		ios.ForbidEmbeddedWorkspaceRegexpFilter,
		ios.ForbidGitDirComponentFilter,
		ios.ForbidPodsDirComponentFilter,
		ios.ForbidCarthageDirComponentFilter,
		ios.ForbidFramworkComponentWithExtensionFilter,
		ios.ForbidCordovaLibDirComponentFilter,
		ios.ForbidNodeModulesComponentFilter,
	}

	return utility.FilterPaths(fileList, filters...)
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	projectLocations, err := findProjectLocations(searchDir)
	if err != nil {
		return false, err
	}

	for _, projectLocation := range projectLocations {
		workspaceLocations, err := findWorkspaceLocations(projectLocation)
		if err != nil {
			return false, err
		}

		for _, workspaceLocation := range workspaceLocations {
			ws, err := xcworkspace.Open(workspaceLocation)
			if err != nil {
				return false, nil
			}
			schemeMap, err := ws.Schemes()
			if err != nil {
				return false, nil
			}

			proj := project{
				path:              projectLocation,
				xcodeProjectPaths: map[string][]string{},
			}

			for _, schemes := range schemeMap {
				for _, scheme := range schemes {
					proj.xcodeProjectPaths[workspaceLocation] = append(proj.xcodeProjectPaths[workspaceLocation], scheme.Name)
				}
			}

			scanner.projects = append(scanner.projects, proj)
		}
	}

	if len(scanner.projects) == 0 {
		return false, nil
	}

	return true, nil
}

// ExcludedScannerNames ...
func (Scanner) ExcludedScannerNames() []string {
	return []string{
		string(ios.XcodeProjectTypeIOS),
		android.ScannerName,
	}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	flutterProjectLocationOption := models.NewOption(projectLocationInputTitle, projectLocationInputEnvKey)

	for _, project := range scanner.projects {
		projectPathOption := models.NewOption(ios.ProjectPathInputTitle, ios.ProjectPathInputEnvKey)
		flutterProjectLocationOption.AddOption(project.path, projectPathOption)

		for xcodeWorkspacePath, schemes := range project.xcodeProjectPaths {
			schemeOption := models.NewOption(ios.SchemeInputTitle, ios.SchemeInputEnvKey)
			projectPathOption.AddOption(xcodeWorkspacePath, schemeOption)

			for _, scheme := range schemes {
				exportMethodOption := models.NewOption(ios.IosExportMethodInputTitle, ios.ExportMethodInputEnvKey)
				schemeOption.AddOption(scheme, exportMethodOption)

				for _, exportMethod := range ios.IosExportMethods {
					configOption := models.NewConfigOption(configName)
					exportMethodOption.AddConfig(exportMethod, configOption)
				}
			}
		}
	}

	return *flutterProjectLocationOption, nil, nil
}

// DefaultOptions ...
func (Scanner) DefaultOptions() models.OptionModel {
	flutterProjectLocationOption := models.NewOption(projectLocationInputTitle, projectLocationInputEnvKey)

	projectPathOption := models.NewOption(ios.ProjectPathInputTitle, ios.ProjectPathInputEnvKey)
	flutterProjectLocationOption.AddOption("_", projectPathOption)

	schemeOption := models.NewOption(ios.SchemeInputTitle, ios.SchemeInputEnvKey)
	projectPathOption.AddOption("_", schemeOption)

	exportMethodOption := models.NewOption(ios.IosExportMethodInputTitle, ios.ExportMethodInputEnvKey)
	schemeOption.AddOption("_", exportMethodOption)

	for _, exportMethod := range ios.IosExportMethods {
		configOption := models.NewConfigOption(configName)
		exportMethodOption.AddConfig(exportMethod, configOption)
	}

	return *flutterProjectLocationOption
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	return scanner.DefaultConfigs()
}

// DefaultConfigs ...
func (Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	// primary

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.FlutterInstallStepListItem())

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.FlutterTestStepListItem(
		envmanModels.EnvironmentItemModel{projectLocationInputKey: "$" + projectLocationInputEnvKey},
	))

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

	// deploy

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(false)...)

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.FlutterInstallStepListItem())

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.FlutterTestStepListItem(
		envmanModels.EnvironmentItemModel{projectLocationInputKey: "$" + projectLocationInputEnvKey},
	))

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.FlutterBuildStepListItem(
		envmanModels.EnvironmentItemModel{projectLocationInputKey: "$" + projectLocationInputEnvKey},
	))

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(
		envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.ExportMethodInputKey: "$" + ios.ExportMethodInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: defaultIOSConfiguration},
	))

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

	//

	config, err := configBuilder.Generate(scannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		configName: string(data),
	}, nil
}
