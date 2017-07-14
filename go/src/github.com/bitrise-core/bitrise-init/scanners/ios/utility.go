package ios

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"path/filepath"

	"strings"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-xcode/xcodeproj"
)

const (
	defaultConfigNameFormat = "default-%s-config"
	configNameFormat        = "%s%s-config"
)

const (
	// ProjectPathInputKey ...
	ProjectPathInputKey = "project_path"
	// ProjectPathInputEnvKey ...
	ProjectPathInputEnvKey = "BITRISE_PROJECT_PATH"
	// ProjectPathInputTitle ...
	ProjectPathInputTitle = "Project (or Workspace) path"
)

const (
	// SchemeInputKey ...
	SchemeInputKey = "scheme"
	// SchemeInputEnvKey ...
	SchemeInputEnvKey = "BITRISE_SCHEME"
	// SchemeInputTitle ...
	SchemeInputTitle = "Scheme name"
)

const (
	// ConfigurationInputKey ...
	ConfigurationInputKey = "configuration"
)

const (
	// CarthageCommandInputKey ...
	CarthageCommandInputKey = "carthage_command"
)

const cartfileBase = "Cartfile"
const cartfileResolvedBase = "Cartfile.resolved"

// AllowCartfileBaseFilter ...
var AllowCartfileBaseFilter = utility.BaseFilter(cartfileBase, true)

// ConfigDescriptor ...
type ConfigDescriptor struct {
	HasPodfile           bool
	CarthageCommand      string
	HasTest              bool
	MissingSharedSchemes bool
}

// NewConfigDescriptor ...
func NewConfigDescriptor(hasPodfile bool, carthageCommand string, hasXCTest bool, missingSharedSchemes bool) ConfigDescriptor {
	return ConfigDescriptor{
		HasPodfile:           hasPodfile,
		CarthageCommand:      carthageCommand,
		HasTest:              hasXCTest,
		MissingSharedSchemes: missingSharedSchemes,
	}
}

// NewConfigDescriptorWithName ...
func NewConfigDescriptorWithName(name string) ConfigDescriptor {
	descriptor := ConfigDescriptor{}
	if strings.Contains(name, "-pod") {
		descriptor.HasPodfile = true
	}
	if strings.Contains(name, "-carthage") {
		descriptor.CarthageCommand = "bootstrap"
	}
	if strings.Contains(name, "-test") {
		descriptor.HasTest = true
	}
	if strings.Contains(name, "-missing-shared-schemes") {
		descriptor.MissingSharedSchemes = true
	}
	return descriptor
}

// ConfigName ...
func (descriptor ConfigDescriptor) ConfigName(projectType XcodeProjectType) string {
	qualifiers := ""
	if descriptor.HasPodfile {
		qualifiers += "-pod"
	}
	if descriptor.CarthageCommand != "" {
		qualifiers += "-carthage"
	}
	if descriptor.HasTest {
		qualifiers += "-test"
	}
	if descriptor.MissingSharedSchemes {
		qualifiers += "-missing-shared-schemes"
	}
	return fmt.Sprintf(configNameFormat, string(projectType), qualifiers)
}

// HasCartfileInDirectoryOf ...
func HasCartfileInDirectoryOf(pth string) bool {
	dir := filepath.Dir(pth)
	cartfilePth := filepath.Join(dir, cartfileBase)
	exist, err := pathutil.IsPathExists(cartfilePth)
	if err != nil {
		return false
	}
	return exist
}

// HasCartfileResolvedInDirectoryOf ...
func HasCartfileResolvedInDirectoryOf(pth string) bool {
	dir := filepath.Dir(pth)
	cartfileResolvedPth := filepath.Join(dir, cartfileResolvedBase)
	exist, err := pathutil.IsPathExists(cartfileResolvedPth)
	if err != nil {
		return false
	}
	return exist
}

