package scanners

import (
	"github.com/bitrise-core/bitrise-init/models"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/go-utils/pointers"
	stepmanModels "github.com/bitrise-io/stepman/models"
	"gopkg.in/yaml.v2"
)

const (
	stepActivateSSHKeyIDComposite                 = "activate-ssh-key@3.1.0"
	stepGitCloneIDComposite                       = "git-clone@3.2.0"
	stepCertificateAndProfileInstallerIDComposite = "certificate-and-profile-installer@1.5.0"
	stepDeployToBitriseIoIDComposite              = "deploy-to-bitrise-io@1.2.3"
)

// ScannerInterface ...
type ScannerInterface interface {
	Name() string
	Configure(searchDir string)

	DetectPlatform() (bool, error)

	Options() (models.OptionModel, error)
	DefaultOptions() models.OptionModel

	Configs() (map[string]string, error)
	DefaultConfigs() (map[string]string, error)
}

func customConfigName() string {
	return "custom-config"
}

// CustomConfig ...
func CustomConfig() (map[string]string, error) {
	bitriseDataMap := map[string]string{}
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

	bitriseData := models.BitriseDataWithPrimaryWorkflowSteps(steps)
	data, err := yaml.Marshal(bitriseData)
	if err != nil {
		return map[string]string{}, err
	}

	configName := customConfigName()
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}
