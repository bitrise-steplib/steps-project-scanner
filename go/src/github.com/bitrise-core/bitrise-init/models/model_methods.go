package models

import (
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
)

const (
	// FormatVersion ...
	FormatVersion        = "1.3.1"
	defaultSteplibSource = "https://github.com/bitrise-io/bitrise-steplib.git"
	primaryWorkflowID    = "primary"
	deployWorkflowID     = "deploy"
)

// AddError ...
func (result *ScanResultModel) AddError(platform string, errorMessage string) {
	if result.ErrorsMap == nil {
		result.ErrorsMap = map[string]Errors{}
	}
	if result.ErrorsMap[platform] == nil {
		result.ErrorsMap[platform] = Errors{}
	}
	result.ErrorsMap[platform] = append(result.ErrorsMap[platform], errorMessage)
}

// NewOptionModel ...
func NewOptionModel(title, envKey string) OptionModel {
	return OptionModel{
		Title:  title,
		EnvKey: envKey,

		ValueMap: OptionValueMap{},
	}
}

// NewEmptyOptionModel ...
func NewEmptyOptionModel() OptionModel {
	return OptionModel{
		ValueMap: OptionValueMap{},
	}
}

// GetValues ...
func (option OptionModel) GetValues() []string {
	if option.Config != "" {
		return []string{option.Config}
	}

	values := []string{}
	for value := range option.ValueMap {
		values = append(values, value)
	}
	return values
}

// BitriseDataWithCIWorkflow ...
func BitriseDataWithCIWorkflow(appEnvs []envmanModels.EnvironmentItemModel, steps []bitriseModels.StepListItemModel) bitriseModels.BitriseDataModel {
	workflows := map[string]bitriseModels.WorkflowModel{
		primaryWorkflowID: bitriseModels.WorkflowModel{
			Steps: steps,
		},
	}

	triggerMap := []bitriseModels.TriggerMapItemModel{
		bitriseModels.TriggerMapItemModel{
			PushBranch: "*",
			WorkflowID: primaryWorkflowID,
		},
		bitriseModels.TriggerMapItemModel{
			PullRequestSourceBranch: "*",
			WorkflowID:              primaryWorkflowID,
		},
	}

	app := bitriseModels.AppModel{
		Environments: appEnvs,
	}

	return bitriseModels.BitriseDataModel{
		FormatVersion:        FormatVersion,
		DefaultStepLibSource: defaultSteplibSource,
		TriggerMap:           triggerMap,
		Workflows:            workflows,
		App:                  app,
	}
}

// BitriseDataWithCIAndCDWorkflow ...
func BitriseDataWithCIAndCDWorkflow(appEnvs []envmanModels.EnvironmentItemModel, ciSteps, deploySteps []bitriseModels.StepListItemModel) bitriseModels.BitriseDataModel {
	workflows := map[string]bitriseModels.WorkflowModel{
		primaryWorkflowID: bitriseModels.WorkflowModel{
			Steps: ciSteps,
		},
		deployWorkflowID: bitriseModels.WorkflowModel{
			Steps: deploySteps,
		},
	}

	triggerMap := []bitriseModels.TriggerMapItemModel{
		bitriseModels.TriggerMapItemModel{
			PushBranch: "*",
			WorkflowID: primaryWorkflowID,
		},
		bitriseModels.TriggerMapItemModel{
			PullRequestSourceBranch: "*",
			WorkflowID:              primaryWorkflowID,
		},
	}

	app := bitriseModels.AppModel{
		Environments: appEnvs,
	}

	return bitriseModels.BitriseDataModel{
		FormatVersion:        FormatVersion,
		DefaultStepLibSource: defaultSteplibSource,
		TriggerMap:           triggerMap,
		Workflows:            workflows,
		App:                  app,
	}
}
