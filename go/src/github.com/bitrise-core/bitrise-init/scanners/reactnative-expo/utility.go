package expo

import (
	"github.com/bitrise-core/bitrise-init/scanners/cordova"
	"github.com/bitrise-io/go-utils/log"
)

// Constants ...
const (
	ProjectLocationInputEnvKey = "PROJECT_LOCATION"
	ProjectLocationInputTitle  = "The root directory of an Android project"

	ModuleInputEnvKey = "MODULE"
	ModuleInputTitle  = "Module"
)

func configName(hasAndroidProject, hasIosProject, hasNPMTest bool) string {
	name := "react-native-expo"
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

func defaultConfigName() string {
	return "default-react-native-expo-config"
}

// FindDependencies ...
func FindDependencies(filePath, dep, scrt string) (bool, error) {
	packages, err := cordova.ParsePackagesJSON(filePath)
	if err != nil {
		return false, err
	}

	log.TPrintf("Searching for %s", dep)

	dependencyFound := false
	for dependency := range packages.Dependencies {
		if dependency == dep {
			dependencyFound = true
		}
	}

	if !dependencyFound {
		return false, nil
	}

	log.TPrintf("Searching for %s", scrt)

	dependencyFound = false
	for script := range packages.Scripts {
		if script == scrt {
			dependencyFound = true
		}
	}
	return dependencyFound, nil
}
