package scanner

import (
	"fmt"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

func validateIcon(iconPath string) error {
	const maxImageSize = 1024
	const maxFileSize = 2 * 1e6
	file, err := os.Open(iconPath)
	if err != nil {
		return err
	}

	if fileInfo, err := file.Stat(); err != nil {
		return fmt.Errorf("failed to get icon file stats, error: %s", err)
	} else if fileInfo.Size() > maxFileSize {
		return fmt.Errorf("icon file too large")
	}

	config, err := png.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("invalid png file, error: %s", err)
	}

	if config.Width > maxImageSize || config.Height > maxImageSize {
		return fmt.Errorf("Image dimensions larger than %d", maxImageSize)
	}
	return nil
}

func copyIconsToDir(icons models.Icons, outputDir string) error {
	if exist, err := pathutil.IsDirExists(outputDir); err != nil {
		return err
	} else if !exist {
		return fmt.Errorf("output dir does not exist")
	}

	for iconID, iconPath := range icons {
		if err := validateIcon(iconPath); err != nil {
			log.Warnf("%s", err)
			continue
		}
		if err := copyFile(iconPath, filepath.Join(outputDir, iconID)); err != nil {
			return err
		}
	}
	return nil
}

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
