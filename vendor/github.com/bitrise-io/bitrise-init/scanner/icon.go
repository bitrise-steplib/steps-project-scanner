package scanner

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/go-utils/pathutil"
)

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
