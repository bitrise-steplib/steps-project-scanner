package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/output"
	"github.com/bitrise-io/bitrise-init/scanner"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

func runScanner(searchDir string, outputDir string) error {
	log.TInfof(colorstring.Yellow("CI mode"))
	log.TInfof(colorstring.Yellowf("scan dir: %s", searchDir))
	log.TInfof(colorstring.Yellowf("output dir: %s", outputDir))
	log.TInfof(colorstring.Yellowf("output format: json"))
	fmt.Println()

	currentDir, err := pathutil.AbsPath("./")
	if err != nil {
		return fmt.Errorf("failed to expand path (%s), error: %s", outputDir, err)
	}

	if searchDir == "" {
		searchDir = currentDir
	}
	searchDir, err = pathutil.AbsPath(searchDir)
	if err != nil {
		return fmt.Errorf("failed to expand path (%s), error: %s", outputDir, err)
	}

	outputDir, err = pathutil.AbsPath(outputDir)
	if err != nil {
		return fmt.Errorf("failed to expand path (%s), error: %s", outputDir, err)
	}
	if exist, err := pathutil.IsDirExists(outputDir); err != nil {
		return err
	} else if !exist {
		if err := os.MkdirAll(outputDir, 0700); err != nil {
			return fmt.Errorf("failed to create (%s), error: %s", outputDir, err)
		}
	}
	// ---

	scanResult := scanner.Config(searchDir)

	platforms := []string{}
	for platform := range scanResult.ScannerToOptionRoot {
		platforms = append(platforms, platform)
	}

	format := output.JSONFormat
	if len(platforms) == 0 {
		cmd := command.New("which", "tree")
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil || out == "" {
			log.TErrorf("tree not installed, can not list files")
		} else {
			fmt.Println()
			cmd := command.NewWithStandardOuts("tree", ".", "-L", "3")
			log.TPrintf("$ %s", cmd.PrintableCommandArgs())
			if err := cmd.Run(); err != nil {
				log.TErrorf("failed to list files in current directory, error: %s", err)
			}
		}

		log.TInfof("Saving outputs:")
		scanResult.AddError("general", "No known platform detected")

		outputPth, err := writeScanResult(scanResult, outputDir, format)
		if err != nil {
			return fmt.Errorf("failed to write output, error: %s", err)
		}

		log.TPrintf("scan result: %s", outputPth)
		return fmt.Errorf("no known platform detected")
	}

	// Write output to files
	log.TInfof("Saving outputs:")

	outputPth, err := writeScanResult(scanResult, outputDir, format)
	if err != nil {
		return fmt.Errorf("failed to write output, error: %s", err)
	}

	log.TPrintf("  scan result: %s", outputPth)
	return nil
}

func writeScanResult(scanResult models.ScanResultModel, outputDir string, format output.Format) (string, error) {
	/*
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
	*/

	return output.WriteToFile(scanResult, format, path.Join(outputDir, "result"))
}

/*
func copyIconsToDir(icons models.Icons, outputDir string) error {
	if exist, err := pathutil.IsDirExists(outputDir); err != nil {
		return err
	} else if !exist {
		return fmt.Errorf("output dir does not exist")
	}

	for iconID, iconPath := range icons {
		if err := copyFile(iconPath, filepath.Join(outputDir, iconID)); err != nil {
			return err
		}
	}
	return nil
}
*/

func copyFile(src string, dst string) (err error) {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(dst, data, 0644); err != nil {
		return err
	}

	return nil
}
