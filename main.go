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
)

type config struct {
	ScanDirectory        string          `env:"scan_dir,dir"`
	ResultSubmitURL      string          `env:"scan_result_submit_url"`
	ResultSubmitAPIToken stepconf.Secret `env:"scan_result_submit_api_token"`
	IconCandidatesURL    string          `env:"icon_candidates_url"`
}

func failf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

func uploadResults(URL string, token string, result models.ScanResultModel) error {
	if strings.TrimSpace(token) == "" {
		log.Warnf("Build trigger token is empty.")
	}

	submitURL, err := url.Parse(URL)
	if err != nil {
		return fmt.Errorf("could not parse submit URL, error: %s", err)
	}
	q := submitURL.Query()
	q.Add("api_token", url.QueryEscape(token))
	submitURL.RawQuery = q.Encode()

	bytes, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal results, error: %s", err)
	}

	if err := retry.Times(1).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt != 0 {
			log.Warnf("%d query attempt failed", attempt)
		}

		resp, err := http.Post(submitURL.String(), "application/json", strings.NewReader(string(bytes)))
		if err != nil {
			return fmt.Errorf("failed to submit results, error: %s", err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Errorf("failed to close response body, error: %s", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			log.Errorf("Submit failed, status code: %d, headers: %s", resp.StatusCode, resp.Header)
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

	if !(runtime.GOOS == "darwin" || runtime.GOOS == "linux") {
		failf("Unsupported OS: %s", runtime.GOOS)
	}

	searchDir, err := pathutil.AbsPath(cfg.ScanDirectory)
	if err != nil {
		failf("failed to expand path (%s), error: %s", cfg.ScanDirectory, err)
	}

	result, platformsDetected := scanner.GenerateScanResult(searchDir)

	// Upload results
	if strings.TrimSpace(cfg.ResultSubmitURL) != "" {
		log.Infof("Submitting results...")
		err := uploadResults(cfg.ResultSubmitURL, string(cfg.ResultSubmitAPIToken), result)
		if err != nil {
			failf("Failed to submit results, error: %s", err)
		}
		log.Donef("Submitted.")
	}

	// Upload icons
	if strings.TrimSpace(cfg.IconCandidatesURL) != "" {
		if err := uploadIcons(result.Icons,
			iconCandidateQuery{
				URL:               cfg.IconCandidatesURL,
				buildTriggerToken: string(cfg.ResultSubmitAPIToken),
			}); err != nil {
			log.Warnf("Failed to submit icons, error: %s", err)
		}
	}

	if !platformsDetected {
		printDirTree()
		failf("No known platform detected")
	}
	log.Donef("Scan finished.")
}
