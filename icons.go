package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
)

type appIconCandidate struct {
	FileName string `json:"filename"`
	FileSize int64  `json:"filesize"`
}

type appIconCandidateURL struct {
	FileName  string `json:"filename"`
	FileSize  int64  `json:"filesize"`
	UploadURL string `json:"upload_url"`
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
		if err := validateIcon(path); err != nil {
			log.Warnf("Invalid icon file, error: %s", err)
			continue
		}
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
			return fmt.Errorf("array lenght deos not match to file size reported to the API, "+
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

func validateIcon(iconPath string) error {
	const maxImageSize = 1024
	const maxFileSize = 2 * 1e6
	file, err := os.Open(iconPath)
	if err != nil {
		return err
	}

	if fileInfo, err := file.Stat(); err != nil {
		return fmt.Errorf("failed to get icon file stats, error: %s", err)
	} else if fileInfo.Size() > maxFileSize {
		return fmt.Errorf("icon file too large")
	}

	config, err := png.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("invalid png file, error: %s", err)
	}

	if config.Width > maxImageSize || config.Height > maxImageSize {
		return fmt.Errorf("image dimensions larger than %d", maxImageSize)
	}
	return nil
}
