package ios

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/analytics"
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-xcode/xcodeproj"
)

const (
	defaultConfigNameFormat = "default-%s-config"
	configNameFormat        = "%s%s-config"
	iconFailureTag          = "icon_lookup"
)

const (
	// ProjectPathInputKey ...
	ProjectPathInputKey = "project_path"
	// ProjectPathInputEnvKey ...
	ProjectPathInputEnvKey = "BITRISE_PROJECT_PATH"
	// ProjectPathInputTitle ...
	ProjectPathInputTitle = "Project or Workspace path"
	// ProjectPathInputSummary ...
	ProjectPathInputSummary = "The location of your Xcode project or Xcode workspace files, stored as an Environment Variable. In your Workflows, you can specify paths relative to this path."
)

const (
	// SchemeInputKey ...
	SchemeInputKey = "scheme"
	// SchemeInputEnvKey ...
	SchemeInputEnvKey = "BITRISE_SCHEME"
	// SchemeInputTitle ...
	SchemeInputTitle = "Scheme name"
	// SchemeInputSummary ...
	SchemeInputSummary = "An Xcode scheme defines a collection of targets to build, a configuration to use when building, and a collection of tests to execute. Only shared schemes are detected automatically but you can use any scheme as a target on Bitrise. You can change the scheme at any time in your Env Vars."
)

const (
	// DistributionMethodInputKey ...
	DistributionMethodInputKey = "distribution_method"
	// DistributionMethodEnvKey ...
	DistributionMethodEnvKey = "BITRISE_DISTRIBUTION_METHOD"
	// DistributionMethodInputTitle ...
	DistributionMethodInputTitle = "Distribution method"
	// DistributionMethodInputSummary ...
	DistributionMethodInputSummary = "The export method used to create an .ipa file in your builds, stored as an Environment Variable. You can change this at any time, or even create several .ipa files with different export methods in the same build."
)

const (
	// ExportMethodInputKey ...
	ExportMethodInputKey = "export_method"
	// ExportMethodEnvKey ...
	ExportMethodEnvKey = "BITRISE_EXPORT_METHOD"
	// ExportMethodInputTitle ...
	ExportMethodInputTitle = "Application export method\nNOTE: `none` means: Export a copy of the application without re-signing."
	// ExportMethodInputSummary ...
	ExportMethodInputSummary = "The export method used to create an .app file in your builds, stored as an Environment Variable. You can change this at any time, or even create several .app files with different export methods in the same build."
)

// XCConfigContentInputKey ...
const XCConfigContentInputKey = "xcconfig_content"

// IosExportMethods ...
var IosExportMethods = []string{"app-store", "ad-hoc", "enterprise", "development"}

const (
	// ExportXCArchiveProductInputKey ...
	ExportXCArchiveProductInputKey = "product"

	// ExportXCArchiveProductInputAppClipValue ...
	ExportXCArchiveProductInputAppClipValue = "app-clip"
)

// MacExportMethods ...
var MacExportMethods = []string{"app-store", "developer-id", "development", "none"}

const (
	// ConfigurationInputKey ...
	ConfigurationInputKey = "configuration"
)

const (
	// AutomaticCodeSigningInputKey ...
	AutomaticCodeSigningInputKey = "automatic_code_signing"
	// AutomaticCodeSigningInputAPIKeyValue ...
	AutomaticCodeSigningInputAPIKeyValue = "api-key"
)

const (
	// CarthageCommandInputKey ...
	CarthageCommandInputKey = "carthage_command"
)

const cartfileBase = "Cartfile"
const cartfileResolvedBase = "Cartfile.resolved"

// AllowCartfileBaseFilter ...
var AllowCartfileBaseFilter = pathutil.BaseFilter(cartfileBase, true)

// ConfigDescriptor ...
type ConfigDescriptor struct {
	HasPodfile           bool
	CarthageCommand      string
	HasTest              bool
	HasAppClip           bool
	ExportMethod         string
	MissingSharedSchemes bool
}

