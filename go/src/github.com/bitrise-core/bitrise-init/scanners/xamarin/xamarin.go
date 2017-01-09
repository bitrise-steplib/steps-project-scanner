package xamarin

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
)

const (
	scannerName = "xamarin"
)

const (
	solutionExtension          = ".sln"
	solutionConfigurationStart = "GlobalSection(SolutionConfigurationPlatforms) = preSolution"
	solutionConfigurationEnd   = "EndGlobalSection"
	projectTypeGUIDExp         = `(?i)<ProjectTypeGuids>(?P<project_type_guids>.*)<\/ProjectTypeGuids>`

	includeMonoTouchAPIPattern   = `Include="monotouch"`
	includeXamarinIosAPIPattern  = `Include="Xamarin.iOS"`
	includeMonoAndroidAPIPattern = `Include="Mono.Android"`

	monoTouchAPI   = "monotouch"
	xamarinIosAPI  = "Xamarin.iOS"
	monoAndroidAPI = "Mono.Android"

	includeXamarinUITestFrameworkPattern = `Include="Xamarin.UITest`
	includeNunitFrameworkPattern         = `Include="nunit.framework`
	includeNunitLiteFrameworkExp         = `(?i)Include=".*.NUnitLite"`

	xamarinUITestType = "Xamarin UITest"
	nunitTestType     = "NUnit test"
	nunitLiteTestType = "NUnitLite test"
)

const (
	xamarinSolutionKey    = "xamarin_solution"
	xamarinSolutionTitle  = "Path to the Xamarin Solution file"
	xamarinSolutionEnvKey = "BITRISE_PROJECT_PATH"

	xamarinConfigurationKey    = "xamarin_configuration"
	xamarinConfigurationTitle  = "Xamarin solution configuration"
	xamarinConfigurationEnvKey = "BITRISE_XAMARIN_CONFIGURATION"

	xamarinPlatformKey    = "xamarin_platform"
	xamarinPlatformTitle  = "Xamarin solution platform"
	xamarinPlatformEnvKey = "BITRISE_XAMARIN_PLATFORM"

	xamarinIosLicenceKey   = "xamarin_ios_license"
	xamarinIosLicenceTitle = "Xamarin.iOS License"

	xamarinAndroidLicenceKey   = "xamarin_android_license"
	xamarinAndroidLicenceTitle = "Xamarin.Android License"

	xamarinMacLicenseKey   = "xamarin_mac_license"
	xamarinMacLicenseTitle = "Xamarin.Mac License"
)

var (
	projectTypeGUIDMap = map[string][]string{
		"Xamarin.iOS": []string{
			"E613F3A2-FE9C-494F-B74E-F63BCB86FEA6",
			"6BC8ED88-2882-458C-8E55-DFD12B67127B",
			"F5B4F3BC-B597-4E2B-B552-EF5D8A32436F",
			"FEACFBD2-3405-455C-9665-78FE426C6842",
			"8FFB629D-F513-41CE-95D2-7ECE97B6EEEC",
			"EE2C853D-36AF-4FDB-B1AD-8E90477E2198",
		},
		"Xamarin.Android": []string{
			"EFBA0AD7-5A72-4C68-AF49-83D382785DCF",
			"10368E6C-D01B-4462-8E8B-01FC667A7035",
		},
		"MonoMac": []string{
			"1C533B1C-72DD-4CB1-9F6B-BF11D93BCFBE",
			"948B3504-5B70-4649-8FE4-BDE1FB46EC69",
		},
		"Xamarin.Mac": []string{
			"42C0BBD9-55CE-4FC1-8D90-A7348ABAFB23",
			"A3F8F2AB-B479-4A4A-A458-A89E7DC349F1",
		},
		"Xamarin.tvOS": []string{
			"06FA79CB-D6CD-4721-BB4B-1BD202089C55",
		},
	}
)

//--------------------------------------------------
// Utility
//--------------------------------------------------

