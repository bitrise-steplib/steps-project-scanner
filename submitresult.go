package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/step"
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

func (c *resultClient) uploadErrorResult(stepID string, err error) error {
	if stepError, ok := err.(*step.Error); ok && len(stepError.Recommendations) > 0 {
		return c.uploadResults(models.ScanResultModel{
			ScannerToErrorsWithRecomendations: map[string]models.ErrorsWithRecommendations{
				"general": {
					models.ErrorWithRecommendations{
						Error:           fmt.Sprintf("Error in step %s: %v", stepID, stepError.Err),
						Recommendations: stepError.Recommendations,
					},
				},
			},
		})
	}
	return c.uploadResults(models.ScanResultModel{
		ScannerToErrors: map[string]models.Errors{
			"general": {fmt.Sprintf("Error in step %s: %v", stepID, err)},
		},
	})
}

func (c *resultClient) uploadResults(result models.ScanResultModel) error {
	bytes, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal results, error: %v", err)
	}

	if err := retry.Times(1).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt != 0 {
			log.TWarnf("%d query attempt failed", attempt)
		}

		req, err := http.NewRequest(http.MethodPost, c.URL.String(), strings.NewReader(string(bytes)))
		if err != nil {
			return fmt.Errorf("faield to create http reques: %v", err)
		}
		req.Header.Add("Conent-Type", "application/json")

		reqDump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			log.TWarnf("failed to dump request: %v", err)
		}
		log.TDebugf("Request: %s", reqDump)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.TErrorf("failed to send request: url: %s, request: %s", c.URL.String(), reqDump)

			return fmt.Errorf("failed to submit results: %v", err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.TErrorf("failed to close response body: %v", err)
			}
		}()

		resp.Header.Del("Set-Cookie") // Removing sensitive info
		respDump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			log.TWarnf("failed to dump response: %s", err)
			respDump, err = httputil.DumpResponse(resp, false)
			if err != nil {
				log.TWarnf("failed to dump response: %s", err)
			}
		}

		if resp.StatusCode != http.StatusOK {
			log.TErrorf("Submit failed, url: %s request: %s, response: %s", c.URL.String(), reqDump, respDump)

			return fmt.Errorf("failed to submit results, status code: %d", resp.StatusCode)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
