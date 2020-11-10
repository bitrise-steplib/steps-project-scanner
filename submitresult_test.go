package main

import (
	"errors"
	"net/url"
	"reflect"
	"testing"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/step"
)

func Test_resultClient_buildErrorScanResultModel(t *testing.T) {
	type fields struct {
		URL *url.URL
	}
	type args struct {
		stepID string
		err    error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   models.ScanResultModel
	}{
		{
			name: "buildErrorScanResultModel (standard error)",
			fields: fields{
				URL: &url.URL{},
			},
			args: args{
				stepID: "git-clone",
				err:    errors.New("standar error"),
			},
			want: models.ScanResultModel{
				ScannerToErrorsWithRecommendations: map[string]models.ErrorsWithRecommendations{
					"general": {
						models.ErrorWithRecommendations{
							Error: "Error in step git-clone: standar error",
							Recommendations: step.Recommendation{
								"DetailedError": map[string]string{
									"Title":       "standar error",
									"Description": "For more information, please see the log.",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "buildErrorScanResultModel (with step.Error without recommendations)",
			fields: fields{
				URL: &url.URL{},
			},
			args: args{
				stepID: "git-clone",
				err: step.NewError(
					"git-clone",
					"tag",
					errors.New("step error without recommendations"),
					"shortMsg: step error without recommendations"),
			},
			want: models.ScanResultModel{
				ScannerToErrorsWithRecommendations: map[string]models.ErrorsWithRecommendations{
					"general": {
						models.ErrorWithRecommendations{
							Error: "Error in step git-clone: step error without recommendations",
							Recommendations: step.Recommendation{
								"DetailedError": map[string]string{
									"Title":       "step error without recommendations",
									"Description": "For more information, please see the log.",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "buildErrorScanResultModel (with step.Error with recommendations without DetailedError)",
			fields: fields{
				URL: &url.URL{},
			},
			args: args{
				stepID: "git-clone",
				err: step.NewErrorWithRecommendations(
					"git-clone",
					"tag",
					errors.New("with step.Error with recommendations without DetailedError"),
					"shortMsg: with step.Error with recommendations without DetailedError",
					step.Recommendation{
						"BranchRecommendation": []string{"master", "feature1"},
					}),
			},
			want: models.ScanResultModel{
				ScannerToErrorsWithRecommendations: map[string]models.ErrorsWithRecommendations{
					"general": {
						models.ErrorWithRecommendations{
							Error: "Error in step git-clone: with step.Error with recommendations without DetailedError",
							Recommendations: step.Recommendation{
								"BranchRecommendation": []string{"master", "feature1"},
								"DetailedError": map[string]string{
									"Title":       "with step.Error with recommendations without DetailedError",
									"Description": "For more information, please see the log.",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "buildErrorScanResultModel (with step.Error with recommendations with DetailedError)",
			fields: fields{
				URL: &url.URL{},
			},
			args: args{
				stepID: "git-clone",
				err: step.NewErrorWithRecommendations(
					"git-clone",
					"tag",
					errors.New("with step.Error with recommendations with DetailedError"),
					"shortMsg: with step.Error with recommendations with DetailedError",
					step.Recommendation{
						"BranchRecommendation": []string{"master", "feature1"},
						"DetailedError": map[string]string{
							"Title":       "We couldn't find the branch 'mastre'.",
							"Description": "Please choose another branch and try again.",
						},
					}),
			},
			want: models.ScanResultModel{
				ScannerToErrorsWithRecommendations: map[string]models.ErrorsWithRecommendations{
					"general": {
						models.ErrorWithRecommendations{
							Error: "Error in step git-clone: with step.Error with recommendations with DetailedError",
							Recommendations: step.Recommendation{
								"BranchRecommendation": []string{"master", "feature1"},
								"DetailedError": map[string]string{
									"Title":       "We couldn't find the branch 'mastre'.",
									"Description": "Please choose another branch and try again.",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &resultClient{
				URL: tt.fields.URL,
			}
			if got := c.buildErrorScanResultModel(tt.args.stepID, tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resultClient.buildErrorScanResultModel() = %v, want %v", got, tt.want)
			}
		})
	}
}
