package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
)

type appIconCandidateURL struct {
	FileName  string `json:"filename"`
	FileSize  int64  `json:"filesize"`
	UploadURL string `json:"upload_url,omitempty"`
}

type iconCandidateQuery struct {
	URL               string
	buildTriggerToken string
}

func uploadIcons(icons []models.Icon, query iconCandidateQuery) error {
	log.TInfof("Validating app icons.")
	icons = filterValidIcons(icons)

	log.TInfof("Submitting app icons...")
	nameToPath := map[string]string{}
	for _, icon := range icons {
		nameToPath[icon.Filename] = icon.Path
	}

	var candidates []appIconCandidateURL
	for name, path := range nameToPath {
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.TWarnf("Failed to get file (%s) info: %s", path, err)
			continue
		}
		if !fileInfo.IsDir() && fileInfo.Size() != 0 {
			// Using the generated name instead of the filesystem name as it is unique
			candidates = append(candidates, appIconCandidateURL{
				FileName: name,
				FileSize: fileInfo.Size(),
			})
		}
	}

	if len(candidates) == 0 {
		log.Warnf("No valid icons specified.")
		return nil
	}

	candidateURLs, err := getUploadURLs(query, candidates)
	if err != nil {
		return fmt.Errorf("failed to get candidate target URLs: %w", err)
	}

	for _, candidateURL := range candidateURLs {
		if err := uploadIcon(nameToPath[candidateURL.FileName], candidateURL); err != nil {
			return fmt.Errorf("failed to upload icon: %w", err)
		}
	}

	log.TDonef("submitted")
	return nil
}

func getUploadURLs(query iconCandidateQuery, appIcons []appIconCandidateURL) ([]appIconCandidateURL, error) {
	if query.URL == "" {
		return nil, fmt.Errorf("query URL is empty")
	}
	if query.buildTriggerToken == "" {
		return nil, fmt.Errorf("no token specified for URL: %s", query.URL)
	}
	if len(appIcons) == 0 {
		return nil, fmt.Errorf("no icons to submit")
	}

	data, err := json.Marshal(appIcons)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json: %w", err)
	}

	var uploadURLs []appIconCandidateURL
	if err := retry.Times(3).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt > 0 {
			log.TWarnf("%d query attempt failed", attempt)
		}

		request, err := http.NewRequest(http.MethodPost, query.URL, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		request.Header.Set("Authorization", fmt.Sprintf("token %s", query.buildTriggerToken))
		request.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			return fmt.Errorf("failed to submit: %w", err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.TErrorf("Failed to close response body: %w", err)
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("invalid status code: %d, headers: %s, body: %s", resp.StatusCode, resp.Header, body)
		}

		var decoded map[string][]appIconCandidateURL
		if err = json.Unmarshal(body, &decoded); err != nil {
			return fmt.Errorf("failed to unmarshal resoponse body: %w", err)
		}
		URLs, found := decoded["data"]
		if !found {
			return fmt.Errorf("no data key found in response json")
		}
		uploadURLs = URLs
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to upload: %w", err)
	}
	return uploadURLs, nil
}

func uploadIcon(filePath string, iconCandidate appIconCandidateURL) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file (%s): %w", filePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.TWarnf("failed to close file: %w", err)
		}
	}()

	// If a byte array is passed to http.NewRequest, the Content-lenght header is set to its lenght.
	// That does not seem apply to a stream (as it has no defined lenght).
	// The Content-lenght header is signed by S3, so has to match to the filesize sent
	// in the getUploadURL() function.
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("can not read file: %w", err)
	}
	if int64(len(data)) != iconCandidate.FileSize {
		return fmt.Errorf("array lenght deos not match to file size reported to the API, "+
			"actual: %d, expected: %d",
			len(data), iconCandidate.FileSize)
	}

	if iconCandidate.UploadURL == "" {
		return fmt.Errorf("target URL is empty, %v+", iconCandidate)
	}

	if err := retry.Times(3).Wait(5 * time.Second).Try(func(attemp uint) error {
		if attemp != 0 {
			log.TWarnf("%d query attemp failed", attemp)
		}

		request, err := http.NewRequest(http.MethodPut, iconCandidate.UploadURL, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		request.Header.Add("Content-Type", "image/png")

		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			return fmt.Errorf("failed to submit: %w", err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.TErrorf("Failed to close response body: %w", err)
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read respnse body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("invalid status code: %d, headers: %s, body: %s", resp.StatusCode, resp.Header, body)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}
	return nil
}

func filterValidIcons(icons []models.Icon) []models.Icon {
	var validIcons []models.Icon
	for _, icon := range icons {
		if err := validateIcon(icon.Path); err != nil {
			log.TWarnf("Invalid icon file (%v+): %w", icon, err)
			continue
		}
		validIcons = append(validIcons, icon)
	}
	return validIcons
}

func validateIcon(iconPath string) error {
	const maxImageSize = 1024
	const maxFileSize = 2 * 1e6
	file, err := os.Open(iconPath)
	if err != nil {
		return err
	}

	if fileInfo, err := file.Stat(); err != nil {
		return fmt.Errorf("failed to get icon file stats: %w", err)
	} else if fileInfo.Size() > maxFileSize {
		return fmt.Errorf("icon file larger than 2 MB")
	}

	config, err := png.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("invalid png file: %w", err)
	}

	if config.Width > maxImageSize || config.Height > maxImageSize {
		return fmt.Errorf("image dimensions larger than %d", maxImageSize)
	}
	return nil
}
