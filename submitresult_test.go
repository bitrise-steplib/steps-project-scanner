package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bitrise-io/bitrise-init/errormapper"
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/step"
)

func Test_resultClient_buildErrorScanResultModel(t *testing.T) {
	type args struct {
		stepID string
		err    error
	}
	tests := []struct {
		name string
		args args
		want models.ScanResultModel
	}{
		{
			name: "buildErrorScanResultModel (standard error)",
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
								"DetailedError": errormapper.DetailedError{
									Title:       "standar error",
									Description: "For more information, please see the log.",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "buildErrorScanResultModel (with step.Error without recommendations)",
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
								"DetailedError": errormapper.DetailedError{
									Title:       "step error without recommendations",
									Description: "For more information, please see the log.",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "buildErrorScanResultModel (with step.Error with recommendations without DetailedError)",
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
								"DetailedError": errormapper.DetailedError{
									Title:       "with step.Error with recommendations without DetailedError",
									Description: "For more information, please see the log.",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "buildErrorScanResultModel (with step.Error with recommendations with DetailedError)",
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
							"Description": "Please choose another branch and try again.",
							"Title":       "We couldn't find the branch 'mastre'.",
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
									"Description": "Please choose another branch and try again.",
									"Title":       "We couldn't find the branch 'mastre'.",
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
			if got := buildErrorScanResultModel(tt.args.stepID, tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resultClient.buildErrorScanResultModel() = %v, want %v", got, tt.want)
			}
		})
	}
}
