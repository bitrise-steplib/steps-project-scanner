package main

import (
	"errors"
	"os"
	"testing"
)

func TestDataBuild(t *testing.T) {
	t.Log("it creates complex data")
	{
		key := "BITRISE_BUILD_SLUG"
		var value string
		var err error
		if value = os.Getenv(key); value != "" {
			err = os.Unsetenv(key)
			defer func() { err = os.Setenv(key, value) }()
		} else {
			defer func() { err = os.Unsetenv(key) }()
		}
		err = os.Setenv(key, "testSlug")
		if err != nil {
			t.Fatalf("test setup: os.Setenv() returned error: %s", err)
		}

		data := buildData(errors.New("testError"))
		if v, ok := data["error"]; !ok || v != "testError" {
			t.Fatalf("data := buildData(); data['error'] != 'testError'")
		}
		if v, ok := data["source"]; !ok || v != "scanner" {
			t.Fatalf("data := buildData(); data['source'] != 'scanner'")
		}
	}
}