// Detect ...
func Detect(projectType XcodeProjectType, searchDir string) (bool, error) {
	fileList, err := utility.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return false, err
	}

	log.Infoft("Filter relevant Xcode project files")

	relevantXcodeprojectFiles, err := FilterRelevantProjectFiles(fileList, projectType)
	if err != nil {
		return false, err
	}

	log.Printft("%d Xcode %s project files found", len(relevantXcodeprojectFiles), string(projectType))
	for _, xcodeprojectFile := range relevantXcodeprojectFiles {
		log.Printft("- %s", xcodeprojectFile)
	}

	if len(relevantXcodeprojectFiles) == 0 {
		log.Printft("platform not detected")
		return false, nil
	}

	log.Doneft("Platform detected")

	return true, nil
}

func printMissingSharedSchemesAndGenerateWarning(projectPth, defaultGitignorePth string, targets []xcodeproj.TargetModel) string {
	isXcshareddataGitignored := false
	if exist, err := pathutil.IsPathExists(defaultGitignorePth); err != nil {
		log.Warnft("Failed to check if .gitignore file exists at: %s, error: %s", defaultGitignorePth, err)
	} else if exist {
		isGitignored, err := utility.FileContains(defaultGitignorePth, "xcshareddata")
		if err != nil {
			log.Warnft("Failed to check if xcshareddata gitignored, error: %s", err)
		} else {
			isXcshareddataGitignored = isGitignored
		}
	}

	log.Printft("")
	log.Errorft("No shared schemes found, adding recreate-user-schemes step...")
	log.Errorft("The newly generated schemes may differ from the ones in your project.")

	message := `No shared schemes found for project: ` + projectPth + `.` + "\n"

	if isXcshareddataGitignored {
		log.Errorft("Your gitignore file (%s) contains 'xcshareddata', maybe shared schemes are gitignored?", defaultGitignorePth)
		log.Errorft("If not, make sure to share your schemes, to have the expected behaviour.")

		message += `Your gitignore file (` + defaultGitignorePth + `) contains 'xcshareddata', maybe shared schemes are gitignored?` + "\n"
	} else {
		log.Errorft("Make sure to share your schemes, to have the expected behaviour.")
	}

	message += `Automatically generated schemes may differ from the ones in your project.
Make sure to <a href="http://devcenter.bitrise.io/ios/frequent-ios-issues/#xcode-scheme-not-found">share your schemes</a> for the expected behaviour.`

	log.Printft("")

	log.Warnft("%d user schemes will be generated", len(targets))
	for _, target := range targets {
		log.Warnft("- %s", target.Name)
	}

	log.Printft("")

	return message
}

func detectCarthageCommand(projectPth string) (string, string) {
	carthageCommand := ""
	warning := ""

	if HasCartfileInDirectoryOf(projectPth) {
		if HasCartfileResolvedInDirectoryOf(projectPth) {
			carthageCommand = "bootstrap"
		} else {
			dir := filepath.Dir(projectPth)
			cartfilePth := filepath.Join(dir, "Cartfile")

			warning = fmt.Sprintf(`Cartfile found at (%s), but no Cartfile.resolved exists in the same directory.
It is <a href="https://github.com/Carthage/Carthage/blob/master/Documentation/Artifacts.md#cartfileresolved">strongly recommended to commit this file to your repository</a>`, cartfilePth)

			carthageCommand = "update"
		}
	}

	return carthageCommand, warning
}

