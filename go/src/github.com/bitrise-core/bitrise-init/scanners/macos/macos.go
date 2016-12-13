package macos

import (
	"errors"
	"fmt"
	"path/filepath"

	yaml "gopkg.in/yaml.v1"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-tools/go-xcode/xcodeproj"
)

var (
	log = utility.NewLogger()
)

const scannerName = "macos"

const defaultConfigName = "default-macos-config"

const (
	projectPathKey    = "project_path"
	projectPathTitle  = "Project (or Workspace) path"
	projectPathEnvKey = "BITRISE_PROJECT_PATH"

	schemeKey    = "scheme"
	schemeTitle  = "Scheme name"
	schemeEnvKey = "BITRISE_SCHEME"
)

// ConfigDescriptor ...
type ConfigDescriptor struct {
	HasPodfile           bool
	HasTest              bool
	MissingSharedSchemes bool
}

func (descriptor ConfigDescriptor) String() string {
	name := "macos-"
	if descriptor.HasPodfile {
		name = name + "pod-"
	}
	if descriptor.HasTest {
		name = name + "test-"
	}
	if descriptor.MissingSharedSchemes {
		name = name + "missing-shared-schemes-"
	}
	return name + "config"
}

// Scanner ...
type Scanner struct {
	fileList []string

	xcodeProjectAndWorkspaceFiles []string

	configDescriptors []ConfigDescriptor
}

