package scanners

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

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

	includeMonoTouchAPIPattern   = `Include="monotouch"`
	includeXamarinIosAPIPattern  = `Include="Xamarin.iOS"`
	includeMonoAndroidAPIPattern = `Include="Mono.Android"`

	monoTouchAPI   = "monotouch"
	xamarinIosAPI  = "Xamarin.iOS"
	monoAndroidAPI = "Mono.Android"

	includeXamarinUITestFrameworkPattern = `Include="Xamarin.UITest`
	includeNunitFrameworkPattern         = `Include="nunit.framework`
	includeNunitLiteFrameworkPattern     = `Include="MonoTouch.NUnitLite`

	xamarinUITestType = "Xamarin UITest"
	nunitTestType     = "NUnit test"
	nunitLiteTestType = "NUnitLite test"
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

	stepXamarinBuilderIDComposite = "xamarin-builder@1.3.0"

	xamarinIosLicenceKey    = "xamarin_ios_license"
	xamarinIosLicenceTitle  = "Xamarin.iOS License"
	xamarinIosLicenceEnvKey = "__XAMARIN_IOS_LICENSE_VALUE__"

	xamarinAndroidLicenceKey    = "xamarin_android_license"
	xamarinAndroidLicenceTitle  = "Xamarin.Android License"
	xamarinAndroidLicenceEnvKey = "__XAMARIN_ANDROID_LICENSE_VALUE__"

	stepXamarinUserManagementIDComposite    = "xamarin-user-management@1.0.2"
	stepNugetRestoreIDComposite             = "nuget-restore@0.9.1"
	stepXamarinComponentsRestoreIDComposite = "xamarin-components-restore@0.9.0"
)

//--------------------------------------------------
// Utility
//--------------------------------------------------

func filterSolutionFiles(fileList []string) []string {
	files := utility.FilterFilesWithExtensions(fileList, solutionExtension)

	componentsProjectExp := regexp.MustCompile(`.*Components/.+.sln`)

	relevantFiles := []string{}
	for _, file := range files {
		isComponentsSolution := false
		if componentsProjectExp.FindString(file) != "" {
			isComponentsSolution = true
		}

		if !isComponentsSolution {
			relevantFiles = append(relevantFiles, file)
		}
	}

	sort.Sort(utility.ByComponents(relevantFiles))
	return relevantFiles
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
	} else if utility.CaseInsensitiveContains(content, includeMonoTouchAPIPattern) {
		return monoTouchAPI, nil
	} else if utility.CaseInsensitiveContains(content, includeXamarinIosAPIPattern) {
		return xamarinIosAPI, nil
	}

	return "", nil
}

