package ios

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
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

var (
	logger = utility.NewLogger()
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
	filteredFiles := utility.FilterFilesWithBasPaths(fileList, podFileBasePath)
	relevantFiles := []string{}

	for _, file := range filteredFiles {
		if !isRelevantPodfile(file) {
			continue
		}

		relevantFiles = append(relevantFiles, file)
	}

	sort.Sort(utility.ByComponents(relevantFiles))

	return relevantFiles
}

func hasTest(schemeFile string) (bool, error) {
	file, err := os.Open(schemeFile)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Warnf("Failed to close file (%s), err: %s", schemeFile, err)
		}
	}()

	testTargetExp := regexp.MustCompile(`BuildableName = ".+.xctest"`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if testTargetExp.FindString(line) != "" {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func filterProjectOrWorkspaceSharedSchemes(fileList []string, projectOrWorkspace string) ([]SchemeModel, error) {
	filteredFiles := utility.FilterFilesWithExtensions(fileList, schemeFileExtension)
	projectScharedSchemesDir := path.Join(projectOrWorkspace, "xcshareddata/xcschemes/")

	schemeFiles := []string{}
	for _, file := range filteredFiles {
		if strings.HasPrefix(file, projectScharedSchemesDir) {
			schemeFiles = append(schemeFiles, file)
		}
	}

	schemes := []SchemeModel{}
	for _, schemeFile := range schemeFiles {
		schemeWithExt := filepath.Base(schemeFile)
		ext := filepath.Ext(schemeWithExt)
		scheme := strings.TrimSuffix(schemeWithExt, ext)
		hasTest, err := hasTest(schemeFile)
		if err != nil {
			return []SchemeModel{}, err
		}

		schemes = append(schemes, SchemeModel{
			Name:    scheme,
			HasTest: hasTest,
		})
	}

	return schemes, nil
}

func workspaceProjects(workspace string) ([]string, error) {
	projects := []string{}

	xcworkspacedataPth := path.Join(workspace, "contents.xcworkspacedata")
	if exist, err := pathutil.IsPathExists(xcworkspacedataPth); err != nil {
		return []string{}, err
	} else if !exist {
		return []string{}, fmt.Errorf("contents.xcworkspacedata does not exist at: %s", xcworkspacedataPth)
	}

	xcworkspacedataStr, err := fileutil.ReadStringFromFile(xcworkspacedataPth)
	if err != nil {
		return []string{}, err
	}

	xcworkspacedataLines := strings.Split(xcworkspacedataStr, "\n")
	fileRefStart := false
	regexp := regexp.MustCompile(`location = "(.+):(.+).xcodeproj"`)

	for _, line := range xcworkspacedataLines {
		if strings.Contains(line, "<FileRef") {
			fileRefStart = true
			continue
		}

		if fileRefStart {
			fileRefStart = false
			matches := regexp.FindStringSubmatch(line)
			if len(matches) == 3 {
				projectName := matches[2]
				projects = append(projects, projectName+".xcodeproj")
			}
		}
	}

	return projects, nil
}

func isWorkspace(pth string) bool {
	return strings.HasSuffix(pth, ".xcworkspace")
}

func configName(hasPodfile, hasTest bool) string {
	name := "ios-"
	if hasPodfile {
		name = name + "pod-"
	}
	if hasTest {
		name = name + "test-"
	}
	return name + "config"
}

func defaultConfigName() string {
	return "default-ios-config"
}

//--------------------------------------------------
// Scanner
//--------------------------------------------------

// Scanner ...
type Scanner struct {
	SearchDir                     string
	FileList                      []string
	XcodeProjectAndWorkspaceFiles []string

	HasPodFile bool
	HasTest    bool
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
	logger.Info("Searching for xcodeproj files")

	xcodeProjectFiles := filterXcodeprojectFiles(fileList)
	scanner.XcodeProjectAndWorkspaceFiles = xcodeProjectFiles

	logger.InfofDetails("%d xcodeproj file(s) detected:", len(xcodeProjectFiles))
	for _, file := range xcodeProjectFiles {
		logger.InfofDetails("  - %s", file)
	}

	if len(xcodeProjectFiles) == 0 {
		logger.InfofDetails("platform not detected")

		return false, nil
	}

	logger.InfofReceipt("platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	// Check for Podfiles
	logger.InfoSection("Searching for Podfiles")
	warnings := models.Warnings{}

	podFiles := filterPodFiles(scanner.FileList)
	scanner.HasPodFile = (len(podFiles) > 0)

	logger.InfofDetails("%d Podfile(s) detected:", len(podFiles))
	for _, file := range podFiles {
		logger.InfofDetails("  - %s", file)
	}

	podfileWorkspaceProjectMap := map[string]string{}
	for _, podFile := range podFiles {
		logger.InfofSection("Inspecting Podfile: %s", podFile)

		var err error
		podfileWorkspaceProjectMap, err = utility.GetRelativeWorkspaceProjectPathMap(podFile, scanner.SearchDir)
		if err != nil {
			return models.OptionModel{}, models.Warnings{}, err
		}

		logger.InfoDetails("workspace mapping:")
		for workspace, linkedProject := range podfileWorkspaceProjectMap {
			logger.InfofDetails(" - %s -> %s", workspace, linkedProject)
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
	projectPathOption := models.NewOptionModel(projectPathTitle, projectPathEnvKey)
	isValidProjectFound := false

	for _, project := range cleanProjectFiles {
		isWorkspace := isWorkspace(project)
		if isWorkspace {
			logger.InfofSection("Inspecting workspace file: %s", project)
		} else {
			logger.InfofSection("Inspecting project file: %s", project)
		}

		schemes := []SchemeModel{}
		validProjects := []string{}

		// ---
		if isWorkspace {
			// If project is workspace (and not CocoaPods)
			// workspace shared schemes are the schared schemes inside the workspace
			// and the referred projects owned shared schemes

			// Collect workspace shared scehemes
			workspaceSchemes, err := filterProjectOrWorkspaceSharedSchemes(scanner.FileList, project)
			if err != nil {
				return models.OptionModel{}, models.Warnings{}, err
			}
			logger.InfofDetails("workspace own shared schemes: %v", workspaceSchemes)

			// Collect referred project shared scehemes
			workspaceProjects, err := workspaceProjects(project)
			if err != nil {
				return models.OptionModel{}, models.Warnings{}, err
			}

			for _, workspaceProject := range workspaceProjects {
				logger.InfofDetails("inspecting referred project: %s", workspaceProject)
				workspaceProjectSchemes, err := filterProjectOrWorkspaceSharedSchemes(scanner.FileList, workspaceProject)
				if err != nil {
					return models.OptionModel{}, models.Warnings{}, err
				}

				workspaceSchemes = append(workspaceSchemes, workspaceProjectSchemes...)
				logger.InfofDetails("  referred project's shared schemes: %v", workspaceProjectSchemes)
			}

			validProjects = []string{project}
			schemes = workspaceSchemes
		} else {
			validProjectMap := map[string]bool{}
			found := utility.MapStringStringHasValue(podfileWorkspaceProjectMap, project)
			if found {
				// CocoaPods will generate a workspace for this project
				for workspace, linkedProject := range podfileWorkspaceProjectMap {
					if linkedProject == project {
						logger.InfofDetails("workspace will be generated by CocoaPods: %s", workspace)
						// We should use the generated workspace instead of the project
						validProjectMap[workspace] = true
					}
				}
			} else {
				// Standalone project
				validProjectMap[project] = true
			}

			for p := range validProjectMap {
				validProjects = append(validProjects, p)
			}

			projectSchemes, err := filterProjectOrWorkspaceSharedSchemes(scanner.FileList, project)
			if err != nil {
				return models.OptionModel{}, models.Warnings{}, err
			}

			schemes = projectSchemes
		}
		// ---

		logger.InfofReceipt("valid projects: %v", validProjects)
		logger.InfofReceipt("found shared schemes: %v", schemes)

		if len(schemes) == 0 {
			log.Warn("No shared scheme found")
			warnings = append(warnings, fmt.Sprintf("no shared scheme found for project: %s", project))
			continue
		}

		isValidProjectFound = true

		for _, validProject := range validProjects {
			schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)
			for _, scheme := range schemes {
				hasTest := scheme.HasTest
				if hasTest {
					scanner.HasTest = true
				}

				configOption := models.NewEmptyOptionModel()
				configOption.Config = configName(scanner.HasPodFile, hasTest)

				schemeOption.ValueMap[scheme.Name] = configOption
			}

			projectPathOption.ValueMap[validProject] = schemeOption
		}
	}

	if !isValidProjectFound {
		projectPathOption = models.NewEmptyOptionModel()
	}

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

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
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
	if scanner.HasPodFile {
		stepList = append(stepList, steps.CocoapodsInstallStepListItem())
	}

	if scanner.HasTest {
		// XcodeTest
		inputs := []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
			envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
		}

		stepListWithTest := append(stepList, steps.XcodeTestStepListItem(inputs))

		// XcodeArchive
		stepListWithTest = append(stepListWithTest, steps.XcodeArchiveStepListItem(inputs))

		// DeployToBitriseIo
		stepListWithTest = append(stepListWithTest, steps.DeployToBitriseIoStepListItem())

		bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(stepListWithTest)
		data, err := yaml.Marshal(bitriseData)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		configName := configName(scanner.HasPodFile, true)
		bitriseDataMap[configName] = string(data)
	}

	// XcodeArchive
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
	}

	stepList = append(stepList, steps.XcodeArchiveStepListItem(inputs))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := configName(scanner.HasPodFile, false)
	bitriseDataMap[configName] = string(data)

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

	// XcodeArchive
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
	}

	stepList = append(stepList, steps.XcodeArchiveStepListItem(inputs))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := defaultConfigName()
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}
