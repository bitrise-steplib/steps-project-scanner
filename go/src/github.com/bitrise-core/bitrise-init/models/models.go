package models

import (
	bitriseModels "github.com/bitrise-io/bitrise/models"
)

const (
	primaryWorkflowID = "primary"
)

// Warnings ...
type Warnings []string

// ScanResultModel ...
type ScanResultModel struct {
	OptionsMap  map[string]OptionModel      `json:"options,omitempty" yaml:"options,omitempty"`
	ConfigsMap  map[string]BitriseConfigMap `json:"configs,omitempty" yaml:"configs,omitempty"`
	WarningsMap map[string]Warnings         `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// OptionValueMap ...
type OptionValueMap map[string]OptionModel

// OptionModel ...
type OptionModel struct {
	Title  string `json:"title,omitempty"  yaml:"title,omitempty"`
	EnvKey string `json:"env_key,omitempty"  yaml:"env_key,omitempty"`

	ValueMap OptionValueMap `json:"value_map,omitempty"  yaml:"value_map,omitempty"`
	Config   string         `json:"config,omitempty"  yaml:"config,omitempty"`
}

// BitriseConfigMap ...
type BitriseConfigMap map[string]string

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

// BitriseDataWithPrimaryWorkflowSteps ...
func BitriseDataWithPrimaryWorkflowSteps(steps []bitriseModels.StepListItemModel) bitriseModels.BitriseDataModel {
	workflows := map[string]bitriseModels.WorkflowModel{
		primaryWorkflowID: bitriseModels.WorkflowModel{
			Steps: steps,
		},
	}

	triggerMap := []bitriseModels.TriggerMapItemModel{
		bitriseModels.TriggerMapItemModel{
			Pattern:              "*",
			IsPullRequestAllowed: true,
			WorkflowID:           primaryWorkflowID,
		},
	}

	bitriseData := bitriseModels.BitriseDataModel{
		FormatVersion:        "1.2.0",
		DefaultStepLibSource: "https://github.com/bitrise-io/bitrise-steplib.git",
		TriggerMap:           triggerMap,
		Workflows:            workflows,
	}

	return bitriseData
}
