package cli

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v2"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/output"
	"github.com/bitrise-core/bitrise-init/scanners"
	"github.com/bitrise-core/bitrise-init/scanners/android"
	"github.com/bitrise-core/bitrise-init/scanners/fastlane"
	"github.com/bitrise-core/bitrise-init/scanners/ios"
	"github.com/bitrise-core/bitrise-init/scanners/xamarin"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/codegangsta/cli"
)

const (
	defaultOutputDir = "_defaults"
)

func initManualConfig(c *cli.Context) {
	PrintHeader(c)

	//
	// Config
	isCI := c.GlobalBool("ci")
	outputDir := c.String("output-dir")
	formatStr := c.String("format")

	currentDir, err := pathutil.AbsPath("./")
	if err != nil {
		log.Fatalf("Failed to get current directory, error: %s", err)
	}

	if outputDir == "" {
		outputDir = filepath.Join(currentDir, defaultOutputDir)
	}
	outputDir, err = pathutil.AbsPath(outputDir)
	if err != nil {
		log.Fatalf("Failed to get abs path (%s), error: %s", outputDir, err)
	}

	if formatStr == "" {
		formatStr = output.YAMLFormat.String()
	}
	format, err := output.ParseFormat(formatStr)
	if err != nil {
		log.Fatalf("Failed to parse format, err: %s", err)
	}
	if format != output.JSONFormat && format != output.YAMLFormat {
		log.Fatalf("Not allowed output format (%v), options: [%s, %s]", format, output.YAMLFormat.String(), output.JSONFormat.String())
	}

	if isCI {
		log.Info(colorstring.Yellow("CI mode"))
	}
	log.Info(colorstring.Yellowf("output dir: %s", outputDir))
	log.Info(colorstring.Yellowf("output format: %s", format))
	fmt.Println()

	//
	// Scan
	projectScanners := []scanners.ScannerInterface{
		new(android.Scanner),
		new(xamarin.Scanner),
		new(ios.Scanner),
		new(fastlane.Scanner),
	}

	projectTypeOptionMap := map[string]models.OptionModel{}
	projectTypeConfigMap := map[string]models.BitriseConfigMap{}

	for _, detector := range projectScanners {
		detectorName := detector.Name()

		option := detector.DefaultOptions()

		log.Debug()
		log.Debug("Analyze result:")
		bytes, err := yaml.Marshal(option)
		if err != nil {
			log.Fatalf("Failed to marshal option, err: %s", err)
		}
		log.Debugf("\n%v", string(bytes))

		projectTypeOptionMap[detectorName] = option

		configs, err := detector.DefaultConfigs()
		if err != nil {
			log.Fatalf("Failed create default configs, error: %s", err)
		}

		for name, config := range configs {
			log.Debugf("  name: %s", name)

			bytes, err := yaml.Marshal(config)
			if err != nil {
				log.Fatalf("Failed to marshal option, err: %s", err)
			}
			log.Debugf("\n%v", string(bytes))
		}

		projectTypeConfigMap[detectorName] = configs
	}

	customConfigs, err := scanners.CustomConfig()
	if err != nil {
		log.Fatalf("Failed create default custom configs, error: %s", err)
	}

	projectTypeConfigMap["custom"] = customConfigs

	//
	// Write output to files
	if isCI {
		log.Infof(colorstring.Blue("Saving outputs:"))

		scanResult := models.ScanResultModel{
			OptionsMap: projectTypeOptionMap,
			ConfigsMap: projectTypeConfigMap,
		}

		if err := os.MkdirAll(outputDir, 0700); err != nil {
			log.Fatalf("Failed to create (%s), err: %s", outputDir, err)
		}

		pth := path.Join(outputDir, "result")
		if err := output.Print(scanResult, format, pth); err != nil {
			log.Fatalf("Failed to print result, error: %s", err)
		}
		log.Infof("  scan result: %s", colorstring.Blue(pth))

		return
	}

	//
	// Write output to files
	if isCI {
		log.Infof(colorstring.Blue("Saving outputs:"))

		scanResult := models.ScanResultModel{
			OptionsMap: projectTypeOptionMap,
			ConfigsMap: projectTypeConfigMap,
		}

		if err := os.MkdirAll(outputDir, 0700); err != nil {
			log.Fatalf("Failed to create (%s), error: %s", outputDir, err)
		}

		pth := path.Join(outputDir, "result")
		if err := output.Print(scanResult, format, pth); err != nil {
			log.Fatalf("Failed to print result, error: %s", err)
		}
		log.Infof("  scan result: %s", colorstring.Blue(pth))

		return
	}

	//
	// Select option
	log.Infof(colorstring.Blue("Collecting inputs:"))

	for detectorName, option := range projectTypeOptionMap {
		log.Infof("  Scanner: %s", colorstring.Blue(detectorName))

		// Init
		platformOutputDir := path.Join(outputDir, detectorName)
		if exist, err := pathutil.IsDirExists(platformOutputDir); err != nil {
			log.Fatalf("Failed to check if path (%s) exis, error: %s", platformOutputDir, err)
		} else if exist {
			if err := os.RemoveAll(platformOutputDir); err != nil {
				log.Fatalf("Failed to cleanup (%s), error: %s", platformOutputDir, err)
			}
		}

		if err := os.MkdirAll(platformOutputDir, 0700); err != nil {
			log.Fatalf("Failed to create (%s), error: %s", platformOutputDir, err)
		}

		// Collect inputs
		configPth := ""
		appEnvs := []envmanModels.EnvironmentItemModel{}

		var walkDepth func(option models.OptionModel)

		walkDepth = func(option models.OptionModel) {
			optionEnvKey, selectedValue, err := askForValue(option)
			if err != nil {
				log.Fatalf("Failed to ask for vale, error: %s", err)
			}

			if optionEnvKey == "" {
				configPth = selectedValue
			} else {
				appEnvs = append(appEnvs, envmanModels.EnvironmentItemModel{
					optionEnvKey: selectedValue,
				})
			}

			nestedOptions, found := option.ValueMap[selectedValue]
			if !found {
				return
			}

			walkDepth(nestedOptions)
		}

		walkDepth(option)

		log.Debug()
		log.Debug("Selected app envs:")
		aBytes, err := yaml.Marshal(appEnvs)
		if err != nil {
			log.Fatalf("Failed to marshal appEnvs, error: %s", err)
		}
		log.Debugf("\n%v", string(aBytes))

		configMap := projectTypeConfigMap[detectorName]
		configStr := configMap[configPth]

		var config bitriseModels.BitriseDataModel
		if err := yaml.Unmarshal([]byte(configStr), &config); err != nil {
			log.Fatalf("Failed to unmarshal config, error: %s", err)
		}

		config.App.Environments = appEnvs

		log.Debug()
		log.Debug("Config:")
		log.Debugf("  name: %s", configPth)
		aBytes, err = yaml.Marshal(config)
		if err != nil {
			log.Fatalf("Failed to marshal config, error: %s", err)
		}
		log.Debugf("\n%v", string(aBytes))

		// Write config to file
		pth := path.Join(platformOutputDir, configPth)
		if err := output.Print(config, format, pth); err != nil {
			log.Fatalf("Failed to print result, error: %s", err)
		}
		log.Infof("  bitrise.yml template: %s", colorstring.Blue(pth))
		fmt.Println()
	}
}