func projectType(guids []string) string {
	for _, GUID := range guids {
		for projectType, projectTypeGUIDs := range projectTypeGUIDMap {
			for _, aGUID := range projectTypeGUIDs {
				if aGUID == GUID {
					return projectType
				}
			}
		}
	}

	return ""
}

func filterSolutionFiles(fileList []string) ([]string, error) {
	allowSolutionExtensionFilter := utility.ExtensionFilter(solutionExtension, true)
	forbidComponentsSolutionFilter := utility.RegexpFilter(`.*Components/.+.sln`, false)
	files, err := utility.FilterPaths(fileList,
		allowSolutionExtensionFilter,
		forbidComponentsSolutionFilter)
	if err != nil {
		return []string{}, err
	}

	return files, nil
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

func getProjectGUIDs(projectFile string) ([]string, error) {
	projectTypeGUIDSExp := regexp.MustCompile(projectTypeGUIDExp)
	content, err := fileutil.ReadStringFromFile(projectFile)
	if err != nil {
		return []string{}, err
	}

	lines := strings.Split(content, "\n")
	guidsStr := ""
	for _, line := range lines {
		match := projectTypeGUIDSExp.FindStringSubmatch(line)
		if len(match) == 2 {
			guidsStr = match[1]
		}
	}

	guids := []string{}
	guidsSplit := strings.Split(guidsStr, ";")
	for _, guidStr := range guidsSplit {
		guid := guidStr
		if strings.HasPrefix(guid, "{") {
			guid = strings.TrimPrefix(guid, "{")
		}

		if strings.HasSuffix(guid, "}") {
			guid = strings.TrimSuffix(guid, "}")
		}

		guids = append(guids, guid)
	}

	return guids, nil
}

func getProjectTestType(projectFile string) (string, error) {
	content, err := fileutil.ReadStringFromFile(projectFile)
	if err != nil {
		return "", err
	}

	return projectTestType(content), nil
}

func projectTestType(projectFileContent string) string {
	if utility.CaseInsensitiveContains(projectFileContent, includeXamarinUITestFrameworkPattern) {
		return xamarinUITestType
	} else if utility.CaseInsensitiveContains(projectFileContent, includeNunitFrameworkPattern) {
		return nunitTestType
	} else {
		lines := strings.Split(projectFileContent, string(filepath.Separator))
		nunitLiteExp := regexp.MustCompile(includeNunitLiteFrameworkExp)

		for _, line := range lines {
			if nunitLiteExp.FindString(line) != "" {
				return nunitLiteTestType
			}
		}
	}

	return ""
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

func configName(hasNugetPackages, hasXamarinComponents bool) string {
	name := "xamarin-"
	if hasNugetPackages {
		name = name + "nuget-"
	}
	if hasXamarinComponents {
		name = name + "components-"
	}
	return name + "config"
}

func defaultConfigName() string {
	return "default-xamarin-config"
}

//--------------------------------------------------
// Scanner
//--------------------------------------------------

// Scanner ...
type Scanner struct {
	SearchDir     string
	FileList      []string
	SolutionFiles []string

	HasNugetPackages     bool
	HasXamarinComponents bool

	HasIosProject     bool
	HasAndroidProject bool
	HasMacProject     bool
	HasTVOSProject    bool
}

// Name ...
func (scanner Scanner) Name() string {
	return scannerName
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	fileList, err := utility.ListPathInDirSortedByComponents(searchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}
	scanner.FileList = fileList

	// Search for solution file
	log.Infoft("Searching for solution files")

	solutionFiles, err := filterSolutionFiles(fileList)
	if err != nil {
		return false, fmt.Errorf("failed to search for solution files, error: %s", err)
	}

	scanner.SolutionFiles = solutionFiles

	log.Printft("%d solution files detected", len(solutionFiles))
	for _, file := range solutionFiles {
		log.Printft("- %s", file)
	}

	if len(solutionFiles) == 0 {
		log.Printft("platform not detected")
		return false, nil
	}

	log.Doneft("Platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	log.Infoft("Searching for NuGet packages & Xamarin Components")

	warnings := models.Warnings{}

	for _, file := range scanner.FileList {
		// Search for nuget packages
		if scanner.HasNugetPackages == false {
			baseName := filepath.Base(file)
			if baseName == "packages.config" {
				scanner.HasNugetPackages = true
			}
		}

		// If adding a component:
		// /Components/[COMPONENT_NAME]/ dir added
		// ItemGroup/XamarinComponentReference added to the project
		// packages.config added to the project's folder
		if scanner.HasXamarinComponents == false {
			componentsExp := regexp.MustCompile(".*Components/.+")
			if result := componentsExp.FindString(file); result != "" {
				scanner.HasXamarinComponents = true
			}
		}

		if scanner.HasNugetPackages && scanner.HasXamarinComponents {
			break
		}
	}

	if scanner.HasNugetPackages {
		log.Printft("Nuget packages found")
	} else {
		log.Printft("NO Nuget packages found")
	}

	if scanner.HasXamarinComponents {
		log.Printft("Xamarin Components found")
	} else {
		log.Printft("NO Xamarin Components found")
	}

	// Check for solution configs
	validSolutionMap := map[string]map[string][]string{}
	for _, solutionFile := range scanner.SolutionFiles {
		log.Infoft("Inspecting solution file: %s", solutionFile)

		configs, err := getSolutionConfigs(solutionFile)
		if err != nil {
			log.Warnft("Failed to get solution configs, error: %s", err)
			warnings = append(warnings, fmt.Sprintf("Failed to get solution (%s) configs, error: %s", solutionFile, err))
			continue
		}

		if len(configs) > 0 {
			log.Printft("%d configurations found", len(configs))
			for config, platforms := range configs {
				log.Printft("- %s with platforms: %v", config, platforms)
			}

			validSolutionMap[solutionFile] = configs
		} else {
			log.Warnft("No config found for %s", solutionFile)
			warnings = append(warnings, fmt.Sprintf("No configs found for solution: %s", solutionFile))
		}
	}

	if len(validSolutionMap) == 0 {
		log.Errorft("No valid solution file found")
		return models.OptionModel{}, warnings, errors.New("No valid solution file found")
	}

	// Check for solution projects
	xamarinSolutionOption := models.NewOptionModel(xamarinSolutionTitle, xamarinSolutionEnvKey)

	for solutionFile, configMap := range validSolutionMap {
		xamarinConfigurationOption := models.NewOptionModel(xamarinConfigurationTitle, xamarinConfigurationEnvKey)

		for config, platforms := range configMap {
			xamarinPlatformOption := models.NewOptionModel(xamarinPlatformTitle, xamarinPlatformEnvKey)
			for _, platform := range platforms {
				configOption := models.NewEmptyOptionModel()
				configOption.Config = configName(scanner.HasNugetPackages, scanner.HasXamarinComponents)

				xamarinPlatformOption.ValueMap[platform] = configOption
			}

			xamarinConfigurationOption.ValueMap[config] = xamarinPlatformOption
		}

		xamarinSolutionOption.ValueMap[solutionFile] = xamarinConfigurationOption
	}

	return xamarinSolutionOption, warnings, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionModel {
	configOption := models.NewEmptyOptionModel()
	configOption.Config = defaultConfigName()

	xamarinSolutionOption := models.NewOptionModel(xamarinSolutionTitle, xamarinSolutionEnvKey)
	xamarinConfigurationOption := models.NewOptionModel(xamarinConfigurationTitle, xamarinConfigurationEnvKey)
	xamarinPlatformOption := models.NewOptionModel(xamarinPlatformTitle, xamarinPlatformEnvKey)

	xamarinPlatformOption.ValueMap["_"] = configOption
	xamarinConfigurationOption.ValueMap["_"] = xamarinPlatformOption
	xamarinSolutionOption.ValueMap["_"] = xamarinConfigurationOption

	return xamarinSolutionOption
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// Script
	stepList = append(stepList, steps.ScriptSteplistItem(steps.ScriptDefaultTitle))

	// CertificateAndProfileInstaller
	stepList = append(stepList, steps.CertificateAndProfileInstallerStepListItem())

	// XamarinUserManagement
	inputs := []envmanModels.EnvironmentItemModel{}
	if scanner.HasIosProject {
		inputs = append(inputs, envmanModels.EnvironmentItemModel{xamarinIosLicenceKey: "yes"})
	}
	if scanner.HasAndroidProject {
		inputs = append(inputs, envmanModels.EnvironmentItemModel{xamarinAndroidLicenceKey: "yes"})
	}
	if scanner.HasMacProject {
		inputs = append(inputs, envmanModels.EnvironmentItemModel{xamarinMacLicenseKey: "yes"})
	}

	stepList = append(stepList, steps.XamarinUserManagementStepListItem(inputs))

	// NugetRestore
	if scanner.HasNugetPackages {
		stepList = append(stepList, steps.NugetRestoreStepListItem())
	}

	// XamarinComponentsRestore
	if scanner.HasXamarinComponents {
		stepList = append(stepList, steps.XamarinComponentsRestoreStepListItem())
	}

	// XamarinArchive
	inputs = []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{xamarinSolutionKey: "$" + xamarinSolutionEnvKey},
		envmanModels.EnvironmentItemModel{xamarinConfigurationKey: "$" + xamarinConfigurationEnvKey},
		envmanModels.EnvironmentItemModel{xamarinPlatformKey: "$" + xamarinPlatformEnvKey},
	}

	stepList = append(stepList, steps.XamarinArchiveStepListItem(inputs))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	bitriseData := models.BitriseDataWithCIWorkflow([]envmanModels.EnvironmentItemModel{}, stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := configName(scanner.HasNugetPackages, scanner.HasXamarinComponents)
	bitriseDataMap := models.BitriseConfigMap{
		configName: string(data),
	}

	return bitriseDataMap, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	stepList := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	stepList = append(stepList, steps.ActivateSSHKeyStepListItem())

	// GitClone
	stepList = append(stepList, steps.GitCloneStepListItem())

	// Script
	stepList = append(stepList, steps.ScriptSteplistItem(steps.ScriptDefaultTitle))

	// CertificateAndProfileInstaller
	stepList = append(stepList, steps.CertificateAndProfileInstallerStepListItem())

	// XamarinUserManagement
	inputs := []envmanModels.EnvironmentItemModel{}
	stepList = append(stepList, steps.XamarinUserManagementStepListItem(inputs))

	// NugetRestore
	stepList = append(stepList, steps.NugetRestoreStepListItem())

	// XamarinComponentsRestore
	stepList = append(stepList, steps.XamarinComponentsRestoreStepListItem())

	// XamarinArchive
	inputs = []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{xamarinSolutionKey: "$" + xamarinSolutionEnvKey},
		envmanModels.EnvironmentItemModel{xamarinConfigurationKey: "$" + xamarinConfigurationEnvKey},
		envmanModels.EnvironmentItemModel{xamarinPlatformKey: "$" + xamarinPlatformEnvKey},
	}

	stepList = append(stepList, steps.XamarinArchiveStepListItem(inputs))

	// DeployToBitriseIo
	stepList = append(stepList, steps.DeployToBitriseIoStepListItem())

	bitriseData := models.BitriseDataWithCIWorkflow([]envmanModels.EnvironmentItemModel{}, stepList)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := defaultConfigName()
	bitriseDataMap := models.BitriseConfigMap{
		configName: string(data),
	}

	return bitriseDataMap, nil
}
