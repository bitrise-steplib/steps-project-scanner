package reactnative

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

const scannerName = "react-native"

const (
	// workDirInputKey is a key of the working directory step input.
	workDirInputKey = "workdir"
)

const (
	isExpoBasedProjectInputTitle   = "Is this an [Expo](https://expo.dev)-based React Native project?"
	isExpoBasedProjectInputSummary = "Default deploy workflow runs builds on Expo Application Services (EAS) for Expo-based React Native projects.\nOtherwise native iOS and Android build steps will be used."
)

// Scanner implements the project scanner for plain React Native and Expo based projects.
type Scanner struct {
	searchDir      string
	iosScanner     *ios.Scanner
	androidScanner *android.Scanner

	hasTest         bool
	hasYarnLockFile bool
	packageJSONPth  string

	isExpoBased bool
}

// NewScanner creates a new scanner instance.
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name implements ScannerInterface.Name function.
func (Scanner) Name() string {
	return scannerName
}

func isExpoBasedProject(packageJSONPth string) (bool, error) {
	packages, err := utility.ParsePackagesJSON(packageJSONPth)
	if err != nil {
		return false, fmt.Errorf("failed to parse package json file (%s): %s", packageJSONPth, err)
	}

	if _, found := packages.Dependencies["expo"]; !found {
		return false, nil
	}

	expoAppConfigFiles := []string{"app.json", "app.config.js", "app.config.ts"}
	for _, base := range expoAppConfigFiles {
		expoAppConfigPth := filepath.Join(filepath.Dir(packageJSONPth), base)
		exist, err := pathutil.IsPathExists(expoAppConfigPth)
		if err != nil {
			return false, fmt.Errorf("failed to check if Expo app config exists at: %s: %s", expoAppConfigPth, err)
		}
		if exist {
			return true, nil
		}
	}

	return false, nil
}

// hasNativeProjects reports whether the project directory contains ios and android native project.
func hasNativeProjects(searchDir, projectDir string, iosScanner *ios.Scanner, androidScanner *android.Scanner) (bool, bool, error) {
	absProjectDir, err := pathutil.AbsPath(projectDir)
	if err != nil {
		return false, false, err
	}

	iosProjectDetected := false
	iosDir := filepath.Join(absProjectDir, "ios")
	if exist, err := pathutil.IsDirExists(iosDir); err != nil {
		return false, false, err
	} else if exist {
		if detected, err := iosScanner.DetectPlatform(searchDir); err != nil {
			return false, false, err
		} else if detected {
			iosProjectDetected = true
		}
	}

	androidProjectDetected := false
	androidDir := filepath.Join(absProjectDir, "android")
	if exist, err := pathutil.IsDirExists(androidDir); err != nil {
		return false, false, err
	} else if exist {
		if detected, err := androidScanner.DetectPlatform(searchDir); err != nil {
			return false, false, err
		} else if detected {
			androidProjectDetected = true
		}
	}

	return iosProjectDetected, androidProjectDetected, nil
}

// DetectPlatform implements ScannerInterface.DetectPlatform function.
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	scanner.searchDir = searchDir

	log.TInfof("Collect package.json files")

	packageJSONPths, err := CollectPackageJSONFiles(searchDir)
	if err != nil {
		return false, err
	}

	log.TPrintf("%d package.json file detected", len(packageJSONPths))
	log.TPrintf("Filter relevant package.json files")

	isExpoBased := false
	var packageFile string

	for _, packageJSONPth := range packageJSONPths {
		log.TPrintf("Checking: %s", packageJSONPth)

		expoBased, err := isExpoBasedProject(packageJSONPth)
		if err != nil {
			log.TWarnf("failed to determine if project is Expo based: %s", err)
		} else if expoBased {
			log.TPrintf("Project uses expo: %v", expoBased)
			isExpoBased = true
			packageFile = packageJSONPth
			// TODO: This break drops other package.json files
			break
		}

		log.TPrintf("Project uses expo: %v", expoBased)

		if scanner.iosScanner == nil {
			scanner.iosScanner = ios.NewScanner()
			scanner.iosScanner.ExcludeAppIcon = true
		}
		if scanner.androidScanner == nil {
			scanner.androidScanner = android.NewScanner()
			scanner.androidScanner.ExcludeAppIcon = true
		}

		projectDir := filepath.Dir(packageJSONPth)
		ios, android, err := hasNativeProjects(searchDir, projectDir, scanner.iosScanner, scanner.androidScanner)
		if err != nil {
			log.TWarnf("failed to check native projects: %s", err)
		} else {
			log.TPrintf("Found native ios project: %v", ios)
			log.TPrintf("Found native android project: %v", android)
		}

		if ios || android {
			// Treating the project as a plain React Native project
			packageFile = packageJSONPth
			break
		}
	}

	if packageFile == "" {
		return false, nil
	}

	scanner.isExpoBased = isExpoBased
	scanner.packageJSONPth = packageFile

	// determine Js dependency manager
	if scanner.hasYarnLockFile, err = containsYarnLock(filepath.Dir(scanner.packageJSONPth)); err != nil {
		return false, err
	}
	log.TPrintf("Js dependency manager for %s is yarn: %t", scanner.packageJSONPth, scanner.hasYarnLockFile)

	packages, err := utility.ParsePackagesJSON(scanner.packageJSONPth)
	if err != nil {
		return false, err
	}

	if _, found := packages.Scripts["test"]; found {
		scanner.hasTest = true
	}
	log.TPrintf("Test script found in package.json: %v", scanner.hasTest)

	return true, nil
}

// Options implements ScannerInterface.Options function.
func (scanner *Scanner) Options() (options models.OptionNode, warnings models.Warnings, icons models.Icons, err error) {
	if scanner.isExpoBased {
		options, warnings, err = scanner.expoOptions()
	} else {
		options, warnings, err = scanner.options()
	}

	return
}

// Configs implements ScannerInterface.Configs function.
func (scanner *Scanner) Configs(isPrivateRepo bool) (models.BitriseConfigMap, error) {
	if scanner.isExpoBased {
		return scanner.expoConfigs(isPrivateRepo)
	}

	return scanner.configs(isPrivateRepo)
}

// DefaultOptions implements ScannerInterface.DefaultOptions function.
func (scanner *Scanner) DefaultOptions() models.OptionNode {
	expoOption := models.NewOption(isExpoBasedProjectInputTitle, isExpoBasedProjectInputSummary, "", models.TypeSelector)

	expoDefaultOptions := scanner.expoDefaultOptions()
	expoOption.AddOption("yes", &expoDefaultOptions)

	defaultOptions := scanner.defaultOptions()
	expoOption.AddOption("no", &defaultOptions)

	return *expoOption
}

// DefaultConfigs implements ScannerInterface.DefaultConfigs function.
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	configs, err := scanner.defaultConfigs()
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	for k, v := range configs {
		configMap[k] = v
	}

	expoConfigs, err := scanner.expoDefaultConfigs()
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	for k, v := range expoConfigs {
		configMap[k] = v
	}

	return configMap, nil
}

// ExcludedScannerNames implements ScannerInterface.ExcludedScannerNames function.
func (Scanner) ExcludedScannerNames() []string {
	return []string{
		string(ios.XcodeProjectTypeIOS),
		string(ios.XcodeProjectTypeMacOS),
		android.ScannerName,
	}
}
