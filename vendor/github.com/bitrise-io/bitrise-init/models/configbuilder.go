package models

import (
	bitriseModels "github.com/bitrise-io/bitrise/v2/models"
	envmanModels "github.com/bitrise-io/envman/v2/models"
)

// WorkflowID ...
type WorkflowID string

// PipelineID ...
type PipelineID string

const (
	// PrimaryWorkflowID ...
	PrimaryWorkflowID WorkflowID = "primary"
	// DeployWorkflowID ...
	DeployWorkflowID WorkflowID = "deploy"

	// FormatVersion ...
	FormatVersion = bitriseModels.FormatVersion

	defaultSteplibSource = "https://github.com/bitrise-io/bitrise-steplib.git"
)

// ConfigBuilderModel ...
type ConfigBuilderModel struct {
	workflowBuilderMap map[WorkflowID]*workflowBuilderModel
	pipelineBuilderMap map[PipelineID]*pipelineBuilderModel
}

// NewDefaultConfigBuilder ...
func NewDefaultConfigBuilder() *ConfigBuilderModel {
	return &ConfigBuilderModel{
		workflowBuilderMap: map[WorkflowID]*workflowBuilderModel{},
		pipelineBuilderMap: map[PipelineID]*pipelineBuilderModel{},
	}
}

// AppendStepListItemsTo ...
func (builder *ConfigBuilderModel) AppendStepListItemsTo(workflow WorkflowID, items ...bitriseModels.StepListItemModel) {
	workflowBuilder := builder.workflowBuilderMap[workflow]
	if workflowBuilder == nil {
		workflowBuilder = newDefaultWorkflowBuilder()
		builder.workflowBuilderMap[workflow] = workflowBuilder
	}
	workflowBuilder.appendStepListItems(items...)
}

// AppendStepListItemTo ...
func (builder *ConfigBuilderModel) SetGraphPipelineWorkflowTo(pipeline PipelineID, workflow WorkflowID, item bitriseModels.GraphPipelineWorkflowModel) {
	pipelineBuilder := builder.pipelineBuilderMap[pipeline]
	if pipelineBuilder == nil {
		pipelineBuilder = newDefaultPipelineBuilder()
		builder.pipelineBuilderMap[pipeline] = pipelineBuilder
	}
	pipelineBuilder.setGraphPipelineWorkflow(workflow, item)
}

// SetWorkflowDescriptionTo ...
func (builder *ConfigBuilderModel) SetWorkflowDescriptionTo(workflow WorkflowID, description string) {
	workflowBuilder := builder.workflowBuilderMap[workflow]
	if workflowBuilder == nil {
		workflowBuilder = newDefaultWorkflowBuilder()
		builder.workflowBuilderMap[workflow] = workflowBuilder
	}
	workflowBuilder.Description = description
}

// SetWorkflowSummaryTo ...
func (builder *ConfigBuilderModel) SetWorkflowSummaryTo(workflow WorkflowID, summary string) {
	workflowBuilder := builder.workflowBuilderMap[workflow]
	if workflowBuilder == nil {
		workflowBuilder = newDefaultWorkflowBuilder()
		builder.workflowBuilderMap[workflow] = workflowBuilder
	}
	workflowBuilder.Summary = summary
}

// Generate ...
func (builder *ConfigBuilderModel) Generate(projectType string, appEnvs ...envmanModels.EnvironmentItemModel) (bitriseModels.BitriseDataModel, error) {
	pipelines := map[string]bitriseModels.PipelineModel{}
	for pipelineID, pipelineBuilder := range builder.pipelineBuilderMap {
		pipelines[string(pipelineID)] = pipelineBuilder.generate()
	}

	workflows := map[string]bitriseModels.WorkflowModel{}
	for workflowID, workflowBuilder := range builder.workflowBuilderMap {
		workflows[string(workflowID)] = workflowBuilder.generate()
	}

	app := bitriseModels.AppModel{
		Environments: appEnvs,
	}

	return bitriseModels.BitriseDataModel{
		FormatVersion:        FormatVersion,
		DefaultStepLibSource: defaultSteplibSource,
		ProjectType:          projectType,
		Pipelines:            pipelines,
		Workflows:            workflows,
		App:                  app,
	}, nil
}