// GenerateOptions ...
func GenerateOptions(projectType XcodeProjectType, searchDir string) (models.OptionModel, []ConfigDescriptor, models.Warnings, error) {
	warnings := models.Warnings{}

	fileList, err := utility.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return models.OptionModel{}, []ConfigDescriptor{}, models.Warnings{}, err
	}

	// Separate workspaces and standalon projects
	projectFiles, err := FilterRelevantProjectFiles(fileList, projectType)
	if err != nil {
		return models.OptionModel{}, []ConfigDescriptor{}, models.Warnings{}, err
	}

	workspaceFiles, err := FilterRelevantWorkspaceFiles(fileList, projectType)
	if err != nil {
		return models.OptionModel{}, []ConfigDescriptor{}, models.Warnings{}, err
	}

	standaloneProjects, workspaces, err := CreateStandaloneProjectsAndWorkspaces(projectFiles, workspaceFiles)
	if err != nil {
		return models.OptionModel{}, []ConfigDescriptor{}, models.Warnings{}, err
	}

	// Create cocoapods workspace-project mapping
	log.Infoft("Searching for Podfile")

	podfiles, err := FilterRelevantPodfiles(fileList)
	if err != nil {
		return models.OptionModel{}, []ConfigDescriptor{}, models.Warnings{}, err
	}

	log.Printft("%d Podfiles detected", len(podfiles))

	for _, podfile := range podfiles {
		log.Printft("- %s", podfile)

		workspaceProjectMap, err := GetWorkspaceProjectMap(podfile, projectFiles)
		if err != nil {
			warning := fmt.Sprintf("Failed to determine cocoapods project-workspace mapping, error: %s", err)
			warnings = append(warnings, warning)
			log.Warnf(warning)
			continue
		}

		aStandaloneProjects, aWorkspaces, err := MergePodWorkspaceProjectMap(workspaceProjectMap, standaloneProjects, workspaces)
		if err != nil {
			warning := fmt.Sprintf("Failed to create cocoapods project-workspace mapping, error: %s", err)
			warnings = append(warnings, warning)
			log.Warnf(warning)
			continue
		}

		standaloneProjects = aStandaloneProjects
		workspaces = aWorkspaces
	}

	// Carthage
	log.Infoft("Searching for Cartfile")

	cartfiles, err := FilterRelevantCartFile(fileList)
	if err != nil {
		return models.OptionModel{}, []ConfigDescriptor{}, models.Warnings{}, err
	}

	log.Printft("%d Cartfiles detected", len(cartfiles))
	for _, file := range cartfiles {
		log.Printft("- %s", file)
	}

	// Create config descriptors & options
	configDescriptors := []ConfigDescriptor{}

	defaultGitignorePth := filepath.Join(searchDir, ".gitignore")

	projectPathOption := models.NewOption(ProjectPathInputTitle, ProjectPathInputEnvKey)

	// Standalon Projects
	for _, project := range standaloneProjects {
		log.Infoft("Inspecting standalone project file: %s", project.Pth)

		schemeOption := models.NewOption(SchemeInputTitle, SchemeInputEnvKey)
		projectPathOption.AddOption(project.Pth, schemeOption)

		carthageCommand, warning := detectCarthageCommand(project.Pth)
		if warning != "" {
			warnings = append(warnings, warning)
		}

		log.Printft("%d shared schemes detected", len(project.SharedSchemes))

		if len(project.SharedSchemes) == 0 {
			message := printMissingSharedSchemesAndGenerateWarning(project.Pth, defaultGitignorePth, project.Targets)
			if message != "" {
				warnings = append(warnings, message)
			}

			for _, target := range project.Targets {
				configDescriptor := NewConfigDescriptor(false, carthageCommand, target.HasXCTest, true)
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewConfigOption(configDescriptor.ConfigName(projectType))
				schemeOption.AddConfig(target.Name, configOption)
			}
		} else {
			for _, scheme := range project.SharedSchemes {
				log.Printft("- %s", scheme.Name)

				configDescriptor := NewConfigDescriptor(false, carthageCommand, scheme.HasXCTest, false)
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewConfigOption(configDescriptor.ConfigName(projectType))
				schemeOption.AddConfig(scheme.Name, configOption)
			}
		}
	}

	// Workspaces
	for _, workspace := range workspaces {
		log.Infoft("Inspecting workspace file: %s", workspace.Pth)

		schemeOption := models.NewOption(SchemeInputTitle, SchemeInputEnvKey)
		projectPathOption.AddOption(workspace.Pth, schemeOption)

		carthageCommand, warning := detectCarthageCommand(workspace.Pth)
		if warning != "" {
			warnings = append(warnings, warning)
		}

		sharedSchemes := workspace.GetSharedSchemes()
		log.Printft("%d shared schemes detected", len(sharedSchemes))

		if len(sharedSchemes) == 0 {
			targets := workspace.GetTargets()

			message := printMissingSharedSchemesAndGenerateWarning(workspace.Pth, defaultGitignorePth, targets)
			if message != "" {
				warnings = append(warnings, message)
			}

			for _, target := range targets {
				configDescriptor := NewConfigDescriptor(workspace.IsPodWorkspace, carthageCommand, target.HasXCTest, true)
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewConfigOption(configDescriptor.ConfigName(projectType))
				schemeOption.AddConfig(target.Name, configOption)
			}
		} else {
			for _, scheme := range sharedSchemes {
				log.Printft("- %s", scheme.Name)

				configDescriptor := NewConfigDescriptor(workspace.IsPodWorkspace, carthageCommand, scheme.HasXCTest, false)
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewConfigOption(configDescriptor.ConfigName(projectType))
				schemeOption.AddConfig(scheme.Name, configOption)
			}
		}
	}

	configDescriptors = RemoveDuplicatedConfigDescriptors(configDescriptors, projectType)

	if len(configDescriptors) == 0 {
		log.Errorft("No valid %s config found", string(projectType))
		return models.OptionModel{}, []ConfigDescriptor{}, warnings, fmt.Errorf("No valid %s config found", string(projectType))
	}

	return *projectPathOption, configDescriptors, warnings, nil
}

