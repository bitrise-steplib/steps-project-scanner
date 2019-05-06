package golang

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	envmanModels "github.com/bitrise-io/envman/models"
	giturls "github.com/whilp/git-urls"
	yaml "gopkg.in/yaml.v2"
)

const scannerName = "go"

const (
	configName = "go-config"
)

// Scanner ...
type Scanner struct {
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
	matches, err := filepath.Glob(filepath.Clean(searchDir) + "/*.go")
	if err != nil {
		return false, err
	}
	anyGoFileFound := matches != nil
	return anyGoFileFound, nil
}

// ExcludedScannerNames ...
func (*Scanner) ExcludedScannerNames() []string {
	return []string{}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, error) {
	return scanner.DefaultOptions(), models.Warnings{}, nil
}

// DefaultOptions ...
func (*Scanner) DefaultOptions() models.OptionNode {
	options := models.OptionNode{
		Title:  "Working directory (Some tools work best if your project is cloned into the GOPATH)",
		EnvKey: "GO_WORK_DIR",
		ChildOptionMap: map[string]*models.OptionNode{
			goWorkDir(): &models.OptionNode{
				Title: "Do you want to include a linter in the workflow? (You can modify the linter settings later in the Workflow Editor.)",
				ChildOptionMap: map[string]*models.OptionNode{
					"yes": &models.OptionNode{Config: configName},
					"no":  &models.OptionNode{Config: configName},
				},
			},
		},
	}

	// options.AddConfig("yes", &models.OptionNode{Config: configName})
	// options.AddConfig("no", &models.OptionNode{Config: configName})
	return options
}

// Configs ...
func (*Scanner) Configs() (models.BitriseConfigMap, error) {
	return confGen()
}

// DefaultConfigs ...
func (*Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	return confGen()
}

func confGen() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID,
		steps.ChangeWorkdirStepListItem(
			envmanModels.EnvironmentItemModel{"path": "$GO_WORK_DIR"},
		),
		steps.ActivateSSHKeyStepListItem(),
		steps.GitCloneStepListItem(),
		steps.GoLintBuildStepListItem(),
		steps.GoTestBuildStepListItem(),
		steps.DeployToBitriseIoStepListItem(),
	)

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

func goWorkDir() string {
	basePath := "$GOPATH/src"
	repoURL := os.Getenv("GIT_REPOSITORY_URL")
	if repoURL == "" {
		return ""
	}

	uri, err := giturls.Parse(repoURL)
	if err != nil {
		return ""
	}

	// Sometimes the path has opening and trailing slash
	path := strings.TrimPrefix(strings.TrimSuffix(uri.Path, "/"), "/")
	path = strings.TrimSuffix(path, ".git")

	return fmt.Sprintf("%s/%s/%s", basePath, uri.Host, path)
}
