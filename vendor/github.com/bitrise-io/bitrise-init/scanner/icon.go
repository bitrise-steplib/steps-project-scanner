package scanner

import (
	"fmt"
	"os"
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

	for _, icon := range icons {
		if err := copyFile(icon.Path, filepath.Join(outputDir, icon.Filename)); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src string, dst string) (err error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err = os.WriteFile(dst, data, 0644); err != nil {
		return err
	}

	return nil
}
