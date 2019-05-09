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

	"github.com/bitrise-io/bitrise-init/output"
	"github.com/bitrise-io/bitrise-init/scanner"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
)

type config struct {
	ScanDirectory        string          `env:"scan_dir,dir"`
	OutputDirectory      string          `env:"output_dir,required"`
	ResultSubmitURL      string          `env:"scan_result_submit_url"`
	ResultSubmitAPIToken stepconf.Secret `env:"scan_result_submit_api_token"`
	IconCandidatesURL    string          `env:"icon_candidates_url"`
}

type appIconCandidate struct {
	FileName string `json:"filename"`
	FileSize int64  `json:"filesize"`
}

type appIconCandidateURL struct {
	FileName  string `json:"filename"`
	FileSize  int64  `json:"filesize"`
	UploadURL string `json:"upload_url"`
}

func failf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

func uploadResults(URL string, token string, scanResultPath string) error {
	if err := retry.Times(1).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt != 0 {
			log.Warnf("%d query attempt failed", attempt)
		}

		if strings.TrimSpace(token) == "" {
			log.Warnf("Build trigger token is empty.")
		}

		result, err := os.Open(scanResultPath)
		if err != nil {
			return fmt.Errorf("could not open results file, error: %s", err)
		}

		submitURL, err := url.Parse(URL)
		if err != nil {
			return fmt.Errorf("could not parse submit URL, error: %s", err)
		}

		q := submitURL.Query()
		q.Add("api_token", url.QueryEscape(token))
		submitURL.RawQuery = q.Encode()

		resp, err := http.Post(submitURL.String(), "application/json", result)
		if err != nil {
			return fmt.Errorf("failed to submit results, error: %s", err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Errorf("failed to close response body, error: %s", err)
			}
		}()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read respnse body, error: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to submit results, status code: %d, headers: %s, body: %s", resp.StatusCode, resp.Header, body)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to submit, error: %s", err)
	}
	return nil
}

type iconCandidateQuery struct {
	URL               string
	buildTriggerToken string
}

func uploadIcons(iconsDir string, query iconCandidateQuery) error {
	entries, err := ioutil.ReadDir(iconsDir)
	if err != nil && !os.IsNotExist(err) {
		log.Warnf("failed to read app icons, error: %s", err)
	}
	if len(entries) == 0 {
		return nil
	}

	log.Infof("Submitting app icons...")

	var candidates []appIconCandidate
	for _, fileInfo := range entries {
		if !fileInfo.IsDir() && fileInfo.Size() != 0 {
			candidates = append(candidates, appIconCandidate{
				FileName: fileInfo.Name(),
				FileSize: fileInfo.Size(),
			})
		}
	}
	candidateURLs, err := getUploadURL(query, candidates)
	if err != nil {
		return fmt.Errorf("failed to get candidate target URLs, error: %s", err)
	}

	for _, candidateURL := range candidateURLs {
		if err := uploadIcon(iconsDir, candidateURL); err != nil {
			return fmt.Errorf("failed to upload icon, error: %s", err)
		}
	}

	log.Donef("submitted")
	return nil
}

func getUploadURL(query iconCandidateQuery, appIcons []appIconCandidate) ([]appIconCandidateURL, error) {
	var uploadURLs []appIconCandidateURL
	if err := retry.Times(3).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("%d query attempt failed", attempt)
		}

		data, err := json.Marshal(appIcons)
		if err != nil {
			return fmt.Errorf("failed to marshal json, error: %s", err)
		}

		request, err := http.NewRequest(http.MethodPost, query.URL, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create request")
		}
		request.Header.Set("Authorization", fmt.Sprintf("token %s", query.buildTriggerToken))
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

		if err = json.Unmarshal(body, &decoded); err != nil {
			return fmt.Errorf("failed to unmarshal resoponse body, error: %s", err)
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

		filePath := path.Join(basePath, iconCandidate.FileName)
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file (%s), error: %s", filePath, err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Warnf("failed to close file, error: %s", err)
			}
		}()

		// If a byte array is passed to http.NewRequest, the Content-lenght header is set to its lenght.
		// That does not seem apply to a stream (as it has no defined lenght).
		// The Content-lenght header is signed by S3, so has to match to the filesize sent
		// in the getUploadURL() function.
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return fmt.Errorf("can not read file, error: %s", err)
		}
		if int64(len(data)) != iconCandidate.FileSize {
			return fmt.Errorf("Array lenght deos not match to file size reported to the API, "+
				"actual: %d, expected: %d",
				len(data), iconCandidate.FileSize)
		}

		request, err := http.NewRequest(http.MethodPut, iconCandidate.UploadURL, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create request, error: %s", err)
		}

		request.Header.Add("Content-Type", "image/png")

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
			return fmt.Errorf("invalid status code: %d, headers: %s, body: %s", resp.StatusCode, resp.Header, body)
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
	stepconf.Print(cfg)

	if !(runtime.GOOS == "darwin" || runtime.GOOS == "linux") {
		failf("Unsupported OS: %s", runtime.GOOS)
	}

	searchDir, err := pathutil.AbsPath(cfg.ScanDirectory)
	if err != nil {
		failf("failed to expand path (%s), error: %s", cfg.ScanDirectory, err)
	}

	outputDir, err := pathutil.AbsPath(cfg.OutputDirectory)
	if err != nil {
		failf("failed to expand path (%s), error: %s", cfg.OutputDirectory, err)
	}
	if exist, err := pathutil.IsDirExists(outputDir); err != nil {
		failf("failed to check if dir (%s) exists, error: %s", outputDir, err)
	} else if !exist {
		if err := os.MkdirAll(outputDir, 0700); err != nil {
			failf("failed to create dir (%s), error: %s", outputDir, err)
		}
	}

	_, scannerError := scanner.GenerateAndWriteResults(searchDir, outputDir, output.JSONFormat)
	if scannerError != nil {
		log.Warnf("Scanner error: %s", scannerError)
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
		err := uploadResults(cfg.ResultSubmitURL, string(cfg.ResultSubmitAPIToken), scanResultPath)
		if err != nil {
			failf("Failed to submit results, error: %s", err)
		}
		log.Donef("Submitted.")
	}

	// Upload icons
	if strings.TrimSpace(cfg.IconCandidatesURL) != "" {
		if err := uploadIcons(path.Join(cfg.OutputDirectory, "icons"),
			iconCandidateQuery{
				URL:               cfg.IconCandidatesURL,
				buildTriggerToken: string(cfg.ResultSubmitAPIToken),
			}); err != nil {
			log.Warnf("Failed to submit icons, error: %s", err)
		}
	}

	if scannerError != nil {
		failf("Scanner failed.")
	}
	log.Donef("Scan finished.")
}
