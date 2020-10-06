package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanner"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-steplib/steps-activate-ssh-key/activatesshkey"
	"github.com/bitrise-steplib/steps-git-clone/gitclone"
)

type config struct {
	ScanDirectory        string          `env:"scan_dir,dir"`
	ResultSubmitURL      string          `env:"scan_result_submit_url"`
	ResultSubmitAPIToken stepconf.Secret `env:"scan_result_submit_api_token"`
	IconCandidatesURL    string          `env:"icon_candidates_url"`
	DebugLog             bool            `env:"verbose_log,opt[false,true]"`

	// Enable activate SSH key and git clone
	EnableRepoClone bool `env:"enable_repo_clone,opt[yes,no]"`

	// Activate SSH Key step
	SSHRsaPrivateKey stepconf.Secret `env:"ssh_rsa_private_key"`

	// Git clone step
	RepositoryURL string `env:"repository_url"`
	Branch        string `env:"branch"`
}

func failf(format string, args ...interface{}) {
	log.TErrorf(format, args...)
	os.Exit(1)
}

func uploadResults(submitURL *url.URL, result models.ScanResultModel) error {
	bytes, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal results, error: %s", err)
	}

	if err := retry.Times(1).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt != 0 {
			log.TWarnf("%d query attempt failed", attempt)
		}

		resp, err := http.Post(submitURL.String(), "application/json", strings.NewReader(string(bytes)))
		if err != nil {
			return fmt.Errorf("failed to submit results, error: %s", err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.TErrorf("failed to close response body, error: %s", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			log.TErrorf("Submit failed, status code: %d, headers: %s", resp.StatusCode, resp.Header)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body, error: %s", err)
			}
			return fmt.Errorf("failed to submit results, body: %s", body)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to submit, error: %s", err)
	}
	return nil
}

func printDirTree() {
	cmd := command.New("which", "tree")
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil || out == "" {
		log.TErrorf("tree not installed, can not list files")
	} else {
		fmt.Println()
		cmd := command.NewWithStandardOuts("tree", ".", "-L", "3")
		log.TPrintf("$ %s", cmd.PrintableCommandArgs())
		if err := cmd.Run(); err != nil {
			log.TErrorf("Failed to list files in current directory, error: %s", err)
		}
	}
}

func main() {
	var cfg config
	if err := stepconf.Parse(&cfg); err != nil {
		failf("Invalid configuration: %s", err)
	}
	stepconf.Print(cfg)

	if cfg.EnableRepoClone {
		if strings.TrimSpace(cfg.RepositoryURL) == "" {
			failf("Repository URL input missing.")
		}
		if strings.TrimSpace(cfg.Branch) == "" {
			failf("Repository branch input missing.")
		}
	}

	log.SetEnableDebugLog(cfg.DebugLog)

	if !(runtime.GOOS == "darwin" || runtime.GOOS == "linux") {
		failf("Unsupported OS: %s", runtime.GOOS)
	}

	if cfg.EnableRepoClone {
		// Activate SSH key is optional
		if cfg.SSHRsaPrivateKey != "" {
			if err := activatesshkey.Execute(activatesshkey.Config{
				SSHRsaPrivateKey:        cfg.SSHRsaPrivateKey,
				SSHKeySavePath:          "$HOME/.ssh/steplib_ssh_step_id_rsa",
				IsRemoveOtherIdentities: false,
			}); err != nil {
				failf("Activate SSH key failed: %v", err)
			}
		}

		// Git clone
		if err := gitclone.Execute(gitclone.Config{
			RepositoryURL: cfg.RepositoryURL,
			CloneIntoDir:  cfg.ScanDirectory, // Using same directory later to run scan
			Commit:        "",
			Tag:           "",
			Branch:        cfg.Branch,

			BranchDest:      "",
			PRID:            0,
			PRRepositoryURL: "",
			PRMergeBranch:   "",
			ResetRepository: false,
			CloneDepth:      0,

			// BuildURL and BuildAPIToken used for merging only
			BuildURL:      "",
			BuildAPIToken: "",

			UpdateSubmodules: true,
			ManualMerge:      true,
		}); err != nil {
			failf("Git clone step failed: %v", err)
		}
	}

	searchDir, err := pathutil.AbsPath(cfg.ScanDirectory)
	if err != nil {
		failf("failed to expand path (%s), error: %s", cfg.ScanDirectory, err)
	}

	result, platformsDetected := scanner.GenerateScanResult(searchDir)

	// Upload results
	if strings.TrimSpace(cfg.ResultSubmitURL) != "" {
		log.TInfof("Submitting results...")

		if strings.TrimSpace(string(cfg.ResultSubmitAPIToken)) == "" {
			log.TWarnf("Build trigger token is empty.")
		}
		submitURL, err := url.Parse(cfg.ResultSubmitURL)
		if err != nil {
			failf("could not parse submit URL, error: %s", err)
		}
		q := submitURL.Query()
		q.Add("api_token", url.QueryEscape(string(cfg.ResultSubmitAPIToken)))
		submitURL.RawQuery = q.Encode()

		if err := uploadResults(submitURL, result); err != nil {
			failf("Failed to submit results, error: %s", err)
		}
		log.TDonef("Submitted.")
	}

	// Upload icons
	if strings.TrimSpace(cfg.IconCandidatesURL) != "" {
		if err := uploadIcons(result.Icons,
			iconCandidateQuery{
				URL:               cfg.IconCandidatesURL,
				buildTriggerToken: string(cfg.ResultSubmitAPIToken),
			}); err != nil {
			log.TWarnf("Failed to submit icons, error: %s", err)
		}
	}

	if !platformsDetected {
		printDirTree()
		failf("No known platform detected")
	}
	log.TDonef("Scan finished.")
}
