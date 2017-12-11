package scanner

import (
	"fmt"
	"os"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
)

// Config ...
func Config(searchDir string) models.ScanResultModel {
	result := models.ScanResultModel{}

	//
	// Setup
	currentDir, err := os.Getwd()
	if err != nil {
		result.AddError("general", fmt.Sprintf("Failed to expand current directory path, error: %s", err))
		return result
	}

	if searchDir == "" {
		searchDir = currentDir
	} else {
		absScerach, err := pathutil.AbsPath(searchDir)
		if err != nil {
			result.AddError("general", fmt.Sprintf("Failed to expand path (%s), error: %s", searchDir, err))
			return result
		}
		searchDir = absScerach
	}

	if searchDir != currentDir {
		if err := os.Chdir(searchDir); err != nil {
			result.AddError("general", fmt.Sprintf("Failed to change dir, to (%s), error: %s", searchDir, err))
			return result
		}
		defer func() {
			if err := os.Chdir(currentDir); err != nil {
				log.TWarnf("Failed to change dir, to (%s), error: %s", searchDir, err)
			}
		}()
	}
	// ---

	//
	// Scan
	projectScanners := scanners.ActiveScanners

	projectTypeErrorMap := map[string]models.Errors{}
	projectTypeWarningMap := map[string]models.Warnings{}
	projectTypeOptionMap := map[string]models.OptionModel{}
	projectTypeConfigMap := map[string]models.BitriseConfigMap{}

	excludedScannerNames := []string{}

	log.TInfof(colorstring.Blue("Running scanners:"))
	fmt.Println()

	for _, detector := range projectScanners {
		detectorName := detector.Name()
		detectorWarnings := []string{}
		detectorErrors := []string{}

		log.TInfof("Scanner: %s", colorstring.Blue(detectorName))

		if sliceutil.IsStringInSlice(detectorName, excludedScannerNames) {
			log.TWarnf("scanner is marked as excluded, skipping...")
			fmt.Println()
			continue
		}

		log.TPrintf("+------------------------------------------------------------------------------+")
		log.TPrintf("|                                                                              |")

		detected, err := detector.DetectPlatform(searchDir)
		if err != nil {
			log.TErrorf("Scanner failed, error: %s", err)
			detectorWarnings = append(detectorWarnings, err.Error())
			projectTypeWarningMap[detectorName] = detectorWarnings
			detected = false
		}

		if !detected {
			log.TPrintf("|                                                                              |")
			log.TPrintf("+------------------------------------------------------------------------------+")
			fmt.Println()
			continue
		}

		options, projectWarnings, err := detector.Options()
		detectorWarnings = append(detectorWarnings, projectWarnings...)

		if err != nil {
			log.TErrorf("Analyzer failed, error: %s", err)
			detectorWarnings = append(detectorWarnings, err.Error())
			projectTypeWarningMap[detectorName] = detectorWarnings

			log.TPrintf("|                                                                              |")
			log.TPrintf("+------------------------------------------------------------------------------+")
			fmt.Println()
			continue
		}

		projectTypeWarningMap[detectorName] = detectorWarnings
		projectTypeOptionMap[detectorName] = options

		// Generate configs
		configs, err := detector.Configs()
		if err != nil {
			log.TErrorf("Failed to generate config, error: %s", err)
			detectorErrors = append(detectorErrors, err.Error())
			projectTypeErrorMap[detectorName] = detectorErrors
			continue
		}

		projectTypeConfigMap[detectorName] = configs

		log.TPrintf("|                                                                              |")
		log.TPrintf("+------------------------------------------------------------------------------+")

		exludedScanners := detector.ExcludedScannerNames()
		if len(exludedScanners) > 0 {
			log.TWarnf("Scanner will exclude scanners: %v", exludedScanners)
			excludedScannerNames = append(excludedScannerNames, exludedScanners...)
		}

		fmt.Println()
	}
	// ---

	return models.ScanResultModel{
		PlatformOptionMap:    projectTypeOptionMap,
		PlatformConfigMapMap: projectTypeConfigMap,
		PlatformWarningsMap:  projectTypeWarningMap,
		PlatformErrorsMap:    projectTypeErrorMap,
	}
}
