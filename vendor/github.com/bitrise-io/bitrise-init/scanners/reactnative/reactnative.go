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
	isExpoCLIInputTitle   = "Was the project created using the Expo CLI?"
	isExpoCLIInputSummary = "If your React Native app was created with the Expo CLI, Bitrise will automatically insert the **Expo Eject** Step to your Workflows."
)

// Scanner implements the project scanner for plain React Native and Expo based projects.
type Scanner struct {
	searchDir      string
	iosScanner     *ios.Scanner
	androidScanner *android.Scanner

	hasTest         bool
	hasYarnLockFile bool
	packageJSONPth  string

	usesExpo    bool
	usesExpoKit bool
}

// NewScanner creates a new scanner instance.
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name implements ScannerInterface.Name function.
func (Scanner) Name() string {
	return scannerName
}

// isExpoProject reports whether a project is Expo based.
func isExpoProject(packageJSONPth string) (bool, error) {
	packages, err := utility.ParsePackagesJSON(packageJSONPth)
	if err != nil {
		return false, fmt.Errorf("failed to parse package json file (%s): %s", packageJSONPth, err)
	}

	if _, found := packages.Dependencies["expo"]; !found {
		return false, nil
	}

	// app.json file is a required part of an expo projects and shoulb be placed next to the root package.json file
	appJSONPth := filepath.Join(filepath.Dir(packageJSONPth), "app.json")
	exist, err := pathutil.IsPathExists(appJSONPth)
	if err != nil {
		return false, fmt.Errorf("failed to check if app.json file (%s) exist: %s", appJSONPth, err)
	}
	return exist, nil
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

	usesExpo := false
	var packageFile string

	for _, packageJSONPth := range packageJSONPths {
		log.TPrintf("Checking: %s", packageJSONPth)

		expo, err := isExpoProject(packageJSONPth)
		if err != nil {
			log.TWarnf("failed to check if project uses Expo: %s", err)
		} else {
			log.TPrintf("Project uses Expo: %v", expo)
		}

		if expo {
			usesExpo = true
			packageFile = packageJSONPth
			break
		}

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
			log.TPrintf("Has native ios project: %v", ios)
			log.TPrintf("Has native android project: %v", android)
		}

		if ios || android {
			packageFile = packageJSONPth
			break
		}
	}

	if packageFile == "" {
		return false, nil
	}

	scanner.usesExpo = usesExpo
	scanner.packageJSONPth = packageFile

	// determine Js dependency manager
	if scanner.hasYarnLockFile, err = containsYarnLock(filepath.Dir(scanner.packageJSONPth)); err != nil {
		return false, err
	}
	log.TPrintf("Js dependency manager for %s npm: %t", scanner.packageJSONPth, scanner.hasYarnLockFile)

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
	if scanner.usesExpo {
		options, warnings, err = scanner.expoOptions()
	} else {
		options, warnings, err = scanner.options()
	}
	return
}

// Configs implements ScannerInterface.Configs function.
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	if scanner.usesExpo {
		return scanner.expoConfigs()
	}
	return scanner.configs()
}

// DefaultOptions implements ScannerInterface.DefaultOptions function.
func (scanner *Scanner) DefaultOptions() models.OptionNode {
	expoOption := models.NewOption(isExpoCLIInputTitle, isExpoCLIInputSummary, "", models.TypeSelector)

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
