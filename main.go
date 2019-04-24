package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/log"
)

type config struct {
	ScanDirectory        string `env:"scan_dir,required"`
	OutputDirectory      string `env:"output_dir,required"`
	ResultSubmitURL      string `env:"scan_result_submit_url"`
	ResultSubmitAPIToken string `env:"scan_result_submit_url"`
}

func failf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

func main() {
	var cfg config
	if err := stepconf.Parse(&cfg); err != nil {
		failf("Invalid configuration: %s", err)
	}

	if !(runtime.GOOS == "darwin" || runtime.GOOS == "linux") {
		failf("Unsupported OS: %s", runtime.GOOS)
	}

	log.Infof("Creating scanner binary...")

	initialWD, err := os.Getwd()
	if err != nil {
		failf("Failed to get working directory.")
	}

	currentDirectory := os.Getenv("BITRISE_SOURCE_DIR")
	if err := os.Chdir(path.Join(currentDirectory, "go", "src", "github.com", "bitrise-io", "bitrise-init")); err != nil {
		failf("Failed to change directory.")
	}

	binaryDir, err := ioutil.TempDir("", "")
	if err != nil {
		failf("Failed to create temp dir.")
	}
	initBinary := path.Join(binaryDir, "init")

	buildCmd := command.New("go", "build", "-o", initBinary)
	buildCmd.AppendEnvs(fmt.Sprintf("GOPATH=%s", path.Join(currentDirectory, "go")))
	fmt.Println()
	log.Warnf("$ %s", buildCmd.PrintableCommandArgs())
	fmt.Println()

	if out, err := buildCmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		if errorutil.IsExitStatusError(err) {
			failf("Failed to build bitrise-init: %s", out)
		} else {
			failf("Failed to run command: %s", err)
		}
	}

	os.Chdir(initialWD)

	log.Donef("created at: %s", initBinary)

	log.Infof("Running scanner...")
	initCmd := command.New(initBinary, "--ci", "config",
		"--dir", cfg.ScanDirectory,
		"--output-dir", cfg.OutputDirectory,
		"--format", "json")
	fmt.Println()
	log.Warnf("$ %s", initCmd.PrintableCommandArgs())
	fmt.Println()
	exitCode, err := initCmd.SetStdout(os.Stdout).SetStderr(os.Stderr).RunAndReturnExitCode()
	if err != nil {
		failf("Failed to run command.")
	}

	scanResultPath := path.Join(cfg.OutputDirectory, "result.json")
	if _, err := os.Stat(scanResultPath); os.IsNotExist(err) {
		failf("No scan result found at %s", scanResultPath)
	} else if err != nil {
		failf("Failed to get file info, error: %s", err)
	}

	// Upload results
	if strings.TrimSpace(cfg.ResultSubmitURL) != "" {
		if strings.TrimSpace(cfg.ResultSubmitAPIToken) == "" {
			failf("Submit URL needs to be defined if and only if API Token is defined.")
		}

		log.Infof("Submitting results...")

		result, err := os.Open(scanResultPath)
		if err != nil {
			failf("Could not open results file.")
		}

		submitURL, err := url.Parse(cfg.ResultSubmitURL)
		if err != nil {
			failf("Could not parse submit URL.")
		}
		submitURL.Query().Set("api_token", url.QueryEscape(cfg.ResultSubmitAPIToken))

		http.Post(submitURL.String(), "application/json", result)
		log.Donef("submitted")
	}

	if exitCode == 0 {
		log.Donef("scan finished")
	} else {
		failf("scanner failed")
	}
}
