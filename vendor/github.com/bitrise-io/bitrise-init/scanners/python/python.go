package python

import (
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/log"
)

const (
	scannerName = "python"

	projectDirInputTitle   = "Python Project Directory"
	projectDirInputSummary = "The directory containing the Python project files (requirements.txt, pyproject.toml, etc.)"
	projectDirInputEnvKey  = "PYTHON_PROJECT_DIR"

	packageManagerInputTitle   = "Package Manager"
	packageManagerInputSummary = "The package manager used in the project"

	pythonVersionInputTitle   = "Python version"
	pythonVersionInputSummary = "The Python version to be used for the project. Use exact (3.12.0) or partial (3.12:latest, 3:installed) versions."
	pythonVersionEnvKey       = "PYTHON_VERSION"
)

var packageManagers = []string{"pip", "poetry", "uv"}

type project struct {
	projectRelDir       string
	packageManager      string
	hasPytest           bool
	pythonVersion       string
	devRequirementsFile string
}

// Scanner implements ScannerInterface for Python projects.
type Scanner struct {
	searchDir   string
	projectDirs []string // relative paths, populated by DetectPlatform
	projects    []project // populated by Options()
}

// NewScanner creates a new Scanner instance.
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name returns the scanner name.
func (s *Scanner) Name() string {
	return scannerName
}

// DetectPlatform checks whether searchDir contains a Python project.
// It only confirms the platform is present; detailed analysis happens in Options().
func (s *Scanner) DetectPlatform(searchDir string) (bool, error) {
	s.searchDir = searchDir

	dirs, err := collectPythonProjectDirs(searchDir)
	if err != nil {
		log.TWarnf("%s", err)
		log.TPrintf("Platform not detected")
		return false, nil
	}

	for _, dir := range dirs {
		relDir, err := utility.RelPath(searchDir, dir)
		if err != nil {
			log.TWarnf("failed to get relative project dir path: %s", err)
			continue
		}
		log.TPrintf("Python project found: %s", relDir)
		s.projectDirs = append(s.projectDirs, relDir)
	}

	if len(s.projectDirs) == 0 {
		log.TPrintf("Platform not detected")
		return false, nil
	}

	log.TSuccessf("Platform detected")
	return true, nil
}

// ExcludedScannerNames returns scanners to skip when this scanner detects.
func (s *Scanner) ExcludedScannerNames() []string {
	return []string{}
}

// Options performs detailed analysis for each detected project dir and builds the option tree.
func (s *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	for _, relDir := range s.projectDirs {
		absDir := filepath.Join(s.searchDir, relDir)
		log.TPrintf("Checking: %s", relDir)

		pkgMgr := detectPackageManager(absDir)
		pythonVersion := detectPythonVersion(absDir)
		hasPytest := detectTestRunner(absDir)
		devReqFile := detectDevRequirementsFile(absDir)
		detectFramework(absDir)

		s.projects = append(s.projects, project{
			projectRelDir:       relDir,
			packageManager:      pkgMgr,
			pythonVersion:       pythonVersion,
			hasPytest:           hasPytest,
			devRequirementsFile: devReqFile,
		})
	}

	return generateOptions(s.projects)
}

// Configs generates the pre-made bitrise.yml templates for each detected project.
func (s *Scanner) Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	return generateConfigs(s.projects, sshKeyActivation)
}

// DefaultOptions returns the option tree for the manual configuration flow.
func (s *Scanner) DefaultOptions() models.OptionNode {
	projectDirOption := models.NewOption(projectDirInputTitle, projectDirInputSummary, projectDirInputEnvKey, models.TypeUserInput)
	versionOption := models.NewOption(pythonVersionInputTitle, pythonVersionInputSummary, pythonVersionEnvKey, models.TypeUserInput)
	pkgMgrOption := models.NewOption(packageManagerInputTitle, packageManagerInputSummary, "", models.TypeSelector)

	projectDirOption.AddOption(models.UserInputOptionDefaultValue, versionOption)
	versionOption.AddOption(models.UserInputOptionDefaultValue, pkgMgrOption)

	for _, pm := range packageManagers {
		descriptor := createDefaultConfigDescriptor(pm)
		configOption := models.NewConfigOption(configName(descriptor), nil)
		pkgMgrOption.AddConfig(pm, configOption)
	}

	return *projectDirOption
}

// DefaultConfigs generates the static bitrise.yml templates for the manual configuration flow.
func (s *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configs := models.BitriseConfigMap{}

	for _, pm := range packageManagers {
		descriptor := createDefaultConfigDescriptor(pm)
		config, err := generateConfigBasedOn(descriptor, models.SSHKeyActivationConditional)
		if err != nil {
			return nil, err
		}
		configs[configName(descriptor)] = config
	}

	return configs, nil
}
