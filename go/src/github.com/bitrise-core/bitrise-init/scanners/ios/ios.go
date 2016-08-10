package ios

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/xcode-utils/xcodeproj"
)

var (
	log = utility.NewLogger()
)

const (
	scannerName = "ios"
)

const (
	xcodeprojExtension   = ".xcodeproj"
	xcworkspaceExtension = ".xcworkspace"
	podFileBasePath      = "Podfile"
	schemeFileExtension  = ".xcscheme"
)

const (
	projectPathKey    = "project_path"
	projectPathTitle  = "Project (or Workspace) path"
	projectPathEnvKey = "BITRISE_PROJECT_PATH"

	schemeKey    = "scheme"
	schemeTitle  = "Scheme name"
	schemeEnvKey = "BITRISE_SCHEME"
)

var (
	embeddedWorkspacePathRegexp    = regexp.MustCompile(`.+\.xcodeproj/.+\.xcworkspace`)
	scanProjectPathRegexpBlackList = []*regexp.Regexp{embeddedWorkspacePathRegexp}

	gitFolderName           = ".git"
	podsFolderName          = "Pods"
	carthageFolderName      = "Carthage"
	scanFolderNameBlackList = []string{gitFolderName, podsFolderName, carthageFolderName}

	frameworkExt           = ".framework"
	scanFolderExtBlackList = []string{frameworkExt}
)

//--------------------------------------------------
// Utility
//--------------------------------------------------

func isPathMatchRegexp(pth string, regexp *regexp.Regexp) bool {
	return (regexp.FindString(pth) != "")
}

func isPathContainsComponent(pth, component string) bool {
	pathComponents := strings.Split(pth, string(filepath.Separator))
	for _, c := range pathComponents {
		if c == component {
			return true
		}
	}
	return false
}

func isPathContainsComponentWithExtension(pth, ext string) bool {
	pathComponents := strings.Split(pth, string(filepath.Separator))
	for _, c := range pathComponents {
		e := filepath.Ext(c)
		if e == ext {
			return true
		}
	}
	return false
}

func isDir(pth string) (bool, error) {
	fileInf, err := os.Lstat(pth)
	if err != nil {
		return false, err
	}
	if fileInf == nil {
		return false, errors.New("no file info available")
	}
	return fileInf.IsDir(), nil
}

func isRelevantProject(pth string, isTest bool) (bool, error) {
	// xcodeproj & xcworkspace should be a dir
	if !isTest {
		if is, err := isDir(pth); err != nil {
			return false, err
		} else if !is {
			return false, nil
		}
	}

	for _, regexp := range scanProjectPathRegexpBlackList {
		if isPathMatchRegexp(pth, regexp) {
			return false, nil
		}
	}

	for _, folderName := range scanFolderNameBlackList {
		if isPathContainsComponent(pth, folderName) {
			return false, nil
		}
	}

	for _, folderExt := range scanFolderExtBlackList {
		if isPathContainsComponentWithExtension(pth, folderExt) {
			return false, nil
		}
	}

	return true, nil
}

func filterXcodeprojectFiles(fileList []string, isTest bool) ([]string, error) {
	filteredFiles := utility.FilterFilesWithExtensions(fileList, xcodeprojExtension, xcworkspaceExtension)
	relevantFiles := []string{}

	for _, file := range filteredFiles {
		if is, err := isRelevantProject(file, isTest); err != nil {
			return []string{}, err
		} else if !is {
			continue
		}

		relevantFiles = append(relevantFiles, file)
	}

	sort.Sort(utility.ByComponents(relevantFiles))

	return relevantFiles, nil
}

func isRelevantPodfile(pth string) bool {
	basename := filepath.Base(pth)
	if !utility.CaseInsensitiveEquals(basename, "podfile") {
		return false
	}

	for _, folderName := range scanFolderNameBlackList {
		if isPathContainsComponent(pth, folderName) {
			return false
		}
	}

	for _, folderExt := range scanFolderExtBlackList {
		if isPathContainsComponentWithExtension(pth, folderExt) {
			return false
		}
	}

	return true
}

func filterPodFiles(fileList []string) []string {
	podfiles := []string{}

	for _, file := range fileList {
		if isRelevantPodfile(file) {
			podfiles = append(podfiles, file)
		}
	}

	if len(podfiles) == 0 {
		return []string{}
	}

	sort.Sort(utility.ByComponents(podfiles))

	return podfiles
}

func configName(hasPodfile, hasTest, missingSharedSchemes bool) string {
	name := "ios-"
	if hasPodfile {
		name = name + "pod-"
	}
	if hasTest {
		name = name + "test-"
	}
	if missingSharedSchemes {
		name = name + "missing-shared-schemes-"
	}
	return name + "config"
}

