package activatesshkey

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
)

// Config is the activate SSH key step configuration
type Config struct {
	SSHRsaPrivateKey        stepconf.Secret `env:"ssh_rsa_private_key,required"`
	SSHKeySavePath          string          `env:"ssh_key_save_path,required"`
	IsRemoveOtherIdentities bool            `env:"is_remove_other_identities,required"`
	Verbose                 bool            `env:"verbose"`
}

// Execute activates a given SSH key
func Execute(cfg Config) error {
	// Remove SSHRsaPrivateKey from envs
	if err := unsetEnvsBy(string(cfg.SSHRsaPrivateKey)); err != nil {
		return newStepError(
			"removing_private_key_data_failed",
			fmt.Errorf("failed to remove private key data from envs: %v", err),
			"Failed to remove private key data from envs",
		)
	}

	if err := ensureSavePath(cfg.SSHKeySavePath); err != nil {
		return newStepError(
			"creating_ssh_save_path_failed",
			fmt.Errorf("failed to create the provided path: %v", err),
			"Failed to create the provided path",
		)
	}

	// OpenSSH_8.1p1 on macOS requires a newline at at the end of
	// private key using the new format (starting with -----BEGIN OPENSSH PRIVATE KEY-----).
	// See https://www.openssh.com/txt/release-7.8 for new format description.
	if err := fileutil.WriteStringToFile(cfg.SSHKeySavePath, string(cfg.SSHRsaPrivateKey)+"\n"); err != nil {
		return newStepError(
			"writing_ssh_key_failed",
			fmt.Errorf("failed to write the SSH key to the provided path: %v", err),
			"Failed to write the SSH key to the provided path",
		)
	}

	if err := os.Chmod(cfg.SSHKeySavePath, 0600); err != nil {
		return newStepError(
			"changing_ssh_key_permission_failed",
			fmt.Errorf("failed to change file's access permission: %v", err),
			"Failed to change file's access permission",
		)
	}

	if err := restartAgent(cfg.IsRemoveOtherIdentities); err != nil {
		return newStepError(
			"restarting_ssh_agent_failed",
			fmt.Errorf("failed to restart SSH Agent: %v", err),
			"Failed to restart SSH Agent",
		)
	}

	if err := checkPassphrase(cfg.SSHKeySavePath); err != nil {
		return newStepError(
			"ssh_key_requries_passphrase",
			fmt.Errorf("SSH key requires passphrase: %v", err),
			"SSH key requires passphrase",
		)
	}

	fmt.Println()
	log.Donef("Success")
	log.Printf("The SSH key was saved to %s", cfg.SSHKeySavePath)
	log.Printf("and was successfully added to ssh-agent.")

	return nil
}
