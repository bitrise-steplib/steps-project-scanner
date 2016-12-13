package scanner

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/pathutil"
)

// Config ...
func Config(searchDir string) (models.ScanResultModel, error) {

	//
	// Setup
	currentDir, err := os.Getwd()
	if err != nil {
		return models.ScanResultModel{}, fmt.Errorf("Failed to expand current directory path, error: %s", err)
	}

	if searchDir == "" {
		searchDir = currentDir
	} else {
		absScerach, err := pathutil.AbsPath(searchDir)
		if err != nil {
			return models.ScanResultModel{}, fmt.Errorf("Failed to expand path (%s), error: %s", searchDir, err)
		}
		searchDir = absScerach
	}

	if searchDir != currentDir {
		if err := os.Chdir(searchDir); err != nil {
			return models.ScanResultModel{}, fmt.Errorf("Failed to change dir, to (%s), error: %s", searchDir, err)
		}
		defer func() {
			if err := os.Chdir(currentDir); err != nil {
				log.Warnf("Failed to change dir, to (%s), error: %s", searchDir, err)
			}
		}()
	}
	// ---

	//
	// Scan
	projectScanners := scanners.ActiveScanners
	projectTypeWarningMap := map[string]models.Warnings{}
	projectTypeOptionMap := map[string]models.OptionModel{}
	projectTypeConfigMap := map[string]models.BitriseConfigMap{}

	log.Infof(colorstring.Blue("Running scanners:"))
	fmt.Println()

	for _, detector := range projectScanners {
		detectorName := detector.Name()
		log.Infof("Scanner: %s", colorstring.Blue(detectorName))

		log.Info("+------------------------------------------------------------------------------+")
		log.Info("|                                                                              |")

		detectorWarnings := []string{}
		detected, err := detector.DetectPlatform(searchDir)
		if err != nil {
			log.Errorf("Scanner failed, error: %s", err)
			detectorWarnings = append(detectorWarnings, err.Error())
			projectTypeWarningMap[detectorName] = detectorWarnings
			detected = false
		}

		if !detected {
			log.Info("|                                                                              |")
			log.Info("+------------------------------------------------------------------------------+")
			fmt.Println()
			continue
		}

		options, projectWarnings, err := detector.Options()
		detectorWarnings = append(detectorWarnings, projectWarnings...)

		if err != nil {
			log.Errorf("Analyzer failed, error: %s", err)
			detectorWarnings = append(detectorWarnings, err.Error())
			projectTypeWarningMap[detectorName] = detectorWarnings
			continue
		}

		projectTypeWarningMap[detectorName] = detectorWarnings
		projectTypeOptionMap[detectorName] = options

		// Generate configs
		configs, err := detector.Configs()
		if err != nil {
			return models.ScanResultModel{}, fmt.Errorf("Failed create configs, error: %s", err)
		}

		projectTypeConfigMap[detectorName] = configs

		log.Info("|                                                                              |")
		log.Info("+------------------------------------------------------------------------------+")
		fmt.Println()
	}
	// ---

	return models.ScanResultModel{
		OptionsMap:  projectTypeOptionMap,
		ConfigsMap:  projectTypeConfigMap,
		WarningsMap: projectTypeWarningMap,
	}, nil
}