func getProjectTestType(projectFile string) (string, error) {
	content, err := fileutil.ReadStringFromFile(projectFile)
	if err != nil {
		return "", err
	}

	if utility.CaseInsensitiveContains(content, includeXamarinUITestFrameworkPattern) {
		return xamarinUITestType, nil
	} else if utility.CaseInsensitiveContains(content, includeNunitLiteFrameworkPattern) {
		return nunitLiteTestType, nil
	} else if utility.CaseInsensitiveContains(content, includeNunitFrameworkPattern) {
		return nunitTestType, nil
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

func xamarinDefaultConfigName() string {
	return "default-xamarin-config"
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
	logger.Info("Searching for solution files")

	solutionFiles := filterSolutionFiles(fileList)
	detector.SolutionFiles = solutionFiles

	logger.InfofDetails("%d solution file(s) detected", len(solutionFiles))

	if len(solutionFiles) == 0 {
		logger.InfofDetails("platform not detected")
		return false, nil
	}

	logger.InfofReceipt("platform detected")

	return true, nil
}

// Options ...
func (detector *Xamarin) Options() (models.OptionModel, error) {
	logger.InfoSection("Searching for Nuget packages & Xamarin Components")

	for _, file := range detector.FileList {
		// Search for nuget packages
		if detector.HasNugetPackages == false {
			baseName := filepath.Base(file)
			if baseName == "packages.config" {
				detector.HasNugetPackages = true
			}
		}

		// If adding a component:
		// /Components/[COMPONENT_NAME]/ dir added
		// ItemGroup/XamarinComponentReference added to the project
		// packages.config added to the project's folder
		if detector.HasXamarinComponents == false {
			componentsExp := regexp.MustCompile(".*Components/.+")
			if result := componentsExp.FindString(file); result != "" {
				detector.HasXamarinComponents = true
			}
		}

		if detector.HasNugetPackages && detector.HasXamarinComponents {
			break
		}
	}

	if detector.HasNugetPackages {
		logger.InfofReceipt("Nuget packages found")
	} else {
		logger.InfofDetails("NO Nuget packages found")
	}

	if detector.HasXamarinComponents {
		logger.InfofReceipt("Xamarin Components found")
	} else {
		logger.InfofDetails("NO Xamarin Components found")
	}

	// Check for solution configs
	validSolutionMap := map[string]map[string][]string{}
	for _, solutionFile := range detector.SolutionFiles {
		logger.InfofSection("Inspecting solution file: %s", solutionFile)

		configs, err := getSolutionConfigs(solutionFile)
		if err != nil {
			return models.OptionModel{}, err
		}

		if len(configs) > 0 {
			logger.InfofReceipt("found configs: %v", configs)

			validSolutionMap[solutionFile] = configs
		} else {
			log.Warnf("No config found for %s", solutionFile)
		}
	}

	if len(validSolutionMap) == 0 {
		return models.OptionModel{}, errors.New("No valid solution file found")
	}

	// Check for solution projects
	xamarinProjectOption := models.NewOptionModel(xamarinProjectTitle, xamarinProjectEnvKey)

	for solutionFile, configMap := range validSolutionMap {
		projects, err := getProjects(solutionFile)
		if err != nil {
			return models.OptionModel{}, err
		}

		// Inspect projects
		for _, project := range projects {
			logger.InfofSection("  Inspecting project file: %s", project)

			api, err := getProjectPlatformAPI(project)
			if err != nil {
				return models.OptionModel{}, err
			}

			if api == "" {
				testType, err := getProjectTestType(project)
				if err != nil {
					return models.OptionModel{}, err
				}

				if testType == "" {
					log.Warn("    No platform api or test framework found")
					continue
				}

				logger.InfofDetails("  %s test project", testType)
			} else {
				logger.InfofDetails("  %s project", api)
			}
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

// DefaultOptions ...
func (detector *Xamarin) DefaultOptions() models.OptionModel {
	xamarinProjectOption := models.NewOptionModel(xamarinProjectTitle, xamarinProjectEnvKey)

	xamarinConfigurationOption := models.NewOptionModel(xamarinConfigurationTitle, xamarinConfigurationEnvKey)

	xamarinPlatformOption := models.NewOptionModel(xamarinPlatformTitle, xamarinPlatformEnvKey)

	configOption := models.NewEmptyOptionModel()
	configOption.Config = xamarinDefaultConfigName()

	xamarinPlatformOption.ValueMap["_"] = configOption

	xamarinConfigurationOption.ValueMap["_"] = xamarinPlatformOption

	xamarinProjectOption.ValueMap["_"] = xamarinConfigurationOption

	return xamarinProjectOption
}

// Configs ...
func (detector *Xamarin) Configs() (map[string]string, error) {
	steps := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	steps = append(steps, bitriseModels.StepListItemModel{
		stepActivateSSHKeyIDComposite: stepmanModels.StepModel{
			RunIf: pointers.NewStringPtr(`{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}`),
		},
	})

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
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return map[string]string{}, err
	}

	configName := xamarinConfigName(detector.HasNugetPackages, detector.HasXamarinComponents)
	bitriseDataMap := map[string]string{
		configName: string(data),
	}

	return bitriseDataMap, nil
}

// DefaultConfigs ...
func (detector *Xamarin) DefaultConfigs() (map[string]string, error) {
	steps := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	steps = append(steps, bitriseModels.StepListItemModel{
		stepActivateSSHKeyIDComposite: stepmanModels.StepModel{
			RunIf: pointers.NewStringPtr(`{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}`),
		},
	})
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
	steps = append(steps, bitriseModels.StepListItemModel{
		stepNugetRestoreIDComposite: stepmanModels.StepModel{},
	})

	// XamarinComponentsRestore
	steps = append(steps, bitriseModels.StepListItemModel{
		stepXamarinComponentsRestoreIDComposite: stepmanModels.StepModel{},
	})

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
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return map[string]string{}, err
	}

	configName := xamarinDefaultConfigName()
	bitriseDataMap := map[string]string{
		configName: string(data),
	}

	return bitriseDataMap, nil
}
