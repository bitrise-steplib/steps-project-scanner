package scanners

import (
	"fmt"
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
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pointers"
	stepmanModels "github.com/bitrise-io/stepman/models"
)

const (
	xamarinDetectorName = "xamarin"
)

const (
	solutionExtension          = ".sln"
	solutionConfigurationStart = "GlobalSection(SolutionConfigurationPlatforms) = preSolution"
	solutionConfigurationEnd   = "EndGlobalSection"

	includemonoTouchAPIPattern   = `Include="monotouch"`
	includeXamarinIosAPIPattern  = `Include="Xamarin.iOS"`
	includeMonoAndroidAPIPattern = `Include="Mono.Android"`

	monoTouchAPI   = "monotouch"
	xamarinIosAPI  = "Xamarin.iOS"
	monoAndroidAPI = "Mono.Android"
)

const (
	xamarinProjectKey    = "xamarin_project"
	xamarinProjectTitle  = "Path to Xamarin Solution"
	xamarinProjectEnvKey = "BITRISE_PROJECT_PATH"

	xamarinConfigurationKey    = "xamarin_configuration"
	xamarinConfigurationTitle  = "Xamarin project configuration"
	xamarinConfigurationEnvKey = "BITRISE_XAMARIN_CONFIGURATION"

	xamarinPlatformKey    = "xamarin_platform"
	xamarinPlatformTitle  = "Xamarin platform"
	xamarinPlatformEnvKey = "BITRISE_XAMARIN_PLATFORM"

	stepXamarinBuilderIDComposite = "xamarin-builder@1.1.3"

	xamarinIosLicenceKey    = "xamarin_ios_license"
	xamarinIosLicenceTitle  = "Xamarin.iOS License"
	xamarinIosLicenceEnvKey = "__XAMARIN_IOS_LICENSE_VALUE__"

	xamarinAndroidLicenceKey    = "xamarin_android_license"
	xamarinAndroidLicenceTitle  = "Xamarin.Android License"
	xamarinAndroidLicenceEnvKey = "__XAMARIN_ANDROID_LICENSE_VALUE__"

	stepXamarinUserManagementIDComposite = "xamarin-user-management@1.0.1"

	stepNugetRestoreIDComposite             = "nuget-restore@0.9.0"
	stepXamarinComponentsRestoreIDComposite = "xamarin-components-restore@0.9.0"
)

//--------------------------------------------------
// Utility
//--------------------------------------------------

func filterSolutionFiles(fileList []string) []string {
	files := utility.FilterFilesWithExtensions(fileList, solutionExtension)
	sort.Sort(utility.ByComponents(files))
	return files
}

func getSolutionConfigs(solutionFile string) (map[string][]string, error) {
	content, err := fileutil.ReadStringFromFile(solutionFile)
	if err != nil {
		return map[string][]string{}, err
	}

	configMap := map[string][]string{}
	isNextLineScheme := false

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, solutionConfigurationStart) {
			isNextLineScheme = true
			continue
		}

		if strings.Contains(line, solutionConfigurationEnd) {
			isNextLineScheme = false
			continue
		}

		if isNextLineScheme {
			split := strings.Split(line, "=")
			if len(split) == 2 {
				configCompositStr := strings.TrimSpace(split[1])
				configSplit := strings.Split(configCompositStr, "|")

				if len(configSplit) == 2 {
					config := configSplit[0]
					platform := configSplit[1]

					platforms := configMap[config]
					platforms = append(platforms, platform)

					configMap[config] = platforms
				}
			} else {
				return map[string][]string{}, fmt.Errorf("failed to parse config line (%s)", line)
			}
		}
	}

	return configMap, nil
}

func getProjectPlatformAPI(projectFile string) (string, error) {
	content, err := fileutil.ReadStringFromFile(projectFile)
	if err != nil {
		return "", err
	}

	if utility.CaseInsensitiveContains(content, includeMonoAndroidAPIPattern) {
		return monoAndroidAPI, nil
	} else if utility.CaseInsensitiveContains(content, includemonoTouchAPIPattern) {
		return monoTouchAPI, nil
	} else if utility.CaseInsensitiveContains(content, includeXamarinIosAPIPattern) {
		return xamarinIosAPI, nil
	}

	return "", nil
}

func getProjects(solutionFile string) ([]string, error) {
	content, err := fileutil.ReadStringFromFile(solutionFile)
	if err != nil {
		return []string{}, err
	}

	projectDir := filepath.Dir(solutionFile)
	projectExp := regexp.MustCompile(`Project\(\"[^\"]*\"\)\s*=\s*\"[^\"]*\",\s*\"([^\"]*.csproj)\"`)

	projects := []string{}
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		matches := projectExp.FindStringSubmatch(line)
		if len(matches) > 1 {
			project := strings.Replace(matches[1], "\\", "/", -1)
			projectPath := path.Join(projectDir, project)
			projects = append(projects, projectPath)
		}
	}

	return projects, nil
}

func xamarinConfigName(hasNugetPackages, hasXamarinComponents bool) string {
	name := "xamarin-"
	if hasNugetPackages {
		name = name + "nuget-"
	}
	if hasXamarinComponents {
		name = name + "components-"
	}
	return name + "config"
}

//--------------------------------------------------
// Detector
//--------------------------------------------------

