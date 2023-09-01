package main

import (
	"encoding/json"
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
	"github.com/bitrise-io/go-utils/retry"
	cmdv2 "github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	logv2 "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-activate-ssh-key/activatesshkey"
	"github.com/bitrise-steplib/steps-git-clone/gitclone"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/bitriseapi"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/tracker"
	"github.com/bitrise-steplib/steps-git-clone/transport"
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

	// Git HTTP credentials
	GitHTTPUsername stepconf.Secret `env:"git_http_username"`
	GitHTTPPassword stepconf.Secret `env:"git_http_password"`

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
	GitHTTPUsername  stepconf.Secret
	GitHTTPPassword  stepconf.Secret
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

	redactedURL := redactURL(cfg.RepositoryURL)

	// Activate SSH key is optional
	if cfg.SSHRsaPrivateKey != "" {
		if err := activatesshkey.Execute(activatesshkey.Config{
			SSHRsaPrivateKey:        cfg.SSHRsaPrivateKey,
			SSHKeySavePath:          path.Join(pathutil.UserHomeDir(), ".ssh", "steplib_ssh_step_id_rsa"),
			IsRemoveOtherIdentities: false,
		}); err != nil {
			return newStepError(
				"activate_ssh_key_failed",
				err,
				fmt.Sprintf("Activating SSH key for %s failed", redactedURL),
			)
		}
	}

	// Activate Git HTTP credentials
	if err := transport.Setup(transport.Config{
		URL:          cfg.RepositoryURL,
		HTTPUsername: string(cfg.GitHTTPUsername),
		HTTPPassword: string(cfg.GitHTTPPassword),
	}); err != nil {
		return newStepError(
			"activate_git_http_credentials_failed",
			err,
			fmt.Sprintf("Activating Git HTTP credentials for %s failed", redactedURL),
		)
	}

	// Git clone
	logger := logv2.NewLogger()
	envRepo := env.NewRepository()

	stepTracker := tracker.NewStepTracker(envRepo, logger)
	cmdFactory := cmdv2.NewFactory(envRepo)
	// patchSource and mergeRefChecker used for merging only
	// build URL and build api token don't apply here
	patchSource := bitriseapi.NewPatchSource("", "")
	mergeRefChecker := bitriseapi.NewMergeRefChecker("", "", retry.NewHTTPClient(), logger, stepTracker)
	gitcloner := gitclone.NewGitCloner(logger, stepTracker, cmdFactory, patchSource, mergeRefChecker)
	config := gitclone.Config{
		RepositoryURL: cfg.RepositoryURL,
		CloneIntoDir:  cfg.CloneIntoDir, // Using the same directory later to run scan
		Branch:        cfg.Branch,

		UpdateSubmodules: true,
	}
	if _, err := gitcloner.CheckoutState(config); err != nil {
		if _, ok := err.(*step.Error); ok {
			return err
		}

		hasSSH := len(cfg.SSHRsaPrivateKey) > 0
		hasUser := len(cfg.GitHTTPUsername) > 0
		hasPass := len(cfg.GitHTTPPassword) > 0
		return newStepError(
			"git_clone_failed",
			err,
			fmt.Sprintf("Git clone for %s - %s branch failed (ssh: %t, user: %t, pass: %t)", redactedURL, cfg.Branch, hasSSH, hasUser, hasPass),
		)
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
	// Local file path can be specified with the 'path::' prefix. This can be used for debugging scan results locally.
	isLocalResultSubmitURL := strings.HasPrefix(cfg.ResultSubmitURL, "path::")
	if strings.TrimSpace(cfg.ResultSubmitURL) != "" && !isLocalResultSubmitURL {
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
			GitHTTPUsername:  cfg.GitHTTPUsername,
			GitHTTPPassword:  cfg.GitHTTPPassword,
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

	hasSSHKey := cfg.SSHRsaPrivateKey != ""
	result, platformsDetected := scanner.GenerateScanResult(searchDir, hasSSHKey)

	// Store results
	shouldSaveToFile := isLocalResultSubmitURL
	shouldStoreResult := resultClient != nil || shouldSaveToFile
	if shouldStoreResult {
		resultBytes, err := json.MarshalIndent(result, "", "\t")
		if err != nil {
			failf("failed to marshal results: %v", err)
		}

		if shouldSaveToFile {
			resultPth := strings.TrimPrefix(cfg.ResultSubmitURL, "path::")
			log.TInfof("Writing results: %s...", resultPth)
			if err := os.WriteFile(resultPth, resultBytes, os.ModePerm); err != nil {
				failf("Could not write results: %s", err)
			}

			log.TDonef("Results file created.")
		} else {
			log.TInfof("Submitting results...")
			if err := resultClient.uploadResults(resultBytes); err != nil {
				failf("Could not send results: %s", err)
			}

			log.TDonef("Results submitted.")
		}
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
