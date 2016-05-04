package cli

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"encoding/json"

	"gopkg.in/yaml.v2"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-core/bitrise-plugins-init/models"
	"github.com/bitrise-core/bitrise-plugins-init/scanners"
	"github.com/bitrise-core/bitrise-plugins-init/version"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/goinp/goinp"
	"github.com/codegangsta/cli"
)

func askForValue(option models.OptionModel) (string, string, error) {
	optionValues := option.GetValues()

	selectedValue := ""
	if len(optionValues) == 1 {
		selectedValue = optionValues[0]
	} else {
		question := fmt.Sprintf("Select: %s (%s)", option.Title, option.Key)
		answer, err := goinp.SelectFromStrings(question, optionValues)
		if err != nil {
			return "", "", err
		}

		selectedValue = answer
	}

	return option.EnvKey, selectedValue, nil
}

func initConfig(c *cli.Context) {
	//
	// Config
	isCI := c.GlobalBool("ci")
	isPrivate := c.Bool("private")
	searchDir := c.String("dir")
	outputDir := c.String("output-dir")

	currentDir, err := pathutil.AbsPath("./")
	if err != nil {
		log.Fatalf("Failed to get current directory, error: %s", err)
	}

	if searchDir == "" {
		searchDir = currentDir
		// searchDir = "/Users/godrei/Develop/bitrise/sample-apps/sample-apps-ios-cocoapods"
		searchDir = "/Users/godrei/Develop/bitrise/sample-apps/sample-apps-android"
		// searchDir = "/Users/godrei/Develop/bitrise/sample-apps/sample-apps-xamarin-uitest"
		// searchDir = "/Users/godrei/Develop/bitrise/sample-apps/fastlane-example"
	}

	if outputDir == "" {
		outputDir = filepath.Join(currentDir, "scan_result")
	}

	fmt.Println()
	log.Info(colorstring.Greenf("Running scanner v%s", version.VERSION))
	fmt.Println()

	if isCI {
		log.Info(colorstring.Yellow("plugin runs in CI mode"))
	}
	if isPrivate {
		log.Info(colorstring.Yellow("scanning private repository"))
	}
	log.Info(colorstring.Yellowf("scan dir: %s", searchDir))
	log.Info(colorstring.Yellowf("output dir: %s", outputDir))
	fmt.Println()

	//
	// Scan
	platformDetectors := []scanners.ScannerInterface{
		new(scanners.Android),
		new(scanners.Xamarin),
		new(scanners.Ios),
		new(scanners.Fastlane),
	}
	optionsMap := map[string][]models.OptionModel{}
	configsMap := map[string]map[string]bitriseModels.BitriseDataModel{}

	log.Infof(colorstring.Blue("Running scanners:"))
	for _, detector := range platformDetectors {
		detectorName := detector.Name()
		log.Infof("  Scanner: %s", colorstring.Blue(detectorName))

		detector.Configure(searchDir)
		detected, err := detector.DetectPlatform()
		if err != nil {
			log.Fatalf("Scanner failed, error: %s", err)
		}

		if !detected {
			log.Info("  Platform not detected")
			fmt.Println()
			continue
		}

		log.Info("  Platform detected")
		log.Info("  +------------------------------------------------------------------------------+")
		log.Info("  |                                                                              |")

		options, err := detector.Analyze()
		if err != nil {
			log.Fatalf("Analyzer failed, error: %s", err)
		}

		log.Debug()
		log.Debug("Analyze result:")
		bytes, err := yaml.Marshal(options)
		if err != nil {
			log.Fatalf("Failed to marshal options, err: %s", err)
		}
		log.Debugf("\n%v", string(bytes))

		optionsMap[detectorName] = options

		// Generate configs
		log.Debug()
		log.Debug("Generated configs:")
		configs := detector.Configs(isPrivate)
		for name, config := range configs {
			log.Debugf("  name: %s", name)

			bytes, err := yaml.Marshal(config)
			if err != nil {
				log.Fatalf("Failed to marshal options, err: %s", err)
			}
			log.Debugf("\n%v", string(bytes))
		}

		configsMap[detectorName] = configs

		log.Info("  |                                                                              |")
		log.Info("  +------------------------------------------------------------------------------+")
		fmt.Println()
	}

	//
	// Write output to files
	if isCI {
		log.Infof(colorstring.Blue("Saving outputs:"))

		scanResult := models.ScanResultModel{
			OptionsMap: optionsMap,
			ConfigsMap: configsMap,
		}

		if err := os.MkdirAll(outputDir, 0700); err != nil {
			log.Fatalf("Failed to create (%s), err: %s", outputDir, err)
		}

		pth := path.Join(outputDir, "result.json")

		scanResultBytes, err := json.MarshalIndent(scanResult, "", "\t")
		if err != nil {
			log.Fatalf("Failed to marshal scan result, error: %s", err)
		}

		if err := fileutil.WriteBytesToFile(pth, scanResultBytes); err != nil {
			log.Fatalf("Failed to save scan result, err: %s", err)
		}
		log.Infof("  scan result: %s", colorstring.Blue(pth))

		return
	}

	//
	// Select options
	log.Infof(colorstring.Blue("Collecting inputs:"))

	for detectorName, options := range optionsMap {
		log.Infof("  Scanner: %s", colorstring.Blue(detectorName))

		// Init
		platformOutputDir := path.Join(outputDir, detectorName)
		if exist, err := pathutil.IsDirExists(platformOutputDir); err != nil {
			log.Fatalf("Failed to check if path (%s) exis, err: %s", platformOutputDir, err)
		} else if exist {
			if err := os.RemoveAll(platformOutputDir); err != nil {
				log.Fatalf("Failed to cleanup (%s), err: %s", platformOutputDir, err)
			}
		}

		if err := os.MkdirAll(platformOutputDir, 0700); err != nil {
			log.Fatalf("Failed to create (%s), err: %s", platformOutputDir, err)
		}

		// Collect inputs
		configPth := ""
		appEnvs := []envmanModels.EnvironmentItemModel{}

		var walkWidth func(options []models.OptionModel)

		walkDepth := func(option models.OptionModel) {
			optionEnvKey, selectedValue, err := askForValue(option)
			if err != nil {
				log.Fatalf("Failed to ask for vale of key (%s), error: %s", option.Key, err)
			}

			if optionEnvKey == "" {
				configPth = selectedValue
			} else {
				appEnvs = append(appEnvs, envmanModels.EnvironmentItemModel{
					optionEnvKey: selectedValue,
				})
			}

			nestedOptions := option.ValueMap[selectedValue]
			if len(nestedOptions) == 0 {
				return
			}

			walkWidth(nestedOptions)
		}

		walkWidth = func(options []models.OptionModel) {
			for _, option := range options {
				walkDepth(option)
			}
		}

		walkWidth(options)

		log.Debug()
		log.Debug("Selected app envs:")
		aBytes, err := yaml.Marshal(appEnvs)
		if err != nil {
			log.Fatalf("Failed to marshal appEnvs, err: %s", err)
		}
		log.Debugf("\n%v", string(aBytes))

		configMap := configsMap[detectorName]
		config := configMap[configPth]
		config.App.Environments = appEnvs

		log.Debug()
		log.Debug("Config:")
		log.Debugf("  name: %s", configPth)
		aBytes, err = yaml.Marshal(config)
		if err != nil {
			log.Fatalf("Failed to marshal config, err: %s", err)
		}
		log.Debugf("\n%v", string(aBytes))

		// Write config to file
		configBytes, err := yaml.Marshal(config)
		if err != nil {
			log.Fatalf("Failed to marshal config, error: %#v", err)
		}

		pth := path.Join(platformOutputDir, configPth+".yml")
		if err := fileutil.WriteBytesToFile(pth, configBytes); err != nil {
			log.Fatalf("Failed to save configs, err: %s", err)
		}
		log.Infof("  bitrise.yml template: %s", colorstring.Blue(pth))
		fmt.Println()
	}
}
