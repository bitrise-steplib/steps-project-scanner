package main

import (
	"bytes"
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

type iconCandidateQuery struct {
	URL               string
	buildTriggerToken string
}

func uploadIcons(icons []models.Icon, query iconCandidateQuery) error {
	log.Infof("Submitting app icons...")

	nameToPath := map[string]string{}
	for _, icon := range icons {
		nameToPath[icon.Filename] = icon.Path
	}

	var candidates []appIconCandidate
	for name, path := range nameToPath {
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Warnf("Failed to get file (%s) info, error: ", path, err)
			continue
		}
		if !fileInfo.IsDir() && fileInfo.Size() != 0 {
			// Using the generated name instead of the filesystem name as it is unique
			candidates = append(candidates, appIconCandidate{
				FileName: name,
				FileSize: fileInfo.Size(),
			})
		}
	}

	candidateURLs, err := getUploadURL(query, candidates)
	if err != nil {
		return fmt.Errorf("failed to get candidate target URLs, error: %s", err)
	}

	for _, candidateURL := range candidateURLs {
		if err := uploadIcon(nameToPath[candidateURL.FileName], candidateURL); err != nil {
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

func uploadIcon(filePath string, iconCandidate appIconCandidateURL) error {
	if err := retry.Times(3).Wait(5 * time.Second).Try(func(attemp uint) error {
		if attemp != 0 {
			log.Warnf("%d query attemp failed", attemp)
		}

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

	result, scannerError := scanner.GenerateAndWriteResults(searchDir, outputDir, output.JSONFormat)
	if scannerError != nil {
		log.Warnf("Scanner error: %s", scannerError)
	}

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

	if scannerError != nil {
		failf("Scanner failed.")
	}
	log.Donef("Scan finished.")
}
