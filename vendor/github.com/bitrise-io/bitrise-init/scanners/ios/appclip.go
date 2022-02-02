package ios

import (
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-xcode/xcodeproj"
)

func schemeHasAppClipTarget(scheme xcodeproj.SchemeModel, targets []xcodeproj.TargetModel) bool {
	for _, target := range targets {
		for _, referenceID := range scheme.BuildableReferenceIDs {
			if referenceID == target.ID && target.HasAppClip {
				return true
			}
		}
	}

	return false
}

func shouldAppendExportAppClipStep(hasAppClip bool, exportMethod string) bool {
	return hasAppClip &&
		(exportMethod == "development" || exportMethod == "ad-hoc")
}

func appendExportAppClipStep(configBuilder *models.ConfigBuilderModel, workflowID models.WorkflowID) {
	exportXCArchiveStepInputModels := []envmanModels.EnvironmentItemModel{
		{ProjectPathInputKey: "$" + ProjectPathInputEnvKey},
		{SchemeInputKey: "$" + SchemeInputEnvKey},
		{ExportXCArchiveProductInputKey: ExportXCArchiveProductInputAppClipValue},
		{DistributionMethodInputKey: "$" + DistributionMethodEnvKey},
		{AutomaticCodeSigningKey: AutomaticCodeSigningValue},
	}
	configBuilder.AppendStepListItemsTo(workflowID, steps.ExportXCArchiveStepListItem(exportXCArchiveStepInputModels...))
}
