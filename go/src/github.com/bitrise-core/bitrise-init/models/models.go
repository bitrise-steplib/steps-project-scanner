package models

import bitriseModels "github.com/bitrise-io/bitrise/models"

// OptionModel ...
type OptionModel struct {
	Title  string `json:"title,omitempty" yaml:"title,omitempty"`
	EnvKey string `json:"env_key,omitempty" yaml:"env_key,omitempty"`

	ChildOptionMap map[string]*OptionModel `json:"value_map,omitempty" yaml:"value_map,omitempty"`
	Config         string                  `json:"config,omitempty" yaml:"config,omitempty"`

	Components []string     `json:"-" yaml:"-"`
	Head       *OptionModel `json:"-" yaml:"-"`
}

// BitriseConfigMap ...
type BitriseConfigMap map[string]string

// Warnings ...
type Warnings []string

// Errors ...
type Errors []string

// ScanResultModel ...
type ScanResultModel struct {
	PlatformOptionMap    map[string]OptionModel      `json:"options,omitempty" yaml:"options,omitempty"`
	PlatformConfigMapMap map[string]BitriseConfigMap `json:"configs,omitempty" yaml:"configs,omitempty"`
	PlatformWarningsMap  map[string]Warnings         `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	PlatformErrorsMap    map[string]Errors           `json:"errors,omitempty" yaml:"errors,omitempty"`
}

type workflowBuilderModel struct {
	PrepareSteps    []bitriseModels.StepListItemModel
	DependencySteps []bitriseModels.StepListItemModel
	MainSteps       []bitriseModels.StepListItemModel
	DeploySteps     []bitriseModels.StepListItemModel

	steps []bitriseModels.StepListItemModel
}

// WorkflowID ...
type WorkflowID string

const (
	// PrimaryWorkflowID ...
	PrimaryWorkflowID WorkflowID = "primary"
	// DeployWorkflowID ...
	DeployWorkflowID WorkflowID = "deploy"
)

// ConfigBuilderModel ...
type ConfigBuilderModel struct {
	workflowBuilderMap map[WorkflowID]*workflowBuilderModel
}
