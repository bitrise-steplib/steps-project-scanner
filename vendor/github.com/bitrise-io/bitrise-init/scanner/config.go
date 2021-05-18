package scanner

import (
	"errors"
	"fmt"
	"os"

	"github.com/bitrise-io/bitrise-init/analytics"
	"github.com/bitrise-io/bitrise-init/errormapper"
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners"
	"github.com/bitrise-io/go-steputils/step"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
)

const otherProjectType = "other"

type status int

const (
	// in case DetectPlatform() returned error, or false
	notDetected status = iota
	// in case DetectPlatform() returned true, but Options() or Config() returned an error
	detectedWithErrors
	// in case DetectPlatform() returned true, Options() and Config() returned no error
	detected
)

const (
	optionsFailedTag        = "options_failed"
	configsFailedTag        = "configs_failed"
	detectPlatformFailedTag = "detect_platform_failed"
	noPlatformDetectedTag   = "no_platform_detected"
)

type scannerOutput struct {
	status status

	// can always be set
	// warnings returned by DetectPlatform(), Options()
	warnings                   models.Warnings
	warningsWithRecommendation []models.ErrorWithRecommendations

	// set if scanResultStatus is scanResultDetectedWithErrors
	// errors returned by Config()
	errors                   models.Errors
	errorsWithRecommendation []models.ErrorWithRecommendations

	// set if scanResultStatus is scanResultDetected
	options          models.OptionNode
	configs          models.BitriseConfigMap
	icons            models.Icons
	excludedScanners []string
}

func (o *scannerOutput) AddErrors(tag string, errs ...string) {
	for _, err := range errs {
		recommendation := mapRecommendation(tag, err)
		if recommendation != nil {
			o.errorsWithRecommendation = append(o.errorsWithRecommendation, models.ErrorWithRecommendations{
				Error:           err,
				Recommendations: recommendation,
			})
			return
		}

		o.errors = append(o.errors, err)
	}
}

func (o *scannerOutput) AddWarnings(tag string, errs ...string) {
	for _, err := range errs {
		recommendation := mapRecommendation(tag, err)
		if recommendation != nil {
			o.warningsWithRecommendation = append(o.warningsWithRecommendation, models.ErrorWithRecommendations{
				Error:           err,
				Recommendations: recommendation,
			})
			return
		}

		o.warnings = append(o.warnings, err)
	}
}