func defaultConfigName() string {
	return "default-ios-config"
}

//--------------------------------------------------
// Scanner
//--------------------------------------------------

// ConfigDescriptor ...
type ConfigDescriptor struct {
	HasPodfile           bool
	HasTest              bool
	MissingSharedSchemes bool
}

func (descriptor ConfigDescriptor) String() string {
	name := "ios-"
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
	SearchDir                     string
	FileList                      []string
	XcodeProjectAndWorkspaceFiles []string

	configDescriptors []ConfigDescriptor
}

// Name ...
func (scanner Scanner) Name() string {
	return scannerName
}

// Configure ...
func (scanner *Scanner) Configure(searchDir string) {
	scanner.SearchDir = searchDir
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform() (bool, error) {
	fileList, err := utility.FileList(scanner.SearchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", scanner.SearchDir, err)
	}
	scanner.FileList = fileList

	// Search for xcodeproj file
	log.Info("Searching for .xcodeproj & .xcworkspace files")

	xcodeProjectFiles, err := filterXcodeprojectFiles(fileList, false)
	if err != nil {
		return false, fmt.Errorf("failed to collect .xcodeproj & .xcworkspace files, error: %s", err)
	}
	scanner.XcodeProjectAndWorkspaceFiles = xcodeProjectFiles

	log.Details("%d project file(s) detected", len(xcodeProjectFiles))
	for _, file := range xcodeProjectFiles {
		log.Details("- %s", file)
	}

	if len(xcodeProjectFiles) == 0 {
		log.Details("platform not detected")

		return false, nil
	}

	log.Done("Platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	//
	// Create Pod workspace - project mapping
	log.Info("Searching for Podfiles")
	warnings := models.Warnings{}

	podFiles := filterPodFiles(scanner.FileList)

	log.Details("%d Podfile(s) detected", len(podFiles))
	for _, file := range podFiles {
		log.Details("- %s", file)
	}

	podfileWorkspaceProjectMap := map[string]string{}
	for _, podFile := range podFiles {
		log.Info("Inspecting Podfile: %s", podFile)

		var err error
		podfileWorkspaceProjectMap, err = utility.GetRelativeWorkspaceProjectPathMap(podFile, scanner.SearchDir)
		if err != nil {
			log.Warn("Analyze Podfile (%s) failed", podFile)
			if podfileContent, err := fileutil.ReadStringFromFile(podFile); err != nil {
				log.Warn("Failed to read Podfile (%s)", podFile)
			} else {
				fmt.Println(podfileContent)
				fmt.Println("")
			}
			return models.OptionModel{}, models.Warnings{}, err
		}

		log.Details("workspace mapping:")
		for workspace, linkedProject := range podfileWorkspaceProjectMap {
			log.Details("- %s -> %s", workspace, linkedProject)
		}
	}
	// -----

	//
	// Separate projects and workspaces
	log.Info("Separate projects and workspaces")
	projects := []ProjectModel{}
	workspaces := []WorkspaceModel{}

	for _, workspaceOrProjectPth := range scanner.XcodeProjectAndWorkspaceFiles {
		if xcodeproj.IsXCodeProj(workspaceOrProjectPth) {
			project := ProjectModel{Pth: workspaceOrProjectPth}
			projects = append(projects, project)
		} else {
			workspace := WorkspaceModel{Pth: workspaceOrProjectPth}
			workspaces = append(workspaces, workspace)
		}
	}
	// -----

	//
	// Separate standalone projects, standalone workspaces and pod projects
	standaloneProjects := []ProjectModel{}
	standaloneWorkspaces := []WorkspaceModel{}
	podProjects := []ProjectModel{}

	for _, project := range projects {
		if !utility.MapStringStringHasValue(podfileWorkspaceProjectMap, project.Pth) {
			standaloneProjects = append(standaloneProjects, project)
		}
	}

	log.Details("%d Standalone project(s) detected", len(standaloneProjects))
	for _, project := range standaloneProjects {
		log.Details("- %s", project.Pth)
	}

	for _, workspace := range workspaces {
		if _, found := podfileWorkspaceProjectMap[workspace.Pth]; !found {
			standaloneWorkspaces = append(standaloneWorkspaces, workspace)
		}
	}

	log.Details("%d Standalone workspace(s) detected", len(standaloneWorkspaces))
	for _, workspace := range standaloneWorkspaces {
		log.Details("- %s", workspace.Pth)
	}

	for podWorkspacePth, linkedProjectPth := range podfileWorkspaceProjectMap {
		project, found := FindProjectWithPth(projects, linkedProjectPth)
		if !found {
			log.Warn("workspace mapping contains project (%s), but not found in project list", linkedProjectPth)
			continue
		}

		workspace, found := FindWorkspaceWithPth(workspaces, podWorkspacePth)
		if !found {
			workspace = WorkspaceModel{Pth: podWorkspacePth}
		}

		workspace.GeneratedByPod = true

		project.PodWorkspace = workspace
		podProjects = append(podProjects, project)
	}

	log.Details("%d Pod project(s) detected", len(podProjects))
	for _, project := range podProjects {
		log.Details("- %s -> %s", project.Pth, project.PodWorkspace.Pth)
	}
	// -----

	//
	// Analyze projects and workspaces
	analyzedProjects := []ProjectModel{}
	analyzedWorkspaces := []WorkspaceModel{}

	for _, project := range standaloneProjects {
		log.Info("Inspecting standalone project file: %s", project.Pth)

		schemes := []SchemeModel{}

		schemeXCtestMap, err := xcodeproj.ProjectSharedSchemes(project.Pth)
		if err != nil {
			log.Warn("Failed to get shared schemes, error: %s", err)
			continue
		}

		log.Details("%d shared scheme(s) detected", len(schemeXCtestMap))
		for scheme, hasXCTest := range schemeXCtestMap {
			log.Details("- %s", scheme)

			schemes = append(schemes, SchemeModel{Name: scheme, HasXCTest: hasXCTest, Shared: true})
		}

		if len(schemeXCtestMap) == 0 {
			log.Details("")
			log.Error("No shared schemes found, adding recreate-user-schemes step...")
			log.Error("The newly generated schemes, may differs from the ones in your project.")
			log.Error("Make sure to share your schemes, to have the expected behaviour.")
			log.Details("")

			warnings = append(warnings, fmt.Sprintf("no shared scheme found for project: %s", project.Pth))

			targetXCTestMap, err := xcodeproj.ProjectTargets(project.Pth)
			if err != nil {
				log.Warn("Failed to get targets, error: %s", err)
				continue
			}

			log.Warn("%d user scheme(s) will be generated", len(targetXCTestMap))
			for target, hasXCTest := range targetXCTestMap {
				log.Warn("- %s", target)

				schemes = append(schemes, SchemeModel{Name: target, HasXCTest: hasXCTest, Shared: false})
			}
		}

		project.Schemes = schemes
		analyzedProjects = append(analyzedProjects, project)
	}

	for _, workspace := range standaloneWorkspaces {
		log.Info("Inspecting standalone workspace file: %s", workspace.Pth)

		schemes := []SchemeModel{}

		schemeXCtestMap, err := xcodeproj.WorkspaceSharedSchemes(workspace.Pth)
		if err != nil {
			log.Warn("Failed to get shared schemes, error: %s", err)
			continue
		}

		log.Details("%d shared scheme(s) detected", len(schemeXCtestMap))
		for scheme, hasXCTest := range schemeXCtestMap {
			log.Details("- %s", scheme)

			schemes = append(schemes, SchemeModel{Name: scheme, HasXCTest: hasXCTest, Shared: true})
		}

		if len(schemeXCtestMap) == 0 {
			log.Details("")
			log.Error("No shared schemes found, adding recreate-user-schemes step...")
			log.Error("The newly generated schemes, may differs from the ones in your project.")
			log.Error("Make sure to share your schemes, to have the expected behaviour.")
			log.Details("")

			warnings = append(warnings, fmt.Sprintf("no shared scheme found for project: %s", workspace.Pth))

			targetXCTestMap, err := xcodeproj.WorkspaceTargets(workspace.Pth)
			if err != nil {
				log.Warn("Failed to get targets, error: %s", err)
				continue
			}

			log.Warn("%d user scheme(s) will be generated", len(targetXCTestMap))
			for target, hasXCTest := range targetXCTestMap {
				log.Warn("- %s", target)

				schemes = append(schemes, SchemeModel{Name: target, HasXCTest: hasXCTest, Shared: false})
			}
		}

		workspace.Schemes = schemes
		analyzedWorkspaces = append(analyzedWorkspaces, workspace)
	}

	for _, project := range podProjects {
		log.Info("Inspecting pod project file: %s", project.Pth)

		schemes := []SchemeModel{}

		schemeXCtestMap, err := xcodeproj.ProjectSharedSchemes(project.Pth)
		if err != nil {
			log.Warn("Failed to get shared schemes, error: %s", err)
			continue
		}

		log.Details("%d shared scheme(s) detected", len(schemeXCtestMap))
		for scheme, hasXCTest := range schemeXCtestMap {
			log.Details("- %s", scheme)

			schemes = append(schemes, SchemeModel{Name: scheme, HasXCTest: hasXCTest, Shared: true})
		}

		if len(schemeXCtestMap) == 0 {
			log.Details("")
			log.Error("No shared schemes found, adding recreate-user-schemes step...")
			log.Error("The newly generated schemes, may differs from the ones in your project.")
			log.Error("Make sure to share your schemes, to have the expected behaviour.")
			log.Details("")

			warnings = append(warnings, fmt.Sprintf("no shared scheme found for project: %s", project.Pth))

			targetXCTestMap, err := xcodeproj.ProjectTargets(project.Pth)
			if err != nil {
				log.Warn("Failed to get targets, error: %s", err)
				continue
			}

			log.Warn("%d user scheme(s) will be generated", len(targetXCTestMap))
			for target, hasXCTest := range targetXCTestMap {
				log.Warn("- %s", target)

				schemes = append(schemes, SchemeModel{Name: target, HasXCTest: hasXCTest, Shared: false})
			}
		}

		project.PodWorkspace.Schemes = schemes
		analyzedWorkspaces = append(analyzedWorkspaces, project.PodWorkspace)
	}
	// -----

	//
	// Create config descriptors
	configDescriptors := []ConfigDescriptor{}
	projectPathOption := models.NewOptionModel(projectPathTitle, projectPathEnvKey)

	for _, project := range analyzedProjects {
		schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)

		for _, scheme := range project.Schemes {
			configDescriptor := ConfigDescriptor{
				HasPodfile:           false,
				HasTest:              scheme.HasXCTest,
				MissingSharedSchemes: !scheme.Shared,
			}
			configDescriptors = append(configDescriptors, configDescriptor)

			configOption := models.NewEmptyOptionModel()
			configOption.Config = configDescriptor.String()

			schemeOption.ValueMap[scheme.Name] = configOption
		}

		projectPathOption.ValueMap[project.Pth] = schemeOption
	}

	for _, workspace := range analyzedWorkspaces {
		schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)

		for _, scheme := range workspace.Schemes {
			configDescriptor := ConfigDescriptor{
				HasPodfile:           workspace.GeneratedByPod,
				HasTest:              scheme.HasXCTest,
				MissingSharedSchemes: !scheme.Shared,
			}
			configDescriptors = append(configDescriptors, configDescriptor)

			configOption := models.NewEmptyOptionModel()
			configOption.Config = configDescriptor.String()

			schemeOption.ValueMap[scheme.Name] = configOption
		}

		projectPathOption.ValueMap[workspace.Pth] = schemeOption
	}
	// -----

	if len(configDescriptors) == 0 {
		return models.OptionModel{}, warnings, errors.New("No valid config found")
	}

	scanner.configDescriptors = configDescriptors

	return projectPathOption, warnings, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionModel {
	configOption := models.NewEmptyOptionModel()
	configOption.Config = defaultConfigName()

	projectPathOption := models.NewOptionModel(projectPathTitle, projectPathEnvKey)
	schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)

	schemeOption.ValueMap["_"] = configOption
	projectPathOption.ValueMap["_"] = schemeOption

	return projectPathOption
}

