package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v2"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/codegangsta/cli"
)

const (
	defaultOutputDir = "_defaults"
)

func manualInitConfig(c *cli.Context) {
	//
	// Config
	isCI := c.GlobalBool("ci")
	outputDir := c.String("output-dir")

	currentDir, err := pathutil.AbsPath("./")
	if err != nil {
		log.Fatalf("Failed to get current directory, error: %s", err)
	}

	if outputDir == "" {
		outputDir = filepath.Join(currentDir, defaultOutputDir)
	}

	fmt.Println()
	log.Info(colorstring.Greenf("Running %s v%s", c.App.Name, c.App.Version))
	fmt.Println()

	if isCI {
		log.Info(colorstring.Yellow("CI mode"))
	}
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

	optionsMap := map[string]models.OptionModel{}
	configsMap := map[string]map[string]bitriseModels.BitriseDataModel{}

	for _, detector := range platformDetectors {
		detectorName := detector.Name()

		option := detector.DefaultOptions()

		log.Debug()
		log.Debug("Analyze result:")
		bytes, err := yaml.Marshal(option)
		if err != nil {
			log.Fatalf("Failed to marshal option, err: %s", err)
		}
		log.Debugf("\n%v", string(bytes))

		optionsMap[detectorName] = option

		configs := detector.DefaultConfigs()

		for name, config := range configs {
			log.Debugf("  name: %s", name)

			bytes, err := yaml.Marshal(config)
			if err != nil {
				log.Fatalf("Failed to marshal option, err: %s", err)
			}
			log.Debugf("\n%v", string(bytes))
		}

		configsMap[detectorName] = configs
	}

	//
	// Write output to files
	if isCI {
		log.Infof(colorstring.Blue("Saving outputs:"))

		scanResult := models.ScanResultModel{
			OptionMap:  optionsMap,
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
}
