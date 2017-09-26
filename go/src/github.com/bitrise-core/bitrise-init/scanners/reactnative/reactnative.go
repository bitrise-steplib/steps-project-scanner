package reactnative

import (
	"errors"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners/android"
	"github.com/bitrise-core/bitrise-init/scanners/cordova"
	"github.com/bitrise-core/bitrise-init/scanners/ios"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

// Name ...
const Name = "react-native"

const (
	workDirInputKey = "workdir"
)

// Scanner ...
type Scanner struct {
	searchDir      string
	iosScanner     *ios.Scanner
	androidScanner *android.Scanner
	hasNPMTest     bool
	packageJSONPth string
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (Scanner) Name() string {
	return Name
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	scanner.searchDir = searchDir

	log.Infoft("Collect package.json files")

	packageJSONPths, err := CollectPackageJSONFiles(searchDir)
	if err != nil {
		return false, err
	}

	log.Printft("%d package.json file detected", len(packageJSONPths))

	log.Infoft("Filter relevant package.json files")

	relevantPackageJSONPths := []string{}
	iosScanner := ios.NewScanner()
	androidScanner := android.NewScanner()
	for _, packageJSONPth := range packageJSONPths {
		log.Printft("checking: %s", packageJSONPth)

		projectDir := filepath.Dir(packageJSONPth)

		iosProjectDetected := false
		iosDir := filepath.Join(projectDir, "ios")
		if exist, err := pathutil.IsDirExists(iosDir); err != nil {
			return false, err
		} else if exist {
			if detected, err := iosScanner.DetectPlatform(scanner.searchDir); err != nil {
				return false, err
			} else if detected {
				iosProjectDetected = true
			}
		}

		androidProjectDetected := false
		androidDir := filepath.Join(projectDir, "android")
		if exist, err := pathutil.IsDirExists(androidDir); err != nil {
			return false, err
		} else if exist {
			if detected, err := androidScanner.DetectPlatform(scanner.searchDir); err != nil {
				return false, err
			} else if detected {
				androidProjectDetected = true
			}
		}

		if iosProjectDetected || androidProjectDetected {
			relevantPackageJSONPths = append(relevantPackageJSONPths, packageJSONPth)
		} else {
			log.Warnft("no ios nor android project found, skipping package.json file")
		}
	}

	if len(relevantPackageJSONPths) == 0 {
		return false, nil
	}

	scanner.packageJSONPth = relevantPackageJSONPths[0]

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	warnings := models.Warnings{}

	var rootOption models.OptionModel

	// react options
	packages, err := cordova.ParsePackagesJSON(scanner.packageJSONPth)
	if err != nil {
		return models.OptionModel{}, warnings, err
	}

	hasNPMTest := false
	if _, found := packages.Scripts["test"]; found {
		hasNPMTest = true
		scanner.hasNPMTest = true
	}

	projectDir := filepath.Dir(scanner.packageJSONPth)

	// android options
	var androidOptions *models.OptionModel
	androidDir := filepath.Join(projectDir, "android")
	if exist, err := pathutil.IsDirExists(androidDir); err != nil {
		return models.OptionModel{}, warnings, err
	} else if exist {
		androidScanner := android.NewScanner()

		if detected, err := androidScanner.DetectPlatform(scanner.searchDir); err != nil {
			return models.OptionModel{}, warnings, err
		} else if detected {
			options, warns, err := androidScanner.Options()
			warnings = append(warnings, warns...)
			if err != nil {
				return models.OptionModel{}, warnings, err
			}

			androidOptions = &options
			scanner.androidScanner = androidScanner
		}
	}

	// ios options
	var iosOptions *models.OptionModel
	iosDir := filepath.Join(projectDir, "ios")
	if exist, err := pathutil.IsDirExists(iosDir); err != nil {
		return models.OptionModel{}, warnings, err
	} else if exist {
		iosScanner := ios.NewScanner()

		if detected, err := iosScanner.DetectPlatform(scanner.searchDir); err != nil {
			return models.OptionModel{}, warnings, err
		} else if detected {
			options, warns, err := iosScanner.Options()
			warnings = append(warnings, warns...)
			if err != nil {
				return models.OptionModel{}, warnings, err
			}

			iosOptions = &options
			scanner.iosScanner = iosScanner
		}
	}

	if androidOptions == nil && iosOptions == nil {
		return models.OptionModel{}, warnings, errors.New("no ios nor android project detected")
	}
	// ---

	if androidOptions != nil {
		if iosOptions == nil {
			// we only found an android project
			// we need to update the config names
			lastChilds := androidOptions.LastChilds()
			for _, child := range lastChilds {
				for _, child := range child.ChildOptionMap {
					if child.Config == "" {
						return models.OptionModel{}, warnings, fmt.Errorf("no config for option: %s", child.String())
					}

					configName := configName(true, false, hasNPMTest)
					child.Config = configName
				}
			}
		} else {
			// we have both ios and android projects
			// we need to remove the android option's config names,
			// since ios options will hold them
			androidOptions.RemoveConfigs()
		}

		rootOption = *androidOptions
	}

	if iosOptions != nil {
		lastChilds := iosOptions.LastChilds()
		for _, child := range lastChilds {
			for _, child := range child.ChildOptionMap {
				if child.Config == "" {
					return models.OptionModel{}, warnings, fmt.Errorf("no config for option: %s", child.String())
				}

				configName := configName(scanner.androidScanner != nil, true, hasNPMTest)
				child.Config = configName
			}
		}

		if androidOptions == nil {
			// we only found an ios project
			rootOption = *iosOptions
		} else {
			// we have both ios and android projects
			// we attach ios options to the android options
			rootOption.AttachToLastChilds(iosOptions)
		}

	}

	return rootOption, warnings, nil
}

// DefaultOptions ...
func (Scanner) DefaultOptions() models.OptionModel {
	gradleFileOption := models.NewOption(android.GradleFileInputTitle, android.GradleFileInputEnvKey)

	gradlewPthOption := models.NewOption(android.GradlewPathInputTitle, android.GradlewPathInputEnvKey)
	gradleFileOption.AddOption("_", gradlewPthOption)

	projectPathOption := models.NewOption(ios.ProjectPathInputTitle, ios.ProjectPathInputEnvKey)
	gradlewPthOption.AddOption("_", projectPathOption)

	schemeOption := models.NewOption(ios.SchemeInputTitle, ios.SchemeInputEnvKey)
	projectPathOption.AddOption("_", schemeOption)

	configOption := models.NewConfigOption(defaultConfigName())
	schemeOption.AddConfig("_", configOption)

	return *gradleFileOption
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	packageJSONDir := filepath.Dir(scanner.packageJSONPth)
	relPackageJSONDir, err := utility.RelPath(scanner.searchDir, packageJSONDir)
	if err != nil {
		return models.BitriseConfigMap{}, fmt.Errorf("Failed to get relative config.xml dir path, error: %s", err)
	}
	if relPackageJSONDir == "." {
		// config.xml placed in the search dir, no need to change-dir in the workflows
		relPackageJSONDir = ""
	}

	workdirEnvList := []envmanModels.EnvironmentItemModel{}
	if relPackageJSONDir != "" {
		workdirEnvList = append(workdirEnvList, envmanModels.EnvironmentItemModel{workDirInputKey: relPackageJSONDir})
	}

	if scanner.hasNPMTest {
		configBuilder := models.NewDefaultConfigBuilder()

		// ci
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "test"})...))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

		// cd
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(false)...)
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))

		// android cd
		if scanner.androidScanner != nil {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem())
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.GradleRunnerStepListItem(
				envmanModels.EnvironmentItemModel{android.GradleFileInputKey: "$" + android.GradleFileInputEnvKey},
				envmanModels.EnvironmentItemModel{android.GradleTaskInputKey: "assembleRelease"},
				envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.GradlewPathInputEnvKey},
			))
		}

		// ios cd
		if scanner.iosScanner != nil {
			for _, descriptor := range scanner.iosScanner.ConfigDescriptors {
				configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

				if descriptor.MissingSharedSchemes {
					configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.RecreateUserSchemesStepListItem(
						envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
					))
				}

				if descriptor.HasPodfile {
					configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CocoapodsInstallStepListItem())
				}

				if descriptor.CarthageCommand != "" {
					configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CarthageStepListItem(
						envmanModels.EnvironmentItemModel{ios.CarthageCommandInputKey: descriptor.CarthageCommand},
					))
				}

				configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(
					envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
					envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
					envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
				))

				configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

				bitriseDataModel, err := configBuilder.Generate(Name)
				if err != nil {
					return models.BitriseConfigMap{}, err
				}

				data, err := yaml.Marshal(bitriseDataModel)
				if err != nil {
					return models.BitriseConfigMap{}, err
				}

				configName := configName(scanner.androidScanner != nil, true, true)
				configMap[configName] = string(data)
			}
		} else {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

			bitriseDataModel, err := configBuilder.Generate(Name)
			if err != nil {
				return models.BitriseConfigMap{}, err
			}

			data, err := yaml.Marshal(bitriseDataModel)
			if err != nil {
				return models.BitriseConfigMap{}, err
			}

			configName := configName(scanner.androidScanner != nil, false, true)
			configMap[configName] = string(data)
		}
	} else {
		configBuilder := models.NewDefaultConfigBuilder()

		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))

		if scanner.androidScanner != nil {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.InstallMissingAndroidToolsStepListItem())
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.GradleRunnerStepListItem(
				envmanModels.EnvironmentItemModel{android.GradleFileInputKey: "$" + android.GradleFileInputEnvKey},
				envmanModels.EnvironmentItemModel{android.GradleTaskInputKey: "assembleRelease"},
				envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.GradlewPathInputEnvKey},
			))
		}

		if scanner.iosScanner != nil {
			for _, descriptor := range scanner.iosScanner.ConfigDescriptors {
				configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

				if descriptor.MissingSharedSchemes {
					configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.RecreateUserSchemesStepListItem(
						envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
					))
				}

				if descriptor.HasPodfile {
					configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CocoapodsInstallStepListItem())
				}

				if descriptor.CarthageCommand != "" {
					configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CarthageStepListItem(
						envmanModels.EnvironmentItemModel{ios.CarthageCommandInputKey: descriptor.CarthageCommand},
					))
				}

				configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.XcodeArchiveStepListItem(
					envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
					envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
					envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
				))

				configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

				bitriseDataModel, err := configBuilder.Generate(Name)
				if err != nil {
					return models.BitriseConfigMap{}, err
				}

				data, err := yaml.Marshal(bitriseDataModel)
				if err != nil {
					return models.BitriseConfigMap{}, err
				}

				configName := configName(scanner.androidScanner != nil, true, false)
				configMap[configName] = string(data)
			}
		} else {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

			bitriseDataModel, err := configBuilder.Generate(Name)
			if err != nil {
				return models.BitriseConfigMap{}, err
			}

			data, err := yaml.Marshal(bitriseDataModel)
			if err != nil {
				return models.BitriseConfigMap{}, err
			}

			configName := configName(scanner.androidScanner != nil, false, false)
			configMap[configName] = string(data)
		}
	}

	return configMap, nil
}

// DefaultConfigs ...
func (Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	// ci
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{"command": "install"}))
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{"command": "test"}))
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

	// cd
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(false)...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{"command": "install"}))

	// android
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem())
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.GradleRunnerStepListItem(
		envmanModels.EnvironmentItemModel{android.GradleFileInputKey: "$" + android.GradleFileInputEnvKey},
		envmanModels.EnvironmentItemModel{android.GradleTaskInputKey: "assembleRelease"},
		envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.GradlewPathInputEnvKey},
	))

	// ios
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(
		envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
	))

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

	bitriseDataModel, err := configBuilder.Generate(Name)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(bitriseDataModel)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := defaultConfigName()
	configMap := models.BitriseConfigMap{
		configName: string(data),
	}
	return configMap, nil
}

// ExcludedScannerNames ...
func (Scanner) ExcludedScannerNames() []string {
	return []string{
		string(ios.XcodeProjectTypeIOS),
		string(ios.XcodeProjectTypeMacOS),
		android.ScannerName,
	}
}
