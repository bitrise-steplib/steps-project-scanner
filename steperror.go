package main

import (
	"github.com/bitrise-io/go-steputils/step"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/steperror"
)

func newStepError(tag string, err error, shortMsg string) *step.Error {
	return step.NewError("project-scanner", tag, err, shortMsg)
}

func stepErrorFromGitClone(err *steperror.Error) *step.Error {
	return &step.Error{
		StepID:          err.StepID,
		Tag:             err.Tag,
		ShortMsg:        err.ShortMsg,
		Err:             err.Err,
		Recommendations: step.Recommendation(err.Recommendations),
	}
}
