package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
)

type resultClient struct {
	URL *url.URL
}

func newResultClient(resultSubmitURL string, resultSubmitAPIToken stepconf.Secret) (*resultClient, error) {
	submitURL, err := url.Parse(resultSubmitURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse submit URL, error: %s", err)
	}

	q := submitURL.Query()
	q.Add("api_token", url.QueryEscape(string(resultSubmitAPIToken)))
	submitURL.RawQuery = q.Encode()

	return &resultClient{
		URL: submitURL,
	}, nil
}

func (c *resultClient) uploadErrorResult(stepID, tag string, err error, shortMsg string) error {
	result := models.ScanResultModel{
		ScannerToErrors: map[string]models.Errors{
			"general": models.Errors{fmt.Sprintf("Error in step %s: %v", err)},
		},
	}

	return c.uploadResults(result)
}

func (c *resultClient) uploadResults(result models.ScanResultModel) error {
	bytes, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal results, error: %s", err)
	}

	if err := retry.Times(1).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt != 0 {
			log.TWarnf("%d query attempt failed", attempt)
		}

		resp, err := http.Post(c.URL.String(), "application/json", strings.NewReader(string(bytes)))
		if err != nil {
			return fmt.Errorf("failed to submit results, error: %s", err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.TErrorf("failed to close response body, error: %s", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			log.TErrorf("Submit failed, status code: %d, headers: %s", resp.StatusCode, resp.Header)
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