// NewConfigDescriptor ...
func NewConfigDescriptor(hasPodfile bool, carthageCommand string, hasXCTest, hasAppClip bool, exportMethod string, missingSharedSchemes bool) ConfigDescriptor {
	return ConfigDescriptor{
		HasPodfile:           hasPodfile,
		CarthageCommand:      carthageCommand,
		HasTest:              hasXCTest,
		HasAppClip:           hasAppClip,
		ExportMethod:         exportMethod,
		MissingSharedSchemes: missingSharedSchemes,
	}
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
	if descriptor.HasAppClip {
		qualifiers += fmt.Sprintf("-app-clip-%s", descriptor.ExportMethod)
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
	fileList, err := pathutil.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return false, err
	}

	log.TInfof("Filter relevant Xcode project files")

	relevantXcodeprojectFiles, err := FilterRelevantProjectFiles(fileList, projectType)
	if err != nil {
		return false, err
	}

	log.TPrintf("%d Xcode %s project files found", len(relevantXcodeprojectFiles), string(projectType))
	for _, xcodeprojectFile := range relevantXcodeprojectFiles {
		log.TPrintf("- %s", xcodeprojectFile)
	}

	if len(relevantXcodeprojectFiles) == 0 {
		log.TPrintf("platform not detected")
		return false, nil
	}

	log.TSuccessf("Platform detected")

	return true, nil
}

func fileContains(pth, str string) (bool, error) {
	content, err := fileutil.ReadStringFromFile(pth)
	if err != nil {
		return false, err
	}

	return strings.Contains(content, str), nil
}

