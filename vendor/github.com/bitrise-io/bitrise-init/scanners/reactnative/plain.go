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
	if len(scanner.androidProjects) > 0 {
		androidOptions = models.NewOption(android.ProjectLocationInputTitle, android.ProjectLocationInputSummary, android.ProjectLocationInputEnvKey, models.TypeSelector)
		for _, project := range scanner.androidProjects {
			warnings = append(warnings, project.Warnings...)

			// This config option is removed when merging with ios config. This way no change is needed for the working options merging.
			configOption := models.NewConfigOption("glue-only", nil)
			moduleOption := models.NewOption(android.ModuleInputTitle, android.ModuleInputSummary, android.ModuleInputEnvKey, models.TypeUserInput)
			variantOption := models.NewOption(android.VariantInputTitle, android.VariantInputSummary, android.VariantInputEnvKey, models.TypeOptionalUserInput)

			androidOptions.AddOption(project.RelPath, moduleOption)
			moduleOption.AddOption("app", variantOption)
			variantOption.AddConfig("", configOption)
		}
	}

	// ios options
	var iosOptions *models.OptionNode
	iosDir := filepath.Join(projectDir, "ios")
	if exist, err := pathutil.IsDirExists(iosDir); err != nil {
		return models.OptionNode{}, warnings, err
	} else if exist {
		scanner.iosScanner.SuppressPodFileParseError = true
		if detected, err := scanner.iosScanner.DetectPlatform(scanner.searchDir); err != nil {
			return models.OptionNode{}, warnings, err
		} else if detected {
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

				configName := configName(androidOptions != nil, true, scanner.hasTest)
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
	androidOptions := (&android.Scanner{}).DefaultOptions()
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
	hasAndroidProject := len(scanner.androidProjects) > 0
	if hasAndroidProject {
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

			configName := configName(hasAndroidProject, true, scanner.hasTest)
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

		configName := configName(hasAndroidProject, false, scanner.hasTest)
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
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, getTestSteps("", true, true)...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepListV2(false)...)

	// deploy
	configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepListV2(steps.PrepareListParams{
		ShouldIncludeCache:       false,
		ShouldIncludeActivateSSH: true,
	})...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, getTestSteps("", true, true)...)

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

func getTestSteps(workDir string, hasYarnLockFile, hasTest bool) []bitriseModels.StepListItemModel {
	var testSteps []bitriseModels.StepListItemModel

	if hasYarnLockFile {
		testSteps = append(testSteps, steps.YarnStepListItem("install", workDir))
		if hasTest {
			testSteps = append(testSteps, steps.YarnStepListItem("test", workDir))
		}
	} else {
		testSteps = append(testSteps, steps.NpmStepListItem("install", workDir))
		if hasTest {
			testSteps = append(testSteps, steps.NpmStepListItem("test", workDir))
		}
	}

	return testSteps
}

func (scanner *Scanner) getTestSteps(workDir string) []bitriseModels.StepListItemModel {
	return getTestSteps(workDir, scanner.hasYarnLockFile, scanner.hasTest)
}
