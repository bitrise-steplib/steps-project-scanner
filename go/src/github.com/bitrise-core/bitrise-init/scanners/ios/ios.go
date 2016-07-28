package ios

import (
	"errors"
	"fmt"
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

// SchemeModel ...
type SchemeModel struct {
	Name    string
	HasTest bool
}

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

func isRelevantProject(pth string) bool {
	for _, regexp := range scanProjectPathRegexpBlackList {
		if isPathMatchRegexp(pth, regexp) {
			return false
		}
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

func filterXcodeprojectFiles(fileList []string) []string {
	filteredFiles := utility.FilterFilesWithExtensions(fileList, xcodeprojExtension, xcworkspaceExtension)
	relevantFiles := []string{}

	for _, file := range filteredFiles {
		if !isRelevantProject(file) {
			continue
		}

		relevantFiles = append(relevantFiles, file)
	}

	sort.Sort(utility.ByComponents(relevantFiles))

	return relevantFiles
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
	log.Info("Searching for xcodeproj files")

	xcodeProjectFiles := filterXcodeprojectFiles(fileList)
	scanner.XcodeProjectAndWorkspaceFiles = xcodeProjectFiles

	log.InfofDetails("%d xcodeproj file(s) detected:", len(xcodeProjectFiles))
	for _, file := range xcodeProjectFiles {
		log.InfofDetails("  - %s", file)
	}

	if len(xcodeProjectFiles) == 0 {
		log.InfofDetails("platform not detected")

		return false, nil
	}

	log.InfofReceipt("platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	// Check for Podfiles
	log.InfoSection("Searching for Podfiles")
	warnings := models.Warnings{}

	podFiles := filterPodFiles(scanner.FileList)

	log.InfofDetails("%d Podfile(s) detected:", len(podFiles))
	for _, file := range podFiles {
		log.InfofDetails("  - %s", file)
	}

	podfileWorkspaceProjectMap := map[string]string{}
	for _, podFile := range podFiles {
		log.InfofSection("Inspecting Podfile: %s", podFile)

		var err error
		podfileWorkspaceProjectMap, err = utility.GetRelativeWorkspaceProjectPathMap(podFile, scanner.SearchDir)
		if err != nil {
			log.Warnf("Analyze Podfile (%s) failed", podFile)
			if podfileContent, err := fileutil.ReadStringFromFile(podFile); err != nil {
				log.Warnf("Failed to read Podfile (%s)", podFile)
			} else {
				fmt.Println(podfileContent)
				fmt.Println("")
			}
			return models.OptionModel{}, models.Warnings{}, err
		}

		log.InfoDetails("workspace mapping:")
		for workspace, linkedProject := range podfileWorkspaceProjectMap {
			log.InfofDetails(" - %s -> %s", workspace, linkedProject)
		}
	}

	// Remove CocoaPods workspaces
	cleanProjectFiles := []string{}
	for _, projectOrWorkspace := range scanner.XcodeProjectAndWorkspaceFiles {
		// workspace will generated by CocoaPods
		_, found := podfileWorkspaceProjectMap[projectOrWorkspace]
		if !found {
			cleanProjectFiles = append(cleanProjectFiles, projectOrWorkspace)
		}
	}

	// Inspect Projects
	configDescriptors := []ConfigDescriptor{}
	projectPathOption := models.NewOptionModel(projectPathTitle, projectPathEnvKey)

	for _, project := range cleanProjectFiles {
		isWorkspace := xcodeproj.IsXCWorkspace(project)

		if isWorkspace {
			log.InfofSection("Inspecting workspace file: %s", project)
		} else {
			log.InfofSection("Inspecting project file: %s", project)
		}

		validProjectMap := map[string]bool{}
		schemeXCTestMap := map[string]bool{}

		missingSharedSchemes := false
		hasPodFile := false

		// ---
		if isWorkspace {
			// Collect workspace shared scehemes
			workspaceSchemeXCTestMap, err := xcodeproj.WorkspaceSharedSchemes(project)
			if err != nil {
				return models.OptionModel{}, models.Warnings{}, err
			}

			log.InfofDetails("workspace shared schemes: %v", workspaceSchemeXCTestMap)

			if len(workspaceSchemeXCTestMap) == 0 {
				log.Warnf("No shared schemes found, adding recreate-user-schemes step...")

				warnings = append(warnings, fmt.Sprintf("no shared scheme found for project: %s", project))
				missingSharedSchemes = true

				targetXCTestMap, err := xcodeproj.WorkspaceTargets(project)
				if err != nil {
					return models.OptionModel{}, models.Warnings{}, err
				}

				log.InfofDetails("workspace user schemes: %v", targetXCTestMap)

				workspaceSchemeXCTestMap = targetXCTestMap
			}

			validProjectMap[project] = true
			schemeXCTestMap = workspaceSchemeXCTestMap
		} else {
			found := utility.MapStringStringHasValue(podfileWorkspaceProjectMap, project)
			if found {
				// CocoaPods will generate a workspace for this project
				hasPodFile = true

				for workspace, linkedProject := range podfileWorkspaceProjectMap {
					if linkedProject == project {
						log.InfofDetails("workspace will be generated by CocoaPods: %s", workspace)
						// We should use the generated workspace instead of the project
						validProjectMap[workspace] = true
					}
				}
			} else {
				// Standalone project
				validProjectMap[project] = true
			}

			projectSchemeXCtestMap, err := xcodeproj.ProjectSharedSchemes(project)
			if err != nil {
				return models.OptionModel{}, models.Warnings{}, err
			}

			log.InfofDetails("project shared schemes: %v", projectSchemeXCtestMap)

			if len(projectSchemeXCtestMap) == 0 {
				log.Warnf("No shared schemes found, adding recreate-user-schemes step...")

				warnings = append(warnings, fmt.Sprintf("no shared scheme found for project: %s", project))
				missingSharedSchemes = true

				targetXCTestMap, err := xcodeproj.ProjectTargets(project)
				if err != nil {
					return models.OptionModel{}, models.Warnings{}, err
				}

				log.InfofDetails("project user schemes: %v", targetXCTestMap)

				projectSchemeXCtestMap = targetXCTestMap
			}

			schemeXCTestMap = projectSchemeXCtestMap
		}
		// ---

		log.InfofReceipt("found schemes: %v", schemeXCTestMap)

		if len(schemeXCTestMap) == 0 {
			return models.OptionModel{}, models.Warnings{}, errors.New("No shared schemes found, or failed to create user schemes")
		}

		for validProject := range validProjectMap {
			schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)
			for schemeName, hasXCtest := range schemeXCTestMap {
				configDescriptor := ConfigDescriptor{
					HasPodfile:           hasPodFile,
					HasTest:              hasXCtest,
					MissingSharedSchemes: missingSharedSchemes,
				}
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewEmptyOptionModel()
				configOption.Config = configDescriptor.String()

				schemeOption.ValueMap[schemeName] = configOption
			}

			projectPathOption.ValueMap[validProject] = schemeOption
		}
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
