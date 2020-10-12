package activatesshkey

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

func unsetEnvsBy(value string) error {
	for _, env := range os.Environ() {
		key, val := splitEnv(env)

		if val == value {
			if err := os.Unsetenv(key); err != nil {
				return err
			}

			if err := tools.ExportEnvironmentWithEnvman(key, ""); err != nil {
				return err
			}
			log.Debugf("%s has been unset", key)
		}
	}
	return nil
}

func splitEnv(env string) (string, string) {
	e := strings.Split(env, "=")
	return e[0], strings.Join(e[1:], "=")
}

func ensureSavePath(savePath string) error {
	dirpath := filepath.Dir(savePath)
	return os.MkdirAll(dirpath, 0700)
}

func restartAgent(removeOtherIdentities bool) error {
	var shouldStartNewAgent bool
	cmd := command.New("ssh-add", "-l")
	cmd.SetStderr(os.Stderr)
	log.Printf("$ %s", cmd.PrintableCommandArgs())

	returnValue, err := cmd.RunAndReturnExitCode()
	if err != nil {
		log.Debugf("Exit code: %s", err)
	}

	//  as stated in the man page (https://developer.apple.com/library/mac/documentation/Darwin/Reference/ManPages/man1/ssh-add.1.html)
	//  ssh-add returns the exit code 2 if it could not connect to the ssh-agent
	if returnValue == 2 {
		log.Printf("ssh_agent_check_result: %d", returnValue)
		log.Printf("ssh-agent not started")
		shouldStartNewAgent = true
	} else {
		// ssh-agent loaded and accessible
		log.Printf("ssh_agent_check_result: %d", returnValue)
		fmt.Printf("running / accessible ssh-agent detected")
		if removeOtherIdentities {
			// remove all keys from the current agent
			cmdRemove := command.New("ssh-add", "-D")
			cmdRemove.SetStdout(os.Stdout)
			cmdRemove.SetStderr(os.Stderr)

			fmt.Println()
			fmt.Println()
			log.Printf("$ %s", cmdRemove.PrintableCommandArgs())

			if err := cmdRemove.Run(); err != nil {
				return err
			}

			// try to kill the agent
			cmdKill := command.New("ssh-agent", "-k")
			cmdKill.SetStdout(os.Stdout)
			cmdKill.SetStderr(os.Stderr)

			fmt.Println()
			log.Printf("$ %s", cmdKill.PrintableCommandArgs())

			returnValue, err := cmdKill.RunAndReturnExitCode()
			if err != nil {
				log.Printf("Exit code: %s", err)
			}

			if returnValue == 0 {
				shouldStartNewAgent = true
			}
		}
	}

	if shouldStartNewAgent {
		fmt.Printf("starting a new ssh-agent and exporting connection information with envman")
		cmd := command.New("ssh-agent")

		fmt.Println()
		log.Printf("$ %s", cmd.PrintableCommandArgs())

		returnValue, err := cmd.RunAndReturnTrimmedOutput()
		if err != nil {
			log.Debugf("Exit code: %s", err)
		}

		fmt.Printf("Expose SSH_AUTH_SOCK for the new ssh-agent, with envman")

		returnValue = strings.TrimPrefix(returnValue, "SSH_AUTH_SOCK=")
		returnValue = strings.Split(returnValue, ";")[0]

		if err = os.Setenv("SSH_AUTH_SOCK", returnValue); err != nil {
			return fmt.Errorf("Failed to set SSH_AUTH_SOCK env")
		}

		return tools.ExportEnvironmentWithEnvman("SSH_AUTH_SOCK", returnValue)
	}
	return nil
}

// No passphrase allowed, fail if ssh-add prompts for one
// (in case the key can't be added without a passphrase)
func checkPassphrase(savePath string) error {

	spawnString := `expect <<EOD
spawn ssh-add ` + savePath + `
expect {
	"Enter passphrase for" {
		exit 1
	}
	"Identity added" {
		exit 0
	}
}
send "nopass\n"
EOD
if [ $? -ne 0 ] ; then
exit 1
fi`

	pth, err := pathutil.NormalizedOSTempDirPath("spawn")
	if err != nil {
		return err
	}

	filePth := filepath.Join(pth, "tmp_spawn.sh")
	if err := fileutil.WriteStringToFile(filePth, spawnString); err != nil {
		return fmt.Errorf("failed to write the SSH key to the provided path, %s", err)
	}

	if err := os.Chmod(filePth, 0770); err != nil {
		return err
	}

	cmd := command.New("bash", "-c", filePth)
	cmd.SetStderr(os.Stderr)
	cmd.SetStdout(os.Stdout)

	fmt.Println()
	log.Printf("$ %s", cmd.PrintableCommandArgs())

	exitCode, err := cmd.RunAndReturnExitCode()
	if err != nil {
		log.Debugf("Exit code: %s", err)
	}

	if exitCode != 0 {
		log.Errorf("\nExit code: %d", exitCode)
		return fmt.Errorf("failed to add the SSH key to ssh-agent with an empty passphrase")
	}

	return nil
}
