package reactnative

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/pathutil"
)

// CollectPackageJSONFiles collects package.json files, with react-native dependency.
func CollectPackageJSONFiles(searchDir string) ([]string, error) {
	fileList, err := utility.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return nil, err
	}

	filters := []utility.FilterFunc{
		utility.BaseFilter("package.json", true),
		utility.ComponentFilter("node_modules", false),
	}
	packageFileList, err := utility.FilterPaths(fileList, filters...)
	if err != nil {
		return nil, err
	}

	relevantPackageFileList := []string{}
	for _, packageFile := range packageFileList {
		packages, err := utility.ParsePackagesJSON(packageFile)
		if err != nil {
			return nil, err
		}

		_, found := packages.Dependencies["react-native"]
		if found {
			relevantPackageFileList = append(relevantPackageFileList, packageFile)
		}
	}

	return relevantPackageFileList, nil
}

func containsYarnLock(absPackageJSONDir string) (bool, error) {
	if exist, err := pathutil.IsPathExists(filepath.Join(absPackageJSONDir, "yarn.lock")); err != nil {
		return false, fmt.Errorf("Failed to check if yarn.lock file exists in the workdir: %s", err)
	} else if exist {
		return true, nil
	}
	return false, nil
}