// Xamarin ...
type Xamarin struct {
	SearchDir     string
	FileList      []string
	SolutionFiles []string

	HasNugetPackages     bool
	HasXamarinComponents bool
}

// Name ...
func (detector Xamarin) Name() string {
	return xamarinDetectorName
}

// Configure ...
func (detector *Xamarin) Configure(searchDir string) {
	detector.SearchDir = searchDir
}

// DetectPlatform ...
func (detector *Xamarin) DetectPlatform() (bool, error) {
	fileList, err := utility.FileList(detector.SearchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", detector.SearchDir, err)
	}
	detector.FileList = fileList

	// Search for solution file
	solutionFiles := filterSolutionFiles(fileList)
	detector.SolutionFiles = solutionFiles

	if len(solutionFiles) == 0 {
		return false, nil
	}

	return true, nil
}

// Analyze ...
func (detector *Xamarin) Analyze() (models.OptionModel, error) {
	for _, file := range detector.FileList {
		baseName := filepath.Base(file)
		if baseName == "packages.config" {
			detector.HasNugetPackages = true
		}

		// If adding a component:
		// /Components/[COMPONENT_NAME]/ dir added
		// ItemGroup/XamarinComponentReference added to the project
		// packages.config added to the project's folder
		componentsExp := regexp.MustCompile(".+/Components/.+")
		if result := componentsExp.FindString(file); result != "" {
			detector.HasXamarinComponents = true
		}
	}

	// Check for solution configs
	validSolutionMap := map[string]map[string][]string{}
	for _, solutionFile := range detector.SolutionFiles {
		configs, err := getSolutionConfigs(solutionFile)
		if err != nil {
			return models.OptionModel{}, err
		}

		if len(configs) > 0 {
			validSolutionMap[solutionFile] = configs
		} else {
			log.Warnf("No config found for %s", solutionFile)
		}
	}

	// Check for solution projects
	xamarinProjectOption := models.NewOptionModel(xamarinProjectTitle, xamarinProjectEnvKey)

	for solutionFile, configMap := range validSolutionMap {
		projects, err := getProjects(solutionFile)
		if err != nil {
			return models.OptionModel{}, err
		}

		// Inspect projects
		apis := []string{}
		for _, project := range projects {
			log.Infof("Inspecting project file: %s", project)

			api, err := getProjectPlatformAPI(project)
			if err != nil {
				return models.OptionModel{}, err
			}

			if api == "" {
				continue
			}

			apis = append(apis, api)
		}

		xamarinConfigurationOption := models.NewOptionModel(xamarinConfigurationTitle, xamarinConfigurationEnvKey)

		for config, platforms := range configMap {
			xamarinPlatformOption := models.NewOptionModel(xamarinPlatformTitle, xamarinPlatformEnvKey)
			for _, platform := range platforms {
				configOption := models.NewEmptyOptionModel()
				configOption.Config = xamarinConfigName(detector.HasNugetPackages, detector.HasXamarinComponents)

				xamarinPlatformOption.ValueMap[platform] = configOption
			}

			xamarinConfigurationOption.ValueMap[config] = xamarinPlatformOption
		}

		xamarinProjectOption.ValueMap[solutionFile] = xamarinConfigurationOption
	}

	return xamarinProjectOption, nil
}

// Configs ...
func (detector *Xamarin) Configs(isPrivate bool) map[string]bitriseModels.BitriseDataModel {
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

	// XamarinUserManagement
	inputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{xamarinIosLicenceKey: "$" + xamarinIosLicenceEnvKey},
		envmanModels.EnvironmentItemModel{xamarinAndroidLicenceKey: "$" + xamarinAndroidLicenceEnvKey},
	}

	steps = append(steps, bitriseModels.StepListItemModel{
		stepXamarinUserManagementIDComposite: stepmanModels.StepModel{
			Inputs: inputs,
			RunIf:  pointers.NewStringPtr(".IsCI"),
		},
	})

	// NugetRestore
	if detector.HasNugetPackages {
		steps = append(steps, bitriseModels.StepListItemModel{
			stepNugetRestoreIDComposite: stepmanModels.StepModel{},
		})
	}

	// XamarinComponentsRestore
	if detector.HasXamarinComponents {
		steps = append(steps, bitriseModels.StepListItemModel{
			stepXamarinComponentsRestoreIDComposite: stepmanModels.StepModel{},
		})
	}

	// XamarinBuilder
	inputs = []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{xamarinProjectKey: "$" + xamarinProjectEnvKey},
		envmanModels.EnvironmentItemModel{xamarinConfigurationKey: "$" + xamarinConfigurationEnvKey},
		envmanModels.EnvironmentItemModel{xamarinPlatformKey: "$" + xamarinPlatformEnvKey},
	}

	steps = append(steps, bitriseModels.StepListItemModel{
		stepXamarinBuilderIDComposite: stepmanModels.StepModel{
			Inputs: inputs,
		},
	})

	// DeployToBitriseIo
	steps = append(steps, bitriseModels.StepListItemModel{
		stepDeployToBitriseIoIDComposite: stepmanModels.StepModel{},
	})

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(steps)

	configName := xamarinConfigName(detector.HasNugetPackages, detector.HasXamarinComponents)
	bitriseDataMap := map[string]bitriseModels.BitriseDataModel{
		configName: bitriseData,
	}

	return bitriseDataMap
}
