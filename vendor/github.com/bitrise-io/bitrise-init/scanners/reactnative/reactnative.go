package reactnative

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/detectors/gradle"
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/scanners/java"
	"github.com/bitrise-io/bitrise-init/scanners/nodejs"
	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

const scannerName = "react-native"

const (
	projectDirInputTitle   = "React Native project directory"
	projectDirInputSummary = "Path of the directory containing the project's `package.json` file."
	projectDirInputEnvKey  = "WORKDIR"

	isExpoBasedProjectInputTitle   = "Is this an [Expo](https://expo.dev)-based React Native project?"
	isExpoBasedProjectInputSummary = "Default deploy workflow runs builds on Expo Application Services (EAS) for Expo-based React Native projects.\nOtherwise native iOS and Android build steps will be used."
)

type project struct {
	projectRelDir string

	hasTest         bool
	hasYarnLockFile bool

	// non-Expo; native projects
	iosProjects    ios.DetectResult
	androidProject *gradle.Project
}

// Scanner implements the project scanner for plain React Native and Expo based projects.
type Scanner struct {
	isExpoBased bool
	projects    []project

	configDescriptors []configDescriptor
}

// NewScanner creates a new scanner instance.
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name implements ScannerInterface.Name function.
func (Scanner) Name() string {
	return scannerName
}

func isExpoBasedProject(packageJSONPth string) (bool, error) {
	packages, err := utility.ParsePackagesJSON(packageJSONPth)
	if err != nil {
		return false, fmt.Errorf("failed to parse package json file (%s): %s", packageJSONPth, err)
	}

	if _, found := packages.Dependencies["expo"]; !found {
		return false, nil
	}

	expoAppConfigFiles := []string{"app.json", "app.config.js", "app.config.ts"}
	for _, base := range expoAppConfigFiles {
		expoAppConfigPth := filepath.Join(filepath.Dir(packageJSONPth), base)
		exist, err := pathutil.IsPathExists(expoAppConfigPth)
		if err != nil {
			return false, fmt.Errorf("failed to check if Expo app config exists at: %s: %s", expoAppConfigPth, err)
		}
		if exist {
			return true, nil
		}
	}

	return false, nil
}

func hasNativeIOSProject(projectDir string, iosScanner *ios.Scanner) (bool, ios.DetectResult, error) {
	absProjectDir, err := pathutil.AbsPath(projectDir)
	if err != nil {
		return false, ios.DetectResult{}, err
	}

	iosDir := filepath.Join(absProjectDir, "ios")
	if exist, err := pathutil.IsDirExists(iosDir); err != nil || !exist {
		return false, ios.DetectResult{}, err
	}

	detected, err := iosScanner.DetectPlatform(projectDir)

	return detected, iosScanner.DetectResult, err
}

func hasNativeAndroidProject(projectDir string, androidScanner *android.Scanner) (bool, *gradle.Project, error) {
	absProjectDir, err := pathutil.AbsPath(projectDir)
	if err != nil {
		return false, nil, err
	}

	androidDir := filepath.Join(absProjectDir, "android")
	if exist, err := pathutil.IsDirExists(androidDir); err != nil || !exist {
		return false, nil, err
	}

	if detected, err := androidScanner.DetectPlatform(projectDir); err != nil || !detected {
		return false, nil, err
	}
	if len(androidScanner.Results) == 0 {
		return false, nil, err
	}

	return true, &(androidScanner.Results[0].GradleProject), nil
}

func getNativeProjects(packageJSONPth, relPackageJSONDir string) (ios.DetectResult, *gradle.Project) {
	var (
		iosScanner     = ios.NewScanner()
		androidScanner = android.NewScanner()
	)
	iosScanner.ExcludeAppIcon = true
	iosScanner.SuppressPodFileParseError = true

	projectDir := filepath.Dir(packageJSONPth)
	isIOSProject, iosProjects, err := hasNativeIOSProject(projectDir, iosScanner)
	if err != nil {
		log.TWarnf("failed to check native iOS projects: %s", err)
	}
	log.TPrintf("Found native ios project: %v", isIOSProject)

	isAndroidProject, androidProject, err := hasNativeAndroidProject(projectDir, androidScanner)
	if err != nil {
		log.TWarnf("failed to check native Android projects: %s", err)
	}
	log.TPrintf("Found native android project: %v", isAndroidProject)

	// Update native projects paths relative to search dir (otherwise would be relative to package.json dir).
	var newIosProjects []ios.Project
	for _, p := range iosProjects.Projects {
		p.RelPath = filepath.Join(relPackageJSONDir, p.RelPath)
		newIosProjects = append(newIosProjects, p)
	}
	iosProjects.Projects = newIosProjects

	androidProject.RootDirEntry.RelPath = filepath.Join(relPackageJSONDir, androidProject.RootDirEntry.RelPath)

	return iosProjects, androidProject
}

