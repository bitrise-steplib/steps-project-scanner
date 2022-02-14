package reactnative

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/log"
	"gopkg.in/yaml.v2"
)

const (
	expoConfigName        = "react-native-expo-config"
	expoDefaultConfigName = "default-" + expoConfigName
)

const (
	expoProjectDirInputTitle   = "Expo project directory"
	expoProjectDirInputSummary = "Path of the directory containing the project's  `package.json` and app configuration file (`app.json`, `app.config.js`, `app.config.ts`)."
	expoProjectDirInputEnvKey  = "WORKDIR"

	expoPlatformInputTitle   = "Platform to build"
	expoPlatformInputSummary = "Which platform should be built by the deploy workflow?"
	expoPlatformInputEnvKey  = "PLATFORM"
)

// expoOptions implements ScannerInterface.Options function for Expo based React Native projects.
func (scanner *Scanner) expoOptions() (models.OptionNode, models.Warnings, error) {
	platformOption := models.NewOption(expoPlatformInputTitle, expoPlatformInputSummary, expoPlatformInputEnvKey, models.TypeSelector)
	configOption := models.NewConfigOption(expoConfigName, nil)

	for _, platform := range steps.RunEASBuildPlatforms {
		platformOption.AddConfig(platform, configOption)
	}

	return *platformOption, nil, nil
}

// expoConfigs implements ScannerInterface.Configs function for Expo based React Native projects.
func (scanner *Scanner) expoConfigs(isPrivateRepo bool) (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	// determine workdir
	packageJSONDir := filepath.Dir(scanner.packageJSONPth)
	relPackageJSONDir, err := utility.RelPath(scanner.searchDir, packageJSONDir)
	if err != nil {
		return models.BitriseConfigMap{}, fmt.Errorf("Failed to get relative package.json dir path, error: %s", err)
	}
	if relPackageJSONDir == "." {
		// package.json placed in the search dir, no need to change-dir in the workflows
		relPackageJSONDir = ""
	}
	log.TPrintf("Working directory: %v", relPackageJSONDir)

	// primary workflow
	primaryDescription := expoPrimaryWorkflowDescription
	if !scanner.hasTest {
		primaryDescription = expoPrimaryWorkflowNoTestsDescription
	}

	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.SetWorkflowDescriptionTo(models.PrimaryWorkflowID, primaryDescription)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepListV2(steps.PrepareListParams{
		ShouldIncludeCache:       false,
		ShouldIncludeActivateSSH: isPrivateRepo,
	})...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, scanner.getTestSteps(relPackageJSONDir)...)

	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepListV2(false)...)

	// deploy workflow
	deployDescription := expoDeployWorkflowDescription
	if !scanner.hasTest {
		deployDescription = expoDeployWorkflowNoTestsDescription
	}

	configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployDescription)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepListV2(steps.PrepareListParams{
		ShouldIncludeCache:       false,
		ShouldIncludeActivateSSH: isPrivateRepo,
	})...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, scanner.getTestSteps(relPackageJSONDir)...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.RunEASBuildStepListItem(relPackageJSONDir, "$"+expoPlatformInputEnvKey))
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

	// generate bitrise.yml
	bitriseDataModel, err := configBuilder.Generate(scannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(bitriseDataModel)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configMap[expoConfigName] = string(data)

	return configMap, nil
}

// expoDefaultOptions implements ScannerInterface.DefaultOptions function for Expo based React Native projects.
func (Scanner) expoDefaultOptions() models.OptionNode {
	workDirOption := models.NewOption(expoProjectDirInputTitle, expoProjectDirInputSummary, expoProjectDirInputEnvKey, models.TypeUserInput)
	platformOption := models.NewOption(expoPlatformInputTitle, expoPlatformInputSummary, expoPlatformInputEnvKey, models.TypeSelector)
	configOption := models.NewConfigOption(expoDefaultConfigName, nil)

	workDirOption.AddConfig("", platformOption)
	for _, platform := range steps.RunEASBuildPlatforms {
		platformOption.AddConfig(platform, configOption)
	}

	return *workDirOption
}

// expoDefaultConfigs implements ScannerInterface.DefaultConfigs function for Expo based React Native projects.
func (scanner Scanner) expoDefaultConfigs() (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	// primary workflow
	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.SetWorkflowDescriptionTo(models.PrimaryWorkflowID, expoPrimaryWorkflowDescription)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepListV2(steps.PrepareListParams{
		ShouldIncludeCache:       false,
		ShouldIncludeActivateSSH: true,
	})...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, getTestSteps("$"+expoProjectDirInputEnvKey, true, true)...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepListV2(false)...)

	// deploy workflow
	configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, expoDeployWorkflowDescription)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepListV2(steps.PrepareListParams{
		ShouldIncludeCache:       false,
		ShouldIncludeActivateSSH: true,
	})...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, getTestSteps("$"+expoProjectDirInputEnvKey, true, true)...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.RunEASBuildStepListItem("$"+expoProjectDirInputEnvKey, "$"+expoPlatformInputEnvKey))
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

	// generate bitrise.yml
	bitriseDataModel, err := configBuilder.Generate(scannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(bitriseDataModel)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configMap[expoDefaultConfigName] = string(data)

	return configMap, nil
}
