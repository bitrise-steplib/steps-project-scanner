package scanners

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	stepmanModels "github.com/bitrise-io/stepman/models"
)

const (
	iosDetectorName = "ios"
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

	stepCocoapodsInstallIDComposite = "cocoapods-install@1.1.0"
	stepXcodeArchiveIDComposite     = "xcode-archive@1.7.0"
	stepXcodeTestIDComposite        = "xcode-test@1.13.3"
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

func iOSConfigName(hasPodfile, hasTest bool) string {
	name := "ios-"
	if hasPodfile {
		name = name + "pod-"
	}
	if hasTest {
		name = name + "test-"
	}
	return name + "config"
}

//--------------------------------------------------
// Detector
//--------------------------------------------------

// Ios ...
type Ios struct {
	SearchDir         string
	FileList          []string
	XcodeProjectFiles []string

	HasPodFile bool
	HasTest    bool
}

// Name ...
func (detector Ios) Name() string {
	return iosDetectorName
}

// Configure ...
func (detector *Ios) Configure(searchDir string) {
	detector.SearchDir = searchDir
}

// DetectPlatform ...
func (detector *Ios) DetectPlatform() (bool, error) {
	fileList, err := utility.FileList(detector.SearchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", detector.SearchDir, err)
	}
	detector.FileList = fileList

	// Search for xcodeproj/xcworkspace file
	logger.InfoSection("Searching for xcodeproj/xcworkspace files")

	xcodeProjectFiles := filterXcodeprojectFiles(fileList)
	detector.XcodeProjectFiles = xcodeProjectFiles

	logger.InfofDetails("%d xcodeproj/xcworkspace files detected", len(xcodeProjectFiles))

	if len(xcodeProjectFiles) == 0 {
		logger.InfofDetails("platform not detected")

		return false, nil
	}

	logger.InfofReceipt("platform detected")

	return true, nil
}

// Analyze ...
func (detector *Ios) Analyze() (models.OptionModel, error) {
	// Check for Podfiles
	logger.InfoSection("Searching for Podfiles")

	podFiles := filterPodFiles(detector.FileList)
	detector.HasPodFile = (len(podFiles) > 0)

	logger.InfofDetails("%d Podfiles detected", len(podFiles))

	workspaceMap := map[string]string{}
	for _, podFile := range podFiles {
		logger.InfofSection("Inspecting Podfile: %s", podFile)

		if err := os.Setenv("pod_file_path", podFile); err != nil {
			return models.OptionModel{}, err
		}

		podfileWorkspaceMap, err := utility.GetWorkspaces(detector.SearchDir)
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
	for _, project := range detector.XcodeProjectFiles {
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

		schemes, err := filterSharedSchemes(detector.FileList, project)
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
				detector.HasTest = true
			}

			configOption := models.NewEmptyOptionModel()
			configOption.Config = iOSConfigName(detector.HasPodFile, hasTest)

			schemeOption.ValueMap[scheme.Name] = configOption
		}

		projectPathOption.ValueMap[project] = schemeOption
	}

	return projectPathOption, nil
}

// Configs ...
func (detector *Ios) Configs(isPrivate bool) map[string]bitriseModels.BitriseDataModel {
	bitriseDataMap := map[string]bitriseModels.BitriseDataModel{}
	steps := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	if isPrivate {
		steps = append(steps, bitriseModels.StepListItemModel{
			stepActivateSSHKeyIDComposite: stepmanModels.StepModel{},
		})
	}

	// GitClone
	steps = append(steps, bitriseModels.StepListItemModel{
		stepGitCloneIDComposite: stepmanModels.StepModel{},
	})

	// CertificateAndProfileInstaller
	steps = append(steps, bitriseModels.StepListItemModel{
		stepCertificateAndProfileInstallerIDComposite: stepmanModels.StepModel{},
	})

	// CocoapodsInstall
	if detector.HasPodFile {
		steps = append(steps, bitriseModels.StepListItemModel{
			stepCocoapodsInstallIDComposite: stepmanModels.StepModel{},
		})
	}

	// XcodeTest
	if detector.HasTest {
		inputs := []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
			envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
		}

		stepsWithTest := append(steps, bitriseModels.StepListItemModel{
			stepXcodeTestIDComposite: stepmanModels.StepModel{
				Inputs: inputs,
			},
		})

		// XcodeArchive
		stepsWithTest = append(stepsWithTest, bitriseModels.StepListItemModel{
			stepXcodeArchiveIDComposite: stepmanModels.StepModel{
				Inputs: inputs,
			},
		})

		// DeployToBitriseIo
		stepsWithTest = append(stepsWithTest, bitriseModels.StepListItemModel{
			stepDeployToBitriseIoIDComposite: stepmanModels.StepModel{},
		})

		workflows := map[string]bitriseModels.WorkflowModel{
			"primary": bitriseModels.WorkflowModel{
				Steps: stepsWithTest,
			},
		}

		bitriseData := bitriseModels.BitriseDataModel{
			Workflows:            workflows,
			FormatVersion:        "1.1.0",
			DefaultStepLibSource: "https://github.com/bitrise-io/bitrise-steplib.git",
		}

		configName := iOSConfigName(detector.HasPodFile, true)
		bitriseDataMap[configName] = bitriseData
	}

	// XcodeArchive
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
	}

	steps = append(steps, bitriseModels.StepListItemModel{
		stepXcodeArchiveIDComposite: stepmanModels.StepModel{
			Inputs: inputs,
		},
	})

	// DeployToBitriseIo
	steps = append(steps, bitriseModels.StepListItemModel{
		stepDeployToBitriseIoIDComposite: stepmanModels.StepModel{},
	})

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(steps)

	configName := iOSConfigName(detector.HasPodFile, false)
	bitriseDataMap[configName] = bitriseData

	return bitriseDataMap
}
