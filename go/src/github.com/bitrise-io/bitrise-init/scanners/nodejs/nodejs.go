package nodejs

import (
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	envmanModels "github.com/bitrise-io/envman/models"
	yaml "gopkg.in/yaml.v2"
)

const scannerName = "nodejs"

const (
	configName = "node-config"
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
	// searchDir, _ = filepath.Abs(searchDir)
	// matches, err := filepath.Glob(searchDir + "/package.json")
	// if err != nil {
	// 	return false, err
	// }
	// _ := matches != nil
	return true, nil
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
		Title: "We don't need to ask you anything. We know what we're doing.",
		ChildOptionMap: map[string]*models.OptionNode{
			"Are you sure?": &models.OptionNode{Config: configName},
			"All right ...": &models.OptionNode{Config: configName},
		},
	}

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
		steps.ActivateSSHKeyStepListItem(),
		steps.GitCloneStepListItem(),
		// steps.CachePullStepListItem(),
		steps.YarnStepListItem(),
		// steps.CachePushStepListItem(envmanModels.EnvironmentItemModel{"compress_archive": "true", "cache_paths": "$BITRISE_SOURCE_DIR/node_modules -> $BITRISE_SOURCE_DIR/yarn.lock"}),
		steps.YarnStepListItem(envmanModels.EnvironmentItemModel{"command": "lint"}),
		steps.YarnStepListItem(envmanModels.EnvironmentItemModel{"command": "build"}),
		steps.YarnStepListItem(envmanModels.EnvironmentItemModel{"command": "test"}),
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
