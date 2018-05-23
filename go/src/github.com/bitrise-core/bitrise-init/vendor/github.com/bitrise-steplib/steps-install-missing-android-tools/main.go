package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/bitrise-steplib/steps-install-missing-android-tools/androidcomponents"
	"github.com/bitrise-tools/go-steputils/input"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-android/sdk"
)

// ConfigsModel ...
type ConfigsModel struct {
	GradlewPath string
	AndroidHome string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		GradlewPath: os.Getenv("gradlew_path"),
		AndroidHome: os.Getenv("ANDROID_HOME"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")
	log.Printf("- GradlewPath: %s", configs.GradlewPath)
	log.Printf("- AndroidHome: %s", configs.AndroidHome)
}

func (configs ConfigsModel) validate() error {
	if err := input.ValidateIfPathExists(configs.GradlewPath); err != nil {
		return errors.New("Issue with input GradlewPath: " + err.Error())
	}

	if err := input.ValidateIfNotEmpty(configs.AndroidHome); err != nil {
		return errors.New("Issue with input AndroidHome: " + err.Error())
	}

	return nil
}

// -----------------------
// --- Functions
// -----------------------

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

// -----------------------
// --- Main
// -----------------------

func main() {
	// Input validation
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		failf(err.Error())
	}

	fmt.Println()
	log.Infof("Preparation")

	// Set executable permission for gradlew
	log.Printf("Set executable permission for gradlew")
	if err := os.Chmod(configs.GradlewPath, 0770); err != nil {
		failf("Failed to set executable permission for gradlew, error: %s", err)
	}

	// Initialize Android SDK
	log.Printf("Initialize Android SDK")
	androidSdk, err := sdk.New(configs.AndroidHome)
	if err != nil {
		failf("Failed to initialize Android SDK, error: %s", err)
	}

	androidcomponents.SetLogger(log.NewDefaultLogger(false))

	// Ensure android licences
	log.Printf("Ensure android licences")

	if err := androidcomponents.InstallLicences(androidSdk); err != nil {
		failf("Failed to ensure android licences, error: %s", err)
	}

	// Ensure required Android SDK components
	fmt.Println()
	log.Infof("Ensure required Android SDK components")

	if err := androidcomponents.Ensure(androidSdk, configs.GradlewPath); err != nil {
		failf("Failed to ensure android components, error: %s", err)
	}

	fmt.Println()
	log.Donef("Required SDK components are installed")
}
