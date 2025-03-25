package nodejs

import (
	"slices"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/go-utils/log"
)

// Options
const (
	ScannerName = "node-js"

	// Files to look for
	packageJson = "package.json"

	projectRootDirInputTitle   = "Project Root Directory"
	projectRootDirInputSummary = "The directory containing the package.json file"
	projectRootDirInputEnvKey  = "NODEJS_ROOT_DIR"

	packageManagerInputTitle   = "Package Manager"
	packageManagerInputSummary = "The package manager used in the project"
	packageManagerInputEnvKey  = "NODEJS_PACKAGE_MANAGER"

	nodeVersionInputTitle   = "Node Version"
	nodeVersionInputSummary = "The version of Node.js used in the project. To use the latest Node version, leave this empty"
	nodeVersionInputEnvKey  = "NODEJS_VERSION"
)

type packageManager struct {
	name     string
	lockFile string
}

var pkgManagers = []packageManager{
	{"npm", "package-lock.json"},
	{"yarn", "yarn.lock"},
}

// Scanner implements the Scanner interface for Node.js projects
type Scanner struct {
	packageJsonPath string
	packageManager  string
	nodeVersion     string
	buildScript     string
	lintScript      string
	testScript      string
	scripts         []string
}

// NewScanner creates a new scanner instance.
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name returns the name of the scanner
func (scanner *Scanner) Name() string {
	return ScannerName
}

// DetectPlatform checks if the given search directory contains a Node.js project
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	scanner.packageJsonPath = checkPackageJSON(searchDir)
	if scanner.packageJsonPath == "" {
		log.TPrintf("Platform not detected")
		return false, nil
	}

	scanner.nodeVersion = checkNodeVersion(searchDir)
	scanner.packageManager = checkPackageManager(searchDir)
	scanner.scripts = checkPackageScripts(scanner.packageJsonPath)

	if slices.Contains(scanner.scripts, "build") {
		log.TPrintf("- build - found")
		scanner.buildScript = "build"
	} else {
		log.TPrintf("- build - not found")
	}

	if slices.Contains(scanner.scripts, "lint") {
		log.TPrintf("- lint - found")
		scanner.lintScript = "lint"
	} else {
		log.TPrintf("- lint - not found")
	}

	if slices.Contains(scanner.scripts, "test") {
		log.TPrintf("- test - found")
		scanner.testScript = "test"
	} else {
		log.TPrintf("- test - not found")
	}

	log.TSuccessf("Platform detected")

	return true, nil
}

func (scanner *Scanner) ExcludedScannerNames() []string {
	return []string{}
}

// Options returns the options for the scanner
func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	projectRootOption := models.NewOption(projectRootDirInputTitle, projectRootDirInputSummary, projectRootDirInputEnvKey, models.TypeSelector)
	var nodeVersionOption *models.OptionNode
	if scanner.nodeVersion != "" {
		nodeVersionOption = models.NewOption(nodeVersionInputTitle, nodeVersionInputSummary, nodeVersionInputEnvKey, models.TypeSelector)
	} else {
		nodeVersionOption = models.NewOption(nodeVersionInputTitle, nodeVersionInputSummary, nodeVersionInputEnvKey, models.TypeOptionalUserInput)
	}
	packageManagerOption := models.NewOption(packageManagerInputTitle, packageManagerInputSummary, packageManagerInputEnvKey, models.TypeSelector)

	if scanner.packageManager != "" {
		configOption := models.NewConfigOption(configName(scanner.packageManager, false), nil)
		packageManagerOption.AddConfig(scanner.packageManager, configOption)
	} else {
		for _, pkgMgr := range pkgManagers {
			configOption := models.NewConfigOption(configName(pkgMgr.name, false), nil)
			packageManagerOption.AddConfig(pkgMgr.name, configOption)
		}
	}

	projectRootOption.AddOption(models.UserInputOptionDefaultValue, nodeVersionOption)
	nodeVersionOption.AddOption(models.UserInputOptionDefaultValue, packageManagerOption)

	return *projectRootOption, models.Warnings{}, nil, nil
}

// DefaultOptions returns the default options for the scanner
func (scanner *Scanner) DefaultOptions() models.OptionNode {
	projectRootOption := models.NewOption(projectRootDirInputTitle, projectRootDirInputSummary, projectRootDirInputEnvKey, models.TypeUserInput)
	nodeVersionOption := models.NewOption(nodeVersionInputTitle, nodeVersionInputSummary, nodeVersionInputEnvKey, models.TypeOptionalUserInput)
	packageManagerOption := models.NewOption(packageManagerInputTitle, packageManagerInputSummary, packageManagerInputEnvKey, models.TypeSelector)

	for _, pkgMgr := range pkgManagers {
		configOption := models.NewConfigOption(configName(pkgMgr.name, true), nil)
		packageManagerOption.AddConfig(pkgMgr.name, configOption)
	}

	projectRootOption.AddOption(models.UserInputOptionDefaultValue, nodeVersionOption)
	nodeVersionOption.AddOption(models.UserInputOptionDefaultValue, packageManagerOption)

	return *projectRootOption
}

// Configs returns the default configurations for the scanner
func (scanner *Scanner) Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	configs := models.BitriseConfigMap{}

	if scanner.packageManager != "" {
		config, err := generateConfig(sshKeyActivation, scanner.packageManager, scanner.nodeVersion)
		if err != nil {
			return nil, err
		}
		configs[configName(scanner.packageManager, false)] = config
	} else {
		for _, pkgMgr := range pkgManagers {
			config, err := generateConfig(sshKeyActivation, pkgMgr.name, scanner.nodeVersion)
			if err != nil {
				return nil, err
			}
			configs[configName(pkgMgr.name, false)] = config
		}
	}

	return configs, nil
}

func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configs := models.BitriseConfigMap{}

	for _, pkgMgr := range pkgManagers {
		config, err := generateConfig(models.SSHKeyActivationConditional, pkgMgr.name, "")
		if err != nil {
			return nil, err
		}
		configs[configName(pkgMgr.name, true)] = config
	}

	return configs, nil
}