// GenerateDefaultOptions ...
func GenerateDefaultOptions(projectType XcodeProjectType) models.OptionModel {
	projectPathOption := models.NewOption(ProjectPathInputTitle, ProjectPathInputEnvKey)

	schemeOption := models.NewOption(SchemeInputTitle, SchemeInputEnvKey)
	projectPathOption.AddOption("_", schemeOption)

	configOption := models.NewConfigOption(fmt.Sprintf(defaultConfigNameFormat, string(projectType)))
	schemeOption.AddConfig("_", configOption)

	return *projectPathOption
}

// GenerateConfigBuilder ...
func GenerateConfigBuilder(projectType XcodeProjectType, hasPodfile, hasTest, missingSharedSchemes bool, carthageCommand string, isIncludeCache bool) models.ConfigBuilderModel {
	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(isIncludeCache)...)

	// CI
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

	if missingSharedSchemes {
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.RecreateUserSchemesStepListItem(
			envmanModels.EnvironmentItemModel{ProjectPathInputKey: "$" + ProjectPathInputEnvKey},
		))
	}

	if hasPodfile {
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CocoapodsInstallStepListItem())
	}

	if carthageCommand != "" {
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CarthageStepListItem(
			envmanModels.EnvironmentItemModel{CarthageCommandInputKey: carthageCommand},
		))
	}

	xcodeTestAndArchiveStepInputModels := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{ProjectPathInputKey: "$" + ProjectPathInputEnvKey},
		envmanModels.EnvironmentItemModel{SchemeInputKey: "$" + SchemeInputEnvKey},
	}

	if hasTest {
		switch projectType {
		case XcodeProjectTypeIOS:
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.XcodeTestStepListItem(xcodeTestAndArchiveStepInputModels...))
		case XcodeProjectTypeMacOS:
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.XcodeTestMacStepListItem(xcodeTestAndArchiveStepInputModels...))
		}
	}
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(isIncludeCache)...)

	// CD
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(isIncludeCache)...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

	if missingSharedSchemes {
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.RecreateUserSchemesStepListItem(
			envmanModels.EnvironmentItemModel{ProjectPathInputKey: "$" + ProjectPathInputEnvKey},
		))
	}

	if hasPodfile {
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CocoapodsInstallStepListItem())
	}

	if carthageCommand != "" {
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CarthageStepListItem(
			envmanModels.EnvironmentItemModel{CarthageCommandInputKey: carthageCommand},
		))
	}

	if hasTest {
		switch projectType {
		case XcodeProjectTypeIOS:
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeTestStepListItem(xcodeTestAndArchiveStepInputModels...))
		case XcodeProjectTypeMacOS:
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeTestMacStepListItem(xcodeTestAndArchiveStepInputModels...))
		}
	}

	switch projectType {
	case XcodeProjectTypeIOS:
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(xcodeTestAndArchiveStepInputModels...))
	case XcodeProjectTypeMacOS:
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveMacStepListItem(xcodeTestAndArchiveStepInputModels...))
	}
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(isIncludeCache)...)

	return *configBuilder
}

