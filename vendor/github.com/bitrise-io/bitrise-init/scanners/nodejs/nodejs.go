package nodejs

import (
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/log"
)

// Options
const (
	ScannerName = "node-js"

	runTestsWorkflowID = models.WorkflowID("run_tests")

	projectDirInputTitle   = "Project Directory"
	projectDirInputSummary = "The directory containing the package.json file"
	projectDirInputEnvKey  = "NODEJS_PROJECT_DIR"

	packageManagerInputTitle   = "Package Manager"
	packageManagerInputSummary = "The package manager used in the project"

	nodeVersionInputTitle           = "Node.js version"
	nodeVersionInputSummary         = "The Node.js version to be used for the project. Use exact (20.10.0) or partial (22:latest, 20:installed) versions."
	nodeVersionEnvKey               = "NODEJS_VERSION"
	nodeVersionInstallScriptContent = `#!/usr/bin/env bash
set -euxo pipefail

bitrise tools install nodejs $NODEJS_VERSION
`
)

type packageManager struct {
	name     string
	lockFile string
}

var pkgManagers = []packageManager{
	{"npm", "package-lock.json"},
	{"yarn", "yarn.lock"},
}

type project struct {
	projectRelDir  string
	packageManager string
	scripts        []string
	hasTest        bool
	hasLint        bool
	framework      string
	nodeVersion    string
}

// Scanner implements the Scanner interface for Node.js projects
type Scanner struct {
	projects []project
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
	pkgJsonPaths, err := utility.CollectPackageJSONFiles(searchDir)
	if err != nil {
		log.TWarnf("%s", err)
		log.TPrintf("Platform not detected")
		return false, nil
	}

	for _, packageJsonPath := range pkgJsonPaths {
		log.TPrintf("Checking: %s", packageJsonPath)

		// determine workdir
		pkgJsonDir := filepath.Dir(packageJsonPath)

		pkgMgr := checkPackageManager(pkgJsonDir)
		results, err := checkPackageScripts(packageJsonPath)
		if err != nil {
			log.TWarnf("Failed to check package scripts: %s", err)
			continue
		}
		framework := detectFramework(packageJsonPath)
		nodeVersion := detectNodeVersion(pkgJsonDir, packageJsonPath)

		projectRelDir, err := utility.RelPath(searchDir, pkgJsonDir)
		if err != nil {
			log.TWarnf("failed to get relative package.json dir path: %s", err)
			continue
		}

		project := project{
			projectRelDir:  projectRelDir,
			packageManager: pkgMgr,
			scripts:        results.scripts,
			hasTest:        results.hasTest,
			hasLint:        results.hasLint,
			framework:      framework,
			nodeVersion:    nodeVersion,
		}

		scanner.projects = append(scanner.projects, project)
	}

	if len(scanner.projects) == 0 {
		log.TPrintf("Platform not detected")
		return false, nil
	}

	log.TSuccessf("Platform detected")
	return true, nil
}

func (scanner *Scanner) ExcludedScannerNames() []string {
	return []string{}
}

// Options returns the options for the scanner
func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	return generateOptions(scanner.projects)
}

// Configs returns the default configurations for the scanner
func (scanner *Scanner) Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	return generateConfigs(scanner.projects, sshKeyActivation)
}

// DefaultOptions returns the default options for the scanner
func (scanner *Scanner) DefaultOptions() models.OptionNode {
	projectRootOption := models.NewOption(projectDirInputTitle, projectDirInputSummary, projectDirInputEnvKey, models.TypeUserInput)
	packageManagerOption := models.NewOption(packageManagerInputTitle, packageManagerInputSummary, "", models.TypeSelector)
	nodeVersionOption := models.NewOption(nodeVersionInputTitle, nodeVersionInputSummary, nodeVersionEnvKey, models.TypeUserInput)

	projectRootOption.AddOption(models.UserInputOptionDefaultValue, nodeVersionOption)
	nodeVersionOption.AddOption(models.UserInputOptionDefaultValue, packageManagerOption)

	for _, pkgMgr := range pkgManagers {
		defaultDescriptor := createDefaultConfigDescriptor(pkgMgr.name)
		configOption := models.NewConfigOption(configName(defaultDescriptor), nil)
		packageManagerOption.AddConfig(pkgMgr.name, configOption)
	}

	return *projectRootOption
}

func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configs := models.BitriseConfigMap{}

	for _, pkgMgr := range pkgManagers {
		defaultDescriptor := createDefaultConfigDescriptor(pkgMgr.name)
		config, err := generateConfigBasedOn(defaultDescriptor, models.SSHKeyActivationConditional)
		if err != nil {
			return nil, err
		}
		configs[configName(defaultDescriptor)] = config
	}

	return configs, nil
}
