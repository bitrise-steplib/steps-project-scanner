package reactnative

import (
	"fmt"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/steps"
	bitriseModels "github.com/bitrise-io/bitrise/v2/models"
	envmanModels "github.com/bitrise-io/envman/v2/models"
	"gopkg.in/yaml.v2"
)

const (
	defaultConfigName = "default-react-native-config"

	defaultModule  = "app"
	defaultVariant = "Debug"
)

type configDescriptor struct {
	hasIOS, hasAndroid bool
	hasTest            bool
	hasYarnLockFile    bool
	ios                ios.ConfigDescriptor
}

func (d configDescriptor) configName() string {
	name := "react-native"
	if d.hasAndroid {
		name += "-android"
	}
	if d.hasIOS {
		name += "-ios"
		if d.ios.HasPodfile {
			name += "-pod"
		}
		if d.ios.CarthageCommand != "" {
			name += "-carthage"
		}
	}
	if d.hasTest {
		name += "-test"
	}
	if d.hasYarnLockFile {
		name += "-yarn"
	}

	return name + "-config"
}

func generateIOSOptions(result ios.DetectResult, hasAndroid, hasTests, hasYarnLockFile bool) (*models.OptionNode, models.Warnings, []configDescriptor) {
	var (
		warnings    models.Warnings
		descriptors []configDescriptor
	)

	projectPathOption := models.NewOption(ios.ProjectPathInputTitle, ios.ProjectPathInputSummary, ios.ProjectPathInputEnvKey, models.TypeSelector)
	for _, project := range result.Projects {
		warnings = append(warnings, project.Warnings...)

		schemeOption := models.NewOption(ios.SchemeInputTitle, ios.SchemeInputSummary, ios.SchemeInputEnvKey, models.TypeSelector)
		projectPathOption.AddOption(project.RelPath, schemeOption)

		for _, scheme := range project.Schemes {
			exportMethodOption := models.NewOption(ios.DistributionMethodInputTitle, ios.DistributionMethodInputSummary, ios.DistributionMethodEnvKey, models.TypeSelector)
			schemeOption.AddOption(scheme.Name, exportMethodOption)

			for _, exportMethod := range ios.IosExportMethods {
				iosConfig := ios.NewConfigDescriptor(
					project.IsPodWorkspace,
					project.CarthageCommand,
					scheme.HasXCTests,
					scheme.HasAppClip,
					result.HasSPMDependencies,
					false,
					exportMethod)
				descriptor := configDescriptor{
					hasIOS:          true,
					hasAndroid:      hasAndroid,
					hasTest:         hasTests,
					hasYarnLockFile: hasYarnLockFile,
					ios:             iosConfig,
				}
				descriptors = append(descriptors, descriptor)

				exportMethodOption.AddConfig(exportMethod, models.NewConfigOption(descriptor.configName(), nil))
			}
		}
	}

	return projectPathOption, warnings, descriptors
}

// options implements ScannerInterface.Options function for plain React Native projects.
func (scanner *Scanner) options(project project) (models.OptionNode, models.Warnings) {
	var (
		rootOption     models.OptionNode
		allDescriptors []configDescriptor
		warnings       models.Warnings
	)

	// Android
	if len(project.androidProjects) > 0 {
		androidOptions := models.NewOption(android.ProjectLocationInputTitle, android.ProjectLocationInputSummary, android.ProjectLocationInputEnvKey, models.TypeSelector)
		rootOption = *androidOptions

		for _, androidProject := range project.androidProjects {
			warnings = append(warnings, androidProject.Warnings...)

			moduleOption := models.NewOption(android.ModuleInputTitle, android.ModuleInputSummary, android.ModuleInputEnvKey, models.TypeUserInput)
			variantOption := models.NewOption(android.VariantInputTitle, android.VariantInputSummary, android.VariantInputEnvKey, models.TypeOptionalUserInput)

			androidOptions.AddOption(androidProject.RelPath, moduleOption)
			moduleOption.AddOption(defaultModule, variantOption)

			if len(project.iosProjects.Projects) == 0 {
				descriptor := configDescriptor{
					hasAndroid:      true,
					hasTest:         project.hasTest,
					hasYarnLockFile: project.hasYarnLockFile,
				}
				allDescriptors = append(allDescriptors, descriptor)

				variantOption.AddConfig(defaultVariant, models.NewConfigOption(descriptor.configName(), nil))

				continue
			}

			iosOptions, iosWarnings, descriptors := generateIOSOptions(project.iosProjects, true, project.hasTest, project.hasYarnLockFile)
			warnings = append(warnings, iosWarnings...)
			allDescriptors = append(allDescriptors, descriptors...)

			variantOption.AddOption(defaultVariant, iosOptions)
		}
	} else {
		options, iosWarnings, descriptors := generateIOSOptions(project.iosProjects, false, project.hasTest, project.hasYarnLockFile)
		rootOption = *options
		warnings = append(warnings, iosWarnings...)
		allDescriptors = append(allDescriptors, descriptors...)
	}

	scanner.configDescriptors = removeDuplicatedConfigDescriptors(append(scanner.configDescriptors, allDescriptors...))

	return rootOption, warnings
}

