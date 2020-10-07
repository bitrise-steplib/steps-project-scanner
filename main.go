package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/bitrise-io/bitrise-init/scanner"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
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

type repoConfig struct {
	CloneIntoDir     string
	RepositoryURL    string
	SSHRsaPrivateKey stepconf.Secret
	Branch           string
}

func cloneRepo(cfg repoConfig) error {
	if strings.TrimSpace(cfg.RepositoryURL) == "" {
		return &stepError{
			StepID:   "project-scanner",
			Tag:      "input_parse_failed",
			ShortMsg: "Repository URL unspecified",
			Err:      errors.New("repository URL input missing"),
		}
	}
	if strings.TrimSpace(cfg.Branch) == "" {
		return &stepError{
			StepID:   "project-scanner",
			Tag:      "input_parse_failed",
			ShortMsg: "Repository branch unspecified",
			Err:      errors.New("repository bracnh input missing"),
		}
	}

	// Activate SSH key is optional
	if cfg.SSHRsaPrivateKey != "" {
		if err := activatesshkey.Execute(activatesshkey.Config{
			SSHRsaPrivateKey:        cfg.SSHRsaPrivateKey,
			SSHKeySavePath:          "$HOME/.ssh/steplib_ssh_step_id_rsa",
			IsRemoveOtherIdentities: false,
		}); err != nil {
			if _, ok := err.(*activatesshkey.StepError); !ok {
				return &stepError{
					StepID:   "activate-ssh-key",
					Tag:      "unknown_error",
					ShortMsg: "Unknown error occured",
					Err:      err,
				}
			}
			return err
		}
	}

	// Git clone
	if err := gitclone.Execute(gitclone.Config{
		RepositoryURL: cfg.RepositoryURL,
		CloneIntoDir:  cfg.CloneIntoDir, // Using same directory later to run scan
		Branch:        cfg.Branch,

		// BuildURL and BuildAPIToken used for merging only
		BuildURL:      "",
		BuildAPIToken: "",

		UpdateSubmodules: true,
		ManualMerge:      true,
	}); err != nil {
		if _, ok := err.(*gitclone.StepError); !ok {
			return &stepError{
				StepID:   "git-clone",
				Tag:      "unknown_error",
				ShortMsg: "Unknown error occured",
				Err:      err,
			}
		}
		return err
	}

	return nil
}

func main() {
	var cfg config
	if err := stepconf.Parse(&cfg); err != nil {
		failf("Invalid configuration: %s", err)
	}
	stepconf.Print(cfg)
	log.SetEnableDebugLog(cfg.DebugLog)

	var resultClient *resultClient
	if strings.TrimSpace(cfg.ResultSubmitURL) != "" {
		if strings.TrimSpace(string(cfg.ResultSubmitAPIToken)) == "" {
			log.TWarnf("Build trigger token is empty.")
		}

		var err error
		if resultClient, err = newResultClient(cfg.RepositoryURL, cfg.ResultSubmitAPIToken); err != nil {
			failf(fmt.Sprintf("%v", err))
		}
	}

	if !(runtime.GOOS == "darwin" || runtime.GOOS == "linux") {
		failf("Unsupported OS: %s", runtime.GOOS)
	}

	if cfg.EnableRepoClone {
		handleStepError := func(stepID, tag string, err error, shortMsg string) {
			if resultClient != nil {
				resultClient.uploadErrorResult(stepID, tag, err, shortMsg)
			}
			LogError(stepID, tag, err, shortMsg)
		}

		if err := cloneRepo(repoConfig{
			CloneIntoDir:     cfg.ScanDirectory,
			RepositoryURL:    cfg.RepositoryURL,
			SSHRsaPrivateKey: cfg.SSHRsaPrivateKey,
			Branch:           cfg.Branch,
		}); err != nil {
			switch stepErr := err.(type) {
			case *activatesshkey.StepError:
				handleStepError(stepErr.StepID, stepErr.Tag, stepErr.Err, stepErr.ShortMsg)
			case *gitclone.StepError:
				handleStepError(stepErr.StepID, stepErr.Tag, stepErr.Err, stepErr.ShortMsg)
			case *stepError:
				handleStepError(stepErr.StepID, stepErr.Tag, stepErr.Err, stepErr.ShortMsg)
			default:
				handleStepError("project-scanner", "unknown_error", err, "Unknown error occured")
			}

			failf("%v", err)
		}
	}

	searchDir, err := pathutil.AbsPath(cfg.ScanDirectory)
	if err != nil {
		failf("failed to expand path (%s), error: %s", cfg.ScanDirectory, err)
	}

	result, platformsDetected := scanner.GenerateScanResult(searchDir)

	// Upload results
	if resultClient != nil {
		log.TInfof("Submitting results...")
		if err := resultClient.uploadResults(result); err != nil {
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
