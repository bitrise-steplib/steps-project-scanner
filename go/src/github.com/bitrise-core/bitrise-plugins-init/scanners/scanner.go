package scanners

import (
	"github.com/bitrise-core/bitrise-plugins-init/models"
	bitriseModels "github.com/bitrise-io/bitrise/models"
)

const (
	stepActivateSSHKeyIDComposite                 = "activate-ssh-key@3.1.0"
	stepGitCloneIDComposite                       = "git-clone@3.1.1"
	stepCertificateAndProfileInstallerIDComposite = "certificate-and-profile-installer@1.4.0"
	stepDeployToBitriseIoIDComposite              = "deploy-to-bitrise-io@1.2.2"
)

// ScannerInterface ...
type ScannerInterface interface {
	Name() string
	Configure(searchDir string)
	DetectPlatform() (bool, error)
	Analyze() ([]models.OptionModel, error)
	Configs(isPrivate bool) map[string]bitriseModels.BitriseDataModel
}