// defaultOptions implements ScannerInterface.DefaultOptions function for plain React Native projects.
func (scanner *Scanner) defaultOptions() models.OptionNode {
	androidOptions := models.NewOption(android.ProjectLocationInputTitle, android.ProjectLocationInputSummary, android.ProjectLocationInputEnvKey, models.TypeUserInput)
	moduleOption := models.NewOption(android.ModuleInputTitle, android.ModuleInputSummary, android.ModuleInputEnvKey, models.TypeUserInput)
	variantOption := models.NewOption(android.VariantInputTitle, android.VariantInputSummary, android.VariantInputEnvKey, models.TypeOptionalUserInput)

	androidOptions.AddOption("android", moduleOption)
	moduleOption.AddOption(defaultModule, variantOption)

	projectPathOption := models.NewOption(ios.ProjectPathInputTitle, ios.ProjectPathInputSummary, ios.ProjectPathInputEnvKey, models.TypeUserInput)
	schemeOption := models.NewOption(ios.SchemeInputTitle, ios.SchemeInputSummary, ios.SchemeInputEnvKey, models.TypeUserInput)

	variantOption.AddOption(defaultVariant, projectPathOption)
	projectPathOption.AddOption(models.UserInputOptionDefaultValue, schemeOption)

	exportMethodOption := models.NewOption(ios.DistributionMethodInputTitle, ios.DistributionMethodInputSummary, ios.DistributionMethodEnvKey, models.TypeSelector)
	for _, exportMethod := range ios.IosExportMethods {
		schemeOption.AddOption(models.UserInputOptionDefaultValue, exportMethodOption)

		exportMethodOption.AddConfig(exportMethod, models.NewConfigOption(defaultConfigName, nil))
	}

	return *androidOptions
}

func (scanner *Scanner) configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	if len(scanner.configDescriptors) == 0 {
		return models.BitriseConfigMap{}, fmt.Errorf("invalid state, no config descriptors found")
	}

	for _, descriptor := range scanner.configDescriptors {
		configBuilder := models.NewDefaultConfigBuilder()

		testSteps := getTestSteps("$"+projectDirInputEnvKey, descriptor.hasYarnLockFile, descriptor.hasTest)
		// ci
		primaryDescription := primaryWorkflowNoTestsDescription
		if descriptor.hasTest {
			primaryDescription = primaryWorkflowDescription
		}

		configBuilder.SetWorkflowDescriptionTo(models.PrimaryWorkflowID, primaryDescription)
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: sshKeyActivation,
		})...)
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.RestoreNPMCache())
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, testSteps...)
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.SaveNPMCache())
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList()...)

		// cd
		configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
			SSHKeyActivation: sshKeyActivation,
		})...)
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, testSteps...)

		// android cd
		if descriptor.hasAndroid {
			projectLocationEnv := "$" + android.ProjectLocationInputEnvKey

			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
				envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
			))
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
				envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: projectLocationEnv},
				envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
				envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
			))
		}

		// ios cd
		if descriptor.hasIOS {
			if descriptor.ios.HasPodfile {
				configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CocoapodsInstallStepListItem())
			}

			if descriptor.ios.CarthageCommand != "" {
				configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CarthageStepListItem(
					envmanModels.EnvironmentItemModel{ios.CarthageCommandInputKey: descriptor.ios.CarthageCommand},
				))
			}

			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(
				envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
				envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
				envmanModels.EnvironmentItemModel{ios.DistributionMethodInputKey: "$" + ios.DistributionMethodEnvKey},
				envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
				envmanModels.EnvironmentItemModel{ios.AutomaticCodeSigningInputKey: ios.AutomaticCodeSigningInputAPIKeyValue},
			))
		}

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList()...)

		bitriseDataModel, err := configBuilder.Generate(scannerName)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(bitriseDataModel)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		configMap[descriptor.configName()] = string(data)
	}

	return configMap, nil
}

func (scanner *Scanner) defaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	// primary
	configBuilder.SetWorkflowDescriptionTo(models.PrimaryWorkflowID, primaryWorkflowDescription)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		SSHKeyActivation: models.SSHKeyActivationConditional,
	})...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.RestoreNPMCache())
	// Assuming project uses yarn and has tests
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, getTestSteps("", true, true)...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.SaveNPMCache())
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList()...)

	// deploy
	configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		SSHKeyActivation: models.SSHKeyActivationConditional,
	})...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, getTestSteps("", true, true)...)

	// android
	projectLocationEnv := "$" + android.ProjectLocationInputEnvKey

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
		envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
	))
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
		envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: projectLocationEnv},
		envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
		envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
	))

	// ios
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(
		envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.DistributionMethodInputKey: "$" + ios.DistributionMethodEnvKey},
		envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
		envmanModels.EnvironmentItemModel{ios.AutomaticCodeSigningInputKey: ios.AutomaticCodeSigningInputAPIKeyValue},
	))

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList()...)

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

func removeDuplicatedConfigDescriptors(configDescriptors []configDescriptor) []configDescriptor {
	descritorNameMap := map[string]configDescriptor{}
	for _, descriptor := range configDescriptors {
		name := descriptor.configName()
		descritorNameMap[name] = descriptor
	}

	descriptors := []configDescriptor{}
	for _, descriptor := range descritorNameMap {
		descriptors = append(descriptors, descriptor)
	}

	return descriptors
}
