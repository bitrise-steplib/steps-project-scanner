package ruby

import (
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/log"
)

const (
	scannerName            = "ruby"
	projectDirInputTitle   = "Project Directory"
	projectDirInputSummary = "The directory containing the Gemfile"
	projectDirInputEnvKey  = "RUBY_PROJECT_DIR"
)

type project struct {
	projectRelDir  string
	hasBundler     bool
	hasRakefile    bool
	testFramework  string
	rubyVersion    string
	hasRails       bool
	databases      []databaseGem
	dbYMLInfo      databaseYMLInfo
	mongoidYMLInfo mongoidYMLInfo
}

type Scanner struct {
	projects []project
}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (scanner *Scanner) Name() string {
	return scannerName
}

func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	gemfilePaths, err := collectGemfiles(searchDir)
	if err != nil {
		log.TWarnf("%s", err)
		log.TPrintf("Platform not detected")
		return false, nil
	}

	for _, gemfilePath := range gemfilePaths {
		log.TPrintf("Checking: %s", gemfilePath)

		// determine workdir
		gemfileDir := filepath.Dir(gemfilePath)

		hasBundler := checkBundler(gemfileDir)
		hasRakefile := checkRakefile(gemfileDir)
		testFw := detectTestFramework(gemfileDir)
		rubyVersion := readRubyVersion(gemfileDir)
		hasRails := detectRails(gemfileDir)
		databases := detectDatabases(gemfileDir)
		var dbYMLInfo databaseYMLInfo
		if hasRelationalDB(databases) {
			dbYMLInfo = parseDatabaseYML(gemfileDir, databases)
		}
		var mongoidInfo mongoidYMLInfo
		if _, ok := findMongoDBGem(databases); ok {
			mongoidInfo = parseMongoidYML(gemfileDir)
		}

		projectRelDir, err := utility.RelPath(searchDir, gemfileDir)
		if err != nil {
			log.TWarnf("failed to get relative Gemfile dir path: %s", err)
			continue
		}

		project := project{
			projectRelDir:  projectRelDir,
			hasBundler:     hasBundler,
			hasRakefile:    hasRakefile,
			testFramework:  testFw,
			rubyVersion:    rubyVersion,
			hasRails:       hasRails,
			databases:      databases,
			dbYMLInfo:      dbYMLInfo,
			mongoidYMLInfo: mongoidInfo,
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

func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	return generateOptions(scanner.projects)
}

func (scanner *Scanner) Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	return generateConfigs(scanner.projects, sshKeyActivation)
}

func (scanner *Scanner) DefaultOptions() models.OptionNode {
	projectRootOption := models.NewOption(projectDirInputTitle, projectDirInputSummary, projectDirInputEnvKey, models.TypeUserInput)
	rubyVersionOption := models.NewOption(rubyVersionInputTitle, rubyVersionInputSummary, rubyVersionEnvKey, models.TypeUserInput)

	projectRootOption.AddOption(models.UserInputOptionDefaultValue, rubyVersionOption)

	defaultDescriptor := createDefaultConfigDescriptor()
	configOption := models.NewConfigOption(configName(defaultDescriptor), nil)
	rubyVersionOption.AddConfig(models.UserInputOptionDefaultValue, configOption)

	return *projectRootOption
}

func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configs := models.BitriseConfigMap{}

	defaultDescriptor := createDefaultConfigDescriptor()
	config, err := generateConfigBasedOn(defaultDescriptor, models.SSHKeyActivationConditional)
	if err != nil {
		return nil, err
	}
	configs[configName(defaultDescriptor)] = config

	return configs, nil
}