// Name ...
func (scanner Scanner) Name() string {
	return scannerName
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	fileList, err := utility.FileList(searchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}
	scanner.fileList = fileList

	// Search for xcodeproj and xcworkspace files
	log.Info("Searching for macOS .xcodeproj & .xcworkspace files")

	relevantXcodeProjectFiles, err := utility.FilterRelevantXcodeProjectFiles(fileList, false)
	if err != nil {
		return false, fmt.Errorf("failed to collect .xcodeproj & .xcworkspace files, error: %s", err)
	}

	if len(relevantXcodeProjectFiles) == 0 {
		log.Details("platform not detected")
		return false, nil
	}

	// Separate xcodeproj and xcworkspace files
	projects := []string{}
	workspaces := []string{}

	for _, projectOrWorkspace := range relevantXcodeProjectFiles {
		if xcodeproj.IsXCodeProj(projectOrWorkspace) {
			projects = append(projects, projectOrWorkspace)
		} else {
			workspaces = append(workspaces, projectOrWorkspace)
		}
	}

	// Filter xcodeproj and xcworkspace files with iphoneos sdk
	macosxXcodeProjectFileMap := map[string]bool{}

	for _, project := range projects {
		pbxprojPth := filepath.Join(project, "project.pbxproj")
		sdks, err := xcodeproj.GetBuildConfigSDKs(pbxprojPth)
		if err != nil {
			return false, err
		}
		for _, sdk := range sdks {
			if sdk == "macosx" {
				macosxXcodeProjectFileMap[project] = true
			}
		}
	}

	for _, workspace := range workspaces {
		referredProjects, err := xcodeproj.WorkspaceProjectReferences(workspace)
		if err != nil {
			return false, err
		}

		// Only deal with relevant projects
		filteredProjects := []string{}
		for _, project := range projects {
			for _, projectToCheck := range projects {
				if project == projectToCheck {
					filteredProjects = append(filteredProjects, project)
				}
			}
		}
		referredProjects = filteredProjects
		// ---

		for _, project := range referredProjects {
			pbxprojPth := filepath.Join(project, "project.pbxproj")
			sdks, err := xcodeproj.GetBuildConfigSDKs(pbxprojPth)
			if err != nil {
				return false, err
			}
			for _, sdk := range sdks {
				if sdk == "macosx" {
					macosxXcodeProjectFileMap[project] = true
				}
			}
		}
	}

	if len(macosxXcodeProjectFileMap) == 0 {
		log.Details("platform not detected")
		return false, nil
	}

	log.Details("")
	log.Done("Platform detected")

	macosxXcodeProjectFiles := []string{}
	for iphoneosXcodeProjectFile := range macosxXcodeProjectFileMap {
		macosxXcodeProjectFiles = append(macosxXcodeProjectFiles, iphoneosXcodeProjectFile)
	}

	scanner.xcodeProjectAndWorkspaceFiles = macosxXcodeProjectFiles

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	warnings := models.Warnings{}

	// Separate workspaces and standalone projects
	workspaces := []xcodeproj.WorkspaceModel{}

	projectsToCheck := []string{}
	for _, projectOrWorkspace := range scanner.xcodeProjectAndWorkspaceFiles {
		if xcodeproj.IsXCodeProj(projectOrWorkspace) {
			projectsToCheck = append(projectsToCheck, projectOrWorkspace)
		}
	}

	for _, projectOrWorkspace := range scanner.xcodeProjectAndWorkspaceFiles {
		if xcodeproj.IsXCWorkspace(projectOrWorkspace) {
			workspace, err := xcodeproj.NewWorkspace(projectOrWorkspace, projectsToCheck...)
			if err != nil {
				return models.OptionModel{}, models.Warnings{}, fmt.Errorf("failed to analyze workspace (%s), error: %s", projectOrWorkspace, err)
			}

			workspaces = append(workspaces, workspace)
		}
	}

	if len(workspaces) > 0 {
		log.Details("%d workspace file(s) detected", len(workspaces))
		for _, workspace := range workspaces {
			projects := []string{}
			for _, project := range workspace.Projects {
				projects = append(projects, project.Name)
			}
			log.Details("- %s (projects: %v)", workspace.Name, projects)
		}
	}

	projects := []xcodeproj.ProjectModel{}

	for _, projectOrWorkspace := range scanner.xcodeProjectAndWorkspaceFiles {
		if !xcodeproj.IsXCodeProj(projectOrWorkspace) {
			continue
		}

		contained := false

		for _, workspace := range workspaces {
			for _, project := range workspace.Projects {
				if project.Pth == projectOrWorkspace {
					contained = true
				}
			}
		}

		if !contained {
			project, err := xcodeproj.NewProject(projectOrWorkspace)
			if err != nil {
				return models.OptionModel{}, models.Warnings{}, fmt.Errorf("failed to analyze project (%s), error: %s", projectOrWorkspace, err)
			}

			projects = append(projects, project)
		}
	}

	if len(projects) > 0 {
		log.Details("%d project file(s) detected", len(projects))
		for _, project := range projects {
			log.Details("- %s", project.Name)
		}
	}
	// ---

	// Create cocoapods project-workspace mapping
	log.Info("Searching for Podfiles")

	podFiles := utility.FilterRelevantPodFiles(scanner.fileList)

	log.Details("%d Podfile(s) detected", len(podFiles))
	for _, file := range podFiles {
		log.Details("- %s", file)
	}

	for _, podfile := range podFiles {
		workspaceProjectMap, err := utility.GetWorkspaceProjectMap(podfile)
		if err != nil {
			log.Warn("Analyze Podfile (%s) failed, error: %s", podfile, err)
			warnings = append(warnings, fmt.Sprintf("Failed to analyze Podfile: (%s), error: %s", podfile, err))
			continue
		}

		log.Details("")
		log.Details("cocoapods workspace-project mapping:")
		for workspacePth, linkedProjectPth := range workspaceProjectMap {
			log.Details("- %s -> %s", workspacePth, linkedProjectPth)

			podWorkspace := xcodeproj.WorkspaceModel{}

			projectFound := false

			for _, workspace := range workspaces {
				if workspace.Pth == workspacePth {
					podWorkspace = workspace

					for _, project := range workspace.Projects {
						if project.Pth == linkedProjectPth {
							projectFound = true
						}
					}

					if !projectFound {
						return models.OptionModel{}, models.Warnings{}, fmt.Errorf("workspace (%s) is exists, but does not conatins project (%s)", workspace.Name, linkedProjectPth)
					}
				}
			}
			podWorkspace.IsPodWorkspace = true

			if !projectFound {
				for _, project := range projects {
					if project.Pth == linkedProjectPth {
						projectFound = true
						podWorkspace.Projects = append(podWorkspace.Projects, project)
					}
				}
			}

			if !projectFound {
				return models.OptionModel{}, models.Warnings{}, fmt.Errorf("project (%s) not found", linkedProjectPth)
			}
		}
	}
	// ---

	//
	// Analyze projects and workspaces
	for _, project := range projects {
		log.Info("Inspecting standalone project file: %s", project.Pth)

		log.Details("%d shared scheme(s) detected", len(project.SharedSchemes))
		for _, scheme := range project.SharedSchemes {
			log.Details("- %s", scheme.Name)
		}

		if len(project.SharedSchemes) == 0 {
			log.Details("")
			log.Error("No shared schemes found, adding recreate-user-schemes step...")
			log.Error("The newly generated schemes may differ from the ones in your project.")
			log.Error("Make sure to share your schemes, to have the expected behaviour.")
			log.Details("")

			message := `No shared schemes found for project: ` + project.Pth + `.
	Automatically generated schemes for this project.
	These schemes may differ from the ones in your project.
	Make sure to <a href="https://developer.apple.com/library/ios/recipes/xcode_help-scheme_editor/Articles/SchemeManage.html">share your schemes</a> for the expected behaviour.`

			warnings = append(warnings, fmt.Sprintf(message))

			log.Warn("%d user scheme(s) will be generated", len(project.Targets))
			for _, target := range project.Targets {
				log.Warn("- %s", target.Name)
			}
		}
	}

	for _, workspace := range workspaces {
		log.Info("Inspecting workspace file: %s", workspace.Pth)

		sharedSchemes := workspace.GetSharedSchemes()
		log.Details("%d shared scheme(s) detected", len(sharedSchemes))
		for _, scheme := range sharedSchemes {
			log.Details("- %s", scheme.Name)
		}

		if len(sharedSchemes) == 0 {
			log.Details("")
			log.Error("No shared schemes found, adding recreate-user-schemes step...")
			log.Error("The newly generated schemes, may differs from the ones in your project.")
			log.Error("Make sure to share your schemes, to have the expected behaviour.")
			log.Details("")

			message := `No shared schemes found for project: ` + workspace.Pth + `.
	Automatically generated schemes for this project.
	These schemes may differ from the ones in your project.
	Make sure to <a href="https://developer.apple.com/library/ios/recipes/xcode_help-scheme_editor/Articles/SchemeManage.html">share your schemes</a> for the expected behaviour.`

			warnings = append(warnings, fmt.Sprintf(message))

			targets := workspace.GetTargets()
			log.Warn("%d user scheme(s) will be generated", len(targets))
			for _, target := range targets {
				log.Warn("- %s", target.Name)
			}
		}
	}
	// -----

	//
	// Create config descriptors
	configDescriptors := []ConfigDescriptor{}
	projectPathOption := models.NewOptionModel(projectPathTitle, projectPathEnvKey)

	for _, project := range projects {
		schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)

		if len(project.SharedSchemes) == 0 {
			for _, target := range project.Targets {
				configDescriptor := ConfigDescriptor{
					HasPodfile:           false,
					HasTest:              target.HasXCTest,
					MissingSharedSchemes: true,
				}
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewEmptyOptionModel()
				configOption.Config = configDescriptor.String()

				schemeOption.ValueMap[target.Name] = configOption
			}
		} else {
			for _, scheme := range project.SharedSchemes {
				configDescriptor := ConfigDescriptor{
					HasPodfile:           false,
					HasTest:              scheme.HasXCTest,
					MissingSharedSchemes: false,
				}
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewEmptyOptionModel()
				configOption.Config = configDescriptor.String()

				schemeOption.ValueMap[scheme.Name] = configOption
			}
		}

		projectPathOption.ValueMap[project.Pth] = schemeOption
	}

	for _, workspace := range workspaces {
		schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)

		schemes := workspace.GetSharedSchemes()

		if len(schemes) == 0 {
			targets := workspace.GetTargets()

			for _, target := range targets {
				configDescriptor := ConfigDescriptor{
					HasPodfile:           workspace.IsPodWorkspace,
					HasTest:              target.HasXCTest,
					MissingSharedSchemes: true,
				}
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewEmptyOptionModel()
				configOption.Config = configDescriptor.String()

				schemeOption.ValueMap[target.Name] = configOption
			}
		} else {
			for _, scheme := range schemes {
				configDescriptor := ConfigDescriptor{
					HasPodfile:           workspace.IsPodWorkspace,
					HasTest:              scheme.HasXCTest,
					MissingSharedSchemes: false,
				}
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewEmptyOptionModel()
				configOption.Config = configDescriptor.String()

				schemeOption.ValueMap[scheme.Name] = configOption
			}
		}

		projectPathOption.ValueMap[workspace.Pth] = schemeOption
	}
	// -----

	if len(configDescriptors) == 0 {
		log.Error("No valid iOS config found")
		return models.OptionModel{}, warnings, errors.New("No valid config found")
	}

	scanner.configDescriptors = configDescriptors

	return projectPathOption, warnings, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionModel {
	configOption := models.NewEmptyOptionModel()
	configOption.Config = defaultConfigName

	projectPathOption := models.NewOptionModel(projectPathTitle, projectPathEnvKey)
	schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)

	schemeOption.ValueMap["_"] = configOption
	projectPathOption.ValueMap["_"] = schemeOption

	return projectPathOption
}