// RemoveDuplicatedConfigDescriptors ...
func RemoveDuplicatedConfigDescriptors(configDescriptors []ConfigDescriptor, projectType XcodeProjectType) []ConfigDescriptor {
	descritorNameMap := map[string]ConfigDescriptor{}
	for _, descriptor := range configDescriptors {
		name := descriptor.ConfigName(projectType)
		descritorNameMap[name] = descriptor
	}

	descriptors := []ConfigDescriptor{}
	for _, descriptor := range descritorNameMap {
		descriptors = append(descriptors, descriptor)
	}

	return descriptors
}

// GenerateConfig ...
func GenerateConfig(projectType XcodeProjectType, configDescriptors []ConfigDescriptor, isIncludeCache bool) (models.BitriseConfigMap, error) {
	bitriseDataMap := models.BitriseConfigMap{}
	for _, descriptor := range configDescriptors {
		configBuilder := GenerateConfigBuilder(projectType, descriptor.HasPodfile, descriptor.HasTest, descriptor.MissingSharedSchemes, descriptor.CarthageCommand, isIncludeCache)

		config, err := configBuilder.Generate(string(projectType))
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		bitriseDataMap[descriptor.ConfigName(projectType)] = string(data)
	}

	return bitriseDataMap, nil
}

// GenerateDefaultConfig ...
func GenerateDefaultConfig(projectType XcodeProjectType, isIncludeCache bool) (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(isIncludeCache)...)

	// CI
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.RecreateUserSchemesStepListItem(
		envmanModels.EnvironmentItemModel{ProjectPathInputKey: "$" + ProjectPathInputEnvKey},
	))

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CocoapodsInstallStepListItem())

	xcodeTestAndArchiveStepInputModels := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{ProjectPathInputKey: "$" + ProjectPathInputEnvKey},
		envmanModels.EnvironmentItemModel{SchemeInputKey: "$" + SchemeInputEnvKey},
	}

	switch projectType {
	case XcodeProjectTypeIOS:
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.XcodeTestStepListItem(xcodeTestAndArchiveStepInputModels...))
	case XcodeProjectTypeMacOS:
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.XcodeTestMacStepListItem(xcodeTestAndArchiveStepInputModels...))
	}
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(true)...)

	// CD
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(isIncludeCache)...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.RecreateUserSchemesStepListItem(
		envmanModels.EnvironmentItemModel{ProjectPathInputKey: "$" + ProjectPathInputEnvKey},
	))

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CocoapodsInstallStepListItem())

	switch projectType {
	case XcodeProjectTypeIOS:
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeTestStepListItem(xcodeTestAndArchiveStepInputModels...))
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(xcodeTestAndArchiveStepInputModels...))
	case XcodeProjectTypeMacOS:
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeTestMacStepListItem(xcodeTestAndArchiveStepInputModels...))
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveMacStepListItem(xcodeTestAndArchiveStepInputModels...))
	}
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(true)...)

	config, err := configBuilder.Generate(string(projectType))
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		fmt.Sprintf(defaultConfigNameFormat, string(projectType)): string(data),
	}, nil
}