func generateConfig(hasPodfile, hasTest, missingSharedSchemes bool) bitriseModels.BitriseDataModel {
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// Script
	stepList = append(stepList, steps.ScriptSteplistItem())

	// CertificateAndProfileInstaller
	stepList = append(stepList, steps.CertificateAndProfileInstallerStepListItem())

	// CocoapodsInstall
	if hasPodfile {
		stepList = append(stepList, steps.CocoapodsInstallStepListItem())
	}

	// RecreateUserSchemes
	if missingSharedSchemes {
		stepList = append(stepList, steps.RecreateUserSchemesStepListItem([]envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		}))
	}

	// XcodeTest
	if hasTest {
		stepList = append(stepList, steps.XcodeTestStepListItem([]envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
			envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
		}))
	}

	// XcodeArchive
	stepList = append(stepList, steps.XcodeArchiveStepListItem([]envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
	}))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	return models.BitriseDataWithDefaultTriggerMapAndPrimaryWorkflowSteps(stepList)
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
	bitriseDataMap := models.BitriseConfigMap{}
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// Script
	stepList = append(stepList, steps.ScriptSteplistItem())

	// CertificateAndProfileInstaller
	stepList = append(stepList, steps.CertificateAndProfileInstallerStepListItem())

	// CocoapodsInstall
	stepList = append(stepList, steps.CocoapodsInstallStepListItem())

	stepList = append(stepList, steps.RecreateUserSchemesStepListItem([]envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
	}))

	// XcodeArchive
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
	}

	// RecreateUserSchemes
	stepList = append(stepList, steps.XcodeArchiveStepListItem(inputs))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	bitriseData := models.BitriseDataWithDefaultTriggerMapAndPrimaryWorkflowSteps(stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := defaultConfigName()
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}