// Config ...
func Config(searchDir string) models.ScanResultModel {
	result := models.ScanResultModel{}

	//
	// Setup
	currentDir, err := os.Getwd()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to expand current directory path: %s", err)
		result.AddErrorWithRecommendation("general", models.ErrorWithRecommendations{
			Error: errorMsg,
			Recommendations: step.Recommendation{
				errormapper.DetailedErrorRecKey: newDetectPlatformFailedGenericDetail(errorMsg),
			},
		})
		return result
	}

	if searchDir == "" {
		searchDir = currentDir
	} else {
		absScerach, err := pathutil.AbsPath(searchDir)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to expand path (%s): %s", searchDir, err)
			result.AddErrorWithRecommendation("general", models.ErrorWithRecommendations{
				Error: errorMsg,
				Recommendations: step.Recommendation{
					errormapper.DetailedErrorRecKey: newDetectPlatformFailedGenericDetail(errorMsg),
				},
			})
			return result
		}
		searchDir = absScerach
	}

	if searchDir != currentDir {
		if err := os.Chdir(searchDir); err != nil {
			errorMsg := fmt.Sprintf("Failed to change dir, to (%s): %s", searchDir, err)
			result.AddErrorWithRecommendation("general", models.ErrorWithRecommendations{
				Error: errorMsg,
				Recommendations: step.Recommendation{
					errormapper.DetailedErrorRecKey: newDetectPlatformFailedGenericDetail(errorMsg),
				},
			})
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
	log.TInfof(colorstring.Blue("Running scanners:"))
	fmt.Println()

	// Collect scanner outputs, by scanner name
	scannerToOutput := map[string]scannerOutput{}
	{
		projectScannerToOutputs := runScanners(scanners.ProjectScanners, searchDir)
		detectedProjectTypes := getDetectedScannerNames(projectScannerToOutputs)
		log.Printf("Detected project types: %s", detectedProjectTypes)
		fmt.Println()

		// Project types are needed by tool scanners, to create decision tree on which project type
		// to actually use in bitrise.yml
		if len(detectedProjectTypes) == 0 {
			detectedProjectTypes = []string{otherProjectType}
		}
		for _, toolScanner := range scanners.AutomationToolScanners {
			toolScanner.(scanners.AutomationToolScanner).SetDetectedProjectTypes(detectedProjectTypes)
		}

		toolScannerToOutputs := runScanners(scanners.AutomationToolScanners, searchDir)
		detectedAutomationToolScanners := getDetectedScannerNames(toolScannerToOutputs)
		log.Printf("Detected automation tools: %s", detectedAutomationToolScanners)
		fmt.Println()

		// Merge project and tool scanner outputs
		scannerToOutput = toolScannerToOutputs
		for scanner, scannerOutput := range projectScannerToOutputs {
			scannerToOutput[scanner] = scannerOutput
		}
	}

	scannerToWarnings := map[string]models.Warnings{}
	scannerToWarningsWithRecommendation := map[string]models.ErrorsWithRecommendations{}

	scannerToErrors := map[string]models.Errors{}
	scannerToErrorsWithRecommendations := map[string]models.ErrorsWithRecommendations{}

	scannerToOptions := map[string]models.OptionNode{}
	scannerToConfigMap := map[string]models.BitriseConfigMap{}
	icons := models.Icons{}
	for scanner, scannerOutput := range scannerToOutput {
		// Currently the tests except an empty warning list if no warnings
		// are created in the not detect case.
		if scannerOutput.status == notDetected && (len(scannerOutput.warnings) > 0 || len(scannerOutput.warningsWithRecommendation) > 0) ||
			scannerOutput.status != notDetected {
			scannerToWarnings[scanner] = scannerOutput.warnings
			scannerToWarningsWithRecommendation[scanner] = scannerOutput.warningsWithRecommendation
		}
		if (len(scannerOutput.errors) > 0 || len(scannerOutput.errorsWithRecommendation) > 0) &&
			(scannerOutput.status == detected || scannerOutput.status == detectedWithErrors) {
			scannerToErrors[scanner] = scannerOutput.errors
			scannerToErrorsWithRecommendations[scanner] = scannerOutput.errorsWithRecommendation
		}
		if len(scannerOutput.configs) > 0 && scannerOutput.status == detected {
			scannerToOptions[scanner] = scannerOutput.options
			scannerToConfigMap[scanner] = scannerOutput.configs
		}
		icons = append(icons, scannerOutput.icons...)
	}
	return models.ScanResultModel{
		ScannerToOptionRoot:                  scannerToOptions,
		ScannerToBitriseConfigMap:            scannerToConfigMap,
		ScannerToWarnings:                    scannerToWarnings,
		ScannerToErrors:                      scannerToErrors,
		ScannerToErrorsWithRecommendations:   scannerToErrorsWithRecommendations,
		ScannerToWarningsWithRecommendations: scannerToWarningsWithRecommendation,
		Icons:                                icons,
	}
}

func runScanners(scannerList []scanners.ScannerInterface, searchDir string) map[string]scannerOutput {
	scannerOutputs := map[string]scannerOutput{}
	var excludedScannerNames []string
	for _, scanner := range scannerList {
		log.TInfof("Scanner: %s", colorstring.Blue(scanner.Name()))
		if sliceutil.IsStringInSlice(scanner.Name(), excludedScannerNames) {
			log.TWarnf("scanner is marked as excluded, skipping...")
			fmt.Println()
			continue
		}

		log.TPrintf("+------------------------------------------------------------------------------+")
		log.TPrintf("|                                                                              |")
		scannerOutput := runScanner(scanner, searchDir)
		log.TPrintf("|                                                                              |")
		log.TPrintf("+------------------------------------------------------------------------------+")
		fmt.Println()

		scannerOutputs[scanner.Name()] = scannerOutput
		excludedScannerNames = append(excludedScannerNames, scannerOutput.excludedScanners...)
	}
	return scannerOutputs
}

// Collect output of a specific scanner
func runScanner(detector scanners.ScannerInterface, searchDir string) scannerOutput {
	output := scannerOutput{}

	if isDetect, err := detector.DetectPlatform(searchDir); err != nil {
		data := detectorErrorData(detector.Name(), err)
		analytics.LogError(detectPlatformFailedTag, data, "%s detector DetectPlatform failed", detector.Name())

		log.TErrorf("Scanner failed, error: %s", err)

		output.status = notDetected
		output.AddWarnings(detectPlatformFailedTag, err.Error())
		return output
	} else if !isDetect {
		output.status = notDetected
		return output
	}

	options, projectWarnings, icons, err := detector.Options()
	output.AddWarnings(optionsFailedTag, []string(projectWarnings)...)
	for _, warning := range projectWarnings {
		data := detectorErrorData(detector.Name(), errors.New(warning))
		analytics.LogWarn(optionsFailedTag, data, "%s detector Options warning", detector.Name())
	}

	if err != nil {
		data := detectorErrorData(detector.Name(), err)
		analytics.LogError(optionsFailedTag, data, "%s detector Options failed", detector.Name())

		log.TErrorf("Analyzer failed, error: %s", err)

		// Error returned as a warning
		output.status = detectedWithErrors
		output.AddWarnings(optionsFailedTag, err.Error())
		return output
	}

	// Generate configs
	configs, err := detector.Configs()
	if err != nil {
		data := detectorErrorData(detector.Name(), err)
		analytics.LogError(configsFailedTag, data, "%s detector Configs failed", detector.Name())

		log.TErrorf("Failed to generate config, error: %s", err)

		output.status = detectedWithErrors
		output.AddErrors(configsFailedTag, err.Error())
		return output
	}

	scannerExcludedScanners := detector.ExcludedScannerNames()
	if len(scannerExcludedScanners) > 0 {
		log.TWarnf("Scanner will exclude scanners: %v", scannerExcludedScanners)
	}

	output.status = detected
	output.options = options
	output.configs = configs
	output.icons = icons
	output.excludedScanners = scannerExcludedScanners
	return output
}

func getDetectedScannerNames(scannerOutputs map[string]scannerOutput) (names []string) {
	for scanner, scannerOutput := range scannerOutputs {
		if scannerOutput.status == detected {
			names = append(names, scanner)
		}
	}
	return
}
