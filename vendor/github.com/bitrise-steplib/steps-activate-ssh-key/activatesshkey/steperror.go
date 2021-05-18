package activatesshkey

import (
	"github.com/bitrise-io/go-steputils/step"
)

func newStepError(tag string, err error, shortMsg string) *step.Error {
	return step.NewError("activate-ssh-key", tag, err, shortMsg)
}
