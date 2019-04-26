package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
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

type appIconCandidate struct {
	FileName string `json:"filename"`
	FileSize int    `json:"filesize"`
}

type appIconCandidateURL struct {
	FileName  string `json:"filename"`
	FileSize  int    `json:"filesize"`
	UploadURL string `json:"upload_url"`
}

func uploadResults(URL string, token string, scanResultPath string) error {
	if err := retry.Times(3).Wait(5 * time.Second).Try(func(attempt uint) error {
		if strings.TrimSpace(token) == "" {
			return fmt.Errorf("submit URL needs to be defined if and only if API Token is defined")
		}

		result, err := os.Open(scanResultPath)
		if err != nil {
			return fmt.Errorf("could not open results file")
		}

		submitURL, err := url.Parse(URL)
		if err != nil {
			return fmt.Errorf("could not parse submit URL")
		}
		submitURL.Query().Set("api_token", url.QueryEscape(token))

		response, err := http.Post(submitURL.String(), "application/json", result)

		defer func() {
			if err := response.Body.Close(); err != nil {
				log.Errorf("failed to close response body, error: %s", err)
			}
		}()

		if err != nil {
			return fmt.Errorf("failed to submit results, error: %s", err)
		}
		if !(response.StatusCode != http.StatusOK) {
			return fmt.Errorf("failed to submit results, status code: %d", response.StatusCode)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to submit, error: %s", err)
	}
	return nil
}

func getUploadURL(url string, buildTriggerToken string, appIcons []appIconCandidate) ([]appIconCandidateURL, error) {
	var uploadURLs []appIconCandidateURL
	if err := retry.Times(3).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("%d query attempt failed", attempt)
		}

		data, err := json.Marshal(appIcons)
		if err != nil {
			return fmt.Errorf("failed to marshal json")
		}

		request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create request")
		}
		request.Header.Set("Authorization", fmt.Sprintf("token %s", buildTriggerToken))
		request.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			return fmt.Errorf("failed to submit, error: %s", err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Errorf("Failed to close response body, error: %s", err)
			}
		}()

		if err != nil {
			return fmt.Errorf("failed to submit, err: %s", err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read respnse body, error: %s", err)
		}

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("invalid status code: %d, headers: %s, body: %s", resp.StatusCode, resp.Header, body)
		}

		decoded := map[string][]appIconCandidateURL{
			"data": uploadURLs,
		}

		err = json.Unmarshal(body, &decoded)
		if err != nil {
			return fmt.Errorf("failed to unmarshal resoponse bodys")
		}
		uploadURLs = decoded["data"]
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to upload, error: %s", err)
	}
	return uploadURLs, nil
}

func uploadIcon(basePath string, iconCandidate appIconCandidateURL) error {
	if err := retry.Times(3).Wait(5 * time.Second).Try(func(attemp uint) error {
		if attemp != 0 {
			log.Warnf("%d query attemp failed", attemp)
		}

		file, err := os.Open(path.Join(basePath, iconCandidate.FileName))
		if err != nil {
			return fmt.Errorf("failed to open file")
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Warnf("failed to close file")
			}
		}()

		data, err := ioutil.ReadAll(file)
		if err != nil {
			return fmt.Errorf("can not read file")
		}
		if len(data) != iconCandidate.FileSize {
			return fmt.Errorf("content-lenght has to match signed URL")
		}

		request, err := http.NewRequest(http.MethodPut, iconCandidate.UploadURL, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create request")
		}

		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			return fmt.Errorf("failed to submit, error: %s", err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Errorf("Failed to close response body, error: %s", err)
			}
		}()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read respnse body, error: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Invalid status code: %d, headers: %s, body: %s", resp.StatusCode, resp.Header, body)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to upload, error: %s", err)
	}
	return nil
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
		log.Infof("Submitting results...")
		err := uploadResults(cfg.ResultSubmitURL, cfg.ResultSubmitAPIToken, scanResultPath)
		if err != nil {
			failf("Failed to submit results, error: %s", err)
		}
		log.Donef("submitted")
	}

	// Upload icons
	if exitCode == 0 {
		log.Donef("scan finished")
	} else {
		failf("scanner failed")
	}
}
