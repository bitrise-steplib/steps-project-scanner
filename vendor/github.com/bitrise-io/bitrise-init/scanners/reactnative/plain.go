package reactnative

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/steps"
	"github.com/bitrise-io/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/pathutil"
	"gopkg.in/yaml.v2"
)

const (
	defaultConfigName = "default-react-native-config"
)

// configName generates a config name based on the inputs.
func configName(hasAndroidProject, hasIosProject, hasTest bool) string {
	name := "react-native"
	if hasAndroidProject {
		name += "-android"
	}
	if hasIosProject {
		name += "-ios"
	}
	if hasTest {
		name += "-test"
	}
	return name + "-config"
}

// options implements ScannerInterface.Options function for plain React Native projects.
func (scanner *Scanner) options() (models.OptionNode, models.Warnings, error) {
	warnings := models.Warnings{}
	var rootOption models.OptionNode
	projectDir := filepath.Dir(scanner.packageJSONPth)

	// android options
	var androidOptions *models.OptionNode
	androidDir := filepath.Join(projectDir, "android")
	if exist, err := pathutil.IsDirExists(androidDir); err != nil {
		return models.OptionNode{}, warnings, err
	} else if exist {
		if detected, err := scanner.androidScanner.DetectPlatform(scanner.searchDir); err != nil {
			return models.OptionNode{}, warnings, err
		} else if detected {
			// only the first match we need
			scanner.androidScanner.ExcludeTest = true
			scanner.androidScanner.ProjectRoots = []string{scanner.androidScanner.ProjectRoots[0]}

			options, warns, _, err := scanner.androidScanner.Options()
			warnings = append(warnings, warns...)
			if err != nil {
				return models.OptionNode{}, warnings, err
			}

			androidOptions = &options
		}
	}

	// ios options
	var iosOptions *models.OptionNode
	iosDir := filepath.Join(projectDir, "ios")
	if exist, err := pathutil.IsDirExists(iosDir); err != nil {
		return models.OptionNode{}, warnings, err
	} else if exist {
		if detected, err := scanner.iosScanner.DetectPlatform(scanner.searchDir); err != nil {
			return models.OptionNode{}, warnings, err
		} else if detected {
			scanner.iosScanner.SuppressPodFileParseError = true
			options, warns, _, err := scanner.iosScanner.Options()
			warnings = append(warnings, warns...)
			if err != nil {
				return models.OptionNode{}, warnings, err
			}

			iosOptions = &options
		}
	}

	if androidOptions == nil && iosOptions == nil {
		return models.OptionNode{}, warnings, errors.New("no ios nor android project detected")
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
						return models.OptionNode{}, warnings, fmt.Errorf("no config for option: %s", child.String())
					}

					configName := configName(true, false, scanner.hasTest)
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
					return models.OptionNode{}, warnings, fmt.Errorf("no config for option: %s", child.String())
				}

				configName := configName(scanner.androidScanner != nil, true, scanner.hasTest)
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

// defaultOptions implements ScannerInterface.DefaultOptions function for plain React Native projects.
func (scanner *Scanner) defaultOptions() models.OptionNode {
	androidOptions := (&android.Scanner{ExcludeTest: true}).DefaultOptions()
	androidOptions.RemoveConfigs()

	iosOptions := (&ios.Scanner{}).DefaultOptions()
	for _, child := range iosOptions.LastChilds() {
		for _, child := range child.ChildOptionMap {
			child.Config = defaultConfigName
		}
	}

	androidOptions.AttachToLastChilds(&iosOptions)

	return androidOptions
}

// configs implements ScannerInterface.Configs function for plain React Native projects.
func (scanner *Scanner) configs() (models.BitriseConfigMap, error) {
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

	if scanner.hasTest {
		configBuilder := models.NewDefaultConfigBuilder()

		// ci
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
		if scanner.hasYarnLockFile {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "test"})...))
		} else {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "test"})...))
		}
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

		// cd
		configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(false)...)
		if scanner.hasYarnLockFile {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		} else {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		}

		// android cd
		if scanner.androidScanner != nil {
			projectLocationEnv := "$" + android.ProjectLocationInputEnvKey

			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
				envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
			))
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
				envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: projectLocationEnv},
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
					envmanModels.EnvironmentItemModel{ios.DistributionMethodInputKey: "$" + ios.DistributionMethodEnvKey},
					envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
				))

				configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

				bitriseDataModel, err := configBuilder.Generate(scannerName)
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

			bitriseDataModel, err := configBuilder.Generate(scannerName)
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
		configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)

		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))

		if scanner.androidScanner != nil {
			projectLocationEnv := "$" + android.ProjectLocationInputEnvKey

			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
				envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
			))
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
				envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: projectLocationEnv},
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
					envmanModels.EnvironmentItemModel{ios.DistributionMethodInputKey: "$" + ios.DistributionMethodEnvKey},
					envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
				))

				configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

				bitriseDataModel, err := configBuilder.Generate(scannerName)
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

			bitriseDataModel, err := configBuilder.Generate(scannerName)
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

// defaultConfigs implements ScannerInterface.DefaultConfigs function for plain React Native projects.
func (scanner *Scanner) defaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	// ci
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{"command": "install"}))
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{"command": "test"}))
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

	// cd
	configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(false)...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{"command": "install"}))

	// android
	projectLocationEnv := "$" + android.ProjectLocationInputEnvKey

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
		envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
	))
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
		envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: projectLocationEnv},
	))

	// ios
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(
		envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.DistributionMethodInputKey: "$" + ios.DistributionMethodEnvKey},
		envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
	))

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

	bitriseDataModel, err := configBuilder.Generate(scannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(bitriseDataModel)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := defaultConfigName
	configMap := models.BitriseConfigMap{
		configName: string(data),
	}

	return configMap, nil
}
