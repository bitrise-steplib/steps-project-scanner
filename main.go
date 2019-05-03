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
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
)

type config struct {
	ScanDirectory        string          `env:"scan_dir,required"`
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
			return fmt.Errorf("could not open results file")
		}

		submitURL, err := url.Parse(URL)
		if err != nil {
			return fmt.Errorf("could not parse submit URL")
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

func uploadIcons(iconsDir string, iconCandidateURL string, buildTriggerToken string) error {
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
	candidateURLs, err := getUploadURL(iconCandidateURL, buildTriggerToken, candidates)
	if err != nil {
		return fmt.Errorf("failed to get candidate target URLs, error: %s", err)
	}

	for _, candidateURL := range candidateURLs {
		err := uploadIcon(iconsDir, candidateURL)
		if err != nil {
			return fmt.Errorf("failed to upload icon, error: %s", err)
		}
	}

	log.Donef("submitted")
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
		if int64(len(data)) != iconCandidate.FileSize {
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

	log.TInfof(colorstring.Yellowf("scan dir: %s", cfg.ScanDirectory))
	log.TInfof(colorstring.Yellowf("output dir: %s", cfg.OutputDirectory))
	fmt.Println()

	searchDir, err := pathutil.AbsPath(cfg.ScanDirectory)
	if err != nil {
		panic(fmt.Errorf("failed to expand path (%s), error: %s", cfg.ScanDirectory, err))
	}

	outputDir, err := pathutil.AbsPath(cfg.OutputDirectory)
	if err != nil {
		panic(fmt.Errorf("failed to expand path (%s), error: %s", cfg.OutputDirectory, err))
	}
	if exist, err := pathutil.IsDirExists(outputDir); err != nil {
		panic(err)
	} else if !exist {
		if err := os.MkdirAll(outputDir, 0700); err != nil {
			panic(fmt.Errorf("failed to create (%s), error: %s", outputDir, err))
		}
	}

	_, scannerError := scanner.GenerateAndWriteResults(searchDir, outputDir, output.YAMLFormat)
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
		err := uploadIcons(path.Join(cfg.OutputDirectory, "icons"), cfg.IconCandidatesURL, string(cfg.ResultSubmitAPIToken))
		if err != nil {
			log.Warnf("Failed to submit icons, error: %s", err)
		}
	}

	if scannerError != nil {
		failf("Scanner failed.")
	}
	log.Donef("Scan finished.")
}
