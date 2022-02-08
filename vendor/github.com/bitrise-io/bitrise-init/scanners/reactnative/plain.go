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
	bitriseModels "github.com/bitrise-io/bitrise/models"
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
func (scanner *Scanner) configs(isPrivateRepo bool) (models.BitriseConfigMap, error) {
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

	configBuilder := models.NewDefaultConfigBuilder()

	// ci
	primaryDescription := primaryWorkflowNoTestsDescription
	if scanner.hasTest {
		primaryDescription = primaryWorkflowDescription
	}

	configBuilder.SetWorkflowDescriptionTo(models.PrimaryWorkflowID, primaryDescription)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepListV2(steps.PrepareListParams{
		ShouldIncludeCache:       false,
		ShouldIncludeActivateSSH: isPrivateRepo,
	})...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, scanner.getTestSteps(relPackageJSONDir)...)

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepListV2(false)...)

	// cd
	configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepListV2(steps.PrepareListParams{
		ShouldIncludeCache:       false,
		ShouldIncludeActivateSSH: isPrivateRepo,
	})...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, scanner.getTestSteps(relPackageJSONDir)...)

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
				envmanModels.EnvironmentItemModel{ios.AutomaticCodeSigningInputKey: ios.AutomaticCodeSigningInputAPIKeyValue},
			))

			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepListV2(false)...)

			bitriseDataModel, err := configBuilder.Generate(scannerName)
			if err != nil {
				return models.BitriseConfigMap{}, err
			}

			data, err := yaml.Marshal(bitriseDataModel)
			if err != nil {
				return models.BitriseConfigMap{}, err
			}

			configName := configName(scanner.androidScanner != nil, true, scanner.hasTest)
			configMap[configName] = string(data)
		}
	} else {
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepListV2(false)...)

		bitriseDataModel, err := configBuilder.Generate(scannerName)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(bitriseDataModel)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		configName := configName(scanner.androidScanner != nil, false, scanner.hasTest)
		configMap[configName] = string(data)
	}

	return configMap, nil
}

// defaultConfigs implements ScannerInterface.DefaultConfigs function for plain React Native projects.
func (scanner *Scanner) defaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	// primary
	configBuilder.SetWorkflowDescriptionTo(models.PrimaryWorkflowID, primaryWorkflowDescription)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepListV2(steps.PrepareListParams{
		ShouldIncludeCache:       false,
		ShouldIncludeActivateSSH: true,
	})...)
	// Assuming project uses yarn and has tests
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.YarnStepListItem(envmanModels.EnvironmentItemModel{"command": "install"}))
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.YarnStepListItem(envmanModels.EnvironmentItemModel{"command": "test"}))
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepListV2(false)...)

	// deploy
	configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepListV2(steps.PrepareListParams{
		ShouldIncludeCache:       false,
		ShouldIncludeActivateSSH: true,
	})...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.YarnStepListItem(envmanModels.EnvironmentItemModel{"command": "install"}))
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.YarnStepListItem(envmanModels.EnvironmentItemModel{"command": "test"}))

	// android
	projectLocationEnv := "$" + android.ProjectLocationInputEnvKey

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
		envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
	))
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
		envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: projectLocationEnv},
	))

	// ios
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(
		envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.DistributionMethodInputKey: "$" + ios.DistributionMethodEnvKey},
		envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
		envmanModels.EnvironmentItemModel{ios.AutomaticCodeSigningInputKey: ios.AutomaticCodeSigningInputAPIKeyValue},
	))

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepListV2(false)...)

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

func (scanner *Scanner) getTestSteps(workDir string) []bitriseModels.StepListItemModel {
	var (
		testSteps      = []bitriseModels.StepListItemModel{}
		workdirEnvList = []envmanModels.EnvironmentItemModel{}
	)

	if workDir != "" {
		workdirEnvList = append(workdirEnvList, envmanModels.EnvironmentItemModel{workDirInputKey: workDir})
	}

	if scanner.hasYarnLockFile {
		testSteps = append(testSteps, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		if scanner.hasTest {
			testSteps = append(testSteps, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "test"})...))
		}
	} else {
		testSteps = append(testSteps, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		if scanner.hasTest {
			testSteps = append(testSteps, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "test"})...))
		}
	}

	return testSteps
}
