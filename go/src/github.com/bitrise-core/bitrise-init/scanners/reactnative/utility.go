package reactnative

import (
	"github.com/bitrise-core/bitrise-init/scanners/cordova"
	"github.com/bitrise-core/bitrise-init/utility"
)

// CollectPackageJSONFiles - Collects package.json files, with react-native dependency
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
		packages, err := cordova.ParsePackagesJSON(packageFile)
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

func configName(hasAndroidProject, hasIosProject, hasNPMTest bool) string {
	name := "reactnative"
	if hasAndroidProject {
		name += "-android"
	}
	if hasIosProject {
		name += "-ios"
	}
	if hasNPMTest {
		name += "-test"
	}
	name += "-config"
	return name
}