func printMissingSharedSchemesAndGenerateWarning(projectPth, defaultGitignorePth string, targets []xcodeproj.TargetModel) string {
	isXcshareddataGitignored := false
	if exist, err := pathutil.IsPathExists(defaultGitignorePth); err != nil {
		log.TWarnf("Failed to check if .gitignore file exists at: %s, error: %s", defaultGitignorePth, err)
	} else if exist {
		isGitignored, err := fileContains(defaultGitignorePth, "xcshareddata")
		if err != nil {
			log.TWarnf("Failed to check if xcshareddata gitignored, error: %s", err)
		} else {
			isXcshareddataGitignored = isGitignored
		}
	}

	log.TPrintf("")
	log.TErrorf("No shared schemes found, adding recreate-user-schemes step...")
	log.TErrorf("The newly generated schemes may differ from the ones in your project.")

	message := `No shared schemes found for project: ` + projectPth + `.` + "\n"

	if isXcshareddataGitignored {
		log.TErrorf("Your gitignore file (%s) contains 'xcshareddata', maybe shared schemes are gitignored?", defaultGitignorePth)
		log.TErrorf("If not, make sure to share your schemes, to have the expected behaviour.")

		message += `Your gitignore file (` + defaultGitignorePth + `) contains 'xcshareddata', maybe shared schemes are gitignored?` + "\n"
	} else {
		log.TErrorf("Make sure to share your schemes, to have the expected behaviour.")
	}

	message += `Automatically generated schemes may differ from the ones in your project.
Make sure to <a href="http://devcenter.bitrise.io/ios/frequent-ios-issues/#xcode-scheme-not-found">share your schemes</a> for the expected behaviour.`

	log.TPrintf("")

	log.TWarnf("%d user schemes will be generated", len(targets))
	for _, target := range targets {
		log.TWarnf("- %s", target.Name)
	}

	log.TPrintf("")

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

func projectPathByScheme(projects []xcodeproj.ProjectModel, targetScheme string) string {
	for _, p := range projects {
		for _, s := range p.SharedSchemes {
			if s.Name == targetScheme {
				return p.Pth
			}
		}
	}
	return ""
}

// GenerateOptions ...
func GenerateOptions(projectType XcodeProjectType, searchDir string, excludeAppIcon, suppressPodFileParseError bool) (models.OptionNode, []ConfigDescriptor, models.Icons, models.Warnings, error) {
	warnings := models.Warnings{}

	fileList, err := pathutil.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return models.OptionNode{}, []ConfigDescriptor{}, nil, models.Warnings{}, err
	}

	// Separate workspaces and standalon projects
	projectFiles, err := FilterRelevantProjectFiles(fileList, projectType)
	if err != nil {
		return models.OptionNode{}, []ConfigDescriptor{}, nil, models.Warnings{}, err
	}

	workspaceFiles, err := FilterRelevantWorkspaceFiles(fileList, projectType)
	if err != nil {
		return models.OptionNode{}, []ConfigDescriptor{}, nil, models.Warnings{}, err
	}

	standaloneProjects, workspaces, err := CreateStandaloneProjectsAndWorkspaces(projectFiles, workspaceFiles)
	if err != nil {
		return models.OptionNode{}, []ConfigDescriptor{}, nil, models.Warnings{}, err
	}

	var exportMethodInputTitle string
	var exportMethodInputSummary string
	var exportMethodEnvKey string
	var exportMethods []string

	if projectType == XcodeProjectTypeIOS {
		exportMethodInputTitle = DistributionMethodInputTitle
		exportMethodInputSummary = DistributionMethodInputSummary
		exportMethodEnvKey = DistributionMethodEnvKey
		exportMethods = IosExportMethods
	} else {
		exportMethodInputTitle = ExportMethodInputTitle
		exportMethodInputSummary = ExportMethodInputSummary
		exportMethodEnvKey = ExportMethodEnvKey
		exportMethods = MacExportMethods
	}

	// Create cocoapods workspace-project mapping
	log.TInfof("Searching for Podfile")

	podfiles, err := FilterRelevantPodfiles(fileList)
	if err != nil {
		return models.OptionNode{}, []ConfigDescriptor{}, nil, models.Warnings{}, err
	}

	log.TPrintf("%d Podfiles detected", len(podfiles))

	for _, podfile := range podfiles {
		log.TPrintf("- %s", podfile)

		podfileParser := podfileParser{
			podfilePth:                podfile,
			suppressPodFileParseError: suppressPodFileParseError,
		}

		workspaceProjectMap, err := podfileParser.GetWorkspaceProjectMap(projectFiles)
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
	log.TInfof("Searching for Cartfile")

	cartfiles, err := FilterRelevantCartFile(fileList)
	if err != nil {
		return models.OptionNode{}, []ConfigDescriptor{}, nil, models.Warnings{}, err
	}

	log.TPrintf("%d Cartfiles detected", len(cartfiles))
	for _, file := range cartfiles {
		log.TPrintf("- %s", file)
	}

	// Create config descriptors & options
	configDescriptors := []ConfigDescriptor{}

	defaultGitignorePth := filepath.Join(searchDir, ".gitignore")

	projectPathOption := models.NewOption(ProjectPathInputTitle, ProjectPathInputSummary, ProjectPathInputEnvKey, models.TypeSelector)

	// App icons, merged from every project
	iconsForAllProjects := models.Icons{}

	// Standalone Projects
	for _, project := range standaloneProjects {
		log.TInfof("Inspecting standalone project file: %s", project.Pth)

		schemeOption := models.NewOption(SchemeInputTitle, SchemeInputSummary, SchemeInputEnvKey, models.TypeSelector)
		projectPathOption.AddOption(project.Pth, schemeOption)

		projectPath, err := filepath.Abs(filepath.Join(searchDir, project.Pth))
		if err != nil {
			return models.OptionNode{},
				[]ConfigDescriptor{},
				nil,
				warnings,
				fmt.Errorf("failed to get project path, error: %s", err)
		}

		carthageCommand, warning := detectCarthageCommand(project.Pth)
		if warning != "" {
			warnings = append(warnings, warning)
		}

		log.TPrintf("%d shared schemes detected", len(project.SharedSchemes))

		if len(project.SharedSchemes) == 0 {
			message := printMissingSharedSchemesAndGenerateWarning(project.Pth, defaultGitignorePth, project.Targets)
			if message != "" {
				warnings = append(warnings, message)
			}

			for _, target := range project.Targets {

				exportMethodOption := models.NewOption(exportMethodInputTitle, exportMethodInputSummary, exportMethodEnvKey, models.TypeSelector)
				schemeOption.AddOption(target.Name, exportMethodOption)

				iconIDs := []string{}
				if !excludeAppIcon {
					icons, err := lookupIconByTargetName(projectPath, target.Name, searchDir)
					if err != nil {
						log.Warnf("could not get icons for app: %s, error: %s", projectPath, err)
						analytics.LogInfo(iconFailureTag, analytics.DetectorErrorData(string(XcodeProjectTypeIOS), err), "Failed to lookup ios icons")
					}
					iconsForAllProjects = append(iconsForAllProjects, icons...)
					for _, icon := range icons {
						iconIDs = append(iconIDs, icon.Filename)
					}
				}

				for _, exportMethod := range exportMethods {
					configDescriptor := NewConfigDescriptor(false, carthageCommand, target.HasXCTest, target.HasAppClip, exportMethod, true)
					configDescriptors = append(configDescriptors, configDescriptor)
					configOption := models.NewConfigOption(configDescriptor.ConfigName(projectType), iconIDs)

					exportMethodOption.AddConfig(exportMethod, configOption)
				}
			}
		} else {
			for _, scheme := range project.SharedSchemes {
				log.TPrintf("- %s", scheme.Name)

				exportMethodOption := models.NewOption(exportMethodInputTitle, exportMethodInputSummary, exportMethodEnvKey, models.TypeSelector)
				schemeOption.AddOption(scheme.Name, exportMethodOption)

				iconIDs := []string{}
				if !excludeAppIcon {
					icons, err := lookupIconBySchemeName(projectPath, scheme.Name, searchDir)
					if err != nil {
						log.Warnf("could not get icons for app: %s, error: %s", projectPath, err)
						analytics.LogInfo(iconFailureTag, analytics.DetectorErrorData(string(XcodeProjectTypeIOS), err), "Failed to lookup ios icons")
					}
					iconsForAllProjects = append(iconsForAllProjects, icons...)
					for _, icon := range icons {
						iconIDs = append(iconIDs, icon.Filename)
					}
				}

				for _, exportMethod := range exportMethods {
					configDescriptor := NewConfigDescriptor(false, carthageCommand, scheme.HasXCTest, schemeHasAppClipTarget(scheme, project.Targets), exportMethod, false)
					configDescriptors = append(configDescriptors, configDescriptor)
					configOption := models.NewConfigOption(configDescriptor.ConfigName(projectType), iconIDs)

					exportMethodOption.AddConfig(exportMethod, configOption)
				}
			}
		}
	}

	// Workspaces
	for _, workspace := range workspaces {
		log.TInfof("Inspecting workspace file: %s", workspace.Pth)

		schemeOption := models.NewOption(SchemeInputTitle, SchemeInputSummary, SchemeInputEnvKey, models.TypeSelector)
		projectPathOption.AddOption(workspace.Pth, schemeOption)

		carthageCommand, warning := detectCarthageCommand(workspace.Pth)
		if warning != "" {
			warnings = append(warnings, warning)
		}

		sharedSchemes := workspace.GetSharedSchemes()
		log.TPrintf("%d shared schemes detected", len(sharedSchemes))

		if len(sharedSchemes) == 0 {
			targets := workspace.GetTargets()

			message := printMissingSharedSchemesAndGenerateWarning(workspace.Pth, defaultGitignorePth, targets)
			if message != "" {
				warnings = append(warnings, message)
			}

			// Workspace path need not exist as it could be generated by cocoapods
			for _, project := range workspace.Projects { // Not reusing targets as project path is needed
				for _, target := range project.Targets {
					exportMethodOption := models.NewOption(exportMethodInputTitle, exportMethodInputSummary, exportMethodEnvKey, models.TypeSelector)
					schemeOption.AddOption(target.Name, exportMethodOption)

					iconIDs := []string{}
					if !excludeAppIcon {
						icons, err := lookupIconByTargetName(project.Pth, target.Name, searchDir)
						if err != nil {
							log.Warnf("could not get icons for app: %s, error: %s", project.Pth, err)
							analytics.LogInfo(iconFailureTag, analytics.DetectorErrorData(string(XcodeProjectTypeIOS), err), "Failed to lookup ios icons")
						}
						iconsForAllProjects = append(iconsForAllProjects, icons...)
						for _, icon := range icons {
							iconIDs = append(iconIDs, icon.Filename)
						}
					}

					for _, exportMethod := range exportMethods {
						configDescriptor := NewConfigDescriptor(workspace.IsPodWorkspace, carthageCommand, target.HasXCTest, target.HasAppClip, exportMethod, true)
						configDescriptors = append(configDescriptors, configDescriptor)
						configOption := models.NewConfigOption(configDescriptor.ConfigName(projectType), iconIDs)

						exportMethodOption.AddConfig(exportMethod, configOption)
					}
				}
			}
		} else {
			for _, scheme := range sharedSchemes {
				log.TPrintf("- %s", scheme.Name)

				exportMethodOption := models.NewOption(exportMethodInputTitle, exportMethodInputSummary, exportMethodEnvKey, models.TypeSelector)
				schemeOption.AddOption(scheme.Name, exportMethodOption)

				iconIDs := []string{}
				if !excludeAppIcon {
					// Workspace path need not exist as it could be generated by cocoapods
					projectPathRel := projectPathByScheme(workspace.Projects, scheme.Name)
					if projectPathRel == "" {
						warningMsg := fmt.Sprintf("could not get project path (%s) for scheme (%s) and workspace (%s), error: %s",
							projectPathRel, scheme.Name, workspace.Pth, err)
						log.Warnf(warningMsg)
						warnings = append(warnings, warningMsg)
						continue
					}
					projectPath, err := filepath.Abs(filepath.Join(searchDir, projectPathRel))
					if err != nil {
						warningMsg := fmt.Sprintf("could not get absolute path, error: %s", err)
						log.Warnf(warningMsg)
						warnings = append(warnings, warningMsg)
						continue
					}

					icons, err := lookupIconBySchemeName(projectPath, scheme.Name, searchDir)
					if err != nil {
						log.Warnf("could not get icons for app: %s, error: %s", projectPath, err)
						analytics.LogInfo(iconFailureTag, analytics.DetectorErrorData(string(XcodeProjectTypeIOS), err), "Failed to lookup ios icons")
					}
					iconsForAllProjects = append(iconsForAllProjects, icons...)
					for _, icon := range icons {
						iconIDs = append(iconIDs, icon.Filename)
					}
				}

				for _, exportMethod := range exportMethods {
					// only add appclip for development and ad-hoc
					configDescriptor := NewConfigDescriptor(workspace.IsPodWorkspace, carthageCommand, scheme.HasXCTest, schemeHasAppClipTarget(scheme, workspace.GetTargets()), exportMethod, false)
					configDescriptors = append(configDescriptors, configDescriptor)
					configOption := models.NewConfigOption(configDescriptor.ConfigName(projectType), iconIDs)

					exportMethodOption.AddConfig(exportMethod, configOption)
				}
			}
		}
	}

	configDescriptors = RemoveDuplicatedConfigDescriptors(configDescriptors, projectType)

	if len(configDescriptors) == 0 {
		log.TErrorf("No valid %s config found", string(projectType))
		return models.OptionNode{}, []ConfigDescriptor{}, nil, warnings, fmt.Errorf("No valid %s config found", string(projectType))
	}

	return *projectPathOption, configDescriptors, iconsForAllProjects, warnings, nil
}

// GenerateDefaultOptions ...
func GenerateDefaultOptions(projectType XcodeProjectType) models.OptionNode {
	projectPathOption := models.NewOption(ProjectPathInputTitle, ProjectPathInputSummary, ProjectPathInputEnvKey, models.TypeUserInput)

	schemeOption := models.NewOption(SchemeInputTitle, SchemeInputSummary, SchemeInputEnvKey, models.TypeUserInput)
	projectPathOption.AddOption("", schemeOption)

	var exportMethodInputTitle string
	var exportMethodInputSummary string
	var exportMethodEnvKey string
	var exportMethods []string

	if projectType == XcodeProjectTypeIOS {
		exportMethodInputTitle = DistributionMethodInputTitle
		exportMethodInputSummary = DistributionMethodInputSummary
		exportMethodEnvKey = DistributionMethodEnvKey
		exportMethods = IosExportMethods
	} else {
		exportMethodInputTitle = ExportMethodInputTitle
		exportMethodInputSummary = ExportMethodInputSummary
		exportMethodEnvKey = ExportMethodEnvKey
		exportMethods = MacExportMethods
	}

	exportMethodOption := models.NewOption(exportMethodInputTitle, exportMethodInputSummary, exportMethodEnvKey, models.TypeSelector)
	schemeOption.AddOption("", exportMethodOption)

	for _, exportMethod := range exportMethods {
		configOption := models.NewConfigOption(fmt.Sprintf(defaultConfigNameFormat, string(projectType)), nil)
		exportMethodOption.AddConfig(exportMethod, configOption)
	}

	return *projectPathOption
}

// GenerateConfigBuilder ...
func GenerateConfigBuilder(
	projectType XcodeProjectType,
	isPrivateRepository,
	hasPodfile,
	hasTest,
	hasAppClip,
	missingSharedSchemes,
	includeCache bool,
	carthageCommand,
	exportMethod string,
) models.ConfigBuilderModel {
	configBuilder := models.NewDefaultConfigBuilder()

	params := workflowSetupParams{
		projectType:          projectType,
		configBuilder:        configBuilder,
		isPrivateRepository:  isPrivateRepository,
		includeCache:         includeCache,
		missingSharedSchemes: missingSharedSchemes,
		hasTests:             hasTest,
		hasAppClip:           hasAppClip,
		hasPodfile:           hasPodfile,
		carthageCommand:      carthageCommand,
		exportMethod:         exportMethod,
	}

	createPrimaryWorkflow(params)
	createDeployWorkflow(params)

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
func GenerateConfig(projectType XcodeProjectType, configDescriptors []ConfigDescriptor, isPrivateRepository bool) (models.BitriseConfigMap, error) {
	bitriseDataMap := models.BitriseConfigMap{}
	for _, descriptor := range configDescriptors {
		configBuilder := GenerateConfigBuilder(
			projectType,
			isPrivateRepository,
			descriptor.HasPodfile,
			descriptor.HasTest,
			descriptor.HasAppClip,
			descriptor.MissingSharedSchemes,
			true,
			descriptor.CarthageCommand,
			descriptor.ExportMethod)

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
func GenerateDefaultConfig(projectType XcodeProjectType) (models.BitriseConfigMap, error) {
	configBuilder := GenerateConfigBuilder(
		projectType,
		true,
		true,
		true,
		false,
		true,
		true,
		"",
		"")

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
