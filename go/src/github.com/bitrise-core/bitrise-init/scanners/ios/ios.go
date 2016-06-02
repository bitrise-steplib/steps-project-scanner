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

func filterXcodeprojectFiles(fileList []string) []string {
	filteredFiles := utility.FilterFilesWithExtensions(fileList, xcodeprojExtension, xcworkspaceExtension)

	relevantFiles := []string{}
	workspaceEmbeddedInProjectExp := regexp.MustCompile(`.+.xcodeproj/.+.xcworkspace`)
	podProjectExp := regexp.MustCompile(`.*/Pods/.+.xcodeproj`)

	for _, file := range filteredFiles {
		isWorkspaceEmbeddedInProject := false
		if workspaceEmbeddedInProjectExp.FindString(file) != "" {
			isWorkspaceEmbeddedInProject = true
		}

		isPodProject := false
		if podProjectExp.FindString(file) != "" {
			isPodProject = true
		}

		if !isWorkspaceEmbeddedInProject && !isPodProject {
			relevantFiles = append(relevantFiles, file)
		}
	}

	sort.Sort(utility.ByComponents(relevantFiles))

	return relevantFiles
}

func filterPodFiles(fileList []string) []string {
	filteredFiles := utility.FilterFilesWithBasPaths(fileList, podFileBasePath)
	relevantFiles := []string{}

	for _, file := range filteredFiles {
		if !strings.Contains(file, ".git/") {
			relevantFiles = append(relevantFiles, file)
		}
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

func filterSharedSchemes(fileList []string, project string) ([]SchemeModel, error) {
	filteredFiles := utility.FilterFilesWithExtensions(fileList, schemeFileExtension)
	projectScharedSchemesDir := path.Join(project, "xcshareddata/xcschemes/")

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
	SearchDir         string
	FileList          []string
	XcodeProjectFiles []string

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

	// Search for xcodeproj/xcworkspace file
	logger.Info("Searching for xcodeproj/xcworkspace files")

	xcodeProjectFiles := filterXcodeprojectFiles(fileList)
	scanner.XcodeProjectFiles = xcodeProjectFiles

	logger.InfofDetails("%d xcodeproj/xcworkspace file(s) detected", len(xcodeProjectFiles))

	if len(xcodeProjectFiles) == 0 {
		logger.InfofDetails("platform not detected")

		return false, nil
	}

	logger.InfofReceipt("platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, error) {
	// Check for Podfiles
	logger.InfoSection("Searching for Podfiles")

	podFiles := filterPodFiles(scanner.FileList)
	scanner.HasPodFile = (len(podFiles) > 0)

	logger.InfofDetails("%d Podfile(s) detected", len(podFiles))

	workspaceMap := map[string]string{}
	for _, podFile := range podFiles {
		logger.InfofSection("Inspecting Podfile: %s", podFile)

		if err := os.Setenv("pod_file_path", podFile); err != nil {
			return models.OptionModel{}, err
		}

		podfileWorkspaceMap, err := utility.GetWorkspaces(scanner.SearchDir)
		if err != nil {
			return models.OptionModel{}, err
		}

		logger.InfofDetails("result workspace map: %v", podfileWorkspaceMap)

		for workspace, project := range podfileWorkspaceMap {
			workspaceMap[workspace] = project
		}
	}

	// Check if project is generated by Pod
	validProjects := []string{}
	for _, project := range scanner.XcodeProjectFiles {
		_, found := workspaceMap[project]

		if found {
			logger.InfofDetails("workspace will be generated by CocoaPods: %s", project)
			for _, linkedProject := range workspaceMap {
				if linkedProject == project {
					validProjects = append(validProjects, project)
				}
			}
		} else {
			validProjects = append(validProjects, project)
		}
	}

	logger.InfofReceipt("standalone projects: %v", validProjects)

	projectPathOption := models.NewOptionModel(projectPathTitle, projectPathEnvKey)

	// Inspect projects
	for _, project := range validProjects {
		logger.InfofSection("Inspecting project file: %s", project)

		schemes, err := filterSharedSchemes(scanner.FileList, project)
		if err != nil {
			return models.OptionModel{}, err
		}

		logger.InfofReceipt("found schemes: %v", schemes)

		if len(schemes) == 0 {
			log.Warn("No shared scheme found")
			continue
		}

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

		projectPathOption.ValueMap[project] = schemeOption
	}

	return projectPathOption, nil
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
func (scanner *Scanner) Configs() (map[string]string, error) {
	bitriseDataMap := map[string]string{}
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

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
			return map[string]string{}, err
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
		return map[string]string{}, err
	}

	configName := configName(scanner.HasPodFile, false)
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (map[string]string, error) {
	bitriseDataMap := map[string]string{}
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

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
		return map[string]string{}, err
	}

	configName := defaultConfigName()
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}
