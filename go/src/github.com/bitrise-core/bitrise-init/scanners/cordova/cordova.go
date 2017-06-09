package cordova

import (
	"fmt"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners/android"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

// ScannerName ...
const ScannerName = "cordova"

const (
	configName        = "cordova-config"
	defaultConfigName = "default-cordova-config"
)

// Step Inputs
const (
	workDirInputKey    = "workdir"
	workDirInputTitle  = "Directory of Cordova Config.xml"
	workDirInputEnvKey = "CORDOVA_WORK_DIR"
)

const (
	platformInputKey    = "platform"
	platformInputTitle  = "Platform to use in cordova-cli commands"
	platformInputEnvKey = "CORDOVA_PLATFORM"
)

const (
	targetInputKey = "target"
	targetEmulator = "emulator"
)

//------------------
// ScannerInterface
//------------------

// Scanner ...
type Scanner struct {
	cordovaConfigPth    string
	relCordovaConfigDir string
	searchDir           string
	hasKarmaJasmineTest bool
	hasJasmineTest      bool
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (scanner Scanner) Name() string {
	return ScannerName
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	fileList, err := utility.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}

	// Search for config.xml file
	log.Infoft("Searching for config.xml file")

	configXMLPth, err := utility.FilterRootConfigXMLFile(fileList)
	if err != nil {
		return false, fmt.Errorf("failed to search for config.xml file, error: %s", err)
	}

	log.Printft("config.xml: %s", configXMLPth)

	if configXMLPth == "" {
		log.Printft("platform not detected")
		return false, nil
	}

	widget, err := utility.ParseConfigXML(configXMLPth)
	if err != nil {
		log.Printft("can not parse config.xml as a Cordova widget, error: %s", err)
		log.Printft("platform not detected")
		return false, nil
	}

	// ensure it is a cordova widget
	if !strings.Contains(widget.XMLNSCDV, "cordova.apache.org") {
		log.Printft("config.xml propert: xmlns:cdv does not contain cordova.apache.org")
		log.Printft("platform not detected")
		return false, nil
	}

	// ensure it is not an ionic project
	projectBaseDir := filepath.Dir(configXMLPth)

	if exist, err := pathutil.IsPathExists(filepath.Join(projectBaseDir, "ionic.project")); err != nil {
		return false, fmt.Errorf("failed to check if project is an ionic project, error: %s", err)
	} else if exist {
		log.Printft("ionic.project file found seems to be an ionic project")
		return false, nil
	}

	if exist, err := pathutil.IsPathExists(filepath.Join(projectBaseDir, "ionic.config.json")); err != nil {
		return false, fmt.Errorf("failed to check if project is an ionic project, error: %s", err)
	} else if exist {
		log.Printft("ionic.config.json file found seems to be an ionic project")
		return false, nil
	}

	log.Doneft("Platform detected")

	scanner.cordovaConfigPth = configXMLPth
	scanner.searchDir = searchDir

	return true, nil
}

// ExcludedScannerNames ...
func (scanner *Scanner) ExcludedScannerNames() []string {
	return []string{
		string(utility.XcodeProjectTypeIOS),
		string(utility.XcodeProjectTypeMacOS),
		android.ScannerName,
	}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	warnings := models.Warnings{}
	projectRootDir := filepath.Dir(scanner.cordovaConfigPth)

	packagesJSONPth := filepath.Join(projectRootDir, "package.json")
	packages, err := utility.ParsePackagesJSON(packagesJSONPth)
	if err != nil {
		return models.OptionModel{}, warnings, err
	}

	// Search for karma/jasmine tests
	log.Printft("Searching for karma/jasmine test")

	karmaTestDetected := false

	karmaJasmineDependencyFound := false
	for dependency := range packages.Dependencies {
		if strings.Contains(dependency, "karma-jasmine") {
			karmaJasmineDependencyFound = true
		}
	}
	if !karmaJasmineDependencyFound {
		for dependency := range packages.DevDependencies {
			if strings.Contains(dependency, "karma-jasmine") {
				karmaJasmineDependencyFound = true
			}
		}
	}
	log.Printft("karma-jasmine dependency found: %v", karmaJasmineDependencyFound)

	if karmaJasmineDependencyFound {
		karmaConfigJSONPth := filepath.Join(projectRootDir, "karma.conf.js")
		if exist, err := pathutil.IsPathExists(karmaConfigJSONPth); err != nil {
			return models.OptionModel{}, warnings, err
		} else if exist {
			karmaTestDetected = true
		}
	}
	log.Printft("karma.conf.js found: %v", karmaTestDetected)

	scanner.hasKarmaJasmineTest = karmaTestDetected
	// ---

	// Search for jasmine tests
	jasminTestDetected := false

	if !karmaTestDetected {
		log.Printft("Searching for jasmine test")

		jasmineDependencyFound := false
		for dependency := range packages.Dependencies {
			if strings.Contains(dependency, "jasmine") {
				jasmineDependencyFound = true
				break
			}
		}
		if !jasmineDependencyFound {
			for dependency := range packages.DevDependencies {
				if strings.Contains(dependency, "jasmine") {
					jasmineDependencyFound = true
					break
				}
			}
		}
		log.Printft("jasmine dependency found: %v", jasmineDependencyFound)

		if jasmineDependencyFound {
			jasmineConfigJSONPth := filepath.Join(projectRootDir, "spec", "support", "jasmine.json")
			if exist, err := pathutil.IsPathExists(jasmineConfigJSONPth); err != nil {
				return models.OptionModel{}, warnings, err
			} else if exist {
				jasminTestDetected = true
			}
		}

		log.Printft("jasmine.json found: %v", jasminTestDetected)

		scanner.hasJasmineTest = jasminTestDetected
	}
	// ---

	// Get relative config.xml dir
	cordovaConfigDir := filepath.Dir(scanner.cordovaConfigPth)
	relCordovaConfigDir, err := utility.RelPath(scanner.searchDir, cordovaConfigDir)
	if err != nil {
		return models.OptionModel{}, warnings, fmt.Errorf("Failed to get relative config.xml dir path, error: %s", err)
	}
	if relCordovaConfigDir == "." {
		// config.xml placed in the search dir, no need to change-dir in the workflows
		relCordovaConfigDir = ""
	}
	scanner.relCordovaConfigDir = relCordovaConfigDir
	// ---

	// Options
	var rootOption *models.OptionModel

	platforms := []string{"ios", "android", "ios,android"}

	if relCordovaConfigDir != "" {
		rootOption = models.NewOption(workDirInputTitle, workDirInputEnvKey)

		projectTypeOption := models.NewOption(platformInputTitle, platformInputEnvKey)
		rootOption.AddOption(relCordovaConfigDir, projectTypeOption)

		for _, platform := range platforms {
			configOption := models.NewConfigOption(configName)
			projectTypeOption.AddConfig(platform, configOption)
		}
	} else {
		rootOption = models.NewOption(platformInputTitle, platformInputEnvKey)

		for _, platform := range platforms {
			configOption := models.NewConfigOption(configName)
			rootOption.AddConfig(platform, configOption)
		}
	}
	// ---

	return *rootOption, warnings, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionModel {
	workDirOption := models.NewOption(workDirInputTitle, workDirInputEnvKey)

	projectTypeOption := models.NewOption(platformInputTitle, platformInputEnvKey)
	workDirOption.AddOption("_", projectTypeOption)

	platforms := []string{
		"ios",
		"android",
		"ios,android",
	}
	for _, platform := range platforms {
		configOption := models.NewConfigOption(defaultConfigName)
		projectTypeOption.AddConfig(platform, configOption)
	}

	return *workDirOption
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder(false)

	workdirEnvList := []envmanModels.EnvironmentItemModel{}
	if scanner.relCordovaConfigDir != "" {
		workdirEnvList = append(workdirEnvList, envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey})
	}

	configBuilder.AppendDependencyStepList(steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))

	if scanner.hasJasmineTest || scanner.hasKarmaJasmineTest {
		// CI
		if scanner.hasKarmaJasmineTest {
			configBuilder.AppendMainStepList(steps.KarmaJasmineTestRunnerStepListItem(workdirEnvList...))
		} else if scanner.hasJasmineTest {
			configBuilder.AppendMainStepList(steps.JasmineTestRunnerStepListItem(workdirEnvList...))
		}

		// CD
		configBuilder.AddDefaultWorkflowBuilder(models.DeployWorkflowID, false)

		configBuilder.AppendDependencyStepListTo(models.DeployWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))

		if scanner.hasKarmaJasmineTest {
			configBuilder.AppendMainStepListTo(models.DeployWorkflowID, steps.KarmaJasmineTestRunnerStepListItem(workdirEnvList...))
		} else if scanner.hasJasmineTest {
			configBuilder.AppendMainStepListTo(models.DeployWorkflowID, steps.JasmineTestRunnerStepListItem(workdirEnvList...))
		}

		configBuilder.AppendMainStepListTo(models.DeployWorkflowID, steps.GenerateCordovaBuildConfigStepListItem())

		cordovaArchiveEnvs := []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{platformInputKey: "$" + platformInputEnvKey},
			envmanModels.EnvironmentItemModel{targetInputKey: targetEmulator},
		}
		if scanner.relCordovaConfigDir != "" {
			cordovaArchiveEnvs = append(cordovaArchiveEnvs, envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey})
		}
		configBuilder.AppendMainStepListTo(models.DeployWorkflowID, steps.CordovaArchiveStepListItem(cordovaArchiveEnvs...))

		config, err := configBuilder.Generate(ScannerName)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		return models.BitriseConfigMap{
			configName: string(data),
		}, nil
	}

	configBuilder.AppendMainStepList(steps.GenerateCordovaBuildConfigStepListItem())

	cordovaArchiveEnvs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{platformInputKey: "$" + platformInputEnvKey},
		envmanModels.EnvironmentItemModel{targetInputKey: targetEmulator},
	}
	if scanner.relCordovaConfigDir != "" {
		cordovaArchiveEnvs = append(cordovaArchiveEnvs, envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey})
	}
	configBuilder.AppendMainStepList(steps.CordovaArchiveStepListItem(cordovaArchiveEnvs...))

	config, err := configBuilder.Generate(ScannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		configName: string(data),
	}, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder(false)

	configBuilder.AppendDependencyStepList(steps.NpmStepListItem(
		envmanModels.EnvironmentItemModel{"command": "install"},
		envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey}))

	configBuilder.AppendMainStepList(steps.GenerateCordovaBuildConfigStepListItem())

	configBuilder.AppendMainStepList(steps.CordovaArchiveStepListItem(
		envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey},
		envmanModels.EnvironmentItemModel{platformInputKey: "$" + platformInputEnvKey},
		envmanModels.EnvironmentItemModel{targetInputKey: targetEmulator}))

	config, err := configBuilder.Generate(ScannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		defaultConfigName: string(data),
	}, nil
}
