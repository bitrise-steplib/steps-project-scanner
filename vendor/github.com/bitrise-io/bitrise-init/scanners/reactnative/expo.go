package reactnative

import (
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
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
func (scanner *Scanner) expoOptions() models.OptionNode {
	platformOption := models.NewOption(expoPlatformInputTitle, expoPlatformInputSummary, expoPlatformInputEnvKey, models.TypeSelector)
	configOption := models.NewConfigOption(expoConfigName, nil)

	for _, platform := range steps.RunEASBuildPlatforms {
		platformOption.AddConfig(platform, configOption)
	}

	return *platformOption
}

// expoConfigs implements ScannerInterface.Configs function for Expo based React Native projects.
func (scanner *Scanner) expoConfigs(project project, isPrivateRepo bool) (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	if project.projectRelDir == "." {
		// package.json placed in the search dir, no need to change-dir in the workflows
		project.projectRelDir = ""
	}
	testSteps := getTestSteps(project.projectRelDir, project.hasYarnLockFile, project.hasTest)

	// primary workflow
	primaryDescription := expoPrimaryWorkflowDescription
	if !project.hasTest {
		primaryDescription = expoPrimaryWorkflowNoTestsDescription
	}

	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.SetWorkflowDescriptionTo(models.PrimaryWorkflowID, primaryDescription)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		ShouldIncludeActivateSSH: isPrivateRepo,
	})...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.RestoreNPMCache())
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, testSteps...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.SaveNPMCache())
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList()...)

	// deploy workflow
	deployDescription := expoDeployWorkflowDescription
	if !project.hasTest {
		deployDescription = expoDeployWorkflowNoTestsDescription
	}

	configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployDescription)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		ShouldIncludeActivateSSH: isPrivateRepo,
	})...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, testSteps...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.RunEASBuildStepListItem(project.projectRelDir, "$"+expoPlatformInputEnvKey))
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList()...)

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
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		ShouldIncludeActivateSSH: true,
	})...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.RestoreNPMCache())
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, getTestSteps("$"+expoProjectDirInputEnvKey, true, true)...)
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.SaveNPMCache())
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList()...)

	// deploy workflow
	configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, expoDeployWorkflowDescription)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(steps.PrepareListParams{
		ShouldIncludeActivateSSH: true,
	})...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, getTestSteps("$"+expoProjectDirInputEnvKey, true, true)...)
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.RunEASBuildStepListItem("$"+expoProjectDirInputEnvKey, "$"+expoPlatformInputEnvKey))
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList()...)

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