func generateConfig(hasPodfile, hasTest, missingSharedSchemes bool) bitriseModels.BitriseDataModel {
	//
	// Prepare steps
	prepareSteps := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	prepareSteps = append(prepareSteps, steps.ActivateSSHKeyStepListItem())

	// GitClone
	prepareSteps = append(prepareSteps, steps.GitCloneStepListItem())

	// Script
	prepareSteps = append(prepareSteps, steps.ScriptSteplistItem(steps.ScriptDefaultTitle))

	// CertificateAndProfileInstaller
	prepareSteps = append(prepareSteps, steps.CertificateAndProfileInstallerStepListItem())

	if hasPodfile {
		// CocoapodsInstall
		prepareSteps = append(prepareSteps, steps.CocoapodsInstallStepListItem())
	}

	if missingSharedSchemes {
		// RecreateUserSchemes
		prepareSteps = append(prepareSteps, steps.RecreateUserSchemesStepListItem([]envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		}))
	}
	// ----------

	//
	// CI steps
	ciSteps := append([]bitriseModels.StepListItemModel{}, prepareSteps...)

	if hasTest {
		// XcodeTestMac
		ciSteps = append(ciSteps, steps.XcodeTestMacStepListItem([]envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
			envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
		}))
	}

	// DeployToBitriseIo
	ciSteps = append(ciSteps, steps.DeployToBitriseIoStepListItem())
	// ----------

	//
	// Deploy steps
	deploySteps := append([]bitriseModels.StepListItemModel{}, prepareSteps...)

	if hasTest {
		// XcodeTestMac
		deploySteps = append(deploySteps, steps.XcodeTestMacStepListItem([]envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
			envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
		}))
	}

	// XcodeArchiveMac
	deploySteps = append(deploySteps, steps.XcodeArchiveMacStepListItem([]envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
	}))

	// DeployToBitriseIo
	deploySteps = append(deploySteps, steps.DeployToBitriseIoStepListItem())
	// ----------

	return models.BitriseDataWithCIAndCDWorkflow([]envmanModels.EnvironmentItemModel{}, ciSteps, deploySteps)
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	descriptors := []ConfigDescriptor{}
	descritorNameMap := map[string]bool{}

	for _, descriptor := range scanner.configDescriptors {
		_, exist := descritorNameMap[descriptor.String()]
		if !exist {
			descriptors = append(descriptors, descriptor)
		}
	}

	bitriseDataMap := models.BitriseConfigMap{}
	for _, descriptor := range descriptors {
		configName := descriptor.String()
		bitriseData := generateConfig(descriptor.HasPodfile, descriptor.HasTest, descriptor.MissingSharedSchemes)
		data, err := yaml.Marshal(bitriseData)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}
		bitriseDataMap[configName] = string(data)
	}

	return bitriseDataMap, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	//
	// Prepare steps
	prepareSteps := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	prepareSteps = append(prepareSteps, steps.ActivateSSHKeyStepListItem())

	// GitClone
	prepareSteps = append(prepareSteps, steps.GitCloneStepListItem())

	// Script
	prepareSteps = append(prepareSteps, steps.ScriptSteplistItem(steps.ScriptDefaultTitle))

	// CertificateAndProfileInstaller
	prepareSteps = append(prepareSteps, steps.CertificateAndProfileInstallerStepListItem())

	// CocoapodsInstall
	prepareSteps = append(prepareSteps, steps.CocoapodsInstallStepListItem())

	// RecreateUserSchemes
	prepareSteps = append(prepareSteps, steps.RecreateUserSchemesStepListItem([]envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
	}))
	// ----------

	//
	// CI steps
	ciSteps := append([]bitriseModels.StepListItemModel{}, prepareSteps...)

	// XcodeTestMac
	ciSteps = append(ciSteps, steps.XcodeTestMacStepListItem([]envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
	}))

	// DeployToBitriseIo
	ciSteps = append(ciSteps, steps.DeployToBitriseIoStepListItem())
	// ----------

	//
	// Deploy steps
	deploySteps := append([]bitriseModels.StepListItemModel{}, prepareSteps...)

	// XcodeTestMac
	deploySteps = append(deploySteps, steps.XcodeTestStepListItem([]envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
	}))

	// XcodeArchiveMac
	deploySteps = append(deploySteps, steps.XcodeArchiveMacStepListItem([]envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
	}))

	// DeployToBitriseIo
	deploySteps = append(deploySteps, steps.DeployToBitriseIoStepListItem())
	// ----------

	config := models.BitriseDataWithCIAndCDWorkflow([]envmanModels.EnvironmentItemModel{}, ciSteps, deploySteps)
	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := defaultConfigName
	bitriseDataMap := models.BitriseConfigMap{}
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}
