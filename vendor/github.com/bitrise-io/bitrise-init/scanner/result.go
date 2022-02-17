package scanner

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/analytics"
	"github.com/bitrise-io/bitrise-init/errormapper"
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/output"
	"github.com/bitrise-io/go-steputils/step"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

// GenerateScanResult runs the scanner, returns the results and if any platform was detected.
func GenerateScanResult(searchDir string, isPrivateRepository bool) (models.ScanResultModel, bool) {
	scanResult := Config(searchDir, isPrivateRepository)

	logUnknownTools(searchDir)

	var platforms []string
	for platform := range scanResult.ScannerToOptionRoot {
		platforms = append(platforms, platform)
	}

	if len(platforms) == 0 {
		errorMessage := "No known platform detected"
		analytics.LogError(noPlatformDetectedTag, nil, errorMessage)

		scanResult.AddErrorWithRecommendation("general", models.ErrorWithRecommendations{
			Error: errorMessage,
			Recommendations: step.Recommendation{
				"NoPlatformDetected":            true,
				errormapper.DetailedErrorRecKey: newNoPlatformDetectedGenericDetail(),
			},
		})
		return scanResult, false
	}
	return scanResult, true
}

// GenerateAndWriteResults runs the scanner and saves results to the given output dir.
func GenerateAndWriteResults(searchDir string, outputDir string, format output.Format) (models.ScanResultModel, error) {
	result, detected := GenerateScanResult(searchDir, true)

	// Write output to files
	log.TInfof("Saving outputs:")
	outputPth, err := writeScanResult(result, outputDir, format)
	if err != nil {
		return result, fmt.Errorf("Failed to write output, error: %s", err)
	}
	log.TPrintf("scan result: %s", outputPth)

	if !detected {
		printDirTree()
		return result, fmt.Errorf("No known platform detected")
	}
	return result, nil
}

func printDirTree() {
	cmd := command.New("which", "tree")
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil || out == "" {
		log.TErrorf("tree not installed, can not list files")
	} else {
		fmt.Println()
		cmd := command.NewWithStandardOuts("tree", ".", "-L", "3")
		log.TPrintf("$ %s", cmd.PrintableCommandArgs())
		if err := cmd.Run(); err != nil {
			log.TErrorf("Failed to list files in current directory, error: %s", err)
		}
	}
}

func writeScanResult(scanResult models.ScanResultModel, outputDir string, format output.Format) (string, error) {
	if len(scanResult.Icons) != 0 {
		const iconDirName = "icons"
		iconsOutputDir := filepath.Join(outputDir, iconDirName)
		if err := os.MkdirAll(iconsOutputDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create icons directory")
		}
		if err := copyIconsToDir(scanResult.Icons, iconsOutputDir); err != nil {
			return "", fmt.Errorf("failed to copy icons, error: %s", err)
		}
	}

	return output.WriteToFile(scanResult, format, path.Join(outputDir, "result"))
}

func logUnknownTools(searchDir string) {
	for _, detector := range UnknownToolDetectors {
		result, err := detector.DetectToolIn(searchDir)
		if err != nil {
			log.Warnf("Failed to detect %s: %s", detector.ToolName(), err)
		}
		if result.Detected {
			data := map[string]interface{}{
				"project_tree": result.ProjectTree,
			}
			analytics.LogInfo("tool-detector", data, "Tool detected: %s", detector.ToolName())
			log.Debugf("Tool detected: %s", detector.ToolName())
		}
	}
}
