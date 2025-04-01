package models

import (
	bitriseModels "github.com/bitrise-io/bitrise/v2/models"
)

type pipelineBuilderModel struct {
	Workflows   map[WorkflowID]bitriseModels.GraphPipelineWorkflowModel
	Description string
	Summary     string
}

func newDefaultPipelineBuilder() *pipelineBuilderModel {
	return &pipelineBuilderModel{
		Workflows: map[WorkflowID]bitriseModels.GraphPipelineWorkflowModel{},
	}
}

func (builder *pipelineBuilderModel) setGraphPipelineWorkflow(workflow WorkflowID, item bitriseModels.GraphPipelineWorkflowModel) {
	builder.Workflows[workflow] = item
}

func (builder *pipelineBuilderModel) generate() bitriseModels.PipelineModel {
	workflows := bitriseModels.GraphPipelineWorkflowListItemModel{}
	for workflowID, workflow := range builder.Workflows {
		workflows[string(workflowID)] = workflow
	}

	return bitriseModels.PipelineModel{
		Workflows:   workflows,
		Description: builder.Description,
		Summary:     builder.Summary,
	}
}
