package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/bitrise-io/bitrise-init/errormapper"
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

func buildErrorScanResultModel(stepID string, err error) models.ScanResultModel {
	var errWithRec models.ErrorWithRecommendations
	// It's a stepError
	if stepError, ok := err.(*step.Error); ok {
		rec := stepError.Recommendations

		// If it doesn't have recommendations, create one
		if rec == nil {
			rec = step.Recommendation{}
		}

		// Check for DetailedError field, if not present, fill it with generic DetailedError
		if rec[errormapper.DetailedErrorRecKey] == nil {
			rec[errormapper.DetailedErrorRecKey] = errormapper.DetailedError{
				Title:       stepError.Err.Error(),
				Description: "For more information, please see the log.",
			}
		}

		// Create the error with recommendation model
		errWithRec = models.ErrorWithRecommendations{
			Error:           fmt.Sprintf("Error in step %s: %v", stepID, stepError.Err),
			Recommendations: rec,
		}
	} else {
		// It's a standard error, fallback to the generic DetailedError
		errWithRec = models.ErrorWithRecommendations{
			Error: fmt.Sprintf("Error in step %s: %v", stepID, err),
			Recommendations: step.Recommendation{
				errormapper.DetailedErrorRecKey: errormapper.DetailedError{
					Title:       err.Error(),
					Description: "For more information, please see the log.",
				},
			},
		}
	}

	return models.ScanResultModel{
		ScannerToErrorsWithRecommendations: map[string]models.ErrorsWithRecommendations{
			"general": {
				errWithRec,
			},
		},
	}
}

func (c *resultClient) uploadErrorResult(stepID string, err error) error {
	result := buildErrorScanResultModel(stepID, err)
	return c.uploadResults(result)
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
