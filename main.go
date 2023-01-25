package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/bitrise-io/bitrise-init/scanner"
	"github.com/bitrise-io/go-steputils/step"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	cmdv2 "github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	logv2 "github.com/bitrise-io/go-utils/v2/log"
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
	EnableRepoClone bool `env:"enable_repo_clone"`

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
	cfg.RepositoryURL = strings.TrimSpace(cfg.RepositoryURL)
	cfg.Branch = strings.TrimSpace(cfg.Branch)
	if cfg.RepositoryURL == "" {
		return newStepError(
			"input_parse_failed",
			errors.New("repository URL input missing"),
			"Repository URL unspecified",
		)
	}
	if cfg.Branch == "" {
		return newStepError(
			"input_parse_failed",
			errors.New("repository bracnh input missing"),
			"Repository branch unspecified",
		)
	}

	// Activate SSH key is optional
	if cfg.SSHRsaPrivateKey != "" {
		if err := activatesshkey.Execute(activatesshkey.Config{
			SSHRsaPrivateKey:        cfg.SSHRsaPrivateKey,
			SSHKeySavePath:          path.Join(pathutil.UserHomeDir(), ".ssh", "steplib_ssh_step_id_rsa"),
			IsRemoveOtherIdentities: false,
		}); err != nil {
			return err
		}
	}

	// Git clone
	logger := logv2.NewLogger()
	envRepo := env.NewRepository()
	tracker := gitclone.NewStepTracker(envRepo, logger)
	cmdFactory := cmdv2.NewFactory(envRepo)
	gitcloner := gitclone.NewGitCloner(logger, tracker, cmdFactory)
	config := gitclone.Config{
		RepositoryURL: cfg.RepositoryURL,
		CloneIntoDir:  cfg.CloneIntoDir, // Using same directory later to run scan
		Branch:        cfg.Branch,

		// BuildURL and BuildAPIToken used for merging only
		BuildURL:      "",
		BuildAPIToken: "",

		UpdateSubmodules: true,
	}
	if _, err := gitcloner.CheckoutState(config); err != nil {
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
		if resultClient, err = newResultClient(cfg.ResultSubmitURL, cfg.ResultSubmitAPIToken); err != nil {
			failf(fmt.Sprintf("%v", err))
		}
	}

	if !(runtime.GOOS == "darwin" || runtime.GOOS == "linux") {
		failf("Unsupported OS: %s", runtime.GOOS)
	}

	if cfg.EnableRepoClone {
		handleStepError := func(stepID, tag string, err error, shortMsg string) {
			LogError(stepID, tag, err, shortMsg)
			if resultClient != nil {
				if err := resultClient.uploadErrorResult(stepID, err); err != nil {
					log.TWarnf("Failed to submit result: %s", err)
				}
			}
		}

		if err := cloneRepo(repoConfig{
			CloneIntoDir:     cfg.ScanDirectory,
			RepositoryURL:    cfg.RepositoryURL,
			SSHRsaPrivateKey: cfg.SSHRsaPrivateKey,
			Branch:           cfg.Branch,
		}); err != nil {
			if stepError, ok := err.(*step.Error); ok {
				handleStepError(stepError.StepID, stepError.Tag, stepError, stepError.ShortMsg)
			} else {
				wrappedStepError := newStepError("error_cast_failed", err, "Failed to cast error")
				handleStepError(wrappedStepError.StepID, wrappedStepError.Tag, wrappedStepError.Err, wrappedStepError.ShortMsg)
			}

			failf("%v", err)
		}
	}

	searchDir, err := pathutil.AbsPath(cfg.ScanDirectory)
	if err != nil {
		failf("failed to expand path (%s), error: %s", cfg.ScanDirectory, err)
	}

	isPrivateRepo := cfg.SSHRsaPrivateKey != ""
	result, platformsDetected := scanner.GenerateScanResult(searchDir, isPrivateRepo)

	// Upload results
	if resultClient != nil {
		log.TInfof("Submitting results...")
		if err := resultClient.uploadResults(result); err != nil {
			failf("Could not send back results: %s", err)
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
