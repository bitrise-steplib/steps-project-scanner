package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bitrise-io/go-steputils/step"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/steperror"
)

func Test_stepErrorFromGitClone(t *testing.T) {
	cloneErr := errors.New("couldn't find remote ref")

	tests := []struct {
		name string
		err  *steperror.Error
		want *step.Error
	}{
		{
			name: "without recommendations",
			err:  steperror.NewError("git-clone", "fetch_failed", cloneErr, "Fetching repository has failed"),
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "fetch_failed",
				ShortMsg: "Fetching repository has failed",
				Err:      cloneErr,
			},
		},
		{
			name: "with recommendations",
			err: steperror.NewErrorWithRecommendations("git-clone", "fetch_failed", cloneErr, "Fetching repository has failed", steperror.Recommendation{
				"BranchRecommendation": []string{"main", "develop"},
			}),
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "fetch_failed",
				ShortMsg: "Fetching repository has failed",
				Err:      cloneErr,
				Recommendations: step.Recommendation{
					"BranchRecommendation": []string{"main", "develop"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stepErrorFromGitClone(tt.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stepErrorFromGitClone() = %v, want %v", got, tt.want)
			}
		})
	}
}
