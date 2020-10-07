package main

import "fmt"

// stepError is an error occuring top level in a step
type stepError struct {
	StepID, Tag, ShortMsg string
	Err                   error
}

func (e *stepError) Error() string {
	return fmt.Sprintf("Error in step %s: %s, %v", e.ShortMsg, e.Err.Error())
}