// DetectPlatform implements ScannerInterface.DetectPlatform function.
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	log.TInfof("Collecting package.json files")

	packageJSONPths, err := CollectPackageJSONFiles(searchDir)
	if err != nil {
		return false, err
	}

	log.TPrintf("%d package.json file detected", len(packageJSONPths))
	for _, path := range packageJSONPths {
		log.TPrintf("- %s", path)
	}

	log.TPrintf("Filtering relevant package.json files")
	for _, packageJSONPth := range packageJSONPths {
		log.TPrintf("Checking: %s", packageJSONPth)

		isExpoBased, err := isExpoBasedProject(packageJSONPth)
		if err != nil {
			log.TWarnf("failed to determine if project is Expo based: %s", err)
		}

		log.TPrintf("Project uses expo: %v", isExpoBased)

		// determine workdir
		packageJSONDir := filepath.Dir(packageJSONPth)
		relPackageJSONDir, err := utility.RelPath(searchDir, packageJSONDir)
		if err != nil {
			return false, fmt.Errorf("failed to get relative package.json dir path: %s", err)
		}

		var (
			iosProjects    ios.DetectResult
			androidProject *gradle.Project
		)
		if !isExpoBased {
			iosProjects, androidProject = getNativeProjects(packageJSONPth, relPackageJSONDir)
			if len(iosProjects.Projects) == 0 && androidProject == nil {
				continue
			}
		}

		// determine Js dependency manager
		hasYarnLockFile, err := containsYarnLock(filepath.Dir(packageJSONPth))
		if err != nil {
			return false, err
		}
		log.TPrintf("Js dependency manager for %s is yarn: %t", packageJSONPth, hasYarnLockFile)

		packages, err := utility.ParsePackagesJSON(packageJSONPth)
		if err != nil {
			return false, err
		}

		_, hasTests := packages.Scripts["test"]
		log.TPrintf("Test script found in package.json: %v", hasTests)

		result := project{
			projectRelDir:   relPackageJSONDir,
			hasTest:         hasTests,
			hasYarnLockFile: hasYarnLockFile,
			iosProjects:     iosProjects,
			androidProject:  androidProject,
		}

		if isExpoBased {
			scanner.projects = []project{result}
			scanner.isExpoBased = true

			break
		}

		scanner.projects = append(scanner.projects, result)
	}

	if len(scanner.projects) == 0 {
		return false, nil
	}

	return true, nil
}

// Options implements ScannerInterface.Options function.
func (scanner *Scanner) Options() (options models.OptionNode, allWarnings models.Warnings, icons models.Icons, err error) {
	if scanner.isExpoBased {
		options = scanner.expoOptions()
	} else {
		projectRootOption := models.NewOption(projectDirInputTitle, projectDirInputSummary, projectDirInputEnvKey, models.TypeSelector)
		options = *projectRootOption

		for _, project := range scanner.projects {
			options, warnings := scanner.options(project)
			allWarnings = append(allWarnings, warnings...)

			projectRootOption.AddOption(project.projectRelDir, &options)
		}
	}

	return
}

func (scanner *Scanner) Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	if scanner.isExpoBased {
		return scanner.expoConfigs(scanner.projects[0], sshKeyActivation)
	}

	return scanner.configs(sshKeyActivation)
}

// DefaultOptions implements ScannerInterface.DefaultOptions function.
func (scanner *Scanner) DefaultOptions() models.OptionNode {
	expoOption := models.NewOption(isExpoBasedProjectInputTitle, isExpoBasedProjectInputSummary, "", models.TypeSelector)

	expoDefaultOptions := scanner.expoDefaultOptions()
	expoOption.AddOption("yes", &expoDefaultOptions)

	defaultOptions := scanner.defaultOptions()
	expoOption.AddOption("no", &defaultOptions)

	return *expoOption
}

// DefaultConfigs implements ScannerInterface.DefaultConfigs function.
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	configs, err := scanner.defaultConfigs()
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	for k, v := range configs {
		configMap[k] = v
	}

	expoConfigs, err := scanner.expoDefaultConfigs()
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	for k, v := range expoConfigs {
		configMap[k] = v
	}

	return configMap, nil
}

// ExcludedScannerNames implements ScannerInterface.ExcludedScannerNames function.
func (Scanner) ExcludedScannerNames() []string {
	return []string{
		string(ios.XcodeProjectTypeIOS),
		string(ios.XcodeProjectTypeMacOS),
		android.ScannerName,
		nodejs.ScannerName,
		java.ProjectType,
	}
}
