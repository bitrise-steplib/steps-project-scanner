package ionic

import (
	"fmt"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/cordova"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/steps"
	"github.com/bitrise-io/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

const scannerName = "ionic"

const (
	configName        = "ionic-config"
	defaultConfigName = "default-ionic-config"
)

// Step Inputs
const (
	workDirInputKey     = "workdir"
	workDirInputTitle   = "Directory of the Ionic config.xml file"
	workDirInputEnvKey  = "IONIC_WORK_DIR"
	workDirInputSummary = "The working directory of your Ionic project is where you store your config.xml file. This location is stored as an Environment Variable. In your Workflows, you can specify paths relative to this path. You can change this at any time."
)

const (
	platformInputKey     = "platform"
	platformInputTitle   = "The platform to use in ionic-cli commands"
	platformInputEnvKey  = "IONIC_PLATFORM"
	platformInputSummary = "The target platform for your builds, stored as an Environment Variable. Your options are iOS, Android, or both. You can change this in your Env Vars at any time."
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
	ionicConfigPath     string
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
func (Scanner) Name() string {
	return scannerName
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	fileList, err := pathutil.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}

	// Ensure it is an ionic project
	ionicConfigPath, err := FilterRootFile(fileList, "ionic.config.json")
	if err != nil {
		return false, fmt.Errorf("failed to check if project is an ionic project, error: %s", err)
	}

	// Check the existence of the old ionic.project file
	if ionicConfigPath == "" {
		ionicConfigPath, err = FilterRootFile(fileList, "ionic.project")
		if err != nil {
			return false, fmt.Errorf("failed to check if project is an ionic project, error: %s", err)
		}
	}

	if ionicConfigPath == "" {
		log.Printf("No ionic.project file nor ionic.config.json found.")
		return false, nil
	}

	log.TSuccessf("Platform detected")

	scanner.ionicConfigPath = ionicConfigPath
	scanner.searchDir = searchDir

	return true, nil
}

// ExcludedScannerNames ...
func (Scanner) ExcludedScannerNames() []string {
	return []string{
		string(ios.XcodeProjectTypeIOS),
		string(ios.XcodeProjectTypeMacOS),
		cordova.ScannerName,
		android.ScannerName,
	}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	warnings := models.Warnings{}

	projectRootDir := filepath.Dir(scanner.ionicConfigPath)

	packagesJSONPth := filepath.Join(projectRootDir, "package.json")
	packages, err := utility.ParsePackagesJSON(packagesJSONPth)
	if err != nil {
		return models.OptionNode{}, warnings, nil, err
	}

	// Search for karma/jasmine tests
	log.TPrintf("Searching for karma/jasmine test")

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
	log.TPrintf("karma-jasmine dependency found: %v", karmaJasmineDependencyFound)

	if karmaJasmineDependencyFound {
		karmaConfigJSONPth := filepath.Join(projectRootDir, "karma.conf.js")
		if exist, err := pathutil.IsPathExists(karmaConfigJSONPth); err != nil {
			return models.OptionNode{}, warnings, nil, err
		} else if exist {
			karmaTestDetected = true
		}
	}
	log.TPrintf("karma.conf.js found: %v", karmaTestDetected)

	scanner.hasKarmaJasmineTest = karmaTestDetected
	// ---

	// Search for jasmine tests
	jasminTestDetected := false

	if !karmaTestDetected {
		log.TPrintf("Searching for jasmine test")

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
		log.TPrintf("jasmine dependency found: %v", jasmineDependencyFound)

		if jasmineDependencyFound {
			jasmineConfigJSONPth := filepath.Join(projectRootDir, "spec", "support", "jasmine.json")
			if exist, err := pathutil.IsPathExists(jasmineConfigJSONPth); err != nil {
				return models.OptionNode{}, warnings, nil, err
			} else if exist {
				jasminTestDetected = true
			}
		}

		log.TPrintf("jasmine.json found: %v", jasminTestDetected)

		scanner.hasJasmineTest = jasminTestDetected
	}
	// ---

	// Configure Cordova
	cordovaConfigExist, err := pathutil.IsPathExists(filepath.Join(projectRootDir, "config.xml"))
	if err != nil {
		return models.OptionNode{},
			warnings,
			nil,
			fmt.Errorf("failed to search for config.xml file: %s", err)
	}

	log.TPrintf("config.xml: %s", filepath.Join(projectRootDir, "config.xml"))

	if !cordovaConfigExist {
		warning := fmt.Sprintf("Cordova config.xml not found.")
		warnings = append(warnings, warning)
	}

	// Get relative config.xml dir
	cordovaConfigDir := filepath.Dir(scanner.ionicConfigPath)
	relCordovaConfigDir, err := utility.RelPath(scanner.searchDir, cordovaConfigDir)
	if err != nil {
		return models.OptionNode{},
			warnings,
			nil,
			fmt.Errorf("Failed to get relative config.xml dir path, error: %s", err)
	}
	if relCordovaConfigDir == "." {
		// config.xml placed in the search dir, no need to change-dir in the workflows
		relCordovaConfigDir = ""
	}
	scanner.relCordovaConfigDir = relCordovaConfigDir
	// ---

	// Options
	var rootOption *models.OptionNode

	platforms := []string{"ios", "android", "ios,android"}

	if relCordovaConfigDir != "" {
		rootOption = models.NewOption(workDirInputTitle, workDirInputSummary, workDirInputEnvKey, models.TypeSelector)

		projectTypeOption := models.NewOption(platformInputTitle, platformInputSummary, platformInputEnvKey, models.TypeSelector)
		rootOption.AddOption(relCordovaConfigDir, projectTypeOption)

		for _, platform := range platforms {
			configOption := models.NewConfigOption(configName, nil)
			projectTypeOption.AddConfig(platform, configOption)
		}
	} else {
		rootOption = models.NewOption(platformInputTitle, platformInputSummary, platformInputEnvKey, models.TypeSelector)

		for _, platform := range platforms {
			configOption := models.NewConfigOption(configName, nil)
			rootOption.AddConfig(platform, configOption)
		}
	}
	// ---

	return *rootOption, warnings, nil, nil
}

// DefaultOptions ...
func (Scanner) DefaultOptions() models.OptionNode {
	workDirOption := models.NewOption(workDirInputTitle, workDirInputSummary, workDirInputEnvKey, models.TypeUserInput)

	projectTypeOption := models.NewOption(platformInputTitle, platformInputSummary, platformInputEnvKey, models.TypeSelector)
	workDirOption.AddOption("", projectTypeOption)

	platforms := []string{
		"ios",
		"android",
		"ios,android",
	}
	for _, platform := range platforms {
		configOption := models.NewConfigOption(defaultConfigName, nil)
		projectTypeOption.AddConfig(platform, configOption)
	}

	return *workDirOption
}

// Configs ...
func (scanner *Scanner) Configs(_ bool) (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)

	workdirEnvList := []envmanModels.EnvironmentItemModel{}
	if scanner.relCordovaConfigDir != "" {
		workdirEnvList = append(workdirEnvList, envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey})
	}

	if scanner.hasJasmineTest || scanner.hasKarmaJasmineTest {
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))

		// CI
		if scanner.hasKarmaJasmineTest {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.KarmaJasmineTestRunnerStepListItem(workdirEnvList...))
		} else if scanner.hasJasmineTest {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.JasmineTestRunnerStepListItem(workdirEnvList...))
		}
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

		// CD
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(false)...)
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))

		if scanner.hasKarmaJasmineTest {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.KarmaJasmineTestRunnerStepListItem(workdirEnvList...))
		} else if scanner.hasJasmineTest {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.JasmineTestRunnerStepListItem(workdirEnvList...))
		}

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.GenerateCordovaBuildConfigStepListItem())

		ionicArchiveEnvs := []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{platformInputKey: "$" + platformInputEnvKey},
			envmanModels.EnvironmentItemModel{targetInputKey: targetEmulator},
		}
		if scanner.relCordovaConfigDir != "" {
			ionicArchiveEnvs = append(ionicArchiveEnvs, envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey})
		}
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.IonicArchiveStepListItem(ionicArchiveEnvs...))
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

		config, err := configBuilder.Generate(scannerName)
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

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.GenerateCordovaBuildConfigStepListItem())

	ionicArchiveEnvs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{platformInputKey: "$" + platformInputEnvKey},
		envmanModels.EnvironmentItemModel{targetInputKey: targetEmulator},
	}
	if scanner.relCordovaConfigDir != "" {
		ionicArchiveEnvs = append(ionicArchiveEnvs, envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey})
	}
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.IonicArchiveStepListItem(ionicArchiveEnvs...))
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

	config, err := configBuilder.Generate(scannerName)
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
func (Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(
		envmanModels.EnvironmentItemModel{"command": "install"},
		envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey}))

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.GenerateCordovaBuildConfigStepListItem())
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.IonicArchiveStepListItem(
		envmanModels.EnvironmentItemModel{workDirInputKey: "$" + workDirInputEnvKey},
		envmanModels.EnvironmentItemModel{platformInputKey: "$" + platformInputEnvKey},
		envmanModels.EnvironmentItemModel{targetInputKey: targetEmulator}))

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

	config, err := configBuilder.Generate(scannerName)
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
