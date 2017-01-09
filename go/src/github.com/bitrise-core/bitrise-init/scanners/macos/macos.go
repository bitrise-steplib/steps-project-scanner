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
	"github.com/bitrise-io/go-utils/log"
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
	searchDir         string
	fileList          []string
	projectFiles      []string
	configDescriptors []ConfigDescriptor
}

// Name ...
func (scanner Scanner) Name() string {
	return scannerName
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	scanner.searchDir = searchDir

	fileList, err := utility.ListPathInDirSortedByComponents(searchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}
	scanner.fileList = fileList

	// Search for xcodeproj
	log.Infoft("Searching for Xcode project files")

	xcodeprojectFiles, err := utility.FilterPaths(fileList, utility.AllowXcodeProjExtFilter)
	if err != nil {
		return false, err
	}

	log.Printft("%d Xcode project files found", len(xcodeprojectFiles))
	for _, xcodeprojectFile := range xcodeprojectFiles {
		log.Printft("- %s", xcodeprojectFile)
	}

	if len(xcodeprojectFiles) == 0 {
		log.Printft("platform not detected")
		return false, nil
	}

	log.Infoft("Filter relevant Xcode project files")

	xcodeprojectFiles, err = utility.FilterPaths(xcodeprojectFiles,
		utility.AllowIsDirectoryFilter,
		utility.ForbidEmbeddedWorkspaceRegexpFilter,
		utility.ForbidGitDirComponentFilter,
		utility.ForbidPodsDirComponentFilter,
		utility.ForbidCarthageDirComponentFilter,
		utility.ForbidFramworkComponentWithExtensionFilter,
		utility.AllowMacosxSDKFilter,
	)
	if err != nil {
		return false, err
	}

	log.Printft("%d Xcode macOS project files found", len(xcodeprojectFiles))
	for _, xcodeprojectFile := range xcodeprojectFiles {
		log.Printft("- %s", xcodeprojectFile)
	}

	if len(xcodeprojectFiles) == 0 {
		log.Printft("platform not detected")
		return false, nil
	}

	scanner.projectFiles = xcodeprojectFiles

	log.Doneft("Platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	warnings := models.Warnings{}

	projectFiles := scanner.projectFiles
	workspaceFiles, err := utility.FilterPaths(scanner.fileList,
		utility.AllowXCWorkspaceExtFilter,
		utility.AllowIsDirectoryFilter,
		utility.ForbidEmbeddedWorkspaceRegexpFilter,
		utility.ForbidGitDirComponentFilter,
		utility.ForbidPodsDirComponentFilter,
		utility.ForbidCarthageDirComponentFilter,
		utility.ForbidFramworkComponentWithExtensionFilter,
		utility.AllowMacosxSDKFilter,
	)
	if err != nil {
		return models.OptionModel{}, models.Warnings{}, err
	}

	standaloneProjects, workspaces, err := utility.CreateStandaloneProjectsAndWorkspaces(projectFiles, workspaceFiles)
	if err != nil {
		return models.OptionModel{}, models.Warnings{}, err
	}

	//
	// Create cocoapods workspace-project mapping
	log.Infoft("Searching for Podfiles")

	podfiles, err := utility.FilterPaths(scanner.fileList,
		utility.AllowPodfileBaseFilter,
		utility.ForbidGitDirComponentFilter,
		utility.ForbidPodsDirComponentFilter,
		utility.ForbidCarthageDirComponentFilter,
		utility.ForbidFramworkComponentWithExtensionFilter)
	if err != nil {
		return models.OptionModel{}, models.Warnings{}, err
	}

	log.Printft("%d Podfiles detected", len(podfiles))
	for _, file := range podfiles {
		log.Printft("- %s", file)
	}

	for _, podfile := range podfiles {
		workspaceProjectMap, err := utility.GetWorkspaceProjectMap(podfile, projectFiles)
		if err != nil {
			return models.OptionModel{}, models.Warnings{}, err
		}

		standaloneProjects, workspaces, err = utility.MergePodWorkspaceProjectMap(workspaceProjectMap, standaloneProjects, workspaces)
		if err != nil {
			return models.OptionModel{}, models.Warnings{}, err
		}
	}
	// ---

	//
	// Analyze projects and workspaces
	defaultGitignorePth := filepath.Join(scanner.searchDir, ".gitignore")
	isXcshareddataGitignored, err := utility.FileContains(defaultGitignorePth, "xcshareddata")
	if err != nil {
		log.Warnf("Failed to check if xcshareddata gitignored, error: %s", err)
	}

	for _, project := range standaloneProjects {
		log.Infoft("Inspecting standalone project file: %s", project.Pth)

		log.Printft("%d shared schemes detected", len(project.SharedSchemes))
		for _, scheme := range project.SharedSchemes {
			log.Printft("- %s", scheme.Name)
		}

		if len(project.SharedSchemes) == 0 {
			log.Printft("")
			log.Errorft("No shared schemes found, adding recreate-user-schemes step...")
			log.Errorft("The newly generated schemes may differ from the ones in your project.")
			if isXcshareddataGitignored {
				log.Errorft("Your gitignore file (%s) contains 'xcshareddata', maybe shared schemes are gitignored?", defaultGitignorePth)
				log.Errorft("If not, make sure to share your schemes, to have the expected behaviour.")
			} else {
				log.Errorft("Make sure to share your schemes, to have the expected behaviour.")
			}
			log.Printft("")

			message := `No shared schemes found for project: ` + project.Pth + `.`
			if isXcshareddataGitignored {
				message += `
Your gitignore file (` + defaultGitignorePth + `) (%s) contains 'xcshareddata', maybe shared schemes are gitignored?`
			}
			message += `
Automatically generated schemes may differ from the ones in your project.
Make sure to <a href="http://devcenter.bitrise.io/ios/frequent-ios-issues/#xcode-scheme-not-found">share your schemes</a> for the expected behaviour.`

			warnings = append(warnings, fmt.Sprintf(message))

			log.Warnft("%d user schemes will be generated", len(project.Targets))
			for _, target := range project.Targets {
				log.Warnft("- %s", target.Name)
			}
		}
	}

	for _, workspace := range workspaces {
		log.Infoft("Inspecting workspace file: %s", workspace.Pth)

		sharedSchemes := workspace.GetSharedSchemes()
		log.Printft("%d shared schemes detected", len(sharedSchemes))
		for _, scheme := range sharedSchemes {
			log.Printft("- %s", scheme.Name)
		}

		if len(sharedSchemes) == 0 {
			log.Printft("")
			log.Errorft("No shared schemes found, adding recreate-user-schemes step...")
			log.Errorft("The newly generated schemes may differ from the ones in your project.")
			if isXcshareddataGitignored {
				log.Errorft("Your gitignore file (%s) contains 'xcshareddata', maybe shared schemes are gitignored?", defaultGitignorePth)
				log.Errorft("If not, make sure to share your schemes, to have the expected behaviour.")
			} else {
				log.Errorft("Make sure to share your schemes, to have the expected behaviour.")
			}
			log.Printft("")

			message := `No shared schemes found for project: ` + workspace.Pth + `.`
			if isXcshareddataGitignored {
				message += `
Your gitignore file (` + defaultGitignorePth + `) contains 'xcshareddata', maybe shared schemes are gitignored?`
			}
			message += `
Automatically generated schemes may differ from the ones in your project.
Make sure to <a href="http://devcenter.bitrise.io/ios/frequent-ios-issues/#xcode-scheme-not-found">share your schemes</a> for the expected behaviour.`

			warnings = append(warnings, fmt.Sprintf(message))

			targets := workspace.GetTargets()
			log.Warnft("%d user schemes will be generated", len(targets))
			for _, target := range targets {
				log.Warnft("- %s", target.Name)
			}
		}
	}
	// -----

	//
	// Create config descriptors
	configDescriptors := []ConfigDescriptor{}
	projectPathOption := models.NewOptionModel(projectPathTitle, projectPathEnvKey)

	// Add Standalon Project options
	for _, project := range standaloneProjects {
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

	// Add Workspace options
	for _, workspace := range workspaces {
		schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)

		sharedSchemes := workspace.GetSharedSchemes()
		if len(sharedSchemes) == 0 {
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
			for _, scheme := range sharedSchemes {
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
		log.Errorft("No valid macOS config found")
		return models.OptionModel{}, warnings, errors.New("No valid macOS config found")
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
	deploySteps = append(deploySteps, steps.XcodeTestMacStepListItem([]envmanModels.EnvironmentItemModel{
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
